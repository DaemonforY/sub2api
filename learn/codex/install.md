---
title: 安装与登录
description: 用官方脚本、npm、Homebrew 或二进制安装 Codex CLI，学会升级、卸载、三种登录方式，以及看懂 ~/.codex 目录里的每个文件。
---

# 安装与登录

这一页把 Codex CLI 装到你的电脑上，并让它能调用模型。分三步：选一种方式安装 → 验证安装 → 选一种方式登录。最后介绍 `~/.codex` 目录，以后改配置、排查问题都绕不开它。

如果你打算用本站的 HiveGPT 接口，可以先完成安装，登录部分直接跳到 [接入 HiveGPT](/codex/hivegpt)。

## 系统要求

| 项目 | 要求 |
|---|---|
| 操作系统 | macOS 12 及以上；Ubuntu 20.04+ / Debian 10+ 等主流 Linux；Windows 11（原生或 WSL2） |
| 内存 | 至少 4 GB，建议 8 GB |
| Git | 可选但强烈建议，2.23 以上；看 diff、回退改动、`/review` 都依赖它 |
| Node.js | 只有用 npm 安装时才需要，16 以上，建议装当前的 LTS 版本 |

Codex CLI 本身是 Rust 编译的单个可执行文件，不依赖 Node.js 运行；npm 包只是帮你下载对应平台二进制的一层外壳。

## 选择安装方式

四种方式装出来的是同一个程序，选一种即可。不确定就按下表选：

| 方式 | 适合谁 | 升级方式 |
|---|---|---|
| 官方安装脚本 | 不想装 Node.js，想要最省事 | `codex update` 或重跑脚本 |
| npm | 已经有 Node.js 的前端/全栈开发者 | `codex update` 或 `npm i -g @openai/codex` |
| Homebrew | macOS 上习惯用 brew 管理软件 | `brew upgrade --cask codex` |
| GitHub Release 二进制 | 离线环境、服务器、要固定版本 | 手动下载替换 |

### 方式一：官方安装脚本

macOS / Linux：

```bash
curl -fsSL https://chatgpt.com/codex/install.sh | sh
```

Windows（PowerShell）：

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://chatgpt.com/codex/install.ps1 | iex"
```

脚本默认从 OpenAI 的发布服务器下载，下载不到时会自动退回 GitHub Releases。

::: warning 运行远程脚本前先看一眼
`curl ... | sh` 会直接执行下载到的脚本。上面是官方地址，可以放心；但养成习惯：对任何来源的一键脚本，先在浏览器里打开链接看看内容再运行。
:::

### 方式二：npm

```bash
npm install -g @openai/codex
```

国内网络下载慢，可以临时换用镜像源：

```bash
npm install -g @openai/codex --registry=https://registry.npmmirror.com
```

macOS / Linux 上如果报 `EACCES` 权限错误，不要用 `sudo npm`，而是用 nvm、fnm 这类版本管理器安装 Node.js，全局包就会装在你自己的用户目录下。pnpm、bun 用户用各自的全局安装命令也可以。

### 方式三：Homebrew（macOS）

```bash
brew install --cask codex
```

注意是 `--cask`。Homebrew 收录新版本通常会比 npm 晚一点。

### 方式四：GitHub Release 二进制

到 [github.com/openai/codex/releases](https://github.com/openai/codex/releases) 下载对应平台的压缩包：

| 平台 | 文件名 |
|---|---|
| macOS Apple 芯片 | `codex-aarch64-apple-darwin.tar.gz` |
| macOS Intel 芯片 | `codex-x86_64-apple-darwin.tar.gz` |
| Linux x86_64 | `codex-x86_64-unknown-linux-musl.tar.gz` |
| Linux arm64 | `codex-aarch64-unknown-linux-musl.tar.gz` |

解压后里面只有一个带平台名的文件，改名为 `codex` 放进 PATH：

```bash
tar -xzf codex-x86_64-unknown-linux-musl.tar.gz
mkdir -p ~/.local/bin
mv codex-x86_64-unknown-linux-musl ~/.local/bin/codex
chmod +x ~/.local/bin/codex
# 确认 ~/.local/bin 在 PATH 里，不在就把下面这行加到 ~/.bashrc 或 ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

### Windows 说明

Windows 上可以直接在 PowerShell 里用（安装脚本或 npm），Codex 有专门的 Windows 沙箱；也可以在 WSL2 里按 Linux 的方式安装，对依赖 Bash 的项目兼容性更好。两种方式的取舍、沙箱初始化和路径问题，见 [Windows 上使用](/codex/windows)。

### IDE 扩展和桌面端

- **IDE 扩展**：在 VS Code、Cursor、Windsurf 等编辑器的扩展市场搜索 Codex，认准 OpenAI 发布的官方扩展。详见 [IDE 扩展](/codex/ide)。
- **桌面端**：2026 年 7 月 10 日之后 Codex 桌面应用并入 ChatGPT 桌面端。已装好 CLI 的话，运行 `codex app` 会打开它，没装会引导你下载。详见 [桌面应用与界面](/codex/app)。

它们和 CLI 共用 `~/.codex` 里的配置和登录状态，下面的登录做一次就够。

## 验证安装

```bash
codex --version
```

输出类似 `codex-cli 0.142.3` 就说明装好了。如果提示「command not found」，多半是安装目录不在 PATH 里：npm 方式检查 `npm prefix -g` 输出的目录下的 `bin` 是否在 PATH 中；脚本和二进制方式重开一个终端再试。

装好后还可以跑一次自检，它会检查安装、配置、登录和运行环境：

```bash
codex doctor
```

## 升级与卸载

