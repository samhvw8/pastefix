"""Build dist/tablefix.alfredworkflow: the alfred/ files plus a bundled copy of tablefix.

The bundled copy runs under macOS's /usr/bin/python3, so the workflow needs nothing installed.
"""

import plistlib
import re
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "src" / "tablefix" / "__init__.py"
OUT = ROOT / "dist" / "tablefix.alfredworkflow"


def main():
    code = SRC.read_text()
    version = re.search(r'__version__ = "([^"]+)"', code).group(1)
    plist = plistlib.loads((ROOT / "alfred" / "info.plist").read_bytes())
    plist["version"] = version
    OUT.parent.mkdir(exist_ok=True)
    with zipfile.ZipFile(OUT, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("info.plist", plistlib.dumps(plist))
        for name, text in [("filter.py", (ROOT / "alfred" / "filter.py").read_text()), ("tablefix.py", code)]:
            info = zipfile.ZipInfo(name)
            info.external_attr = 0o755 << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, text)
    print(f"{OUT} ({version})", file=sys.stderr)


if __name__ == "__main__":
    main()
