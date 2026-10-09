---
title: Cherry Studio 配置自定义 API（OpenAI 兼容）图文教程
description: 在 Cherry Studio 里添加自定义服务商、填写 API 地址和 API 密钥、获取模型列表，接入 HiveGPT 使用 GPT 等模型；附 API 地址自动补全规则、检测失败和模型不可用的排查方法。
---

# Cherry Studio 配置自定义 API

**三步接入：** 在 Cherry Studio 的「设置 → 模型服务」里添加一个服务商，类型选 **OpenAI**；**API 地址**填 `https://hivegpt.cn`，**API 密钥**填你的 HiveGPT Key；点「获取模型列表」添加 `gpt-5.5` 等模型，回到对话里选择它就能用。

> 更新于 2026-10，基于 Cherry Studio 当前版本。界面改版时菜单位置可能略有不同。

## 准备

- 安装 [Cherry Studio](https://cherry-ai.com)（Windows / macOS / Linux 都有）。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=cherry-studio)。

## 第一步：添加服务商

1. 打开 Cherry Studio，点左下角的 **设置**（齿轮图标），进入 **模型服务**。
2. 在服务商列表底部点 **添加**，名称随便写，例如 `HiveGPT`；提供商类型选 **OpenAI**。
3. 打开这个服务商右上角的开关，启用它。

## 第二步：填 API 地址和密钥

| 字段 | 填什么 |
|---|---|
| API 密钥 | 你的 HiveGPT Key（`sk-` 开头） |
| API 地址 | `https://hivegpt.cn` |

**API 地址不要加 `/v1`。** Cherry Studio 会自动在后面补上 `/v1/chat/completions`，你填 `https://hivegpt.cn/v1` 就会变成 `/v1/v1/...`，请求失败。输入框下方会显示最终请求的地址，确认它是 `https://hivegpt.cn/v1/chat/completions` 即可。

::: tip 想完全按自己填的地址请求
在地址末尾加 `#`，Cherry Studio 就不再自动补路径，例如 `https://hivegpt.cn/v1/chat/completions#`。一般用不到，自动补全就够了。
:::

## 第三步：添加模型

1. 点 **获取模型列表**，Cherry Studio 会读取你的 Key 能用的模型。
2. 在弹出的列表里，点模型右侧的 **+** 把要用的模型加进来，例如 `gpt-5.5`。
3. 点 **检测**，选一个刚添加的模型。显示连接成功就配置好了。

获取不到列表时，也可以手动添加模型，模型 ID 填 `gpt-5.5` 这类名字。你的分组能用哪些模型，以 [`/v1/models`](/connect/#查看可用模型) 的返回为准。

## 第四步：开始对话

回到对话界面，点顶部的模型名称，切换到 `HiveGPT` 下的模型，发一句话测试。每次对话的用量和费用可以在 HiveGPT 的 [使用记录](https://hivegpt.cn/usage) 里查到。

## 检测失败怎么办

| 现象 | 原因 | 处理 |
|---|---|---|
| 404 | API 地址多写了 `/v1` | 改成 `https://hivegpt.cn` |
| 401、提示 API Key 无效 | Key 复制不全，或已删除 / 停用 | 到 [API 密钥](https://hivegpt.cn/keys) 页重新复制完整的 Key |
| 提示模型不存在、分组不支持 | 这个模型不在你 Key 的分组里 | 用「获取模型列表」重新选，或换一个分组的 Key |
| 提示余额不足 | 余额为 0，或订阅额度用完 | [充值或订阅](https://hivegpt.cn/purchase) 后重试，同一个 Key 继续用 |
| 一直转圈 | 网络问题，或上游繁忙 | 先用 [curl 命令](/connect/#验证-key-能用) 测 Key，能通就是 Cherry Studio 的网络设置问题（如代理） |

更多地址相关的报错，看 [Base URL 末尾要不要加 /v1](/connect/base-url)。

## 常见问题

**能同时用多个服务商吗？** 可以。HiveGPT 只是模型服务里的一项，和其他服务商互不影响，对话时切换模型即可。

**知识库要用的嵌入模型也能用 HiveGPT 吗？** 如果你的分组里有嵌入模型（`/v1/models` 里能看到 `embedding` 字样的模型），在第三步把它加进来，类型选嵌入即可；没有就用其他服务商的嵌入模型。

**画图能用吗？** Cherry Studio 的对话主要走 Chat Completions。想用 gpt-image-2 画图，用 HiveGPT 的无限画布更方便。
