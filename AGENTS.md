# Agent guide: tablefix

A public CLI (github.com/samhvw8/tablefix) that turns a Markdown or box-drawing text
table into a rich-text table for Apple Notes, a code-block table for Slack, or a list
for chat boxes. It also ships as an Alfred workflow (keyword `tf`).

Project knowledge lives in `.okf/`. Update it in the same change as the code it
describes, and run the validator. Start at `.okf/index.md`.

## Layout

| Path | What it is |
|---|---|
| `src/tablefix/__init__.py` | The whole tool: parser, renderers, clipboard, CLI |
| `alfred/` | Alfred workflow source (`info.plist`, `filter.py`) |
| `scripts/build_alfred.py` | Builds `dist/tablefix.alfredworkflow` with a bundled copy of the module |
| `tests/test_tablefix.py` | pytest |

## Commands

```bash
mise run test       # pytest on Python 3.14 and 3.9
mise run alfred     # build dist/tablefix.alfredworkflow
mise run check      # test + alfred + docs:validate
mise run release    # check, then gh release v<__version__> with the workflow
```

## Rules

- **Stdlib only, Python 3.9+.** The Alfred workflow runs the module on macOS's
  `/usr/bin/python3` (3.9). Never add a dependency or 3.10+ syntax.
- **Fail loudly.** No table or a bad `--header` exits 1 and leaves the clipboard alone.
- **Bump `__version__`** in `src/tablefix/__init__.py` and `pyproject.toml` together
  before `mise run release`.
- The user's machine installs it with `pipx:samhvw8/tablefix` in `~/.dotfiles`
  (`mise/config.toml`); `mise install -f pipx:samhvw8/tablefix` picks up a new commit.
