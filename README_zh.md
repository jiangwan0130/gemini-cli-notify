# gemini-cli-notify

[English](README.md) | [简体中文](README_zh.md)

为 [Gemini CLI](https://github.com/google-gemini/gemini-cli) 提供的 Windows 包装工具，在以下情况会发送 **Windows 桌面通知 (Toast Notification)**：

- **Gemini 需要你确认操作**（例如文件编辑、命令执行）—— 以便你及时切回终端。
- **Gemini 完成回复**—— 让你知道输出已经准备就绪。

当你在 Gemini 工作时切换到其他窗口，这个工具将非常有用。只要它需要你的注意，你就会收到桌面通知。

## 工作原理

`gemini-cli-notify` 会在一个 [ConPTY](https://devblogs.microsoft.com/commandline/windows-command-line-introducing-the-windows-pseudo-console-conpty/) (Windows 伪控制台) 环境中启动 `gemini`，并轮询控制台标题。Gemini CLI 会使用状态图标更新控制台标题：

| 图标 | 含义 | 通知内容 |
|------|---------|--------------|
| ✋ | 需要用户确认 | "需要你确认操作，请切回终端" |
| ✦ / ⏲ → ◇ | 已完成回复 | "已完成回复，请切回终端查看" |

## 安装

### 通过 Release 页面下载

从 [Releases](https://github.com/jiangwan0130/gemini-cli-notify/releases) 页面下载最新的 `.exe` 可执行文件，并将其放入系统环境变量 `PATH` 中。

### 从源码安装

```bash
go install github.com/jiangwan0130/gemini-cli-notify@latest
```

> 需要 Go 1.23+ 并且仅支持 Windows 系统。

## 使用方法

将 `gemini-cli-notify` 当作 `gemini` 的平级替代品直接使用即可：

```bash
gemini-cli-notify "解释这段代码"
```

所有传入的参数都会直接被转发给 `gemini`。

### 提示：创建别名 (Alias)

你可以将其添加到你的 PowerShell 配置文件 (`$PROFILE`) 中：

```powershell
Set-Alias gemini gemini-cli-notify
```

## 构建 (Build)

```bash
go build -o gemini-cli-notify.exe .
```

## 环境要求

- Windows 10 1809+ (支持 ConPTY)
- 已安装 [Gemini CLI](https://github.com/google-gemini/gemini-cli) 并确保它已加入系统环境变量 `PATH` 中。

## 开源协议

[MIT](LICENSE)
