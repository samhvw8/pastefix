---
type: Architecture
title: Output targets and clipboard
description: The five targets, how input and output pick stdin, stdout or the clipboard, and how rich text reaches the pasteboard.
tags: [output, clipboard, macos]
resource: clipboard.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T17:10:00Z'
---

# Targets

| Target | Output |
|---|---|
| `notes` | HTML table + RTF on the clipboard, box table as the plain-text fallback |
| `slack` | Box table in a `````` code block, cells wrapped at `--width` (default 40) |
| `list` | One block per row: bold first cell, then `• Header: value` bullets (`--bold` sets the marker) |
| `box` | Box table without the fence |
| `md` | Aligned Markdown table (`Column N` names when there is no header) |

# Input and output

- Input is stdin only when stdin is a FIFO or a regular file; otherwise the clipboard
  (`pbpaste`). Checking for a terminal instead hung forever under agent shells and
  Alfred, where stdin is an open non-terminal that never closes.
- Output goes to the clipboard when stdout is a terminal (`x/term`), or with `--copy`;
  otherwise it is printed. `--stdout` forces printing.
- The target may come first or after the flags (`tablefix --width 30 slack`). Clipboard
  use off macOS fails with a message that says to pipe instead.

# Rich-text clipboard

`textutil -convert rtf` turns the HTML table into RTF (a real `\trowd`/`\cell` table).
`clipboard.go` writes the RTF to a temp file, then a JXA script (`osascript -l JavaScript`) clears `NSPasteboard` and sets
`public.html`, `public.rtf` and `public.utf8-plain-text` in one go, so each app takes
the richest type it understands.
