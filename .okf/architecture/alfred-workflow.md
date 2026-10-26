---
type: Architecture
title: Alfred workflow
description: How `tablefix alfred install` writes the tf workflow, why it calls the binary by absolute path, and how reinstall finds it.
tags: [alfred]
resource: alfred.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T17:10:00Z'
---

# Commands

| Command | Does |
|---|---|
| `tablefix alfred install [--bin P] [--keyword tf]` | Writes `info.plist` into Alfred's workflows folder, or updates the installed one |
| `tablefix alfred uninstall` | Removes the folder whose plist has the tablefix bundle id |
| `tablefix alfred export [-o F]` | Writes a `.alfredworkflow` zip to import by hand |
| `tablefix alfred filter` | The Script Filter itself: prints Alfred JSON |

# Flow

Script Filter (keyword, `<bin> alfred filter`) → Run Script (`/bin/bash`, argv) → Post
Notification (`{query}`).

- `alfred filter` parses the clipboard. With a table it lists the five targets with the
  row count; without one it shows one invalid item with the error, so Enter does nothing.
- Run Script: `<bin> "$1" --copy < /dev/null 2>&1 | tail -1`. `--copy` is needed because
  stdout is not a terminal; `< /dev/null` makes it read the clipboard. The last line
  (success or error) becomes the notification.

# Binary path

Alfred runs scripts without the user's shell `PATH`, so the plist holds an absolute,
shell-quoted path. `--bin` sets it; otherwise `tablefix` found on `PATH` (with mise activated
that is `…/installs/go-github-com-samhvw8-tablefix/latest/bin/tablefix`, whose `latest`
link mise moves on upgrade; or `~/go/bin`), else the running binary. Moving the binary
means running `alfred install` again.

# Finding the workflow

Alfred's folder is `~/Library/Application Support/Alfred/Alfred.alfredpreferences/workflows`,
or the sync folder from `defaults read com.runningwithcrayons.Alfred-Preferences syncfolder`.
Install and uninstall match a folder by bundle id `com.github.samhvw8.tablefix` (read
with `plutil`, so binary plists work too). The object uids are fixed, so a reinstall
changes only what differs. Install also deletes `filter.py` and `tablefix.py` left by
the Python-era workflow.
