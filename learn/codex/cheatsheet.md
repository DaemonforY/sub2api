---
title: 速查表
description: Codex CLI 一页速查：安装登录、启动参数、子命令、斜杠命令、快捷键、配置键、沙箱与审批组合、exec 参数、MCP 命令和文件位置。
---

# 速查表

这一页把 Codex CLI 最常用的东西整理成表格，方便忘了的时候查。内容以 codex-cli **0.142.3** 的 `--help` 和 OpenAI 官方仓库源码为准；版本更新后个别命令可能变化，遇到对不上的情况，以 `codex --help`、TUI 里的 `/` 菜单和 `?` 快捷键提示为准。

各项的详细解释见对应页面：[命令行参数](/codex/cli-reference)、[斜杠命令](/codex/slash-commands)、[config.toml 基础](/codex/config)、[沙箱与审批](/codex/sandbox)。

## 安装、更新与登录

| 操作 | 命令 |
|---|---|
| 安装（需要 Node.js 18+） | `npm install -g @openai/codex` |
| 安装（macOS Homebrew） | `brew install --cask codex` |
| 查看版本 | `codex --version` |
| 更新 | `codex update`，或重新执行 npm 安装命令 |
| 自检安装、配置、认证 | `codex doctor`（`--summary` 只看汇总，`--json` 输出脱敏报告） |
| ChatGPT 账号登录（浏览器） | `codex login` |
| 无浏览器的机器登录 | `codex login --device-auth` |
| 用 API Key 登录（从 stdin 读） | `printenv OPENAI_API_KEY \| codex login --with-api-key` |
| 查看登录状态 | `codex login status` |
| 退出登录 | `codex logout` |
| 接入 HiveGPT | 用控制台「使用密钥」弹窗生成配置，见 [接入 HiveGPT](/codex/hivegpt) |

## 启动参数（codex）

| 参数 | 作用 |
|---|---|
| `codex "指令"` | 带着第一条消息启动 |
| `-m, --model <模型>` | 指定模型，如 `-m gpt-5.5` |
| `-s, --sandbox <模式>` | `read-only` / `workspace-write` / `danger-full-access` |
| `-a, --ask-for-approval <策略>` | `untrusted` / `on-request` / `never`（`on-failure` 已废弃） |
| `--dangerously-bypass-approvals-and-sandbox` | 无沙箱、无审批，别名 `--yolo`。只在外部已隔离的环境用 |
| `-C, --cd <目录>` | 指定工作根目录 |
| `--add-dir <目录>` | 额外可写目录，可重复 |
| `-i, --image <文件>` | 附加图片，可多个 |
| `-c key=value` | 临时覆盖配置，值按 TOML 解析，如 `-c model_reasoning_effort="high"` |
| `-p, --profile <名字>` | 叠加 `~/.codex/<名字>.config.toml` |
| `--enable / --disable <功能>` | 开关功能开关，等价于 `-c features.<名字>=true/false` |
| `--search` | 开启实时网页搜索 |
| `--oss`、`--local-provider` | 使用本地模型（lmstudio / ollama） |
| `--no-alt-screen` | 不用全屏模式，保留终端滚动记录 |
| `--strict-config` | 配置里有不认识的字段就报错 |
| `--remote <地址>` | 连接远程 app server（实验性） |

## 子命令

