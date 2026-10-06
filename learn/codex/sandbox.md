---
title: 沙箱与审批
description: Codex 如何限制模型执行的命令：三种沙箱模式、审批策略、各平台的沙箱实现、网络与可写目录设置，以及推荐的权限组合。
---

# 沙箱与审批

Codex 会在你的电脑上真实地运行命令：装依赖、跑测试、改文件、调 git。模型偶尔会误判，比如删错目录、改到项目外的配置、把数据发到不该去的地方。所以 Codex 用两道闸门管住它：

- **沙箱（sandbox）**：从操作系统层面限制命令**能**做什么，比如只能写当前项目、不能联网。命令越界会直接失败，不靠模型自觉。
- **审批（approval）**：决定什么时候**停下来问你**，比如命令需要越出沙箱时，由你点同意或拒绝。

两者配合，才是你在 Codex 里看到的「权限」。这一页讲清楚每个选项的含义、底层怎么实现、怎么组合，以及怎么排查「为什么这个命令被拦了」。

## 沙箱模式：能做什么

`sandbox_mode` 有三个值：

| 模式 | 读文件 | 写文件 | 网络 | 适合 |
|---|---|---|---|---|
| `read-only` | 可以 | 不可以 | 不可以 | 阅读陌生代码、只做分析和问答 |
| `workspace-write` | 可以 | 仅工作区和临时目录 | 默认不可以 | 日常开发，绝大多数情况 |
| `danger-full-access` | 可以 | 任何地方 | 可以 | 已经在容器或虚拟机里隔离的环境 |

`workspace-write` 下「工作区」指启动 Codex 的目录（或 `-C` 指定的目录），外加：

- 系统临时目录 `/tmp` 和 `$TMPDIR`（可用 `exclude_slash_tmp`、`exclude_tmpdir_env_var` 排除）；
- `--add-dir` 或 `writable_roots` 额外加入的目录。

工作区里的 `.git`、`.codex` 等元数据目录会被保护为只读，防止命令篡改 Git 历史或改写 Codex 自己的项目配置。需要提交代码时，Codex 会请求越出沙箱，由你批准。

::: tip 默认是哪种
没有写 `sandbox_mode` 时，默认是 `read-only`。但第一次进入项目时 Codex 会问你是否信任它，只要做过信任决定，默认就变成 `workspace-write`。所以大多数人实际用到的默认值是 `workspace-write`。
:::

## 审批策略：什么时候问你

`approval_policy`（命令行 `-a`）决定 Codex 遇到需要额外权限的操作时怎么办：

| 值 | 行为 | 适合 |
|---|---|---|
| `on-request` | 默认值。沙箱内的操作直接做；需要越出沙箱（写工作区外、联网、执行受限命令）时，由模型发起请求，你来批准 | 交互式使用 |
| `never` | 从不询问。越界操作直接失败，失败信息交还给模型自己想办法 | `codex exec`、CI、无人值守 |
| `granular` | 细粒度：按类别决定哪些请求可以弹给你，其余自动拒绝 | 需要精确控制的团队 |

还有两个历史值需要知道：

- `untrusted`：只有少数公认安全的只读命令（`ls`、`cat` 等）免审批，其他一律先问。现在主要作为**未信任项目**的内部默认值；0.142 的 `-a` 还接受它，最新源码已不允许在配置里手写。
- `on-failure`：先在沙箱里跑，失败了再问是否不带沙箱重试。已废弃，最新源码里它被当成 `on-request` 处理。

细粒度写法示例：

```toml
[approval_policy.granular]
sandbox_approval = true      # 命令越界请求，弹给我
rules = true                 # execpolicy 规则要求确认的命令，弹给我
mcp_elicitations = true      # MCP 服务的确认请求，弹给我
request_permissions = false  # 模型主动申请扩大权限，直接拒绝
skill_approval = false       # Skill 脚本的执行确认，直接拒绝
```

### 审批弹窗怎么选

Codex 请求执行命令时，会显示完整的命令和理由，通常有这几种选择：同意这一次；本次会话内对同类命令都同意；拒绝并告诉 Codex 换个做法。

原则很简单：**看懂了再同意**。尤其留意 `rm -rf`、`git push --force`、`curl ... | sh`、修改 shell 配置文件、往项目外写文件这类操作。不确定就拒绝，并在输入框里说明你希望它怎么做。

