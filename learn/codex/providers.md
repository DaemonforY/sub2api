---
title: 接入 DeepSeek 等其他模型
description: 理解 model_providers 的工作方式，接入 DeepSeek（官方脚本或手动配置）、Ollama / LM Studio 本地模型和 Azure，用配置档在多个提供方之间切换，并了解兼容性上的坑。
---

# 接入 DeepSeek 等其他模型

Codex 并不绑定 OpenAI 官方接口。只要对方提供兼容的 Responses API，就可以在 `config.toml` 里把它声明为一个「模型提供方」（model provider），让 Codex 改用它的模型。[接入 HiveGPT](/codex/hivegpt) 就是这种方式；这一页讲清楚背后的原理，并给出 DeepSeek、本地模型、Azure 的接法，以及如何在多个提供方之间切换。

## 原理：model_providers

Codex 发请求时要知道三件事：发到哪个地址、用什么协议、带什么凭据。这些都写在 `[model_providers.标识]` 一段里，再用顶层的 `model_provider` 选中它：

```toml
model_provider = "myprovider"   # 选用哪个提供方
model = "some-model"            # 发给该提供方的模型名

[model_providers.myprovider]
name = "My Provider"            # 显示名
base_url = "https://api.example.com/v1"
wire_api = "responses"
env_key = "MY_PROVIDER_API_KEY" # 从这个环境变量读取 Key
```

`[model_providers.*]` 中常用的字段（以 Codex 源码为准）：

| 字段 | 作用 |
|---|---|
| `name` | 显示名称 |
| `base_url` | 接口根地址，Codex 会在后面拼 `/responses` 等路径 |
| `wire_api` | 通信协议。**目前只支持 `"responses"`**，旧的 `"chat"` 已移除，写了会报错 |
| `env_key` | 存放 API Key 的环境变量名，Codex 用它组成 `Authorization: Bearer ...` |
| `env_key_instructions` | 环境变量缺失时给用户看的提示 |
| `experimental_bearer_token` | 直接把 Key 写在配置里。官方不推荐，能用 `env_key` 就用 `env_key` |
| `requires_openai_auth` | 设为 `true` 时走 Codex 的登录流程，从 `auth.json` 取凭据；默认 `false` |
| `query_params` | 追加到 URL 上的查询参数，例如 Azure 的 `api-version` |
| `http_headers` | 固定的额外请求头 |
| `env_http_headers` | 值从环境变量读取的请求头，变量为空时不发送 |
| `request_max_retries` / `stream_max_retries` | 请求失败、流式中断后的重试次数 |
| `stream_idle_timeout_ms` | 流式响应多久没动静算断开 |
| `supports_websockets` | 是否支持 Responses 的 WebSocket 传输；第三方接口一般保持默认（不支持） |

还有两个顶层配置经常配合使用：

- `model_catalog_json`：指向一个模型目录 JSON 文件，告诉 Codex 每个模型的上下文窗口、支持的推理强度、工具调用方式等。第三方模型不在 Codex 内置目录里，**没有目录时 Codex 只能用通用的默认值猜**，容易出现上下文估算不准、推理强度选项不对等问题。
- `model_reasoning_effort`：默认推理强度，取值必须是该模型支持的档位。

::: warning 内置提供方的名字不能覆盖
Codex 内置了 `openai`、`ollama`、`lmstudio`、`amazon-bedrock` 等几个提供方。你在配置里写 `[model_providers.openai]` 去改它的地址是**无效的**，会被内置定义忽略。要么换一个自己的标识（比如 HiveGPT 弹窗用的是大写的 `OpenAI`），要么用顶层的 `openai_base_url` 只改内置 openai 的地址。
:::

## DeepSeek：官方方式

2026 年 8 月，DeepSeek 官方宣布支持接入 Codex，先支持的是 `deepseek-v4-flash`，`deepseek-v4-pro` 随后跟进。DeepSeek 的接口原生支持 Responses 协议，所以可以直接作为提供方接入。

准备工作：

