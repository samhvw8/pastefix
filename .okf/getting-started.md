---
type: Reference
title: Getting started — tablefix
description: What tablefix is, how its parts fit, and where each piece of knowledge lives.
tags: [getting-started]
status: stable
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T16:30:00Z'
---

# Overview

tablefix turns a text table (Markdown pipes or box-drawing lines) into a form that
pastes well in the target app: a rich-text table for Apple Notes and other editors, a
code-block box table for Slack, or a list for chat boxes. It is one stdlib-only Python
module, `src/tablefix/__init__.py`, with no dependencies, so it also runs on the macOS
system Python 3.9.

It was built after an AI-made box table pasted into Apple Notes showed up as rows of
pipes with drawn lines between them (Notes renders `├───┼───┤` as strokes).

# Layout

| Path | What it is |
|---|---|
| `src/tablefix/__init__.py` | Parser, renderers, clipboard writer, CLI (`main`) |
| `alfred/` | Alfred workflow source: `info.plist`, `filter.py` (Script Filter) |
| `scripts/build_alfred.py` | Zips `alfred/` plus the module as `tablefix.py` into `dist/tablefix.alfredworkflow` |
| `tests/` | pytest, run on the current Python and on 3.9 |

# Concepts

- [Parsing](/architecture/parsing.md): which tables are read, how rows and the header are found.
- [Output and clipboard](/architecture/output.md): the five targets and how the clipboard is written.
- [Alfred workflow](/architecture/alfred-workflow.md): the self-contained workflow and its release.
- [One target per destination app](/decisions/targets-per-app.md): why Slack gets a code block or a list.
