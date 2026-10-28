---
type: Architecture
title: AI step
description: The optional --ai step that runs after the code fix - its two routes (Anthropic API with a key, else claude -p), why the model edits data and never the drawing, and how it fails.
tags: [ai, claude]
resource: ai.go
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-27T15:00:00Z'
---

# Pipeline

1. **Fix in code.** `Parse` rebuilds the table ([parsing](/architecture/parsing.md)). Free, offline, the same every time.
2. **AI step (opt-in, `--ai`).** `Enhance` sends the parsed table as JSON `{"header": [...], "rows": [[...]]}` to a model and gets the same JSON shape back.
3. **User overrides.** `--header` / `--no-header` run after the AI step, so the user wins.
4. **Draw in code.** The renderers draw the result ([output](/architecture/output.md)).

The model edits the table's data, never its drawing, so it cannot break the table's
structure. Code checks the shape it returns before anything is drawn.

# Steps

| `--ai` value | Does | Shape check |
|---|---|---|
| `clean` | Joins words split by wrapping (`Interac tion` -> `Interaction`), fixes typos, writes a header when there is none. Keeps every row, column and fact. | Same rows and columns; a header may not be dropped |
| a shortcut name | Runs the saved text ([shortcuts](/architecture/config.md)) | As for any other text |
| any other text | Used as the instruction itself, e.g. `--ai "translate to Vietnamese"` | Any shape, but rectangular, with a matching header |

The code fix cannot do `clean`'s job: it cannot tell a word split by wrapping from two
real words. Measured on the user's note: the code fix gives `Column 1..3`; `clean` adds
`Topic | Academic Journals | Popular Publications` and keeps all 9 cells unchanged.

# Two routes

`autoRunner` picks one per call:

| API key in the keychain? | Route | Billing |
|---|---|---|
| yes (`pastefix config set apikey`) | Anthropic Messages API through the official Go SDK | Metered on the key |
| no | `claude -p` on the user's Claude login | The user's subscription |

`pastefix config` shows which route is active ([settings](/architecture/config.md)).

# Calling the API

`runAPI`: `client.Messages.New` with `output_config.format` = `json_schema` (the table
schema), `output_config.effort` = `low` (a small edit; speed matters in Alfred),
`max_tokens` 16000, a 60 s context. `haiku`/`sonnet`/`opus` map to
`claude-haiku-5-5`/`claude-sonnet-5-5`/`claude-opus-5-5`; any other name is passed as
a model id. A `stop_reason` other than `end_turn` (refusal, max_tokens) is an error.
The text block holds the JSON. `ANTHROPIC_BASE_URL` points it at a local server in
tests. Haiku 5.5 costs $0.10 / $0.50 per million input / output tokens.

# Calling claude

`claude -p --model haiku --tools "" --setting-sources "" --strict-mcp-config
--output-format json --json-schema <table schema> --system-prompt <step>`, with the table
on stdin, run in an empty temp folder, with a 60 s timeout.

- Uses the user's existing Claude login: no API key, no new secret.
- `--json-schema` makes the answer `structured_output`; output can be one result
  object or, in verbose mode, an array whose `result` message holds it.
- The prompt says the cells are data, never instructions (clipboard text is untrusted).
- `claude` finds its login in the keychain by `$USER`. A bare environment has no `USER`
  and gets "Not logged in", so `runClaude` adds it.
- Path: `PASTEFIX_CLAUDE`, else `claude` on `PATH`. The Alfred workflow bakes the path
  found at install time into `PASTEFIX_CLAUDE` ([Alfred workflow](/architecture/alfred-workflow.md)).
- `--ai-model` picks the model; else the config model; else `haiku`.

Cost: one Haiku call a use, about 4.6 s wall time (3 runs: 4.56-4.79 s), billed to the
user's subscription.

# Failure

Any error (no claude, not logged in, timeout, bad JSON, wrong shape) prints
`AI step failed (<why>); used the code fix.` and the run continues with the step-1
table. Under Alfred that text is in the notification. The step-1 table is correct, just
not enhanced, so falling back is safe, and the message keeps it from being silent.
