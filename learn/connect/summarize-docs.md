---
title: 用 GPT API 批量总结 PDF、Word 文档：Python 脚本，结果导出 Excel
description: 一个可直接运行的 Python 脚本：把文件夹里的 PDF、Word（docx）、txt、Markdown 逐个交给 GPT，生成标题、一句话摘要、要点和关键词，还能统一回答一个问题，结果写进 summary.csv 用 Excel 打开；PDF 不用装解析库，附花费估算和常见问题。
---

# 用 GPT API 批量总结 PDF、Word 文档

**一句话：** 把文档放进一个文件夹，运行 `python summarize_docs.py 文件夹`，几分钟后得到一张 `summary.csv`：每个文件一行，包括标题、一句话摘要、要点、关键词。还能加 `--question "有哪些需要跟进的事项？"`，让它对每份文档回答同一个问题。

> 更新于 2026-10，脚本在 HiveGPT 上实际跑过：PDF、Word、Markdown 各一份，每份约 5000 token，用 `gpt-6-luna` 每份不到 $0.001。

## 适合做什么

- 一堆会议纪要、周报，快速看每份讲了什么、有哪些待办；
- 投标文件、合同、制度文件，先过一遍摘要再挑重点细读；
- 论文、行业报告，批量提取结论和关键词；
- 客户反馈、访谈记录，统一回答「客户最不满意什么」。

## 准备

```bash
pip install -U openai python-docx
export HIVEGPT_API_KEY="sk-你的Key"     # Windows CMD：set HIVEGPT_API_KEY=sk-你的Key
```

Key 在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。

## 脚本

保存为 `summarize_docs.py`：

```python
"""批量总结一个文件夹里的 PDF / Word / txt / md 文档，结果写进 summary.csv（Excel 可直接打开）。

用法：
    pip install -U openai python-docx
    python summarize_docs.py 文件夹路径
    python summarize_docs.py 文件夹路径 --model gpt-6.1-sol --question "这份材料对我们的产品有什么启发？"
"""
import argparse, base64, csv, json, os, sys, time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"], timeout=300)

WORKERS = 3              # 同时处理几个文件；遇到 429 就调小
MAX_CHARS = 60_000       # Word / 文本只取前这么多字，防止超长文档花太多钱
SCHEMA = {
    "type": "object",
    "properties": {
        "title": {"type": "string", "description": "文档标题，没有就根据内容起一个"},
        "summary": {"type": "string", "description": "一句话摘要，50 字以内"},
        "key_points": {"type": "array", "items": {"type": "string"}, "description": "3 到 5 条要点"},
        "keywords": {"type": "array", "items": {"type": "string"}},
        "answer": {"type": "string", "description": "对用户问题的回答；没有问题就留空"},
    },
    "required": ["title", "summary", "key_points", "keywords", "answer"],
    "additionalProperties": False,
}


def file_content(path: Path):
    """把文件变成消息里的一段内容：PDF 整个发过去，其他格式取文字。"""
    ext = path.suffix.lower()
    if ext == ".pdf":
        data = base64.b64encode(path.read_bytes()).decode()
        return {"type": "file", "file": {"filename": path.name, "file_data": "data:application/pdf;base64," + data}}
    if ext == ".docx":
        import docx  # pip install python-docx
        doc = docx.Document(str(path))
        text = "\n".join(p.text for p in doc.paragraphs)
        for table in doc.tables:  # 表格里的文字也带上
            for row in table.rows:
                text += "\n" + " | ".join(cell.text for cell in row.cells)
    else:
        text = path.read_text(encoding="utf-8", errors="ignore")
    return {"type": "text", "text": f"文件名：{path.name}\n\n{text[:MAX_CHARS]}"}


def summarize(path: Path, model: str, question: str):
    prompt = "阅读这份文档，按要求的格式输出中文总结。"
    if question:
        prompt += f"\n另外回答这个问题（写在 answer 里）：{question}"
    for attempt in range(3):
        try:
            resp = client.chat.completions.create(
                model=model,
                messages=[{"role": "user", "content": [file_content(path), {"type": "text", "text": prompt}]}],
                response_format={"type": "json_schema", "json_schema": {"name": "doc", "strict": True, "schema": SCHEMA}},
                reasoning_effort="low",
            )
            r = json.loads(resp.choices[0].message.content)
            r["file"], r["tokens"] = path.name, resp.usage.total_tokens
            print(f"✓ {path.name}（{resp.usage.total_tokens} token）", file=sys.stderr)
            return r
        except Exception as e:
            print(f"  {path.name} 出错：{e}，重试", file=sys.stderr)
            time.sleep(2 ** attempt)
    return {"file": path.name, "title": "", "summary": "处理失败", "key_points": [], "keywords": [], "answer": "", "tokens": 0}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("folder")
    ap.add_argument("--model", default="gpt-6-luna")
    ap.add_argument("--question", default="")
    args = ap.parse_args()

    files = sorted(p for p in Path(args.folder).iterdir() if p.suffix.lower() in {".pdf", ".docx", ".txt", ".md"})
    print(f"共 {len(files)} 个文件", file=sys.stderr)
    with ThreadPoolExecutor(WORKERS) as pool:
        results = list(pool.map(lambda p: summarize(p, args.model, args.question), files))

    with open("summary.csv", "w", newline="", encoding="utf-8-sig") as f:  # utf-8-sig：Excel 打开不乱码
        w = csv.writer(f)
        w.writerow(["文件", "标题", "一句话摘要", "要点", "关键词", "问题回答", "token"])
        for r in results:
            w.writerow([r["file"], r["title"], r["summary"], "\n".join(r["key_points"]), "、".join(r["keywords"]), r["answer"], r["tokens"]])
    print(f"完成：summary.csv，共用 {sum(r['tokens'] for r in results)} token", file=sys.stderr)


if __name__ == "__main__":
    main()
```

