---
title: Skills
description: Skill 是一个带 SKILL.md 的文件夹，把某类任务的做法、脚本和参考资料打包起来，Codex 需要时才加载。本页讲存放位置、文件格式、触发方式，并从零写一个发布前检查 skill。
---

# Skills

AGENTS.md 适合写「这个项目的规矩」，但有些知识只在特定任务里才用得上：怎样发版、怎样写数据库迁移、怎样按公司模板生成周报。把这些全塞进 AGENTS.md，每次会话都要背着它们，既占上下文又分散注意力。

Skill 解决的就是这个问题。一个 skill 是一个文件夹，里面有一份 `SKILL.md` 说明「什么时候用、怎么做」，还可以带上脚本、模板和参考文档。Codex 平时只知道每个 skill 的名字和一句话描述，真正需要时才去读完整内容。这一页讲 skill 的结构、放在哪里、怎么触发，并从零写一个可用的例子。

## Skill 的结构

最小的 skill 只需要一个文件：

```text
release-check/
└── SKILL.md
```

完整一些的 skill 会附带资源：

```text
release-check/
├── SKILL.md            # 必需：元数据 + 操作说明
├── scripts/            # 可选：需要确定性执行的脚本
│   └── preflight.sh
├── references/         # 可选：需要时再读的参考资料
│   └── changelog-style.md
├── assets/             # 可选：模板、图标等
│   └── release-notes.tmpl.md
└── agents/
    └── openai.yaml     # 可选：界面显示、调用策略、依赖声明
```

`SKILL.md` 开头是 YAML frontmatter，后面是正文：

```markdown
---
name: release-check
description: 发布新版本前的检查清单。当用户要发版、打 tag、准备 release、写更新日志时使用。
metadata:
  short-description: 发版前检查
---

# 发布前检查
（正文：具体步骤）
```

根据源码（`codex-rs/skills/src/parser.rs`），frontmatter 的规则是：

| 字段 | 是否必需 | 说明 |
|---|---|---|
| `name` | 可选 | 不写时用文件夹名；最长 64 个字符，建议用小写字母和连字符 |
| `description` | 必需 | 缺少就会加载失败；多行会被合并成一行 |
| `metadata.short-description` | 可选 | 界面里显示的简短说明 |

## 按需加载：为什么 description 最重要

Codex 加载 skill 分两个阶段：

1. **会话开始时**：只把每个 skill 的名字、描述和路径放进一个「可用技能列表」。这个列表有预算，默认约为模型上下文窗口的 2%。skill 太多时，描述会被缩短，极端情况下有的 skill 会被挤出列表。
2. **决定使用时**：Codex 才去读完整的 `SKILL.md`，再按正文的指引，按需读取 `references/` 里的文档或运行 `scripts/` 里的脚本。

所以，正文可以写得详细，但 description 必须写好：它决定了 Codex 能不能在正确的时机想起这个 skill。好的 description 要回答两件事：**做什么**、**什么时候用**，并把触发关键词放在前面。

| 写法 | 问题 |
|---|---|
| `description: 发布工具` | 太短，看不出何时该用 |
| `description: 这是一个非常强大的、可以帮助你完成各种与版本发布相关工作的综合性技能……` | 关键信息在后面，一被缩短就只剩废话 |
| `description: 发版前检查清单。用户要发版、打 tag、准备 release、写 CHANGELOG 时使用。` | 先说做什么，再列触发场景 |

## 放在哪里

Codex 会从多个位置扫描 skill。以下按源码（`codex-rs/ext/skills/src/host_roots.rs`）整理：

| 范围 | 位置 | 适合放什么 |
|---|---|---|
| 仓库 | 从项目根到当前目录，每一层的 `.agents/skills/` | 团队共享、随代码版本走的 skill |
| 项目配置 | 项目的 `.codex/skills/`（需信任该项目） | 同上，和项目的 `.codex/config.toml` 放在一起 |
| 用户 | `~/.agents/skills/` | 你个人在所有项目都想用的 skill |
| 用户（旧位置） | `~/.codex/skills/` | 仍然有效，为兼容保留 |
| 管理员 | `/etc/codex/skills/` | 机器或容器级别统一下发 |
| 系统内置 | `~/.codex/skills/.system/` | Codex 自带，启动时自动释放到这里 |
| 插件 | 已安装插件携带的 skill | 通过 `/plugins` 安装和管理 |

