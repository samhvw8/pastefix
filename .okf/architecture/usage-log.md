---
type: Component
title: Usage log, stats and last target
description: The temporary usage log that decides which targets and commands to cut, pastefix stats, the host tests, and the last-target default.
tags: [usage, stats, cuts]
resource: usage.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-29T14:10:00Z'
status: draft
sources:
  - id: alfred-uid
    resource: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
    title: Script Filter JSON Format - Alfred Help
stale_after: 2026-11-28
---

# Why

Nothing recorded which targets were used: shell history had no pastefix lines, and
Alfred runs skip the shell. Cuts (targets, `config`, the box parser) need real numbers.

# Usage log

- `usage.jsonl` next to the config file (so `PASTEFIX_CONFIG` in tests moves it too).
- One JSON line per conversion and per `config`/`prompt` command, from the CLI and from
  Alfred: `at`, `target`, `input` (pipe, box, wrapped, none = no table), `exit`, `via`
  (cli, alfred), `ai` (clean, a shortcut name, or custom; never the free instruction
  text), `ai_fallback`.
- `--help` and `--version` are not logged. A failed write prints a warning and never
  fails the run.
- It stops writing after `usageLogEnd` (2026-11-28). Then `pastefix stats` says the log
  ended and asks for the cut decision; delete the logger in `usage.go` after that.

# stats

`pastefix stats` prints runs, failures and the last date per target and command, the
input kinds, cli vs alfred, and AI fallbacks. Zero runs and zero failures: "candidate to
cut". Failures only: "fix or cut". It only reports; the owner cuts.

# Cut rule

Cut a target only when both signals say so: the log shows no use, and a host test shows
the app now renders a raw paste by itself. Host tests are in the README table "Host
tests" (fixed 3-line input, raw pipe and tab-separated, plus pastefix output as the
control). 2026-10-29: Notes still needs `notes`; Slack not tested (signed out).

# Last target

`pastefix` with no target reuses the last target copied to the clipboard
(`state.json`, `{"last": ...}`), else `notes`. Only a clipboard run with an explicit
target changes it. A piped run (a script) always defaults to `notes`. Alfred needs no
change: each target item has a fixed `uid`, and Alfred ranks items by use.[^alfred-uid]

[^alfred-uid]: Alfred help: a uid lets Alfred learn from the user's picks and order results by them.
