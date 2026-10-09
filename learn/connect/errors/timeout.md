---
title: API 请求超时、流式输出中断怎么办：timeout、stream disconnected、502 排查
description: 调用 HiveGPT 等 OpenAI 兼容接口时请求超时（timeout、Request timed out、APITimeoutError）、流式输出中途断开（stream disconnected）、长回答报 502 / 504 的原因和处理：长输出用流式、调大客户端超时、检查代理，失败请求不扣费。
---

# 请求超时、流式输出中断

**长回答一定要用流式输出（`stream: true`）。** 非流式请求要等模型把整段写完才返回，超过约 2 分钟就可能被中途的代理断开（502）；流式输出边生成边返回，几分钟的长回答也能完整收到。其次，把客户端的超时调到 5 分钟以上，并检查本机代理是否稳定。超时失败的请求不扣费。

> 更新于 2026-10。

## 先看是哪一种

| 现象 | 多半是 | 处理 |
|---|---|---|
| 等了 1–2 分钟后报 502，内容提到 Proxy Read Timeout / 120 秒 | 非流式请求，回答太长 | 改用流式输出 |
| 客户端报 `timeout`、`Request timed out`、`APITimeoutError` | 客户端自己的超时设得太短 | 调大客户端超时，或改用流式 |
| 流式输出写到一半停了、报 `stream disconnected` | 网络或本机代理中断 | 检查代理；在工具里重试，SDK 设置自动重试 |
| 报 504「上游生成超时（超过 2 分钟没有返回）」 | 画图时尺寸、质量太高或上游繁忙 | 调低尺寸 / 质量，稍后重试 |
| 很久才开始出字，但能正常完成 | 推理模型在思考，或上游排队 | 正常；调低推理强度会快一些 |

## HiveGPT 这一侧的时间限制

- **等待上游开始响应：最长 10 分钟。** 推理模型思考时间长，不会因为「还没出第一个字」被过早断开。
- **流式输出：每 10 秒发一次保活信号**，防止中间的代理因为「一直没数据」而断开连接；如果上游连续 3 分钟没有任何输出，才会结束这次请求。
- **非流式请求：最好控制在 2 分钟以内。** 上游链路对单次非流式响应有约 120 秒的读取上限，超过会返回 502。

实际情况：最近一周，HiveGPT 上最长的一次流式请求持续了 7 分多钟并正常完成，而非流式请求最长约 2.5 分钟。

## 各工具里怎么改

**Python SDK**

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key="sk-你的Key",
    timeout=600,     # 单次请求最长等 10 分钟
    max_retries=3,   # 超时、502 / 503 自动重试
)

stream = client.chat.completions.create(
    model="gpt-5.5",
    messages=[{"role": "user", "content": "写一篇 3000 字的产品方案"}],
    stream=True,     # 长回答用流式
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

**Node.js SDK**：`new OpenAI({ baseURL, apiKey, timeout: 600_000, maxRetries: 3 })`，请求里加 `stream: true`，见 [Node.js 完整示例](/connect/nodejs#流式输出)。

**curl**：默认不会超时，但非流式请求同样受 2 分钟的限制，长回答加上 `"stream": true`。

**桌面客户端（Cherry Studio、Chatbox 等）**：一般默认就是流式输出；如果关掉过「流式输出」开关，重新打开。

**沉浸式翻译、Zotero 等翻译插件**：单次翻译很短，超时多半是网络或并发太高，调低「每秒最大请求数」，见 [沉浸式翻译](/connect/immersive-translate)。

**Codex**：Codex 本身就用流式。频繁出现 `stream disconnected` 时，看 [Codex 常见问题排查](/codex/troubleshooting)。

## 检查本机网络和代理

很多「超时」其实是本机网络的问题：

1. **浏览器打开 https://hivegpt.cn 是否正常。** 打不开就是网络问题。
2. **终端里设置过代理吗？** 如果设置了 `HTTPS_PROXY` / `HTTP_PROXY`，但代理软件没开或不稳定，所有请求都会超时。HiveGPT 在国内可以直接访问，不需要代理：

   ```bash
   unset HTTPS_PROXY HTTP_PROXY https_proxy http_proxy
   ```

3. **公司或学校网络**有时会断开长时间的连接，换个网络试一下。

## 超时扣钱吗

不扣。超时、502、504 的请求都不计费，可以在 [使用记录](https://hivegpt.cn/usage) 里确认。流式输出中途断开时，已经生成的部分按实际输出计费，见 [token 怎么算钱](/connect/tokens)。

其他报错看 [报错速查](/connect/errors/)。
