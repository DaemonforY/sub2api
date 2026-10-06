---
title: 高级配置
description: Codex 的 profile 配置文件、多个模型提供方、项目级配置、环境变量、功能开关、通知脚本、状态栏、日志与 OpenTelemetry 等进阶配置。
---

# 高级配置

[config.toml 基础](/codex/config) 解决的是「一份配置用到底」。用久了会遇到更具体的需求：工作项目和个人项目想用不同的模型和权限；同时接 HiveGPT 和另一家模型服务；某个仓库要有自己的设置；任务跑完想收到手机推送；团队要把调用数据接进监控系统。这一页逐个讲这些场景的配置方法。

所有键名都按官方源码核对过；Codex 迭代很快，个别细节以 `codex --help` 和 [官方文档](https://developers.openai.com/codex) 为准。

## Profile：一键切换整套配置

### 新写法：每个 profile 一个文件

Profile 是一组配置的集合，启动时用 `-p 名称` 选用。当前版本里，profile 是 CODEX_HOME 下的**独立文件**：

```text
~/.codex/config.toml          主配置，始终加载
~/.codex/deep.config.toml     profile「deep」
~/.codex/safe.config.toml     profile「safe」
```

Profile 文件只需要写**和主配置不同的部分**，它会叠加在主配置之上：

```toml
# ~/.codex/deep.config.toml：攻坚难题用
model_reasoning_effort = "xhigh"
plan_mode_reasoning_effort = "xhigh"
model_reasoning_summary = "detailed"
```

```toml
# ~/.codex/safe.config.toml：看陌生仓库、只读分析用
sandbox_mode = "read-only"
approval_policy = "on-request"

[shell_environment_policy]
inherit = "core"
```

使用：

```bash
codex -p deep "排查订单服务偶发的超时"
codex -p safe -C ~/Downloads/some-repo "这个项目做什么的？有没有可疑代码？"
codex exec -p deep "分析 benchmark 结果并给出优化建议"
```

### 从旧写法迁移

旧版本把 profile 写在主配置里，用 `profile = "名称"` 选默认 profile：

```toml
# 旧写法，新版本不再支持
profile = "deep"

[profiles.deep]
model_reasoning_effort = "high"
```

新版本遇到 `profile = "…"` 会直接报错拒绝启动；`-p deep` 时如果主配置里还留着 `[profiles.deep]` 表，也会报冲突。迁移只需三步：

1. 把 `[profiles.deep]` 下面的内容剪切到新文件 `~/.codex/deep.config.toml`，去掉表头；
2. 删掉主配置里的 `profile = "deep"` 和空的 `[profiles.deep]`；
3. 以后用 `codex -p deep` 启动。想要「默认就用 deep」，直接把这些值写回主配置。

::: tip shell 别名更顺手
常用的 profile 可以做成别名：`alias cdeep='codex -p deep'`、`alias csafe='codex -p safe'`。
:::

## 多个模型提供方

`model_providers` 表可以定义任意多个 OpenAI 兼容的模型服务，再用 `model_provider` 选默认值，或者用 `-c` / profile 临时切换。HiveGPT 弹窗生成的配置本身就是一个自定义提供方。

```toml
# 默认用 HiveGPT（这部分由「使用密钥」弹窗生成，这里只作示意）
model_provider = "OpenAI"
model = "gpt-5.5"

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://hivegpt.cn/v1"
wire_api = "responses"
requires_openai_auth = true

# 另一个提供方：Key 从环境变量读取
[model_providers.other]
name = "Other Provider"
base_url = "https://api.example.com/v1"
wire_api = "responses"
env_key = "OTHER_API_KEY"            # 从这个环境变量读取 Key
request_max_retries = 4              # HTTP 请求失败重试次数
stream_max_retries = 5               # 流式连接断开后的重连次数
stream_idle_timeout_ms = 300000      # 流式响应多久没动静算超时
```

再配一个 profile 专门切过去：

```toml
# ~/.codex/other.config.toml
model_provider = "other"
model = "provider-model-name"
```

```bash
export OTHER_API_KEY="..."
codex -p other
```

提供方的常用字段：

| 字段 | 作用 |
|---|---|
| `name` | 显示名称 |
| `base_url` | 接口地址，一般以 `/v1` 结尾 |
| `wire_api` | 接口协议，当前只支持 `responses` |
| `env_key` | 存放 Key 的环境变量名 |
| `requires_openai_auth` | 为 true 时使用 `auth.json` / 登录态里的凭据 |
| `http_headers` | 每次请求附带的固定请求头 |
| `env_http_headers` | 请求头的值从环境变量读取，例如 `{ "X-Org" = "MY_ORG_ENV" }` |
| `query_params` | 附加到 URL 上的查询参数 |
| `request_max_retries`、`stream_max_retries`、`stream_idle_timeout_ms` | 重试与超时 |

::: warning 两个硬限制
- **只支持 Responses 协议**。当前源码里 `wire_api` 只有 `responses` 一个值，只提供 Chat Completions 接口的服务不能直接接入。
- **不能覆盖内置 ID**。`openai`、`ollama`、`lmstudio` 等是内置提供方，自定义时换个名字（HiveGPT 配置里用的 `OpenAI` 大小写不同，所以不冲突）。

另外，提供方配置只能写在用户配置里，写在项目的 `.codex/config.toml` 里会被忽略，防止仓库把你的请求和 Key 引到别处。
:::

只是想把内置 `openai` 提供方指向一个代理地址，不必新建提供方，写 `openai_base_url = "https://…/v1"` 即可。接入 DeepSeek 等第三方模型的完整步骤见 [接入 DeepSeek 等其他模型](/codex/providers)。

## 项目级配置

在仓库里放一个 `.codex/config.toml`，就能给这个项目单独设置模型、权限、MCP 服务等，并随代码提交给团队：

```toml
# 仓库根目录/.codex/config.toml
model_reasoning_effort = "high"
sandbox_mode = "workspace-write"

[sandbox_workspace_write]
network_access = true

[mcp_servers.project-docs]
command = "node"
args = ["./tools/docs-mcp.js"]
```

规则要点：

- **只在项目被信任时生效**。Codex 会读取它，但对未信任的项目这一层处于禁用状态。信任记录在用户配置的 `[projects."路径"]` 里。
- Codex 会从当前目录一路向上查找 `.codex/config.toml`，直到项目根（默认以 `.git` 所在目录为根，可用 `project_root_markers` 改），沿途找到的都会加载。
- 项目配置**高于**用户配置和 profile，**低于**命令行参数。
- 安全敏感的键在项目层被禁止：`model_provider`、`model_providers`、`openai_base_url`、`chatgpt_base_url`、`notify`、`profile`、`profiles`、`otel` 等。

同一个 `.codex/` 目录里还可以放项目级的规则、钩子，仓库的 `.agents/skills/` 里放项目 Skills，见 [规则与钩子](/codex/hooks) 和 [Skills](/codex/skills)。

## 环境变量

| 变量 | 作用 |
|---|---|
| `CODEX_HOME` | 配置和数据目录，默认 `~/.codex`。改它就能拥有一套完全独立的配置 |
| `CODEX_SQLITE_HOME` | 本地状态数据库的目录，默认跟随 `CODEX_HOME` |
| `CODEX_API_KEY` | 用 API Key 认证（主要用于 `codex exec` 等自动化场景），优先于已保存的凭据 |
| `CODEX_CA_CERTIFICATE` | 自定义 CA 证书路径，公司网络做 HTTPS 解密时需要；也认 `SSL_CERT_FILE` |
| `RUST_LOG` | 日志级别，例如 `RUST_LOG=debug` |
| 提供方的 `env_key` | 你在 `model_providers` 里指定的 Key 变量 |

`CODEX_HOME` 最实用的用法是隔离：

```bash
# 给某个客户项目一套完全独立的配置、会话记录和凭据
export CODEX_HOME="$HOME/.codex-client-a"
codex
```

```powershell
$env:CODEX_HOME = "$HOME\.codex-client-a"
codex
```

## 功能开关

Codex 的很多新功能先以开关形式发布。查看全部开关和状态：

```bash
codex features list
```

输出三列：功能名、阶段、当前是否启用。阶段的含义：

| 阶段 | 含义 | 建议 |
|---|---|---|
| `stable` | 稳定，通常默认开启 | 放心用 |
| `experimental` | 实验性，可以试 | 遇到问题先关掉它 |
| `under development` | 开发中 | 一般不要开，会有警告 |
| `deprecated` | 即将移除 | 尽早迁移 |
| `removed` | 已移除，开关不再起作用 | 从配置里删掉 |

三种打开方式，按持久程度排列：

```bash
codex --enable memories            # 只对这一次启动生效
codex features enable memories     # 写入 config.toml，长期生效
```

```toml
# 手写在 config.toml 或 profile 文件里
[features]
memories = true
prevent_idle_sleep = true   # 任务运行时防止电脑休眠（实验性）
```

在会话里也可以用 `/experimental` 交互式地切换实验功能。开了开发中的功能又嫌警告烦，可以设 `suppress_unstable_features_warning = true`。

## 通知脚本

`notify` 会在每轮任务结束时运行你指定的程序，并把一段 JSON 作为最后一个参数传入。JSON 大致是这样：

```json
{
  "type": "agent-turn-complete",
  "thread-id": "b5f6c1c2-...",
  "turn-id": "12345",
  "cwd": "/Users/me/work/api",
  "client": "codex-tui",
  "input-messages": ["给用户列表加分页"],
  "last-assistant-message": "分页已完成，测试全部通过。"
}
```

一个跨平台的 Python 示例，macOS 弹系统通知，Linux 用 `notify-send`，同时可选推送到 webhook：

```python
#!/usr/bin/env python3
# ~/.codex/notify.py
import json, os, platform, subprocess, sys, urllib.request

def main() -> None:
    event = json.loads(sys.argv[-1])
    if event.get("type") != "agent-turn-complete":
        return
    project = os.path.basename(event.get("cwd", "")) or "Codex"
    message = (event.get("last-assistant-message") or "任务完成")[:120]

    system = platform.system()
    if system == "Darwin":
        script = f'display notification {json.dumps(message)} with title {json.dumps("Codex · " + project)}'
        subprocess.run(["osascript", "-e", script], check=False)
    elif system == "Linux":
        subprocess.run(["notify-send", f"Codex · {project}", message], check=False)

    webhook = os.environ.get("CODEX_NOTIFY_WEBHOOK")
    if webhook:
        body = json.dumps({"text": f"[{project}] {message}"}).encode()
        req = urllib.request.Request(webhook, body, {"Content-Type": "application/json"})
        urllib.request.urlopen(req, timeout=5)

if __name__ == "__main__":
    main()
```

```toml
# config.toml 顶层
notify = ["python3", "/Users/me/.codex/notify.py"]
```

`notify` 和 `[tui] notifications` 的区别：前者是外部程序，CLI、`exec` 都会触发，适合推送到别的设备；后者是终端自带的桌面通知，只在交互界面里有，零配置。两者可以同时开。出于安全考虑，`notify` 只能写在用户配置里，项目配置里写了不生效。

## 界面定制

### 状态栏和终端标题

```toml
[tui]
status_line = ["model-with-reasoning", "git-branch", "context-remaining", "current-dir"]
terminal_title = ["project-name", "run-state"]
```

不设置时状态栏默认显示模型与推理强度、当前目录、会话名。可用的项目有 `model`、`model-with-reasoning`、`current-dir`、`project-name`、`git-branch`、`branch-changes`、`run-state`、`permissions`、`approval-mode`、`context-remaining`、`context-used`、`used-tokens`、`codex-version`、`thread-id` 等。更省事的办法是在会话里用 `/statusline`、`/title` 勾选，结果会自动写回配置。

### 快捷键、主题和其他

```toml
[tui]
theme = "github-dark"          # 也可以用 /theme 预览后选择
vim_mode_default = true        # 输入框默认 Vim 模式
alternate_screen = "never"     # 不占用备用屏幕，tmux 用户常用
mouse_scroll_speed = 2.0       # 鼠标滚轮速度倍数
animations = false             # 关闭动画，远程 SSH 时更流畅
auto_recap = true              # 终端失去焦点时自动生成进度回顾
```

快捷键可以在 `[tui.keymap]` 里改，但结构较复杂，建议用 `/keymap` 交互式修改。

## MCP 服务配置

MCP 服务写在 `[mcp_servers.名称]` 表里，可以放在用户配置，也可以放在项目配置。最简单的添加方式是 `codex mcp add`，它会替你写好配置。字段、OAuth 登录、工具白名单等详见 [MCP](/codex/mcp)。

## 信任的项目

```toml
[projects."/Users/me/work/api"]
trust_level = "trusted"

[projects."/Users/me/Downloads/unknown-repo"]
trust_level = "untrusted"
```

信任决定影响三件事：项目级 `.codex/config.toml` 和钩子、规则是否加载；没写 `sandbox_mode` 时的默认沙箱；没写 `approval_policy` 时的默认审批方式（未信任项目会更保守，除了少数只读命令，其余都要先问你）。

键是项目的绝对路径，Git 仓库里会按仓库根目录匹配。想撤销信任，删掉对应的表或改成 `untrusted`。

## 日志与可观测性

### 本地日志

日志默认写在 `~/.codex/log/`，交互界面的日志文件是 `codex-tui.log`。调试问题时提高日志级别：

```bash
RUST_LOG=debug codex
tail -f ~/.codex/log/codex-tui.log
```

想换目录用 `log_dir = "/path/to/logs"`。

### OpenTelemetry

团队想统计调用量、耗时、工具使用情况，可以把 Codex 的事件导出到 OTLP 兼容的后端（如 Grafana、Jaeger、Datadog 的 OTLP 入口）：

```toml
[otel]
environment = "dev"             # 标记环境：dev / staging / prod
log_user_prompt = false         # 是否在导出数据里包含用户提示词，默认不包含

[otel.exporter.otlp-http]
endpoint = "https://otel.example.com/v1/logs"
protocol = "binary"             # binary 或 json
headers = { "x-api-key" = "..." }
```

也可以用 gRPC：

```toml
[otel.exporter.otlp-grpc]
endpoint = "https://otel.example.com:4317"
```

`trace_exporter`、`metrics_exporter` 用同样的写法分别配置链路和指标导出。提示词、模型回复可能包含敏感信息，`log_user_prompt`、`log_agent_responses` 默认都是关闭的，打开前想清楚数据会流向哪里。`otel` 只能写在用户配置或管理员配置里。

### 关闭分析与反馈

```toml
[analytics]
enabled = false   # 关闭本机的使用分析

[feedback]
enabled = false   # 关闭 /feedback 反馈收集
```

## 少用但值得知道的键

| 键 | 作用 |
|---|---|
| `developer_instructions` | 给模型的附加指令，所有项目通用，适合写语言偏好、提交规范 |
| `model_instructions_file` | 用文件**替换**内置的系统指令。官方强烈不建议，容易让 Codex 变笨 |
| `model_context_window` | 手动指定上下文窗口大小，第三方模型识别不出时使用 |
| `model_auto_compact_token_limit` | 上下文达到多少 token 自动压缩 |
| `tool_output_token_limit` | 单个工具输出保留进上下文的 token 上限 |
| `background_terminal_max_timeout` | 后台终端输出的最长轮询时间，默认 5 分钟 |
| `project_root_markers` | 判定项目根目录的标记文件，默认 `[".git"]` |
| `cli_auth_credentials_store` | 凭据存储位置：`file`（默认，即 auth.json）、`keyring`（系统钥匙串）或 `auto`（有钥匙串就用） |

## 小结

- Profile 现在是独立文件 `~/.codex/名称.config.toml`，用 `-p 名称` 叠加；旧的 `profile =` / `[profiles.x]` 需要迁移。
- 多个模型提供方写在 `model_providers`，只支持 `responses` 协议，提供方设置只能放在用户配置。
- 项目级 `.codex/config.toml` 仅在信任后生效，优先级高于用户配置，但敏感键被禁止。
- `CODEX_HOME` 可以隔离出一整套配置；`codex features list` 管理功能开关。
- `notify` 适合推送到外部，`[tui] notifications` 适合桌面提醒；团队监控用 `[otel]`。

下一步：[沙箱与审批](/codex/sandbox)
