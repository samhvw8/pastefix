---
type: Architecture
title: Parsing text tables
description: How tablefix reads Markdown pipe tables and box-drawing tables, joins wrapped cells and detects the header.
tags: [parser]
resource: src/tablefix/__init__.py
generated:
  by: claude-code/claude-opus-5-5
  at: '2026-10-26T16:30:00Z'
---

# Rules

- `│`, `┃` and `║` are read as `|`. A line with `|` and some text is a data line.
  `\|` stays a literal pipe inside a cell.
- A **rule line** has only rule characters (`─━═=-+:|` and the box corners and joints).
  It is a box rule if it has a box character, `=` or `+`; a Markdown separator if it
  has only `-`, `:` and `|`.
- **Box mode** (any box rule present): the data lines between two rules form one row,
  and their cell fragments are joined with a space. This rejoins cells that a terminal
  or tablefix's own `--width` wrapped. In Markdown mode each data line is a row.
- Short rows are padded with empty cells to the widest row.
- **Header:** the first row is a header when the only rule between rows is after row 1,
  or when the rule after row 1 is heavy (`═`, `━`, `=`). Rules after every row (the
  Apple Notes case) mean no header. A two-row table with one rule is read as having a
  header; `--no-header` overrides, and `--header "A,B"` names the columns.
- No row found, or a `--header` count that differs from the column count, exits 1
  with a message and writes nothing.

The box renderer draws the header rule as `╞═╪═╡`, so its own output parses back to
the same header and rows (tested).

Column widths count East Asian wide characters as 2 and combining marks as 0.
