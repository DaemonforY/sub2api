---
title: Chatbox 接入第三方 API：API 主机和 API 路径怎么填
description: 在 Chatbox 里添加自定义提供方（OpenAI API 兼容），接入 HiveGPT 使用 GPT 等模型：API 主机填 https://hivegpt.cn/v1、API 路径填 /chat/completions，获取模型列表，以及检查失败、404、模型不可用的排查方法。
---

# Chatbox 接入第三方 API

**三步接入：** 在 Chatbox 的「设置 → 模型提供方」底部点「添加」，API 模式选 **OpenAI API 兼容**；**API 主机**填 `https://hivegpt.cn/v1`，**API 路径**保持 `/chat/completions`，**API 密钥**填你的 HiveGPT Key；点「获取」加载模型列表，选 `gpt-5.5` 等模型后保存，新建对话就能用。

> 更新于 2026-10，基于 Chatbox 当前版本。不同版本里「API 主机」也可能叫「API 域名」或「Base URL」，意思一样。

## 准备

- 安装 [Chatbox](https://chatboxai.app)（Windows / macOS / Linux / 手机端都有），也可以用它的网页版。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=chatbox)。

## 第一步：添加自定义提供方

1. 打开 Chatbox，点左下角的 **设置**，进入 **模型提供方**。
2. 在列表底部点 **添加**，名称写 `HiveGPT`，API 模式选 **OpenAI API 兼容**，确认添加。

## 第二步：填写接口信息

| 字段 | 填什么 |
|---|---|
| API 密钥 | 你的 HiveGPT Key（`sk-` 开头） |
| API 主机 | `https://hivegpt.cn/v1` |
| API 路径 | `/chat/completions`（默认值，不用改） |

**主机和路径要配对。** Chatbox 实际请求的是「主机 + 路径」，拼起来必须是 `https://hivegpt.cn/v1/chat/completions`。下面两种写法都对，任选一种：

| API 主机 | API 路径 |
|---|---|
| `https://hivegpt.cn/v1` | `/chat/completions` |
| `https://hivegpt.cn` | `/v1/chat/completions` |

不要两边都写 `/v1`，也不要两边都不写。

## 第三步：添加模型并测试

1. 点 **检查**，确认 Key 和地址能用。
2. 点 **获取**，从 HiveGPT 加载你的分组可用的模型，勾选要用的模型，例如 `gpt-5.5`。获取不到时点 **新建**，手动填模型 ID。
3. 保存后回到对话界面，新建对话，在输入框上方的模型选择里切换到 `HiveGPT` 下的模型，发一句话测试。

你的分组能用哪些模型，以 [`/v1/models`](/connect/#查看可用模型) 的返回为准。每次对话的用量和费用可以在 [使用记录](https://hivegpt.cn/usage) 里查到。

## 检查失败怎么办

| 现象 | 原因 | 处理 |
|---|---|---|
| 404，提示「接口地址多了一个 /v1」 | 主机和路径都写了 `/v1` | 按上面的配对表改一边 |
| 404，提示「接口地址少了 /v1」，或返回一段 HTML | 主机和路径都没写 `/v1` | 主机改成 `https://hivegpt.cn/v1` |
| 401，「API Key 无效或已被删除」 | Key 复制不全，或已删除 | 见 [401 报错](/connect/errors/401) |
| 「不支持模型」 | 模型不在你 Key 的分组里 | 见 [模型不存在 / 分组不支持模型](/connect/errors/model) |
| 「账户余额不足」 | 余额为 0，或订阅额度用完 | 见 [余额不足 / 额度已用完](/connect/errors/quota) |
| 网络错误、一直转圈 | 网络不稳定 | 在提供方的高级设置里打开「改善网络兼容性」再试 |

更多报错看 [报错速查](/connect/errors/)。

## 常见问题

**手机上的 Chatbox 也能这样配吗？** 能，设置项和桌面版一样。

**能用 HiveGPT 画图吗？** Chatbox 的对话走 Chat Completions。想用 gpt-image-2 画图，用 HiveGPT 的无限画布更方便。

**和 Cherry Studio 有什么区别？** 都是多模型对话客户端，配置思路一样，Cherry Studio 的 API 地址不加 `/v1`，见 [Cherry Studio 配置教程](/connect/cherry-studio)。
