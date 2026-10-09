---
title: Python 调用 GPT API 完整示例：对话、流式、多轮、JSON、函数调用、并发
description: 用 OpenAI 官方 Python SDK 调用 HiveGPT 等 OpenAI 兼容接口的完整代码：base_url 怎么填、流式输出、多轮对话、结构化 JSON 输出、Function Calling、异步并发、超时重试和错误处理，以及 APIConnectionError、unsupported parameter 等常见报错。
---

# Python 调用 GPT API 完整示例

**只改两处就能用：** 安装 `openai` 包，创建客户端时把 `base_url` 设为 `https://hivegpt.cn/v1`、`api_key` 设为你的 HiveGPT Key，其余代码和调用 OpenAI 官方接口完全一样。

```python
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key="sk-你的Key")
resp = client.chat.completions.create(
    model="gpt-5.5",
    messages=[{"role": "user", "content": "用一句话介绍你自己"}],
)
print(resp.choices[0].message.content)
```

下面每段代码都是完整的脚本，复制保存成 `.py` 文件就能运行。

> 更新于 2026-10，基于 `openai` Python SDK 3.x（1.x / 2.x 的写法相同）。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备

```bash
pip install -U openai
```

Key 放进环境变量，不要写死在代码里，也不要提交到 Git：

::: code-group

```bash [macOS / Linux]
export HIVEGPT_API_KEY="sk-你的Key"
```

```powershell [Windows PowerShell]
$env:HIVEGPT_API_KEY="sk-你的Key"
```

:::

还没有 Key：到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个，分组选 GPT 类分组。

::: tip 不改代码，只改环境变量
OpenAI SDK 会读取 `OPENAI_BASE_URL` 和 `OPENAI_API_KEY`。把它们设成 HiveGPT 的地址和 Key，写的是 `OpenAI()`（不带参数）的现成项目也能直接切到 HiveGPT：

```bash
export OPENAI_BASE_URL="https://hivegpt.cn/v1"
export OPENAI_API_KEY="sk-你的Key"
```
:::

## 基本对话

```python
import os
from openai import OpenAI

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
)

resp = client.chat.completions.create(
    model="gpt-5.5",
    messages=[
        {"role": "system", "content": "你是一位耐心的编程老师，回答简洁。"},
        {"role": "user", "content": "Python 的列表和元组有什么区别？"},
    ],
)
print(resp.choices[0].message.content)
print("本次用量：", resp.usage.total_tokens, "tokens")
```

