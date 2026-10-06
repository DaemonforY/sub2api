---
title: config.toml 基础
description: Codex 配置文件在哪、多个来源谁覆盖谁、TOML 怎么写，以及最常用的 20 个配置项和一份带注释的推荐配置。
---

# config.toml 基础

Codex 的 CLI、IDE 扩展和桌面应用共用一份配置：`~/.codex/config.toml`。默认模型、推理强度、权限、通知、界面偏好都写在这里，写一次以后每次启动都生效，不用每次敲参数。

这一页先讲清楚配置文件放在哪、多个配置来源的优先级，再用十分钟过一遍 TOML 语法，然后逐个介绍最常用的配置项，最后给一份可以直接参考的完整配置。键名和默认值都按官方源码核对过。

## 配置文件在哪

| 系统 | 路径 |
|---|---|
| macOS / Linux | `~/.codex/config.toml` |
| Windows | `%USERPROFILE%\.codex\config.toml`，例如 `C:\Users\你的用户名\.codex\config.toml` |

`~/.codex` 这个目录叫 **CODEX_HOME**，除了配置还放着认证文件 `auth.json`、会话记录 `sessions/`、输入历史 `history.jsonl`、日志 `log/` 等。可以用环境变量 `CODEX_HOME` 把它整个挪到别处，见 [高级配置](/codex/config-advanced)。

文件不存在就自己建：

```bash
mkdir -p ~/.codex
touch ~/.codex/config.toml
```

```powershell
New-Item -ItemType Directory -Force "$HOME\.codex"
New-Item -ItemType File -Force "$HOME\.codex\config.toml"
```

如果你按 [接入 HiveGPT](/codex/hivegpt) 配置过，这个文件已经存在，开头是弹窗生成的模型提供方设置。下面要加的配置项接着往后写即可。

## 谁覆盖谁：配置的优先级

同一个配置项可能出现在好几个地方。Codex 会按顺序把它们叠在一起，**后面的覆盖前面的**：

| 顺序 | 来源 | 说明 |
|---|---|---|
| 1 | 内置默认值 | 什么都不写时的行为 |
| 2 | 系统配置 | `/etc/codex/config.toml`（Windows 在 `%ProgramData%\OpenAI\Codex\`），通常由管理员放置 |
| 3 | 用户配置 | `~/.codex/config.toml`，你平时改的就是它 |
| 4 | Profile | 用 `-p 名称` 选中的 `~/.codex/名称.config.toml` |
| 5 | 项目配置 | 项目里的 `.codex/config.toml`，**只在项目被信任时生效** |
| 6 | 运行时 | 命令行的 `-c`、`-m`、`-s` 等参数，以及在界面里用 `/model`、`/permissions` 做的切换 |

一句话记住：**命令行 > 项目 > profile > 用户配置 > 默认值**。

另外有两点容易被忽略：

- 项目配置来自仓库内容，安全起见有一批键不允许在项目里设置，例如 `model_provider`、`model_providers`、`notify`、`profile`、`otel`、`openai_base_url`。这些只能写在用户配置里。
- 企业管理员可以通过 `requirements.toml` 设置**约束**（例如禁止 `danger-full-access`），约束不是覆盖，你的配置违反约束时会被拒绝或回退。

想看某个值最终来自哪一层，在会话里输入 `/debug-config`。

## TOML 十分钟速成

config.toml 用的是 TOML 格式。会下面这些就够了：

```toml
# 井号后面是注释

# 键 = 值；字符串必须加双引号
model = "gpt-5.5"

# 布尔值和数字不加引号
hide_agent_reasoning = false
project_doc_max_bytes = 65536

# 数组用方括号
notify = ["python3", "/Users/me/.codex/notify.py"]

# [表名] 开始一个表，之后的键都属于这个表，直到下一个 [表名]
[tui]
notifications = true
theme = "dracula"

# 嵌套表用点号
[sandbox_workspace_write]
network_access = true

