---
title: MCP
description: 用 MCP 给 Codex 接上外部工具：codex mcp 命令、config.toml 里 stdio 和 HTTP 两种写法、环境变量、超时、工具白名单与审批、OAuth 登录、常用服务器示例和排错。
---

# MCP

Codex 自带的能力是读写文件和运行命令。想让它查 GitHub issue、操作浏览器、检索最新的库文档、查询数据库，就需要给它接上额外的工具。MCP（Model Context Protocol，模型上下文协议）是这类连接的通用标准：工具的提供方写一个 MCP 服务器，任何支持 MCP 的客户端（Codex、各种 IDE 和其他编程代理）都能直接使用。

这一页讲怎样在 Codex 里添加、配置和管理 MCP 服务器，给出几个常用服务器的配置，并说明怎么排查连接问题。

## 基本概念

- **MCP 服务器**：一个提供「工具」的程序。比如 GitHub 的 MCP 服务器提供「搜索 issue」「读取 PR」等工具。
- **两种连接方式**：
  - **stdio**：Codex 在本机启动服务器进程，通过标准输入输出通信。大多数用 `npx` 或 `uvx` 启动的服务器都是这种。
  - **Streamable HTTP**：服务器运行在远端（或本机某个端口），Codex 通过 URL 连接。托管服务通常用这种，常配合 token 或 OAuth 登录。
- **工具调用**：连接成功后，服务器的工具会出现在 Codex 的工具列表里，模型在需要时调用。调用会受审批设置约束。

## 用命令管理服务器

`codex mcp` 子命令会直接读写 `~/.codex/config.toml`，适合快速添加：

```bash
# 添加 stdio 服务器：-- 之后是启动命令
codex mcp add context7 -- npx -y @upstash/context7-mcp

# 启动命令需要环境变量时用 --env（仅 stdio 可用，可重复）；some-postgres-mcp 是示意，换成你实际使用的包
codex mcp add pg --env DATABASE_URL=postgresql://localhost/devdb -- npx -y some-postgres-mcp

# 添加 HTTP 服务器
codex mcp add openaiDocs --url https://developers.openai.com/mcp

# HTTP 服务器用环境变量里的 token 认证（变量名，不是 token 本身）
codex mcp add github --url https://api.githubcopilot.com/mcp/ --bearer-token-env-var GITHUB_PAT

# 查看、详情、删除
codex mcp list
codex mcp list --json
codex mcp get github
codex mcp remove pg

# 需要 OAuth 的服务器：登录和退出
codex mcp login figma
codex mcp login figma --scopes read,write
codex mcp logout figma
```

在会话里，输入 `/mcp` 可以看到当前连接的服务器和工具；`/mcp verbose` 显示更详细的状态；`/mcp login 名字` 可以直接在会话里发起 OAuth 登录。

## 在 config.toml 里配置

命令行只覆盖最常用的选项，更细的配置要直接编辑 `~/.codex/config.toml`。每个服务器是一个 `[mcp_servers.名字]` 表。以下字段依据源码 `codex-rs/config/src/mcp_types.rs` 整理。

### stdio 服务器

```toml
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]

[mcp_servers.pg]
command = "uvx"
args = ["some-postgres-mcp"]                   # 示意包名
cwd = "/Users/me/projects/shop"                 # 服务器进程的工作目录
env = { DATABASE_URL = "postgresql://localhost/devdb" }   # 直接写入的变量
env_vars = ["PGPASSWORD"]                       # 从你的 shell 环境透传的变量名
```

::: warning stdio 服务器不会继承你全部的环境变量
这是最常见的坑。出于安全考虑，Codex 启动 stdio 服务器时只传递少量基础变量（`HOME`、`PATH`、`SHELL`、`USER`、`LANG`、`TERM`、`TMPDIR`、`TZ` 等）。你在终端里 `export GITHUB_TOKEN=...` 之后，服务器进程**看不到**它。需要的变量要么写进 `env`，要么把变量名列进 `env_vars` 让 Codex 透传。密钥类的值推荐用 `env_vars`，避免明文写进配置文件。
:::

### HTTP 服务器

```toml
[mcp_servers.github]
url = "https://api.githubcopilot.com/mcp/"
bearer_token_env_var = "GITHUB_PAT"             # 以 Authorization: Bearer 发送该变量的值

[mcp_servers.internal-docs]
url = "https://mcp.example.internal/mcp"
http_headers = { "X-Team" = "platform" }        # 固定的请求头
env_http_headers = { "X-Api-Key" = "DOCS_API_KEY" }   # 请求头的值取自环境变量
```

