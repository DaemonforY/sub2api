---
title: Responses API 教程：和 Chat Completions 有什么区别、怎么迁移（Python 示例）
description: OpenAI Responses API 怎么用：和 Chat Completions 的区别与选择，Python 示例覆盖基本调用、instructions、多轮对话（previous_response_id 不可用时怎么带历史）、流式事件、函数调用、JSON Schema 结构化输出、推理强度，以及在 HiveGPT 上哪些功能可用。
---

# Responses API 教程

**先说结论：** Responses API（`/v1/responses`）是 OpenAI 新一代接口，GPT-6 等新模型官方推荐用它；老的 Chat Completions（`/v1/chat/completions`）依然可用，兼容的工具最多。HiveGPT 两个都支持。**已有代码用 Chat Completions 跑得好好的，不用急着改；新项目、要用推理模型和工具调用的，建议用 Responses。**

> 更新于 2026-10，本页每个示例都在 HiveGPT 上用 openai Python 3.26 实际跑过。

## 两个接口有什么区别

| | Chat Completions | Responses |
|---|---|---|
| 地址 | `/v1/chat/completions` | `/v1/responses` |
| 输入 | `messages` 数组 | `input`：一个字符串，或消息数组 |
| 系统提示词 | `{"role": "system"}` 消息 | `instructions` 参数 |
| 取回答 | `resp.choices[0].message.content` | `resp.output_text` |
| 函数调用 | `tools` 里套一层 `function` | `tools` 直接写 `name`、`parameters` |
| 结构化输出 | `response_format` | `text.format` |
| 推理强度 | `reasoning_effort` | `reasoning={"effort": ...}` |
| 第三方工具支持 | 几乎所有工具 | Codex、OpenCode 等较新的工具 |

## 基本调用

```python
import os
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])

resp = client.responses.create(
    model="gpt-6.1-sol",
    instructions="用一句话回答",
    input="天空为什么是蓝的？",
)
print(resp.output_text)
print(resp.usage.input_tokens, resp.usage.output_tokens)
```

`output_text` 把所有文本输出拼好了，大多数时候用它就够。完整结构在 `resp.output` 里，是一个列表，可能包含推理摘要、文本消息、函数调用等不同类型的项。

## 多轮对话

OpenAI 官方的 Responses 可以用 `previous_response_id` 让服务端记住上一轮。**HiveGPT 目前不支持这个参数**，会报 400：`previous_response_id requires an OpenAI API-key account`。

做法和 Chat Completions 一样：自己保存历史，每次把完整记录放进 `input`：

```python
history = [{"role": "user", "content": "记住：我叫小明。只回复好的"}]
r1 = client.responses.create(model="gpt-6.1-sol", input=history, store=False)

history += [
    {"role": "assistant", "content": r1.output_text},
    {"role": "user", "content": "我叫什么？"},
]
r2 = client.responses.create(model="gpt-6.1-sol", input=history, store=False)
print(r2.output_text)  # 小明
```

`store=False` 表示不在服务端保存这次对话。历史越长，每次请求的输入 token 越多，聊太久就总结一下前文或者只保留最近几轮，见 [上下文太长](/connect/errors/context-length)。

## 流式输出

```python
with client.responses.stream(model="gpt-6.1-sol", input="写一首关于秋天的短诗") as stream:
    for event in stream:
        if event.type == "response.output_text.delta":
            print(event.delta, end="", flush=True)
    final = stream.get_final_response()
print("\n用量：", final.usage.total_tokens)
```

Responses 的流式是「事件」：`response.created`、`response.output_text.delta`（一段新文字）、`response.completed`（结束，带用量）等。只关心文字的话，处理 `response.output_text.delta` 即可。

## 函数调用

和 Chat Completions 的区别：工具定义少一层 `function`；模型的调用和你的执行结果都作为 `input` 里的项传回去。