| 子命令 | 作用 |
|---|---|
| `codex exec` / `codex e` | 非交互运行，见 [非交互模式与 CI/CD](/codex/exec) |
| `codex review` | 非交互代码审查 |
| `codex resume` | 恢复以前的会话（默认弹出列表，`--last` 直接接最近一次，`--all` 不限当前目录） |
| `codex fork` | 从以前的会话分叉出新会话 |
| `codex archive / unarchive / delete` | 归档、取消归档、永久删除会话 |
| `codex apply` / `codex a` | 把 Codex 最近产生的 diff 用 `git apply` 应用到本地 |
| `codex mcp` | 管理 MCP 服务器（见下文） |
| `codex mcp-server` | 把 Codex 本身作为 MCP 服务器（stdio）运行 |
| `codex plugin` | 管理插件 |
| `codex features list / enable / disable` | 查看和开关功能开关 |
| `codex sandbox <命令>` | 在 Codex 的沙箱里运行任意命令，调试沙箱用 |
| `codex debug models` | 输出当前生效的模型目录（JSON） |
| `codex completion <shell>` | 生成 shell 补全脚本 |
| `codex app` | 打开桌面应用（没装会打开安装页） |
| `codex cloud` | 浏览和应用云端任务（实验性） |
| `codex app-server` | 运行 app server（实验性） |

## 斜杠命令

在 TUI 输入框里输入 `/` 会弹出菜单，可以继续输入过滤。下表是 0.142.3 中可见的命令：

| 命令 | 作用 |
|---|---|
| `/model` | 选择模型和推理强度 |
| `/permissions` | 选择权限预设（Read Only / Default / Full Access） |
| `/personality` | 选择 Codex 的沟通风格 |
| `/plan [任务]` | 切换到计划模式，可直接带上任务 |
| `/goal` | 为长时间任务设置或查看目标 |
| `/review [指令]` | 审查改动：对比分支、未提交改动、某个提交，或自定义要求 |
| `/diff` | 查看 git diff（包含未跟踪文件） |
| `/mention` | 提及一个文件（也可以直接输入 `@`） |
| `/init` | 生成 `AGENTS.md` |
| `/compact` | 压缩对话，避免超出上下文 |
| `/new` | 在当前会话中开始新对话 |
| `/clear` | 清屏并开始新对话 |
| `/resume` | 恢复已保存的对话 |
| `/fork` | 分叉当前对话 |
| `/side`、`/btw` | 在临时分叉里开一个旁支对话，不影响主线 |
| `/rename` | 重命名当前会话 |
| `/archive` | 归档当前会话并退出 |
| `/delete` | 永久删除当前会话并退出 |
| `/copy` | 复制上一条回复（Markdown） |
| `/raw` | 切换原始滚动模式，方便在终端里选择复制 |
| `/status` | 显示当前会话配置和 token 用量 |
| `/usage` | 查看账号用量（ChatGPT 登录时有意义） |
| `/debug-config` | 显示配置分层和来源，排查配置不生效 |
| `/mcp` | 列出已配置的 MCP 工具，`/mcp verbose` 看详情 |
| `/skills` | 使用 Skills |
| `/hooks` | 查看和管理生命周期钩子 |
| `/apps`、`/plugins` | 管理应用、浏览插件 |
| `/memories` | 配置记忆的使用和生成 |
| `/agent` | 切换当前查看的子代理线程 |
| `/ps` | 列出后台终端 |
| `/stop` | 停止所有后台终端 |
| `/approve` | 对最近一次自动审查拒绝的操作批准重试一次 |
| `/experimental` | 开关实验性功能 |
| `/import` | 从 Claude Code 导入设置、项目和最近对话 |
| `/ide` | 带上 IDE 里的选区、打开的文件等上下文 |
| `/app` | 在桌面应用中继续当前会话（macOS / Windows） |
| `/keymap` | 重新映射快捷键 |
| `/vim` | 切换输入框的 Vim 模式 |
| `/theme` | 选择代码高亮主题 |
| `/statusline`、`/title` | 配置状态栏、终端标题显示的内容 |
| `/pets` | 选择或隐藏终端宠物 |
| `/feedback` | 把日志发送给维护者 |
| `/logout` | 退出登录 |
| `/quit`、`/exit` | 退出 Codex |
| `/setup-default-sandbox`、`/sandbox-add-read-dir` | Windows 沙箱的设置和授权读取目录 |

