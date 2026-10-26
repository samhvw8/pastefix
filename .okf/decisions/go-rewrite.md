---
type: Decision
title: Rewrite in Go, with the CLI installing the Alfred workflow
description: Why tablefix moved from a Python package to one Go binary that writes its own Alfred workflow.
tags: [decision, go, alfred]
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T17:10:00Z'
---

# Decision

On 2026-10-26, the user asked for a Go CLI that links the Alfred workflow itself.

- **One binary.** `go install` or mise's `go:` backend gives a single file with no
  interpreter. The Python version had to stay stdlib-only and 3.9-compatible because
  the workflow ran on `/usr/bin/python3`.
- **The CLI owns the workflow.** The Python workflow bundled a second copy of the
  code (built by a script). Now `tablefix alfred install` writes a plist that calls the
  installed binary, so there is one copy, and an upgrade needs no workflow rebuild.
- **Behavior kept.** On 120 sample runs (4 inputs × 5 targets × 6 flag sets), Go output
  matched Python byte for byte except 3 runs at `--width 12`, where Go no longer splits
  at a hyphen (see [parsing](/architecture/parsing.md)).

# Costs

Dependencies: `github.com/mattn/go-runewidth` (display width) and `golang.org/x/term`
(terminal check). Releases ship macOS and Linux binaries; installing from source needs Go.
