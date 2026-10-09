---
title: 用 SDK 调用 HiveGPT
description: 用 OpenAI 官方 Python / Node.js SDK 或 curl 调用 HiveGPT：对话、流式输出、Responses 接口和生成图片，只需改 base_url 和 API Key。
---

# 用 SDK 调用

HiveGPT 的接口和 OpenAI 一致，直接用 OpenAI 官方 SDK，只改两处：`base_url` 和 API Key。

这页是速览。多轮对话、JSON 输出、函数调用、并发、错误处理的完整代码见 [Python 完整示例](/connect/python)、[Node.js 完整示例](/connect/nodejs)；用框架的看 [LangChain 接入](/connect/langchain)；Java、Go 见 [Java](/connect/java)、[Go](/connect/go)。

先把 Key 放进环境变量，不要写死在代码里：

::: code-group

```bash [macOS / Linux]
export HIVEGPT_API_KEY="sk-你的Key"
```

```powershell [Windows PowerShell]
$env:HIVEGPT_API_KEY="sk-你的Key"
```

:::

## Python

```bash
pip install openai
```

```python
import os
from openai import OpenAI

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
)

# 对话
resp = client.chat.completions.create(
    model="gpt-5.5",
    messages=[{"role": "user", "content": "用三句话解释什么是 API"}],
)
print(resp.choices[0].message.content)

# 流式输出：边生成边打印
stream = client.chat.completions.create(
    model="gpt-5.5",
    messages=[{"role": "user", "content": "写一首关于秋天的四行小诗"}],
    stream=True,
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

## Node.js

```bash
npm install openai
```

```js
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: 'https://hivegpt.cn/v1',
  apiKey: process.env.HIVEGPT_API_KEY,
})

const resp = await client.chat.completions.create({
  model: 'gpt-5.5',
  messages: [{ role: 'user', content: '用三句话解释什么是 API' }],
})
console.log(resp.choices[0].message.content)
```

## Responses 接口

Codex 等新工具用的是 Responses 接口，HiveGPT 同样支持：

```python
resp = client.responses.create(
    model="gpt-5.5",
    input="给我三个学习 Python 的小项目点子",
)
print(resp.output_text)
```

## 生成图片

分组支持画图时，可以调用图片接口（模型名以 `/v1/models` 返回的为准，例如 `gpt-image-2`）：

```python
import base64

img = client.images.generate(
    model="gpt-image-2",
    prompt="一只在图书馆看书的橘猫，水彩风格",
    size="1024x1024",
)
with open("cat.png", "wb") as f:
    f.write(base64.b64decode(img.data[0].b64_json))
```

画图按张计费，比对话贵，建议先用小尺寸试效果。也可以直接用 [无限画布](https://canvas.hivegpt.cn/?utm_source=learn&utm_medium=connect)，不用写代码。

## curl

```bash
curl https://hivegpt.cn/v1/chat/completions \
  -H "Authorization: Bearer $HIVEGPT_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-5.5", "messages": [{"role": "user", "content": "你好"}]}'
```

## 下一步

- 想系统地学怎么用 API 做应用：看 [A 路线 · AI 应用开发入门](/a/)，课里的例子可以在页面上直接运行。
- 每次调用的模型、Token 和费用，都在 [使用记录](https://hivegpt.cn/usage) 里。
- 报错排查见 [报错速查](/connect/errors/)。
