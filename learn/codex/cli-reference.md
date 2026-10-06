---
title: 命令行参数
description: codex 命令的全局参数、全部子命令和 -c 覆盖配置的写法，按 codex-cli 0.142 的帮助输出和官方源码整理。
---

# 命令行参数

`codex` 不只是一个打开交互界面的命令。它有一组全局参数，可以在启动时临时指定模型、权限和工作目录；还有二十多个子命令，覆盖非交互执行、代码审阅、会话管理、MCP 配置、沙箱测试等场景。这一页把它们整理成表，并给出能直接复制的例子。

内容以本机 `codex-cli 0.142.3` 的 `codex --help` 为基准，同时对照了官方仓库最新源码；两者有差异的地方会单独说明。

::: tip 随时查帮助
任何命令后面加 `--help` 都能看到完整说明，例如 `codex exec --help`、`codex mcp add --help`。`-h` 显示简版，`--help` 显示详细版。
:::

## 基本形式

```bash
codex [全局参数] [提示词]          # 启动交互界面，可选地带一句开场任务
codex [全局参数] 子命令 [参数]      # 运行某个子命令
```

几个最常见的启动方式：

```bash
codex                                   # 在当前目录打开交互界面
codex "解释一下这个仓库的目录结构"        # 打开界面并立即发出第一条消息
codex -C ~/work/api "跑一下测试并修复失败用例"   # 指定工作目录
codex exec "给 README 补一节安装说明"     # 非交互执行，做完就退出
```

## 全局参数

这些参数用于交互模式，大部分也能用在 `exec`、`resume`、`fork` 等子命令上。

### 模型与配置

| 参数 | 说明 | 示例 |
|---|---|---|
| `-m, --model` | 本次使用的模型 | `codex -m gpt-5.5` |
| `-c, --config key=value` | 临时覆盖任意配置项，可重复 | `-c model_reasoning_effort="high"` |
| `-p, --profile 名称` | 叠加 `~/.codex/名称.config.toml` 这份配置 | `codex -p deep` |
| `--enable 功能` | 打开一个功能开关，等于 `-c features.功能=true` | `--enable memories` |
| `--disable 功能` | 关闭一个功能开关 | `--disable hooks` |
| `--strict-config` | config.toml 里出现本版本不认识的键时直接报错 | 排查拼写错误 |
| `--oss` | 使用本地开源模型提供方 | `codex --oss` |
| `--local-provider` | 和 `--oss` 搭配，指定 `lmstudio` 或 `ollama` | `--oss --local-provider ollama` |

::: warning -p 的含义变了
早期版本的 profile 写在 `config.toml` 的 `[profiles.名称]` 表里。现在 `-p 名称` 读取的是**单独的文件** `~/.codex/名称.config.toml`，叠加在主配置之上。新版本里再写 `profile = "名称"` 会直接报错，迁移方法见 [高级配置](/codex/config-advanced)。
:::

### 权限与沙箱

| 参数 | 说明 |
|---|---|
| `-s, --sandbox` | 沙箱模式：`read-only`、`workspace-write`、`danger-full-access` |
| `-a, --ask-for-approval` | 审批策略，见下表 |
| `--add-dir 目录` | 额外允许写入的目录，可重复 |
| `--dangerously-bypass-approvals-and-sandbox` | 不审批、不沙箱，直接执行一切命令。别名 `--yolo` |
| `--dangerously-bypass-hook-trust` | 本次运行跳过钩子的信任确认，只用于已审核过钩子的自动化 |

`-a` 在 0.142 的可选值：

| 值 | 含义 |
|---|---|
| `on-request` | 由模型判断什么时候需要请示你（默认，推荐） |
| `never` | 从不询问，失败直接交还给模型；适合脚本和 CI |
| `untrusted` | 只有 `ls`、`cat` 这类「安全」命令免审批，其余都问 |
| `on-failure` | 已废弃，不要再用 |

::: warning 不存在 --full-auto
很多老教程里的 `--full-auto` 已经被移除，0.142 会报 `unexpected argument`。想要「工作区内自动执行、越界才问」，直接写：

```bash
codex -s workspace-write -a on-request
```

最新源码里 `-a` 只保留 `on-request` 和 `never`，`untrusted` 不再接受；另外新增了 `--approve-for-me`，等价于「工作区可写 + 按需审批 + 由自动审查代理替你判断」。这些以你本机 `codex --help` 为准。
:::

`--dangerously-bypass-approvals-and-sandbox`（`--yolo`）让模型生成的命令不经任何限制直接在你的机器上跑。它只适合本身已经隔离的环境，比如一次性的 Docker 容器或 CI 虚拟机。细节见 [沙箱与审批](/codex/sandbox)。

### 输入与界面

