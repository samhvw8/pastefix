# Update Log

## 2026-10-26
* **Addition**: Project adopted from the `tablefix` script in dotfiles and an Alfred workflow built the same day. Packaged as a stdlib-only module with a `tablefix` entry point; the Alfred workflow now bundles its own copy and runs on `/usr/bin/python3`. Tests run on Python 3.14 and 3.9. Concepts: [parsing](/architecture/parsing.md), [output](/architecture/output.md), [Alfred workflow](/architecture/alfred-workflow.md), [targets per app](/decisions/targets-per-app.md).
