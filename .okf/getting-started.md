---
type: Reference
title: Getting started — pastefix
description: What pastefix is, how its parts fit, and where each piece of knowledge lives.
tags: [getting-started]
status: stable
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-27T15:00:00Z'
---

# Overview

pastefix turns a text table (Markdown pipes or box-drawing lines) into a form that
pastes well in the target app: a rich-text table for Apple Notes and other editors, a
code-block box table for Slack, or a list for chat boxes. An optional AI step (`--ai`)
then lets a model improve the text, while code still draws the table. It is one Go binary (a
`main` package at the repo root) and installs itself into Alfred as a workflow.
It began as a Python script named tablefix; see [the Go rewrite](/decisions/go-rewrite.md)
and [the rename](/decisions/rename-pastefix.md).

It was built after an AI-made box table pasted into Apple Notes showed up as rows of
pipes with drawn lines between them (Notes renders `├───┼───┤` as strokes).

# Layout

| Path | What it is |
|---|---|
| `table.go` | `Parse` and the renderers |
| `clipboard.go` | macOS clipboard read and rich-text write |
| `ai.go` | The optional `--ai` step: Anthropic API with a key, else `claude -p`; checks the returned shape |
| `config.go` | Settings file, API key in the keychain, `config` and `prompt` commands |
| `alfred.go` | `pastefix alfred …`: writes, finds and removes the workflow |
| `usage.go` | Temporary usage log, `pastefix stats`, the last-target default |
| `main.go` | CLI flags and the stdin/stdout/clipboard rules |
| `pastefix_test.go`, `ai_test.go`, `config_test.go`, `usage_test.go` | Go tests (fake runner, fake keychain, local API server) |

# Concepts

- [Parsing](/architecture/parsing.md): which tables are read, how rows and the header are found.
- [Output and clipboard](/architecture/output.md): the five targets and how the clipboard is written.
- [AI step](/architecture/ai-step.md): `--ai clean` or a free instruction; the model edits data, code draws.
- [Alfred workflow](/architecture/alfred-workflow.md): how the CLI writes the `pf` workflow; ⌥-Enter adds the AI step.
- [Settings and prompt shortcuts](/architecture/config.md): config file, API key in the keychain, `/` commands in Alfred.
- [One target per destination app](/decisions/targets-per-app.md): why Slack gets a code block or a list.
- [Go rewrite](/decisions/go-rewrite.md): why Go, and why the CLI installs the workflow.
- [Rename to pastefix](/decisions/rename-pastefix.md): why, and how the old Alfred workflow is updated in place.
