---
title: Worktree 与并行任务
description: 用 git worktree 给每个 Codex 任务一份独立的工作目录，同时跑多个任务而互不干扰，并安全地合并和清理。
---

# Worktree 与并行任务

Codex 干一个任务常常要几分钟到几十分钟。等待的时候，你很自然会想：能不能再开一个 Codex 去做另一件事？可以，但如果两个会话在**同一个目录**里工作，就会出问题：A 在改 `api.ts`，B 也在改 `api.ts`；A 跑测试时，B 改了一半的代码让测试挂掉；最后 `git diff` 里两件事的改动混在一起，分不开。

解决办法是 Git 自带的 worktree：同一个仓库，检出多份独立的工作目录，每个 Codex 会话占一份。这一页讲 worktree 的基本操作、怎样配合多个 Codex 会话、桌面应用里的 Worktree 模式，以及合并、清理、端口和依赖冲突这些实际会遇到的问题。

## git worktree 是什么

一个普通的 Git 仓库只有一个工作目录。`git worktree add` 可以再检出一个目录，它和原目录**共享同一个 `.git` 对象库**（提交、分支、远程都是同一套），但文件、暂存区、当前分支各自独立。

| 做法 | 磁盘占用 | 分支共享 | 切换成本 | 适合 |
|---|---|---|---|---|
| `git stash` + 切分支 | 无 | 是 | 每次切换都要收拾现场 | 偶尔临时切一下 |
| 再 `git clone` 一份 | 完整一份历史 | 否，要 push/pull 同步 | 低 | 完全独立的实验 |
| `git worktree add` | 只多一份工作文件 | 是，提交立刻互相可见 | 低 | 并行任务 |

在 worktree A 里提交的东西，在主目录里马上能 `git log` 看到、能 `git merge`，不需要推到远程再拉回来。这正是并行开发需要的。

### 基本命令

```bash
# 在仓库根目录执行。新建分支 fix-login，并检出到旁边的目录
git worktree add ../shop-fix-login -b fix-login

# 基于指定起点（比如远程 main）新建
git worktree add ../shop-feat-export -b feat-export origin/main

# 检出已有分支
git worktree add ../shop-hotfix hotfix/1.4.2

# 查看所有 worktree
git worktree list

# 删除 worktree（目录里有未提交改动时会拒绝，加 --force 强删）
git worktree remove ../shop-fix-login

# 手动删了目录之后，清理 Git 里残留的记录
git worktree prune
```

::: warning 同一个分支不能同时检出两次
`git worktree add ../x main` 在主目录已经检出 main 时会报错：Git 不允许同一个分支同时出现在两个 worktree 里，否则两边提交会互相覆盖。每个 worktree 用自己的分支，或者用 `--detach` 检出一个不在任何分支上的提交。
:::

### 新 worktree 里没有什么

worktree 只包含**被 Git 跟踪的文件**。下面这些东西不会自动出现：

- `node_modules`、`.venv`、`target/` 这类依赖和构建产物；
- `.env`、`.env.local` 等被 `.gitignore` 忽略的本地配置；
- 你在主目录里**还没提交**的改动。

所以新建 worktree 之后，通常要先做一次初始化，后面「依赖与环境」一节有现成脚本。

## 手动配合多个 Codex 会话

最通用的做法，不依赖任何特定版本的功能：

```bash
cd ~/work/shop

# 两个任务，两个 worktree，各自一个分支
git worktree add ../shop-fix-login  -b fix-login
git worktree add ../shop-feat-export -b feat-export

# 终端 1
cd ../shop-fix-login && codex "登录时偶发 500，日志见 logs/err.txt，找到原因并修复，补回归测试"

# 终端 2
cd ../shop-feat-export && codex "在订单列表页加「导出 CSV」按钮，字段同列表列，后端接口放在 /api/orders/export"
```

也可以不 `cd`，用 `-C` 指定目录：

```bash
codex -C ../shop-feat-export "……"
```

无人值守的任务可以直接用 `codex exec` 放到后台：

```bash
codex exec -C ../shop-fix-deps -s workspace-write -o ../fix-deps.md \
  "把 lodash 从 4.17.15 升级到最新 4.x，修复因此产生的类型错误，跑通测试" &
```

每个 Codex 会话只看到、只修改自己目录里的文件。它们共享的只有 Git 对象库，所以可以互相看到对方的**提交**，但不会碰到对方**正在改的文件**。

::: tip 用 tmux 管理多个会话
任务多了，开一堆终端窗口不好找。用 tmux 每个任务一个窗口，按名字切换：

```bash
tmux new-session -d -s codex -n login  -c ../shop-fix-login  'codex'
tmux new-window      -t codex -n export -c ../shop-feat-export 'codex'
tmux attach -t codex
```
:::

### 在 worktree 里提交：注意沙箱

