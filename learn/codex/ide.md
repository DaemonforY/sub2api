---
title: IDE 扩展
description: 在 VS Code、Cursor、Windsurf 等编辑器里安装 Codex 扩展，登录或接入 HiveGPT，用选中代码和打开的文件做上下文，切换 Chat / Agent 模式，并和 CLI 共用同一份配置。
---

# IDE 扩展

如果你大部分时间待在编辑器里，来回切到终端或另一个窗口去问 Codex 会打断节奏。Codex 的 IDE 扩展把同一个代理放进编辑器侧边栏：它知道你当前打开了哪个文件、选中了哪几行，改完的代码直接出现在编辑器里，用编辑器自带的 diff 审阅。

这一页讲安装、登录（包括用 HiveGPT 的 Key）、侧边栏怎么用、上下文怎么带进去、三档权限模式怎么选，以及和 CLI 共用配置的细节。

## 支持哪些编辑器

| 编辑器 | 安装方式 | 说明 |
|---|---|---|
| VS Code / VS Code Insiders | 扩展市场搜索 Codex，发布者 OpenAI | 扩展 ID 是 `openai.chatgpt` |
| Cursor | 扩展面板搜索，或命令行安装 | 基于 VS Code，用同一个扩展 |
| Windsurf | 同上 | 基于 VS Code，用同一个扩展 |
| 其他 VS Code 系编辑器 | 视其扩展市场是否收录而定 | 以官方说明为准 |
| JetBrains 系列（IntelliJ、PyCharm 等） | 通过 JetBrains 自家的 AI 功能接入 Codex | 入口、登录方式与 VS Code 扩展不同，以 JetBrains 和 OpenAI 官方说明为准 |

::: warning 认准发布者
扩展市场里有不少名字带 Codex 或 ChatGPT 的第三方扩展。安装前确认发布者是 OpenAI、扩展 ID 是 `openai.chatgpt`，避免装到来路不明的扩展把代码和 Key 发到别处。
:::

命令行安装：

```bash
code --install-extension openai.chatgpt      # VS Code
cursor --install-extension openai.chatgpt    # Cursor
windsurf --install-extension openai.chatgpt  # Windsurf
```

装好后，活动栏会多出一个 Codex 图标。侧边栏比较窄的话，可以把 Codex 面板拖到右侧的辅助侧边栏，和文件树分开放，用起来更舒服。

## 登录或接入 HiveGPT

扩展第一次打开时会让你选择登录方式：

- **用 ChatGPT 账号登录**：本地任务和云端任务都能用。
- **用 API Key**：只能跑本地任务。

扩展和 CLI、桌面应用读的是同一个 `~/.codex` 目录（Windows 是 `%USERPROFILE%\.codex`）。所以要接 HiveGPT，不需要在扩展里另外配置，按 [接入 HiveGPT](/codex/hivegpt) 生成 `~/.codex/config.toml` 和 `~/.codex/auth.json` 就行，然后重启编辑器（或在命令面板执行「Developer: Reload Window」）。扩展启动时读到 `auth.json` 里的 Key，就会直接用 `config.toml` 里配置的服务地址和 `gpt-5.5` 模型。

::: tip 验证接入是否生效
在扩展里发一句「你现在用的是什么模型？用一句话回答」只能看出模型的自述，不可靠。更准确的办法是去 HiveGPT 控制台的使用记录里看有没有这次请求，或者在终端跑一次 `codex`，用 `/status` 看当前的模型和服务商，两者配置相同。
:::

## 侧边栏的基本用法

侧边栏就是一个对话面板。底部输入框附近有几样东西：

| 位置 | 作用 |
|---|---|
| 模式切换 | Chat / Agent / Agent（完全访问），见下一节 |
| 模型与推理强度 | 默认 `gpt-5.5`，其他可选项以 `/model` 或下拉列表为准 |
| 本地 / 云端 | 任务在本机跑，还是委派到云端（需 ChatGPT 账号） |
| 自动上下文开关 | 是否自动带上当前文件和选区 |
| `@` | 引用工作区里的其他文件 |
| `/` | 斜杠命令，例如 `/review` 审查当前改动 |
| 附件 | 贴图或截图，比如报错截图、设计稿 |

Codex 改完文件后，面板里会列出改动的文件，点击就用编辑器的 diff 视图打开。审阅的方式和你审同事的代码一样：看不懂的地方直接在对话里追问「第 42 行为什么要加这个判断」。

## 上下文是怎么带进去的

这是 IDE 扩展相比 CLI 最大的便利。打开自动上下文后，每次发消息会带上：

- 当前激活的文件及其路径
- 你在编辑器里选中的代码范围和内容
- 其他打开的标签页的文件列表

所以很多问题不需要再解释「我说的是哪个文件」：

