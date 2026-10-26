# Update Log

## 2026-10-26
* **Update**: Rewritten in Go ([decision](/decisions/go-rewrite.md)). The Python package, its pytest suite and the bundled-copy workflow build are gone; `tablefix alfred install|uninstall|export|filter` now writes the workflow, which calls the binary by absolute path ([Alfred workflow](/architecture/alfred-workflow.md)). Wrapping no longer splits at hyphens ([parsing](/architecture/parsing.md)). 19 Go tests.
* **Addition**: Project adopted from the `tablefix` script in dotfiles and an Alfred workflow built the same day. Packaged as a stdlib-only module with a `tablefix` entry point; the Alfred workflow now bundles its own copy and runs on `/usr/bin/python3`. Tests run on Python 3.14 and 3.9. Concepts: [parsing](/architecture/parsing.md), [output](/architecture/output.md), [Alfred workflow](/architecture/alfred-workflow.md), [targets per app](/decisions/targets-per-app.md).
