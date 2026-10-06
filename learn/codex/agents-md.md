---
title: AGENTS.md
description: AGENTS.md 是写给 Codex 的项目说明书：它从哪里加载、多个文件怎样合并、大小上限多少、该写什么不该写什么，以及一份可以直接改用的完整示例。
---

# AGENTS.md

每次开新会话都要重新交代一遍「这个项目用 pnpm」「改完要跑 make test」「别动 generated 目录」，很快就会让人厌烦，而且总有漏掉的时候。`AGENTS.md` 就是解决这个问题的：它是放在仓库里、专门写给编程代理看的说明文件。Codex 启动时会自动读取它，把内容当作这次会话的项目指令。

`AGENTS.md` 不是 Codex 私有的格式，它是多家编程代理共同支持的约定，所以写一份，团队里用不同工具的人都能受益。这一页讲 Codex 怎样查找和合并这些文件、写什么最有用，并给出一份完整示例。

## Codex 怎样找到 AGENTS.md

以下规则依据 openai/codex 源码（`codex-rs/core/src/agents_md.rs` 等）整理。

### 第一层：全局指令

Codex 先看自己的主目录（默认 `~/.codex`，可以用环境变量 `CODEX_HOME` 改到别处）：

1. 有 `~/.codex/AGENTS.override.md` 且内容非空，就用它；
2. 否则用 `~/.codex/AGENTS.md`；
3. 两个都没有或都为空，就没有全局指令。

全局文件只会用一个，适合写跨项目的个人习惯，比如「回复用中文」「提交信息用 Conventional Commits」。

### 第二层：项目指令

接着 Codex 确定项目根目录：从当前工作目录往上找，遇到含有 `.git` 的目录就认定为根。然后**从根目录到当前目录，逐级**在每一层查找指令文件。每一层按下面的顺序取**第一个存在的文件**：

1. `AGENTS.override.md`
2. `AGENTS.md`
3. `project_doc_fallback_filenames` 里配置的备用文件名（按配置顺序）

找到的文件按「根目录在前、当前目录在后」的顺序拼接，越靠近当前目录的越晚出现。模型被告知：嵌套更深的 AGENTS.md 在冲突时优先，你在对话里直接给出的指令又优先于所有 AGENTS.md。

举个例子，你在 `repo/services/payments` 里启动 Codex：

```text
repo/
├── .git/
├── AGENTS.md                      ← 加载（第 1 段）
└── services/
    ├── AGENTS.md                  ← 加载（第 2 段）
    ├── payments/
    │   ├── AGENTS.md              ← 跳过：同目录有 override
    │   └── AGENTS.override.md     ← 加载（第 3 段）
    └── search/
        └── AGENTS.md              ← 启动时不加载：不在根到当前目录的路径上
```

几个容易误会的地方：

- **当前目录以下的子目录不会在启动时加载**。上例中 `services/search/AGENTS.md` 不会出现在初始指令里。不过 Codex 的系统提示会告诉模型：去子目录或其他目录改文件时，要主动检查那里有没有适用的 AGENTS.md。
- **找不到 `.git` 时只看当前目录**，不会一路往上找到你的家目录。如果你的项目不是 Git 仓库，或者用别的标记识别根目录，可以在 `config.toml` 里设置 `project_root_markers`。
- **未受信任的项目不加载项目级 AGENTS.md**。第一次在某个目录启动 Codex 时它会询问是否信任该目录；如果选择不信任，只有全局指令生效。
- **空文件会被跳过**，只含空白字符的文件等于不存在。

### 大小上限

项目级指令的总大小默认上限是 **32 KiB**（`project_doc_max_bytes = 32768`）。按拼接顺序累计，超出的部分会被截断，后面的文件可能整个读不到。全局 AGENTS.md 不占这个额度。

```toml
# ~/.codex/config.toml
project_doc_max_bytes = 65536                         # 放宽到 64 KiB
project_doc_fallback_filenames = ["CLAUDE.md", "TEAM_GUIDE.md"]
project_root_markers = [".git", ".hg", "go.work"]     # 识别项目根的标记
```

::: tip 别急着调大上限
32 KiB 大约是一两万汉字，远超一份好 AGENTS.md 的需要。指令越长，模型越难抓住重点，还会占用每轮对话的上下文。内容超了，首先该做的是精简，或者把子模块的规则拆到子目录的 AGENTS.md 里。
:::

