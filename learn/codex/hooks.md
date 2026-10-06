---
title: 规则与钩子
description: 用 execpolicy 规则决定哪些命令直接放行、需要确认或一律禁止；用 hooks 在工具调用前后、会话开始和结束时运行自己的脚本；用 notify 接收完成通知。附可直接使用的示例和选择表。
---

# 规则与钩子

AGENTS.md 和提示词是「请求」：你告诉模型应该怎么做，它大概率会照做，但没有强制力。有些事情不能只靠模型自觉，比如「绝不允许执行 `rm -rf`」「每次改完文件都要格式化」「任务完成后发个通知」。这类需求需要由 Codex 程序本身来执行，而不是交给模型判断。

Codex 提供了三种机制：

- **规则（execpolicy rules）**：按命令前缀匹配 shell 命令，决定放行、询问还是禁止。
- **钩子（hooks）**：在特定事件发生时运行你的脚本，脚本可以阻止操作、补充上下文或做收尾工作。
- **notify**：每轮任务结束时调用一个外部程序，常用来发桌面或聊天通知。

本页内容依据官方源码（`codex-rs/execpolicy/`、`codex-rs/hooks/`、`codex-rs/core/src/exec_policy.rs`）整理。

::: tip 代码风格规范不属于这里
「用 4 空格缩进」「公共函数写注释」这类写给模型的规范，应该写进 [AGENTS.md](/codex/agents-md)。这一页的规则和钩子是由程序强制执行的机制，不是给模型看的文字说明。
:::

## 规则：控制哪些命令能执行

### 文件位置和格式

规则文件以 `.rules` 结尾，放在配置目录的 `rules/` 子目录里：

- 用户级：`~/.codex/rules/*.rules`
- 项目级：项目的 `.codex/rules/*.rules`（需要信任该项目）

文件使用 Starlark 语法（类似 Python），核心是 `prefix_rule`：

```python
# ~/.codex/rules/team.rules

# 推送和重置需要人工确认
prefix_rule(
    pattern = ["git", ["push", "reset"]],
    decision = "prompt",
    justification = "推送和重置会影响远端或丢失改动，需要人工确认",
    match = ["git push origin main", "git reset --hard HEAD~1"],
    not_match = ["git status", "git pushx"],
)

# 禁止递归强制删除，并给出替代做法
prefix_rule(
    pattern = ["rm", ["-rf", "-fr"]],
    decision = "forbidden",
    justification = "不要使用 rm -rf；需要清理时先用 git clean -n 预览，再请用户确认",
)

# 常用的只读或测试命令直接放行
prefix_rule(pattern = ["pnpm", ["test", "lint", "typecheck"]])
prefix_rule(pattern = ["go", ["test", "vet", "build"]])
```

| 参数 | 说明 |
|---|---|
| `pattern` | 按顺序匹配的命令前缀。某个位置写成列表表示「其中任意一个」 |
| `decision` | `allow`（放行，默认值）、`prompt`（询问）、`forbidden`（禁止） |
| `justification` | 规则的原因，会出现在确认提示或拒绝信息里；禁止时建议写上替代做法 |
| `match` / `not_match` | 示例命令，加载时会校验，相当于规则的单元测试，写错了会直接报错 |

匹配是按「词」进行的前缀匹配：`["git", "push"]` 能匹配 `git push origin main`，但不能匹配 `git pushx`，也不能匹配参数顺序不同的写法。多条规则同时命中时，取最严格的结果：`forbidden` 高于 `prompt`，`prompt` 高于 `allow`。

### 先测试再生效

`codex execpolicy check` 可以在不启动会话的情况下检查某条命令会命中什么：

```bash
codex execpolicy check --rules ~/.codex/rules/team.rules git push origin main
```

```json
{"matchedRules":[{"prefixRuleMatch":{"matchedPrefix":["git","push"],"decision":"prompt","justification":"推送和重置会影响远端或丢失改动，需要人工确认"}}],"decision":"prompt"}
```

没有规则命中时输出 `{"matchedRules":[]}`，这时按你的审批策略和 Codex 内置的安全判断处理。可以传多个 `--rules`，加 `--pretty` 格式化输出。

::: warning allow 不只是「免确认」
命令的每一段都被 `allow` 规则明确放行时，Codex 不仅跳过确认，还会让它**在沙箱之外运行**。所以 `allow` 只给你完全信任的命令，比如测试、构建、lint。不要写 `prefix_rule(pattern = ["bash"])`、`["python"]`、`["npx"]` 这类能执行任意代码的前缀。
:::

### 审批时「始终允许」会写进规则

