---
title: Cline 配置自定义 API（OpenAI Compatible）：VS Code 里接入 GPT 模型
description: 在 VS Code 的 Cline 插件里选 OpenAI Compatible，填 Base URL https://hivegpt.cn/v1、API Key 和 Model ID，配好上下文窗口、最大输出和价格，接入 HiveGPT 的 GPT 模型写代码；附 Roo Code 的填法（已停止维护）和常见报错。
---

# Cline 配置自定义 API

**三步接入：** 点 Cline 面板上的 ⚙️ 设置，API Provider 选 **OpenAI Compatible**；**Base URL** 填 `https://hivegpt.cn/v1`，**API Key** 填你的 HiveGPT Key，**Model ID** 填 `gpt-5.5` 等你分组里的模型；展开 Model Configuration 填好上下文窗口和最大输出，保存后在 Cline 里发一个任务试试。

> 更新于 2026-10，基于 Cline 当前版本（界面是英文）。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备

- VS Code（或 Cursor、Windsurf 等基于 VS Code 的编辑器），在扩展市场搜索 **Cline** 安装。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=cline)。

## 第一步：选 OpenAI Compatible

1. 点左侧活动栏的 Cline 图标，打开 Cline 面板。
2. 点面板右上角的 ⚙️ 进入设置。
3. **API Provider** 选 **OpenAI Compatible**。

## 第二步：填接口信息

| 字段 | 填什么 |
|---|---|
| Base URL | `https://hivegpt.cn/v1` |
| API Key | 你的 HiveGPT Key（`sk-` 开头） |
| Model ID | `gpt-5.5`（下拉里没有就选 Use custom model ID 手动输入） |

Custom Headers、Azure 相关的选项（Set Azure API version、Use Azure Identity Authentication）都不用填。

## 第三步：填 Model Configuration

展开 **Model Configuration**，按你选的模型填（下面是 `gpt-5.5` 的数值）：

| 字段 | 填什么 | 说明 |
|---|---|---|
| Supports Images | 勾选 | gpt-5.5 能看图，可以把截图发给 Cline |
| Context Window Size | `1050000` | 模型的上下文上限；想控制花费可以填小一些，如 `200000` |
| Max Output Tokens | `32000` | 单次最多输出多少，上限 128000 |
| Input Price / 1M tokens | `5` | 用来在 Cline 里显示花费估算 |
| Output Price / 1M tokens | `30` | 同上 |
| Temperature | 不填 | 推理类模型只接受默认值，填了可能报错 |

价格只影响 Cline 显示的花费估算，实际扣费以 HiveGPT 的 [使用记录](https://hivegpt.cn/usage) 为准。其他模型的上下文、价格看 [模型广场](https://hivegpt.cn/model-plaza)。

## 第四步：试一下

保存设置，回到 Cline 面板，输入一个小任务，比如：

> 在当前项目根目录新建 hello.py，打印「你好，Cline」，然后运行它。

Cline 会列出它要做的每一步，修改文件、运行命令前会请你确认。

## 省钱提示

Cline 每一步都会把项目文件和之前的上下文一起发给模型，一个任务用掉几十万 token 很常见。

- 一个任务完成后点 **New Task** 开新任务，不要在一个任务里一直加需求。
- 简单的改动换便宜的模型，在 Model ID 里切换即可。
- 给 Cline 单独建一个 Key 并设额度上限，见 [token 怎么算钱](/connect/tokens)。

## Roo Code 怎么填

Roo Code 的 GitHub 仓库已在 2026 年 5 月归档，插件还能用但不再更新，新项目建议用 Cline。已经在用的话：

1. 打开 Roo Code 的 **设置 → 提供商**（英文界面是 Settings → Providers）。
2. **API 提供商** 选 **OpenAI Compatible**。
3. **OpenAI 基础 URL** 填 `https://hivegpt.cn/v1`，**API 密钥** 填你的 Key，**模型** 填 `gpt-5.5`。
4. 保持 **启用流式传输** 勾选；**使用 Azure 服务**、**启用 R1 模型参数** 不要勾。
5. 在模型信息里填 **上下文窗口大小** `1050000`、**最大输出 Token 数** `32000`，勾选 **图像支持**。

Roo Code 只支持模型原生的工具调用（function calling），GPT 系列模型都支持。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 404，提示「接口地址少了 /v1」或「多了一个 /v1」 | Base URL 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 401，API Key 无效 | Key 复制不全或已停用 | 见 [401 报错](/connect/errors/401) |
| 提示分组「不支持模型」 | Model ID 不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 400：`Unsupported parameter: 'temperature'` | 填了 Temperature | 清空 Temperature |
| 429 或一直重试 | 请求太快或额度用完 | 见 [429 报错](/connect/errors/429)、[余额 / 额度](/connect/errors/quota) |
| Cline 不按要求调用工具、反复出错 | 模型能力不够 | Cline 的提示词很复杂，换更强的模型 |

Cline 和 Codex 都能做 AI 编程。Codex 在 HiveGPT 上有一键生成的配置，见 [Codex 接入 HiveGPT](/codex/hivegpt)。