### override 文件的用途

`AGENTS.override.md` 会让同目录的 `AGENTS.md` 被完全跳过（不是合并）。典型用途：

- **临时覆盖**：排查问题时想让 Codex 换一套规则，在 `~/.codex/` 放一个 override，用完删掉即可恢复。
- **个人本地定制**：团队的 `AGENTS.md` 提交在仓库里，你在本地放一个 `AGENTS.override.md`（加进 `.gitignore`），写上团队内容加你个人的补充。注意它是**替换**，团队文件里的内容要自己复制过来。

### 验证实际加载了什么

最直接的方法是问它：

```bash
codex "列出你本次会话加载的指令文件路径，并用三句话概括其中的规则。不要修改任何文件。"
```

想看模型实际收到的原始输入，可以用调试命令把提示内容渲染出来：

```bash
codex debug prompt-input "hello" | less
```

## 用 /init 生成初稿

在项目里输入 `/init`，Codex 会扫描目录结构、构建脚本、测试配置和 Git 历史，生成一份名为「Repository Guidelines」的 `AGENTS.md`，包含项目结构、构建与测试命令、代码风格、测试规范、提交与 PR 规范等章节，篇幅控制在几百字。如果当前目录已经有 `AGENTS.md`，它不会覆盖。

`/init` 的结果是起点，不是终稿。生成之后务必人工过一遍：

- 删掉它猜错或者太泛的内容（比如「保持代码整洁」）。
- 补上只有团队才知道的规则：哪些目录是生成的不能手改、哪些命令很慢不要随便跑、发布流程的注意事项。
- 核对每条命令都真的能跑。

## 写什么，不写什么

AGENTS.md 的读者是一个能力很强、但对你项目一无所知的新同事。写那些**它从代码里看不出来、但做错了代价很大**的东西。

| 值得写 | 不必写 |
|---|---|
| 构建、测试、lint 的准确命令，以及哪个命令最快 | 「写高质量代码」这类空话 |
| 改完必须跑的检查，和通过标准 | 语言或框架的通用教程 |
| 生成代码的位置和重新生成的命令 | 代码里一眼就能看出的目录结构细节 |
| 不能手改、不能删除的文件或目录 | 完整的 API 文档（给链接或路径即可） |
| 项目特有的约定（命名、错误处理、日志格式） | 经常变化的信息（版本号、当前迭代任务） |
| 容易踩的坑和历史教训 | 密钥、密码、内部服务地址 |
| 提交信息和 PR 的格式要求 | 和本仓库无关的个人偏好（放全局文件） |

写法上的建议：

- **用命令代替描述**。「运行测试」不如 `go test -tags=unit ./...`。
- **说明原因**。「不要用 npm，因为混用会损坏 node_modules」比单纯禁止更容易被遵守，遇到没写到的情况，模型也能按这个理由推断。
- **短句、列表、分节**。模型和人一样，扫读结构清晰的文本效果最好。
- **规则要可执行、可检查**。「函数不超过 50 行」能检查；「函数要简短」不能。

## 一份完整示例

下面是一个虚构的 Vue 3 + Go 全栈项目的根目录 `AGENTS.md`，可以按自己的项目改写：

