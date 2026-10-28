# pastefix

*Formerly `tablefix`.*

Copy a text table from ChatGPT, Claude or a terminal, and paste it as a table that
looks right where it lands.

Pasting a Markdown table (`| a | b |`) or a box-drawing table (`│ a │ b │`) into
Apple Notes, Slack or a chat box gives you a mess of pipes and lines. `pastefix`
reads the table from the clipboard and puts a better version back:

| Target | Where to paste | What you get |
|---|---|---|
| `notes` | Apple Notes, Pages, Mail, Google Docs, Notion, Word | A real table (rich text on the clipboard) |
| `slack` | Slack | An aligned box table in a code block |
| `list` | Messenger, Zalo, iMessage, WhatsApp, Slack | One block per row, with bullets |
| `md` | GitHub, Obsidian, anything Markdown | A clean, aligned Markdown table |
| `box` | Terminals, plain-text email | A box table without the code fence |

Slack messages cannot show tables at all, in mrkdwn or in Block Kit, so `slack` and
`list` are the two good options there.

## Install

A single Go binary with no runtime dependencies:

```bash
go install github.com/samhvw8/pastefix@latest
# or, with mise
mise use -g go:github.com/samhvw8/pastefix
```

Clipboard use is macOS only (`pbpaste`, `osascript`, `textutil`). On Linux, pipe text
in and out: `pastefix md < table.txt`.

### Alfred workflow

The CLI writes the workflow into Alfred for you, pointing at its own path (Alfred
runs scripts without your shell `PATH`). It needs the Alfred Powerpack.

```bash
pastefix alfred install            # or --keyword tbl, --bin /path/to/pastefix
pastefix alfred uninstall
pastefix alfred export -o pastefix.alfredworkflow   # a file to import by hand
```

Running `install` again updates the same workflow in place. Then:

1. Copy a table.
2. Type `pf` in Alfred. Each target shows how many rows it found on the clipboard.
3. Choose a target and press Enter.
4. Paste with ⌘V.

Press ⌥-Enter instead of Enter to add the [AI step](#ai-step-optional).

To rewrite the table with your own instruction, type it after `pf`:

| You type | Result |
|---|---|
| `pf slack translate to Vietnamese` | Slack only; Enter runs the instruction |
| `pf shorten each cell` | Every target; Enter runs the instruction |
| `pf slack vi` | Runs your saved shortcut `vi` (see [Settings](#settings-and-shortcuts)) |
| `pf no` | `notes`, with no AI step |
| `pf /` | Settings commands: `/config`, `/set apikey KEY`, `/prompt add NAME TEXT`, … |

If you move the binary, run `pastefix alfred install` again.

## Use

```bash
pastefix notes                                 # clipboard -> clipboard, then ⌘V in Notes
pastefix slack --width 30                      # wrap long cells at 30 characters
pastefix list --header "Field,Journals,Magazines"
pastefix list --bold "**"                      # Markdown bold instead of Slack *bold*
pbpaste | pastefix md > table.md               # piped: prints instead of copying
```

- **Input:** stdin when it is a pipe or a file, else the clipboard.
- **Output:** the clipboard when run in a terminal (with a preview), else stdout.
  `--copy` forces the clipboard; `--stdout` forces printing.
- **Header:** detected when only the first row has a rule under it, or a heavy one
  (`═`, `━`, `+===+`). Rules after every row mean there is no header. Override with
  `--header "A,B,C"` or `--no-header`.
- **Wrapped cells:** in a box table, the lines between two rules are one row, so
  cells wrapped over several lines are joined again. `--width` wraps at spaces only,
  so a re-read gives back the same text.
- No table found, or a `--header` with the wrong number of names, exits with an
  error and changes nothing.

## AI step (optional)

After the table is fixed in code, a model can improve its text. The code still draws
the table, so the model cannot break it. It runs one of two ways:

- **No API key (default):** through the [Claude Code](https://claude.com/claude-code)
  CLI on your Claude login. No extra cost; about 5 seconds.
- **With an API key** (`pastefix config set apikey KEY`): directly on the Anthropic
  API. Billed to the key; Haiku costs well under a cent a run.

```bash
pastefix notes --ai clean                        # join wrapped words, fix typos, add a missing header
pastefix list --ai "translate to Vietnamese"     # any other text is the instruction
pastefix list --ai vi                            # a saved shortcut
pastefix md --ai clean --ai-model sonnet         # default model: haiku, or the config model
```

`clean` keeps every row, column and fact; pastefix rejects an answer with a different
shape. In Alfred, press ⌥-Enter instead of Enter, or type an instruction after `pf`. One run takes about 5 seconds. If
the AI step fails (no `claude`, not logged in, a bad answer), pastefix says why and
copies the table fixed in code.

```
| Column 1 | Column 2                                 |     →     | Name  | Venue                                        |
| TOCHI    | ACM Computer- Human Interac tion journal |  --ai     | TOCHI | ACM Computer-Human Interaction journal       |
| PSPI     | Psychological Science in th e Public ... |  clean    | PSPI  | Psychological Science in the Public Interest |
```

## Settings and shortcuts

```bash
pastefix config                                  # list every setting
pastefix config set apikey sk-ant-...            # stored in the macOS keychain, never in a file
pastefix config unset apikey                     # back to your Claude login
pastefix config set model sonnet                 # haiku, sonnet, opus or a model id
pastefix prompt add vi translate to Vietnamese   # shortcut: --ai vi, or pf slack vi
pastefix prompt rm vi
```

In Alfred the same commands start with `/`: `pf /config`, `pf /set apikey sk-ant-...`,
`pf /prompt add vi translate to Vietnamese`. With an empty query, Alfred lists your
shortcuts under the targets. Settings live in
`~/Library/Application Support/pastefix/config.json`.

## Examples

`pastefix slack --width 24`:

```
┌──────────┬────────────────────────┬─────────────────────────┐
│ Field    │ Journals               │ Magazines               │
╞══════════╪════════════════════════╪═════════════════════════╡
│ Learning │ PSPI, Educational      │ Behavioral Scientist    │
│          │ Psychology Review      │                         │
├──────────┼────────────────────────┼─────────────────────────┤
│ Habits   │ Nature Human Behaviour │ Harvard Business Review │
└──────────┴────────────────────────┴─────────────────────────┘
```

`pastefix list`:

```
*Learning*
• Journals: PSPI, Educational Psychology Review
• Magazines: Behavioral Scientist

*Habits*
• Journals: Nature Human Behaviour
• Magazines: Harvard Business Review
```

## Develop

```bash
mise run test      # go test -race
mise run check     # gofmt, vet (macOS + Linux), tests, docs validation
mise run build     # bin/pastefix
mise run release   # tag and publish macOS and Linux binaries
```

## License

MIT