在交互会话里，Codex 请求执行某条命令时，审批选项中有「以后同类命令不再询问」一类的选项。选择它，Codex 会把对应的 `prefix_rule` 追加到 `~/.codex/rules/default.rules`。时间久了，这个文件可能积累不少放行规则，建议定期打开检查，删掉不再需要的条目。

## 钩子：在关键时刻运行你的脚本

钩子（功能开关 `hooks`，当前版本默认开启；旧名 `codex_hooks` 仍被识别）让你在 Codex 生命周期的特定事件上挂脚本。它的设计和其他编程代理的 hooks 接近，事件名和输入输出格式基本一致，迁移现有脚本比较方便。

### 支持的事件

| 事件 | 触发时机 | 支持 matcher | 典型用途 |
|---|---|---|---|
| `SessionStart` | 会话开始或恢复 | 是 | 注入当前分支、待办等上下文 |
| `UserPromptSubmit` | 用户提交提示词后、发送给模型前 | 否 | 拦截含敏感信息的输入、补充上下文 |
| `PreToolUse` | 工具执行前 | 是 | 阻止危险命令、改写参数 |
| `PermissionRequest` | 需要用户审批时 | 是 | 自动批准或拒绝特定请求 |
| `PostToolUse` | 工具执行后 | 是 | 自动格式化、记录日志、给模型反馈 |
| `PreCompact` / `PostCompact` | 上下文压缩前后 | 是 | 保存或恢复关键信息 |
| `SubagentStart` / `SubagentStop` | 子代理开始和结束 | 是 | 记录子代理活动 |
| `Stop` | 一轮任务即将结束 | 否 | 检查是否真的完成，没完成就让它继续 |
| `Interrupt` | 用户中断任务 | 否 | 清理临时资源 |
| `SessionEnd` | 会话结束 | 是 | 收尾、汇总 |

### 配置位置

钩子可以写在 JSON 文件里，也可以写在 `config.toml` 里，效果相同：

- 用户级：`~/.codex/hooks.json`，或 `~/.codex/config.toml` 的 `[hooks]`
- 项目级：项目的 `.codex/hooks.json` 或 `.codex/config.toml`（需要信任项目）
- 插件和管理员下发的钩子

同一层里不要两种格式混用，Codex 会给出警告。JSON 写法：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.codex/hooks/guard_bash.py",
            "timeout": 10,
            "statusMessage": "检查命令安全性"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "apply_patch",
        "hooks": [
          { "type": "command", "command": "python3 ~/.codex/hooks/format_changed.py", "timeout": 60 }
        ]
      }
    ]
  }
}
```

等价的 TOML 写法：

```toml
[[hooks.PreToolUse]]
matcher = "Bash"

