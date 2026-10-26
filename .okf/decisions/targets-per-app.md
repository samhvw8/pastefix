---
type: Decision
title: One target per destination app
description: Why tablefix writes rich text for Notes but a code block or list for Slack and chat apps.
tags: [decision, slack, apple-notes]
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T16:30:00Z'
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
---

# Decision

- **Apple Notes and other editors:** put rich text (HTML + RTF) on the clipboard. A
  pasted rich-text table becomes a native Notes table.[^notes-rich-paste]
- **Slack:** messages do not render Markdown tables, and a typed message cannot use
  tables in mrkdwn or standard Block Kit.[^slack-no-tables] The table block exists only
  for app-posted messages.[^slack-table-block] So `slack` is a monospace box table in a
  code block, and `list` is the option that reads well on a phone.
- **Chat apps** (Messenger, Zalo, iMessage) use proportional fonts and no code blocks,
  so only `list` stays readable.

# Not verified

The ⌘V into Apple Notes itself was not run end to end; the clipboard was checked to
hold an RTF table (5 `\trowd`, 15 `\cell` for a 5×3 table).

[^notes-rich-paste]: r/shortcuts thread: rich text from HTML, copied to the clipboard, pastes as a table.
[^slack-no-tables]: markdowntools.com Slack formatting guide.
[^slack-table-block]: Slack developer docs, table block.