::: details 较新版本中的变化
官方仓库最新源码里还新增或调整了：`/worktree`（在新 worktree 中开始或继续对话）、`/recap`（立即总结当前对话）、`/export`（导出为 Markdown）、`/cd` 与 `/pwd`（切换和查看工作目录）、`/voice`（语音）、`/agents`（代理指挥中心）、`/subagents`（切换子代理）、`/warnings`、`/daemon`、`/tui`。升级后以 `/` 菜单为准。
:::

## 快捷键

以下为默认按键（来自官方源码的默认键位表）。可以用 `/keymap` 或 `config.toml` 的 `[tui.keymap]` 修改；输入框为空时按 `?` 显示当前版本的快捷键提示。

### 输入与发送

| 按键 | 作用 |
|---|---|
| `Enter` | 发送 |
| `Shift+Enter`、`Ctrl+J`、`Alt+Enter` | 换行 |
| `Tab` | Codex 正在工作时，把消息排队，等这一轮结束后发送 |
| `Shift+←`、`Alt+↑` | 编辑排队中的消息 |
| `@` | 搜索并插入文件路径 |
| `!命令` | 直接在本地执行 shell 命令，例如 `!git status` |
| `Ctrl+V`、`Alt+V` | 粘贴剪贴板里的图片 |
| `Ctrl+G` | 用外部编辑器（`$VISUAL` / `$EDITOR`）编写长消息 |
| `Ctrl+R` / `Ctrl+S` | 向前 / 向后搜索历史输入 |
| `?` | 显示快捷键提示 |

### 会话控制

| 按键 | 作用 |
|---|---|
| `Esc` | 打断正在进行的这一轮 |
| `Esc` `Esc`（输入框为空时） | 回到之前的消息，编辑后从那里重新开始 |
| `Ctrl+C` | 打断或清空输入；连按两次退出 |
| `Ctrl+D` | 输入框为空时连按两次退出 |
| `Ctrl+T` | 打开 / 关闭完整对话记录（transcript） |
| `Ctrl+L` | 清屏 |
| `Ctrl+O` | 复制 |
| `Alt+,` / `Shift+↓` | 降低推理强度 |
| `Alt+.` / `Shift+↑` | 提高推理强度 |
| `Alt+R` | 切换原始输出模式 |
| `Ctrl+Z` | 挂起到后台（`fg` 回来），不能被重新映射 |

### 审批弹窗

| 按键 | 作用 |
|---|---|
| `y` | 同意这一次 |
| `a` | 本会话内同类请求都同意 |
| `p` | 同意这一类命令前缀（以后同前缀的命令不再询问） |
| `n`、`Esc` | 拒绝 |

审批弹窗里实际出现哪些选项取决于请求类型，以弹窗显示为准。

### 编辑（Emacs 风格）

| 按键 | 作用 |
|---|---|
| `Ctrl+A` / `Ctrl+E` | 行首 / 行尾 |
| `Ctrl+B` / `Ctrl+F` | 左移 / 右移一个字符 |
| `Alt+B` / `Alt+F` | 左移 / 右移一个词 |
| `Ctrl+W`、`Alt+Backspace` | 删除前一个词 |
| `Ctrl+U` / `Ctrl+K` | 删除到行首 / 行尾 |
| `Ctrl+Y` | 粘贴刚删除的内容 |

## 配置键（~/.codex/config.toml）

