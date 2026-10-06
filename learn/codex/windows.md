---
title: Windows 上使用
description: 在 Windows 上用 Codex 的两条路线——原生 PowerShell 和 WSL2——怎么选、怎么装，Windows 沙箱的几种模式，路径、换行符、中文编码问题，以及常见故障的排查办法。
---

# Windows 上使用

Codex 早期只在 macOS 和 Linux 上有完整体验，Windows 用户基本都靠 WSL。现在 Codex 已经能在 Windows 上原生运行：有原生的安装脚本、原生的沙箱，桌面应用也有 Windows 版。但 WSL2 依然是很多场景下更稳的选择。

这一页帮你选路线、装好、把沙箱配对，并解决 Windows 上特有的路径、换行符和中文乱码问题。

## 先选路线：原生还是 WSL2

| 对比项 | 原生 Windows | WSL2 |
|---|---|---|
| 适合的项目 | .NET、PowerShell 脚本、Windows 桌面程序、依赖 Windows 工具链的项目 | Web 后端、Node.js / Python / Go 项目、部署目标是 Linux 的项目 |
| Codex 执行命令用的 shell | PowerShell | bash / zsh |
| 沙箱 | Windows 专用沙箱（需要一次性管理员授权，见下文） | Linux 沙箱（bubblewrap），与在 Linux 上一致 |
| 和教程、开源项目脚本的兼容性 | 一般，很多脚本默认是 bash | 好 |
| 桌面应用、IDE 扩展 | 直接可用 | IDE 可通过 Remote - WSL 连接 |
| 文件放哪里 | Windows 磁盘 | WSL 自己的 Linux 文件系统里（`~/code`） |

一个简单的判断：**项目最后跑在 Linux 服务器上，就用 WSL2；项目本身是 Windows 生态的，就用原生。** 拿不准的话，先用原生试试，遇到大量 bash 脚本跑不通再换 WSL2。

## 路线一：原生 Windows

### 安装 CLI

在 PowerShell 里运行官方安装脚本：

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://chatgpt.com/codex/install.ps1 | iex"
```

已经装了 Node.js 的话，也可以用 npm：

```powershell
npm install -g @openai/codex
```

装完新开一个终端窗口确认：

```powershell
codex --version
codex doctor     # 检查安装、配置、登录和运行环境
```

接入 HiveGPT 的步骤和其他系统一样，见 [接入 HiveGPT](/codex/hivegpt)。唯一的区别是配置目录：

| 文件 | Windows 上的位置 |
|---|---|
| 用户配置 | `%USERPROFILE%\.codex\config.toml`，即 `C:\Users\你的用户名\.codex\config.toml` |
| 凭据 | `%USERPROFILE%\.codex\auth.json` |
| 会话记录 | `%USERPROFILE%\.codex\sessions\` |
| 系统级配置（管理员下发） | `%ProgramData%\OpenAI\Codex\config.toml` |
| 强制策略（管理员下发） | `%ProgramData%\OpenAI\Codex\requirements.toml` |

快速打开配置文件：

```powershell
notepad $env:USERPROFILE\.codex\config.toml
```

::: tip 在资源管理器里看不到 .codex？
以点开头的文件夹在 Windows 上不会被隐藏，但你可能找错了用户目录。直接在地址栏输入 `%USERPROFILE%\.codex` 回车即可。如果设置过 `CODEX_HOME` 环境变量，配置目录以它为准。
:::

### 桌面应用

Windows 版桌面应用通过 Microsoft Store 分发。在 PowerShell 里运行 `codex app` 会打开商店的安装页，已安装则直接打开并定位到当前目录。用法见 [桌面应用与界面](/codex/app)。

### 选一个好用的终端

推荐 **Windows Terminal + PowerShell 7**：

- 对 Unicode、颜色、全屏界面的支持比老式控制台好得多，Codex 的界面不容易错位。
- PowerShell 7（`pwsh`）默认 UTF-8，中文乱码问题少很多。

```powershell
winget install Microsoft.PowerShell
```

尽量不要在 PowerShell ISE 或很老的 `cmd.exe` 窗口里跑 Codex 的交互界面。

## Windows 沙箱

Codex 在执行命令时会套一层沙箱，限制它能写哪些文件、能不能联网（概念见 [沙箱与审批](/codex/sandbox)）。macOS 用系统自带的 Seatbelt，Linux 用 bubblewrap，Windows 则有自己的实现，在 `config.toml` 的 `[windows]` 段配置：

```toml
[windows]
sandbox = "elevated"   # 可选 elevated / unelevated / mxc
```

| 模式 | 原理 | 需要管理员吗 | 说明 |
|---|---|---|---|
| `elevated`（默认沙箱） | 创建专用的本地沙箱账户运行命令，并用防火墙规则控制联网 | 首次设置需要一次 | 隔离最完整，官方推荐 |
| `unelevated`（非管理员沙箱） | 用受限令牌运行命令 | 不需要 | 隔离较弱，界面上明确提示「被提示注入时风险更高」 |
| `mxc` | 基于微软新的容器隔离机制 | 不需要 | 较新，依赖系统支持；目前默认不启用，以官方说明为准 |

第一次在原生 Windows 上启动 Codex 时，它会提示设置沙箱：

- 选「设置默认沙箱（需要管理员权限）」，系统弹出 UAC 授权，设置过程可能需要几分钟。之后日常使用不需要管理员权限。
- 如果公司电脑拿不到管理员权限，可以选「使用非管理员沙箱」，代价是隔离更弱。

错过了提示也没关系，之后在 CLI 里输入 `/setup-default-sandbox` 可以重新设置。

::: warning 注意这两个误区
- **「非管理员沙箱更安全，因为它不需要管理员」是错的。** 需不需要管理员说的是设置过程，不是沙箱强度。默认沙箱需要管理员，正是因为它要创建独立账户和防火墙规则，隔离反而更彻底。
- **不要为了省事用管理员身份运行 Codex。** 以管理员身份启动终端再跑 Codex，等于把管理员权限交给了代理执行的每一条命令。
:::

::: details 给管理员：批量部署时预先设置沙箱
在统一管理的电脑上，可以用管理员身份预先完成沙箱设置，用户第一次启动就不会再弹窗：

```powershell
# 为当前用户设置默认沙箱
codex sandbox setup --elevated --current-user

