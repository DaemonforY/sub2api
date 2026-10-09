---
title: 接入教程
description: 把 HiveGPT 接进 Codex、OpenCode、CodeBuddy、Cherry Studio 等工具或你自己的代码：需要准备什么、Base URL 和 Key 怎么填、选哪个分组、查可用模型，以及常见报错怎么处理。
---

# 接入教程

HiveGPT 提供和 OpenAI 一致的接口。几乎所有支持「自定义 OpenAI 接口」的工具和 SDK，填上 **Base URL** 和 **API Key** 就能用。

## 先准备好三样东西

| 准备 | 在哪里 | 说明 |
|---|---|---|
| 账号 | [注册 HiveGPT](https://hivegpt.cn/register?utm_source=learn&utm_medium=connect) | 邮箱注册即可 |
| 余额或订阅 | [充值 / 订阅](https://hivegpt.cn/purchase) | 两种方式的区别见 [订阅还是按量](https://hivegpt.cn/pricing) |
| API Key | [API 密钥](https://hivegpt.cn/keys?action=create) | 创建时选的**分组**决定能用哪些模型、怎么计费 |

::: tip 分组怎么选
- 买了订阅：选哪个都行。同一个 Key 在对话和 Codex 请求里会先用订阅额度，额度用完或到期后自动改用余额，不用为订阅和按量各建一个 Key。
- 按量付费：选「GPT-按量」，调用从余额里扣。
- 选错分组是「模型不可用」最常见的原因。一个账号可以建多个 Key，分别用不同分组。
:::

## 通用参数

不管用什么工具，需要填的都是这几项：

| 参数 | 填什么 |
|---|---|
| Base URL / API 地址 | `https://hivegpt.cn/v1` |
| API Key | 你在「API 密钥」页创建的 `sk-` 开头的 Key |
| 接口类型 | OpenAI 兼容（Chat Completions；Codex 用 Responses） |
| 模型 | 你的分组里可用的模型，例如 `gpt-5.5`，查法见下文 |

::: warning 有的工具会自己补 /v1
如果工具提示「接口地址多了一个 /v1」，把 Base URL 换成 `https://hivegpt.cn` 再试（去掉末尾的 `/v1`）；提示「少了 /v1」就反过来。各工具的具体填法见 [Base URL 末尾要不要加 /v1](/connect/base-url)。
:::

## 选一个教程

| 你想用在 | 看这篇 | 大约用时 |
|---|---|---|
| Codex（CLI、IDE 扩展、桌面端） | [Codex 接入 HiveGPT](/codex/hivegpt) | 5 分钟 |
| OpenCode、CodeBuddy、Cherry Studio、Chatbox 等工具 | [在常用工具里配置](/connect/tools) | 3 分钟 |
| 自己写代码（Python、Node.js、curl） | [用 SDK 调用](/connect/sdk)，完整代码见 [Python](/connect/python)、[Node.js](/connect/nodejs) | 5 分钟 |
| LangChain（Python / JS） | [LangChain 接入 HiveGPT](/connect/langchain) | 5 分钟 |
| Java、Spring AI | [Java 调用 GPT API](/connect/java) | 5 分钟 |
| Go | [Go 调用 GPT API](/connect/go) | 5 分钟 |
| 批量处理 Excel 表格 | [用 GPT 批量处理 Excel](/connect/excel) | 10 分钟 |
| 飞书、企业微信群机器人 | [群机器人接入 GPT](/connect/feishu-wecom-bot) | 10 分钟 |
| VS Code 里的 Cline / Roo Code | [Cline 配置自定义 API](/connect/cline) | 3 分钟 |
| Dify 应用和工作流 | [Dify 接入 OpenAI 兼容模型](/connect/dify) | 5 分钟 |
| Obsidian 笔记 | [Obsidian Copilot 接入](/connect/obsidian) | 3 分钟 |
| Zotero 翻译论文、读文献 | [Zotero 接入 GPT API](/connect/zotero) | 5 分钟 |
| n8n 自动化工作流 | [n8n 接入 OpenAI 兼容接口](/connect/n8n) | 5 分钟 |
| 自建聊天站（LobeChat、NextChat） | [LobeChat、NextChat 接入](/connect/lobechat-nextchat) | 10 分钟 |
| Cherry Studio 桌面客户端 | [Cherry Studio 配置教程](/connect/cherry-studio) | 3 分钟 |
| Chatbox 对话客户端 | [Chatbox 接入第三方 API](/connect/chatbox) | 3 分钟 |
| 沉浸式翻译（网页、PDF 翻译） | [沉浸式翻译接入 GPT API](/connect/immersive-translate) | 3 分钟 |

最省事的办法：在「API 密钥」页找到你的 Key，点 **「使用密钥」**。弹窗会按你的分组和系统，直接生成 Codex、OpenCode 等客户端的配置，复制粘贴即可。

## 查看可用模型

不同分组能用的模型不一样，以接口返回为准：

```bash
curl https://hivegpt.cn/v1/models \
  -H "Authorization: Bearer $HIVEGPT_API_KEY"
```

返回的 `data[].id` 就是可以填进工具里的模型名。

## 验证 Key 能用

```bash
curl https://hivegpt.cn/v1/chat/completions \
  -H "Authorization: Bearer $HIVEGPT_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-5.5", "messages": [{"role": "user", "content": "用一句话介绍你自己"}]}'
```

能看到回答，说明 Key、余额（或订阅）和网络都没问题；控制台首页的「三步开始使用」也会自动打勾。之后在任何工具里出问题，基本就是那个工具的配置写错了。

## 常见报错

| 现象 | 原因 | 处理 |
|---|---|---|
| 提示 Key 无效、未授权（401） | Key 复制不全、已删除或已停用 | 到「API 密钥」页确认状态，重新复制完整的 Key，见 [401 报错](/connect/errors/401) |
| 提示某个模型不支持或不存在 | 这个模型不在你 Key 的分组里 | 用上面的 `/v1/models` 查可用模型，或换一个分组的 Key，见 [模型不存在](/connect/errors/model) |
| 提示余额不足、额度已用完 | 余额为 0，或订阅的每日 / 每周 / 每月额度用完 | 充值，或等额度恢复；充值后同一个 Key 会自动改用余额继续，见 [余额 / 额度](/connect/errors/quota) |
| 提示限流、请稍后再试（429） | 请求太快，或上游临时限流 | 过 1–2 分钟再试；这类失败不扣费，见 [429 报错](/connect/errors/429) |
| 上游暂时不可用（502 / 503） | 模型服务临时故障 | 稍后重试；这类失败不扣费 |
| 一直转圈、回复很慢 | 上游繁忙，或者上下文太长 | 换一个较快的模型，或开新会话减少上下文 |

按报错代码查原因，看 [报错速查](/connect/errors/)；费用怎么算看 [token 怎么算钱](/connect/tokens)，和 ChatGPT 会员的区别看 [API 和 ChatGPT Plus](/connect/api-vs-plus)。每次请求的模型、用量和费用都能在 [使用记录](https://hivegpt.cn/usage) 里查到。还有问题，可以问首页右下角的智能客服。

::: danger 保护好你的 Key
Key 等同于余额。不要发到群里、截图里或提交到 Git 仓库。如果泄露了，立刻在「API 密钥」页删除，再建一个新的。
:::
