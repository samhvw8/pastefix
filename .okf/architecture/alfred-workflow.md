---
type: Architecture
title: Alfred workflow
description: The tf keyword workflow, why it bundles its own tablefix, and how it is built and released.
tags: [alfred, release]
resource: alfred/info.plist
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T16:30:00Z'
---

# Flow

Script Filter (keyword `tf`, `filter.py`) → Run Script (`/bin/bash`, argv) → Post
Notification (`{query}`).

- `filter.py` runs the bundled `tablefix.py md --stdout` on the clipboard first. With a
  table it lists the five targets with the row count; without one it shows one invalid
  item with the error, so nothing runs.
- Run Script: `/usr/bin/python3 ./tablefix.py "$1" --copy < /dev/null 2>&1 | tail -1`.
  `--copy` is needed because stdout is not a terminal; `< /dev/null` makes it read the
  clipboard. The last line (success or error) becomes the notification.

# Self-contained

Alfred runs scripts without the user's shell profile, so mise shims and `~/.local/bin`
are not on PATH. The workflow therefore ships its own copy of the module as
`tablefix.py` and runs it with `/usr/bin/python3` (3.9 on current macOS). This is why
the module must stay stdlib-only and 3.9-compatible; `mise run test` checks 3.9.

# Build and release

`mise run alfred` builds `dist/tablefix.alfredworkflow` and stamps the module's
`__version__` into the plist. `mise run release` runs `check`, then
`gh release create v<version>` with the workflow attached.