仓库级 skill 的查找方式和 AGENTS.md 类似：从项目根（默认以 `.git` 识别）到当前工作目录，逐层检查有没有 `.agents/skills`。所以 monorepo 里可以在根目录放通用 skill，在子项目里放专用 skill。

Codex 自带的系统 skill 目前包括 `skill-creator`（创建 skill）、`skill-installer`（从清单或 GitHub 仓库安装 skill）、`openai-docs`、`imagegen` 等，具体以你本机 `/skills` 里列出的为准。不想要内置 skill，可以在配置里关闭：

```toml
# ~/.codex/config.toml
[skills.bundled]
enabled = false
```

## 怎样触发

有三种方式：

1. **用 `$` 显式点名**：在输入框里打 `$`，会弹出 skill 列表，选中后插入，比如 `$release-check 准备发布 v2.3.0`。显式点名时 Codex 一定会加载这个 skill。
2. **用 `/skills` 浏览**：输入 `/skills` 查看当前可用的 skill，从列表里选择使用。
3. **自动匹配**：你描述的任务和某个 skill 的 description 吻合时，Codex 会自己决定使用它。比如说「帮我准备发版」，它可能就会用上 `release-check`。

如果某个 skill 只想手动调用、不希望 Codex 自作主张，可以在 `agents/openai.yaml` 里关闭自动调用：

```yaml
# release-check/agents/openai.yaml
interface:
  display_name: "发布前检查"
  short_description: "发版前跑一遍检查清单"
  default_prompt: "按发布前检查流程检查当前分支"
policy:
  allow_implicit_invocation: false
```

`interface` 里还可以设置图标（`icon_small`、`icon_large`）和主题色（`brand_color`），主要用于桌面应用和 IDE 里的显示。`dependencies` 可以声明 skill 依赖的 MCP 工具，缺少时 Codex 会提示安装。

## 动手写一个：发布前检查

假设你维护一个 Go 后端 + Vue 前端的项目，每次发版前都要做同样几件事：确认工作区干净、跑测试、检查版本号、整理更新日志。下面把它做成一个仓库级 skill。

### 1. 创建目录

```bash
mkdir -p .agents/skills/release-check/{scripts,references}
```

### 2. 写 SKILL.md

````markdown
---
name: release-check
description: 发版前检查清单。用户要发布新版本、打 tag、准备 release 或整理 CHANGELOG 时使用；只做检查和准备，不推送、不打 tag。
metadata:
  short-description: 发版前检查
---

# 发布前检查

目标：在发版前发现问题，并产出一份可直接粘贴的发布说明草稿。
本技能**不执行**任何推送、打 tag、发布操作，这些由用户手动完成。

## 步骤

1. 运行自动检查脚本，并完整阅读输出：

   ```bash
   bash .agents/skills/release-check/scripts/preflight.sh
   ```

   脚本任何一项显示 FAIL，就停下来，向用户报告失败项和原因，不要继续后面的步骤。

2. 确认版本号：
   - 读取 backend/VERSION 和 web/package.json 的 version，二者必须一致。
   - 新版本号由用户给出；用户没给时，根据自上一个 tag 以来的提交类型建议一个
     （有 feat 升次版本号，只有 fix 升修订号），并请用户确认。

3. 整理更新日志：
   - 运行 `git log --oneline $(git describe --tags --abbrev=0)..HEAD` 获取提交列表。
   - 按 references/changelog-style.md 的格式分组整理，写到 CHANGELOG.md 顶部。
   - 只写用户能感知到的变化，合并同类项，去掉纯重构和 CI 调整。

4. 汇报：用下面的格式回复用户。

   ```text
   发布检查：通过 / 未通过
   版本：vX.Y.Z
   检查结果：（逐项列出）
   CHANGELOG：已更新 / 未更新
   下一步需要你手动执行的命令：
     git tag vX.Y.Z && git push origin vX.Y.Z
   ```

## 注意
- 不要修改业务代码来让检查通过；遇到失败只报告。
- 测试失败时附上失败的测试名和关键报错行。
````

### 3. 写检查脚本

把确定性的检查交给脚本，比让模型每次即兴拼命令更可靠：