在 `workspace-write` 沙箱下，Codex 不能写 `.git` 目录（worktree 里 `.git` 是一个指向主仓库的文件，它指向的真实目录同样受保护）。所以让 Codex 执行 `git commit` 时，交互模式会弹出审批请求，你同意后它才能提交。习惯上有两种做法：

- 你自己提交：Codex 改完后，你在这个 worktree 里 `git add -p` 看一遍再提交；
- 让 Codex 提交：遇到审批时同意，或在 `/permissions` 里临时放宽权限。

沙箱的规则详见 [沙箱与审批](/codex/sandbox)。

## 桌面应用里的 Worktree 模式

在 Codex 桌面应用里新建任务时，可以选择任务在哪里运行：

| 模式 | 行为 | 适合 |
|---|---|---|
| 本地（Local） | 直接在你的项目目录里改 | 单个任务、你想马上在编辑器里看到改动 |
| Worktree | 应用自动新建一个 worktree，任务在里面运行 | 同时跑多个任务、不想打扰手头的工作 |

根据官方源码，应用托管的 worktree 有这些特点：

- 默认放在 `~/.codex/worktrees/` 下，每个任务一个随机命名的子目录，里面是以仓库名命名的检出目录；
- 检出的是起点提交的**分离 HEAD**（detached HEAD），不占用任何分支，所以不会和你主目录的分支冲突；
- 起点是某个提交，**你本地未提交的改动不会带过去**。想让任务基于你手头的改动，先提交（哪怕是临时提交）再开任务；
- 应用会自动清理旧的托管 worktree，默认保留最近 15 个。存放位置、是否自动清理、保留数量都可以在应用设置里调整（对应 `config.toml` 中 `[desktop]` 表的 `git-worktree-root`、`worktree-auto-cleanup-enabled`、`worktree-keep-count`）。

任务完成后，在应用里查看改动（diff），满意的话可以在 worktree 里建分支、提交、推送并开 PR，或者把改动带回本地目录继续处理。具体按钮的名字和位置随版本变化，以应用界面为准。桌面应用的整体介绍见 [桌面应用与界面](/codex/app)。

::: details 命令行里的托管 worktree（较新版本）
官方仓库的最新源码给 CLI 增加了同样的托管 worktree：`codex --worktree` 在新的托管 worktree 里开始会话，`codex exec --worktree "…"` 用于非交互任务，TUI 里还有 `/worktree` 斜杠命令。它们和桌面应用共用同一个存放位置，但 CLI 创建的不会被自动清理。本站写作时的 codex-cli 0.142.3 还没有这些参数，升级后可用 `codex --help` 确认。
:::

## 依赖与环境：每个 worktree 都要初始化

worktree 刚建出来是「干净」的，直接让 Codex 跑测试多半会失败在「找不到模块」。可以在仓库里放一个初始化脚本，建完 worktree 就执行：

```bash
#!/usr/bin/env bash
# scripts/worktree-init.sh —— 在新 worktree 的根目录运行
set -euo pipefail
main_dir=$(git worktree list --porcelain | awk '/^worktree /{print $2; exit}')

# 1. 复制本地配置（不要提交到 Git 的那些）
for f in .env .env.local; do
  [ -f "$main_dir/$f" ] && cp "$main_dir/$f" .
done

# 2. 安装依赖（pnpm 的全局存储让多个 worktree 共享包，几乎不占额外空间）
[ -f pnpm-lock.yaml ] && pnpm install --frozen-lockfile
[ -f requirements.txt ] && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt

echo "worktree 初始化完成：$(pwd)"
```

然后把它写进 `AGENTS.md`，Codex 在新目录里遇到依赖缺失时就知道该做什么：

```markdown
## 环境
- 新的 worktree 先运行 `bash scripts/worktree-init.sh`，再运行测试。
- 不要修改 .env 文件。
```

安装依赖需要联网，而 workspace-write 沙箱默认不能联网。最简单的是你自己在开任务前跑一次初始化脚本；或者让 Codex 跑时同意它的联网请求。

### 端口和外部资源冲突

两个 worktree 同时启动开发服务器，会抢同一个端口；共用一个数据库，测试数据会互相污染。常见的处理：

| 资源 | 冲突表现 | 处理 |
|---|---|---|
| 开发服务器端口 | `EADDRINUSE: address already in use :::3000` | 每个 worktree 的 `.env.local` 写不同的 `PORT`；或让框架自动换端口（Vite 默认会顺延） |
| Docker Compose | 容器名、网络、数据卷冲突 | 每个 worktree 设不同的 `COMPOSE_PROJECT_NAME`，例如用目录名 |
| 数据库 | 测试互相删数据 | 每个 worktree 用独立库名，如 `shop_test_fix_login`；或用 SQLite / 测试容器 |
| 缓存目录 | 构建产物互相覆盖 | 用相对仓库目录的缓存路径，不要写死绝对路径 |
| 全局安装的工具 | 一个任务升级了全局 CLI，另一个跟着变 | 依赖写进项目（devDependencies / venv），不要让 Codex 全局安装 |

