//go:build windows

package main

import "testing"

func TestSelectToolFromExecutableName(t *testing.T) {
	cfg, args, err := selectTool("codex-cli-notify.exe", []string{"ask"})
	if err != nil {
		t.Fatalf("selectTool returned error: %v", err)
	}
	if cfg.ID != "codex" {
		t.Fatalf("tool ID = %q, want codex", cfg.ID)
	}
	if len(args) != 1 || args[0] != "ask" {
		t.Fatalf("args = %#v, want [ask]", args)
	}
}

func TestSelectToolFlagOverridesExecutableName(t *testing.T) {
	cfg, args, err := selectTool("gemini-cli-notify.exe", []string{"--tool", "codex", "ask"})
	if err != nil {
		t.Fatalf("selectTool returned error: %v", err)
	}
	if cfg.ID != "codex" {
		t.Fatalf("tool ID = %q, want codex", cfg.ID)
	}
	if len(args) != 1 || args[0] != "ask" {
		t.Fatalf("args = %#v, want [ask]", args)
	}
}

func TestBuildCommandLineQuotesArguments(t *testing.T) {
	got := buildCommandLine(toolConfigs["codex"], []string{"hello world", "--flag"})
	want := `cmd /c codex "hello world" --flag`
	if got != want {
		t.Fatalf("command line = %q, want %q", got, want)
	}
}

func TestGeminiTitleStates(t *testing.T) {
	m := toolConfigs["gemini"].Matcher
	if m("✋ confirm") != titleNeedsConfirmation {
		t.Fatal("Gemini hand title should need confirmation")
	}
	if m("✦ responding") != titleResponding {
		t.Fatal("Gemini sparkle title should be responding")
	}
	if m("◇ idle") != titleIdle {
		t.Fatal("Gemini diamond title should be idle")
	}
}

func TestCodexTitleStates(t *testing.T) {
	m := toolConfigs["codex"].Matcher
	if m("[ ! ] Action Required | gemini-cli-notify") != titleNeedsConfirmation {
		t.Fatal("Codex action-required title should need confirmation")
	}
	if m("⠋ gemini-cli-notify") != titleResponding {
		t.Fatal("Codex spinner title should be responding")
	}
	if m("gemini-cli-notify") != titleIdle {
		t.Fatal("Codex project-only title should be idle")
	}
}