| 键 | 示例 | 说明 |
|---|---|---|
| `model` | `"gpt-5.5"` | 默认模型 |
| `model_provider` | `"OpenAI"` | 使用哪个 provider，对应 `[model_providers.<id>]` |
| `model_reasoning_effort` | `"medium"` | 推理强度，可选值取决于模型 |
| `review_model` | `"gpt-5.5"` | `/review` 使用的模型 |
| `model_catalog_json` | `"~/.codex/codex-models.json"` | 自定义模型目录文件 |
| `model_context_window` | `400000` | 手动指定上下文窗口（一般不用设） |
| `model_auto_compact_token_limit` | `300000` | 达到多少 token 时自动压缩 |
| `approval_policy` | `"on-request"` | 审批策略 |
| `sandbox_mode` | `"workspace-write"` | 沙箱模式 |
| `[sandbox_workspace_write]` `network_access` | `true` | workspace-write 下是否允许联网，默认 `false` |
| `[sandbox_workspace_write]` `writable_roots` | `["/tmp/data"]` | 额外可写目录（绝对路径） |
| `web_search` | `"live"` | `disabled` / `cached` / `indexed` / `live` |
| `developer_instructions` | `"回答用中文"` | 追加给模型的开发者指令 |
| `project_doc_max_bytes` | `32768` | AGENTS.md 等项目说明最多读取的字节数（默认 32 KiB） |
| `project_doc_fallback_filenames` | `["CLAUDE.md"]` | 没有 AGENTS.md 时依次尝试的文件名 |
| `notify` | `["notify-send", "Codex"]` | 任务完成时调用的通知命令 |
| `history.persistence` | `"none"` | 是否把输入写入 `history.jsonl`（默认 `save-all`） |
| `log_dir` | `"/tmp/codex-log"` | 日志目录；显式设置后才会写纯文本 TUI 日志 |
| `shell_environment_policy.inherit` | `"core"` | 传给命令的环境变量：`all` / `core` / `none` |
| `[features]` | `goals = true` | 功能开关，`codex features list` 查看全部 |
| `[tui]` | `alternate_screen = false` | 界面相关设置 |
| `[tui.keymap]` | | 自定义快捷键 |
| `[projects."/abs/path"]` | `trust_level = "trusted"` | 信任某个项目目录 |

### 自定义 provider

```toml
[model_providers.hivegpt]
name = "HiveGPT"
base_url = "https://hivegpt.cn/v1"
wire_api = "responses"          # 只支持 responses，chat 已移除
env_key = "HIVEGPT_API_KEY"     # 从这个环境变量读 Key
# requires_openai_auth = true   # 改为从 auth.json 读 Key（HiveGPT 弹窗默认用这种）
# request_max_retries = 4
# stream_max_retries = 5
# stream_idle_timeout_ms = 300000
```

### MCP 服务器

```toml
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]
env = { DEBUG = "0" }
startup_timeout_sec = 20
tool_timeout_sec = 60
enabled = true

[mcp_servers.docs]
url = "https://example.com/mcp"
bearer_token_env_var = "DOCS_MCP_TOKEN"
```

## 沙箱 × 审批组合

| 组合 | 怎么开 | Codex 能做什么 | 适合 |
|---|---|---|---|
| 只读 + on-request | `/permissions` → Read Only | 读文件；改文件、联网要你批准 | 读代码、问问题 |
| 可写工作区 + on-request | `/permissions` → Default，或 `-s workspace-write` | 在工作区读写、运行命令；联网和写工作区外要批准 | 日常开发（推荐） |
| 可写工作区 + untrusted | `-s workspace-write -a untrusted` | 只有 `ls`、`cat` 这类可信命令自动运行，其余都问 | 不熟悉的仓库、谨慎模式 |
| 只读 + never | `codex exec`（默认） | 只读，从不询问，被拒的操作直接失败 | 无人值守的分析、审查 |
| 可写工作区 + never | `codex exec -s workspace-write` | 工作区内放手做，从不询问 | CI 中改代码 |
| 完全访问 + never | `/permissions` → Full Access | 任意读写、联网，不询问 | 你完全信任的场景 |
| 跳过一切 | `--yolo` | 没有沙箱也没有审批 | 只在一次性容器 / 虚拟机里 |

无论哪种可写模式，工作区里的 `.git`、`.codex` 目录都保持只读。

## exec 常用参数