### 让自动审查替你把关

每个请求都要人看，长任务时会很打断。`approvals_reviewer` 可以把审批请求先交给一个专门的审查代理：它会收集上下文、按风险判断批准或拒绝，高风险的才拦下来。

```toml
approvals_reviewer = "auto_review"   # 默认是 "user"，即由你审批
```

自动审查拒绝了你其实想执行的操作时，在会话里输入 `/approve`，可以批准它重试一次。最新源码还提供 `--approve-for-me` 启动参数，一次性打开「工作区可写 + 按需审批 + 自动审查」，以你本机 `codex --help` 为准。

## 推荐组合

| 场景 | sandbox_mode | approval_policy | 命令行写法 |
|---|---|---|---|
| 分析陌生仓库、只问不改 | `read-only` | `on-request` | `codex -s read-only` |
| 日常开发（推荐默认） | `workspace-write` | `on-request` | `codex -s workspace-write -a on-request` |
| 日常开发，少被打断 | `workspace-write` | `on-request` + 自动审查 | `codex -c approvals_reviewer='"auto_review"'` |
| CI / 脚本，只改仓库 | `workspace-write` | `never` | `codex exec -s workspace-write "…"` |
| CI，只读出报告 | `read-only` | `never` | `codex exec -s read-only "…"` |
| 一次性容器、虚拟机 | `danger-full-access` | `never` | `codex --yolo` |

在会话里用 `/permissions` 可以随时切换，三个内置预设对应上表：**Read Only**（只读 + 按需审批）、**Default**（工作区可写 + 按需审批）、**Full Access**（不受限 + 从不询问）。

::: warning 关于 --full-auto 和 --yolo
- 老教程常见的 `--full-auto` 已从 Codex 中移除，用 `-s workspace-write -a on-request` 代替。
- `--yolo` 是 `--dangerously-bypass-approvals-and-sandbox` 的别名：**没有沙箱，也不审批**，模型生成的任何命令都会以你的用户身份直接执行。它能读你的 SSH 私钥、浏览器数据，能删除家目录，能把文件上传到任意地址。只在用完即弃的容器、CI 虚拟机这类本身已隔离的环境里使用，不要在日常电脑上用。
:::

## 网络访问

`workspace-write` 默认禁止沙箱内命令联网。安装依赖、跑需要访问外部服务的测试时，有两种做法：

**按需批准**：保持默认，命令因为网络失败时 Codex 会请求越出沙箱重试，你批准即可。安全，但交互多。

**直接放开**：

```toml
[sandbox_workspace_write]
network_access = true
```

或者只对这一次：

```bash
codex -c sandbox_workspace_write.network_access=true
```

::: details 更精细的网络控制
较新的版本支持「权限配置」（`[permissions.名称]`，配合 `default_permissions` 使用），可以按域名放行网络、对文件路径设置读写或禁止读取，并内置了 `:read-only`、`:workspace`、`:danger-full-access` 三个基础配置供 `extends` 继承。按域名过滤依赖实验性的网络代理功能（`network_proxy`）。这套语法还在演进，普通用户用 `sandbox_mode` + `network_access` 就够了，需要时以官方文档为准。
:::

注意 `network_access` 只管沙箱里执行的命令。模型调用的内置网页搜索由 `web_search` 配置控制，和它无关。

## 额外的可写目录

工作区之外还需要写的地方，比如 monorepo 里相邻的包、构建缓存：

```bash
# 只对这一次
codex --add-dir ../shared-lib --add-dir ~/.cache/my-tool
```

```toml
# 长期生效，必须是绝对路径
[sandbox_workspace_write]
writable_roots = ["/Users/me/work/shared-lib", "/Users/me/.cache/my-tool"]
```

宁可多加几个具体目录，也不要为了省事切到 `danger-full-access`。

## 各平台的沙箱实现

沙箱不是 Codex 自己模拟的，而是调用操作系统的隔离机制，所以不同系统行为略有差异。

| 平台 | 实现 | 说明 |
|---|---|---|
| macOS | Seatbelt（系统自带的 `sandbox-exec`） | 开箱即用，无需安装 |
| Linux | bubblewrap（`bwrap`）+ seccomp | 优先用 `PATH` 里的系统 `bwrap`，没有则用 Codex 自带的；旧的 Landlock 方案已弃用 |
| WSL2 | 同 Linux | 正常可用 |
| WSL1 | 不支持 | 无法创建用户命名空间，需要沙箱的命令会被拒绝 |
| Windows 原生 | Windows 沙箱（`elevated` / `unelevated` 等级别） | 首次使用需要设置，见下文 |

