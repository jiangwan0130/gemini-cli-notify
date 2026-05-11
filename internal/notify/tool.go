//go:build windows

package notify

import "strings"

type TitleState int

const (
	TitleUnknown TitleState = iota
	TitleIdle
	TitleResponding
	TitleNeedsConfirmation
)

type ToolConfig struct {
	ID          string
	Command     string
	DisplayName string
	AppID       string
	ConfirmBody string
	DoneBody    string
	Matcher     func(string) TitleState
}

func GeminiTool() ToolConfig {
	return ToolConfig{
		ID:          "gemini",
		Command:     "gemini",
		DisplayName: "Gemini CLI",
		AppID:       "Gemini CLI",
		ConfirmBody: "需要你确认操作，请切回终端",
		DoneBody:    "已完成回复，请切回终端查看",
		Matcher:     MatchGeminiTitle,
	}
}

func CodexTool() ToolConfig {
	return ToolConfig{
		ID:          "codex",
		Command:     "codex",
		DisplayName: "Codex CLI",
		AppID:       "Codex CLI",
		ConfirmBody: "需要你确认操作，请切回终端",
		DoneBody:    "已完成回复，请切回终端查看",
		Matcher:     MatchCodexTitle,
	}
}

func MatchGeminiTitle(title string) TitleState {
	if strings.Contains(title, "✋") {
		return TitleNeedsConfirmation
	}
	if strings.Contains(title, "✦") || strings.Contains(title, "⏲") {
		return TitleResponding
	}
	if strings.Contains(title, "◇") {
		return TitleIdle
	}
	return TitleUnknown
}

func MatchCodexTitle(title string) TitleState {
	if strings.Contains(title, "[ ! ] Action Required") ||
		strings.Contains(title, "[ . ] Action Required") {
		return TitleNeedsConfirmation
	}

	for _, frame := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if strings.Contains(title, frame) {
			return TitleResponding
		}
	}

	lower := strings.ToLower(title)
	if strings.Contains(lower, "starting") ||
		strings.Contains(lower, "working") ||
		strings.Contains(lower, "thinking") ||
		strings.Contains(lower, "waiting") {
		return TitleResponding
	}
	if strings.Contains(lower, "ready") {
		return TitleIdle
	}
	if strings.TrimSpace(title) != "" {
		return TitleIdle
	}
	return TitleUnknown
}

func BuildCommandLine(cfg ToolConfig, args []string) string {
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
