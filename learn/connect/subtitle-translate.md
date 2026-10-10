---
title: 用 GPT API 批量翻译 SRT 字幕：Python 脚本（保留时间轴，可输出双语字幕）
description: 一个可直接运行的 Python 脚本，用 GPT API 把 SRT 字幕翻译成中文或其他语言：按批发送、用 JSON Schema 保证条数和顺序不乱、失败自动重试、并发加速，支持双语字幕和术语提示；附花费估算和常见问题。
---

# 用 GPT API 批量翻译 SRT 字幕

**一句话：** 下载下面的脚本，设置好 HiveGPT 的 Key，运行 `python translate_srt.py 视频.srt --bilingual`，就会得到保留原时间轴的双语字幕 `视频.bilingual.srt`。一集 20 分钟的视频（约 300 条字幕）用 `gpt-6-luna` 翻译，大约几美分。

> 更新于 2026-10，脚本在 HiveGPT 上实际跑过：40 条字幕约 30 秒、花费约 $0.0013。

## 准备

- Python 3.9 以上，安装 SDK：`pip install -U openai`
- 一个 HiveGPT 的 API Key（[API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组），放进环境变量：

```bash
export HIVEGPT_API_KEY="sk-你的Key"        # macOS / Linux
set HIVEGPT_API_KEY=sk-你的Key             # Windows CMD
```

- 一个 `.srt` 字幕文件。没有字幕的视频，可以先用剪映、Whisper 等工具识别出原文字幕。

## 脚本

保存为 `translate_srt.py`：

```python
"""把 SRT 字幕翻译成中文（或其他语言），保留编号和时间轴。

用法：
    python translate_srt.py input.srt                 # 输出 input.zh.srt（只有译文）
    python translate_srt.py input.srt --bilingual     # 双语：译文在上，原文在下
    python translate_srt.py input.srt --to 日语 --model gpt-6.1-sol
"""
import argparse, json, os, re, sys, time
from concurrent.futures import ThreadPoolExecutor
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"], timeout=300)

BATCH = 30        # 每次请求翻译多少条字幕
WORKERS = 4       # 同时发几个请求；遇到 429 就调小
EFFORT = "low"    # 推理强度：none / low / medium …，模型不支持 none 时用 low
usage = [0, 0]    # 累计输入、输出 token


def parse_srt(text):
    """返回 [(编号, 时间轴, 字幕文本)]。"""
    blocks = re.split(r"\n\s*\n", text.replace("\r\n", "\n").strip())
    items = []
    for b in blocks:
        lines = b.strip().split("\n")
        if len(lines) >= 2 and "-->" in lines[1]:
            items.append((lines[0].strip(), lines[1].strip(), "\n".join(lines[2:]).strip()))
    return items


SCHEMA = {
    "type": "object",
    "properties": {"items": {"type": "array", "items": {
        "type": "object",
        "properties": {"id": {"type": "integer"}, "text": {"type": "string"}},
        "required": ["id", "text"], "additionalProperties": False,
    }}},
    "required": ["items"], "additionalProperties": False,
}


def translate_batch(batch, target, model, context):
    """batch: [(序号, 原文)]，返回 {序号: 译文}。编号对不上就重试。"""
    payload = [{"id": i, "text": t} for i, t in batch]
    for attempt in range(4):
        try:
            resp = client.chat.completions.create(
                model=model,
                messages=[
                    {"role": "system", "content": (
                        f"你是专业的字幕译者。把每条字幕翻译成{target}，口语化、简洁，适合在屏幕上读。"
                        "每条单独翻译，不要合并或拆分，id 原样返回；人名、品牌名保留原文。"
                        + (f"\n视频背景：{context}" if context else ""))},
                    {"role": "user", "content": json.dumps(payload, ensure_ascii=False)},
                ],
                response_format={"type": "json_schema", "json_schema": {"name": "subs", "strict": True, "schema": SCHEMA}},
                reasoning_effort=EFFORT,  # 翻译不需要深度推理，调低了快很多也省钱
            )
            usage[0] += resp.usage.prompt_tokens
            usage[1] += resp.usage.completion_tokens
            out = {it["id"]: it["text"].strip() for it in json.loads(resp.choices[0].message.content)["items"]}
            if set(out) == {i for i, _ in batch}:
                return out
            print(f"  第 {batch[0][0]} 条起的一批：返回条数不对，重试", file=sys.stderr)
        except Exception as e:  # 429、超时等：等一会儿再试
            print(f"  第 {batch[0][0]} 条起的一批出错：{e}，重试", file=sys.stderr)
        time.sleep(2 ** attempt)
    raise RuntimeError(f"第 {batch[0][0]} 条起的一批翻译失败")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("srt")
    ap.add_argument("--to", default="简体中文")
    ap.add_argument("--model", default="gpt-6-luna")
    ap.add_argument("--bilingual", action="store_true")
    ap.add_argument("--context", default="", help="一句话说明视频内容，帮助统一术语")
    args = ap.parse_args()

    items = parse_srt(open(args.srt, encoding="utf-8-sig").read())
    texts = [(n, t) for n, (_, _, t) in enumerate(items)]
    batches = [texts[i:i + BATCH] for i in range(0, len(texts), BATCH)]
    print(f"{len(items)} 条字幕，分 {len(batches)} 批翻译…", file=sys.stderr)

    translated = {}
    with ThreadPoolExecutor(WORKERS) as pool:
        for part in pool.map(lambda b: translate_batch(b, args.to, args.model, args.context), batches):
            translated.update(part)

    out_path = re.sub(r"\.srt$", "", args.srt) + (".bilingual.srt" if args.bilingual else ".zh.srt")
    with open(out_path, "w", encoding="utf-8") as f:
        for n, (idx, timing, text) in enumerate(items):
            body = translated[n] + ("\n" + text if args.bilingual else "")
            f.write(f"{idx}\n{timing}\n{body}\n\n")
    print(f"完成：{out_path}（输入 {usage[0]} token，输出 {usage[1]} token）", file=sys.stderr)


if __name__ == "__main__":
    main()
```

## 运行

```bash
python translate_srt.py 视频.srt                          # 只有译文：视频.zh.srt
python translate_srt.py 视频.srt --bilingual              # 双语：视频.bilingual.srt
python translate_srt.py 视频.srt --context "Vite + React 入门教程"   # 告诉模型视频讲什么，术语更准
python translate_srt.py 视频.srt --to 日语 --model gpt-6.1-sol
```

翻译效果（双语模式）：

```text
1
00:00:00,000 --> 00:00:02,500
大家好，欢迎回到我们的频道。
Hi everyone, welcome back to the channel.

2
00:00:03,000 --> 00:00:05,500
今天我们来做一个小型 Web 应用。
Today we're going to build a tiny web app.
```

生成的 `.srt` 可以直接拖进播放器（PotPlayer、IINA、VLC），或导入剪映、Premiere 等剪辑软件。

## 脚本是怎么保证不乱的

- **按批翻译**：每次发 30 条，比一条一条发快得多、也便宜（固定的说明只发一次）；又不会一次太多导致漏条。
- **带编号返回**：每条字幕带 `id` 发过去，用 JSON Schema 要求模型按 `id` 原样返回。返回的编号和发过去的对不上，就自动重试，所以不会出现译文错位。
- **时间轴不经过模型**：时间轴和编号由脚本原样写回，模型只碰文字，时间不会被改坏。
- **调低推理强度**：`reasoning_effort="low"`。翻译不需要深度推理，实测比默认设置快好几倍。

## 花多少钱

字幕翻译主要是输入 token。实测 40 条英文短句用了约 9700 个输入 token、600 个输出 token：

| 模型 | 40 条 | 一集约 300 条 |
|---|---|---|
| `gpt-6-luna` | 约 $0.0013 | 约 $0.01 |
| `gpt-6.1-sol` | 约 $0.025 | 约 $0.2 |

日常视频用 `gpt-6-luna` 就够；专业内容（医学、法律、技术讲座）对术语要求高，换 `gpt-6.1-sol` 并加上 `--context`。各模型价格见 [GPT 模型怎么选](/connect/models)。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `KeyError: 'HIVEGPT_API_KEY'` | 没设置环境变量 | 按上面「准备」设置后，在同一个终端里运行 |
| 一直提示「返回条数不对，重试」 | 字幕太长或太乱，模型漏条 | 把 `BATCH` 调小到 15 |
| 429 / 请求太频繁 | 并发太多 | 把 `WORKERS` 调小到 1–2，见 [429 报错](/connect/errors/429) |
| 报 `reasoning_effort` 不支持 | 模型不支持这个参数 | 删掉 `reasoning_effort=EFFORT` 这一行 |
| 译文太书面、太长 | 提示词不够具体 | 改 system 提示词，例如「每条不超过 15 个字」 |
| 人名、术语前后不一致 | 每批单独翻译 | 用 `--context` 写上人名和术语，例如「主讲人 Ada，产品叫 Nova」 |
| 打开 srt 乱码 | 原文件不是 UTF-8 | 用记事本 / VS Code 另存为 UTF-8 再运行 |

**能翻译 ASS、VTT 字幕吗？** 这个脚本只处理 SRT。可以先用 FFmpeg 转格式：`ffmpeg -i 视频.vtt 视频.srt`。

**能直接翻译视频里的声音吗？** 要先把语音识别成字幕（剪映的「识别字幕」、Whisper 等），再用这个脚本翻译。

更多：[批量总结文档](/connect/summarize-docs) · [批量处理 Excel](/connect/excel) · [Python 完整示例](/connect/python) · [Bob、Pot 划词翻译](/connect/bob-pot)
