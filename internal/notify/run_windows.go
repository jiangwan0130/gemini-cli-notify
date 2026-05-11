//go:build windows

package notify

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/UserExistsError/conpty"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleTitle   = kernel32.NewProc("GetConsoleTitleW")
	procGetConsoleScreenB = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type consoleScreenBufferInfo struct {
	Size              [2]int16
	CursorPosition    [2]int16
	Attributes        uint16
	Window            [4]int16
	MaximumWindowSize [2]int16
}

func getConsoleSize() (width, height int) {
	h, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	var info consoleScreenBufferInfo
	r, _, _ := procGetConsoleScreenB.Call(uintptr(h), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 120, 30
	}
	w := int(info.Window[2]-info.Window[0]) + 1
	ht := int(info.Window[3]-info.Window[1]) + 1
	if w <= 0 || ht <= 0 {
		return 120, 30
	}
	return w, ht
}

func getConsoleTitle() string {
	buf := make([]uint16, 512)
	r, _, _ := procGetConsoleTitle.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:r])
}

func getRepoName() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(out))
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func sanitizeForPS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func showToastNotification(cfg ToolConfig, repoName, body string) {
	titleLine := cfg.DisplayName
	if repoName != "" {
		titleLine = fmt.Sprintf("%s  [%s]", cfg.DisplayName, sanitizeForPS(repoName))
	}
	body = sanitizeForPS(body)

	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$xml = [xml]$template.GetXml()
$xml.GetElementsByTagName('text')[0].AppendChild($xml.CreateTextNode('%s')) > $null
$xml.GetElementsByTagName('text')[1].AppendChild($xml.CreateTextNode('%s')) > $null
$audio = $xml.CreateElement('audio')
$audio.SetAttribute('src', 'ms-winsoundevent:Notification.Default')
$xml.toast.AppendChild($audio) > $null
$ser = New-Object Windows.Data.Xml.Dom.XmlDocument
$ser.LoadXml($xml.OuterXml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s').Show([Windows.UI.Notifications.ToastNotification]::new($ser))
`, titleLine, body, sanitizeForPS(cfg.AppID))
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
	go func() { _ = cmd.Wait() }()
}

func Run(cfg ToolConfig, args []string) int {
	repoName := getRepoName()
	cmdLine := BuildCommandLine(cfg, args)

	w, h := getConsoleSize()
	cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(w, h))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s-notify: failed to start: %v\n", cfg.ID, err)
		return 1
	}
	defer cpty.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s-notify: failed to set raw mode: %v\n", cfg.ID, err)
		return 1
	}
	defer term.Restore(fd, oldState)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_, _ = io.Copy(cpty, os.Stdin)
	}()

	go func() {
		_, _ = io.Copy(os.Stdout, cpty)
	}()

	go func() {
		lastW, lastH := w, h
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				curW, curH := getConsoleSize()
				if curW != lastW || curH != lastH {
					lastW, lastH = curW, curH
					_ = cpty.Resize(curW, curH)
				}
			}
		}
	}()

	go func() {
		var mu sync.Mutex
		confirmNotified := false
		wasResponding := false
		completionNotified := false
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				title := getConsoleTitle()
				mu.Lock()
				state := cfg.Matcher(title)
				if state == TitleNeedsConfirmation {
					if !confirmNotified {
						confirmNotified = true
						go showToastNotification(cfg, repoName, cfg.ConfirmBody)
					}
				} else {
					confirmNotified = false
				}
				if state == TitleResponding {
					wasResponding = true
					completionNotified = false
				} else if state == TitleIdle && wasResponding && !completionNotified {
					completionNotified = true
					wasResponding = false
					go showToastNotification(cfg, repoName, cfg.DoneBody)
				}
				mu.Unlock()
			}
		}
	}()

	exitCode, _ := cpty.Wait(context.Background())
	cancel()

	term.Restore(fd, oldState)
	return int(exitCode)
}
