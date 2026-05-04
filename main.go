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

type titleState int

const (
	titleUnknown titleState = iota
	titleIdle
	titleResponding
	titleNeedsConfirmation
)

type toolConfig struct {
	ID          string
	Command     string
	DisplayName string
	AppID       string
	ConfirmBody string
	DoneBody    string
	Matcher     func(string) titleState
}

var toolConfigs = map[string]toolConfig{
	"gemini": {
		ID:          "gemini",
		Command:     "gemini",
		DisplayName: "Gemini CLI",
		AppID:       "Gemini CLI",
		ConfirmBody: "需要你确认操作，请切回终端",
		DoneBody:    "已完成回复，请切回终端查看",
		Matcher:     matchGeminiTitle,
	},
	"codex": {
		ID:          "codex",
		Command:     "codex",
		DisplayName: "Codex CLI",
		AppID:       "Codex CLI",
		ConfirmBody: "需要你确认操作，请切回终端",
		DoneBody:    "已完成回复，请切回终端查看",
		Matcher:     matchCodexTitle,
	},
}

func matchGeminiTitle(title string) titleState {
	if strings.Contains(title, "✋") {
		return titleNeedsConfirmation
	}
	if strings.Contains(title, "✦") || strings.Contains(title, "⏲") {
		return titleResponding
	}
	if strings.Contains(title, "◇") {
		return titleIdle
	}
	return titleUnknown
}

func matchCodexTitle(title string) titleState {
	if strings.Contains(title, "[ ! ] Action Required") ||
		strings.Contains(title, "[ . ] Action Required") {
		return titleNeedsConfirmation
	}

	for _, frame := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(title, frame) {
			return titleResponding
		}
	}

	lower := strings.ToLower(title)
	if strings.Contains(lower, "working") ||
		strings.Contains(lower, "thinking") ||
		strings.Contains(lower, "waiting") {
		return titleResponding
	}
	if strings.TrimSpace(title) != "" {
		return titleIdle
	}
	return titleUnknown
}

func selectTool(exeName string, args []string) (toolConfig, []string, error) {
	if len(args) >= 2 && args[0] == "--tool" {
		cfg, ok := toolConfigs[strings.ToLower(args[1])]
		if !ok {
			return toolConfig{}, nil, fmt.Errorf("unknown tool %q", args[1])
		}
		return cfg, args[2:], nil
	}

	exeName = strings.ToLower(exeName)
	if strings.Contains(exeName, "codex") {
		return toolConfigs["codex"], args, nil
	}
	return toolConfigs["gemini"], args, nil
}

func buildCommandLine(cfg toolConfig, args []string) string {
	parts := []string{"cmd", "/c", cfg.Command}
	for _, arg := range args {
		parts = append(parts, quoteWindowsArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteWindowsArg(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
			continue
		}
		if backslashes > 0 {
			b.WriteString(strings.Repeat("\\", backslashes))
			backslashes = 0
		}
		b.WriteRune(r)
	}
	if backslashes > 0 {
		b.WriteString(strings.Repeat("\\", backslashes*2))
	}
	b.WriteByte('"')
	return b.String()
}

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

// sanitizeForPS escapes single quotes in a string for safe embedding in PowerShell scripts.
func sanitizeForPS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func showToastNotification(cfg toolConfig, repoName, body string) {
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

func main() {
	repoName := getRepoName()

	cfg, args, err := selectTool(os.Args[0], os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cli-notify: %v\n", err)
		os.Exit(2)
	}
	cmdLine := buildCommandLine(cfg, args)

	// Get initial console size
	w, h := getConsoleSize()

	// Start ConPTY
	cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(w, h))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s-notify: failed to start: %v\n", cfg.ID, err)
		os.Exit(1)
	}
	defer cpty.Close()

	// Switch stdin to raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s-notify: failed to set raw mode: %v\n", cfg.ID, err)
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
				state := cfg.Matcher(title)
				// 确认操作通知
				if state == titleNeedsConfirmation {
					if !confirmNotified {
						confirmNotified = true
						go showToastNotification(cfg, repoName, cfg.ConfirmBody)
					}
				} else {
					confirmNotified = false
				}
				// 输出结束通知
				if state == titleResponding {
					wasResponding = true
					completionNotified = false
				} else if state == titleIdle && wasResponding && !completionNotified {
					completionNotified = true
					wasResponding = false
					go showToastNotification(cfg, repoName, cfg.DoneBody)
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
