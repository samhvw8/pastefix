---
type: Architecture
title: Settings and prompt shortcuts
description: Where pastefix keeps its settings (a JSON file, plus the API key in the macOS keychain), the config and prompt commands, and the Alfred / commands that call the same code.
tags: [config, ai, alfred]
resource: config.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-27T15:00:00Z'
---

# Where settings live

| Setting | Stored in | Why there |
|---|---|---|
| `model` | `~/Library/Application Support/pastefix/config.json` (`os.UserConfigDir`) | Plain preference |
| `prompts` (shortcuts) | same file | Plain preference, easy to edit by hand |
| `apikey` | login keychain, service `pastefix`, account `anthropic-api-key` (`/usr/bin/security`) | A secret: never in a file, never printed in full |

`PASTEFIX_CONFIG` overrides the file path. A missing file is an empty config; a
file that exists but is not valid JSON is an error, so a broken file is never
silently ignored. The file is written to a temp file and renamed.

`ANTHROPIC_API_KEY` in the shell is not used: a key set for another tool would
silently move every AI call to metered API billing.

# Commands

The CLI and Alfred both end in `configCommand`, so there is one implementation.

| CLI | Alfred | Does |
|---|---|---|
| `pastefix config` | `/config` | Lists file path, API key (masked `sk-ant-…WXYZ`) and route, model, claude path, prompts |
| `pastefix config set apikey KEY` | `/set apikey KEY` | Saves the key in the keychain; the AI step now calls the Anthropic API |
| `pastefix config unset apikey` | `/unset apikey` | Deletes it; the AI step uses the claude login again |
| `pastefix config set model NAME` | `/set model NAME` | `haiku`, `sonnet`, `opus` or a model id |
| `pastefix config unset model` | `/unset model` | Back to `haiku` |
| `pastefix prompt add NAME TEXT…` | `/prompt add NAME TEXT` | Saves a shortcut |
| `pastefix prompt rm NAME` | `/prompt rm NAME` | Removes it |

Model precedence: `--ai-model`, then `config.model`, then `haiku`.

# Shortcuts

A shortcut is a name for an instruction: `vi` = `translate to Vietnamese`. Then
`pastefix list --ai vi` or Alfred `pf slack vi` runs the text. `Enhance` looks the
name up; the copy message names the shortcut (`AI step: vi`).

A shortcut name must be one word, must not start with `/`, and must not be a target
name or the built-in `clean`: each of those would make a query ambiguous. A shortcut
runs as a free instruction, so it has no same-shape check (only `clean` has one; see
[AI step](/architecture/ai-step.md)).

In Alfred, an empty query lists the shortcuts under the targets; Tab puts the name in
the query ([Alfred workflow](/architecture/alfred-workflow.md)).
