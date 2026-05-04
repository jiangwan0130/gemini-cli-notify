# cli-notify

A Windows wrapper for Gemini CLI / Codex CLI that sends **Windows Toast notifications** when:

- **The CLI needs your confirmation** (e.g. file edits, command execution) — so you can switch back to the terminal
- **The CLI finishes its response** — so you know the output is ready

This is useful when you switch away from the terminal while Gemini or Codex is working. You'll get a desktop notification the moment it needs attention.

Supported tools:

- [Gemini CLI](https://github.com/google-gemini/gemini-cli)
- [Codex CLI](https://github.com/openai/codex)

## How It Works

`*-cli-notify` launches the target CLI inside a [ConPTY](https://devblogs.microsoft.com/commandline/windows-command-line-introducing-the-windows-pseudo-console-conpty/) (Windows Pseudo Console) and polls the console title. Each supported CLI uses different title status markers.

### Gemini

| Icon | Meaning | Notification |
|------|---------|--------------|
| ✋ | Needs user confirmation | "需要你确认操作，请切回终端" |
| ✦ / ⏲ → ◇ | Finished responding | "已完成回复，请切回终端查看" |

### Codex

Codex's default terminal title includes `activity` and `project-name`. The wrapper watches:

| Title marker | Meaning | Notification |
|--------------|---------|--------------|
| `[ ! ] Action Required` / `[ . ] Action Required` | Needs user confirmation | "需要你确认操作，请切回终端" |
| Braille activity spinner → project-only title | Finished responding | "已完成回复，请切回终端查看" |

![Notification Demo](images/notification.png)

## Installation

### From Release

Download the latest `.exe` from the [Releases](https://github.com/jiangwan0130/gemini-cli-notify/releases) page and place it in your `PATH`.

### From Source

```bash
go install github.com/jiangwan0130/gemini-cli-notify@latest
```

> Requires Go 1.23+ and Windows.

## Usage

Use the matching wrapper as a drop-in replacement for the CLI:

```bash
gemini-cli-notify "explain this code"
codex-cli-notify "explain this code"
```

All arguments are forwarded to the selected CLI directly.

You can also force a tool explicitly:

```bash
gemini-cli-notify --tool codex "explain this code"
```

### Tip: Create an alias

Add to your PowerShell profile (`$PROFILE`):

```powershell
Set-Alias gemini gemini-cli-notify
Set-Alias codex codex-cli-notify
```

## Build

```bash
go build -o gemini-cli-notify.exe .
go build -o codex-cli-notify.exe .
```

## Requirements

- Windows 10 1809+ (ConPTY support)
- [Gemini CLI](https://github.com/google-gemini/gemini-cli) installed and in `PATH` for Gemini usage
- [Codex CLI](https://github.com/openai/codex) installed and in `PATH` for Codex usage

## License

[MIT](LICENSE)
