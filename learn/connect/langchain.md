---
title: LangChain 配置自定义 Base URL：ChatOpenAI 接入第三方 OpenAI 兼容接口
description: 在 LangChain（Python 和 LangChain.js）里用 ChatOpenAI 接入 HiveGPT 等 OpenAI 兼容接口：base_url / configuration.baseURL 怎么填、提示词模板和链、流式输出、with_structured_output 结构化输出、bind_tools 工具调用、OpenAIEmbeddings 嵌入（check_embedding_ctx_length），以及常见报错。
---

# LangChain 接入 HiveGPT

**只改创建模型的那一行：** Python 里用 `ChatOpenAI(model="gpt-5.5", base_url="https://hivegpt.cn/v1", api_key=...)`；LangChain.js 里用 `new ChatOpenAI({ model: 'gpt-5.5', apiKey, configuration: { baseURL: 'https://hivegpt.cn/v1' } })`。链、Agent、RAG 等其他代码都不用动。

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(model="gpt-5.5", base_url="https://hivegpt.cn/v1", api_key="sk-你的Key")
print(llm.invoke("用一句话介绍你自己").content)
```

> 更新于 2026-10，基于 `langchain-openai` 1.x（Python）和 `@langchain/openai` 1.x（JS）。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备（Python）

```bash
pip install -U langchain-openai
```

Key 放进环境变量，不要写死在代码里：

::: code-group

```bash [macOS / Linux]
export HIVEGPT_API_KEY="sk-你的Key"
```

```powershell [Windows PowerShell]
$env:HIVEGPT_API_KEY="sk-你的Key"
```

:::

还没有 Key：到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个，分组选 GPT 类分组。

::: tip 只改环境变量
`ChatOpenAI` 会读取 `OPENAI_API_KEY` 和 `OPENAI_BASE_URL`（旧写法 `OPENAI_API_BASE` 也认）。现成的 LangChain 项目，把这两个变量设成 HiveGPT 的 Key 和 `https://hivegpt.cn/v1`，不改代码就能切过来。
:::

## 基本调用和流式输出

```python
import os
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-5.5",
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
    max_retries=3,  # 429 / 5xx 自动重试
    timeout=120,
)

# 一次性返回
resp = llm.invoke([
    ("system", "你是一位耐心的编程老师，回答简洁。"),
    ("human", "什么是装饰器？"),
])
print(resp.content)
print("本次用量：", resp.usage_metadata)

# 流式输出
for chunk in llm.stream("写一首关于秋天的四行小诗"):
    print(chunk.content, end="", flush=True)
print()
```

## 提示词模板和链

```python
import os
from langchain_core.output_parsers import StrOutputParser
from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(model="gpt-5.5", base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])

prompt = ChatPromptTemplate.from_messages([
    ("system", "你是{role}，用不超过 100 字回答。"),
    ("human", "{question}"),
])
chain = prompt | llm | StrOutputParser()

print(chain.invoke({"role": "资深产品经理", "question": "怎么判断一个需求值不值得做？"}))

# 批量执行，max_concurrency 控制同时进行的请求数
answers = chain.batch(
    [{"role": "厨师", "question": "怎么煮出不粘的米饭？"}, {"role": "健身教练", "question": "每周练几次合适？"}],
    config={"max_concurrency": 3},
)
for a in answers:
    print("-", a)
```

## 结构化输出

`with_structured_output` 让模型按 Pydantic 模型输出，直接拿到对象：

```python
import os
from langchain_openai import ChatOpenAI
from pydantic import BaseModel, Field

llm = ChatOpenAI(model="gpt-5.5", base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])


class Review(BaseModel):
    sentiment: str = Field(description="正面 / 负面 / 中性")
    score: int = Field(description="1 到 5 分")
    keywords: list[str] = Field(description="评价里提到的要点")


structured = llm.with_structured_output(Review)
review = structured.invoke("快递很快，包装也好，就是耳机音质一般，低音有点闷。")
print(review.sentiment, review.score, review.keywords)
```

## 工具调用

用 `@tool` 把函数变成工具，`bind_tools` 交给模型，模型决定调用后由你执行，再把结果发回去：

