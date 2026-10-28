---
type: Architecture
title: Alfred workflow
description: How `pastefix alfred install` writes the pf workflow, why it calls the binary by absolute path, and how reinstall finds it.
tags: [alfred]
resource: alfred.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-27T15:00:00Z'
---

# Commands

| Command | Does |
|---|---|
| `pastefix alfred install [--bin P] [--keyword pf]` | Writes `info.plist` into Alfred's workflows folder, or updates the installed one |
| `pastefix alfred uninstall` | Removes the folder whose plist has the pastefix bundle id |
| `pastefix alfred export [-o F]` | Writes a `.alfredworkflow` zip to import by hand |
| `pastefix alfred filter [query]` | The Script Filter itself: prints Alfred JSON |
| `pastefix alfred run ARG` | The Run Script's work: runs a /command, or a target with `$ai` |

# Flow

Script Filter (keyword, `PASTEFIX_CLAUDE='<claude>' <bin> alfred filter "$1"`) → Run
Script (`/bin/bash`, argv) → Post Notification (`{query}`).

- `alfred filter` parses the clipboard. With a table it lists the five targets with the
  row count; without one it shows one invalid item with the error, so Enter does nothing.
- The filter gets the query and matches it itself (`alfredfiltersresults` is off, or
  Alfred would hide every item when the query holds an instruction). `splitQuery` reads it:
  - empty: all targets;
  - one word that starts a target name (`no`): those targets, no AI step;
  - a target name, then words (`slack translate to Vietnamese`): that target, and the
    words become the item's `ai` variable, so Enter runs them as the instruction;
  - anything else (`shorten each cell`): all targets, each with the whole query as `ai`.
  - A shortcut name in the `ai` position (`pf slack vi`) shows its text in the subtitle.
  - With an empty query, the shortcuts follow the targets as invalid items; Tab
    completes the name ([shortcuts](/architecture/config.md)).
- A query that starts with `/` is a settings command (`/config`, `/set apikey KEY`,
  `/prompt add NAME TEXT`, …). An incomplete command lists the matching commands; a
  complete one is one valid item whose arg is the command, with the key masked in the
  title. `/config` lists the settings as rows.
- Each item has an `alt` mod that sets the variable `ai=clean`. The filter has two
  connections to the Run Script (no modifier, and ⌥ = `524288`), so ⌥-Enter runs the
  same target with the [AI step](/architecture/ai-step.md).
- Run Script: `PASTEFIX_CLAUDE='<claude>' <bin> alfred run "$1" < /dev/null 2>&1 | tail -1`.
  `alfred run` reads `$ai` itself, so no shell expansion builds flags. (The earlier
  `${ai:+--ai "$ai"}` split in bash but not in zsh.) A `/` arg runs the command; any
  other arg runs the target with `--copy` (stdout is not a terminal) and `--ai $ai`
  when set. `< /dev/null` makes it read the clipboard. The last line (success or
  error) becomes the notification.
- Both scripts get `PASTEFIX_CLAUDE`: the run script to call claude, the filter so
  `/config` shows the real path. Without claude at install time it is left out; only
  the claude route then fails, with a clear notification.

# Binary path

Alfred runs scripts without the user's shell `PATH`, so the plist holds an absolute,
shell-quoted path. `--bin` sets it; otherwise `pastefix` found on `PATH` (with mise activated
that is `…/installs/go-github-com-samhvw8-pastefix/latest/bin/pastefix`, whose `latest`
link mise moves on upgrade; or `~/go/bin`), else the running binary. Moving the binary
means running `alfred install` again.

# Finding the workflow

Alfred's folder is `~/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows`,
or the sync folder from `defaults read com.runningwithcrayons.Alfred-Preferences syncfolder`.
Install and uninstall match a folder by bundle id `com.github.samhvw8.pastefix`, or the
pre-rename `com.github.samhvw8.tablefix` ([rename](/decisions/rename-pastefix.md)) (read
with `plutil`, so binary plists work too). The object uids are fixed, so a reinstall
changes only what differs. Install also deletes `filter.py` and `tablefix.py` left by
the Python-era workflow.
