---
title: 常见问题排查
description: 按症状排查 Codex 的常见问题：安装、登录、HiveGPT 接口报错（401/403/429/余额/模型不可用/流中断）、模型列表、沙箱、Windows、上下文、速度、MCP、配置不生效，以及怎么看日志和反馈。
---

# 常见问题排查

这一页按「看到了什么现象」来组织。每一条都写成 **现象 → 原因 → 解决**，先找到和你最像的症状，再按步骤处理。大部分问题都能用下面三个命令先缩小范围：

```bash
codex --version        # 版本太旧是很多问题的根源
codex doctor           # 自检安装、配置、认证、运行环境
codex debug models     # 看 Codex 实际加载到的模型目录
```

在 TUI 里，`/status` 看当前使用的模型、provider、沙箱和 token 用量，`/debug-config` 看每个配置项来自哪一层。

## 一、安装

### npm 安装报 EACCES 权限错误

**现象**：`npm install -g @openai/codex` 报 `EACCES: permission denied, mkdir '/usr/local/lib/node_modules/...'`。

**原因**：Node.js 是用系统安装包装的，全局目录属于 root。

**解决**：不要用 `sudo npm install -g`，那会让以后的升级和卸载都要 sudo，还可能把缓存目录的权限搞乱。推荐改用 nvm 管理 Node，全局目录就在你自己的家目录里：

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
# 重开终端后
nvm install --lts
npm install -g @openai/codex
```

不想装 nvm 的话，也可以把 npm 的全局目录改到家目录：

```bash
mkdir -p ~/.npm-global
npm config set prefix ~/.npm-global
echo 'export PATH="$HOME/.npm-global/bin:$PATH"' >> ~/.zshrc   # bash 用户改 ~/.bashrc
source ~/.zshrc
npm install -g @openai/codex
```

macOS 用户也可以直接 `brew install --cask codex`。

### Node.js 版本太低

**现象**：安装或启动时报语法错误，或者提示 Node 版本不受支持。

**原因**：`@openai/codex` 包声明的最低版本是 Node 16，但更老的系统自带 Node 常常更低；而 TypeScript SDK 等周边工具要求 18 以上。

**解决**：`node -v` 查看版本，升级到当前的 LTS 版本（20 或 22）。用 nvm 的话 `nvm install --lts && nvm alias default 'lts/*'`。

### 下载很慢或超时

**现象**：npm 安装卡住、`ETIMEDOUT`、`ECONNRESET`。

**原因**：访问 npm 官方源的网络不稳定。Codex 的 npm 包会再按平台拉取一个几十 MB 的二进制包（如 `@openai/codex-darwin-arm64`），所以比一般的包更容易超时。

**解决**：临时换国内镜像安装：

```bash
npm install -g @openai/codex --registry=https://registry.npmmirror.com
```

需要代理的环境，给 npm 设代理：`npm config set proxy http://127.0.0.1:7890` 和 `npm config set https-proxy http://127.0.0.1:7890`（端口换成你自己的）。

### 装好了却提示 command not found

**现象**：安装成功，但运行 `codex` 提示找不到命令。

**原因**：npm 全局 bin 目录不在 `PATH` 里，或者装在了另一个 Node 版本下（nvm 切换过版本）。

**解决**：

```bash
npm prefix -g          # 全局目录，bin 在它下面
which -a codex         # 看看有没有、有几个
nvm ls                 # 用 nvm 时确认当前版本就是安装时的版本
```

如果 `which -a` 显示多个 codex（比如一个 npm 装的、一个 brew 装的），删掉不用的那个，避免升级了一个、运行的是另一个。

## 二、登录（ChatGPT 账号）

使用 HiveGPT Key 时不需要 `codex login`，可以跳过这一节。

| 现象 | 原因 | 解决 |
|---|---|---|
| 浏览器登录后终端一直等待 | 登录回调要访问本机 `localhost:1455`，端口被占用或被安全软件拦截 | 关掉占用端口的程序后重试；或改用 `codex login --device-auth` 设备码登录 |
| 在 SSH 远程机器上无法打开浏览器 | 远程机器没有浏览器，回调地址在远程机器上 | 用 `codex login --device-auth`；或在本机做 SSH 端口转发 `ssh -L 1455:localhost:1455 user@host` 后再登录 |
| 登录成功，但请求仍用旧的 Key | `auth.json` 里同时有旧数据，或配置里指定了别的 provider | `codex login status` 查看当前认证；检查 `config.toml` 的 `model_provider` |
| 想切换账号 | | `codex logout` 后重新 `codex login` |

