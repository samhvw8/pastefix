#!/usr/bin/python3
"""Alfred Script Filter: list tablefix targets and check the clipboard first."""
import json
import os
import subprocess

# The workflow bundles its own copy of tablefix, so it needs nothing installed.
TOOL = os.path.join(os.path.dirname(os.path.abspath(__file__)), "tablefix.py")
TARGETS = [
    ("notes", "Apple Notes / Pages / Mail / Google Docs", "Real table (rich text)"),
    ("slack", "Slack", "Aligned table in a code block"),
    ("list", "Chat box (Messenger, Zalo, iMessage)", "One block per row with bullets"),
    ("md", "Markdown", "Clean, aligned pipe table"),
    ("box", "Plain text box", "Box table without code fence"),
]

with open(os.devnull) as null:
    r = subprocess.run(["/usr/bin/python3", TOOL, "md", "--stdout"], stdin=null, capture_output=True, text=True)

if r.returncode != 0:
    items = [{"title": "No table on the clipboard", "subtitle": (r.stderr.strip() or "tablefix failed")[:200], "valid": False}]
else:
    rows = max(len(r.stdout.strip().splitlines()) - 2, 0)
    items = [{"uid": t, "title": f"{t} — {name}", "subtitle": f"{desc} · {rows} rows on the clipboard",
              "arg": t, "match": f"{t} {name}", "autocomplete": t} for t, name, desc in TARGETS]
print(json.dumps({"items": items}))
