---
title: 接入 HiveGPT
description: 在 HiveGPT 创建 OpenAI 类分组的 Key，用「使用密钥」弹窗生成 config.toml、auth.json 和模型目录，让 Codex CLI、IDE 扩展和桌面端都走 HiveGPT 调用 gpt-5.5。
---

# 接入 HiveGPT

这是本站推荐的接入方式：Codex 本身照常安装，模型调用改走 [HiveGPT](https://hivegpt.cn) 的接口。配好之后，CLI、IDE 扩展、桌面端用的是同一份配置，默认模型是 `gpt-5.5`。

整个过程大约五分钟，核心是让 HiveGPT 的「使用密钥」弹窗替你生成配置文件，你只负责把内容放到正确的位置。还没装 Codex 的话，先看 [安装与登录](/codex/install)。

## 为什么用 HiveGPT

- **国内网络可以直接访问**，不需要额外的网络工具。
- **不需要海外账号和海外支付方式**，在 HiveGPT 充值后按实际用量从余额扣费，用量明细在「使用记录」里可查。
- **接口兼容 Codex 使用的 Responses API**，Codex 的读写文件、运行命令、审查代码、压缩上下文等功能都能正常使用。

需要说明的是：通过 HiveGPT 调用和用 ChatGPT 账号登录是两种计费方式，ChatGPT 套餐里的 Codex 额度不会用在这里。具体价格以 HiveGPT 站内显示为准。

## 第一步：创建 Key

1. 登录 [HiveGPT](https://hivegpt.cn)，打开左侧「API 密钥」页，点「创建密钥」。
2. **分组**必须选 OpenAI 类的分组，例如「GPT-按量」。Codex 只能调用这类分组下的模型，选错分组是「模型不可用」最常见的原因。
3. 名称随意，建议写清用途，比如「codex-笔记本」。以后哪台设备丢了，可以单独删除对应的 Key。

## 第二步：打开「使用密钥」弹窗

在 Key 列表里找到刚建的 Key，点「使用密钥」。弹窗里依次选择：

1. **客户端**：选「Codex CLI」。IDE 扩展和桌面端也用这一份配置，不用另外找选项。另一个「Codex CLI (WebSocket)」会额外开启 WebSocket 传输，一般用户选普通的「Codex CLI」即可。
2. **认证方式**：两个选项，区别见下表。
3. **系统**：「macOS / Linux」或「Windows」，决定生成的路径写法。

| 认证方式 | 生成的文件 | Key 存在哪 | 适合 |
|---|---|---|---|
| 兼容模式（默认） | `config.toml` + `auth.json` | `auth.json` 的 `OPENAI_API_KEY` 字段 | 大多数人，和旧版 Codex 配置兼容 |
| API Key 模式 | 只有 `config.toml` | `config.toml` 里的 `experimental_bearer_token` | 需要在 Codex 里使用图片生成等依赖客户端图片执行器的功能；想和 ChatGPT 登录共存 |

::: warning API Key 模式需要彻底重启
选 API Key 模式并保存配置后，要**完全退出** Codex 桌面端或 CLI 再重新打开，然后新建一个任务（会话），客户端才会重新注册工具。只是开个新会话不够。
:::

## 第三步：获取模型目录

弹窗里有一块「Codex 模型目录」。点「获取目录」，它会用当前 Key 拉取你这个分组下可用的模型列表，显示「已获取 N 个模型」后点「下载目录」，得到一个 `codex-models.json`。

把它保存到：

- macOS / Linux：`~/.codex/codex-models.json`
- Windows：`%USERPROFILE%\.codex\codex-models.json`

这个文件告诉 Codex 每个模型的名字、上下文窗口、支持哪些推理强度等信息，`/model` 里的列表就来自它。文件里只有模型元数据，没有 Key。

::: tip 先获取目录，再复制配置
获取目录后，弹窗生成的 `config.toml` 会根据目录里 `gpt-5.5` 的默认档位自动补上一行 `model_reasoning_effort`。所以顺序是：先获取目录，再复制下一步的配置。
:::

## 第四步：写入 config.toml 和 auth.json

先确保配置目录存在：

```bash
mkdir -p ~/.codex
```

Windows 按 `Win + R`，输入 `%USERPROFILE%\.codex` 回车；提示找不到就先手动建这个文件夹。

然后按弹窗内容创建或修改文件。兼容模式下生成的 `config.toml` 大致如下（**仅作示意，以弹窗里的内容为准**，地址和模型都由弹窗填好）：

```toml
model_provider = "OpenAI"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_reasoning_effort = "medium"   # 获取目录后才会出现，档位以弹窗为准
disable_response_storage = true
model_catalog_json = "~/.codex/codex-models.json"
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://hivegpt.cn/v1"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true
```

`auth.json` 只有一项：

```json
{
  "OPENAI_API_KEY": "你的 HiveGPT Key"
}
```

几个关键字段的含义：

| 字段 | 作用 |
|---|---|
| `model_provider = "OpenAI"` | 使用下面 `[model_providers.OpenAI]` 这一段定义的提供方。注意大写的 `OpenAI` 是自定义名字，和 Codex 内置的小写 `openai` 不是一回事 |
| `base_url` | HiveGPT 的接口地址 |
| `wire_api = "responses"` | 使用 Responses API。新版 Codex 只支持这一种 |
| `requires_openai_auth = true` | 从 `auth.json` 读取 Key（兼容模式） |
| `model_catalog_json` | 指向上一步下载的模型目录 |
| `review_model` | `/review` 代码审查使用的模型 |

::: warning 已有 config.toml 时怎么合并
弹窗提示「请确保以下内容位于 config.toml 文件的开头部分」。原因是 TOML 里，写在某个 `[表名]` 之后的键都属于那张表，把 `model = ...` 这类顶层键贴到文件末尾，会被当成上一张表的字段而失效。正确做法：

1. 先备份：`cp ~/.codex/config.toml ~/.codex/config.toml.bak`
2. 把弹窗里 `[model_providers.OpenAI]` 之前的顶层键放到文件最开头，删掉原文件里同名的旧键（比如旧的 `model`、`model_provider`）。
3. `[model_providers.OpenAI]` 和 `[features]` 两段放在后面；原文件已有 `[features]` 的话，把 `goals = true` 合并进去，不要出现两个同名表。
4. 原有的 `[mcp_servers.*]`、`[projects.*]` 等配置保持不动。
:::

弹窗生成的 `disable_response_storage`、`network_access`、`windows_wsl_setup_acknowledged` 是为兼容不同版本保留的字段，新版 Codex 会忽略不认识的键，不影响使用。只有在你主动用 `--strict-config` 启动时，才会因为这些键报错。

## 第五步：启动验证

进入任意一个项目文件夹（或新建一个空文件夹）启动：

```bash
mkdir -p ~/codex-hello && cd ~/codex-hello
codex
```

第一次在这个目录启动会问是否信任该文件夹，选信任并继续。然后先问一个不改文件的问题：

```text
用两句话介绍你能在这个文件夹里帮我做什么。
```

能正常回复，就说明 Codex 已经通过 HiveGPT 调用到模型。再做两个检查：

- 输入 `/status`，确认模型是 `gpt-5.5`、提供方是 OpenAI（即 HiveGPT 地址）。
- 输入 `/model`，能看到模型目录里的模型和推理强度选项。

到 HiveGPT 的「使用记录」里，应该能看到刚才这次调用。

## 在 IDE 扩展和桌面端使用

不需要额外配置。IDE 扩展和 ChatGPT 桌面端里的 Codex 都读取同一个 `~/.codex/config.toml`，打开后就会使用 HiveGPT。如果它们之前已经打开着，改完配置后完全退出再打开一次；用 API Key 模式的话，记得按前面的提示彻底重启并新建任务。

## 和 ChatGPT 登录并存

有的人既有 ChatGPT 套餐，又想用 HiveGPT 按量补充。兼容模式下 HiveGPT 的 Key 和 ChatGPT 登录凭据都写在 `auth.json` 里，互相覆盖，频繁切换不方便。推荐用**配置档（profile）**把两套设置分开：主配置保持 ChatGPT 登录，HiveGPT 单独放一个文件。

新版 Codex 的配置档是一个独立文件 `~/.codex/名字.config.toml`，用 `-p 名字` 启动时叠加在主配置之上。创建 `~/.codex/hivegpt.config.toml`：

```toml
model_provider = "hivegpt"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_catalog_json = "~/.codex/codex-models.json"

[model_providers.hivegpt]
name = "HiveGPT"
base_url = "https://hivegpt.cn/v1"
wire_api = "responses"
env_key = "HIVEGPT_API_KEY"   # 从环境变量读取 Key，不写进文件
```

使用时：

```bash
export HIVEGPT_API_KEY="你的 HiveGPT Key"   # 可以写进 ~/.zshrc
codex -p hivegpt                             # 走 HiveGPT
codex                                        # 走 ChatGPT 登录
```

::: warning 旧写法已经不支持
早期教程里在 config.toml 中写 `[profiles.xxx]` 表和顶层 `profile = "xxx"` 的做法，新版本不再支持：顶层 `profile = ...` 会直接报错，要改成上面这种独立文件。更多用法见 [高级配置](/codex/config-advanced)。
:::

另一种更彻底的隔离办法是给 HiveGPT 单独一个配置目录：`CODEX_HOME=~/.codex-hivegpt codex`，两套配置、登录、会话完全分开。缺点是 IDE 扩展和桌面端默认只读 `~/.codex`。

## 常见报错

| 现象 | 常见原因 | 处理 |
|---|---|---|
| 401 Unauthorized | Key 复制不完整、多了空格，或 Key 已被删除 | 重新从弹窗复制；确认 `auth.json` 是合法 JSON |
| 403 Forbidden | Key 被禁用、设置了 IP 白名单而当前 IP 不在其中 | 到「API 密钥」页检查 Key 状态和限制 |
| 模型不可用 / model not found | Key 的分组不是 OpenAI 类；`model` 写了分组里没有的模型 | 换 OpenAI 类分组的 Key；用 `/model` 从目录里选 |
| 提示余额或额度不足 | 账户余额用完 | 在 HiveGPT 充值后重试 |
| 仍然弹出 ChatGPT 登录界面 | 顶层 `model_provider` 没生效，通常是被放到了某个表后面 | 把顶层键挪到文件开头 |
| `wire_api = "chat"` 相关报错 | 用了旧教程的配置 | 改成 `wire_api = "responses"` |

更多问题见 [常见问题排查](/codex/troubleshooting)。也可以运行 `codex doctor` 让它检查配置和连接。

## Key 安全

- `auth.json` 和 API Key 模式下的 `config.toml` 都含有明文 Key，只放在用户目录下，**不要复制进项目文件夹，不要提交到 Git**。
- 不要把 Key 贴进给 Codex 的消息、截图或公开的聊天记录里。
- 每台设备、每个用途单独建 Key；怀疑泄露就到「API 密钥」页删除，新建一个并重新生成配置。
- 团队共用机器时，考虑用 `env_key` 从环境变量读取，或者用各自的 `CODEX_HOME`。

## 小结

- 在 HiveGPT 用 OpenAI 类分组（如「GPT-按量」）创建 Key，点「使用密钥」选「Codex CLI」。
- 先获取并下载 `codex-models.json`，再复制 `config.toml`（和兼容模式下的 `auth.json`）到 `~/.codex`。
- 顶层配置放在 config.toml 开头；`wire_api` 只能是 `responses`。
- 同一份配置在 CLI、IDE 扩展、桌面端通用；要和 ChatGPT 登录并存，用 `hivegpt.config.toml` 配置档加 `codex -p hivegpt`。

下一步：[第一个任务](/codex/first-task)，或者跟着 [C 路线](/c/) 用 Codex 做一个完整项目。