登录相关日志会写到 `~/.codex/log/codex-login.log`，排查时可以看这里。

## 三、使用 HiveGPT 时的接口报错

Codex 遇到接口错误时通常会显示 `unexpected status 401 Unauthorized: …` 这样的信息，冒号后面是 HiveGPT 返回的原文。HiveGPT 的错误提示是中文加英文括号，照着提示处理即可。下面是常见的几类：

| 状态码 / 提示 | 原因 | 解决 |
|---|---|---|
| **401**「API Key 无效或已被删除（Invalid API key）」 | Key 抄错、多了空格、已删除 | 在「API 密钥」页重新点「使用密钥」生成配置，覆盖 `auth.json`；确认没有其他地方的 Key 生效（见下方 provider 说明） |
| **401**「缺少 API Key（API key is required…）」 | Codex 没有带上 Key | 检查 `auth.json` 是否存在、内容是不是 `{"OPENAI_API_KEY": "…"}`；用 `env_key` 时检查环境变量是否真的导出了 |
| **403**「API Key 已停用」「所属分组已停用 / 已删除」 | Key 或它所在的分组不可用 | 到控制台启用 Key，或换一个正常分组的 Key |
| **403**「没有该分组的有效订阅（No active subscription…）」 | 用的是订阅分组的 Key，但订阅过期 | 续费，或换按量分组（如「GPT-按量」）的 Key |
| **403**「账户余额不足（Insufficient account balance）」 | 按量分组余额用完 | 控制台充值，或改用有效订阅分组的 Key |
| **429**「并发上限」「排队的请求太多」 | 同时开的会话、子代理太多，超过 Key 或账号的并发限制 | 减少并行会话；等前面的任务完成；必要时调整 Key 的并发设置 |
| **429**「上游模型服务限流中」 | 上游临时限流 | 等 1–2 分钟重试；Codex 会自动重试几次（`request_max_retries` 默认 4） |
| **503**「暂时没有可处理…请求的上游账号」/ `no available account` | 账号都在忙，或这个分组**不支持你选的模型** | 先 `/model` 换成列表里的 gpt-5.5 试试；确认 Key 属于 OpenAI 类分组；仍不行稍后再试 |
| 「不支持模型 xxx（Model … is not supported…）」 | 配置里的 `model` 写了分组没有的模型 | 把 `model` 改成分组支持的模型，模型列表以 HiveGPT 控制台和 `/model` 为准 |
| 502 / 503「上游服务暂时不可用 / 过载」 | 上游故障 | 稍后重试；这类错误 HiveGPT 不扣费 |

::: tip 先确认请求真的发到了 HiveGPT
`/status` 里看 provider 和 base URL。HiveGPT 生成的配置里，`model_provider = "OpenAI"` 指向的是 `config.toml` 里自定义的 `[model_providers.OpenAI]` 块，`base_url` 应该是 `https://hivegpt.cn/v1`。如果这一块被删掉，Codex 会报 `Model provider ... not found`；如果 `model_provider` 这一行被写到了别的 `[表]` 下面，它就不再是顶层设置，Codex 会使用内置的 OpenAI 官方接口，结果是莫名其妙的 401。这也是弹窗提示「把内容放在 config.toml 开头」的原因：TOML 里写在 `[某个表]` 之后的键都属于那个表。
:::

### stream disconnected / Reconnecting…

**现象**：回复到一半出现 `Reconnecting... 1/5`，或者最终报 `stream disconnected before completion`。

**原因**：流式响应中途断开。可能是本地网络抖动、代理超时、上游生成时间很长中间没有数据。Codex 默认在流断开时重试 5 次（`stream_max_retries`），流空闲超过 300 秒（`stream_idle_timeout_ms`）判为断开。

**解决**：

1. 偶发的，等它自动重连，或者在输入框里说「继续」。
2. 频繁出现，先排查网络和代理：走代理的话，确认代理对长连接没有过短的超时。
3. 可以在 provider 配置里调大重试次数和空闲超时：

```toml
[model_providers.OpenAI]
# …原有的 name / base_url / wire_api 等保持不变
stream_max_retries = 10
stream_idle_timeout_ms = 600000
```

4. 推理强度设得很高时，模型思考时间长，更容易触发空闲超时，可以适当调低。

### wire_api 配错

**现象**：启动就报 `` `wire_api = "chat"` is no longer supported ``，或者 `unknown variant`。

**原因**：Codex 已经移除了 Chat Completions 接口，`wire_api` 只支持 `"responses"`。一些旧教程、给其他工具写的配置里还是 `chat`。