# 表名里有特殊字符（如路径）时，用引号包起来
[projects."/Users/me/work/api"]
trust_level = "trusted"
```

最常见的三个错误：

1. **把顶层键写到了表的下面**。`[tui]` 之后写的 `model = "..."` 会变成 `tui.model`，不会生效。顶层键一律放在文件开头、第一个 `[表]` 之前。
2. **同一个键或同一个表写了两次**，TOML 会直接报错，Codex 启动失败。
3. **Windows 路径的反斜杠**。双引号字符串里 `\` 是转义符，写 `"C:\\work\\api"` 或用单引号的字面量字符串 `'C:\work\api'`。

::: tip 拼错键名不会报错
Codex 默认会忽略不认识的键（启动时可能有一条警告），所以拼错了往往只是「没效果」。怀疑写错时，用 `codex --strict-config` 启动一次，未知键会直接报错并指出位置。
:::

## 最常用的配置项

### 模型相关

```toml
model = "gpt-5.5"                  # 默认模型
model_reasoning_effort = "medium"  # 推理强度：low / medium / high / xhigh，具体取决于模型
model_reasoning_summary = "auto"   # 推理摘要：auto / concise / detailed / none
model_verbosity = "medium"         # 回复详略：low / medium / high
review_model = "gpt-5.5"           # /review 使用的模型，可以和主模型不同
plan_mode_reasoning_effort = "high"  # 计划模式单独的推理强度
```

`model_provider` 指定用哪个模型提供方，默认是内置的 `openai`。接入 HiveGPT 时弹窗会生成一个自定义提供方并把它设为默认，不要手动改它。推理强度怎么选，见 [模型与推理强度](/codex/models)。

### 权限相关

```toml
approval_policy = "on-request"     # 什么时候请示你：on-request（默认）/ never
sandbox_mode = "workspace-write"   # 沙箱：read-only / workspace-write / danger-full-access

[sandbox_workspace_write]
network_access = false             # 沙箱内命令能否联网，默认 false
writable_roots = ["/Users/me/.cache/pnpm"]   # 工作区以外额外可写的目录（绝对路径）
```

不写 `sandbox_mode` 时，默认是 `read-only`；但如果当前项目已经做过信任决定（首次进入项目时 Codex 会问你是否信任），会默认用 `workspace-write`。各种组合的含义和推荐见 [沙箱与审批](/codex/sandbox)。

### 命令的环境变量

Codex 运行 shell 命令时，会把你终端里的环境变量传给子进程。`shell_environment_policy` 控制传哪些：

```toml
[shell_environment_policy]
inherit = "core"                 # all（默认）/ core（只传 HOME、PATH、SHELL、USER 等基础变量）/ none
ignore_default_excludes = false  # 设为 false 才会过滤名字含 KEY、SECRET、TOKEN 的变量
exclude = ["AWS_*", "AZURE_*"]   # 额外排除，支持通配符
set = { NODE_ENV = "development" }   # 强制设置的变量
```

::: warning 默认会把密钥传下去
源码里 `inherit` 默认是 `all`，`ignore_default_excludes` 默认是 `true`，也就是说**默认不过滤**名字带 KEY / SECRET / TOKEN 的变量。如果你的终端里导出了云服务密钥，又不希望模型执行的命令能读到，把 `ignore_default_excludes` 设为 `false`，或者用 `inherit = "core"` 加白名单。
:::

### 网页搜索

```toml
web_search = "cached"   # disabled / cached（默认）/ indexed / live
```

`cached` 使用预先抓取的搜索结果，`live` 实时联网搜索（等同命令行 `--search`）。第三方模型提供方通常不支持内置搜索，用 HiveGPT 时以实际可用为准。

### 通知

任务做完时提醒你，有两种方式。

终端内置通知，最简单：

```toml
[tui]
notifications = true                       # 或者只要部分事件：["agent-turn-complete", "approval-requested"]
notification_method = "auto"               # auto / osc9 / bel
notification_condition = "unfocused"       # unfocused（默认，终端不在前台时才提醒）/ always
```

外部脚本通知，可以发系统通知、推送到手机或企业 IM：

```toml
notify = ["python3", "/Users/me/.codex/notify.py"]
```

Codex 每轮任务结束时会运行这条命令，并把一段 JSON 作为**最后一个参数**传进去，里面有 `type`（`agent-turn-complete`）、`cwd`、`last-assistant-message` 等字段。脚本示例见 [高级配置](/codex/config-advanced)。

### 界面

```toml
hide_agent_reasoning = false   # true 时隐藏推理过程，界面更清爽
file_opener = "vscode"         # 回复里的文件引用用什么打开：vscode（默认）/ vscode-insiders / cursor / windsurf / none

[tui]
theme = "dracula"              # 代码高亮主题，可在会话里用 /theme 挑选后写入
alternate_screen = "auto"      # never 时不占用备用屏幕，保留终端滚动历史
animations = true              # 关掉启动动画和闪烁效果
show_tooltips = true           # 启动时的小提示
vim_mode_default = false       # 输入框默认进入 Vim 模式
```

用 Cursor 的话把 `file_opener` 改成 `"cursor"`，点击回复里的文件路径会直接在 Cursor 中打开对应行。

### 历史与日志

```toml
[history]
persistence = "save-all"   # save-all（默认）/ none：是否把输入历史写入 ~/.codex/history.jsonl
max_bytes = 10485760       # 历史文件上限，超出后丢弃最旧的记录