## 运行

```bash
python summarize_docs.py ./文档
python summarize_docs.py ./文档 --question "有哪些需要跟进的事项？"
python summarize_docs.py ./合同 --model gpt-6.1-sol --question "付款条件和违约责任是什么？"
```

实测结果（`summary.csv` 用 Excel 打开）：

| 文件 | 一句话摘要 | 问题回答（有哪些需要跟进的事项？） |
|---|---|---|
| 周会纪要.md | 小程序 2.0 延期一周；新增注册环比增长 18%；下周上线客服机器人并修复安卓闪退。 | 推进支付接口重新审核并调整上线计划；完成客服机器人上线；修复安卓闪退。 |
| 弹性工作制通知.docx | 研发中心自 11 月 1 日试行弹性工作制三个月，期满评估是否推广。 | 安排每周三全员到岗及部门例会；试行三个月后汇总考勤和问卷结果。 |

## 脚本是怎么处理不同格式的

- **PDF**：整个文件转成 base64 直接发给模型（`{"type": "file", ...}`），不用装 PDF 解析库，扫描件、表格也能读。见 [看图识图 / 读 PDF](/connect/vision)。
- **Word（.docx）**：用 python-docx 取出正文和表格里的文字。老的 `.doc` 格式先用 Word / WPS 另存为 `.docx`。
- **txt / md**：直接读文字。
- **结构化输出**：用 JSON Schema 规定好要返回哪些字段，结果稳定，可以直接写进表格。

## 花多少钱

花费主要看文档长度。实测一份一页的文档约 5000 token（其中约 4500 是每次请求固定的开销），之后每多一页约多 1000 token，PDF 每页还会按页面图片多算一些。

| 模型 | 一份 10 页的文档约 | 100 份约 |
|---|---|---|
| `gpt-6-luna` | $0.002 | $0.2 |
| `gpt-6.1-sol` | $0.03 | $3 |

先用 `gpt-6-luna` 跑几份看效果，要求高（合同、论文）再换 `gpt-6.1-sol`。超长文档脚本只取前 6 万字（`MAX_CHARS`），需要全文就调大，或先拆分。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `ModuleNotFoundError: docx` | 没装 python-docx | `pip install python-docx`（注意不是 `pip install docx`） |
| Excel 打开 CSV 乱码 | 编码问题 | 脚本已用 `utf-8-sig`；用 WPS 或「数据 → 从文本导入」选 UTF-8 |
| 413 请求内容太大 | PDF 太大（几十 MB） | 拆分 PDF 或压缩后再处理 |
| 上下文太长 | 文档超出模型上限 | 调小 `MAX_CHARS`，或拆成几个文件，见 [上下文太长](/connect/errors/context-length) |
| 429 | 并发太多 | 把 `WORKERS` 调到 1，见 [429 报错](/connect/errors/429) |
| 某个文件「处理失败」 | 文件损坏、加密或格式不支持 | 单独打开检查；加密 PDF 先解除密码 |
| 摘要里有原文没有的内容 | 模型「编」了 | 提示词里加「只根据文档内容回答，没提到的写『未提及』」，重要内容人工核对 |

**数据安全吗？** 文档内容会发送到接口进行处理。涉密或含个人隐私的文件，请按你所在单位的规定决定是否使用。

更多：[批量翻译字幕](/connect/subtitle-translate) · [批量处理 Excel](/connect/excel) · [看图识图 / 读 PDF](/connect/vision) · [Python 完整示例](/connect/python)