```bash
#!/usr/bin/env bash
# .agents/skills/release-check/scripts/preflight.sh
set -u
fail=0
check() {  # 用法：check "说明" 命令...
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then echo "PASS  $name"; else echo "FAIL  $name"; fail=1; fi
}

check "工作区没有未提交的改动"   test -z "$(git status --porcelain)"
check "当前在 main 分支"         test "$(git branch --show-current)" = "main"
check "后端单元测试"             bash -c "cd backend && go test -tags=unit ./..."
check "后端静态检查"             bash -c "cd backend && go vet ./..."
check "前端类型检查"             bash -c "cd web && pnpm run typecheck"

v_backend=$(cat backend/VERSION 2>/dev/null)
v_web=$(node -p "require('./web/package.json').version" 2>/dev/null)
check "前后端版本号一致 ($v_backend / $v_web)" test "$v_backend" = "$v_web"

exit $fail
```

```bash
chmod +x .agents/skills/release-check/scripts/preflight.sh
```

### 4. 写参考资料

`references/changelog-style.md` 里放团队的更新日志格式要求和一两个范例。它只在执行到第 3 步时才会被读取，不占平时的上下文。

### 5. 试用

重启 Codex（或开一个新会话），然后：

```text
$release-check 准备发布 1.8.0
```

观察它是否按步骤执行、在脚本失败时是否停下。不满意就改 `SKILL.md`，再试。

::: tip 让 Codex 帮你写 skill
内置的 `skill-creator` 就是干这个的。输入 `$skill-creator`，告诉它你想要什么 skill、什么时候触发、需不需要脚本，它会生成目录和文件。生成后照样要人工审查一遍。
:::

## 启用和禁用

不想删除文件、只想临时停用某个 skill，可以在 `config.toml` 里按路径或名字关闭：

```toml
[[skills.config]]
name = "release-check"
enabled = false

[[skills.config]]
path = "/Users/me/.agents/skills/old-report/SKILL.md"
enabled = false
```

skill 很多、担心挤占上下文时，可以调整技能列表的 token 预算（设置后最高 10000）：

```toml
[skills]
max_context_tokens = 4000
```

## Skill、AGENTS.md、MCP 怎么选

| | AGENTS.md | Skill | MCP |
|---|---|---|---|
| 本质 | 项目说明书 | 打包好的操作流程 | 外部工具和数据的接口 |
| 加载时机 | 每次会话开始都加载 | 只加载描述，用到时再读全文 | 启动时连接，工具随时可调用 |
| 适合放 | 全局约定、命令、硬性约束 | 特定任务的步骤、模板、脚本 | 数据库、浏览器、issue 系统、文档检索 |
| 能否执行代码 | 不能，只是文字 | 能，通过附带的脚本 | 能，由 MCP 服务器执行 |
| 例子 | 「只用 pnpm」「金额用分」 | 发版检查、写迁移、生成周报 | 查 GitHub issue、操作浏览器 |

一个简单的判断：**每次都要遵守的**放 AGENTS.md；**某类任务才需要的做法**做成 skill；**需要连接外部系统的能力**用 [MCP](/codex/mcp)。三者可以配合：skill 的步骤里可以调用某个 MCP 工具，AGENTS.md 里可以写「发版时使用 `$release-check`」。

## 最佳实践

- **一个 skill 只做一件事**。「发版检查」和「写迁移」分成两个 skill，description 各自清晰，匹配才准。
- **description 写触发场景**，把关键词放前面；正文写步骤，越具体越好。
- **确定性的部分交给脚本**。检查、格式转换、调用固定命令，用脚本比让模型每次自己拼命令可靠。
- **大块资料放 references**，并在正文里写明「什么时候读哪个文件」，不要全部塞进 SKILL.md。
- **写清楚边界**。像示例里那样声明「不推送、不打 tag」，有副作用的操作留给人来做。
- **仓库级 skill 提交到 Git**，和代码一起评审、一起演进。
- **审查来源**。从网上安装的 skill 可能包含脚本，安装前读一遍内容，就像审查任何第三方代码一样。

## 小结

- Skill 是一个包含 `SKILL.md` 的文件夹，可附带 scripts、references、assets；`description` 必填，`name` 默认取文件夹名。
- 会话开始只加载名字和描述，用到时才读全文，所以 description 要写清「做什么、何时用」。
- 仓库级放在各层的 `.agents/skills/`，个人的放 `~/.agents/skills/`，旧的 `~/.codex/skills/` 仍可用。
- 用 `$名字` 显式调用、`/skills` 浏览，或让 Codex 按描述自动匹配；`allow_implicit_invocation: false` 可以关闭自动匹配。
- 每次都要遵守的写 AGENTS.md，特定任务的流程做成 skill，连接外部系统用 MCP。

下一步：给 Codex 接上外部工具，见 [MCP](/codex/mcp)。
