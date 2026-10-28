---
type: Decision
title: Rename tablefix to pastefix
description: Why the tool was renamed on 2026-10-27 and how the rename keeps existing installs working.
tags: [decision, naming]
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-27T15:00:00Z'
---

# Decision

On 2026-10-27 the user renamed `tablefix` to `pastefix`, in the same change that added
settings, an API-key route and prompt shortcuts. With free AI instructions the tool
rewrites what you paste, not only the table's drawing.

Checked the same day: no GitHub repo is named exactly `pastefix` (`tablefix`: 9,
`clipfix`: 13), and Homebrew has no formula with that name.

# What changed

- Go module `github.com/samhvw8/pastefix`, binary `pastefix`, env `PASTEFIX_CLAUDE`.
- Alfred bundle id `com.github.samhvw8.pastefix`. `alfred install` also matches the old
  id `com.github.samhvw8.tablefix`, so it updates the existing workflow in place
  instead of adding a second one. The keyword stayed `tf` at first; it is now `pf` (2026-10-28).
- History in `log.md` keeps the old name.