`system` 消息用来设定角色和回答风格；`resp.usage` 是这次请求的 token 用量，和 [使用记录](https://hivegpt.cn/usage) 里的一致。

## 流式输出

回答边生成边显示，不用等整段写完：

```python
import os
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])

stream = client.chat.completions.create(
    model="gpt-5.5",
    messages=[{"role": "user", "content": "写一首关于秋天的四行小诗"}],
    stream=True,
    stream_options={"include_usage": True},  # 最后一块带上用量
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
    if chunk.usage:
        print("\n本次用量：", chunk.usage.total_tokens, "tokens")
```

## 多轮对话

模型本身不记得上一轮说了什么，**每次请求都要把之前的对话一起发过去**：

```python
import os
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])

messages = [{"role": "system", "content": "你是一位旅行规划助手。"}]

for question in ["我五月想去云南玩 5 天", "预算 5000 元够吗？", "那第一天怎么安排？"]:
    messages.append({"role": "user", "content": question})
    resp = client.chat.completions.create(model="gpt-5.5", messages=messages)
    answer = resp.choices[0].message.content
    messages.append({"role": "assistant", "content": answer})
    print(f"问：{question}\n答：{answer}\n")
```

想做成命令行聊天，把 `for` 循环换成 `while True:` 加 `question = input("> ")` 即可。对话越长，每次发送的 token 越多、越贵，长对话可以只保留最近几轮。

## 让模型返回 JSON

要把结果交给程序处理时，用 `parse` 加一个 Pydantic 模型，SDK 会让模型按结构输出并直接解析成对象：

```python
import os
from openai import OpenAI
from pydantic import BaseModel

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])


class Contact(BaseModel):
    name: str
    phone: str
    city: str


resp = client.chat.completions.parse(
    model="gpt-5.5",
    messages=[
        {"role": "system", "content": "从用户的话里提取联系人信息。"},
        {"role": "user", "content": "我是李雷，住在杭州，电话 138 0000 1234"},
    ],
    response_format=Contact,
)
contact = resp.choices[0].message.parsed
print(contact.name, contact.phone, contact.city)
```

需要先 `pip install pydantic`（装 `openai` 时一般已经带上）。只想要「是合法 JSON」而不限定结构时，用 `create` 加 `response_format={"type": "json_object"}`，并在提示词里写明要 JSON。

## 函数调用（Function Calling）

让模型决定什么时候调用你的函数：你描述函数，模型返回要调用哪个、参数是什么，你执行后把结果发回去，模型再组织最终回答。

```python
import json
import os
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])


def get_weather(city: str) -> dict:
    # 换成真实的天气接口
    return {"city": city, "weather": "晴", "temperature": 23}


tools = [{
    "type": "function",
    "function": {
        "name": "get_weather",
        "description": "查询某个城市当前的天气",
        "parameters": {
            "type": "object",
            "properties": {"city": {"type": "string", "description": "城市名，如 北京"}},
            "required": ["city"],
        },
    },
}]

messages = [{"role": "user", "content": "北京今天天气怎么样？适合跑步吗？"}]
resp = client.chat.completions.create(model="gpt-5.5", messages=messages, tools=tools)
msg = resp.choices[0].message

if msg.tool_calls:
    messages.append(msg)
    for call in msg.tool_calls:
        args = json.loads(call.function.arguments)
        result = get_weather(**args)
        messages.append({"role": "tool", "tool_call_id": call.id, "content": json.dumps(result, ensure_ascii=False)})
    resp = client.chat.completions.create(model="gpt-5.5", messages=messages, tools=tools)

print(resp.choices[0].message.content)
```

原理和更多例子见 [Function Calling：让模型调用你的函数](/a/a4)。

## 异步和并发

批量处理很多条数据时，用 `AsyncOpenAI` 并发请求，并用信号量限制同时进行的数量（太多会触发 [429](/connect/errors/429)）：

```python
import asyncio
import os
from openai import AsyncOpenAI

client = AsyncOpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])
limit = asyncio.Semaphore(5)  # 最多同时 5 个请求


async def translate(text: str) -> str:
    async with limit:
        resp = await client.chat.completions.create(
            model="gpt-5.5",
            messages=[{"role": "user", "content": f"翻译成英文，只输出译文：{text}"}],
        )
        return resp.choices[0].message.content


async def main():
    texts = ["今天天气很好", "我们明天见", "这个价格可以接受", "请尽快回复"]
    results = await asyncio.gather(*(translate(t) for t in texts))
    for src, dst in zip(texts, results):
        print(src, "→", dst)


asyncio.run(main())
```

## 超时、重试和错误处理

SDK 遇到 429 和 5xx 会自动重试（默认 2 次），可以调大；业务代码里按错误类型分别处理：

```python
import os
import openai
from openai import OpenAI

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
    max_retries=5,  # 429 / 5xx 自动重试，间隔逐次加长
    timeout=120,    # 单次请求最长等 120 秒
)


def reason(e: openai.APIStatusError) -> str:
    """HiveGPT 的报错说明（中文，括号里是英文原文）。"""
    body = e.body if isinstance(e.body, dict) else {}
    return body.get("message") or e.message


try:
    resp = client.chat.completions.create(
        model="gpt-5.5",
        messages=[{"role": "user", "content": "你好"}],
    )
    print(resp.choices[0].message.content)
except openai.AuthenticationError as e:
    print("Key 无效或已停用：", reason(e))
except openai.NotFoundError as e:
    print("地址或模型名不对：", reason(e))
except openai.RateLimitError as e:
    print("请求太频繁或额度用完：", reason(e))
except openai.APIConnectionError:
    print("连不上服务器，检查网络和代理")
except openai.APIStatusError as e:
    print("其他错误：", e.status_code, reason(e))
```

`reason(e)` 取出的是 HiveGPT 返回的中文说明，照着处理即可（直接打印 `e.message` 会带上状态码和整段 JSON）。各种报错的含义见 [报错速查](/connect/errors/)。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `openai.APIConnectionError: Connection error` | 网络不通，或系统代理拦截了请求 | 浏览器能打开 hivegpt.cn 的话，检查 `HTTPS_PROXY` 等代理环境变量是否指向一个没开的代理 |
| `NotFoundError`，报错说「接口地址少了 /v1」或「多了一个 /v1」 | `base_url` 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| `AuthenticationError` 401 | Key 不对 | 见 [401 报错](/connect/errors/401) |
| `NotFoundError`，报错说分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| `BadRequestError`：`Unsupported parameter: 'max_tokens'` | 新模型不接受 `max_tokens` | 改用 `max_completion_tokens` |
| `BadRequestError`：`temperature` 不支持 | 推理类模型只接受默认值 | 去掉 `temperature`、`top_p` 参数 |
| `KeyError: 'HIVEGPT_API_KEY'` | 环境变量没设，或设在了另一个终端窗口 | 在运行脚本的同一个终端里 `export`，或写进 `.env` 后用 `python-dotenv` 读取 |

## 下一步

- Node.js 版本：[Node.js 调用 OpenAI 兼容接口](/connect/nodejs)
- 用框架搭应用：[LangChain 接入 HiveGPT](/connect/langchain)
- 系统学做 AI 应用：[A 路线 · AI 应用开发入门](/a/)
