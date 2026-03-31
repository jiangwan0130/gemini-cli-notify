//go:build windows

package main

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
	Window            [4]int16 // Left, Top, Right, Bottom
	MaximumWindowSize [2]int16
}

func getConsoleSize() (width, height int) {
	h, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	var info consoleScreenBufferInfo
	r, _, _ := procGetConsoleScreenB.Call(uintptr(h), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 120, 30 // fallback
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
	// take the last path component as repo name
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func showToast(repoName string) {
	titleLine := "Gemini CLI"
	if repoName != "" {
		titleLine = fmt.Sprintf("Gemini CLI  [%s]", repoName)
	}
	bodyLine := "需要你确认操作，请切回终端"

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
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Gemini CLI').Show([Windows.UI.Notifications.ToastNotification]::new($ser))
`, titleLine, bodyLine)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
	// fire-and-forget: don't wait for powershell
	go func() { _ = cmd.Wait() }()
}

func showCompletionToast(repoName string) {
	titleLine := "Gemini CLI"
	if repoName != "" {
		titleLine = fmt.Sprintf("Gemini CLI  [%s]", repoName)
	}
	bodyLine := "已完成回复，请切回终端查看"

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
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Gemini CLI').Show([Windows.UI.Notifications.ToastNotification]::new($ser))
`, titleLine, bodyLine)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
	go func() { _ = cmd.Wait() }()
}

func main() {
	repoName := getRepoName()

	// Build command line
	args := os.Args[1:]
	cmdLine := "cmd /c gemini"
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}

	// Get initial console size
	w, h := getConsoleSize()

	// Start ConPTY
	cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(w, h))
	if err != nil {
		fmt.Fprintf(os.Stderr, "gemini-notify: failed to start: %v\n", err)
		os.Exit(1)
	}
	defer cpty.Close()

	// Switch stdin to raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gemini-notify: failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// stdin → PTY
	go func() {
		_, _ = io.Copy(cpty, os.Stdin)
	}()

	// PTY → stdout
	go func() {
		_, _ = io.Copy(os.Stdout, cpty)
	}()

	// Resize polling
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

	// Title polling → Toast notification
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
				// 确认操作通知（✋）
				if strings.Contains(title, "✋") {
					if !confirmNotified {
						confirmNotified = true
						go showToast(repoName)
					}
				} else {
					confirmNotified = false
				}
				// 输出结束通知（✦ / ⏲ → ◇）
				if strings.Contains(title, "✦") || strings.Contains(title, "⏲") {
					wasResponding = true
					completionNotified = false
				} else if strings.Contains(title, "◇") && wasResponding && !completionNotified {
					completionNotified = true
					wasResponding = false
					go showCompletionToast(repoName)
				}
				mu.Unlock()
			}
		}
	}()

	// Wait for gemini to exit
	exitCode, _ := cpty.Wait(context.Background())
	cancel()

	// Restore terminal (handled by defer) and exit
	term.Restore(fd, oldState)
	os.Exit(int(exitCode))
}
