# Agent guide: pastefix

A public CLI (github.com/samhvw8/pastefix) that turns a Markdown or box-drawing text
table into a rich-text table for Apple Notes, a code-block table for Slack, or a list
for chat boxes. It also ships as an Alfred workflow (keyword `pf`).

Project knowledge lives in `.okf/`. Update it in the same change as the code it
describes, and run the validator. Start at `.okf/index.md`.

## Layout

One `main` package at the root, so `go install github.com/samhvw8/pastefix@latest` works.

| Path | What it is |
|---|---|
| `table.go` | `Parse` and the renderers (`Box`, `Markdown`, `List`, `HTML`) |
| `clipboard.go` | macOS clipboard: `pbpaste`, and HTML + RTF + text via `textutil` and JXA |
| `ai.go` | Optional `--ai` step: Anthropic API (key set) or `claude -p`, JSON schema; the model edits data, code draws |
| `config.go` | Settings file, API key in the keychain, `config`/`prompt` commands (CLI and Alfred `/`) |
| `alfred.go` | `pastefix alfred install/uninstall/export/filter/run`: writes the workflow plist |
| `main.go` | Flags, target choice, stdin/stdout/clipboard rules |
| `pastefix_test.go` | Tests (parser, renderers, CLI with fake clipboard, Alfred plist) |
| `ai_test.go` | AI step tests with a fake runner (no model calls) |
| `config_test.go` | Settings, keychain (faked in `TestMain`), API request via a local server, Alfred `/` commands |

## Commands

```bash
mise run test       # go test -race
mise run check      # gofmt, vet (macOS + Linux), tests, docs:validate
mise run build      # bin/pastefix
mise run release    # check, then gh release v<version> with 4 binaries
```

## Rules

- **Fail loudly.** No table or a bad `--header` exits 1 and leaves the clipboard alone.
- **The AI step never draws.** The model returns `{header, rows}` JSON; code checks the
  shape and renders. A failed AI step prints why and falls back to the code fix.
  Tests never call the real model; check a prompt change by hand on a real table.
- **Tests never touch the real config or keychain.** `TestMain` sets `PASTEFIX_CONFIG`
  and a fake keychain: a real API key would make tests call the paid API.
- **Back up the clipboard before a test that writes it** (`pbpaste > backup`), and
  restore it after.
- **Bump `version`** in `main.go` before `mise run release`.
- **The workflow calls the binary by absolute path.** Alfred has no shell `PATH`.
  Change the workflow only in `alfred.go`'s template, and keep the bundle id (and the
  legacy id match), so `alfred install` keeps updating the same folder.
- The user's machine installs it with `go:github.com/samhvw8/pastefix` in
  `~/.dotfiles/mise/config.toml`. After a push, run
  `GOPROXY=direct mise upgrade go:github.com/samhvw8/pastefix`: the Go proxy serves a
  cached `@latest` for a while, so a plain `mise install -f` reinstalls the old commit.
  The workflow path (`…/latest/bin/pastefix`) does not change on upgrade.
