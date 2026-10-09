---
title: OpenCode 配置第三方 API：opencode.json 接入 GPT 模型（含报错排查）
description: 在开源终端编程助手 OpenCode 里接入 HiveGPT：安装、opencode.json 放在哪、provider 的 baseURL 和 apiKey 怎么写、用环境变量保存 Key、选默认模型和推理强度、AGENTS.md 与权限设置，以及 OpenCode 常见报错。
---

# OpenCode 配置第三方 API

**最快的方法：** 在 HiveGPT 的 [API 密钥](https://hivegpt.cn/keys?action=use) 页点 GPT 类分组 Key 的「使用密钥」，选 **OpenCode**，把生成的 `opencode.json` 保存到 `~/.config/opencode/opencode.json`，在项目目录运行 `opencode`，用 `/models` 选一个模型就能开始写代码。

> 更新于 2026-10，在 OpenCode 1.18 上实际跑通（读写文件、执行命令都正常）。

## 安装

任选一种：

```bash
curl -fsSL https://opencode.ai/install | bash     # macOS / Linux
npm install -g opencode-ai                        # 已装 Node.js
brew install anomalyco/tap/opencode               # macOS Homebrew
```

Windows 可以用 `scoop install opencode` 或 `choco install opencode`，官方更推荐在 WSL 里用。装好后运行 `opencode --version` 确认。

## 配置文件放在哪

| 位置 | 作用 |
|---|---|
| `~/.config/opencode/opencode.json` | 全局配置，所有项目都生效。Windows 是 `%USERPROFILE%\.config\opencode\opencode.json` |
| 项目根目录的 `opencode.json` | 只对这个项目生效，和全局配置合并 |

目录不存在就先创建。**Key 不要写进项目里的 `opencode.json`**，那个文件容易被提交到 Git。

## 手写配置

不用弹窗的话，最小配置是这样：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "openai": {
      "options": {
        "baseURL": "https://hivegpt.cn/v1",
        "apiKey": "{env:HIVEGPT_API_KEY}"
      },
      "models": {
        "gpt-5.5": {
          "name": "GPT-5.5",
          "limit": { "context": 1050000, "output": 128000 },
          "options": { "store": false }
        },
        "gpt-6-luna": {
          "name": "GPT-6 Luna",
          "limit": { "context": 1050000, "output": 128000 },
          "options": { "store": false }
        }
      }
    }
  },
  "model": "openai/gpt-5.5",
  "small_model": "openai/gpt-6-luna"
}
```

几个要点：

- **沿用内置的 `openai` 提供方，只改 `baseURL`**。它走 OpenAI 的 Responses 接口，推理模型的参数都能正常传。
- `baseURL` 要带 `/v1`，见 [Base URL 要不要加 /v1](/connect/base-url)。
- `"store": false` 不能省：接口不支持在服务端保存对话，不写可能报错。
- `{env:HIVEGPT_API_KEY}` 表示从环境变量读 Key。在 `~/.zshrc` 或 `~/.bashrc` 里加一行 `export HIVEGPT_API_KEY="sk-你的Key"`，重开终端生效。
- `model` 是默认模型，写成「提供方/模型名」；`small_model` 用来做起标题这类小任务，用便宜的模型就行。
- `models` 里只列你的分组有的模型，名字以 `/v1/models` 返回的为准。
- 内置的 `openai` 提供方自带一份 OpenAI 模型列表，所以 `/models` 里还会出现 `gpt-5.5-pro`、`-fast` 这类名字。不在你分组里的选了会报「不支持模型」，只选你在 `models` 里写的那几个。

弹窗生成的文件还带着每个模型可选的推理强度（`variants`），比手写的全，推荐直接用弹窗的。

## 开始用

在项目目录里运行 `opencode`，常用操作：

| 操作 | 作用 |
|---|---|
| `/models` | 选择模型 |
| `/init` | 让它读一遍项目，生成 `AGENTS.md`（项目说明，之后每次都会读） |
| Tab | 在 Build（直接改代码）和 Plan（只出方案、改动前先问你）之间切换 |
| `@文件名` | 把某个文件加入对话 |
| `/undo` / `/redo` | 撤销 / 重做上一次改动（项目要是 Git 仓库） |
| `/compact` | 对话太长时压缩上下文，省 token |
| `/sessions` | 切换到以前的会话 |

不进界面、直接跑一个任务：

```bash
opencode run "给 utils.py 里的函数补上类型注解"
opencode run -m openai/gpt-6-luna "解释一下这个项目的目录结构"
```

## 让它改文件前先问你

默认它会直接改文件、执行命令。想更稳妥，在配置里加权限：

```json
{
  "permission": {
    "edit": "ask",
    "bash": { "*": "ask", "git status": "allow", "rm *": "deny" }
  }
}
```

`ask` 是每次先问，`allow` 直接执行，`deny` 禁止。命令规则按顺序匹配，后面的覆盖前面的，所以 `"*"` 放在最前面。

## 省钱提示

- 日常改代码用 `gpt-5.5` 或更便宜的 `gpt-6.1-sol`，简单问答、起标题用 `gpt-6-luna`。各模型价格见 [模型怎么选](/connect/models)。
- 对话越长，每次请求带的上下文越多。一个任务做完就 `/new` 开新会话，长会话用 `/compact`。
- 在 [使用记录](https://hivegpt.cn/usage) 里看每次请求花了多少。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `缺少 API Key：请在请求头 Authorization: Bearer ...` | 环境变量没设，`{env:...}` 读到的是空值 | 在启动 opencode 的同一个终端里 `echo $HIVEGPT_API_KEY` 检查，没有就 export 后重开 |
| `/models` 里没有 HiveGPT 的模型 | 配置文件位置不对，或 JSON 写错 | 确认路径是 `~/.config/opencode/opencode.json`，用 JSON 校验工具检查逗号、引号 |
| `UnknownError` / `Unexpected server error` | `-m` 或 `model` 写的模型名 OpenCode 不认识 | 写成 `openai/模型名`，并在 `models` 里列出这个模型 |
| 提示分组「不支持模型」 | 模型不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 404，提示「接口地址少了 /v1」 | `baseURL` 没带 `/v1` | 改成 `https://hivegpt.cn/v1` |
| 400：`Unsupported parameter: reasoningSummary` | 用了 `@ai-sdk/openai-compatible` 自定义提供方 | 按上面的写法沿用内置 `openai` 提供方 |
| 429 或一直重试 | 请求太快或余额不足 | 见 [429 报错](/connect/errors/429)、[余额 / 额度](/connect/errors/quota) |
| 配置改乱了、启动报 `ProviderInitError` | 本地缓存的旧状态 | 修好配置后删除 `~/.cache/opencode` 再启动 |

**OpenCode 能用 Claude 模型吗？** HiveGPT 目前只提供 GPT 系列模型，OpenCode 里选 GPT 模型即可。

**日志在哪？** `~/.local/share/opencode/log/`，或运行时加 `--print-logs` 直接打印出来。

也想试试 OpenAI 自家的编程工具？见 [Codex 接入 HiveGPT](/codex/hivegpt)。其他编程助手：[Cline](/connect/cline) · [在常用工具里配置](/connect/tools)