| 参数 | 说明 | 示例 |
|---|---|---|
| `-C, --cd 目录` | 指定工作根目录 | `codex -C ./frontend` |
| `-i, --image 文件` | 给第一条消息附图，可多个，逗号分隔 | `codex -i ui.png,error.png "照着截图修样式"` |
| `--search` | 打开实时网页搜索（模型可直接调用 web_search，不逐次审批） | `codex --search` |
| `--no-alt-screen` | 不使用终端备用屏幕，保留滚动历史 | 在 tmux 或需要回看输出时 |
| `--remote 地址` | 把界面连到远程 app server | `--remote ws://host:port` |
| `-V, --version` | 显示版本号 | `codex -V` |

## -c：临时覆盖任何配置

`-c` 是最灵活的参数，`config.toml` 里能写的键，命令行都能临时改。规则只有三条：

1. 嵌套的键用点号连接：`sandbox_workspace_write.network_access`；
2. 等号右边按 **TOML** 解析，字符串要带引号，布尔值和数字不用；
3. 如果右边不是合法的 TOML，就当成普通字符串。

```bash
# 字符串：外层单引号保护内层双引号
codex -c 'model="gpt-5.5"' -c 'model_reasoning_effort="high"'

# 布尔值
codex -c sandbox_workspace_write.network_access=true

# 数组
codex -c 'sandbox_workspace_write.writable_roots=["/tmp/build"]'

# 嵌套表里的单个键
codex -c shell_environment_policy.inherit=all

# 临时关掉某个 MCP 服务
codex -c mcp_servers.github.enabled=false
```

::: details 为什么推荐外层用单引号
在 bash / zsh 里，`-c model="gpt-5.5"` 的双引号会被 shell 吃掉，Codex 收到的是 `model=gpt-5.5`。这种情况下它解析 TOML 失败，退回成字符串，结果碰巧也对；但遇到数组、带特殊字符的值就会出错。养成 `-c 'key="value"'` 的习惯最稳。PowerShell 里同样用单引号包住整个参数。
:::

命令行参数的优先级最高，会盖过 profile 和 `config.toml` 里的值，但只对这一次运行有效。完整的优先级顺序见 [config.toml 基础](/codex/config)。

## 子命令总览

| 子命令 | 作用 |
|---|---|
| `exec`（别名 `e`） | 非交互运行一个任务，适合脚本和 CI |
| `review` | 非交互地做一次代码审阅 |
| `resume` | 恢复之前的交互会话 |
| `fork` | 从之前的会话分叉出新会话 |
| `archive` / `unarchive` | 归档 / 取消归档会话 |
| `delete` | 永久删除会话 |
| `login` / `logout` | 登录、查看登录状态、退出 |
| `mcp` | 管理外部 MCP 服务 |
| `mcp-server` | 把 Codex 自己作为 MCP 服务运行（stdio） |
| `plugin` | 安装、列出、移除插件，管理插件市场 |
| `features` | 查看和切换功能开关 |
| `sandbox` | 在 Codex 的沙箱里运行任意命令，用来测试权限 |
| `apply`（别名 `a`） | 把云端任务生成的 diff 应用到本地 |
| `cloud` | 浏览、提交云端任务（实验性） |
| `app` | 打开桌面应用（未安装时打开安装页） |
| `doctor` | 诊断安装、配置、登录和运行环境 |
| `update` | 升级 Codex |
| `completion` | 生成 shell 补全脚本 |
| `debug` | 调试工具，例如输出模型目录 |
| `app-server`、`remote-control`、`exec-server` | 实验性的服务端组件，一般用户用不到 |

### exec：非交互执行

```bash
codex exec "把 src/utils 下的 var 全部改成 let/const，跑测试确认"
```

`exec` 做完任务就退出，进度打印到 stderr，最终回复打印到 stdout，所以能接管道。常用参数：

| 参数 | 说明 |
|---|---|
| `--json` | 以 JSONL 输出全部事件，便于程序解析 |
| `-o, --output-last-message 文件` | 把最后一条回复写入文件 |
| `--output-schema 文件` | 要求最终回复符合给定的 JSON Schema |
| `--skip-git-repo-check` | 允许在非 Git 仓库里运行 |
| `--ephemeral` | 不把会话记录写到磁盘 |
| `--ignore-user-config` | 不加载 `~/.codex/config.toml`（认证仍用 `CODEX_HOME`） |
| `--ignore-rules` | 不加载用户和项目的 execpolicy 规则 |
| `--color` | `always`、`never`、`auto` |

提示词可以从标准输入读：

```bash
# 用 - 表示从 stdin 读取整个提示词
cat task.md | codex exec -

# 同时给了提示词又有管道输入时，stdin 会作为附加内容追加
git diff | codex exec "用中文总结这些改动，列出风险点"

# 接着上一次 exec 的会话继续
codex exec resume --last "再补上单元测试"
```

`exec` 没有 `-a` 参数，它本来就不会停下来等人审批；权限靠 `-s` 控制。更多用法见 [非交互模式与 CI/CD](/codex/exec)。

### review：命令行里审阅代码

