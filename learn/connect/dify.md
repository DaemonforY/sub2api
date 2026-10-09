---
title: Dify 添加自定义模型供应商：OpenAI-API-compatible 接入 GPT 教程
description: 在 Dify 1.x（云端和私有部署）里安装 OpenAI-API-compatible 插件，填 API Base URL https://hivegpt.cn/v1、API Key 和模型名，配置上下文长度、函数调用、视觉支持，设为系统默认模型并在应用和工作流里使用；附 Docker 私有部署的 SSRF 代理、超时等常见问题。
---

# Dify 接入 OpenAI 兼容模型

**四步接入：** 在 Dify 的「集成（或设置）→ 模型供应商」里安装 **OpenAI-API-compatible** 插件；点「添加模型」，模型类型选 LLM，**模型名称**填 `gpt-5.5`，**API Base URL** 填 `https://hivegpt.cn/v1`，**API Key** 填你的 HiveGPT Key；按下表填好上下文长度和函数调用；保存后在「默认模型」里把它设为系统推理模型，应用和工作流就能用了。

> 更新于 2026-10，基于 Dify 1.x 和 OpenAI-API-compatible 插件。不同版本里菜单可能叫「集成 → 模型供应商」或「设置 → 模型供应商」。

## 准备

- 一个 Dify 工作空间：[Dify 云端](https://cloud.dify.ai) 或私有部署的 Dify 1.x，用管理员账号登录。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=dify)。

## 第一步：安装 OpenAI-API-compatible 插件

1. 点右上角头像，进入 **集成**（旧版本是 **设置**），选 **模型供应商**。
2. 在插件市场里搜索 **OpenAI-API-compatible**，安装官方的那个（作者 langgenius）。

::: tip 也可以用 OpenAI 插件
官方的 **OpenAI** 插件有一个可选的 **API Base URL**，填 `https://hivegpt.cn` 也能用（它会自动补 `/v1`）。但它内置的模型列表里不一定有你要的模型，接第三方接口用 OpenAI-API-compatible 更省事。
:::

## 第二步：添加模型

在 OpenAI-API-compatible 卡片上点 **添加模型**，按下表填：

| 字段 | 填什么 |
|---|---|
| 模型类型 | LLM |
| 模型名称 | `gpt-5.5` |
| 模型显示名称 | 可选，如 `HiveGPT gpt-5.5` |
| API Key | 你的 HiveGPT Key（`sk-` 开头） |
| API Base URL | `https://hivegpt.cn/v1`（**要带 /v1**） |
| 对话类型 | 对话（Chat） |
| 模型上下文长度 | `128000` |
| 最大 token 上限 | `16384` |
| 函数调用类型 | Tool Call |
| 流式函数调用 | 支持 |
| 视觉支持 | 支持（gpt-5.5 能看图） |
| 结构化输出 | 支持 |
| API 类型 | Chat Completions API（默认） |
| token 参数名 | 自动，或选「使用 max_completion_tokens」 |

**这个插件不会自动补 /v1**，API Base URL 必须写成 `https://hivegpt.cn/v1`。保存时 Dify 会真的发一条测试消息验证，通过了才会保存。

上下文长度和最大 token 上限是 Dify 用来截断对话和限制输出的，填大一些更不容易截断，但每次请求可能更贵。其他模型的上限看 [模型广场](https://hivegpt.cn/model-plaza)。

## 第三步：设为默认模型

1. 在模型供应商页右上角点 **默认模型**（有的版本叫「系统模型设置」）。
2. **系统推理模型** 选刚添加的 `gpt-5.5`，保存。

没有单独选模型的应用和节点会用这个默认模型。

## 第四步：在应用和工作流里使用

- **聊天助手 / Agent**：在应用的编排页右上角的模型选择里，选 OpenAI-API-compatible 下的 `gpt-5.5`。
- **工作流**：在 LLM 节点里选择模型；用到工具调用的 Agent 节点，需要第二步里把函数调用类型设为 Tool Call。

## 嵌入模型（知识库）

做知识库需要嵌入模型。先用 [`/v1/models`](/connect/#查看可用模型) 确认你的分组里有嵌入模型（名字里带 `embedding`），有的话在同一个插件里再添加一个，模型类型选 **Text Embedding**，填模型名称、API Base URL、上下文长度，「每批最大分块数」可以先填 `1`。没有的话，知识库的嵌入部分用其他服务商。

## 私有部署（Docker）的常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 保存时报 Credentials validation failed，状态码 404 | API Base URL 少了 `/v1` | 改成 `https://hivegpt.cn/v1` |
| 状态码 401 / 403 | Key 不对，或请求被 Dify 的 SSRF 代理拦截 | 先用 [curl 命令](/connect/#验证-key-能用) 确认 Key 能用；再看 `ssrf_proxy` 容器日志里有没有 `TCP_DENIED`，有的话在 `docker/ssrf_proxy/squid.conf.template` 里放行 hivegpt.cn 并重启 `ssrf_proxy` |
| 安装插件一直转圈 | 服务器下载 Python 依赖太慢 | 在 `.env` 里设置 `PIP_MIRROR_URL` 为国内镜像，并适当调大 `PLUGIN_PYTHON_ENV_INIT_TIMEOUT` |
| 长回答或工作流中途超时 | 插件执行超时 | 调大 `.env` 里的 `PLUGIN_MAX_EXECUTION_TIMEOUT` 和 `PLUGIN_DAEMON_TIMEOUT`（两者保持一致），重启 |
| 提示分组「不支持模型」 | 模型名称不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 429 | 并发太高，比如批量跑工作流 | 见 [429 报错](/connect/errors/429) |

更多报错看 [报错速查](/connect/errors/)。

## 常见问题

**Dify 云端版也要这样配吗？** 是的，步骤一样；云端没有上面那些 Docker 配置问题。

**费用怎么算？** 每次调用按 HiveGPT 的标价从你的余额或订阅里扣，Dify 里显示的价格只是估算。工作流和 Agent 一次可能调用好几次模型，用量以 [使用记录](https://hivegpt.cn/usage) 为准，见 [token 怎么算钱](/connect/tokens)。
