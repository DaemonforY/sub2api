---
title: 安全与团队管理
description: 用威胁模型看清 Codex 的风险，按沙箱、审批、命令规则、网络、敏感文件和环境变量逐层设防，知道会话日志在哪里，并用 requirements.toml 和统一的 AGENTS.md 在团队里落地，最后附一份检查清单。
---

# 安全与团队管理

Codex 能读你的代码、改文件、执行命令，这正是它好用的原因，也是风险所在。一个人用的时候，靠默认设置加上认真看审批弹窗，基本够了；放到团队里，几十个人、几十个仓库、各种权限习惯，就需要统一的规则。

这一页先讲清楚风险从哪里来，再按层介绍防护手段，最后讲团队如何用托管配置统一策略，并给出一份可以直接照着做的检查清单。沙箱和审批的基本用法见 [沙箱与审批](/codex/sandbox)，这里侧重组合和落地。

## 威胁模型：到底在防什么

| 风险 | 怎么发生 | 后果 |
|---|---|---|
| 提示注入 | 代码注释、README、依赖包文档、网页、issue 内容里藏着给 AI 的指令 | 代理被诱导执行危险命令、外发数据 |
| 密钥泄露 | `.env`、配置文件、环境变量里的密钥被读进上下文，或被命令打印出来 | 密钥随对话发给模型服务商，或被写进日志、提交到仓库 |
| 误操作 | 模型理解错需求，或命令写错（`rm -rf` 路径错、`git push --force`） | 文件被删、历史被覆盖、生产数据被改 |
| 供应链 | 代理自己 `npm install` 了一个拼写相近的恶意包 | 恶意代码在你的机器或 CI 上运行 |
| 越权 | 代理有了不该有的权限：完全访问、生产环境凭据、可写的云账号 | 小错误被放大成事故 |

这些风险的共同点是：**你不能指望模型自己不犯错，也不能指望自己每次都看清审批弹窗**。所以防护要分层，任何一层失效时，下一层还能兜住。

## 第一层：沙箱

沙箱从操作系统层面限制 Codex 执行的命令能做什么，模型说什么都绕不过去。

| 模式 | 读 | 写 | 网络 | 用途 |
|---|---|---|---|---|
| `read-only` | 可以 | 不可以 | 不可以 | 审查、分析、回答问题 |
| `workspace-write` | 可以 | 只能写工作区（和临时目录） | 默认不可以 | 日常开发，推荐默认 |
| `danger-full-access` | 可以 | 任何位置 | 可以 | 只在一次性的隔离环境里用 |

`workspace-write` 下有几处细节值得知道：

- 工作区里的 `.git`、`.codex`、`.agents` 目录默认是只读的，代理不能直接改 Git 元数据和自己的配置。
- 需要额外可写目录时用 `writable_roots` 或启动参数 `--add-dir`，不要因此切到完全访问。
- 联网默认关闭，确实需要时单独打开：

```toml
sandbox_mode = "workspace-write"

[sandbox_workspace_write]
network_access = false            # 需要时再改成 true
writable_roots = ["/Users/me/.cache/my-tool"]
```

`--dangerously-bypass-approvals-and-sandbox` 会同时关掉沙箱和审批，名字已经说明了一切：只在本身已经隔离的容器或虚拟机里用。

## 第二层：审批策略

沙箱挡住的操作，可以通过审批让你决定是否放行。`approval_policy` 的取值：

| 取值 | 行为 |
|---|---|
| `untrusted` | 只有已知安全的只读命令（如 `ls`、`cat`）自动执行，其他都要批准 |
| `on-request` | 默认值。在沙箱内正常工作，需要越出沙箱（联网、写工作区外）时请求批准 |
| `never` | 从不询问，超出沙箱的操作直接失败并告诉模型；适合非交互脚本 |
| `granular` | 细粒度：分别决定沙箱越权、规则触发、Skill 脚本、MCP 请求等各类审批是弹给你还是直接拒绝 |

`on-failure` 已经废弃，交互使用请改用 `on-request`，非交互用 `never`。

批准时的几条纪律：

