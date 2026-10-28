"""Builds the 灵感库 poem corpus (lib_poems) from chinese-poetry (MIT, https://github.com/chinese-poetry/chinese-poetry),
plus the periods it lacks (汉 to 南北朝, 隋, 金, 元, 明, 清) from Werneror/Poetry (MIT, https://github.com/Werneror/Poetry).

    python3 -m venv .venv && .venv/bin/pip install opencc
    .venv/bin/python build.py <chinese-poetry dir> <Werneror dir> <out.csv.gz>

<chinese-poetry dir> holds that repository's files at their own paths (全唐诗/poet.tang.0.json …, rank/poet/…);
<Werneror dir> the period CSVs (汉.csv, 明_1.csv …). Late-Qing and later periods are left out: their authors may
still be under copyright. Traditional
characters are converted to simplified (OpenCC t2s). Every row is one work; anthologies (唐诗三百首, 宋词三百首,
千家诗) only mark works as famous. popularity is log10 of the Baidu hit count from rank/, plus 3 for famous works.
Load with load.sh.
"""
import csv, glob, gzip, json, math, os, re, sys

import opencc

T2S = opencc.OpenCC("t2s").convert
raw, raw2, out = sys.argv[1], sys.argv[2], sys.argv[3]
rows = []


def j(path):
    with open(os.path.join(raw, path), encoding="utf-8") as f:
        return json.load(f)


def clean(s):
    return T2S(re.sub(r"\s+", "", s or ""))


def norm(s):
    # For matching anthology entries: simplified, no punctuation or notes.
    return re.sub(r"[^㐀-鿿]", "", clean(s))


def add(collection, dynasty, title, author, lines, notes=None, rank=None, tags=None):
    lines = [clean(l) for l in lines if l and l.strip()]
    if not lines:
        return
    rows.append({
        "collection": collection, "dynasty": dynasty, "title": clean(title) or "无题", "author": clean(author) or "佚名",
        "content": "\n".join(lines), "notes": "\n".join(clean(n) for n in (notes or []) if n),
        "rank": rank, "tags": tags or [],
    })


def shards(pattern):
    def key(p):
        m = re.search(r"\.(\d+)\.json$", p)
        return int(m.group(1)) if m else -1
    return sorted(glob.glob(os.path.join(raw, pattern)), key=key)


def ranks(rank_path):
    p = os.path.join(raw, rank_path)
    return json.load(open(p, encoding="utf-8")) if os.path.exists(p) else []


# 全唐诗 (唐 + 五代) and 全宋诗, with their Baidu counts.
for kind, coll, dyn in (("tang", "tang", "唐"), ("song", "song_shi", "宋")):
    for p in shards(f"全唐诗/poet.{kind}.*.json"):
        n = re.search(r"\.(\d+)\.json$", p).group(1)
        rk = ranks(f"rank/poet/poet.{kind}.rank.{n}.json")
        for i, w in enumerate(json.load(open(p, encoding="utf-8"))):
            r = rk[i] if i < len(rk) and rk[i].get("title") == w.get("title") else None
            add(coll, dyn, w.get("title"), w.get("author"), w.get("paragraphs", []), rank=(r or {}).get("baidu"))

# 宋词: no title, so 词牌·first phrase.
for p in shards("宋词/ci.song.*.json"):
    m = re.search(r"\.(\d+)\.json$", p)
    rk = ranks(f"rank/ci/ci.song.rank.{m.group(1)}.json") if m else []
    for i, w in enumerate(json.load(open(p, encoding="utf-8"))):
        paras = w.get("paragraphs", [])
        first = re.split(r"[，。、；：！？]", clean(paras[0]) if paras else "")[0]
        r = rk[i] if i < len(rk) and rk[i].get("rhythmic") == w.get("rhythmic") else None
        add("song_ci", "宋", f"{w.get('rhythmic', '')}·{first}" if first else w.get("rhythmic"), w.get("author"), paras, rank=(r or {}).get("baidu"))

for w in j("元曲/yuanqu.json"):
    add("yuanqu", "元", w.get("title"), w.get("author"), w.get("paragraphs", []))
for w in j("诗经/shijing.json"):
    add("shijing", "先秦", w.get("title"), "佚名", w.get("content", []), tags=[w.get("chapter", ""), w.get("section", "")])
for w in j("楚辞/chuci.json"):
    add("chuci", "先秦", w.get("title"), w.get("author"), w.get("content", []), tags=[w.get("section", "")])
for p in shards("五代诗词/huajianji/huajianji-*-juan.json") + sorted(glob.glob(os.path.join(raw, "五代诗词/huajianji/huajianji-x-juan.json"))):
    for w in json.load(open(p, encoding="utf-8")):
        add("huajian", "五代", w.get("title"), w.get("author"), w.get("paragraphs", []), notes=w.get("notes"))
for w in j("五代诗词/nantang/poetrys.json"):
    add("nantang", "五代", w.get("title"), w.get("author"), w.get("paragraphs", []), notes=w.get("notes"))
