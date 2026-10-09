---
title: LobeChat、NextChat 接入第三方 API：自建 AI 聊天站配置教程
description: 在 LobeChat（LobeHub）和 NextChat（ChatGPT-Next-Web）里接入 HiveGPT 的 OpenAI 兼容接口：界面里的 API 代理地址 / 接口地址怎么填（一个要 /v1、一个不要），Docker 部署的 OPENAI_PROXY_URL、BASE_URL、CUSTOM_MODELS 等环境变量，访问密码和常见问题。
---

# LobeChat、NextChat 接入第三方 API

**两个工具的地址填法正好相反：LobeChat 的「API 代理地址」填 `https://hivegpt.cn/v1`，NextChat 的 `BASE_URL` / 接口地址填 `https://hivegpt.cn`（不带 /v1）。** 两者都让请求经过它自己的服务器转发：LobeChat 不要打开「使用客户端请求模式」，NextChat 自建时用环境变量配置地址，界面里的接口地址保持默认。

> 更新于 2026-10，基于 LobeHub 2.x 和 NextChat 当前版本。

## 准备

一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=lobechat)。

::: tip 用服务端转发更安全
两个工具都有「浏览器直接请求接口」的模式，HiveGPT 的接口也允许跨域请求，两种都能用。自己部署给别人用时，建议让请求经过 LobeChat / NextChat 自己的服务器转发（这也是它们的默认方式），Key 只保存在服务器上，不会出现在访客的浏览器里。
:::

## 一、LobeChat（LobeHub）

### 在界面里配置

1. 打开 **应用设置 → AI 服务商**，选 **OpenAI**，打开这个服务商。
2. 填写：

| 字段 | 填什么 |
|---|---|
| API Key | 你的 HiveGPT Key |
| API 代理地址 | `https://hivegpt.cn/v1`（**要带 /v1**） |
| 使用客户端请求模式 | **关闭** |

3. 在 **连通性检查** 里点 **检查**。
4. 在 **模型列表** 里点 **获取模型列表**，打开要用的模型；没有的话点 **添加模型**，模型 ID 填 `gpt-5.5`，最大上下文窗口填 `1050000`，打开「支持视觉识别」和「支持技能使用」。
5. 回到对话，在模型选择里选 OpenAI 下的 `gpt-5.5`。

也可以用 **创建自定义 AI 服务商**（请求格式选 openai，代理地址填 `https://hivegpt.cn/v1`），和官方 OpenAI 分开管理。

### 自己部署时用环境变量

LobeHub 2.x 自建需要 PostgreSQL 数据库和几项密钥（`DATABASE_URL`、`KEY_VAULTS_SECRET`、`AUTH_SECRET` 等），完整步骤见 [LobeHub 官方部署文档](https://lobehub.com/docs/self-hosting/platform/docker)。和 HiveGPT 相关的是这三个变量：

```bash
OPENAI_API_KEY=sk-你的Key
OPENAI_PROXY_URL=https://hivegpt.cn/v1
OPENAI_MODEL_LIST=-all,+gpt-5.5=GPT-5.5
```

`OPENAI_MODEL_LIST` 里 `-all` 隐藏所有内置模型，`+` 添加模型，`=` 后面是显示名称。

## 二、NextChat（ChatGPT-Next-Web）

### 自己部署（推荐）

用环境变量配置，请求由 NextChat 的服务器转发：

```bash
docker run -d -p 3000:3000 \
  -e OPENAI_API_KEY=sk-你的Key \
  -e BASE_URL=https://hivegpt.cn \
  -e CUSTOM_MODELS=-all,+gpt-5.5 \
  -e DEFAULT_MODEL=gpt-5.5 \
  -e CODE=设置一个访问密码 \
  -e HIDE_USER_API_KEY=1 \
  yidadaa/chatgpt-next-web
```

| 变量 | 作用 |
|---|---|
| `BASE_URL` | `https://hivegpt.cn`，**不要带 /v1**，NextChat 会自己加 |
| `OPENAI_API_KEY` | 你的 HiveGPT Key |
| `CUSTOM_MODELS` | `-all` 隐藏内置模型，`+gpt-5.5` 添加模型，多个用英文逗号隔开 |
| `DEFAULT_MODEL` | 默认模型 |
| `CODE` | **访问密码**，多个用逗号隔开。不设的话任何人打开网址都能用你的 Key |
| `HIDE_USER_API_KEY` | 设为 1 时用户不能填自己的 Key |

部署到 Vercel 时，在项目的环境变量里设置同样的变量。

打开 `http://服务器地址:3000`，在设置里输入访问密码就能用。

### 在界面里配置

已经部署好的 NextChat，也可以在 **设置 → 自定义接口** 里填：

| 字段 | 填什么 |
|---|---|
| 模型服务商 | OpenAI |
| 接口地址 | 保持默认，用服务器上配置的 `BASE_URL` |
| API Key | 你的 HiveGPT Key |
| 自定义模型名 | `gpt-5.5`，多个用英文逗号隔开 |

接口地址改成 `https://hivegpt.cn` 时会变成浏览器直接请求，也能用，但 Key 会保存在当前浏览器里。部署给别人用时，保持默认、用服务器端的 `BASE_URL` 更安全。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| LobeChat 回复是空的 | API 代理地址少了 `/v1` | 改成 `https://hivegpt.cn/v1` |
| NextChat 报 404，提示「接口地址多了一个 /v1」 | `BASE_URL` 带了 `/v1` | 改成 `https://hivegpt.cn` |
| 网页里报 Failed to fetch | 浏览器连不上接口（网络、代理或地址写错） | 检查接口地址；或 LobeChat 关闭「使用客户端请求模式」、NextChat 接口地址保持默认，改由服务器转发 |
| NextChat 报 403「you are not allowed to use X model」 | 模型不在 `CUSTOM_MODELS` 里 | 在 `CUSTOM_MODELS` 里加上 |
| 401 | Key 不对或已停用 | 见 [401 报错](/connect/errors/401) |
| 提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 余额很快用完 | 网址被别人用了 | NextChat 设置 `CODE` 访问密码；给这个 Key 设额度上限 |

自己用、不想部署的话，桌面客户端更简单，见 [Cherry Studio](/connect/cherry-studio)、[Chatbox](/connect/chatbox)。