- **看清完整命令**再点，尤其是管道、重定向、`curl ... | sh` 这类组合。
- 「本次会话总是允许」只给明确无害的命令，比如 `pnpm test`；不要给 `git push`、`rm`、`docker`。
- 如果你发现自己在不看内容地连续点允许，说明审批太频繁了。应该用下一层的命令规则把安全命令放行，而不是切到完全访问。

审批还可以交给一个自动审查的子代理来处理（`approvals_reviewer = "auto_review"`）：它会收集上下文、按风险判断批准或拒绝，你只需要处理它拿不准的情况。在 CLI 里用 `/approve` 可以对它最近拒绝的一次操作放行一次。

## 第三层：命令规则

命令规则（rules）按命令前缀决定「直接允许 / 询问 / 禁止」，比审批更精确。规则文件放在 `~/.codex/rules/` 下，扩展名 `.rules`，用 Starlark 语法：

```python
# ~/.codex/rules/default.rules
prefix_rule(
    pattern = ["pnpm", "test"],
    decision = "allow",
    justification = "跑测试是只读且频繁的操作",
)

prefix_rule(
    pattern = ["git", "push"],
    decision = "prompt",
    justification = "推送前必须由人确认",
)

prefix_rule(
    pattern = ["rm", "-rf"],
    decision = "forbidden",
    justification = "禁止递归强删，需要清理时让人来做",
)
```

`forbidden` 是硬性禁止，即使你在弹窗里想批准也不行。项目级的规则可以放在仓库的 `.codex/rules/` 下，和代码一起评审。

## 第四层：网络

网络是数据外发和下载恶意代码的主要通道，原则是**默认关，按需开，开也要窄**：

- 本地：`workspace-write` 默认不联网；需要装依赖时，让 Codex 在审批里请求，或你自己在终端里装好再让它继续。
- 网页搜索：`web_search` 可选 `disabled`、`cached`、`indexed`、`live`。`live` 会实时访问网页，提示注入风险最高；团队可以限制只允许 `disabled` 或 `cached`。
- 云端任务：代理阶段默认不联网，开启时用域名白名单并限制 HTTP 方法，见 [云端任务与 GitHub](/codex/cloud)。
- 更细的控制（按域名放行或拒绝、经过本地代理）可以通过权限配置里的 `network` 段实现，属于进阶用法，以官方文档为准。

## 第五层：敏感文件和环境变量

### 不让敏感文件进上下文

只要代理读了一个文件，内容就会作为上下文发给模型服务商。几条做法：

1. **密钥不进仓库**：`.env`、`*.pem`、`credentials.json` 加进 `.gitignore`，仓库里只放 `.env.example`。
2. **在 AGENTS.md 里写明**（见 [AGENTS.md](/codex/agents-md)）：

```markdown
## 安全
- 不要读取、打印或修改 .env、*.pem、secrets/ 下的任何文件
- 需要配置项时，参考 .env.example 中的变量名，不要猜测真实值
- 不要在命令输出中回显任何环境变量的值
```

3. **用权限配置硬性禁止读取**。AGENTS.md 只是「请求」，新版本的权限配置可以从沙箱层面禁止读取匹配的文件：

```toml
default_permissions = "safe-dev"

[permissions.safe-dev]
extends = ":workspace"            # 在内置的工作区权限基础上修改

[permissions.safe-dev.filesystem.":workspace_roots"]
"**/*.env" = "deny"
"secrets" = "deny"
```

内置的权限档位有 `:read-only`、`:workspace`、`:danger-full-access`。权限配置是较新的功能，部分平台和沙箱实现对「禁止读取」的支持有限（无法强制执行时会直接拒绝运行，而不是悄悄放宽），以你所用版本的官方文档为准。

### 环境变量默认会传给命令

这一点和很多人的直觉相反：**Codex 默认把你终端里的全部环境变量传给它执行的命令，包括名字里带 KEY、SECRET、TOKEN 的**。代理随手跑一个 `env` 或 `printenv`，你的云账号密钥就进了上下文。

建议在 `config.toml` 里收紧：

