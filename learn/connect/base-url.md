---
title: Base URL 末尾要不要加 /v1？各工具填法对照表
description: OpenAI 兼容接口的 Base URL（API 地址）到底填 https://hivegpt.cn 还是 https://hivegpt.cn/v1，还是要写到 /v1/chat/completions？按工具列出正确填法，以及填错时会看到的 404、返回 HTML、Unexpected token 等报错怎么判断。
---

# Base URL 末尾要不要加 /v1？

**答案：看工具会不会自己往后补路径。** 大多数工具和 SDK 填 `https://hivegpt.cn/v1`；会自动补 `/v1` 的工具（如 Cherry Studio）填 `https://hivegpt.cn`；要求填完整接口地址的工具（如沉浸式翻译）填 `https://hivegpt.cn/v1/chat/completions`。

判断方法很简单：工具最终请求的地址必须是 `https://hivegpt.cn/v1/chat/completions`（对话）或 `https://hivegpt.cn/v1/responses`（Codex）。你填的部分加上工具自己补的部分，拼起来等于它就对了。

> 更新于 2026-10。各工具的界面会改版，以工具自己的说明为准。

## 各工具填法对照表

| 工具 | 填写的字段 | 填什么 |
|---|---|---|
| OpenAI Python SDK | `base_url` | `https://hivegpt.cn/v1` |
| OpenAI Node.js SDK | `baseURL` | `https://hivegpt.cn/v1` |
| LangChain（`ChatOpenAI`） | `base_url` | `https://hivegpt.cn/v1` |
| curl / 自己发 HTTP 请求 | 请求地址 | `https://hivegpt.cn/v1/chat/completions`（完整地址） |
| Codex | `config.toml` 的 `base_url` | `https://hivegpt.cn/v1`，见 [Codex 接入 HiveGPT](/codex/hivegpt) |
| OpenCode | `opencode.json` 的 `baseURL` | `https://hivegpt.cn/v1` |
| CodeBuddy 等 IDE 助手 | 自定义模型的 API 地址 | `https://hivegpt.cn/v1` |
| Cherry Studio | API 地址 | `https://hivegpt.cn`（它会自动补 `/v1/chat/completions`），见 [Cherry Studio 配置教程](/connect/cherry-studio) |
| Chatbox | API 地址 / API 路径 | `https://hivegpt.cn`；单独的 API 路径填 `/v1/chat/completions` |
| 沉浸式翻译 | 自定义 API 接口地址 | `https://hivegpt.cn/v1/chat/completions`（完整地址），见 [沉浸式翻译配置教程](/connect/immersive-translate) |

不在表里的工具，先看它的输入框下方有没有提示（例如「会自动补全 /v1」「请填写完整地址」）。没有提示就先填 `https://hivegpt.cn/v1`，不行再按下面的报错调整。

## 填错了会看到什么

不同的填错方式，报错不一样。对照现象就能知道该加还是该减：

| 你看到的 | 实际请求到了 | 怎么改 |
|---|---|---|
| `404 page not found` | `https://hivegpt.cn/v1/v1/chat/completions`（`/v1` 重复了） | 去掉你填的 `/v1`，改成 `https://hivegpt.cn` |
| `Unexpected token '<'`、「返回的不是合法 JSON」、提示收到了一段 HTML | `https://hivegpt.cn/chat/completions`（少了 `/v1`），服务器返回的是网站页面 | 加上 `/v1`，改成 `https://hivegpt.cn/v1` |
| `404` 且地址里有两段 `chat/completions` | 你填了完整地址，工具又补了一遍 | 只填到 `https://hivegpt.cn/v1` |
| `401`，提示「缺少 API Key」或「API Key 无效」 | 地址是对的，Key 有问题 | 地址不用改，重新复制完整的 Key，见 [常见报错](/connect/#常见报错) |
| 「模型不存在」「分组不支持该模型」 | 地址和 Key 都对，模型名不在你 Key 的分组里 | 用 [`/v1/models`](/connect/#查看可用模型) 查可用的模型名 |

看到 401 或「模型不存在」反而是好消息：说明请求已经到达 HiveGPT，地址填对了。

## 还有几个容易填错的地方

- **不要用 `http://`**，要用 `https://`。
- **末尾不要多一个斜杠**，例如 `https://hivegpt.cn/v1/`。大多数工具能容忍，但有的会拼出 `//chat/completions`。
- **不要带空格或换行**。从网页复制时容易带上，粘贴后检查一下首尾。
- **Base URL 和 Key 是两个字段**，不要把 Key 拼进地址里。

## 用 curl 确认地址

拿不准时，在终端运行这条命令（把 Key 换成你自己的）：

```bash
curl https://hivegpt.cn/v1/chat/completions \
  -H "Authorization: Bearer sk-你的Key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-5.5", "messages": [{"role": "user", "content": "你好"}]}'
```

能返回回答，说明 Key 和接口都没问题，再回到工具里对照上面的表格调整地址。还没有 Key 的话，先到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个。