for w in j("曹操诗集/caocao.json"):
    add("caocao", "汉", w.get("title"), "曹操", w.get("paragraphs", []))
for w in j("纳兰性德/纳兰性德诗集.json"):
    add("nalan", "清", w.get("title"), w.get("author") or "纳兰性德", w.get("para", []))
for juan in j("蒙学/guwenguanzhi.json")["content"]:
    for w in juan["content"]:
        a = clean(w.get("author", ""))
        dyn, _, name = a.partition("：")
        if not name:
            dyn, name = "", a
        add("guwen", dyn, w.get("chapter"), name, w.get("paragraphs", []), tags=[clean(w.get("source", ""))])
for w in j("论语/lunyu.json"):
    add("lunyu", "先秦", w.get("chapter"), "孔子及弟子", w.get("paragraphs", []))

# Werneror/Poetry: one CSV per period, the text in one string (split into lines after sentence ends).
WERNEROR = {"秦": "秦", "汉": "汉", "魏晋": "魏晋", "魏晋末南北朝初": "南北朝", "南北朝": "南北朝", "隋": "隋", "隋末唐初": "隋",
            "辽": "辽", "金": "金", "宋末金初": "金", "金末元初": "元", "元": "元", "元末明初": "明", "明": "明", "明末清初": "清", "清": "清"}
for p in sorted(glob.glob(os.path.join(raw2, "*.csv"))):
    period = re.sub(r"_\d+$", "", os.path.basename(p)[:-4])
    if period not in WERNEROR:
        continue
    with open(p, encoding="utf-8") as f:
        for rec in csv.DictReader(f):
            text = clean(rec.get("内容", ""))
            lines = []
            for l in re.split(r"(?<=[。；])", text):
                # A question or exclamation usually ends half a couplet; split there only in long runs.
                lines += [x for x in re.split(r"(?<=[！？])", l) if x] if len(l) > 40 else [l] if l else []
            add("lidai", WERNEROR[period], rec.get("题目"), rec.get("作者"), lines)

# The same work from two sources (曹操 is in both): keep the first.
seen, kept = set(), []
for r in rows:
    k = (norm(r["author"]), norm(r["content"])[:12])
    if k in seen:
        continue
    seen.add(k)
    kept.append(r)
rows = kept

# Famous: in 唐诗三百首 / 宋词三百首 / 千家诗. Match by title + author, or by the first line.
famous = {}
def mark(title, author, paras, tag):
    first = norm(paras[0])[:7] if paras else ""
    famous.setdefault(("t", norm(title), norm(author)), set()).add(tag)
    if first:
        famous.setdefault(("f", first, norm(author)), set()).add(tag)
for w in j("全唐诗/唐诗三百首.json"):
    mark(w.get("title"), w.get("author"), w.get("paragraphs", []), "唐诗三百首")
for w in j("宋词/宋词三百首.json"):
    mark(w.get("rhythmic"), w.get("author"), w.get("paragraphs", []), "宋词三百首")
for t in j("蒙学/qianjiashi.json")["content"]:
    for w in t["content"]:
        paras = w.get("paragraphs", [])
        # 组诗 come as [{subchapter, paragraphs}]: mark each part.
        parts = [x.get("paragraphs", []) for x in paras if isinstance(x, dict)] or [paras]
        for part in parts:
            mark(w.get("chapter"), re.sub(r"（.*?）", "", w.get("author", "")), part, "千家诗")
for r in rows:
    if r["collection"] == "guwen":
        r["tags"].append("古文观止")
    tags = set()
    first = norm(r["content"].split("\n")[0])[:7]
    title = r["title"].split("·")[0] if r["collection"] == "song_ci" else r["title"]
    for k in (("t", norm(title), norm(r["author"])), ("f", first, norm(r["author"]))):
        tags |= famous.get(k, set())
    r["tags"] = sorted({t for t in r["tags"] if t} | tags)

with gzip.open(out, "wt", encoding="utf-8", newline="") as f:
    w = csv.writer(f, lineterminator="\n")
    w.writerow(["collection", "title", "author", "dynasty", "content", "notes", "tags", "famous", "popularity"])
    for r in rows:
        fam = any(t in ("唐诗三百首", "宋词三百首", "千家诗", "古文观止") for t in r["tags"])
        pop = math.log10(r["rank"] + 1) if r["rank"] else 0.0
        w.writerow([r["collection"], r["title"], r["author"], r["dynasty"], r["content"], r["notes"],
                    "{" + ",".join('"' + t.replace('"', "") + '"' for t in r["tags"]) + "}", "t" if fam else "f",
                    round(pop + (3 if fam else 0), 3)])
from collections import Counter
print(len(rows), "works;", sum(1 for r in rows if any(t in ("唐诗三百首", "宋词三百首", "千家诗") for t in r["tags"])), "famous")
print(Counter(r["collection"] for r in rows))
