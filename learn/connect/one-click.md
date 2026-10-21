---
title: 一键安装 Claude Code 和 Codex
description: 不用懂命令行：Windows 下载一个安装程序双击运行，Mac 粘贴一行命令，自动装好 Claude Code、Codex 并配置 Claude 桌面 App 和 Codex App。全程国内镜像，兼容已经配过 DeepSeek 或登录过官方账号的电脑，随时一键恢复。
---

# 一键安装 Claude Code 和 Codex

在 HiveGPT 网站上点一下，就能把 **Claude Code**、**Codex 命令行**装好，并把 **Claude 桌面 App**、**Codex App** 配成走 HiveGPT。整个过程大约 3 分钟：

- **不用懂命令行**：Windows 是下载一个安装程序双击运行；Mac 只需要打开「终端」粘贴一行。
- **国内网络直接装**：Node.js、Claude Code、Codex 都从国内镜像（npmmirror，失败自动换腾讯云、华为云）下载，不访问 GitHub、claude.ai 或 openai.com。
- **不用管理员权限**：需要的 Node.js 装在你自己的用户文件夹（`~/.hivegpt`）里，不改系统。
- **不怕弄乱原来的配置**：以前配过 DeepSeek 等其它服务、或登录过 Claude / ChatGPT 官方账号都没关系。旧配置会先备份，账号登录不会被删除，随时可以一键恢复。

## 第一步：准备一个 Key