| 参数 | 作用 |
|---|---|
| `codex exec "指令"` | 运行一次后退出 |
| `codex exec -` / 管道输入 | 从 stdin 读指令；参数和管道同时给时，stdin 作为附加材料 |
| `-s workspace-write` | 允许写文件（替代已废弃的 `--full-auto`） |
| `-o, --output-last-message <文件>` | 把最后一条回复写到文件 |
| `--json` | stdout 输出 JSONL 事件流 |
| `--output-schema <文件>` | 用 JSON Schema 约束最终回复 |
| `--skip-git-repo-check` | 允许在非 Git 目录运行 |
| `--ephemeral` | 不保存会话 |
| `--ignore-user-config` | 不读用户的 config.toml |
| `-C`、`-m`、`-c`、`-p`、`-i` | 与交互模式相同 |
| `codex exec resume --last "指令"` | 接着最近一次会话继续 |
| `codex exec review` | 非交互审查，支持 `--uncommitted` / `--base` / `--commit` 或自定义指令（四者互斥） |
| 退出码 | 成功 `0`，失败 `1` |

## MCP 命令

| 操作 | 命令 |
|---|---|
| 添加本地（stdio）服务器 | `codex mcp add context7 -- npx -y @upstash/context7-mcp` |
| 添加并设置环境变量 | `codex mcp add mydb --env DB_URL=postgres://... -- node server.js` |
| 添加远程（HTTP）服务器 | `codex mcp add docs --url https://example.com/mcp` |
| 远程服务器用 Bearer Token | `codex mcp add docs --url https://... --bearer-token-env-var DOCS_MCP_TOKEN` |
| 列出 | `codex mcp list`（`--json` 机器可读） |
| 查看某一个 | `codex mcp get context7` |
| 删除 | `codex mcp remove context7` |
| OAuth 登录 / 登出 | `codex mcp login docs` / `codex mcp logout docs` |
| 会话中查看工具 | `/mcp`、`/mcp verbose` |

## 文件位置

`~` 代表用户主目录；设置了环境变量 `CODEX_HOME` 时，下表中的 `~/.codex` 都换成它。

| 路径 | 内容 |
|---|---|
| `~/.codex/config.toml` | 用户配置 |
| `~/.codex/<名字>.config.toml` | `-p <名字>` 选择的配置档 |
| `~/.codex/auth.json` | 登录凭据或 API Key，**不要分享或提交** |
| `~/.codex/AGENTS.md` | 全局个人指令，对所有项目生效 |
| `<项目>/AGENTS.md`、`AGENTS.override.md` | 项目指令，子目录可以再放一份 |
| `<项目>/.codex/config.toml` | 项目级配置，项目被信任时才生效 |
| `~/.codex/sessions/` | 会话记录，`resume` 从这里读取 |
| `~/.codex/history.jsonl` | 输入历史 |
| `~/.codex/rules/` | 命令规则（execpolicy），如 `default.rules` |
| `~/.agents/skills/`、`<项目>/.agents/skills/` | 用户级、项目级 Skills |
| `~/.codex/log/`、`~/.codex/logs_*.sqlite` | 日志，见 [常见问题排查](/codex/troubleshooting) |
| `~/.codex/worktrees/` | 桌面应用托管的 worktree |
| `/etc/codex/config.toml` | 系统级配置（Unix） |

## 小结

- 先记住四个最常用的：`codex`、`codex exec`、`/model`、`/permissions`。
- 换行用 `Shift+Enter` 或 `Ctrl+J`，打断用 `Esc`，回到之前的消息用连按两次 `Esc`，看完整记录用 `Ctrl+T`。
- 日常开发用「可写工作区 + on-request」，CI 用 `codex exec -s workspace-write`，`--yolo` 只留给一次性容器。
- 配置不生效时先跑 `/debug-config` 或 `codex doctor`。

下一步：[常见问题排查](/codex/troubleshooting)