```python
import os
from langchain_core.messages import HumanMessage
from langchain_core.tools import tool
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(model="gpt-5.5", base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])


@tool
def get_weather(city: str) -> str:
    """查询某个城市当前的天气。"""
    return f"{city}：晴，23°C"  # 换成真实的天气接口


llm_with_tools = llm.bind_tools([get_weather])
messages = [HumanMessage("北京今天天气怎么样？适合跑步吗？")]

ai = llm_with_tools.invoke(messages)
messages.append(ai)
for call in ai.tool_calls:
    messages.append(get_weather.invoke(call))  # 返回 ToolMessage
final = llm_with_tools.invoke(messages)
print(final.content)
```

要让模型自己多步调用工具、直到完成任务，用 LangChain 的 Agent（`langchain` 包的 `create_agent`）或 LangGraph，模型同样传上面这个 `llm`。原理见 [Agent 循环](/a/a6)。

## 嵌入（Embeddings）

做 RAG 知识库要用嵌入模型。先用 `/v1/models` 确认你的分组里有嵌入模型（名字里带 `embedding`），没有的话嵌入部分用其他服务商。

```python
import os
from langchain_openai import OpenAIEmbeddings

embeddings = OpenAIEmbeddings(
    model="text-embedding-3-small",  # 换成你的分组里的嵌入模型
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
    check_embedding_ctx_length=False,  # 直接发送原文，见下方说明
)
vectors = embeddings.embed_documents(["退货政策是 7 天无理由", "发货时间是下单后 48 小时内"])
query = embeddings.embed_query("多久发货？")
print(len(vectors), "条文档，向量维度", len(query))
```

::: tip 为什么要 check_embedding_ctx_length=False
默认情况下，`OpenAIEmbeddings` 会先在本地用 tiktoken 把文字切成 token 编号再发送（第一次运行还要从国外下载编码文件，国内网络常常卡住）。加上这个参数后直接发送原文，兼容性更好；每段文字的长度请自己控制在模型上限以内。
:::

## LangChain.js

```bash
npm install @langchain/openai @langchain/core
```

```js
import { ChatOpenAI } from '@langchain/openai'

const llm = new ChatOpenAI({
  model: 'gpt-5.5',
  apiKey: process.env.HIVEGPT_API_KEY,
  configuration: { baseURL: 'https://hivegpt.cn/v1' },
  maxRetries: 3,
})

const resp = await llm.invoke('用一句话解释什么是向量数据库')
console.log(resp.content)

for await (const chunk of await llm.stream('写一首关于秋天的四行小诗')) {
  process.stdout.write(chunk.content)
}
console.log()
```

JS 里的 `baseURL` 要放在 `configuration` 里，直接写成 `new ChatOpenAI({ baseURL })` 是不生效的。结构化输出用 `llm.withStructuredOutput(zodSchema)`，工具调用用 `llm.bindTools([...])`，用法和 Python 一致。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 401，提示 Key 无效 | `api_key` 没传进去，用了别的 `OPENAI_API_KEY` | 显式传 `api_key`，或检查环境变量，见 [401 报错](/connect/errors/401) |
| 请求跑到了 api.openai.com，连不上或报 Key 不对 | `base_url` 没生效（JS 里没放进 `configuration`） | 按上面的写法改 |
| 404，提示「接口地址少了 /v1」或「多了一个 /v1」 | `base_url` 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 404，提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| `OpenAIEmbeddings` 卡住不动或报 tiktoken 下载失败 | 在下载 tiktoken 编码文件 | 加 `check_embedding_ctx_length=False` |
| 400：`Unsupported parameter: 'temperature'` | 推理类模型只接受默认值 | 创建 `ChatOpenAI` 时不传 `temperature` |
| 429 | 并发太高，如 `batch` 一次跑太多 | 设 `max_concurrency`，见 [429 报错](/connect/errors/429) |

## 下一步

- 不用框架：[Python 调用 GPT API 完整示例](/connect/python)、[Node.js 调用 OpenAI 兼容接口](/connect/nodejs)
- RAG 怎么做：[Embedding 和 RAG 问答](/a/a5)、[RAG 专题](/guide/rag/)
- 系统学做 AI 应用：[A 路线 · AI 应用开发入门](/a/)