1. 登录 [HiveGPT](https://hivegpt.cn/keys?action=create)，在「API 密钥」页点「创建密钥」。
2. **分组决定配哪些工具**：

| Key 的分组 | 会配置 |
|---|---|
| Claude 分组（如「Claude-按量」、Claude 订阅） | Claude Code 命令行 + Claude 桌面 App |
| GPT 分组（如「GPT-按量」、Codex 订阅） | Codex 命令行 + Codex App |
| 同时有 Claude 和 GPT 模型的分组，或 DeepSeek 等其它分组 | 两套都配 |

Claude 分组一般只允许 Claude Code 和 Claude 桌面 App 使用。想 Claude 和 Codex 都用，就建两个 Key，分别运行一次。

## 第二步：拿到安装程序或命令

在 Key 列表里找到这个 Key，点「使用密钥」，最上面就是 **「⚡ 一键安装（推荐）」**，它会自动选好你电脑的系统。刚注册的新用户在「仪表盘」的「三步开始使用」里也能看到同样的内容。

::: warning 安装程序和命令里都带着你的 Key
不要发给别人，也不要贴到群里或截图分享。Windows 的安装程序运行完会自动删除自己。
:::

## Windows：双击运行

1. 点 **「下载一键安装程序」**，得到 `HiveGPT一键安装.cmd`。
2. 浏览器提示「此类文件可能有害」时，选 **「保留」**（Edge 在下载列表里点「…」→「保留」）。
3. 打开「下载」文件夹，**双击** `HiveGPT一键安装.cmd`。
4. 如果弹出「打开文件 - 安全警告」，点 **「运行」**；如果是蓝色的「Windows 已保护你的电脑」，点 **「更多信息」**，再点 **「仍要运行」**。
5. 会打开一个黑色窗口，自动下载和配置。第一次运行要下载 Node.js 和 Claude Code / Codex，大约 1～3 分钟。看到 **「全部完成！」** 后按任意键关闭窗口。

会用 PowerShell 的话，也可以展开「会用 PowerShell？也可以粘贴这一行」，把那行命令粘贴到 PowerShell 里回车，效果一样。

## Mac：粘贴一行命令

1. 按键盘上的 **⌘ + 空格**，输入「终端」，按回车，会打开一个窗口。
2. 回到网页点 **「复制」**，再回到终端窗口按 **⌘ + V** 粘贴，按回车。
3. 等它显示 **「全部完成！」**。

命令长这样（你复制到的那行已经填好了自己的 Key）：

```bash
curl -fsSL https://hivegpt.cn/install.sh | HIVEGPT_KEY='你的 Key' bash
```

Linux 也是这行命令：打开终端（多数系统可以按 `Ctrl + Alt + T`）粘贴运行即可。Alpine 等 musl 系统需要先自己装好 Node.js 22 以上版本。

## 装好之后

- **用桌面 App**：完全退出 Claude 桌面 App 或 Codex App（Mac 按 ⌘Q，Windows 在右下角托盘图标上右键退出），再重新打开。Codex App 要新建一个任务，配置才会生效。
- **用命令行**：**新开一个终端窗口**，进入项目文件夹，运行 `claude` 或 `codex`。
- 到 HiveGPT 的「使用记录」里，能看到刚才的调用。

还没装桌面 App 的话，脚本会先把配置写好，装好 App 后打开就能用。Claude 桌面 App 从 [claude.ai/download](https://claude.ai/download) 下载（需要能访问 claude.ai），Codex App 见 [桌面应用与界面](/codex/app)。

### Claude 桌面 App 之前登录过 claude.ai 账号

脚本把 HiveGPT 写成桌面 App 的「第三方网关」配置，和 claude.ai 登录分开存放，原来的登录不会被删除。重新打开 App 后：

- 出现登录页时，选择用组织 / 第三方配置进入，不要选「用 Claude.ai 登录」；
- 已经进到 claude.ai 账号里的话，在左下角账户菜单里切换到名为 **HiveGPT** 的配置。

以后想切回 claude.ai 账号，在同一个地方切回去即可。

## 已经配过 DeepSeek、CC Switch 或官方账号？

脚本是**合并**原有配置，不是整个覆盖：

| 你原来的情况 | 脚本怎么处理 |
|---|---|
| Claude Code 配过 DeepSeek 等其它服务（`~/.claude/settings.json` 或 `.zshrc` 里的 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_MODEL`） | 换成 HiveGPT 的地址和 Key；HiveGPT 上没有的模型名（比如 `deepseek-chat`）会去掉；终端配置文件里的旧 `export` 行会被注释掉（Windows 是删掉对应的用户环境变量）。权限、hooks、MCP 等其它设置保持不变 |
| 登录过 claude.ai 账号 | 登录保留。Claude Code 会优先用 HiveGPT 的 Key，启动时如果提示 Auth conflict 可以忽略；想彻底只用 HiveGPT，在 Claude Code 里输入 `/logout` |
| Codex 用 ChatGPT 账号登录，或配过 DeepSeek 等其它提供方 | `auth.json` 里的登录不动；`config.toml` 开头的模型和提供方改成 HiveGPT，原来那段提供方配置、MCP、项目信任等都保留 |
| Claude 桌面 App 配过其它网关 | 新增一个「HiveGPT」配置并设为当前，原来的配置还在，可以在 App 的 Developer → Configure Third-Party Inference 里切回 |
| 用 CC Switch 管理 Claude Code | 它切换供应商时会改写 `settings.json`。建议在「使用密钥」弹窗里一键导入 HiveGPT 到 CC Switch，并在 CC Switch 里选中它 |

改动前的文件都备份在 `~/.hivegpt/backups/<时间>/`。

## 换回原来的配置

「使用密钥」弹窗的一键安装下方有「想换回原来的配置」这一行，复制运行即可：

```bash
# Mac / Linux
curl -fsSL https://hivegpt.cn/install.sh | bash -s -- --restore
```

```powershell
# Windows（PowerShell）
$env:HIVEGPT_RESTORE='1'; irm https://hivegpt.cn/install.ps1 | iex
```

它会把最近一次安装改过的文件全部还原。安装过多次的话，每运行一次往回退一步。脚本装的 Node.js 和命令行工具在 `~/.hivegpt` 文件夹里，不需要的话可以直接删掉。

## 常见问题

| 现象 | 处理 |
|---|---|
| 浏览器不让下载 `.cmd` 文件 | 在下载列表里选「保留」；还是不行，就改用 PowerShell 那一行命令 |
| 杀毒软件拦截了安装程序 | 安装程序只是下载并运行 `https://hivegpt.cn/install.ps1`，可以选择允许；不放心就用 PowerShell 那一行，效果一样 |
| 提示 Key 无效（401） | Key 复制不完整或已删除，回「API 密钥」页重新下载 / 复制 |
| 提示 Key 不能用（403） | Key 被停用、设置了 IP 白名单，或没有分配分组，到「API 密钥」页检查 |
| 下载 Node.js 或 Claude Code 失败 | 多是网络不稳，重新运行一次即可；公司网络拦截的话，换手机热点试试 |
| 装完提示找不到 `claude` / `codex` 命令 | 关掉当前终端，**新开一个**再试；新的 PATH 只对新窗口生效 |
| Claude Code 报「模型不存在」 | 当前终端里还留着旧的 `ANTHROPIC_MODEL`，新开一个终端再运行 |
| 桌面 App 里没看到 HiveGPT 的模型 | 要**完全退出**再打开，只关窗口不算；Claude 桌面 App 还要切到「HiveGPT」配置（见上文） |
| Codex 还在要求登录 ChatGPT | 完全退出 Codex App 或 IDE 扩展后重新打开，再新建一个任务 |

## 想自己配置

一键安装只是把下面这些手动步骤自动做了一遍。想逐项了解或手动配置：

- Codex：[接入 HiveGPT](/codex/hivegpt)
- Claude Code、OpenCode 等其它工具：在「使用密钥」弹窗里选对应的客户端，按提示复制配置；其它工具见 [在常用工具里配置](/connect/tools)

高级用法：命令后面可以加参数，比如只配 Codex：`curl -fsSL https://hivegpt.cn/install.sh | HIVEGPT_KEY='你的 Key' bash -s -- --tools=codex`。`--tools=claude` 只配 Claude，`--no-install` 只写配置、不安装命令行工具。
