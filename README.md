# cli-notify

为 Gemini CLI / Codex CLI 提供的 Windows 包装工具，在以下情况会发送 **Windows 桌面通知 (Toast Notification)**：

- **CLI 需要你确认操作**（例如文件编辑、命令执行）—— 以便你及时切回终端。
- **CLI 完成回复**—— 让你知道输出已经准备就绪。

当你在 Gemini 或 Codex 工作时切换到其他窗口，这个工具将非常有用。只要它需要你的注意，你就会收到桌面通知。

当前支持：

- [Gemini CLI](https://github.com/google-gemini/gemini-cli)
- [Codex CLI](https://github.com/openai/codex)

## 工作原理

`*-cli-notify` 会在一个 [ConPTY](https://devblogs.microsoft.com/commandline/windows-command-line-introducing-the-windows-pseudo-console-conpty/) (Windows 伪控制台) 环境中启动目标 CLI，并轮询控制台标题。不同 CLI 使用不同的标题状态标记。

### Gemini

| 图标 | 含义 | 通知内容 |
|------|---------|--------------|
| ✋ | 需要用户确认 | "需要你确认操作，请切回终端" |
| ✦ / ⏲ → ◇ | 已完成回复 | "已完成回复，请切回终端查看" |

### Codex

Codex 默认的终端标题包含 `activity` 和 `project-name`。本工具会监听：

| 标题标记 | 含义 | 通知内容 |
|------|---------|--------------|
| `[ ! ] Action Required` / `[ . ] Action Required` | 需要用户确认 | "需要你确认操作，请切回终端" |
| Braille activity spinner 或 `Starting` / `Working` / `Thinking` / `Waiting` → `Ready` 或仅项目名标题 | 已完成回复 | "已完成回复，请切回终端查看" |

![Notification Demo](images/notification.png)

## 安装

### 通过 Release 页面下载

从 [Releases](https://github.com/jiangwan0130/gemini-cli-notify/releases) 页面下载需要的 `.exe` 可执行文件，并将其放入系统环境变量 `PATH` 中。

### 从源码安装

```bash
go install github.com/jiangwan0130/gemini-cli-notify/cmd/gemini-cli-notify@latest
go install github.com/jiangwan0130/gemini-cli-notify/cmd/codex-cli-notify@latest
```

> 需要 Go 1.23+ 并且仅支持 Windows 系统。

## 使用方法

将对应的 wrapper 当作原 CLI 的平级替代品直接使用即可：

```bash
gemini-cli-notify "解释这段代码"
codex-cli-notify "解释这段代码"
```

所有传入的参数都会直接被转发给对应的 CLI。`gemini-cli-notify` 始终启动 Gemini CLI，`codex-cli-notify` 始终启动 Codex CLI。

### 提示：创建别名 (Alias)

你可以将其添加到你的 PowerShell 配置文件 (`$PROFILE`) 中：

```powershell
Set-Alias gemini gemini-cli-notify
Set-Alias codex codex-cli-notify
```

## 构建 (Build)

```bash
go build -o gemini-cli-notify.exe ./cmd/gemini-cli-notify
go build -o codex-cli-notify.exe ./cmd/codex-cli-notify
```

## 环境要求

- Windows 10 1809+ (支持 ConPTY)
- 使用 Gemini 时：已安装 [Gemini CLI](https://github.com/google-gemini/gemini-cli) 并确保它已加入系统环境变量 `PATH` 中。
- 使用 Codex 时：已安装 [Codex CLI](https://github.com/openai/codex) 并确保它已加入系统环境变量 `PATH` 中。

## 开源协议

[MIT](LICENSE)