**解决**：改成 `wire_api = "responses"`。HiveGPT 支持 Responses 接口，弹窗生成的配置已经是正确的。接入第三方模型时，确认对方提供 Responses 兼容接口，见 [接入 DeepSeek 等其他模型](/codex/providers)。

## 四、模型列表不对

**现象**：`/model` 里看不到 gpt-5.5，或者列出的是一堆你的分组里没有的模型；或者启动时有警告 `Model metadata for 'xxx' not found. Defaulting to fallback metadata`。

**原因**：Codex 用「模型目录」决定能选哪些模型、每个模型的上下文长度和支持的推理强度。HiveGPT 的配置用 `model_catalog_json` 指向本地的 `codex-models.json`。文件不存在、路径写错或内容损坏，Codex 就会退回内置目录，或者把你的模型当成未知模型、用保守的默认参数（上下文可能被算小，导致过早压缩）。

**解决**：

1. 确认文件存在：`ls -l ~/.codex/codex-models.json`。没有的话，在「使用密钥」弹窗里重新下载，放到 `~/.codex/`。
2. 确认 `config.toml` 里 `model_catalog_json` 的路径和文件实际位置一致。Windows 上路径写法见下面的 Windows 一节。
3. 运行 `codex debug models`，输出里应该能看到 gpt-5.5。
4. HiveGPT 上线新模型后，重新下载一次这个文件即可。

## 五、沙箱拒绝命令

**现象**：Codex 执行的命令报 `Operation not permitted`、`Read-only file system`，或者 `npm install` 联网失败；`git commit` 总是失败或要审批。

**原因**：这是沙箱在起作用。

| 被拒的操作 | 原因 | 解决 |
|---|---|---|
| 写文件 | 当前是只读沙箱 | `/permissions` 切到 Default，或启动时 `-s workspace-write` |
| 写工作区外的目录 | workspace-write 只允许写工作区和临时目录 | 启动时 `--add-dir /path`，或配置 `sandbox_workspace_write.writable_roots` |
| 联网（装依赖、调接口） | workspace-write 默认禁网 | 批准它的联网请求；或 `-c sandbox_workspace_write.network_access=true` |
| `git commit`、改 `.git/` | `.git`、`.codex` 在可写模式下也保持只读 | 自己提交；或在审批弹窗中同意 |
| `codex exec` 里任何需要审批的操作 | exec 从不询问，需要审批的直接判失败 | 一开始就给足权限（`-s workspace-write` 等），或调整任务 |

想知道某条命令在沙箱里为什么失败，可以用 `codex sandbox` 单独复现。macOS 上加 `--log-denials` 会打印出被系统沙箱拒绝的具体操作：

```bash
codex sandbox --log-denials -- npm install
```

**Linux 特有**：Codex 在 Linux 上用 bubblewrap 做沙箱，需要能创建用户命名空间。在部分容器、加固过的内核或 WSL1 上做不到，Codex 启动时会给出警告，沙箱内的命令全部失败。解决办法是在宿主上允许非特权用户命名空间、换用 WSL2；在一次性容器里也可以用 `--dangerously-bypass-approvals-and-sandbox`，让容器本身充当隔离。

沙箱的完整说明见 [沙箱与审批](/codex/sandbox)。

## 六、Windows 特有问题

| 现象 | 原因 | 解决 |
|---|---|---|
| PowerShell 报「无法加载文件 codex.ps1，因为在此系统上禁止运行脚本」 | PowerShell 执行策略禁止运行 npm 生成的脚本 | 用 `codex.cmd` 运行；或由你自己决定是否执行 `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` |
| 配置文件不知道放哪 | Windows 的 CODEX_HOME 是 `%USERPROFILE%\.codex` | 在 PowerShell 里 `notepad $env:USERPROFILE\.codex\config.toml` |
| `model_catalog_json` 路径不生效 | TOML 字符串里的反斜杠是转义符 | 用正斜杠 `C:/Users/you/.codex/codex-models.json`，或单引号字面量 `'C:\Users\you\.codex\codex-models.json'` |
| 命令在 WSL 和 Windows 之间行为不一致 | 两边是两套独立的 Codex 安装和 `~/.codex` | 在哪边用就在哪边配置；WSL 里的项目放在 Linux 文件系统（`~/`）下，速度快得多 |
| 沙箱相关报错 | 原生 Windows 沙箱需要初始化 | 在 TUI 中运行 `/setup-default-sandbox`；`[windows] sandbox` 可选 `elevated` / `unelevated` |
| 中文乱码 | 终端代码页不是 UTF-8 | 使用 Windows Terminal；或在 PowerShell 中先执行 `chcp 65001` |