```text
（选中一个函数）
这个函数在 items 为空数组时会返回什么？写一个测试覆盖这种情况。
```

```text
（打开 src/api/user.ts）
这里的错误处理和 src/api/order.ts 不一致，统一成 order.ts 的写法。
```

几条经验：

- **选区越准，回答越准**。问一个函数就只选那个函数，不要全选整个文件。
- **只关心当前文件时，关掉无关标签页**，避免模型被打开的其他文件分散注意力。
- **需要它主动搜索时，明说**：「先搜一下项目里还有哪些地方调用了 `formatPrice`」。自动上下文只是起点，Agent 模式下它会自己去读文件、跑 `rg`。

::: details 在终端 CLI 里也能用编辑器的上下文
如果你更喜欢在编辑器的集成终端里跑 `codex`，可以在 CLI 里输入 `/ide`，把编辑器当前的选区、打开的文件等上下文带进 CLI 会话（需要编辑器里装着并运行 Codex 扩展）。
:::

## Chat、Agent 与完全访问

扩展把权限做成了三档，概念和 CLI 的 `/permissions`、`--sandbox` 一致：

| 模式 | 能做什么 | 什么时候需要你批准 | 适合 |
|---|---|---|---|
| Chat | 只对话，不改文件、不跑命令 | — | 问问题、解释代码、讨论方案 |
| Agent | 在工作区内读写文件、执行命令 | 访问网络、改工作区以外的文件 | 日常开发（默认推荐） |
| Agent（完全访问） | 不受沙箱限制，可联网，可改任何文件 | 不再询问 | 你清楚风险、且环境可丢弃时 |

日常用 Agent 就够了。需要联网（装依赖、调接口）时，它会弹出批准请求，你看清楚命令再点允许。完全访问跳过了所有确认，被提示注入或者理解错需求时损失可能很大，详见 [沙箱与审批](/codex/sandbox) 和 [安全与团队管理](/codex/security)。

## 和 CLI 共用的配置

扩展会读取 `~/.codex/config.toml` 里的大部分设置，常用的有：

```toml
model = "gpt-5.5"
model_reasoning_effort = "medium"

# 默认权限
sandbox_mode = "workspace-write"
approval_policy = "on-request"

# 模型回答里提到的文件路径，用哪个编辑器打开
file_opener = "vscode"   # 可选 vscode / vscode-insiders / cursor / windsurf / none
```

此外这些也是共享的：

- 全局 `~/.codex/AGENTS.md` 和项目里的 `AGENTS.md`（[AGENTS.md](/codex/agents-md)）
- MCP 服务器配置（[MCP](/codex/mcp)）
- Skills（[Skills](/codex/skills)）
- 会话记录，所以在 CLI 里可以 `codex resume` 找到在扩展里聊过的会话

## 委派到云端

用 ChatGPT 账号登录时，可以把任务切到云端执行：任务在 OpenAI 的容器里跑，你可以关掉编辑器去做别的，完成后在扩展里查看 diff，再应用到本地或者直接开 PR。云端环境需要先在网页端配置好，细节见 [云端任务与 GitHub](/codex/cloud)。用 HiveGPT 等第三方 Key 时这个选项不可用。

## 常见问题

**扩展一直转圈或提示连接失败？**
先在终端跑一次 `codex`，能正常对话说明配置没问题，问题出在扩展；不能对话就按 [常见问题排查](/codex/troubleshooting) 先修好 CLI。扩展本身的日志在编辑器的「输出」面板里，下拉选择 Codex 相关的通道查看。

**改了 config.toml 但扩展没生效？**
重载编辑器窗口。扩展启动时读取配置，运行中改动不一定会被读到。

**快捷键和其他扩展冲突？**
在编辑器的键盘快捷方式设置里搜索 Codex，查看并修改它的命令绑定。

**公司电脑上装不了扩展市场的扩展？**
可以从扩展市场网页下载 VSIX 安装，或者退而在集成终端里用 CLI，配合 `/ide` 带上下文，效果接近。

**扩展和桌面应用怎么选？**
写代码时顺手问、顺手改，用扩展；同时管理多个长任务、需要 worktree 隔离和集中审阅，用 [桌面应用](/codex/app)。两者配置共用，可以混着用。

## 小结

- VS Code、Cursor、Windsurf 用同一个扩展 `openai.chatgpt`，安装时认准发布者 OpenAI。
- 扩展和 CLI 共用 `~/.codex`，按 [接入 HiveGPT](/codex/hivegpt) 配好后重载窗口即可使用 `gpt-5.5`。
- 当前文件、选区、打开的标签页会作为上下文自动带入，选区越精确越好。
- 日常用 Agent 模式；完全访问会跳过所有确认，谨慎使用。
- 云端委派需要 ChatGPT 账号，第三方 Key 只能跑本地。

下一步：[云端任务与 GitHub](/codex/cloud)
