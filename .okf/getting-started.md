---
type: Reference
title: Getting started — tablefix
description: What tablefix is, how its parts fit, and where each piece of knowledge lives.
tags: [getting-started]
status: stable
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T17:10:00Z'
---

# Overview

tablefix turns a text table (Markdown pipes or box-drawing lines) into a form that
pastes well in the target app: a rich-text table for Apple Notes and other editors, a
code-block box table for Slack, or a list for chat boxes. It is one Go binary (a
`main` package at the repo root) and installs itself into Alfred as a workflow.
It began as a Python script; see [the Go rewrite](/decisions/go-rewrite.md).

It was built after an AI-made box table pasted into Apple Notes showed up as rows of
pipes with drawn lines between them (Notes renders `├───┼───┤` as strokes).

# Layout

| Path | What it is |
|---|---|
| `table.go` | `Parse` and the renderers |
| `clipboard.go` | macOS clipboard read and rich-text write |
| `alfred.go` | `tablefix alfred …`: writes, finds and removes the workflow |
| `main.go` | CLI flags and the stdin/stdout/clipboard rules |
| `tablefix_test.go` | Go tests |

# Concepts

- [Parsing](/architecture/parsing.md): which tables are read, how rows and the header are found.
- [Output and clipboard](/architecture/output.md): the five targets and how the clipboard is written.
- [Alfred workflow](/architecture/alfred-workflow.md): the self-contained workflow and its release.
- [One target per destination app](/decisions/targets-per-app.md): why Slack gets a code block or a list.
- [Go rewrite](/decisions/go-rewrite.md): why Go, and why the CLI installs the workflow.