# 为指定用户设置（需同时指定该用户的 CODEX_HOME）
codex sandbox setup --elevated --user alice --codex-home C:\Users\alice\.codex
```

还可以在 `%ProgramData%\OpenAI\Codex\requirements.toml` 里限制允许的沙箱实现，见 [安全与团队管理](/codex/security)。需要清理旧版沙箱创建的账户和防火墙规则时，以管理员身份运行 `codex sandbox uninstall`。
:::

想验证某条命令在沙箱里的表现，可以用 `codex sandbox` 直接在沙箱中运行它：

```powershell
codex sandbox -- powershell -Command "New-Item C:\test.txt"
```

## 路线二：WSL2

### 安装 WSL 和 Linux 发行版

以管理员身份打开 PowerShell：

```powershell
wsl --install            # 默认安装 Ubuntu
wsl --set-default-version 2
wsl -l -v                # 确认 VERSION 一列是 2
```

重启电脑后从开始菜单打开 Ubuntu，设置 Linux 用户名和密码。

::: warning 必须是 WSL2
WSL1 无法创建 Codex 的 Linux 沙箱所需的用户命名空间，Codex 会拒绝在沙箱中执行命令。`wsl -l -v` 里显示 1 的话，用 `wsl --set-version Ubuntu 2` 转换。
:::

### 在 WSL 里装 Node.js 和 Codex

建议用 nvm 管理 Node.js，避免和 Windows 上装的 Node 混在一起。按 nvm 项目说明装好 nvm 后：

```bash
source ~/.bashrc
nvm install --lts
npm install -g @openai/codex
```

也可以不装 Node，直接用官方安装脚本：

```bash
curl -fsSL https://chatgpt.com/codex/install.sh | sh
```

装好后在 WSL 里按 [接入 HiveGPT](/codex/hivegpt) 配置，配置文件在 WSL 的 `~/.codex/`，和 Windows 那边的 `%USERPROFILE%\.codex` 是**两份独立的配置**，两边都用就要各配一次。

### 把项目放在 Linux 文件系统里

WSL 能通过 `/mnt/c/...` 访问 Windows 磁盘，但跨文件系统读写很慢。Codex 一次任务会读大量文件、跑 `git status`、搜索代码，放在 `/mnt/c` 下会明显卡顿，`npm install` 也可能慢上数倍。

```bash
# 推荐：项目放在 WSL 的家目录
mkdir -p ~/code && cd ~/code
git clone https://github.com/you/your-project.git
cd your-project && codex
```

需要在 Windows 侧查看这些文件时，资源管理器地址栏输入 `\\wsl$` 即可。用 VS Code 的话，在 WSL 里运行 `code .` 会通过 Remote - WSL 打开，Codex 扩展运行在 WSL 一侧。

另外，`CODEX_HOME` 不要设到 `/mnt/c` 下，Windows 磁盘不支持 Codex 后台进程需要的 Unix 权限语义，可能导致启动失败。

## 路径、换行符和编码

### 路径

- 原生 Windows 下，给 Codex 的路径用 `C:\Users\me\project` 或 `C:/Users/me/project` 都可以。
- WSL 里 Windows 路径要换成 `/mnt/c/Users/me/project`。
- 路径里有空格或中文时，命令里记得加引号。Codex 写的命令偶尔会漏，审批时留意。

### 换行符

Windows 默认 CRLF，Linux 和大多数开源项目用 LF。Codex 改文件时一般会保持原文件的换行风格，但混用 Git 的自动转换容易出现「整个文件都被改了」的巨大 diff。建议在仓库里用 `.gitattributes` 固定下来：

```text
* text=auto eol=lf
*.ps1 text eol=crlf
*.bat text eol=crlf
*.cmd text eol=crlf
```

并检查全局设置，避免和 `.gitattributes` 冲突：

```powershell
git config --global core.autocrlf false
```

### 中文乱码

症状：命令输出里的中文变成问号或乱码，Codex 据此误判命令结果。处理办法：

- 换用 PowerShell 7 和 Windows Terminal（最有效）。
- 在 Windows PowerShell 5.1 里临时切到 UTF-8：

```powershell
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
```

- 让 Codex 读写的源文件统一用 UTF-8（无 BOM）保存，可以写进项目的 `AGENTS.md`：「所有文本文件使用 UTF-8 编码，不加 BOM」。

## 常见问题

**`codex` 提示不是内部或外部命令？**
安装后需要新开终端窗口让 PATH 生效。用 npm 安装的，确认 `npm prefix -g` 输出的目录在 PATH 里。

**安装脚本被执行策略拦住？**
用上面带 `-ExecutionPolicy ByPass` 的完整命令，它只对这一次执行生效，不会改系统设置。

**设置默认沙箱失败？**
常见原因是没有管理员权限、或安全软件拦截了账户创建。可以重试，或暂时选非管理员沙箱。公司电脑请联系 IT 用上面「给管理员」的命令统一部署。

**代理 / 公司网络下连不上？**
在启动 Codex 的同一个终端里设置代理环境变量：

```powershell
$env:HTTPS_PROXY = "http://127.0.0.1:7890"
codex
```

**怎么看详细日志？**
Codex 默认把诊断信息记在本地的有界存储里，不再写一个持续增长的文本日志。需要文本日志时，为这次运行指定目录：

```powershell
codex -c log_dir=.\.codex-log
Get-Content .\.codex-log\codex-tui.log -Wait
```

先跑 `codex doctor` 往往就能定位安装、配置或登录问题。

## 小结

- 项目跑在 Linux 上选 WSL2，Windows 生态项目选原生；两边的 `~/.codex` 配置互相独立。
- 原生安装用官方 PowerShell 脚本或 npm，配置在 `%USERPROFILE%\.codex`。
- 首次启动按提示设置默认沙箱（需要一次管理员授权）；非管理员沙箱隔离更弱，不要用管理员身份运行 Codex。
- WSL 里项目放在 Linux 文件系统（`~/code`），不要放在 `/mnt/c`；必须是 WSL2。
- 用 `.gitattributes` 固定换行符，用 PowerShell 7 + Windows Terminal 避免中文乱码。

下一步：[斜杠命令](/codex/slash-commands)