```toml
[shell_environment_policy]
inherit = "core"                  # 只继承 PATH、HOME 等基础变量；可选 all / core / none
ignore_default_excludes = false   # 启用默认过滤：去掉名字含 KEY、SECRET、TOKEN 的变量
exclude = ["AWS_*", "AZURE_*", "DATABASE_URL"]
set = { NODE_ENV = "development" }
```

改完后在 Codex 里让它运行 `env | sort`（Windows 上用 `Get-ChildItem Env:`）检查一下还剩哪些变量。

另外，不要在启动 Codex 的终端里 `export` 生产环境凭据。需要访问测试环境时，用权限最小的专用账号。

## 审计：会话日志在哪里

| 内容 | 位置 | 说明 |
|---|---|---|
| 完整会话记录 | `~/.codex/sessions/年/月/日/rollout-*.jsonl` | 每条消息、每个命令及输出、每次审批，都在这里 |
| 归档的会话 | `~/.codex/archived_sessions/` | 用 `/archive` 或 `codex archive` 归档的 |
| 输入历史 | `~/.codex/history.jsonl` | 你输入过的提示词；`[history] persistence = "none"` 可关闭 |
| 诊断日志 | 默认在本地有界存储里 | 需要文本日志时用 `-c log_dir=...` 指定目录 |

会话文件按日期分目录存放，文件名里带有时间和会话 ID，按日期就能找到某次会话。会话文件里可能包含代理读到的代码和命令输出，本身就是敏感数据：不要同步到公共网盘，删除会话用 `codex delete`。

团队需要集中审计时，可以配置 OpenTelemetry 把事件发到自己的收集端：

```toml
[otel]
environment = "prod"
exporter = { otlp-http = { endpoint = "https://otel.example.com/v1/logs", protocol = "binary" } }
log_user_prompt = false           # 默认不记录提示词原文
```

## 数据与服务商

- Codex 向 Responses API 发请求时固定使用 `store: false`，不要求服务端保存响应。早期版本里的 `disable_response_storage` 配置项已不存在，不需要再设置。
- 用 ChatGPT 账号登录时，数据处理遵循你账号或工作区的数据控制设置；企业套餐的保留策略、数据驻留以官方合同和说明为准。
- 用第三方接口（包括 HiveGPT）时，请求会经过该服务商，适用它的数据政策。处理客户数据或受监管代码前，先确认合规要求允许这样做。
- 不想向 OpenAI 发送使用统计，可以设置 `[analytics] enabled = false`；`/feedback` 会上传日志，涉及敏感项目时谨慎使用。

## 团队管理：托管配置

团队里光靠「请大家这样配」是不够的，Codex 提供了两类由管理员下发的配置：

| 类型 | 文件位置 | 作用 |
|---|---|---|
| 系统级默认配置 | Linux/macOS：`/etc/codex/config.toml`；Windows：`%ProgramData%\OpenAI\Codex\config.toml` | 提供默认值，用户自己的 `~/.codex/config.toml` 可以覆盖 |
| 强制要求 | Linux/macOS：`/etc/codex/requirements.toml`；Windows：`%ProgramData%\OpenAI\Codex\requirements.toml` | 限定允许的取值，用户和项目配置都不能突破 |

macOS 还支持通过 MDM 描述文件下发；使用 ChatGPT 企业工作区的团队，可以由管理员在云端下发配置。旧的 `managed_config.toml` 在 Linux/macOS 上仍会被当作 requirements 读取，在 Windows 上已不再支持，请迁移到上面的路径。

一份 `requirements.toml` 示例：