一个按目录名自动分配端口的小技巧：

```bash
# 加到 worktree-init.sh 末尾：用目录名算一个 3100–3999 之间的端口
name=$(basename "$PWD")
port=$(( 3100 + $(printf '%s' "$name" | cksum | cut -d' ' -f1) % 900 ))
echo "PORT=$port" >> .env.local
echo "COMPOSE_PROJECT_NAME=$name" >> .env.local
```

在给 Codex 的提示词里也提一句「开发服务器端口以 .env.local 里的 PORT 为准」，避免它默认去连 3000。

## 合并回主分支

每个 worktree 的任务完成后，它的分支上就是一组独立的提交。合并方式和普通分支没有区别：

```bash
# 在 worktree 里：确认改动、提交
cd ../shop-fix-login
git status
git add -A && git commit -m "fix: 登录会话为空时返回 401 而不是 500"

# 回到主目录：先看看要合进来什么
cd ~/work/shop
git log --oneline main..fix-login
git diff main...fix-login --stat

# 合并（或者 push 后开 PR，走团队的审查流程）
git merge --no-ff fix-login
```

几个实际建议：

- **先合小的、先合基础的**。两个任务都动了同一个模块时，先合并改动小的那个，再在另一个 worktree 里 `git rebase main` 解决冲突。这一步也可以交给那个 worktree 里的 Codex：「main 已经合入了 fix-login，请 rebase 到最新 main 并解决冲突，然后跑测试」。
- **合并前在 worktree 里跑全量测试**，不要只信 Codex 的总结。
- **合并后再审一次**：在主目录对合并结果跑一次测试，再用 `codex review "审查最近两次合并引入的改动"` 之类的指令过一遍，两个分别正确的改动放在一起也可能出问题。

## 清理

worktree 用完不清理，会留下大量目录和分支：

```bash
# 删除 worktree 目录（分支保留）
git worktree remove ../shop-fix-login

# 分支已合并，删掉
git branch -d fix-login

# 放弃的实验：强删目录和未合并的分支
git worktree remove --force ../shop-try-graphql
git branch -D try-graphql

# 目录被手动删掉或移动过，清理残留记录
git worktree prune

# 检查还剩哪些
git worktree list
```

::: tip 删除前确认没有漏掉的改动
`git worktree remove` 遇到未提交的改动会拒绝执行，这是好事。如果你要加 `--force`，先在那个目录里 `git status` 和 `git stash list` 看一眼。
:::

## 怎样拆分并行任务

并行不是开得越多越好。能并行的任务应该满足：

1. **改动范围不重叠**。「修登录 bug」和「订单页加导出」可以并行；「重构用户模块」和「给用户模块加字段」不要并行，合并时会大量冲突。
2. **各自能独立验收**。每个任务都有自己的测试或检查方法，不依赖另一个任务先完成。
3. **说明足够完整**。并行意味着你不会一直盯着，提示词要写清楚目标、范围、不能动的地方和验收方式（写法见 [提示词最佳实践](/codex/prompting)）。

一个实用的节奏：

| 阶段 | 做法 |
|---|---|
| 拆分 | 先在主目录用只读模式让 Codex 帮你列出任务清单，标出每个任务会动哪些文件 |
| 分派 | 挑互不重叠的 2–4 个任务，各建一个 worktree 开会话 |
| 巡检 | 隔一会儿看看每个会话，回答它的问题、批准审批请求 |
| 验收 | 完成一个就在它的 worktree 里看 diff、跑测试、提交 |
| 合并 | 按「先小后大」合并，冲突交给后合并的那个 worktree 处理 |
| 清理 | 合并后删 worktree 和分支 |

同时进行的任务数，大多数人 2–4 个就到了能认真审查的上限。再多，瓶颈就从「等 Codex」变成了「审不过来」。同时也要留意费用：每个会话都在独立消耗 token，HiveGPT 的用量可以在控制台按 Key 查看。

## 小结

- worktree 让同一仓库有多份独立工作目录，共享提交历史；一个 Codex 会话一个 worktree，互不干扰。
- 新 worktree 没有依赖、`.env` 和未提交改动，用初始化脚本补齐，并在 `AGENTS.md` 里写明。
- 桌面应用的 Worktree 模式自动在 `~/.codex/worktrees/` 下建分离 HEAD 的检出，默认保留最近 15 个。
- 端口、Compose 项目名、数据库名按 worktree 区分；合并时先小后大，合并后再跑一次测试和审查。
- 并行任务要范围不重叠、能独立验收，2–4 个通常就够了。

下一步：[工作流与实战示例](/codex/workflows)
