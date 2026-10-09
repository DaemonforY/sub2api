---
title: Obsidian Copilot 接入 GPT API：自定义 OpenAI 兼容服务商配置教程
description: 在 Obsidian 的 Copilot 插件（V4）里用 BYOK「Add a custom provider」接入 HiveGPT：Base URL 填 https://hivegpt.cn/v1、API key、Model ID，Enable CORS 保持关闭，在 Quick Chat 里用 GPT 总结、改写、问笔记；附旧版本的填法和常见报错。
---

# Obsidian Copilot 接入 GPT API

**四步接入：** 在 Obsidian 的「设置 → Copilot → BYOK」里点 **Add a provider → Add a custom provider**；**Base URL** 填 `https://hivegpt.cn/v1`，**API key** 填你的 HiveGPT Key，**Model ID** 填 `gpt-5.5` 后点 Add；**Enable CORS** 保持关闭，保存。然后在 Copilot 的 Quick Chat 里选这个模型就能用。

> 更新于 2026-10，基于 Obsidian Copilot V4（4.0.x）。旧版本（V2 / V3）的界面不同，见文末。

## 准备

- 在 Obsidian 的「设置 → 第三方插件」里关闭安全模式，点「浏览」，搜索 **Copilot** 安装并启用。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=obsidian)。

## 第一步：添加自定义服务商

1. 打开 Obsidian 设置，在左侧找到 **Copilot**，切到 **BYOK** 标签。
2. 点 **Add a provider**，选 **Add a custom provider**。

## 第二步：填写服务商信息

| 字段 | 填什么 |
|---|---|
| Display name | `HiveGPT` |
| API key | 你的 HiveGPT Key（`sk-` 开头），可以点 Test 检查 |
| Base URL | `https://hivegpt.cn/v1` |
| Enable CORS | **关闭**（默认） |
| Model ID | 输入 `gpt-5.5`，点 Add；也可以用 Search available models 从列表里选 |

点 **Save** 保存。

::: tip 不用打开 Enable CORS
HiveGPT 的接口允许 Obsidian 这类应用直接跨域请求（2026-10 起），Enable CORS 保持关闭，回答会一个字一个字地出现。只有在发消息报网络错误时才需要打开它，打开后回答会在生成完以后一次性显示。
:::

Test 显示 Verified 只说明 Key 有效，不代表每个模型都能用；模型以你的分组为准，见 [查看可用模型](/connect/#查看可用模型)。

## 第三步：在 Quick Chat 里使用

新添加的模型会自动出现在 **Basic → Agents** 的 Quick Chat 模型里。打开 Copilot 的聊天面板，在模型下拉里选 `HiveGPT` 下的 `gpt-5.5`，试试：

> 把当前笔记总结成 5 条要点，每条不超过 20 字。

常用的用法：总结长笔记、改写段落、翻译、根据大纲扩写、整理会议记录。

## 关于知识库问答（Vault QA）

Copilot V4 把笔记库的语义搜索移到了单独的 **Miyo** 应用里（设置 → Copilot → Miyo → Connect），插件里已经不能再配置自定义的嵌入模型。所以「问整个笔记库」这一块目前用不了 HiveGPT 的接口，聊天、总结、改写不受影响。

## 旧版本（V2 / V3）怎么填

如果你的 Copilot 还是旧版本，设置里是 **Add Custom Model**：

1. Model Name 填 `gpt-5.5`。
2. Provider 选 **3rd party (openai-format)**。
3. Base URL 填 `https://hivegpt.cn/v1`，API Key 填你的 Key。
4. CORS 选项不用勾选，点 Verify，再点 Add Model。

建议更新到 V4，旧版本不再推荐使用。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| Test 通过，但发消息没反应、报网络错误 | 网络或代理拦截了请求 | 检查代理设置；仍不行就打开 Enable CORS 试试 |
| 「Error generating response. Please try again」 | 地址、Key 或模型名不对 | 依次检查 Base URL、Key 和 Model ID，见 [报错速查](/connect/errors/) |
| 「No API keys found in this device's Obsidian Keychain」 | Key 只保存在当前设备 | 换电脑或手机后要重新填一次 Key |
| 提示分组「不支持模型」 | Model ID 不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 回答不是一个字一个字出现 | 打开了 Enable CORS | 关闭 Enable CORS |

想在其他客户端里用，看 [Cherry Studio](/connect/cherry-studio) 和 [Chatbox](/connect/chatbox) 的教程。
