---
type: Decision
title: One target per destination app
description: Why pastefix writes rich text for Notes but a code block or list for Slack and chat apps.
tags: [decision, slack, apple-notes]
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-29T14:10:00Z'
verified:
  - by: claude-code/claude-opus-5-5
    at: '2026-10-29T13:54:00Z'
sources:
  - id: slack-no-tables
    resource: https://www.markdowntools.com/blog/markdown-to-slack-formatting-guide
    title: "Markdown to Slack Formatting: What Actually Works"
  - id: slack-table-block
    resource: https://docs.slack.dev/reference/block-kit/blocks/table-block
    title: Table block — Slack Developer Docs
  - id: notes-rich-paste
    resource: https://www.reddit.com/r/shortcuts/comments/18beavc/adding_a_table_to_notes_through_shortcuts/
    title: Adding a table to notes through shortcuts (r/shortcuts)
  - id: slack-paste-table
    resource: https://slack.green/en/blog/how-to-make-a-table-in-slack
    title: "How to Make a Table in Slack: Paste, Canvas or Block Kit"
  - id: slack-pasted-table-block
    resource: https://github.com/kirodotdev/KiroCrew/pull/18786
    title: "fix(slack): keep pasted table blocks in the message text"
---

# Decision

- **Apple Notes and other editors:** put rich text (HTML + RTF) on the clipboard. A
  pasted rich-text table becomes a native Notes table.[^notes-rich-paste]
- **Slack:** a typed message cannot write a table in mrkdwn.[^slack-no-tables] Since the
  table block shipped,[^slack-table-block] Slack turns cells pasted from Excel or Sheets
  (tab-separated text) into a table block,[^slack-pasted-table-block] and one source says
  a pasted Markdown table works too.[^slack-paste-table] `slack` stays a monospace box
  table in a code block, because it shows the same in every client; `list` reads well
  on a phone. Revisit when the [host test](/architecture/usage-log.md) for Slack runs.
- **Chat apps** (Messenger, Zalo, iMessage) use proportional fonts and no code blocks,
  so only `list` stays readable.

# Tested on a real Mac (2026-10-29)

Apple Notes on macOS 27.0.1, ⌘V into a note: a raw pipe table and the same table as
tab-separated text both stay plain text; `pastefix notes` output becomes a real table.
So `notes` is still needed. Slack was not tested: the app on this Mac was signed out.

[^notes-rich-paste]: r/shortcuts thread: rich text from HTML, copied to the clipboard, pastes as a table.
[^slack-no-tables]: markdowntools.com Slack formatting guide.
[^slack-table-block]: Slack developer docs, table block.
[^slack-paste-table]: slack.green, tested 2026-10-06 and 2026-10-28 (one source; not repeated here).
[^slack-pasted-table-block]: KiroCrew PR #18786: Slack sends a pasted Excel or Sheets table as a table block.