HTTP 服务器不支持 `args`、`env`、`cwd` 这些 stdio 专用字段；也没有直接写 token 的字段，token 必须通过 `bearer_token_env_var` 从环境变量读取，这样密钥不会落在配置文件里。

### 通用选项

```toml
[mcp_servers.playwright]
command = "npx"
args = ["@playwright/mcp@latest"]
enabled = true                 # 设为 false 可临时停用，不必删除配置
required = false               # 为 true 时，codex exec 在该服务器启动失败时直接报错退出
startup_timeout_sec = 60       # 启动并列出工具的超时，默认 30 秒
tool_timeout_sec = 120         # 单次工具调用的超时，默认 300 秒
```

::: tip 默认值以你的版本为准
上面的默认超时取自当前官方源码。Codex 更新频繁，不同版本可能不同；首次启动慢的服务器（比如第一次 `npx` 需要下载包），建议显式把 `startup_timeout_sec` 调大。
:::

### 只暴露需要的工具

有的服务器一次提供几十个工具，全部暴露既占上下文，也扩大了误操作的范围。可以用白名单或黑名单收窄：

```toml
[mcp_servers.github]
url = "https://api.githubcopilot.com/mcp/"
bearer_token_env_var = "GITHUB_PAT"
enabled_tools = ["search_issues", "get_issue", "get_pull_request", "list_pull_requests"]
disabled_tools = ["get_pull_request"]   # 在白名单基础上再剔除
```

`enabled_tools` 设置后只有列出的工具会注册；`disabled_tools` 在它之后生效。工具名可以在 `/mcp` 里查看。

### 工具审批

MCP 工具调用也受审批控制。可以为整个服务器设默认值，也可以按工具单独设置：

```toml
[mcp_servers.github]
url = "https://api.githubcopilot.com/mcp/"
bearer_token_env_var = "GITHUB_PAT"
default_tools_approval_mode = "writes"

[mcp_servers.github.tools.search_issues]
approval_mode = "approve"

[mcp_servers.github.tools.merge_pull_request]
approval_mode = "prompt"
```

| 取值 | 行为 |
|---|---|
| `auto`（默认） | 根据工具自己声明的属性判断：标为有破坏性或会访问外部世界的工具需要确认 |
| `prompt` | 每次调用都询问 |
| `writes` | 工具声明为只读的直接放行，其余询问 |
| `approve` | 从不询问，直接执行 |

`approve` 只给你完全信任且没有副作用的工具使用。工具的「只读」「有破坏性」等属性由服务器作者声明，不一定准确，对写操作宁可多问一次。

## OAuth 登录

很多托管的 MCP 服务器用 OAuth 认证。流程是：

1. 配置好 `url`（必要时用 `scopes` 和 `oauth_resource` 指定权限范围和资源参数）。
2. 运行 `codex mcp login 名字`，Codex 会打开浏览器让你授权，并在本机起一个回调地址接收结果。
3. 凭证保存后，之后的会话自动使用，过期会自动刷新。

相关的全局配置：

```toml
# 凭证存放位置：auto（默认，优先系统钥匙串，不可用时退回文件）、keyring、file
mcp_oauth_credentials_store = "auto"
# 固定回调端口，适合需要在防火墙或端口转发里放行的场景
mcp_oauth_callback_port = 8765
# 在远程开发机上使用时，可以指定外部可访问的回调地址
# mcp_oauth_callback_url = "https://devbox.example.com/callback"
```

选 `file` 时凭证写在 `~/.codex/.credentials.json`，同一用户下的其他程序也能读到；有条件尽量用钥匙串。

## 常用服务器示例

以下是几类常用服务器的配置写法。包名和地址以各项目官方说明为准，第一次使用前建议先在终端里手动运行一次启动命令，确认能正常启动。

**库文档检索（Context7）**：让 Codex 查询第三方库的最新文档，减少按过时 API 写代码的情况。

```toml
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]
```

**OpenAI 官方开发者文档**：查询 OpenAI API 和 Codex 自身的文档。

```toml
[mcp_servers.openaiDocs]
url = "https://developers.openai.com/mcp"
```

**GitHub**：查 issue、PR、代码。使用 GitHub 提供的托管服务器和个人访问令牌（建议只给需要的仓库和只读权限）。

