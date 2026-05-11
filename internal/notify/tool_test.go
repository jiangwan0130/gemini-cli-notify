//go:build windows

package notify

import "testing"

func TestBuildCommandLineQuotesArguments(t *testing.T) {
	got := BuildCommandLine(CodexTool(), []string{"hello world", "--flag", `say "hi"`})
	want := `cmd /c codex "hello world" --flag "say \"hi\""`
	if got != want {
		t.Fatalf("command line = %q, want %q", got, want)
	}
}

func TestGeminiTitleStates(t *testing.T) {
	m := GeminiTool().Matcher
	if m("✋ confirm") != TitleNeedsConfirmation {
		t.Fatal("Gemini hand title should need confirmation")
	}
	if m("✦ responding") != TitleResponding {
		t.Fatal("Gemini sparkle title should be responding")
	}
	if m("⏲ running") != TitleResponding {
		t.Fatal("Gemini timer title should be responding")
	}
	if m("◇ idle") != TitleIdle {
		t.Fatal("Gemini diamond title should be idle")
	}
}

func TestCodexTitleStates(t *testing.T) {
	m := CodexTool().Matcher
	if m("[ ! ] Action Required | project") != TitleNeedsConfirmation {
		t.Fatal("Codex action-required title should need confirmation")
	}
	if m("[ . ] Action Required | project") != TitleNeedsConfirmation {
		t.Fatal("Codex animated action-required title should need confirmation")
	}
	if m("⠋ project") != TitleResponding {
		t.Fatal("Codex spinner title should be responding")
	}
	if m("Starting | project") != TitleResponding {
		t.Fatal("Codex starting title should be responding")
	}
	if m("Working | project") != TitleResponding {
		t.Fatal("Codex working title should be responding")
	}
	if m("Thinking | project") != TitleResponding {
		t.Fatal("Codex thinking title should be responding")
	}
	if m("Waiting | project") != TitleResponding {
		t.Fatal("Codex waiting title should be responding")
	}
	if m("Ready | project") != TitleIdle {
		t.Fatal("Codex ready title should be idle")
	}
	if m("project") != TitleIdle {
		t.Fatal("Codex project-only title should be idle")
	}
	if m("") != TitleUnknown {
		t.Fatal("empty Codex title should be unknown")
	}
}