1. Codex 已安装，并且**至少启动过一次**（让 `~/.codex` 目录生成出来）。DeepSeek 的模型目录对 Codex 版本有最低要求，先运行 `codex update` 升到最新。
2. 到 [DeepSeek 开放平台](https://platform.deepseek.com) 创建一个 API Key。

### 方式一：官方一键脚本

macOS / Linux：

```bash
bash <(curl -fsSL https://cdn.deepseek.com/api-docs/codex-deepseek-setup.sh)
```

Windows PowerShell：

```powershell
irm https://cdn.deepseek.com/api-docs/codex-deepseek-setup-en.ps1 | iex
```

::: warning 先读脚本，再运行
这类命令会把远程脚本下载下来直接执行，而且它要修改你的 Codex 配置。运行前先在浏览器里打开上面的地址，确认是从 DeepSeek 官方域名下载、内容和预期一致。
:::

脚本是菜单式的：选择要用的模型后，首次运行会要求输入 API Key（如果环境变量 `DEEPSEEK_API_KEY` 已经有值，就直接用它）。它会做这几件事：

- 把原来的 `~/.codex/config.toml` 备份到 `~/.codex/backup-deepseek/`；
- 写入模型目录 `~/.codex/models.json`，声明 DeepSeek 模型的元数据；
- 在 `config.toml` 里新增 `[model_providers.deepseek]` 并改写必要的顶层字段，原有的 MCP、项目信任等配置保留；
- 写入前先校验文件语法，校验不过就中止。

以后想换模型或恢复原来的配置，重新运行脚本，在菜单里选对应选项即可。配置完成后重启 Codex（桌面端要完全退出再打开），`/model` 里就能看到 DeepSeek 的模型。

### 方式二：手动配置

想自己掌控配置，可以手动写。模型目录文件里除了上下文窗口、推理档位等字段，还包含一大段供 Codex 内部使用的提示词模板，手写容易出错，**建议先用脚本生成 `models.json`**，或者从官方文档获取完整文件，再手动调整 `config.toml`：

```toml
# 以下顶层键放在 config.toml 开头
model_provider = "deepseek"
model = "deepseek-v4-flash"
model_reasoning_effort = "high"          # 该模型的档位是 low / high / max
model_catalog_json = "~/.codex/models.json"

[model_providers.deepseek]
name = "DeepSeek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"
env_key = "DEEPSEEK_API_KEY"
```

然后在 shell 配置文件里设置 Key：

```bash
echo 'export DEEPSEEK_API_KEY="sk-..."' >> ~/.zshrc
source ~/.zshrc
```

PowerShell 可以用 `[Environment]::SetEnvironmentVariable("DEEPSEEK_API_KEY", "sk-...", "User")` 设置用户级环境变量，设置后重开终端。

官方示例里用的是 `experimental_bearer_token` 把 Key 直接写进配置，能用但 Key 会以明文出现在 `config.toml` 里；上面的 `env_key` 写法更安全，也方便把配置文件分享给同事。

::: tip HiveGPT 上的其他平台分组
如果你在 HiveGPT 能使用 DeepSeek 等非 OpenAI 平台的分组，同样可以在「使用密钥」弹窗选「Codex CLI」生成配置。这类配置用 `env_key = "SUB2API_API_KEY"` 从环境变量读 Key，并配套下载模型目录，按弹窗提示操作即可。
:::

## 图形化切换工具

社区里有一些桌面工具（例如 CC Switch）专门用来管理多个提供方：在界面里填好各家的地址和 Key，一键切换，有的还会在本机起一个代理把请求转发到目标服务。它们本质上做的事情和本页一样——改写 `~/.codex/config.toml`，或者让 `base_url` 指向本机代理。

用这类工具前要清楚两点：它们会接触你所有的 API Key，只从项目官方渠道下载；切换出问题时，直接打开 `config.toml` 看它写了什么，比在界面里反复点更快定位。

## 本地模型：Ollama 与 LM Studio

Codex 内置了 `ollama`（默认 `http://localhost:11434/v1`）和 `lmstudio`（默认 `http://localhost:1234/v1`）两个本地提供方，用 `--oss` 参数启用：

```bash
codex --oss                              # 使用配置里的 oss_provider，没配置会让你选
codex --oss --local-provider ollama      # 明确用 Ollama
codex --oss --local-provider lmstudio -m 你的模型名
```

使用 Ollama 且不指定模型时，默认模型是 `gpt-oss:20b`，需要先在 Ollama 里下载好。想固定使用某个本地提供方，在配置里写：

```toml
oss_provider = "ollama"
```

本地服务不在默认端口时，可以用环境变量 `CODEX_OSS_BASE_URL` 或 `CODEX_OSS_PORT` 指定。旧的 `ollama-chat` 提供方已经移除，配置里有的话改成 `ollama`。

::: warning 本地模型的现实预期
Codex 的工作强度很高：一个任务要连续调用很多次工具，上下文动辄几万 token。小参数量的本地模型经常出现工具调用格式错误、改到一半放弃、陷入重复等问题。本地模型适合离线、保密要求高的简单任务；日常开发还是用云端的强模型更省时间。另外本地服务必须支持 Responses API，版本太旧的 Ollama / LM Studio 需要先升级。
:::

## Azure OpenAI

Azure 的部署通过 `query_params` 携带 API 版本：

```toml
model_provider = "azure"
model = "你的部署名"

[model_providers.azure]
name = "Azure OpenAI"
base_url = "https://你的资源名.openai.azure.com/openai"
env_key = "AZURE_OPENAI_API_KEY"
query_params = { api-version = "2025-04-01-preview" }
wire_api = "responses"
```

`api-version` 的具体取值以 Azure 文档中支持 Responses API 的版本为准。Amazon Bedrock 是 Codex 内置的提供方，使用 AWS 凭据认证，配置方式见 [高级配置](/codex/config-advanced)。

## 在多个提供方之间切换

### 临时切换：-c 参数

`-c` 可以在启动时覆盖任意配置项，适合偶尔试一下别的模型：

```bash
codex -c model_provider='"deepseek"' -c model='"deepseek-v4-flash"'
```

值按 TOML 解析，所以规范写法是字符串带双引号、外层再用单引号包住以免被 shell 吃掉；解析失败时 Codex 会把它当普通字符串，因此 `-c model=deepseek-v4-flash` 这样的简写通常也能用。

### 长期切换：配置档

新版 Codex 的配置档是 `~/.codex/名字.config.toml` 文件，用 `-p 名字` 启动时叠加到主配置之上。比如主配置用 HiveGPT，再为 DeepSeek 建一个 `~/.codex/deepseek.config.toml`：

```toml
model_provider = "deepseek"
model = "deepseek-v4-flash"
model_reasoning_effort = "high"
model_catalog_json = "~/.codex/models.json"

[model_providers.deepseek]
name = "DeepSeek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"
env_key = "DEEPSEEK_API_KEY"
```

```bash
codex                 # 主配置：HiveGPT
codex -p deepseek     # DeepSeek
```

`codex exec`、`codex resume` 等子命令同样支持 `-p`。旧版在 `config.toml` 里写 `[profiles.xxx]` 表、顶层 `profile = "xxx"` 的做法已不再支持。更多见 [高级配置](/codex/config-advanced)。

::: tip 每个提供方用自己的模型目录
`model_catalog_json` 是全局的一份文件。主配置和配置档指向不同的目录文件（比如 HiveGPT 用 `codex-models.json`，DeepSeek 用 `models.json`），切换时 `/model` 列表才对得上。这个设置只在启动时读取，切换提供方要重新启动 Codex。
:::

## 兼容性：哪些地方可能不一样

Codex 的很多能力是围绕 OpenAI 模型设计的，换成其他模型后要有心理准备：

| 方面 | 可能的表现 |
|---|---|
| 工具调用 | 模型不熟悉 Codex 的补丁格式和工具约定时，会出现改文件失败、反复重试 |
| 推理强度 | 只能选模型目录里声明的档位；写了不支持的值，可能被忽略或接口报错 |
| 图片输入 | 只支持文本的模型无法处理截图、设计稿 |
| 联网搜索 | 内置的网页搜索依赖提供方支持，第三方接口通常没有 |
| 上下文压缩 | 远程压缩协议不支持时会退回本地压缩，长会话表现可能不同 |
| 计费与用量 | `/status` 里的额度信息针对 ChatGPT 账号，第三方用量要去对方控制台看 |

遇到奇怪的问题，先换回 `gpt-5.5` 跑同一个任务，就能判断是模型能力问题还是配置问题。

## 小结

- 提供方写在 `[model_providers.标识]`，用顶层 `model_provider` 选中；`wire_api` 只能是 `responses`，Key 优先用 `env_key`。
- 第三方模型要配 `model_catalog_json` 模型目录，否则 Codex 只能用默认值猜测模型能力。
- DeepSeek 可以用官方一键脚本接入（运行前先看脚本），也可以手动配置。
- 本地模型用 `codex --oss`，内置 Ollama 和 LM Studio，但能力有限。
- 多个提供方用 `名字.config.toml` 配置档加 `codex -p 名字` 切换。

下一步：[第一个任务](/codex/first-task)
