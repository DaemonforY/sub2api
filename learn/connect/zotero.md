---
title: Zotero 接入 GPT API：翻译论文和 AI 读文献插件配置教程
description: 在 Zotero 7 里用 Translate for Zotero（自定义GPT）翻译 PDF 选中文字、标题和摘要，用 llm-for-zotero 就当前论文提问：接口地址填 https://hivegpt.cn/v1、密钥和模型名怎么填，以及 Request error、密钥格式不对等常见问题。
---

# Zotero 接入 GPT API

**翻译用 Translate for Zotero，提问用 llm-for-zotero，两个插件都填 `https://hivegpt.cn/v1` 和你的 HiveGPT Key。** 翻译插件里要选 **自定义GPT1**（不要选 ChatGPT），接口填 `https://hivegpt.cn/v1/chat/completions`、模型填 `gpt-5.5`；读文献插件选 Customized 服务商、协议选 OpenAI-Compatible Chat，API 基础 URL 填 `https://hivegpt.cn/v1`。

> 更新于 2026-10，基于 Zotero 7、Translate for Zotero 2.4.x 和 llm-for-zotero 3.9.x。插件更新快，菜单名称以你看到的为准。

## 准备

- [Zotero 7](https://www.zotero.org/download/)（Zotero 8 也可以）。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=zotero)。

两个插件都从 GitHub Releases 下载 `.xpi` 文件安装：Zotero 菜单 **工具 → 插件**，点右上角齿轮，选 **从文件安装插件**，选中下载的 `.xpi`。

## 一、翻译：Translate for Zotero

下载：[zotero-pdf-translate Releases](https://github.com/windingwind/zotero-pdf-translate/releases)，文件名是 `translate-for-zotero.xpi`。

### 配置

1. 打开 **编辑 → 设置**（macOS 是 Zotero → 设置），切到 **翻译** 标签。
2. **翻译服务** 选 **自定义GPT1**，点旁边的 **配置**。
3. 按下表填写：

| 字段 | 填什么 |
|---|---|
| 接口 | `https://hivegpt.cn/v1/chat/completions` |
| 模型 | `gpt-5.5`，或分组里更便宜的模型 |
| 温度 | 保持默认 `1.0` |
| 接口格式 | OpenAI (chat/completions) |
| 提示词 | 保持默认，必须包含 `${sourceText}` |
| 流式输出 | 打开 |

4. 回到翻译设置，在 **密钥** 里填你的 HiveGPT Key。

::: warning 为什么不选 ChatGPT
翻译服务里的 ChatGPT 选项只接受 OpenAI 官方格式的 Key，第三方的 Key 会被判为格式不对。用 **自定义GPT1 / 2 / 3** 就没有这个限制，还可以分别配不同的模型。
:::

接口也可以只填 `https://hivegpt.cn/v1`，新版插件会自动补 `/chat/completions`；填完整地址在新旧版本里都能用。

### 使用

- **翻译选中文字**：在 PDF 里选中一段，弹出的翻译框和右侧面板里就会出现译文；也可以按 `Ctrl+T`（macOS 是 `Cmd+T`）。
- **翻译标题和摘要**：在条目列表里右键，选 **翻译标题** 或 **翻译摘要**。
- **翻译更准**：打开 **向GPT/Claude/Gemini服务提供论文标题和摘要**，翻译时会带上论文背景，专业术语更准确（每次请求会多用一些 token）。

## 二、AI 读文献：llm-for-zotero

下载：[llm-for-zotero Releases](https://github.com/yilewang/llm-for-zotero/releases)，安装后重启 Zotero。中文文档见 [llm-for-zotero 文档](https://yilewang.github.io/llm-for-zotero/zh/)。

### 配置

1. 打开 **设置 → llm-for-zotero**。
2. 服务商选 **Customized**，协议选 **OpenAI-Compatible Chat**。
3. 按下表填写：

| 字段 | 填什么 |
|---|---|
| API 基础 URL | `https://hivegpt.cn/v1` |
| 密钥 / API Key | 你的 HiveGPT Key |
| 模型名称 | `gpt-5.5` |

4. 点 **测试连接**，显示成功就配好了。

### 使用

1. 打开一篇论文的 PDF，点右侧工具栏的 **LLM Assistant** 图标。
2. 第一次提问时，插件会把这篇论文作为上下文发给模型，直接问就行：

> 用中文总结这篇论文的研究问题、方法和主要结论，各用 2–3 句话。

3. 也可以选中一段文字加入对话，问「这段公式是什么意思」「这个实验设计有什么局限」。

::: tip 注意花费
整篇论文作为上下文，一次提问可能有几万 token。同一篇论文连续追问，比每次重新打开更省（能命中缓存）。看 [token 怎么算钱](/connect/tokens)。
:::

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 翻译插件提示密钥格式不对 | 选了 ChatGPT 服务 | 改用 **自定义GPT1** |
| 翻译显示 Request error: 404 | 接口地址写错 | 填 `https://hivegpt.cn/v1/chat/completions`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| Request error: 401 | Key 不对或已停用 | 见 [401 报错](/connect/errors/401) |
| Request error: 400，提到 temperature | 推理类模型不接受改过的温度 | 温度改回 `1.0` |
| Request error: 429 | 一次翻译太多（如整篇翻译）或额度用完 | 见 [429 报错](/connect/errors/429)、[余额 / 额度](/connect/errors/quota) |
| llm-for-zotero 看不到论文内容 | PDF 标签页没有加载好 | 关掉 PDF 重新打开，再发一条新消息 |
| 提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |

翻译网页而不是 PDF，用 [沉浸式翻译](/connect/immersive-translate)。