更多见 [Windows 上使用](/codex/windows)。

## 七、上下文太长

**现象**：报 `Codex ran out of room in the model's context window`，或者长对话后回答明显变差、开始忘记前面说过的约束。

**原因**：对话历史、读过的文件、命令输出都会占用上下文。上下文满了之后，模型要么无法继续，要么只能依赖压缩后的摘要。

**解决**：

- 手动 `/compact` 压缩对话，然后继续。Codex 也会在接近上限时自动压缩（`model_auto_compact_token_limit` 可以调整触发点）。
- 一个任务做完就 `/new` 开始新对话，不要在一个会话里连做五件事。
- 提示词里限定范围：「只看 src/billing」比「看看整个项目」省得多。
- 让它别把大文件整个输出：「只读前 50 行」「用 rg 搜索，不要 cat 整个日志」。
- `AGENTS.md` 写得太长也会占上下文，默认最多读取 32 KiB（`project_doc_max_bytes`）。只写真正需要每次都知道的规则。
- 确认模型目录加载正确（见第四节），否则 Codex 可能把上下文窗口算小了。

## 八、响应慢

| 可能的原因 | 判断方法 | 解决 |
|---|---|---|
| 推理强度太高 | `/status` 或状态栏查看推理强度 | 简单任务用 low / medium，`Alt+,` 随时调低 |
| 任务太大太散 | 一直在读文件、跑命令 | 拆成小任务，给出明确的文件范围 |
| 测试或构建命令本身慢 | 时间花在 `npm test` 等命令上 | 告诉它只跑相关的测试：「只运行 src/cart 下的测试」 |
| 网络或代理慢 | 频繁出现 Reconnecting | 检查网络、代理设置 |
| 上游繁忙 | 同一时间段普遍变慢 | 稍后重试，或在 `/model` 中换一个模型 |
| MCP 服务器启动慢 | 每次启动都要等很久 | 见下一节，禁用不用的 MCP 服务器 |

## 九、MCP 服务器起不来

**现象**：启动时提示某个 MCP 服务器启动失败或超时；`/mcp` 里看不到它的工具。

**排查步骤**：

1. `codex mcp list` 确认配置被读到了，`codex mcp get <名字>` 查看完整配置。
2. **把配置里的命令单独在终端里运行一次**，比如 `npx -y @upstash/context7-mcp`。大部分问题在这一步就能看到真正的报错：包名写错、Node 版本不够、缺少环境变量。
3. 首次运行 `npx -y …` 要下载包，可能超过启动超时（默认 30 秒）。先手动运行一次让它缓存，或调大超时：

```toml
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]
startup_timeout_sec = 60
tool_timeout_sec = 120
```

4. 从桌面应用、IDE 插件启动时，`PATH` 可能和终端里不同，找不到 `npx`、`uvx`。把 `command` 写成绝对路径（`which npx` 查看）。
5. 需要的 Key 通过 `env = { … }` 或 `codex mcp add --env` 传入。按源码，Codex 启动本地 MCP 服务器时只传 `HOME`、`PATH`、`SHELL`、`USER` 等少数基础变量和你显式配置的变量，shell 里 export 的其他变量不会自动继承。
6. 远程服务器（`url = …`）需要登录的，运行 `codex mcp login <名字>`。
7. 暂时用不到的服务器设 `enabled = false`，加快启动速度。

详见 [MCP](/codex/mcp)。

## 十、配置不生效

**现象**：改了 `config.toml`，但 Codex 的行为没变。

**原因和解决**，按出现频率排序：

1. **改错了文件**。Codex 读的是 `$CODEX_HOME/config.toml`，没设 `CODEX_HOME` 时是 `~/.codex/config.toml`。`echo $CODEX_HOME` 确认一下。
2. **键写在了错误的表下面**。TOML 里，写在 `[model_providers.OpenAI]` 之后的 `model = "…"` 属于那个表，而不是顶层。顶层的键必须放在文件开头、第一个 `[表]` 之前。
3. **拼错了键名**。Codex 默认会忽略不认识的键，不报错。用 `codex --strict-config` 启动一次，拼错的键会直接报出来。
4. **被更高优先级的层覆盖了**。配置按层叠加，后面的覆盖前面的：

