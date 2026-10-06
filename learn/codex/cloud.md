---
title: 云端任务与 GitHub
description: 在 chatgpt.com/codex 上连接 GitHub、配置云端环境、委派任务和并行尝试，用 codex cloud 命令在终端里管理云端任务，并在 PR 里用 @codex 做代码审查和修复。
---

# 云端任务与 GitHub

本地运行的 Codex 受限于你的电脑：电脑合上任务就停，同一时间也跑不了太多。Codex 云端把任务放到 OpenAI 托管的容器里执行：你提交任务后可以去做别的，甚至同时提交十几个；跑完后拿到一份 diff，满意就开 PR。它和 GitHub 打通后，还能在 PR 里 `@codex` 让它审查或修改代码。

这一页讲云端任务的完整流程：前提条件、环境怎么配、怎么委派和并行、`codex cloud` 命令、GitHub 代码审查，以及在 GitHub Actions 里跑 Codex。

::: warning 云端功能需要 ChatGPT 账号
云端任务、GitHub 里的 `@codex`、自动代码审查都运行在 OpenAI 的基础设施上，需要用 ChatGPT 账号登录，并且账号的套餐包含 Codex（可用套餐和额度以官方说明为准）。

第三方模型接口——包括 HiveGPT 的 Key——**只适用于在你电脑上运行的 CLI、IDE 扩展和桌面应用**。用 API Key 登录时，`codex cloud` 会直接提示需要用 ChatGPT 登录。如果你只用 HiveGPT，可以跳到本页最后的「在 GitHub Actions 里用 HiveGPT」。
:::

## 云端任务是怎么执行的

提交一个云端任务后，大致经过这几步：

1. 创建一个新容器，把你的 GitHub 仓库克隆进去，切到指定分支。
2. 运行环境的初始化脚本（安装依赖等），这个阶段可以联网。
3. 按环境设置决定代理阶段能否联网（默认不能）。
4. Codex 代理开始工作：读代码、改文件、跑测试，直到它认为完成。
5. 你看到结果：改动的 diff、它运行过的命令和输出、一段总结。

所以云端任务能做的事和本地差不多，区别在于它看不到你本机的文件，只能看到 GitHub 上已推送的代码。没推送的改动，云端看不到。

## 第一步：连接 GitHub 并创建环境

