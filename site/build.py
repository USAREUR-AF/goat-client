#!/usr/bin/env python3
"""Check the landing page in site/ against the tree and the recorded captures.

    python3 site/build.py

Fails when a number the page states differs from what it is measured from, or
when the code shown faintly behind the first screen is no longer in the tree.
The page reads site/captures/*.txt as they are. This also copies the drawn
figure in from docs/figures/, so the page does not keep a second copy of it.
"""
import re
import shutil
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / "site"


def measured():
    cli = (SITE / "captures/cli-session.txt").read_text()
    hs = (SITE / "captures/handshake.txt").read_text()
    release = (ROOT / ".github/workflows/release.yml").read_text()
    main = (ROOT / "cmd/goat-client/main.go").read_text()
    gui = release[release.index("name: gui "):]
    return {
        # "# bundle: N bytes", first line of site/captures/cli-session.txt
        "bundle-bytes": re.search(r"^# bundle: (\d+) bytes", cli, re.M).group(1),
        # "--- PASS: TestMobile_RealProtocolHandshake (0.50s)" in site/captures/handshake.txt
        "handshake-s": re.search(r"--- PASS: TestMobile_RealProtocolHandshake \(([\d.]+)s\)", hs).group(1),
        # the gui job's matrix in .github/workflows/release.yml: one row per desktop target
        "desktop-targets": str(len(re.findall(r"^\s+- \{ goos: \w+,\s+goarch: \w+, runner:", gui, re.M))),
        # the "Modes:" line of the usage text in cmd/goat-client/main.go
        "modes": str(len(re.search(r'"Modes: ([^"]+)"', main).group(1).split("|"))),
    }


def check():
    page = (SITE / "index.html").read_text()
    errors = []
    want = measured()
    for key, value in want.items():
        found = re.findall(rf'data-n="{key}">([^<]*)<', page)
        if not found:
            errors.append(f"{key}: the page does not state it")
        errors += [f"{key}: the page says {f}, measured {value}" for f in found if f.strip() != value]
    code = re.search(r'<pre class="bgcode"[^>]*>(.*?)</pre>', page, re.S).group(1)
    code = code.replace("&lt;", "<").replace("&gt;", ">").replace("&amp;", "&")
    if code.strip("\n") not in (ROOT / "internal/bundle/trust.go").read_text():
        errors.append("the code behind the first screen is no longer in internal/bundle/trust.go")
    for e in errors:
        print("site check:", e, file=sys.stderr)
    if not errors:
        print("site check: " + ", ".join(f"{k}={v}" for k, v in want.items()))
    return not errors


def copy():
    (SITE / "assets").mkdir(exist_ok=True)
    for name in ("goat-client-light.svg", "goat-client-dark.svg"):
        shutil.copyfile(ROOT / "docs/figures" / name, SITE / "assets" / name)


if __name__ == "__main__":
    copy()
    sys.exit(0 if check() else 1)
