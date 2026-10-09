---
title: 用 GPT 批量处理 Excel：Python 脚本逐行调用 API，结果写回表格
description: 一个可以直接用的 Python 脚本：读取 Excel 里的一列（评论、商品标题、客户留言……），逐行交给 GPT 分类、提取、改写或翻译，结果写进新的一列；支持并发、自动重试、每 20 行保存、中断后接着做，附常见改法和费用估算。
---

# 用 GPT 批量处理 Excel

**把下面的脚本保存成 `batch_excel.py`，改好要处理的列名和提示词，运行 `python batch_excel.py 你的表格.xlsx`，结果会写到新的一列，另存为「你的表格_结果.xlsx」。** 适合几百到几万行的分类、提取、改写、翻译，比在网页里一条条复制粘贴快得多。

> 更新于 2026-10。脚本用 OpenAI 官方 Python SDK 和 openpyxl，已用 45 行的测试表格实际跑过。

## 准备

```bash
pip install -U openai openpyxl
```

设置环境变量 `HIVEGPT_API_KEY`（Windows PowerShell：`$env:HIVEGPT_API_KEY="sk-你的Key"`）。还没有 Key：到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个，建议单独建一个 Key 并设额度上限。

表格的第一行是表头，比如：

| 编号 | 用户评论 |
|---|---|
| 1 | 快递很快，包装也好 |
| 2 | 耳机音质一般，低音闷 |

## 脚本

```python
"""批量处理 Excel：读取一列文字，逐行交给模型处理，结果写进新的一列。

用法：python batch_excel.py 输入.xlsx
"""
import os
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

import openai
from openai import OpenAI
from openpyxl import load_workbook

INPUT_COLUMN = "用户评论"     # 要处理的那一列的表头
OUTPUT_COLUMN = "分析结果"    # 结果写到这一列（没有会自动新建）
MODEL = "gpt-5.5"
WORKERS = 5                    # 同时处理几行，太大会触发 429
PROMPT = """判断下面这条用户评论的情感（正面 / 负面 / 中性），并用不超过 15 个字概括原因。
只输出一行，格式：情感｜原因

评论：{text}"""

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
    max_retries=5,   # 429 / 5xx 自动重试
    timeout=120,
)


def process(text: str) -> str:
    resp = client.chat.completions.create(
        model=MODEL,
        messages=[{"role": "user", "content": PROMPT.format(text=text)}],
    )
    return resp.choices[0].message.content.strip()


def main(path: str):
    wb = load_workbook(path)
    ws = wb.active
    header = [c.value for c in ws[1]]
    src = header.index(INPUT_COLUMN) + 1
    if OUTPUT_COLUMN in header:
        dst = header.index(OUTPUT_COLUMN) + 1
    else:
        dst = len(header) + 1
        ws.cell(row=1, column=dst, value=OUTPUT_COLUMN)

    # 只处理有内容、还没有结果的行，中断后重新运行会接着做
    todo = [r for r in range(2, ws.max_row + 1)
            if ws.cell(r, src).value and not ws.cell(r, dst).value]
    print(f"共 {len(todo)} 行要处理")

    out_path = path.replace(".xlsx", "_结果.xlsx")
    done = 0
    with ThreadPoolExecutor(max_workers=WORKERS) as pool:
        futures = {pool.submit(process, str(ws.cell(r, src).value)): r for r in todo}
        for future in as_completed(futures):
            row = futures[future]
            try:
                ws.cell(row, dst, value=future.result())
            except openai.APIStatusError as e:
                body = e.body if isinstance(e.body, dict) else {}
                ws.cell(row, dst, value=f"失败：{body.get('message') or e.message}")
            done += 1
            if done % 20 == 0:          # 每 20 行保存一次，防止中途出错白做
                wb.save(out_path)
                print(f"已完成 {done}/{len(todo)}")
    wb.save(out_path)
    print(f"完成，结果保存在 {out_path}")


if __name__ == "__main__":
    main(sys.argv[1])
```

## 要改的地方

脚本开头的几行：

| 变量 | 改成 |
|---|---|
| `INPUT_COLUMN` | 你要处理的那一列的表头 |
| `OUTPUT_COLUMN` | 结果写到哪一列（不存在会新建） |
| `MODEL` | 简单任务换成分组里更便宜的模型，见 [模型广场](https://hivegpt.cn/model-plaza) |
| `WORKERS` | 同时处理几行；遇到 429 就调小 |
| `PROMPT` | 你的任务，`{text}` 会被换成每一行的内容 |

**常见任务的提示词**

- 商品标题改写：`把下面的商品标题改写成 30 字以内、突出卖点的版本，只输出标题：{text}`
- 提取信息：`从下面的客户留言里提取「姓名｜电话｜需求」，没有的写「无」，只输出一行：{text}`
- 翻译：`把下面的内容翻译成英文，只输出译文：{text}`
- 打标签：`从「物流、质量、价格、服务、其他」里选一个最贴切的标签，只输出标签：{text}`

**让输出格式稳定**：在提示词里写清楚「只输出一行」「格式：A｜B」，结果更容易用 Excel 的「分列」拆开。需要严格的结构时，用 JSON 输出，见 [Python 完整示例](/connect/python#让模型返回-json)。

## 中途断了怎么办

脚本每 20 行保存一次。中断后，对生成的「_结果.xlsx」再运行一次，已经有结果的行会跳过，只处理剩下的。处理失败的行会写上「失败：原因」，删掉这些单元格再运行一次就会重做。

## 大概花多少钱

按 `gpt-5.5` 的标价，一条 50 字的评论加上提示词约 120 token 输入、20 token 输出，每行约 $0.0012，**1000 行约 $1.2**。换成便宜的小模型可以再便宜几十倍。实际花费以 [使用记录](https://hivegpt.cn/usage) 为准，算法见 [token 怎么算钱](/connect/tokens)。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `ValueError: '用户评论' is not in list` | 列名不对 | 把 `INPUT_COLUMN` 改成表头里的原文 |
| 很多行显示「失败：…429…」 | 并发太高 | 把 `WORKERS` 调小到 2–3，再运行一次 |
| 「失败：…不支持模型…」 | 模型不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| `.xls` 文件打不开 | openpyxl 只支持 `.xlsx` | 在 Excel / WPS 里另存为 `.xlsx` |
| `KeyError: 'HIVEGPT_API_KEY'` | 环境变量没设置 | 在运行脚本的同一个终端里设置 |

更多报错看 [报错速查](/connect/errors/)。
