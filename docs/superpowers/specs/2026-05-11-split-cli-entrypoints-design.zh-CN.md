# 拆分 CLI 入口设计

## 背景

当前仓库的 `codex-notify` 分支已经是 Gemini/Codex 混用实现。旧的 `origin/main` 仍然是 Gemini-only，而当前本地 `main` 和 `codex-notify` 都指向一个支持多 CLI 工具的提交。

目标方向是保留一个仓库，但对外提供两个独立的可执行文件：

- `gemini-cli-notify.exe`
- `codex-cli-notify.exe`

Windows 终端包装、通知、仓库名识别等公共逻辑只维护一份。

## 设计

将公共实现移动到内部包，并创建两个命令入口：

```text
cmd/
  gemini-cli-notify/
    main.go
  codex-cli-notify/
    main.go
internal/
  notify/
    ...
```

每个命令入口都会把固定的工具配置传给共享包。最终用户路径中不再使用运行时 `--tool` 切换，也不再根据可执行文件名猜测工具类型。

## 行为

`gemini-cli-notify` 始终启动 `gemini`，并使用 Gemini 的标题状态匹配规则：

- `✋` 表示需要用户确认。
- `✦` 或 `⏲` 表示 CLI 正在回复。
- `◇` 表示空闲。

`codex-cli-notify` 始终启动 `codex`，并使用 Codex 的标题状态匹配规则：

- 标题中出现 action-required 文本时，表示需要用户确认。
- 标题中出现 spinner 帧，或者 working/thinking/waiting 文本时，表示 CLI 正在回复。
- 任何非空的项目标题样式文本表示空闲。

两个可执行文件都会保留现有的参数转发、ConPTY 包装、终端尺寸同步、仓库名展示和 Toast 通知行为。

## 文档和构建

更新 README，说明这个仓库维护两个独立命令。构建示例使用：

```powershell
go build -o gemini-cli-notify.exe ./cmd/gemini-cli-notify
go build -o codex-cli-notify.exe ./cmd/codex-cli-notify
```

模块路径暂时保持不变，避免引入额外的仓库迁移工作。

## 测试

将现有测试移动到共享包附近。继续覆盖：

- Gemini 和 Codex 标题匹配器。
- Windows 参数引用和转义。
- 两个工具的命令行构造。

重构后运行 `go test ./...`。
