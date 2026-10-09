---
title: Bob、Pot 划词翻译接入 GPT API：自定义 OpenAI 接口地址怎么填
description: 在 macOS 的 Bob 和跨平台的 Pot 划词翻译里接入 HiveGPT 的 GPT 模型：Bob 的「自定义API Base URL」和 Pot 的「请求地址」分别怎么填、要不要带 /v1、选哪个模型最划算、自定义翻译提示词，以及 404、401、测试失败等常见问题。
---

# Bob、Pot 划词翻译接入 GPT API

**最短答案：**

- **Bob**（macOS）：偏好设置 → 服务 → 文本翻译 → 点 **+** 添加 **OpenAI**，「自定义API Base URL」填 `https://hivegpt.cn`（**不带 /v1**），API Key 填你的 HiveGPT Key，模型选「自定义模型」填 `gpt-6-luna`。
- **Pot**（Windows / macOS / Linux）：偏好设置 → 服务设置 → 翻译 → 添加内置服务 → **OpenAI**，「请求地址」填 `https://hivegpt.cn/v1/chat/completions`，填 Key，模型改成 `gpt-6-luna`。

> 更新于 2026-10，基于 Bob 1.21、Pot 3.0.7。

## 准备

- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=bob-pot)。
- **翻译用 `gpt-6-luna` 就够了**：快、便宜，每 100 万 token 输入 $0.1、输出 $0.5，划词翻译一万次（每次一两句话）大约 $0.5。对译文质量要求高（论文、合同）时换 `gpt-6.1-sol`。

## Bob（macOS）

Bob 在 Mac App Store 下载，OpenAI 翻译是内置服务，不需要装插件。

1. 打开 Bob 的 **偏好设置** → **服务**，左侧选 **文本翻译**。
2. 点列表底部的 **+**，选 **OpenAI**（不要选 Azure OpenAI）。
3. 按下表填写：

| 字段 | 填什么 |
|---|---|
| API Key | 你的 HiveGPT Key（`sk-` 开头） |
| 模型 | 选「自定义模型」，在「自定义模型」框里填 `gpt-6-luna` |
| 自定义API Base URL | `https://hivegpt.cn`（只填域名，不带 `/v1`） |
| 自定义API Path | 保持默认（`/v1/chat/completions`） |
| 温度 | 0.3 左右，翻译要稳定就调低 |

4. 点 **验证**，通过后打开服务开关，点右下角 **保存**。
5. 选中一段文字按 Bob 的划词翻译快捷键，看结果里有没有 OpenAI 这一栏。

Bob 的 OpenAI 服务还有「润色」等模式，同样的填法可以再添加一个服务，模式选润色。

::: tip Base URL 和 Path 是拼起来的
Bob 把「Base URL」和「Path」拼成最终地址。Base URL 填了 `https://hivegpt.cn/v1`，再加上默认的 Path，就变成 `/v1/v1/chat/completions`，会报 404。只填域名即可。
:::

截图翻译：Bob 先用系统自带的 OCR 识别文字，再交给翻译服务，所以截图翻译也会用上你配置的 GPT 模型。

## Pot（Windows / macOS / Linux）

Pot 是开源免费的划词翻译。安装：

```bash
winget install Pylogmon.pot                                       # Windows
brew tap pot-app/homebrew-tap && brew install --cask pot          # macOS
```

Linux 有 deb 包和 AUR（`pot-translation`）。

::: warning Pot 已停止更新
Pot 的代码仓库在 2026-09 已归档，3.0.7 是最后一个版本。现在仍能正常使用，但以后不会再修 bug。只用 Mac 的话 Bob 更省心。
:::

1. 打开 Pot 的 **偏好设置** → **服务设置** → **翻译**。
2. 点 **添加内置服务**，选 **OpenAI**。
3. 按下表填写：

| 字段 | 填什么 |
|---|---|
| 服务提供商 | OpenAI |
| 请求地址 | `https://hivegpt.cn/v1/chat/completions` |
| Api Key | 你的 HiveGPT Key |
| 模型 | 把默认的 `gpt-3.5-turbo` 改成 `gpt-6-luna` |
| 流式输出 | 打开，译文会逐字显示 |

4. 保存时 Pot 会自动用 hello 测试一次翻译，测试通过就保存成功。
5. 到 **热键设置** 给「划词翻译」「截图翻译」设快捷键。Pot 的快捷键默认是空的，不设的话按什么都没反应。

请求地址也可以只填 `https://hivegpt.cn`，Pot 会自动补上 `/v1/chat/completions`。**不要填 `https://hivegpt.cn/v1`**，会被补成 `/v1/v1/chat/completions`。

### 改翻译提示词

Pot 的 OpenAI 服务里有 **Prompt List**，`$text`、`$from`、`$to` 会被替换成原文、源语言和目标语言。比如想让译文更口语：

```text
把下面的内容翻译成$to，用自然的口语表达，专有名词保留英文：
$text
```

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 404 | 地址里有两个 `/v1` | Bob 的 Base URL 只填 `https://hivegpt.cn`；Pot 填完整的 `https://hivegpt.cn/v1/chat/completions` |
| 401 / Key 无效 | Key 复制不全或已停用 | 重新复制，见 [401 报错](/connect/errors/401) |
| 提示分组「不支持模型」 | 模型名写错或不在你的分组里 | 用 `gpt-6-luna`，见 [模型不存在](/connect/errors/model) |
| Pot 提示「测试失败」 | 地址、Key、模型有一项不对 | 看错误信息里的 Http Status：401 查 Key，404 查地址 |
| Pot 按快捷键没反应 | 快捷键默认为空，或和别的软件冲突 | 到热键设置里设置；提示「快捷键已被注册」就换一个组合 |
| Bob 在 macOS 15 上 Option 快捷键失效 | 系统限制 | 按 Bob 的提示安装 BobHelper |
| 开了代理后流式输出不工作（Pot） | Pot 软件内的代理不支持流式 | 用系统代理，或关掉流式输出 |
| 翻译太慢 | 用了推理强度高的大模型 | 换 `gpt-6-luna` |

**翻译一本书大概多少钱？** 按 gpt-6-luna 算，10 万字的中文书，原文加译文约 30 万 token，不到 $0.2。具体见 [token 怎么算钱](/connect/tokens)。

**网页和 PDF 整篇翻译用什么？** 用 [沉浸式翻译](/connect/immersive-translate)，同样能接入 HiveGPT。读论文可以用 [Zotero](/connect/zotero)。

更多：[Base URL 要不要加 /v1](/connect/base-url) · [GPT 模型怎么选](/connect/models) · [报错速查](/connect/errors/)