log_dir = "/Users/me/.codex/log"   # 日志目录，默认 $CODEX_HOME/log
```

在共用电脑上，或者经常在输入框里粘贴敏感内容，可以把 `persistence` 设为 `"none"`。

### 项目说明与上下文

```toml
project_doc_max_bytes = 32768                     # AGENTS.md 等项目说明的总字节上限，默认 32 KiB
project_doc_fallback_filenames = ["CLAUDE.md"]    # 没有 AGENTS.md 时依次尝试的文件名
developer_instructions = "回答一律使用简体中文。"     # 附加给模型的开发者指令，对所有项目生效
model_auto_compact_token_limit = 200000           # 上下文 token 达到多少时自动压缩
```

`project_doc_fallback_filenames` 适合从其他编程助手迁移过来的项目：已有的说明文件不用改名就能被 Codex 读到。AGENTS.md 的写法见 [AGENTS.md](/codex/agents-md)。

### 信任的项目

```toml
[projects."/Users/me/work/api"]
trust_level = "trusted"    # trusted / untrusted
```

第一次在某个目录启动 Codex 时，它会问你是否信任这个项目，你的选择就记在这里。被信任的项目才会加载其中的 `.codex/config.toml`、项目级钩子和规则。

### 其他常用开关

```toml
check_for_update_on_startup = true   # 启动时检查更新，公司统一管理版本时可设为 false
[features]
memories = true                       # 功能开关，名称见 codex features list
```

## 一份带注释的推荐配置

下面是一份适合大多数人日常使用的配置。如果你用 HiveGPT，**保留文件开头弹窗生成的那部分**（`model_provider`、`[model_providers.…]`、`model_catalog_json` 等），把下面的内容合并进去即可，重复的键只留一份。

```toml
# ===== 顶层键：必须放在所有 [表] 之前 =====

# 模型
model = "gpt-5.5"
model_reasoning_effort = "medium"      # 难题用 /model 临时调高
model_reasoning_summary = "auto"
review_model = "gpt-5.5"
plan_mode_reasoning_effort = "high"    # 计划阶段多想一点

# 权限：工作区内放手做，越界再问
approval_policy = "on-request"
sandbox_mode = "workspace-write"

# 体验
file_opener = "vscode"                  # 用 Cursor 改成 "cursor"
hide_agent_reasoning = false
developer_instructions = "回答和代码注释使用简体中文；提交信息使用英文。"

# 任务结束时运行通知脚本（可选，脚本见「高级配置」）
# notify = ["python3", "/Users/me/.codex/notify.py"]

# ===== 表 =====

[sandbox_workspace_write]
network_access = true                  # 允许装依赖、跑需要联网的测试；更谨慎可改为 false
writable_roots = []                    # 需要时加入工作区外的缓存目录

[shell_environment_policy]
inherit = "all"
ignore_default_excludes = false        # 过滤名字含 KEY/SECRET/TOKEN 的环境变量

[tui]
notifications = ["agent-turn-complete", "approval-requested"]
notification_condition = "unfocused"
theme = "dracula"

[history]
persistence = "save-all"
```

改完保存，**重新启动 Codex** 才会读取新配置（`/model`、`/permissions` 这类会话内切换不受影响）。启动后用 `/status` 确认模型和权限是否符合预期。

::: details 配置改坏了怎么办
- 启动直接报 TOML 解析错误：按提示的行号检查引号、方括号是否成对，是否有重复的键或表。
- 某个配置不生效：先 `/debug-config` 看它是不是被项目配置或命令行覆盖了；再用 `codex --strict-config` 检查键名拼写。
- 实在找不到问题：把文件改名备份（`mv ~/.codex/config.toml ~/.codex/config.toml.bak`），从最小配置开始逐段加回来。用 HiveGPT 的话可以重新打开「使用密钥」弹窗生成开头部分。
:::

## 小结

- 配置文件是 `~/.codex/config.toml`，CLI、IDE 扩展、桌面应用共用。
- 优先级：命令行 > 项目 `.codex/config.toml`（需信任）> profile > 用户配置 > 默认值；`/debug-config` 可以查来源。
- 顶层键放在所有 `[表]` 之前；拼错键名默认只会「没效果」，用 `--strict-config` 检查。
- 默认环境变量全部透传给命令，想过滤密钥要设 `ignore_default_excludes = false`。
- 改完重启 Codex，用 `/status` 验证。

下一步：[高级配置](/codex/config-advanced)
