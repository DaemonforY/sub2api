---
title: Continue 插件配置 OpenAI 兼容 API：config.yaml 接入 GPT 模型（VS Code / JetBrains）
description: 在 VS Code 或 JetBrains 的 Continue 插件里接入 HiveGPT：~/.continue/config.yaml 怎么写（provider openai、apiBase、apiKey、roles、capabilities），用 .env 保存 Key，Agent 模式工具调用，为什么不要把 GPT 设成自动补全模型，以及常见报错；附 Continue 停止维护的说明和替代方案。
---

# Continue 插件配置 OpenAI 兼容 API

**最短答案：** 编辑 `~/.continue/config.yaml`，加一个 `provider: openai` 的模型，`apiBase` 填 `https://hivegpt.cn/v1`，`apiKey` 填你的 HiveGPT Key，`model` 填 `gpt-5.5`；保存后在 VS Code 里执行 **Reload Window**，在 Continue 面板顶部选这个模型即可。

::: warning Continue 已停止维护
2026 年 Continue 官方宣布不再维护：GitHub 仓库已改为只读，VS Code 版的最后一个正式版本是 2.0，JetBrains 插件停在 1.0.67。现有版本还能正常使用，但以后不会再修 bug、适配新模型。

新装的话，更推荐还在更新的工具：VS Code 里用 [Cline](/connect/cline)，终端里用 [Codex](/codex/hivegpt) 或 [OpenCode](/connect/opencode)。
:::

> 更新于 2026-10，基于 Continue 2.0（VS Code）。

## 准备

- VS Code 扩展市场搜索 **Continue** 安装（ID `Continue.continue`）；JetBrains 在插件市场搜索 Continue。不需要登录 Continue 账号，本地配置文件就能用。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。

## 第一步：保存 Key

不要把 Key 直接写进配置文件。新建 `~/.continue/.env`（Windows 是 `%USERPROFILE%\.continue\.env`），写一行：

```bash
HIVEGPT_API_KEY=sk-你的Key
```

## 第二步：编辑 config.yaml

打开 `~/.continue/config.yaml`（Windows 是 `%USERPROFILE%\.continue\config.yaml`；也可以在 Continue 面板右上角的设置里打开），写成：

```yaml
name: My Config
version: 0.0.1
schema: v1

models:
  - name: GPT-5.5（HiveGPT）
    provider: openai
    model: gpt-5.5
    apiBase: https://hivegpt.cn/v1
    apiKey: ${{ secrets.HIVEGPT_API_KEY }}
    roles: [chat, edit, apply]
    capabilities: [tool_use, image_input]
    defaultCompletionOptions:
      contextLength: 272000   # 超过 27.2 万 token 会按长上下文价计费，到这里就让 Continue 裁剪历史
      maxTokens: 16000

  - name: GPT-6 Luna（便宜、快）
    provider: openai
    model: gpt-6-luna
    apiBase: https://hivegpt.cn/v1
    apiKey: ${{ secrets.HIVEGPT_API_KEY }}
    roles: [chat, edit, apply]
    capabilities: [tool_use, image_input]
```

要点：

- `provider` 用 `openai`，`apiBase` 带 `/v1`，末尾不要写 `/chat/completions`。
- `roles` 只写 `chat`、`edit`、`apply`。**不要加 `autocomplete` 和 `embed`**，原因见下文。
- `capabilities` 写上 `tool_use`，Agent 模式才会用原生工具调用；写上 `image_input`，才能往对话里贴截图。
- `contextLength` 设成 272000：单次请求输入超过 27.2 万 token 会整次按更高的长上下文价计费，设在这里 Continue 会提前裁掉旧的对话。
- `model` 以你的分组能用的为准，见 [查看可用模型](/connect/#查看可用模型)；怎么选见 [GPT 模型怎么选](/connect/models)。

保存后在 VS Code 按 `Cmd/Ctrl + Shift + P`，执行 **Developer: Reload Window**。

## 第三步：开始用

在 Continue 面板顶部的模型下拉里选「GPT-5.5（HiveGPT）」：

- **Chat**：问问题、解释代码。选中代码后按 `Cmd/Ctrl + L` 加入对话。
- **Edit**：选中代码按 `Cmd/Ctrl + I`，说要怎么改。
- **Agent**：让它自己读文件、改多个文件、运行命令（需要上面写的 `tool_use`）。

## 自动补全（Tab）不要用 GPT

Continue 的自动补全需要专门的「补全模型」（FIM 格式），GPT 这类对话模型没有按这个格式训练，设成补全模型会补出乱七八糟的内容，还会一直消耗 token。HiveGPT 目前不提供补全模型，所以：

- 不要给 HiveGPT 的模型加 `autocomplete` 角色；
- 需要 Tab 补全的话，可以单独配置本地的补全模型（如 Ollama 上的小模型），或者用 IDE 自带的补全。

## 代码库索引

`@Codebase` 已被 Continue 标为弃用，Agent 模式会自己用搜索工具找代码，一般不需要索引。需要索引时，VS Code 版自带本地嵌入模型，保持默认即可；**不要给 HiveGPT 的模型加 `embed` 角色**，HiveGPT 不提供嵌入模型。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 404，提示「接口地址少了 /v1」或「多了一个 /v1」 | `apiBase` 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 401 / 提示缺少 API Key | `.env` 没读到 | 检查 `~/.continue/.env` 里的变量名和 config.yaml 里 `secrets.` 后面的名字一致，Reload Window；还不行就先把 Key 直接写进 `apiKey` 试试 |
| 改了配置没生效、模型不在列表里 | 没重新加载，或 YAML 格式错 | Reload Window；检查缩进（只能用空格）和顶部的 `name`、`version`、`schema` |
| 提示分组「不支持模型」 | `model` 不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| Agent 模式不调用工具、只会说 | 没声明工具能力 | `capabilities` 加上 `tool_use` |
| 贴截图后模型说看不到图 | 没声明图片能力 | `capabilities` 加上 `image_input` |
| Tab 补全出乱码、余额掉得快 | 给 GPT 加了 `autocomplete` 角色 | 从 `roles` 里删掉 `autocomplete` |
| 429 | 请求太频繁或余额不足 | 见 [429 报错](/connect/errors/429)、[余额 / 额度](/connect/errors/quota) |

**日志在哪？** VS Code：命令面板执行 **Developer: Toggle Developer Tools**，看 Console；JetBrains：`~/.continue/logs/core.log`。

**公司网络要走代理？** 在模型下加 `requestOptions: { proxy: http://代理地址:端口 }`，或者使用 VS Code 自己的代理设置。

更多 AI 编程工具：[Cline](/connect/cline) · [OpenCode](/connect/opencode) · [Codex 接入 HiveGPT](/codex/hivegpt) · [在常用工具里配置](/connect/tools)