```python
import json

tools = [{
    "type": "function",
    "name": "get_weather",
    "description": "查询城市当前天气",
    "parameters": {
        "type": "object",
        "properties": {"city": {"type": "string"}},
        "required": ["city"],
    },
}]

question = {"role": "user", "content": "北京天气怎样？"}
r1 = client.responses.create(model="gpt-6.1-sol", input=[question], tools=tools, store=False)

call = next(o for o in r1.output if o.type == "function_call")
args = json.loads(call.arguments)            # {"city": "北京"}
result = {"temp": "18°C", "sky": "晴"}       # 换成你真正查到的结果

r2 = client.responses.create(
    model="gpt-6.1-sol",
    tools=tools,
    store=False,
    input=[
        question,
        call.model_dump(exclude_none=True),  # 模型发起的调用，原样传回
        {"type": "function_call_output", "call_id": call.call_id, "output": json.dumps(result, ensure_ascii=False)},
    ],
)
print(r2.output_text)  # 北京现在晴，18°C。
```

`call_id` 要对上，模型才知道这个结果对应哪次调用。

## 结构化输出（JSON Schema）

要求模型严格按格式返回 JSON：

```python
resp = client.responses.create(
    model="gpt-6.1-sol",
    input="张三，28岁，在杭州做设计师。提取信息",
    text={"format": {
        "type": "json_schema",
        "name": "person",
        "strict": True,
        "schema": {
            "type": "object",
            "properties": {
                "name": {"type": "string"},
                "age": {"type": "integer"},
                "city": {"type": "string"},
            },
            "required": ["name", "age", "city"],
            "additionalProperties": False,
        },
    }},
)
print(json.loads(resp.output_text))  # {'age': 28, 'city': '杭州', 'name': '张三'}
```

`strict: True` 时，`properties` 里的每个字段都要写进 `required`，并且要有 `"additionalProperties": False`。

## 推理强度

```python
resp = client.responses.create(
    model="gpt-6.1-sol",
    input="17×23 等于多少？",
    reasoning={"effort": "low"},
)
print(resp.usage.output_tokens_details.reasoning_tokens)
```

推理（「思考」）消耗的 token 算作输出，按输出价计费。简单任务用 `low`，难题再提高，各模型支持的档位见 [GPT 模型怎么选](/connect/models)。

## 看图、读 PDF

`input` 里可以放图片和文件，见 [GPT 看图识图 API](/connect/vision)。

## 在 HiveGPT 上可用的功能

| 功能 | 状态 |
|---|---|
| 文本、`instructions`、流式 | 可用 |
| 函数调用、JSON Schema 结构化输出 | 可用 |
| `reasoning.effort` | 可用 |
| 图片、PDF 输入 | 可用 |
| `previous_response_id` | 不可用，自己带历史 |
| 联网搜索（`web_search` 工具） | 不可用 |
| 生成图片 | 用 [图片 API](/connect/image-api) |

## 常见问题

**Responses 比 Chat Completions 贵吗？** 不会，同一个模型价格一样，按 token 计费。

**已有的 Chat Completions 代码要改吗？** 不用。HiveGPT 上 GPT-6 系列通过 Chat Completions 调用、带工具调用都正常。只有想用新接口的写法时再迁移。

**报 400：`previous_response_id requires an OpenAI API-key account` 怎么办？** 去掉 `previous_response_id`，按上面「多轮对话」的写法自己带历史记录。

**第三方工具里选「Responses」还是「Chat Completions」？** 工具支持 Responses（如 Codex、OpenCode）就用 Responses，否则选 Chat Completions，效果一样。

**Node.js 怎么写？** 方法名一样：`client.responses.create({ model, input, instructions })`，取 `resp.output_text`，见 [Node.js 完整示例](/connect/nodejs)。

更多：[Python 完整示例](/connect/python) · [用 SDK 调用](/connect/sdk) · [报错速查](/connect/errors/)
