---
title: Node.js 调用 OpenAI 兼容接口完整示例：对话、流式、JSON、函数调用、Express 转发
description: 用 OpenAI 官方 Node.js / TypeScript SDK 调用 HiveGPT 等 OpenAI 兼容接口的完整代码：baseURL 怎么填、流式输出、多轮对话、用 zod 返回结构化 JSON、Function Calling、并发控制、重试和错误处理，以及用 Express 把流式回答转发给前端（Key 不暴露在浏览器里）。
---

# Node.js 调用 OpenAI 兼容接口

**只改两处就能用：** 安装 `openai` 包，创建客户端时把 `baseURL` 设为 `https://hivegpt.cn/v1`、`apiKey` 设为你的 HiveGPT Key，其余代码和调用 OpenAI 官方接口完全一样。

```js
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: 'sk-你的Key' })
const resp = await client.chat.completions.create({
  model: 'gpt-5.5',
  messages: [{ role: 'user', content: '用一句话介绍你自己' }],
})
console.log(resp.choices[0].message.content)
```

下面每段代码都是完整的文件，保存成 `.mjs` 就能用 `node 文件名.mjs` 运行。

> 更新于 2026-10，基于 `openai` Node.js SDK 7.x，Node.js 20 及以上。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备

```bash
npm install openai
```

代码用的是 ES Module（`import`）：文件名用 `.mjs`，或在 `package.json` 里加 `"type": "module"`。

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

