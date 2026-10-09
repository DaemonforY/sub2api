---
title: 报错 context_length_exceeded：上下文太长、maximum context length 怎么解决
description: 调用 GPT 等大模型 API 时报 context_length_exceeded、This model's maximum context length is …、input too long、上下文超出限制的原因和解决方法：清理对话历史、只保留最近几轮、截取文档、用 RAG，以及在 Codex、Cherry Studio 等工具里怎么开新会话。
---

# 上下文太长（context_length_exceeded）

**一次请求里的「输入 + 输出」超过了模型的上下文上限。** 最快的解决办法是开一个新对话，或删掉前面的历史消息；处理长文档时只发和问题相关的部分，或者用 RAG 检索。

> 更新于 2026-10。

## 为什么会超

模型每次请求都要把以下内容一起读进去，它们加起来就是「上下文」：

- 系统提示词（工具自带的往往很长）
- 之前的全部对话
- 你上传或粘贴的文件、网页内容
- 这次的问题
- 再加上这次要输出的内容（`max_completion_tokens`）

`gpt-5.5` 的上下文上限约 105 万 token，日常对话很难超；但上限小一些的模型，或者一次塞进好几本书、一个大代码仓库时，就会超。各模型的上限看 [模型广场](https://hivegpt.cn/model-plaza)。

## 常见的报错

| 报错 | 意思 |
|---|---|
| `context_length_exceeded` | 输入加输出超过了上限 |
| `This model's maximum context length is … tokens. However, you requested … tokens` | 同上，会写出上限和你这次用了多少 |
| `input too long`、`prompt is too long` | 输入本身就超了 |
| `max_tokens is too large` | 输出上限设得比模型允许的大 |

## 解决方法

**在聊天工具里**

1. **开新对话。** 换话题时开新对话，旧对话的内容不会再被带上。
2. **限制带上的历史条数。** Cherry Studio、Chatbox 等客户端的对话设置里可以设「上下文数量」，比如只带最近 10 条。
3. **长文档分段问。** 不要一次粘贴整本书，分章节问，或者先让模型逐段总结，再基于总结提问。

**在编程工具里（Codex、Cline 等）**

1. 一个任务做完就开新会话 / 新任务。
2. 用工具的压缩功能，如 Codex 的 `/compact`，见 [Codex 斜杠命令](/codex/slash-commands)。
3. 在项目里用 `.gitignore` 等方式排除大文件、构建产物和依赖目录。

**自己写代码时**

```python
MAX_TURNS = 10

def trim_history(messages):
    """保留系统消息和最近 MAX_TURNS 轮对话。"""
    system = [m for m in messages if m["role"] == "system"]
    rest = [m for m in messages if m["role"] != "system"]
    return system + rest[-MAX_TURNS * 2:]
```

- 发送前调用 `trim_history`，只带最近几轮。
- 文档太长时，先切成小段，用 [嵌入和 RAG](/a/a5) 找出相关的几段再发给模型。
- 把 `max_completion_tokens` 设成实际需要的长度，不要设成模型的最大值。

## 上下文长 = 更贵

即使没有超限，上下文越长每次请求越贵：对话第 10 轮时，前 9 轮的内容每次都要重新算钱。控制上下文既能避免报错也能省钱，算法见 [token 怎么算钱](/connect/tokens)。

其他报错看 [报错速查](/connect/errors/)。
