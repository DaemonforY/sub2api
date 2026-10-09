---
title: 在常用工具里配置 HiveGPT
description: 在 OpenCode、CodeBuddy 等编程工具，以及 Cherry Studio、Chatbox 等对话客户端里使用 HiveGPT：OpenAI 兼容接口怎么填 Base URL、API Key 和模型名。
---

# 在常用工具里配置

这篇讲 Codex 以外的工具。Codex 请看 [Codex 接入 HiveGPT](/codex/hivegpt)。

所有工具的思路都一样：在设置里找到 **「自定义 / OpenAI 兼容」** 一类的模型服务，填三项：

| 项目 | 填写 |
|---|---|
| API 地址（Base URL） | `https://hivegpt.cn/v1`，个别工具填 `https://hivegpt.cn` |
| API Key | 你在 [API 密钥](https://hivegpt.cn/keys) 页创建的 Key |
| 模型 | 你的分组里可用的模型，例如 `gpt-5.5`，用 [`/v1/models`](/connect/#查看可用模型) 查 |

::: tip 菜单名称以工具当前版本为准
各工具更新很快，下面写的是要找的设置项，具体菜单的位置和叫法可能和你看到的略有不同。
:::

## OpenCode

OpenCode 是开源的终端编程助手。HiveGPT 的「使用密钥」弹窗能直接生成它的配置：

1. 在 [API 密钥](https://hivegpt.cn/keys?action=use) 页找到 GPT 类分组的 Key，点「使用密钥」，选 **OpenCode**。
2. 复制生成的 `opencode.json`，保存到：
   - macOS / Linux：`~/.config/opencode/opencode.json`
   - Windows：`%USERPROFILE%\.config\opencode\opencode.json`

   目录不存在就先创建。
3. 在项目目录里运行 `opencode`，用 `/models` 选择一个 HiveGPT 的模型，随便问一句试试。

生成的配置大致是这样（Key 换成你自己的）：

```json
{
  "provider": {
    "openai": {
      "options": {
        "baseURL": "https://hivegpt.cn/v1",
        "apiKey": "sk-你的Key"
      }
    }
  }
}
```

弹窗生成的文件里还带着各个模型的上下文长度和推理强度，直接用弹窗里的版本即可。手写配置、权限设置和常见报错见 [OpenCode 配置教程](/connect/opencode)。

## CodeBuddy 等 IDE 编程助手

如果你用的 IDE 助手支持「自定义模型」或「OpenAI 兼容接口」：

1. 打开模型设置，新增一个自定义模型（提供商类型选 OpenAI / OpenAI 兼容）。
2. API 地址填 `https://hivegpt.cn/v1`，API Key 填你的 Key，模型名填 `gpt-5.5` 等分组里的模型。
3. 保存后在对话里切换到这个模型，发一句话测试。

只支持官方账号登录、没有自定义模型入口的工具，用不了第三方接口。

## Cherry Studio

API 地址填 `https://hivegpt.cn`（不加 `/v1`，Cherry Studio 会自动补），类型选 OpenAI。完整步骤见 [Cherry Studio 配置教程](/connect/cherry-studio)。

## 沉浸式翻译

自定义 API 接口地址要填完整的 `https://hivegpt.cn/v1/chat/completions`。完整步骤见 [沉浸式翻译接入 GPT API](/connect/immersive-translate)。

## Chatbox

API 模式选 **OpenAI API 兼容**，API 主机填 `https://hivegpt.cn/v1`，API 路径保持 `/chat/completions`。完整步骤见 [Chatbox 接入第三方 API](/connect/chatbox)。

## 还是不通？

1. 地址到底加不加 `/v1`，对照 [Base URL 填法对照表](/connect/base-url)。
2. 先用 [接入教程首页](/connect/#验证-key-能用) 里的 curl 命令测一下 Key 本身是否可用。curl 能通，问题就在工具的配置上。
3. 对照 [报错速查](/connect/errors/) 排查。最常见的是：地址多了或少了 `/v1`，以及模型名不在你 Key 的分组里。
4. 到 [使用记录](https://hivegpt.cn/usage) 看请求有没有到达 HiveGPT：有记录说明网络和 Key 都没问题；没有记录说明请求没发过来。