::: warning 不要在浏览器里直接调用
Key 写进前端代码，任何人打开网页都能看到。前端页面要用 AI，就让你的后端去调用 HiveGPT，见下文 [用 Express 转发给前端](#用-express-转发给前端)。
:::

## 基本对话

```js
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: 'https://hivegpt.cn/v1',
  apiKey: process.env.HIVEGPT_API_KEY,
})

const resp = await client.chat.completions.create({
  model: 'gpt-5.5',
  messages: [
    { role: 'system', content: '你是一位耐心的编程老师，回答简洁。' },
    { role: 'user', content: 'JavaScript 的 let 和 const 有什么区别？' },
  ],
})
console.log(resp.choices[0].message.content)
console.log('本次用量：', resp.usage.total_tokens, 'tokens')
```

不传 `baseURL` 和 `apiKey` 时，SDK 会读环境变量 `OPENAI_BASE_URL` 和 `OPENAI_API_KEY`，所以现成的项目把这两个变量设成 HiveGPT 的地址和 Key 也能直接用。

## 流式输出

```js
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })

const stream = await client.chat.completions.create({
  model: 'gpt-5.5',
  messages: [{ role: 'user', content: '写一首关于秋天的四行小诗' }],
  stream: true,
  stream_options: { include_usage: true }, // 最后一块带上用量
})
for await (const chunk of stream) {
  const text = chunk.choices[0]?.delta?.content
  if (text) process.stdout.write(text)
  if (chunk.usage) console.log('\n本次用量：', chunk.usage.total_tokens, 'tokens')
}
```

## 多轮对话

模型不记得上一轮说了什么，**每次请求都要把之前的对话一起发过去**：

```js
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })

const messages = [{ role: 'system', content: '你是一位旅行规划助手。' }]

for (const question of ['我五月想去云南玩 5 天', '预算 5000 元够吗？', '那第一天怎么安排？']) {
  messages.push({ role: 'user', content: question })
  const resp = await client.chat.completions.create({ model: 'gpt-5.5', messages })
  const answer = resp.choices[0].message.content
  messages.push({ role: 'assistant', content: answer })
  console.log(`问：${question}\n答：${answer}\n`)
}
```

对话越长，每次发送的 token 越多、越贵，长对话可以只保留最近几轮。

## 让模型返回 JSON

用 [zod](https://zod.dev) 描述结构，`parse` 会让模型按结构输出并解析成对象：

```bash
npm install zod
```

```js
import OpenAI from 'openai'
import { zodResponseFormat } from 'openai/helpers/zod'
import { z } from 'zod'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })

const Contact = z.object({
  name: z.string(),
  phone: z.string(),
  city: z.string(),
})

const resp = await client.chat.completions.parse({
  model: 'gpt-5.5',
  messages: [
    { role: 'system', content: '从用户的话里提取联系人信息。' },
    { role: 'user', content: '我是李雷，住在杭州，电话 138 0000 1234' },
  ],
  response_format: zodResponseFormat(Contact, 'contact'),
})
const contact = resp.choices[0].message.parsed
console.log(contact.name, contact.phone, contact.city)
```

## 函数调用（Function Calling）

```js
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })

function getWeather({ city }) {
  // 换成真实的天气接口
  return { city, weather: '晴', temperature: 23 }
}

const tools = [{
  type: 'function',
  function: {
    name: 'get_weather',
    description: '查询某个城市当前的天气',
    parameters: {
      type: 'object',
      properties: { city: { type: 'string', description: '城市名，如 北京' } },
      required: ['city'],
    },
  },
}]

const messages = [{ role: 'user', content: '北京今天天气怎么样？适合跑步吗？' }]
let resp = await client.chat.completions.create({ model: 'gpt-5.5', messages, tools })
const msg = resp.choices[0].message

if (msg.tool_calls?.length) {
  messages.push(msg)
  for (const call of msg.tool_calls) {
    const result = getWeather(JSON.parse(call.function.arguments))
    messages.push({ role: 'tool', tool_call_id: call.id, content: JSON.stringify(result) })
  }
  resp = await client.chat.completions.create({ model: 'gpt-5.5', messages, tools })
}
console.log(resp.choices[0].message.content)
```

原理和更多例子见 [Function Calling：让模型调用你的函数](/a/a4)。

## 批量处理时控制并发

一次把几百个请求全发出去会触发 [429](/connect/errors/429)。用一个简单的工作池，最多同时跑 5 个：

```js
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })

async function translate(text) {
  const resp = await client.chat.completions.create({
    model: 'gpt-5.5',
    messages: [{ role: 'user', content: `翻译成英文，只输出译文：${text}` }],
  })
  return resp.choices[0].message.content
}

async function mapWithLimit(items, limit, fn) {
  const results = new Array(items.length)
  let next = 0
  const worker = async () => {
    while (next < items.length) {
      const i = next++
      results[i] = await fn(items[i])
    }
  }
  await Promise.all(Array.from({ length: limit }, worker))
  return results
}

const texts = ['今天天气很好', '我们明天见', '这个价格可以接受', '请尽快回复']
const results = await mapWithLimit(texts, 5, translate)
texts.forEach((t, i) => console.log(t, '→', results[i]))
```

## 超时、重试和错误处理

SDK 遇到 429 和 5xx 会自动重试（默认 2 次），可以调大；`err.error` 是 HiveGPT 返回的报错内容：

```js
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: 'https://hivegpt.cn/v1',
  apiKey: process.env.HIVEGPT_API_KEY,
  maxRetries: 5,   // 429 / 5xx 自动重试，间隔逐次加长
  timeout: 120_000, // 单次请求最长等 120 秒
})

try {
  const resp = await client.chat.completions.create({
    model: 'gpt-5.5',
    messages: [{ role: 'user', content: '你好' }],
  })
  console.log(resp.choices[0].message.content)
} catch (err) {
  if (err instanceof OpenAI.APIConnectionError) {
    console.error('连不上服务器，检查网络和代理')
  } else if (err instanceof OpenAI.APIError) {
    // 中文说明，括号里是英文原文
    console.error(err.status, err.error?.message ?? err.message)
  } else {
    throw err
  }
}
```

按状态码处理：401 是 Key 的问题，404 是地址或模型名的问题，429 是请求太快或额度用完。各种报错的含义见 [报错速查](/connect/errors/)。

## 用 Express 转发给前端

前端页面要做 AI 对话时，由后端持有 Key、调用 HiveGPT，再把流式回答原样转给浏览器：

```bash
npm install express openai
```

```js
import express from 'express'
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: 'https://hivegpt.cn/v1', apiKey: process.env.HIVEGPT_API_KEY })
const app = express()
app.use(express.json())

app.post('/api/chat', async (req, res) => {
  // 实际项目里先校验登录用户、限制频率和消息长度
  const messages = req.body.messages ?? []
  res.setHeader('Content-Type', 'text/plain; charset=utf-8')
  try {
    const stream = await client.chat.completions.create({ model: 'gpt-5.5', messages, stream: true })
    for await (const chunk of stream) {
      const text = chunk.choices[0]?.delta?.content
      if (text) res.write(text)
    }
  } catch (err) {
    res.write(`\n[出错了：${err.error?.message ?? err.message}]`)
  }
  res.end()
})

app.listen(3000, () => console.log('http://localhost:3000'))
```

前端用 `fetch` 读取流：

```js
const resp = await fetch('/api/chat', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ messages: [{ role: 'user', content: '你好' }] }),
})
const reader = resp.body.getReader()
const decoder = new TextDecoder()
for (;;) {
  const { value, done } = await reader.read()
  if (done) break
  document.querySelector('#answer').textContent += decoder.decode(value, { stream: true })
}
```

## TypeScript

SDK 自带类型，上面的代码改成 `.ts` 直接能用。消息数组可以这样标注类型：

```ts
import OpenAI from 'openai'
import type { ChatCompletionMessageParam } from 'openai/resources/chat/completions'

const messages: ChatCompletionMessageParam[] = [{ role: 'user', content: '你好' }]
```

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `SyntaxError: Cannot use import statement outside a module` | 文件被当成 CommonJS | 文件名改成 `.mjs`，或在 `package.json` 里加 `"type": "module"` |
| `APIConnectionError: Connection error.` | 网络不通，或代理没配置好 | 浏览器能打开 hivegpt.cn 的话，检查终端的代理设置 |
| 404，报错说「接口地址少了 /v1」或「多了一个 /v1」 | `baseURL` 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 401 | Key 不对，或环境变量没读到（`apiKey` 是 `undefined`） | 见 [401 报错](/connect/errors/401)；在运行 node 的同一个终端里设置环境变量 |
| 404，报错说分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 400：`Unsupported parameter: 'max_tokens'` | 新模型不接受 `max_tokens` | 改用 `max_completion_tokens` |

## 下一步

- Python 版本：[Python 调用 GPT API 完整示例](/connect/python)
- 用框架搭应用：[LangChain 接入 HiveGPT](/connect/langchain)
- 系统学做 AI 应用：[A 路线 · AI 应用开发入门](/a/)