[[hooks.PreToolUse.hooks]]
type = "command"
command = "python3 ~/.codex/hooks/guard_bash.py"
timeout = 10
statusMessage = "检查命令安全性"
```

处理器（`hooks` 数组里的每一项）的字段：

| 字段 | 说明 |
|---|---|
| `type` | 通常是 `command`；也支持 `mcp_tool`（调用某个 MCP 服务器的工具，配 `server`、`tool`、`input`） |
| `command` | 要执行的命令，通过你的登录 shell 运行，工作目录是会话目录 |
| `commandWindows` | Windows 上改用的命令（可选） |
| `timeout` | 超时秒数。默认 600 秒；`SessionEnd` 和 `Interrupt` 默认 1 秒、最多 3 秒 |
| `async` | 为 true 时后台运行，不阻塞 Codex，也就不能用来拦截操作 |
| `statusMessage` | 运行时在界面上显示的提示文字 |

### matcher 怎么写

- 不写、写空字符串或 `*`：匹配全部。
- 只含字母、数字、下划线和 `|`：精确匹配，`|` 表示「或」，如 `Bash|apply_patch`。
- 其他写法按正则表达式处理，如 `mcp__github__.*`。

对工具类事件，matcher 匹配的是工具名：

| 工具 | 名称 |
|---|---|
| shell 命令 | `Bash` |
| 文件编辑 | `apply_patch`（也接受 `Write`、`Edit` 作为别名） |
| MCP 工具 | `mcp__服务器名__工具名` |
| 派生子代理 | `spawn_agent`（也接受 `Agent`） |

### 输入和输出

钩子脚本从**标准输入**读到一个 JSON 对象。以 `PreToolUse` 为例，主要字段有：`session_id`、`turn_id`、`cwd`、`hook_event_name`、`model`、`permission_mode`、`transcript_path`、`tool_name`、`tool_use_id` 和 `tool_input`。对 shell 命令，`tool_input` 是 `{"command": "..."}`；对文件编辑，`tool_input.command` 是补丁全文；对 MCP 工具，是该工具的参数。`PostToolUse` 额外带有 `tool_response`，`Stop` 带有 `last_assistant_message` 和 `stop_hook_active`。

脚本通过**退出码**和**标准输出**告诉 Codex 结果：

| 方式 | 效果 |
|---|---|
| 退出码 0，没有输出 | 正常放行 |
| 退出码 0，输出 JSON | 按 JSON 内容处理（见下） |
| 退出码 2，stderr 写原因 | 阻止本次操作，原因反馈给模型（适用于可阻止的事件，如 `PreToolUse`） |
| 其他退出码、超时、输出非法 JSON | 记为钩子失败，**操作照常继续** |

最后一行很重要：钩子出错时是「放行」而不是「拦截」。安全相关的钩子要写得足够健壮，不要依赖它出错时自动兜底，它只是一道额外的防线，不能替代沙箱和审批。

`PreToolUse` 常用的 JSON 输出：

```json
{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": "禁止直接操作生产数据库，请改用只读副本"}}
```

- `permissionDecision: "deny"` 加原因：阻止执行。
- `permissionDecision: "allow"` 必须同时带 `updatedInput`，用于改写工具参数后再执行；只写 allow 不带改写会被视为无效输出。
- `"ask"` 目前不被支持，会被当作无效输出、照常放行。
- `additionalContext`：给模型补充一段说明。

`Stop` 和 `PostToolUse` 可以输出 `{"decision": "block", "reason": "..."}`：对 `Stop`，这会让 Codex 不结束本轮，而是把 reason 当作新的指示继续工作；对 `PostToolUse`，reason 会作为反馈交给模型。

### 信任：新钩子要先审查

钩子会在你的机器上执行任意命令，所以 Codex 对它们有一道信任检查：用户级和项目级的钩子，**需要你审查并确认信任后才会运行**。钩子内容一旦被修改（哪怕只改了超时），就会变成「已修改」状态，需要重新确认。

- 启动时如果发现未信任的钩子，Codex 会提示你审查。
- 会话中输入 `/hooks`，可以查看全部钩子、来源、状态，进行信任、启用或停用。
- 信任状态记录在 `config.toml` 的 `[hooks.state]` 里。
- 自动化环境里，如果钩子来源已经另行审核，可以用 `--dangerously-bypass-hook-trust` 跳过本次运行的信任检查。正如名字所示，这有风险，不要在日常使用中开启。

这也意味着：克隆一个别人的仓库，它的 `.codex/hooks.json` 不会悄悄在你机器上执行。

## 示例一：拦截危险命令

```python
#!/usr/bin/env python3
# ~/.codex/hooks/guard_bash.py —— PreToolUse，matcher = "Bash"
import json, re, sys

data = json.load(sys.stdin)
cmd = data.get("tool_input", {}).get("command", "")
if isinstance(cmd, list):          # 兼容命令以数组形式传入的情况
    cmd = " ".join(cmd)

BLOCKED = [
    (r"\brm\s+-[a-z]*r[a-z]*f|\brm\s+-[a-z]*f[a-z]*r", "禁止 rm -rf，请先用 git clean -n 预览要删除的文件"),
    (r"\bgit\s+push\b.*--force(?!-with-lease)", "禁止强制推送；确有需要请用 --force-with-lease 并先征得用户同意"),
    (r"\bDROP\s+(TABLE|DATABASE)\b", "禁止在命令行执行 DROP，请写迁移文件"),
    (r"\bcurl\b[^|]*\|\s*(ba|z)?sh\b", "禁止下载脚本直接执行"),
]

for pattern, reason in BLOCKED:
    if re.search(pattern, cmd, re.IGNORECASE):
        print(reason, file=sys.stderr)
        sys.exit(2)                 # 退出码 2 + stderr = 阻止

sys.exit(0)
```

这个钩子和上面的 `forbidden` 规则作用相近。区别在于规则只能做前缀匹配，钩子可以用任意逻辑判断（正则、读取配置、检查当前分支等）。两者可以同时使用。

## 示例二：改完文件自动格式化

`apply_patch` 的 `tool_input.command` 是补丁文本，其中 `*** Add File: 路径` 和 `*** Update File: 路径` 两种行标明了被新增和修改的文件。脚本提取这些路径，按扩展名调用格式化工具：

```python
#!/usr/bin/env python3
# ~/.codex/hooks/format_changed.py —— PostToolUse，matcher = "apply_patch"
import json, os, re, subprocess, sys

data = json.load(sys.stdin)
patch = data.get("tool_input", {}).get("command", "")
cwd = data.get("cwd", ".")

files = re.findall(r"^\*\*\* (?:Add|Update) File: (.+)$", patch, re.MULTILINE)
for f in files:
    path = os.path.join(cwd, f.strip())
    if not os.path.exists(path):
        continue
    if path.endswith(".go"):
        subprocess.run(["gofmt", "-w", path])
    elif path.endswith((".ts", ".vue", ".js", ".css", ".md")):
        subprocess.run(["npx", "--no-install", "prettier", "--write", path], cwd=cwd)
    elif path.endswith(".py"):
        subprocess.run(["ruff", "format", path])

sys.exit(0)
```

格式化工具不存在时，对应的 `subprocess.run` 会抛异常，钩子记为失败，但不影响 Codex 继续工作。可以在脚本里自己判断工具是否存在，避免界面上出现失败提示。

## 示例三：会话开始时注入上下文

```bash
#!/usr/bin/env bash
# ~/.codex/hooks/session_context.sh —— SessionStart
branch=$(git branch --show-current 2>/dev/null || echo "非 Git 目录")
dirty=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
ctx="当前分支：${branch}；未提交的文件数：${dirty}。"
printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$ctx"
```

这样每次开会话，模型一开始就知道当前分支和工作区状态，不需要自己再跑一遍 `git status`。

## notify：任务完成时通知你

长任务交给 Codex 后你可能去做别的事，希望它完成时提醒一下。

最简单的是 CLI 自带的桌面通知，在 `config.toml` 里设置：

```toml
[tui]
notifications = true                   # 默认开启
notification_condition = "unfocused"   # 只在终端不在前台时通知；always 表示总是通知
```

需要自定义（比如推送到企业微信、飞书或 Slack），用顶层的 `notify` 配置一个外部程序：

```toml
notify = ["python3", "/Users/me/.codex/notify.py"]
```

每轮任务结束时，Codex 会调用这个程序，并把一个 JSON 字符串作为**最后一个命令行参数**传入，`type` 为 `agent-turn-complete`，还包括 `thread-id`、`turn-id`、`cwd`、`input-messages`（本轮你的输入）和 `last-assistant-message`（Codex 最后的回复）。

```python
#!/usr/bin/env python3
# ~/.codex/notify.py
import json, subprocess, sys

event = json.loads(sys.argv[-1])
if event.get("type") != "agent-turn-complete":
    sys.exit(0)
summary = (event.get("last-assistant-message") or "任务完成")[:80]
# macOS 桌面通知；换成 requests.post(...) 即可推送到聊天工具的 webhook
subprocess.run(["osascript", "-e", f'display notification {json.dumps(summary)} with title "Codex"'])
```

`notify` 是较早的机制，源码中已标注为兼容保留。新的需求优先考虑 `Stop` 钩子，它能拿到更完整的信息，也能影响 Codex 的行为。

## 怎么选

| 需求 | 用什么 | 原因 |
|---|---|---|
| 禁止或必须确认某些命令 | 规则 | 声明式、能用 `execpolicy check` 测试、和审批流程集成 |
| 让常用测试和构建命令免确认 | 规则（`allow`） | 注意 allow 会让命令在沙箱外运行 |
| 按复杂条件拦截操作（正则、分支、时间） | `PreToolUse` 钩子 | 规则只能做前缀匹配 |
| 改完文件后格式化、跑 lint | `PostToolUse` 钩子 | 在每次编辑后自动执行 |
| 给模型补充动态上下文 | `SessionStart` / `UserPromptSubmit` 钩子 | 用 `additionalContext` 注入 |
| 检查任务是否真的完成 | `Stop` 钩子 | `decision: block` 能让它继续工作 |
| 任务完成提醒 | `[tui] notifications` 或 `notify` | 简单直接 |
| 写给模型的编码规范 | AGENTS.md | 不需要程序强制 |

## 小结

- 规则写在 `rules/*.rules`，用 `prefix_rule` 按命令前缀决定 allow、prompt 或 forbidden，多条命中取最严格；用 `codex execpolicy check` 先测试。
- `allow` 规则会让命令跳过确认并在沙箱外运行，只给完全可信的命令使用。
- 钩子写在 `hooks.json` 或 `config.toml` 的 `[hooks]`，支持 `PreToolUse`、`PostToolUse`、`Stop` 等事件；shell 工具名是 `Bash`，编辑工具是 `apply_patch`。
- 钩子用退出码 2 或 `permissionDecision: "deny"` 阻止操作；出错时操作照常继续，所以它不能替代沙箱；新钩子要在 `/hooks` 里审查信任后才会运行。
- 完成通知用 `[tui] notifications` 或 `notify`，复杂场景优先用 `Stop` 钩子。

下一步：了解这些机制和沙箱、审批如何配合，见 [沙箱与审批](/codex/sandbox)；团队统一管理见 [安全与团队管理](/codex/security)。