```toml
[mcp_servers.github]
url = "https://api.githubcopilot.com/mcp/"
bearer_token_env_var = "GITHUB_PAT"
```

**浏览器自动化（Playwright）**：打开页面、点击、截图，适合验证前端改动。

```toml
[mcp_servers.playwright]
command = "npx"
args = ["@playwright/mcp@latest"]
startup_timeout_sec = 90
```

新版 Codex 本身也内置了浏览器操作能力（见 [电脑操控](/codex/computer-use)），简单场景不一定需要再接 Playwright。

**文件系统**：让 Codex 访问工作区之外的某个目录（比如共享的设计文档目录），并且只限这个目录。

```toml
[mcp_servers.design-docs]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/Users/me/Documents/design-docs"]
```

::: warning MCP 服务器是你主动运行的第三方代码
stdio 服务器以你的用户身份在本机运行，不受 Codex 沙箱限制；HTTP 服务器会收到 Codex 发给它的数据。只安装来源可信的服务器，给 token 分配最小权限，并用 `enabled_tools` 限制暴露的工具。
:::

## 项目级配置

上面的配置写在 `~/.codex/config.toml`，对所有项目生效。如果某个服务器只和某个项目有关（比如连接该项目的开发数据库），可以写进项目的 `.codex/config.toml`，格式相同。项目级配置只在你信任该项目后才会加载。

需要临时改某个值时，也可以在启动时用 `-c` 覆盖：

```bash
codex -c 'mcp_servers.playwright.enabled=false'
```

## 把 Codex 作为 MCP 服务器

反过来，Codex 自己也可以作为一个 MCP 服务器，被其他支持 MCP 的程序调用。在 0.142 等版本中，运行：

```bash
codex mcp-server
```

会以 stdio 方式启动，对外提供两个工具：`codex`（开启一个新的 Codex 会话，参数包括 `prompt`、`cwd`、`model`、`sandbox`、`approval-policy` 等）和 `codex-reply`（按会话 ID 继续对话）。这样可以让另一个代理或编排脚本把编码任务委派给 Codex。

在其他客户端里注册它，通常就是一个 stdio 配置，命令为 `codex`、参数为 `mcp-server`。

::: details 版本差异
写作时的最新官方源码里已经找不到 `mcp-server` 子命令，程序化集成的方向转向了 `codex app-server` 和官方 SDK。你本机是否还有这个命令，以 `codex --help` 的输出为准；新项目做集成，优先参考 [官方文档](https://developers.openai.com/codex) 中 SDK 和 app-server 的说明。
:::

## 排错

| 现象 | 排查方向 |
|---|---|
| `/mcp` 里看不到服务器 | `codex mcp list` 确认配置存在；检查 `enabled` 是否为 false；项目级配置要求项目已受信任 |
| 启动超时 | 先在终端手动运行 `command` + `args`，看是否在下载包或报错；调大 `startup_timeout_sec` |
| 服务器报「缺少 token」 | stdio 服务器不继承你的全部环境变量，用 `env` 或 `env_vars` 传入 |
| HTTP 返回 401 | 检查 `bearer_token_env_var` 指向的变量在启动 Codex 的那个终端里是否存在；OAuth 服务器重新 `codex mcp login` |
| 工具调用超时 | 调大 `tool_timeout_sec`；检查服务器端是否卡在某个慢请求 |
| 工具太多、回答变慢 | 用 `enabled_tools` 只保留需要的工具 |
| 配置改了没生效 | 退出 Codex 重新启动；用 `codex mcp get 名字 --json` 核对解析结果 |
| 找不到 `npx` / `uvx` | 确认已安装 Node.js 或 uv，且所在目录在 `PATH` 里；必要时把 `command` 写成绝对路径 |

还可以运行 `codex doctor` 检查本机安装、配置和认证的整体状态。

## 小结

- MCP 是连接外部工具的通用协议，Codex 支持本机 stdio 和远程 Streamable HTTP 两种服务器。
- `codex mcp add/list/get/remove/login/logout` 管理服务器；会话中用 `/mcp` 查看状态。
- stdio 服务器只继承少量基础环境变量，密钥用 `env_vars` 透传；HTTP 服务器的 token 必须走 `bearer_token_env_var`。
- 用 `enabled_tools` / `disabled_tools` 收窄工具，用 `approval_mode` 控制哪些调用需要确认。
- MCP 服务器是第三方代码，只装可信来源，给最小权限。

下一步：让多个代理分工协作，见 [子代理](/codex/subagents)。