Codex 更新频繁，新功能和修复都靠升级获得。启动时如果有新版本，界面上会提示。手动升级：

```bash
codex update          # 自动识别你的安装方式并升级
```

也可以用安装时对应的工具升级：

```bash
npm install -g @openai/codex          # npm
brew upgrade --cask codex             # Homebrew
curl -fsSL https://chatgpt.com/codex/install.sh | sh   # 安装脚本，重跑即可
```

卸载：

```bash
npm uninstall -g @openai/codex        # npm 安装的
brew uninstall --cask codex           # Homebrew 安装的
rm ~/.local/bin/codex                 # 手动放置的二进制，按实际路径删除
```

卸载程序不会删除 `~/.codex` 目录。如果想彻底清理（包括登录凭据和会话记录），确认不再需要后手动删除这个目录。

## 登录：让 Codex 能调用模型

第一次运行 `codex`，如果配置里没有指定别的模型提供方，会出现登录界面，主要选项有：

| 选项 | 说明 | 计费 |
|---|---|---|
| Sign in with ChatGPT | 浏览器里登录 ChatGPT 账号 | 用 ChatGPT 套餐包含的额度 |
| Sign in with Device Code | 在另一台设备上用一次性代码登录 | 同上 |
| Provide your own API key | 填 OpenAI API Key | 按 API 用量付费 |

### 方式一：ChatGPT 账号

```bash
codex login
```

会打开浏览器完成登录，成功后凭据写入 `~/.codex/auth.json`。在没有浏览器的远程服务器上，用设备码方式：

```bash
codex login --device-auth
```

终端会显示一个网址和一次性代码，在任意一台有浏览器的设备上打开网址、输入代码即可。

### 方式二：OpenAI API Key

`codex login` 的 `--with-api-key` 参数从标准输入读取 Key，这样 Key 不会出现在命令历史里：

```bash
# 先把 Key 放进环境变量（只在当前终端有效）
export OPENAI_API_KEY="sk-..."
printenv OPENAI_API_KEY | codex login --with-api-key
```

PowerShell：

```powershell
$env:OPENAI_API_KEY = "sk-..."
$env:OPENAI_API_KEY | codex login --with-api-key
```

查看当前登录状态、退出登录：

```bash
codex login status
codex logout
```

### 方式三：HiveGPT 等兼容接口

如果你没有 ChatGPT 套餐或海外支付方式，或者需要国内网络直连，可以把 Codex 接到兼容 Responses API 的第三方接口。这种方式不走上面的登录界面，而是在 `config.toml` 里声明一个模型提供方，再把 Key 写进 `auth.json` 或配置里。本站推荐用 HiveGPT，完整步骤见 [接入 HiveGPT](/codex/hivegpt)；接 DeepSeek、本地模型等见 [接入 DeepSeek 等其他模型](/codex/providers)。

::: tip 凭据可以存进系统钥匙串
默认凭据以明文 JSON 存在 `~/.codex/auth.json`。在 `config.toml` 里设置 `cli_auth_credentials_store = "keyring"`（或 `"auto"`：钥匙串可用时用钥匙串，否则用文件），ChatGPT 登录的凭据就会存进 macOS 钥匙串、Windows 凭据管理器等系统服务。
:::

## ~/.codex 目录里有什么

Codex 的所有本地数据都放在 `~/.codex`（Windows 是 `%USERPROFILE%\.codex`）。可以用环境变量 `CODEX_HOME` 指定另一个目录，适合想完全隔离两套配置的场景。常见内容：

| 路径 | 作用 | 能不能手动改 |
|---|---|---|
| `config.toml` | 主配置：模型、提供方、沙箱、审批、MCP、功能开关 | 能，最常改的文件 |
| `名字.config.toml` | 配置档（profile），用 `codex -p 名字` 叠加到主配置之上 | 能，见 [高级配置](/codex/config-advanced) |
| `auth.json` | 登录凭据或 API Key | 能，但要当密码保管 |
| `AGENTS.md` | 对所有项目生效的个人指令 | 能 |
| `sessions/` | 会话记录（按日期分目录的 JSONL 文件），`codex resume` 从这里读 | 不建议 |
| `archived_sessions/` | 归档的会话 | 不建议 |
| `history.jsonl` | 输入框的历史记录（上下键翻的就是它） | 不建议 |
| `skills/` | 旧版个人 Skills 位置；新位置是 `~/.agents/skills/` | 能 |
| `rules/` | 命令执行规则（哪些命令允许、禁止、需确认） | 能，见 [规则与钩子](/codex/hooks) |
| `log/`、`*.sqlite` | 日志和内部状态数据库 | 不要动 |

其他目录（插件、缓存、记忆等）随启用的功能出现，不用管。

::: warning 不要把 ~/.codex 整个同步或分享
`auth.json` 里是你的凭据，`sessions/` 里有你所有对话和代码片段。用网盘、dotfiles 仓库同步配置时，只同步 `config.toml`、`AGENTS.md` 和 Skills，排除其他内容。
:::

## 小结

- 四种安装方式任选：官方脚本、`npm install -g @openai/codex`、`brew install --cask codex`、GitHub Release 二进制。
- 用 `codex --version` 验证安装，`codex doctor` 做自检，`codex update` 升级。
- 登录可选 ChatGPT 账号（`codex login`，无浏览器用 `--device-auth`）、OpenAI API Key（`--with-api-key` 从标准输入读）或 HiveGPT 等兼容接口。
- 配置、凭据、会话都在 `~/.codex`；`auth.json` 和 `sessions/` 不要外传。

下一步：[接入 HiveGPT](/codex/hivegpt)
