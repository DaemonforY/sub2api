#!/usr/bin/env python3
"""Turn a single-file HTML game into a static folder the site can host under /games-play/<slug>/.

Single-file builds embed every image and sound as data: URIs in `window.__ASSETS = {path: dataURI}`
and keep code in inline <script> tags. The site's CSP blocks inline scripts and data: audio, and 40 MB
pages are slow, so this:
  1. writes each __ASSETS entry to <out>/<path> and leaves `window.__ASSETS = {}` (both games we host
     fall back to loading the path when an entry is missing);
  2. writes 苏绣's `window.__HZ = {id: gzip+base64}` stroke packs to <out>/data/hanzi/<id>.js, which the
     game loads one at a time when __HZ is absent;
  3. moves every inline <script> into <out>/js/<n>.js, keeping type="module".
Images are optionally recompressed (--webp-quality) without renaming, since the code asks for the
original paths; browsers sniff image formats, and the server sets the type from the extension.

Usage: python3 -I tools/single_html_split.py <in.html> <out-dir> [--title 新标题]
"""
import argparse
import base64
import json
import os
import re
import sys

ASSET_RE = re.compile(r"window\.__ASSETS\s*=\s*")
HZ_RE = re.compile(r";?\s*window\.__HZ\s*=\s*")
SCRIPT_RE = re.compile(r"<script(?P<attrs>[^>]*)>(?P<body>.*?)</script>", re.S)


def take_object(s: str, m: re.Match) -> tuple[dict, int]:
    obj, end = json.JSONDecoder().raw_decode(s, m.end())
    return obj, end


def write(path: str, data: bytes) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(data)


def safe_rel(p: str) -> str:
    p = p.lstrip("/")
    if ".." in p.split("/") or not re.fullmatch(r"[\w\-./]+", p):
        raise SystemExit(f"refusing odd asset path: {p!r}")
    return p


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("src")
    ap.add_argument("out")
    ap.add_argument("--title", default="")
    args = ap.parse_args()

    s = open(args.src, encoding="utf-8").read()
    os.makedirs(args.out, exist_ok=True)
    total = 0

    m = ASSET_RE.search(s)
    if m:
        assets, end = take_object(s, m)
        for path, uri in assets.items():
            head, _, b64 = uri.partition(",")
            if not head.startswith("data:") or ";base64" not in head:
                raise SystemExit(f"unexpected asset value for {path}")
            data = base64.b64decode(b64)
            write(os.path.join(args.out, safe_rel(path)), data)
            total += len(data)
        s = s[: m.start()] + "window.__ASSETS = {}" + s[end:]
        print(f"assets: {len(assets)} files, {total / 1e6:.1f} MB", file=sys.stderr)

    m = HZ_RE.search(s)
    if m:
        packs, end = take_object(s, m)
        for pid, b64 in packs.items():
            if not re.fullmatch(r"[0-9a-f]+", pid):
                raise SystemExit(f"odd hanzi pack id {pid!r}")
            js = f"window.__hzPack({json.dumps(pid)},{json.dumps(b64)});\n"
            write(os.path.join(args.out, "data", "hanzi", f"{pid}.js"), js.encode())
        s = s[: m.start()] + ";" + s[end:]
        print(f"hanzi packs: {len(packs)}", file=sys.stderr)

    n = 0

    def repl(mm: re.Match) -> str:
        nonlocal n
        attrs, body = mm.group("attrs"), mm.group("body")
        if "src=" in attrs or not body.strip():
            return mm.group(0)
        n += 1
        name = f"js/{n:02d}.js"
        write(os.path.join(args.out, name), body.encode())
        keep = " type=\"module\"" if "module" in attrs else ""
        return f'<script{keep} src="{name}"></script>'

    s = SCRIPT_RE.sub(repl, s)
    print(f"scripts: {n} moved out", file=sys.stderr)

    left = re.findall(r"data:[a-z]+/[a-z0-9+.-]+;base64,", s)
    if left:
        print(f"note: {len(left)} data: URIs left in the page (fine for images/fonts, not audio)", file=sys.stderr)
    if args.title:
        s = re.sub(r"<title>.*?</title>", f"<title>{args.title}</title>", s, count=1, flags=re.S)
    write(os.path.join(args.out, "index.html"), s.encode("utf-8"))
    print(f"index.html: {len(s.encode()) / 1e3:.0f} KB", file=sys.stderr)


if __name__ == "__main__":
    main()