```toml
# 只允许这些审批策略和沙箱模式，完全访问被排除
allowed_approval_policies = ["untrusted", "on-request"]
allowed_sandbox_modes = ["read-only", "workspace-write"]

# 网页搜索只允许关闭或缓存模式
allowed_web_search_modes = ["disabled", "cached"]

# 只认管理员下发的钩子，忽略用户和项目里的钩子
allow_managed_hooks_only = true

# 禁用浏览器操控和电脑操控
allow_browser_and_computer_use = false

# 限定登录方式（chatgpt 或 api）
allowed_login_methods = ["api"]

# 所有人都会带上的额外指令
additional_developer_instructions = "不得读取或输出任何密钥文件；推送代码前必须征得用户同意。"

# 全员生效的命令规则
[rules]
prefix_rules = [
  { pattern = [{ token = "rm" }, { token = "-rf" }], decision = "forbidden", justification = "禁止递归强删" },
  { pattern = [{ token = "git" }, { token = "push" }], decision = "prompt", justification = "推送需人工确认" },
]

# Windows 上只允许默认（需管理员设置）的沙箱
[windows]
allowed_sandbox_implementations = ["elevated"]
```

此外还能限制可用的 MCP 服务器（按命令或 URL 识别）、插件、功能开关，甚至强制使用指定的模型服务商。字段较多，以官方文档和你所用版本为准。

排查「为什么我的配置没生效」时，在 CLI 里运行 `/debug-config`，它会列出每一层配置的来源和哪些值被 requirements 限制了。

### 统一的 AGENTS.md 和审查流程

- 在每个仓库根目录放一份 `AGENTS.md`，包含构建测试命令、编码规范和上面那段「安全」规则。可以做成模板，新仓库默认带上。
- 在 `AGENTS.md` 里写 `## Review guidelines`，让本地 `/review` 和 GitHub 上的 `@codex review` 用同一套标准。
- 涉及安全审查、合规检查的场景，用只读模式跑：`codex exec --sandbox read-only "..."` 或 `codex review`，代理只能看不能改。
- 代码合并仍然走正常的人工评审，Codex 的审查是补充而不是替代。

## 团队落地检查清单

::: details 个人开发环境
- [ ] 默认 `sandbox_mode = "workspace-write"`、`approval_policy = "on-request"`
- [ ] 没有在日常配置里使用 `danger-full-access` 或 `--dangerously-bypass-approvals-and-sandbox`
- [ ] `[shell_environment_policy]` 已收紧，`env` 输出里没有密钥
- [ ] `.env` 等密钥文件在 `.gitignore` 里，并通过权限配置或 AGENTS.md 禁止读取
- [ ] 常用安全命令用 rules 放行，危险命令设为 prompt 或 forbidden
- [ ] Windows 上已设置默认沙箱，没有以管理员身份运行 Codex
- [ ] 每轮改动审阅后再提交
:::

::: details 仓库
- [ ] 根目录有 `AGENTS.md`，包含安全规则和 Review guidelines
- [ ] 项目级 `.codex/config.toml`、`.codex/rules/` 纳入代码评审
- [ ] CI 里运行 Codex 时使用只读沙箱，Key 放在 CI 的 secrets 里
- [ ] 保护分支开启，Codex 的改动和人的改动一样走 PR 评审
:::

::: details 组织
- [ ] 下发 `requirements.toml`，排除完全访问，限制网页搜索模式
- [ ] 按需禁用电脑操控和浏览器操控，或限定可用的应用和网站
- [ ] 只允许经过审核的 MCP 服务器和插件；钩子只认管理员下发的
- [ ] 明确允许使用的模型服务商和数据政策，处理受监管数据前完成合规确认
- [ ] 配置 OpenTelemetry 或其他方式集中收集审计事件
- [ ] 新成员入职时讲一遍审批纪律和本清单
:::

## 小结

- 防护要分层：沙箱 → 审批 → 命令规则 → 网络 → 敏感文件与环境变量，任何一层失效都还有下一层。
- 日常用 `workspace-write` + `on-request`，用 rules 减少无谓的审批，而不是切到完全访问。
- Codex 默认会把全部环境变量（包括含 KEY、TOKEN 的）传给命令，记得用 `shell_environment_policy` 收紧。
- 会话完整记录在 `~/.codex/sessions/`，本身也是敏感数据；集中审计用 OpenTelemetry。
- 团队用 `requirements.toml` 强制策略、用统一的 `AGENTS.md` 约定行为，`/debug-config` 排查配置来源。

下一步：[速查表](/codex/cheatsheet)
