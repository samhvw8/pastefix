# Agent guide: tablefix

A public CLI (github.com/samhvw8/tablefix) that turns a Markdown or box-drawing text
table into a rich-text table for Apple Notes, a code-block table for Slack, or a list
for chat boxes. It also ships as an Alfred workflow (keyword `tf`).

Project knowledge lives in `.okf/`. Update it in the same change as the code it
describes, and run the validator. Start at `.okf/index.md`.

## Layout

One `main` package at the root, so `go install github.com/samhvw8/tablefix@latest` works.

| Path | What it is |
|---|---|
| `table.go` | `Parse` and the renderers (`Box`, `Markdown`, `List`, `HTML`) |
| `clipboard.go` | macOS clipboard: `pbpaste`, and HTML + RTF + text via `textutil` and JXA |
| `alfred.go` | `tablefix alfred install/uninstall/export/filter`: writes the workflow plist |
| `main.go` | Flags, target choice, stdin/stdout/clipboard rules |
| `tablefix_test.go` | Tests (parser, renderers, CLI with fake clipboard, Alfred plist) |

## Commands

```bash
mise run test       # go test -race
mise run check      # gofmt, vet (macOS + Linux), tests, docs:validate
mise run build      # bin/tablefix
mise run release    # check, then gh release v<version> with 4 binaries
```

## Rules

- **Fail loudly.** No table or a bad `--header` exits 1 and leaves the clipboard alone.
- **Bump `version`** in `main.go` before `mise run release`.
- **The workflow calls the binary by absolute path.** Alfred has no shell `PATH`.
  Change the workflow only in `alfred.go`'s template, and keep the bundle id, so
  `alfred install` keeps updating the same folder.
- The user's machine installs it with `go:github.com/samhvw8/tablefix` in
  `~/.dotfiles/mise/config.toml`; `mise install -f go:github.com/samhvw8/tablefix`
  picks up a new commit, then `tablefix alfred install` if the path changed.
