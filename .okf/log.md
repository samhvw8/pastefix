# Update Log

## 2026-10-29
* **Fix**: `claude -p` runs with `--no-session-persistence`. Before, every AI step run without an API key left a session folder in `~/.claude/projects` ([AI step](/architecture/ai-step.md)). The repo is now samhvw8/pastefix and the folder `~/workspace/tools/pastefix`. Version 0.5.2.

## 2026-10-28
* **Update**: The default Alfred keyword is now `pf` (was `tf`). `pastefix alfred install` changes it on the installed workflow; `--keyword` still picks another. Version 0.5.1.

## 2026-10-27
* **Update**: Renamed tablefix to pastefix: module, binary, `PASTEFIX_CLAUDE`, Alfred bundle id; install still finds the old workflow ([rename](/decisions/rename-pastefix.md)).
* **Addition**: Settings and prompt shortcuts: `pastefix config [set|unset]`, `pastefix prompt add|rm`, Alfred `/config`, `/set`, `/unset`, `/prompt`; the API key goes in the keychain ([settings](/architecture/config.md)). With a key, the AI step calls the Anthropic API through the Go SDK instead of `claude -p` ([AI step](/architecture/ai-step.md)). The Alfred run script is now `pastefix alfred run "$1"`. Version 0.5.0.
* **Addition**: Alfred takes an AI instruction in the query: `tf slack translate to Vietnamese` (one target) or `tf shorten each cell` (all targets). The filter now matches the query itself ([Alfred workflow](/architecture/alfred-workflow.md)). The run script passes `--ai="$ai"` as one word, so it works in zsh too. Version 0.4.0.
* **Addition**: Optional AI step after the code fix: `--ai clean` or `--ai "<instruction>"`, run through `claude -p` with a JSON schema; the model edits data, code draws ([AI step](/architecture/ai-step.md)). Alfred: ⌥-Enter on any target adds `--ai clean` ([Alfred workflow](/architecture/alfred-workflow.md)). Found while testing in a bare environment: claude needs `$USER` to find its login. Version 0.3.0.

## 2026-10-26
* **Fix**: Under Alfred (no `LANG`), `pbpaste` returned `?` for `│`, so box tables read as "no table". macOS tools now run by absolute path with a UTF-8 locale ([output](/architecture/output.md)); `mise run test:clipboard` covers it.
* **Update**: Rewritten in Go ([decision](/decisions/go-rewrite.md)). The Python package, its pytest suite and the bundled-copy workflow build are gone; `tablefix alfred install|uninstall|export|filter` now writes the workflow, which calls the binary by absolute path ([Alfred workflow](/architecture/alfred-workflow.md)). Wrapping no longer splits at hyphens ([parsing](/architecture/parsing.md)). 19 Go tests.
* **Addition**: Project adopted from the `tablefix` script in dotfiles and an Alfred workflow built the same day. Packaged as a stdlib-only module with a `tablefix` entry point; the Alfred workflow now bundles its own copy and runs on `/usr/bin/python3`. Tests run on Python 3.14 and 3.9. Concepts: [parsing](/architecture/parsing.md), [output](/architecture/output.md), [Alfred workflow](/architecture/alfred-workflow.md), [targets per app](/decisions/targets-per-app.md).
