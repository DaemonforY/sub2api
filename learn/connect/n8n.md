---
title: n8n 接入 OpenAI 兼容接口：Base URL 配置和 AI Agent 工作流教程
description: 在 n8n 里创建 OpenAI 凭证，把 Base URL 改成 https://hivegpt.cn/v1，用 OpenAI Chat Model + AI Agent / Basic LLM Chain 搭一个能用的自动化工作流；附模型选 By ID、Use Responses API、Timeout 等设置和常见报错。
---

# n8n 接入 OpenAI 兼容接口

**在 n8n 里新建一个 OpenAI 凭证，API Key 填你的 HiveGPT Key，Base URL 改成 `https://hivegpt.cn/v1`。** 之后在 AI Agent、Basic LLM Chain 节点上挂一个 **OpenAI Chat Model**，选这个凭证，Model 选 **ID** 并填 `gpt-5.5`，工作流就能调用 GPT 了。

> 更新于 2026-10，基于 n8n 2.x（界面是英文）。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备

- n8n：[n8n Cloud](https://n8n.io) 或自己部署的 n8n。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=n8n)。

## 第一步：创建 OpenAI 凭证

1. 在 n8n 左侧点 **Credentials**（或在任意 OpenAI 节点的 Credential 下拉里选 Create new credential），类型选 **OpenAI**。
2. 按下表填写：

| 字段 | 填什么 |
|---|---|
| API Key | 你的 HiveGPT Key（`sk-` 开头） |
| Organization ID (optional) | 留空 |
| Base URL | `https://hivegpt.cn/v1`（**要带 /v1**） |

3. 点 **Save**。n8n 会用这个 Key 请求一次模型列表，显示连接成功就可以了。

::: tip 把地址填在凭证里
OpenAI Chat Model 节点自己也有一个 Base URL 选项，但填在凭证里更省事：所有用这个凭证的节点都会走 HiveGPT。
:::

## 第二步：搭一个最小工作流

1. 添加触发节点 **Manual Trigger**（手动运行）或 **Schedule Trigger**（定时运行）。
2. 后面接一个 **Basic LLM Chain**（简单问答）或 **AI Agent**（要调用工具时用）。
   - **Source for Prompt (User Message)** 选 **Define below**。
   - **Prompt (User Message)** 写你的指令，可以用表达式引用上一个节点的数据，例如：

     ```text
     用 3 句话总结今天要关注的科技新闻要点：{{ $json.text }}
     ```
3. 点节点下方 Chat Model 处的 **+**，选 **OpenAI Chat Model**：
   - **Credential** 选刚建的 HiveGPT 凭证。
   - **Model** 切到 **ID**，填 `gpt-5.5`。
4. 点 **Execute workflow** 运行，结果在节点输出的 `output` 字段里。

常见的用法：定时抓取 RSS → 让模型总结 → 发到飞书 / 企业微信；表单提交 → 让模型分类和提取信息 → 写进表格。

## OpenAI Chat Model 的几个设置

在节点的 **Options** 里：

| 选项 | 建议 |
|---|---|
| Use Responses API | 默认打开即可（HiveGPT 支持）；如果报错，关掉改用 Chat Completions |
| Timeout | 默认 60000 毫秒，长回答或推理模型改成 `300000` |
| Max Retries | 默认 2，可以改成 3 |
| Maximum Number of Tokens | 限制输出长度，控制花费 |
| Sampling Temperature | 推理类模型不接受改过的温度，报错就删掉这一项 |
| Reasoning Effort | 简单任务选低，更快也更便宜 |

Model 的 **From List** 会从 HiveGPT 读取模型列表；列表加载不出来时，用 **ID** 直接填模型名。

## OpenAI 节点（Message a Model）

除了 Chat Model 子节点，n8n 还有一个单独的 **OpenAI** 节点，选 **Message a Model** 就能直接对话。它用同一个凭证，调用的是 Responses 接口，HiveGPT 同样支持。Model 同样选 **ID** 填 `gpt-5.5`。

## 嵌入（Embeddings OpenAI）

做向量检索时用 **Embeddings OpenAI** 节点，凭证选同一个。前提是你的分组里有嵌入模型（用 [`/v1/models`](/connect/#查看可用模型) 查名字里带 `embedding` 的模型），没有的话嵌入部分用其他服务商。

## 自己部署时的注意事项

- **服务器在国内**：HiveGPT 可以直接访问，不需要设代理。如果服务器上设了 `HTTP_PROXY` / `HTTPS_PROXY`，确认代理可用，或在 `NO_PROXY` 里加上 `hivegpt.cn`。
- **长时间运行的工作流**：除了节点的 Timeout，n8n 还有 `N8N_AI_TIMEOUT_MAX` 等环境变量控制 AI 节点的总时长，默认 1 小时，一般不用改。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 「The resource you are requesting could not be found」 | Base URL 少了或多了 `/v1` | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 「Authorization failed - please check your credentials」 | Key 不对或已停用 | 见 [401 报错](/connect/errors/401) |
| 提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 「The service is receiving too many requests from you」 | 工作流并发太高 | 降低批量节点的并发，见 [429 报错](/connect/errors/429) |
| 「Insufficient quota」 | 余额不足或 Key 额度用完 | 见 [余额 / 额度](/connect/errors/quota) |
| 节点运行到一半超时 | Timeout 太短 | 调大 Timeout，见 [请求超时](/connect/errors/timeout) |
| 「Bad request - please check your parameters」 | 参数不被支持，如 temperature | 删掉 Sampling Temperature 再试 |

更多报错看 [报错速查](/connect/errors/)。