```markdown
# AGENTS.md

## 项目概览
「小账本」：个人记账 Web 应用。后端 Go 1.23 + Gin + Ent（backend/），
前端 Vue 3 + TypeScript + Pinia + Tailwind（web/）。数据库 PostgreSQL 16。

## 目录
- backend/cmd/server/      服务入口
- backend/internal/handler/  HTTP 处理，只做参数绑定和调用 service
- backend/internal/service/  业务逻辑，大部分改动在这里
- backend/ent/schema/       数据模型的唯一来源；backend/ent/ 其余文件是生成的，禁止手改
- web/src/api/              所有后端请求都通过这里的封装，组件里不要直接用 axios

## 常用命令
后端（在 backend/ 下运行）：
- 单元测试：go test -tags=unit ./...       约 20 秒，改完后端必须跑
- 单个测试：go test -tags=unit ./internal/service/ -run TestLedger
- 修改 ent/schema 后：make generate        会重新生成 ent/ 和 wire_gen.go
- 静态检查：golangci-lint run ./...
前端（在 web/ 下运行，只用 pnpm）：
- 类型检查：pnpm run typecheck
- 单个测试：pnpm exec vitest run src/views/ledger/LedgerList.spec.ts
- 不要运行 pnpm dev，它是常驻进程会卡住会话

## 代码约定
- Go 错误用 fmt.Errorf("动作: %w", err) 包装后向上返回，只在 handler 层记录日志
- 金额一律用 int64 存「分」，禁止用 float
- 新接口返回统一结构 {code, message, data}，参考 handler/response.go
- Vue 组件用 <script setup lang="ts">，文案走 i18n（web/src/i18n/zh.ts），不要写死中文

## 依赖
- 前端只用 pnpm；混用 npm 会损坏 node_modules。改了 package.json 必须一起提交 pnpm-lock.yaml
- 新增生产依赖前先说明理由并等我确认

## 完成标准
提交前确认：
1. 后端改动：go test -tags=unit ./... 和 golangci-lint 通过
2. 前端改动：pnpm run typecheck 通过，相关 spec 通过
3. 改了 ent/schema：已运行 make generate，生成文件一并提交
4. 最终回复里列出改动的文件和验证命令的结果

## 提交规范
Conventional Commits，scope 用模块名，例如 feat(ledger): 支持按月导出。
一个提交只做一件事。

## 已知的坑
- 测试依赖 TZ=Asia/Shanghai，CI 里已设置；本地时间相关测试失败先检查时区
- internal/service/report.go 的 SQL 是手写的性能优化版本，改动前先问我
```

前端子目录还可以再放一份更细的 `web/AGENTS.md`，只写前端特有的规则。在 `web/` 里启动 Codex 时，根目录和 `web/` 的两份会依次加载。

## 和 README、CONTRIBUTING 的分工

三者读者不同，内容会有交集，但侧重点不一样：

| 文件 | 读者 | 侧重 |
|---|---|---|
| README.md | 用户、新访客 | 项目是什么、怎么安装和使用 |
| CONTRIBUTING.md | 人类贡献者 | 开发流程、评审规则、沟通方式 |
| AGENTS.md | 编程代理 | 精确的命令、硬性约束、完成标准、坑 |

已经写在 README 里的长内容，AGENTS.md 里不必重复，写一句「本地环境搭建见 docs/dev-setup.md」即可，Codex 需要时会自己去读。反过来，AGENTS.md 里那些「禁止手改 ent/」「金额用分」的硬规则，对人类贡献者同样有用，可以考虑同步到 CONTRIBUTING。

如果团队已经有现成的说明文件（比如别的工具用的 `CLAUDE.md`），不想维护两份，可以把它加进 `project_doc_fallback_filenames`：某一层没有 `AGENTS.md` 时，Codex 会读这个备用文件。

## 团队维护建议

- **当代码一样评审**。AGENTS.md 提交在仓库里，改动走 PR，让团队成员都能看到规则的变化。
- **从真实问题里长出来**。Codex 犯了一次错，想想能不能用一条规则避免下次再犯，再把这条规则写进去。这样积累的内容最有用。
- **定期删减**。过时的命令、已经修复的坑要及时删掉；错误的指令比没有指令更糟。
- **全局放个人习惯，仓库放团队规则**。「我喜欢详细的解释」放 `~/.codex/AGENTS.md`，不要提交到仓库里影响别人。
- **大型 monorepo 分层写**。根目录写全局约定，各子项目写自己的命令和规则，每份都保持短小。
- **不要写敏感信息**。AGENTS.md 会进入模型上下文，也会被所有能访问仓库的人看到。

## 小结

- Codex 先加载 `~/.codex` 下的全局指令（override 优先），再从项目根（默认以 `.git` 识别）到当前目录逐级加载项目指令。
- 每层只取一个文件，顺序是 `AGENTS.override.md`、`AGENTS.md`、备用文件名；项目指令合计默认上限 32 KiB，超出截断。
- 未受信任的目录不加载项目级 AGENTS.md；当前目录以下的子目录不会在启动时加载。
- 用 `/init` 生成初稿后一定人工修改；只写代码里看不出来、做错代价大的规则，命令要准确可执行。
- 把 AGENTS.md 当代码维护：走评审、从真实错误中积累、定期删减。

下一步：把可复用的工作流程打包起来，见 [Skills](/codex/skills)。