| 优先级（低 → 高） | 来源 |
|---|---|
| 1 | 系统配置 `/etc/codex/config.toml` 等 |
| 2 | 用户配置 `~/.codex/config.toml` |
| 3 | 配置档 `~/.codex/<名字>.config.toml`（用 `-p <名字>` 时） |
| 4 | 项目配置 `<项目>/.codex/config.toml`（项目被信任时才生效） |
| 5 | 命令行 `-c key=value`、`-m`、`-s` 等，以及会话中通过 `/model` 等界面的选择 |

在 TUI 里运行 `/debug-config` 可以直接看到每一层的内容和最终生效的值。

5. **项目配置没生效**：`<项目>/.codex/config.toml` 只在项目被信任时加载。第一次在项目里启动 Codex 时会询问是否信任；也可以在用户配置里加 `[projects."/项目的绝对路径"]` 和 `trust_level = "trusted"`。
6. **`-p` 报错**：新版本的 `-p <名字>` 读取的是独立文件 `~/.codex/<名字>.config.toml`。如果主配置里同时存在旧写法 `[profiles.<名字>]` 或 `profile = "<名字>"`，会直接报错，提示你把设置挪到独立文件里。按提示迁移即可。
7. **`-c` 的值没按预期解析**：值按 TOML 解析，字符串要带引号且外面再套一层 shell 引号，例如 `-c 'model="gpt-5.5"'`、`-c model_reasoning_effort='"high"'`。解析失败时会被当作普通字符串。
8. **会话里改过设置**：`/model`、`/permissions` 的选择在当前会话中优先于配置文件。重开会话再看。

## 怎么看日志

| 场景 | 方法 |
|---|---|
| 交互模式的详细日志 | 默认写入 `~/.codex/` 下的日志数据库（`logs_*.sqlite`），`/feedback` 会把相关日志一并上传 |
| 想要纯文本日志文件 | 显式指定 `log_dir`，此时才会写 `codex-tui.log`：`codex -c log_dir=/tmp/codex-log`，然后 `tail -F /tmp/codex-log/codex-tui.log` |
| 调整日志详细程度 | 用环境变量 `RUST_LOG`，例如 `RUST_LOG=debug codex -c log_dir=/tmp/codex-log` |
| `codex exec` 的日志 | 直接打到 stderr，默认只有 error 级别；需要更多时 `RUST_LOG=info codex exec …` |
| 登录日志 | `~/.codex/log/codex-login.log` |
| 某次会话的完整记录 | `~/.codex/sessions/` 下按日期存放的 `.jsonl` 文件 |
| 当前用的是什么配置 | `/status`、`/debug-config`、`codex doctor` |

::: warning 旧教程里的日志位置
很多旧教程说日志在 `~/.codex/log/codex-tui.log`。按官方当前源码，这个文件只有在显式设置 `log_dir` 时才会写入，旧的同名文件甚至会在启动时被清理掉。找不到这个文件是正常的，按上表的方法打开即可。
:::

分享日志前，检查一下里面有没有 Key、内部地址或代码片段。

## 怎么反馈问题

1. **Codex 本身的问题**（崩溃、界面异常、命令行为不对）：在 TUI 中运行 `/feedback`，它会把日志发送给 Codex 维护者；也可以到 [openai/codex 的 GitHub Issues](https://github.com/openai/codex/issues) 搜索或提交，附上 `codex --version`、操作系统、复现步骤和 `codex doctor --json` 的输出。
2. **HiveGPT 接口的问题**（报错提示看不懂、扣费疑问、模型不可用持续很久）：联系 HiveGPT 客服，提供出错的时间、使用的 Key 名称（不要发 Key 本身）、模型名和完整的错误信息。
3. **本教程的问题**：页面内容和你看到的不一致，多半是版本差异。先 `codex update` 升级到最新版本再对照。

## 小结

- 先跑 `codex --version`、`codex doctor`，在会话里用 `/status`、`/debug-config`，能快速定位大部分问题。
- 安装问题多是权限、Node 版本和网络：用 nvm、升级到 LTS、换镜像。
- HiveGPT 报错照着中文提示处理：401 查 Key，403 查分组和订阅，余额不足去充值，429 减少并发，`no available account` 先换成分组支持的模型。
- 模型列表不对就重新下载 `codex-models.json` 并核对 `model_catalog_json` 路径；`wire_api` 只能是 `responses`。
- 配置不生效，多半是改错文件、键放错了表、被更高层覆盖或项目未被信任；纯文本日志需要显式设置 `log_dir`。

下一步：回到 [速查表](/codex/cheatsheet)，或从 [Codex 是什么](/codex/intro) 重新系统地过一遍。
