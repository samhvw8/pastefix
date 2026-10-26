# tablefix

Copy a text table from ChatGPT, Claude or a terminal, and paste it as a table that
looks right where it lands.

Pasting a Markdown table (`| a | b |`) or a box-drawing table (`│ a │ b │`) into
Apple Notes, Slack or a chat box gives you a mess of pipes and lines. `tablefix`
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
go install github.com/samhvw8/tablefix@latest
# or, with mise
mise use -g go:github.com/samhvw8/tablefix
```

Clipboard use is macOS only (`pbpaste`, `osascript`, `textutil`). On Linux, pipe text
in and out: `tablefix md < table.txt`.

### Alfred workflow

The CLI writes the workflow into Alfred for you, pointing at its own path (Alfred
runs scripts without your shell `PATH`). It needs the Alfred Powerpack.

```bash
tablefix alfred install            # or --keyword tbl, --bin /path/to/tablefix
tablefix alfred uninstall
tablefix alfred export -o tablefix.alfredworkflow   # a file to import by hand
```

Running `install` again updates the same workflow in place. Then:

1. Copy a table.
2. Type `tf` in Alfred. Each target shows how many rows it found on the clipboard.
3. Choose a target and press Enter.
4. Paste with ⌘V.

If you move the binary, run `tablefix alfred install` again.

## Use

```bash
tablefix notes                                 # clipboard -> clipboard, then ⌘V in Notes
tablefix slack --width 30                      # wrap long cells at 30 characters
tablefix list --header "Field,Journals,Magazines"
tablefix list --bold "**"                      # Markdown bold instead of Slack *bold*
pbpaste | tablefix md > table.md               # piped: prints instead of copying
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

## Examples

`tablefix slack --width 24`:

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

`tablefix list`:

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
mise run build     # bin/tablefix
mise run release   # tag and publish macOS and Linux binaries
```

## License

MIT