```bash
codex review --uncommitted                 # 审阅所有未提交的改动
codex review --base main                   # 审阅当前分支相对 main 的改动
codex review --commit a1b2c3d              # 审阅某个提交
codex review --base main "重点看并发安全"    # 带自定义要求
```

### resume 与 fork：继续以前的会话

```bash
codex resume                 # 打开选择器（默认只列出当前目录的会话）
codex resume --all           # 列出所有目录的会话
codex resume --last          # 直接接上最近一次
codex resume 会话ID或名称 "继续上次的重构"
codex fork --last            # 从最近一次会话分叉
```

会话名称就是在界面里用 `/rename` 起的名字。

### login / logout

```bash
codex login                          # 浏览器登录 ChatGPT 账号
codex login --device-auth            # 设备码登录，适合远程服务器
printenv OPENAI_API_KEY | codex login --with-api-key   # 用 API Key 登录，从 stdin 读取
codex login status                   # 查看当前登录状态
codex logout                         # 清除保存的凭据
```

用 HiveGPT 时不需要 `codex login`，配置文件里已经写好了 Key，见 [接入 HiveGPT](/codex/hivegpt)。

### mcp：管理 MCP 服务

```bash
codex mcp list                                   # 列出已配置的服务（加 --json 输出 JSON）
codex mcp add docs -- npx -y @upstash/context7-mcp   # 添加 stdio 服务，-- 后面是启动命令
codex mcp add docs --env API_KEY=xxx -- node server.js
codex mcp add linear --url https://mcp.linear.app/mcp  # 添加 HTTP 服务
codex mcp get docs                               # 查看某个服务的配置
codex mcp login linear                           # OAuth 登录
codex mcp remove docs
```

`add` 会写入 `~/.codex/config.toml` 的 `[mcp_servers.名称]`。HTTP 服务需要 token 时用 `--bearer-token-env-var 环境变量名`，不要把 token 明文写进命令。详见 [MCP](/codex/mcp)。

### features：功能开关

```bash
codex features list              # 列出全部功能及其阶段、当前状态
codex features enable memories   # 写入 config.toml，长期打开
codex features disable memories
```

`list` 的第二列是阶段：`stable` 稳定、`experimental` 实验、`under development` 开发中、`deprecated` 将移除、`removed` 已移除。只是临时试一下，用全局参数 `--enable 功能名` 更合适。

### sandbox：测试沙箱

```bash
codex sandbox -- ls ~                       # 在沙箱里跑一个命令
codex sandbox -- touch /etc/test            # 应该被拒绝
codex sandbox -P 配置名 -- npm test          # 指定权限配置
codex sandbox --log-denials -- python a.py  # macOS 上打印被沙箱拦截的操作
```

想知道某个命令在 Codex 里为什么失败，先用它在沙箱里单独跑一遍。原理见 [沙箱与审批](/codex/sandbox)。

### cloud 与 apply：云端任务

```bash
codex cloud                          # 打开云端任务浏览界面
codex cloud list                     # 列出任务
codex cloud exec --env 环境ID "修复 issue #42"   # 提交新任务
codex cloud diff 任务ID               # 查看 diff
codex cloud apply 任务ID              # 把结果应用到本地
```

`cloud` 依赖 ChatGPT 账号登录，标记为实验性。详见 [云端任务与 GitHub](/codex/cloud)。

### 其他实用子命令

```bash
codex doctor                 # 安装、配置、认证、网络一站式体检，报错先跑它
codex update                 # 升级到最新版
codex debug models           # 以 JSON 输出当前可用的模型目录
codex app .                  # 用桌面应用打开当前目录
codex completion zsh > ~/.zfunc/_codex   # 生成补全脚本，支持 bash/zsh/fish/powershell/elvish
```

## 常见组合

```bash
# 只读地让它分析代码，不允许任何改动
codex -s read-only "梳理一下支付模块的调用链"

# 默认的日常组合（大多数情况不写也一样）
codex -s workspace-write -a on-request

# 允许联网安装依赖
codex -c sandbox_workspace_write.network_access=true

# 前端在 web/，同时允许改共享的 packages/ui
codex -C web --add-dir ../packages/ui

# CI 里：不询问、工作区可写、输出 JSON
codex exec -s workspace-write --json "运行 lint 并修复能自动修的问题"

# 难题临时用高推理强度
codex -c 'model_reasoning_effort="xhigh"' "找出这个死锁的原因"
```

## 小结

- 全局参数管「这一次怎么跑」：`-m` 模型、`-s` 沙箱、`-a` 审批、`-C` 目录、`-c` 任意配置。
- `-c` 的值按 TOML 解析，养成 `-c 'key="value"'` 的写法。
- `--full-auto` 已移除；`--yolo` 是 `--dangerously-bypass-approvals-and-sandbox` 的别名，只在隔离环境用。
- `-p 名称` 现在读取 `~/.codex/名称.config.toml` 这份独立文件。
- 遇到问题先跑 `codex doctor`；不确定参数就加 `--help`。

下一步：[config.toml 基础](/codex/config)
