# Split CLI Entrypoints Design

## Context

The repository currently contains a mixed Gemini/Codex implementation on the `codex-notify` branch. The old `origin/main` remains Gemini-only, while the current local `main` and `codex-notify` point at a commit that supports multiple CLI tools from one `main.go`.

The desired direction is to keep one repository, but expose two independent binaries:

- `gemini-cli-notify.exe`
- `codex-cli-notify.exe`

The shared Windows terminal wrapper and notification logic should be maintained once.

## Design

Move the common implementation into an internal package and create two command entrypoints:

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

Each command entrypoint will pass a fixed tool configuration into the shared package. There will be no runtime `--tool` switch and no executable-name guessing in the final user path.

## Behavior

`gemini-cli-notify` will always launch `gemini` and use Gemini title-state matching:

- `✋` means confirmation is needed.
- `✦` or `⏲` means the CLI is responding.
- `◇` means idle.

`codex-cli-notify` will always launch `codex` and use Codex title-state matching:

- Action-required title text means confirmation is needed.
- Spinner frames or starting/working/thinking/waiting title text means the CLI is responding.
- Ready or any non-empty project-style title means idle.

Both binaries will keep existing argument forwarding, ConPTY wrapping, terminal resize handling, repo-name display, and Toast notification behavior.

## Documentation And Build

Update README files to describe the two binaries as separate commands from one repository. Build examples should use:

```powershell
go build -o gemini-cli-notify.exe ./cmd/gemini-cli-notify
go build -o codex-cli-notify.exe ./cmd/codex-cli-notify
```

The module path can remain unchanged for now to avoid extra repository migration work.

## Testing

Move existing tests around the shared package. Keep coverage for:

- Gemini and Codex title matchers.
- Windows argument quoting.
- Command-line construction for both tools.

Run `go test ./...` after the refactor.