**Linux 常见问题**：Docker 容器或部分发行版禁止非特权用户创建用户命名空间，bubblewrap 会启动失败，Codex 启动时会给出警告。可以在宿主机开启用户命名空间，或者既然已经在容器里，就考虑把容器本身当沙箱、在容器内使用 `danger-full-access`。

**Windows**：原生 Windows 上需要先配置沙箱。会话里如果出现 `/setup-default-sandbox` 命令，运行它完成提权版沙箱的设置；也可以在配置里指定：

```toml
[windows]
sandbox = "elevated"   # 或 "unelevated"
```

源码里有一条值得注意的规则：Windows 沙箱未启用时，`workspace-write` 会被自动降级为 `read-only`，表现为「Codex 什么都写不了」。遇到这种情况先检查沙箱设置。更多 Windows 细节见 [Windows 上使用](/codex/windows)。

## 用 codex sandbox 测试

想知道某个命令在 Codex 的沙箱里会怎样，不用开会话，直接测：

```bash
# 在当前配置的沙箱里运行命令
codex sandbox -- touch ./ok.txt          # 工作区内，应成功
codex sandbox -- touch ~/outside.txt     # 工作区外，应被拒绝
codex sandbox -- curl -I https://example.com   # 默认应因网络受限失败

# 临时放开网络再试
codex sandbox -c sandbox_workspace_write.network_access=true -- curl -I https://example.com

# macOS：把被 Seatbelt 拦截的操作打印出来，定位到底碰了哪个路径
codex sandbox --log-denials -- npm install
```

一个命令在 Codex 里莫名失败，用这种方式单独复现，比让模型反复重试高效得多。

## 用规则管住特定命令

沙箱管的是「能碰哪些文件和网络」，管不了「这条命令本身危不危险」。比如工作区内的 `git push --force` 完全在沙箱允许范围内。这类需求用 **execpolicy 规则**：在 `~/.codex/rules/` 或项目的 `.codex/rules/` 下写 `.rules` 文件，按命令前缀决定允许、需要确认还是禁止：

```python
# ~/.codex/rules/default.rules
prefix_rule(
    pattern = ["git", "push", ["--force", "-f"]],
    decision = "forbidden",
    justification = "禁止强推，请使用 --force-with-lease 并先和我确认",
)

prefix_rule(
    pattern = ["npm", "publish"],
    decision = "prompt",
)
```

规则语法、钩子（在工具调用前后运行你的脚本）等内容见 [规则与钩子](/codex/hooks)。

## 排查清单

| 现象 | 可能原因 | 处理 |
|---|---|---|
| 什么文件都写不了 | 当前是 `read-only`；或 Windows 沙箱未启用被降级 | `/status` 看权限，`/permissions` 切到 Default |
| `npm install` / `pip install` 失败 | 沙箱内禁止联网 | 批准越界重试，或打开 `network_access` |
| 写缓存目录失败 | 目录不在工作区内 | `--add-dir` 或 `writable_roots` |
| `git commit` 总要审批 | `.git` 在沙箱内是只读的 | 正常设计；批准即可，或用自动审查减少打扰 |
| Linux 上每条命令都报沙箱错误 | 系统禁止用户命名空间 | 开启用户命名空间，或在隔离容器里改用无沙箱 |
| 配置了却没效果 | 被命令行或项目配置覆盖，或被管理员约束 | `/debug-config` 查来源 |

## 小结

- 沙箱决定命令**能**做什么，审批决定什么时候**问你**；日常用 `workspace-write` + `on-request`。
- `workspace-write` 默认不联网，`.git`、`.codex` 只读；需要时用 `network_access` 和 `writable_roots` 精确放开。
- macOS 用 Seatbelt，Linux 用 bubblewrap，Windows 需要先设置沙箱，否则可写模式会降级为只读。
- `--full-auto` 已移除；`--yolo` 不设任何限制，只在隔离环境用。
- 命令级的禁止和确认交给 execpolicy 规则；排查问题先用 `codex sandbox` 复现。

下一步：[模型与推理强度](/codex/models)