在 [chatgpt.com/codex](https://chatgpt.com/codex) 登录后：

1. 连接 GitHub 账号，授权 Codex 访问需要的仓库。建议只勾选需要的仓库，而不是全部。
2. 在设置里新建一个「环境」（environment），选择对应的仓库。
3. 配置环境的几项内容，见下表。

| 配置项 | 作用 | 建议 |
|---|---|---|
| 容器镜像 | 默认是预装了多种语言运行时的通用镜像，可以指定 Python、Node.js 等的版本 | 和项目 CI 用的版本保持一致 |
| 初始化脚本 | 容器创建后运行，用来装依赖、生成代码 | 写成幂等的，可以重复执行 |
| 维护脚本 | 复用缓存的容器时运行，用来更新依赖 | 比如 `pnpm install` 增量更新 |
| 环境变量 | 初始化脚本和代理阶段都能读到 | 只放非敏感配置 |
| 密钥（secrets） | 只在初始化脚本中可用，代理阶段会被移除 | 私有包仓库的 token 放这里 |
| 代理联网 | 关闭，或开启并限定域名和 HTTP 方法 | 默认关闭，确实需要再开 |

一个 Node.js 项目的初始化脚本示例：

```bash
corepack enable
pnpm install --frozen-lockfile
pnpm run build:types   # 代理改代码前就需要的生成步骤
```

Python 项目：

```bash
pip install -r requirements.txt
pip install -r requirements-dev.txt
```

::: tip 初始化脚本里的 export 不会保留
初始化脚本和代理在不同的 shell 会话里运行，脚本里 `export FOO=bar` 对代理不可见。需要给代理的变量请写在环境变量配置里，或者写进项目的配置文件。
:::

### 代理阶段要不要联网

默认关闭有它的道理：代理联网后，可能读到网页里藏着的恶意指令（提示注入），也可能把代码或密钥发到外面，或者装上有问题的依赖。只有在任务确实需要时（比如要查最新的 API 文档、调用测试环境的接口）才开启，并且：

- 用域名白名单，只放行必要的域名；平台提供了「常见依赖源」这类预设（包含主流的包仓库）。
- 能只允许 `GET`、`HEAD`、`OPTIONS` 就不要放开 `POST`，降低数据外发的可能。

容器会被缓存一段时间以加快后续任务启动；初始化脚本、环境变量或密钥变化后缓存失效，下一个任务会重新初始化。

## 第二步：委派任务

在网页上选择环境和分支，写下任务描述后提交。任务描述的写法和本地一样，要给清楚目标、范围和验收方式（见 [提示词最佳实践](/codex/prompting)）：

```text
在 packages/billing 里，把 calculateInvoice 中的金额计算从 number 改为整数分（cents），
避免浮点误差。改完后运行 pnpm --filter billing test，所有测试必须通过。
不要修改公开导出的函数签名。
```

### 并行多次尝试

同一个任务可以让 Codex 同时做多次尝试（best-of-N），每次独立完成，你从中挑最好的一份。适合方案不唯一的任务，比如「优化这个慢查询」「重构这个组件」。代价是消耗成倍的额度。

也可以同时提交多个不同任务。要注意：几个任务改同一批文件，最后合并时会冲突，尽量让并行的任务各管一块。

### 查看结果、开 PR

任务完成后，在任务页面可以看到：

- 每个文件的 diff
- 代理运行过的命令和输出（测试有没有真的跑过、过没过，在这里核实）
- 代理的总结

不满意可以在同一个任务里继续追加要求；满意就点创建 PR。也可以把 diff 拉到本地处理，见下一节。

## 在终端里管理云端任务

CLI 里的 `codex cloud`（标注为实验性）可以浏览和操作云端任务，前提是用 ChatGPT 登录了 CLI：

```bash
codex cloud                       # 打开交互界面浏览任务
codex cloud list                  # 列出任务（--env 按环境过滤，--limit 最多 20 条）
codex cloud exec --env ENV_ID "修复 README 里失效的链接"   # 提交新任务
codex cloud exec --env ENV_ID --attempts 3 "优化 search 接口的响应时间"  # 3 次并行尝试
codex cloud exec --env ENV_ID --branch feature/x "补充单元测试"  # 指定分支，默认当前分支
codex cloud status TASK_ID        # 查看任务状态
codex cloud diff TASK_ID          # 查看 diff，--attempt N 选第几次尝试
codex cloud apply TASK_ID         # 把 diff 应用到本地工作区，--attempt N 同上
```

`ENV_ID` 在 `codex cloud` 的交互界面里能看到。`codex apply` 是一个更短的写法，会把最近一次任务产生的 diff 用 `git apply` 应用到本地。

本地 `apply` 之后，改动就在你的工作区里了，可以用本地的 Codex（包括走 HiveGPT 的）继续修改、跑测试。

## GitHub 集成：在 PR 里用 @codex

连接 GitHub 后，可以在 Codex 的设置里为仓库开启代码审查。

### 手动请求审查

在 PR 的评论里写：

```text
@codex review
```

可以附带重点：

```text
@codex review 重点看并发安全和错误处理
```

Codex 收到后会先给评论加一个表情表示已接收，然后像普通审查者一样在 PR 上留下审查意见。默认只报告高优先级的问题（P0、P1），不会在拼写和风格上刷屏。

### 自动审查

在设置里开启自动审查后，每个新开的 PR 都会自动触发一次审查，不需要手动 `@codex`。适合团队里每个 PR 都希望有一道机器审查的仓库。

### 用 AGENTS.md 定制审查标准

Codex 审查时会读仓库里的 `AGENTS.md`，并遵循其中的审查指南。在根目录的 `AGENTS.md` 里加一节：

```markdown
## Review guidelines

- 日志里不允许输出手机号、身份证号、邮箱等个人信息
- 所有 /admin 下的路由必须经过鉴权中间件
- SQL 一律使用参数化查询，发现字符串拼接视为 P0
- 文档里的错别字也按 P1 报告
```

更深目录里的 `AGENTS.md` 只对该目录下的改动生效，适合给支付、权限这类敏感模块单独加规则。写法见 [AGENTS.md](/codex/agents-md)。

### 让它直接改代码

在 PR 评论里 `@codex` 后面跟的不是 review，而是其他任务时，会启动一个云端任务：

```text
@codex 修复这个 PR 里失败的 CI
@codex 给新增的 /api/refund 接口补上单元测试
```

完成后它会把结果推回来供你审阅。

::: tip 本地审查不需要 ChatGPT 账号
如果只是想在合并前让 Codex 看看改动，本地就可以做：交互模式里输入 `/review`，或者运行 `codex review`。这条路径用你本地配置的模型，HiveGPT 的 Key 也能用。
:::

## GitHub Actions：openai/codex-action

OpenAI 提供了官方的 GitHub Action [openai/codex-action](https://github.com/openai/codex-action)，在 CI 里安装并运行 Codex。Codex 自己的仓库就用它做 issue 分类和翻译。一个只读审查的骨架：

```yaml
name: Codex PR Review
on:
  pull_request:

jobs:
  review:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      result: ${{ steps.codex.outputs.final-message }}
    steps:
      - uses: actions/checkout@v4
      - id: codex
        uses: openai/codex-action@v1
        with:
          openai-api-key: ${{ secrets.OPENAI_API_KEY }}
          sandbox: read-only          # 只读，不让它改仓库
          safety-strategy: drop-sudo  # 运行前去掉 sudo 权限
          prompt: |
            审查这个 PR 的改动，只列出可能导致线上故障的问题，每条给出文件和行号。
```

版本号可以写主版本，也可以像 Codex 官方仓库那样固定到某个提交哈希，后者更安全。`final-message` 输出是 Codex 最后一条回复，可以在后续步骤里用 `actions/github-script` 发成 PR 评论。其他参数（模型、额外的 codex 参数、允许触发的用户等）以 action 仓库的 README 为准。

::: details 在 GitHub Actions 里用 HiveGPT
不想用 OpenAI 的 Key，也可以在工作流里直接安装 CLI，写入和本地一样的 `config.toml`，用 HiveGPT 的 Key 跑 `codex exec`。Key 放在仓库的 Actions secrets 里，不要写进仓库。完整的脚本、只读沙箱设置和把结果发成评论的方法见 [非交互模式与 CI/CD](/codex/exec)。
:::

## 小结

- 云端任务在 OpenAI 的容器里跑，只能看到 GitHub 上已推送的代码，需要 ChatGPT 账号；HiveGPT 等第三方 Key 只适用于本地。
- 环境配置的重点是初始化脚本、密钥只给初始化阶段、代理默认不联网。
- 网页、IDE、桌面端和 `codex cloud` 都能提交任务；`--attempts` 可以并行多次尝试，`codex cloud apply` 把结果拉回本地。
- PR 里 `@codex review` 做审查，审查标准写在 `AGENTS.md` 的 Review guidelines 里；`@codex` 加其他指令会启动修改任务。
- 不用云端也能审查：本地 `/review` 或 `codex review`，CI 里用 `codex exec` 或 `openai/codex-action`。

下一步：[Windows 上使用](/codex/windows)
