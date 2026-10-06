---
title: 非交互模式与 CI/CD
description: 用 codex exec 在脚本、定时任务和 CI 流水线里运行 Codex：参数、输入输出、退出码、结构化结果、GitHub Actions / GitLab CI 配置和 TypeScript SDK。
---

# 非交互模式与 CI/CD

平时用 Codex 是开一个终端界面（TUI），边看边聊。但有很多活并不需要人盯着：每晚给仓库跑一次依赖检查、每个 PR 自动出一份审查意见、把一批文件按同样的规则改一遍。这些场景要的是「给一段指令，跑完退出，结果写到文件里」，也就是 `codex exec`。

这一页讲清楚 `codex exec` 的参数和行为（以 codex-cli 0.142.3 的 `--help` 和官方源码为准），再给出能直接复制的脚本、GitHub Actions、GitLab CI 配置，以及用 TypeScript SDK 在自己的程序里调用 Codex 的方法。

## exec 和交互模式有什么不同

| 对比项 | `codex`（交互） | `codex exec`（非交互） |
|---|---|---|
| 界面 | 全屏 TUI | 无界面，进度打到 stderr |
| 结束方式 | 你退出才结束 | 当前任务完成就退出 |
| 审批 | 按审批策略弹窗询问 | **固定为 never**：从不询问，需要审批的操作直接判失败并告诉模型 |
| 沙箱 | 由配置或 `/permissions` 决定 | 由 `-s` 或配置决定，没配置时是只读 |
| 最终结果 | 显示在界面里 | 打印到 stdout，可用 `-o` 另存 |
| 适合 | 探索、需要来回讨论的任务 | 脚本、定时任务、CI |

要点是第三行：exec 里没有人可以点「同意」，所以它不接受 `-a/--ask-for-approval` 参数（传了会报 `unexpected argument '-a'`）。Codex 能做什么，完全由沙箱模式决定。

## 基本用法

```bash
# 最简单：在当前 Git 仓库里执行一条指令（默认只读沙箱）
codex exec "列出这个项目的主要模块，以及每个模块的入口文件"

# 允许改工作区里的文件
codex exec -s workspace-write "把 src/utils 里所有 var 改成 let/const，改完运行 npm test"

# 指定工作目录、模型和推理强度
codex exec -C ~/work/shop -m gpt-5.5 -c model_reasoning_effort="high" "找出订单金额计算的 bug 并修复"
```

`exec` 有一个别名 `e`，`codex e "..."` 等价。

### 指令从哪里来：参数、stdin 和两者合用

`codex exec` 读取指令有三种方式，规则来自 `--help` 的说明：

| 写法 | 行为 |
|---|---|
| `codex exec "指令"` | 用参数作为指令 |
| `codex exec` 且 stdin 是管道 | 从 stdin 读全部内容作为指令 |
| `codex exec -` | 强制从 stdin 读指令 |
| `codex exec "指令"` 且 stdin 也有管道输入 | 指令在前，stdin 内容作为附加材料，包在 `<stdin>` 块里附在后面 |

第四种最实用：把日志、diff、报错直接喂给它，指令写在参数里。

```bash
# 把失败的测试输出交给 Codex 分析
npm test 2>&1 | codex exec "下面是测试输出。找出失败原因，只给结论和修改建议，不要改文件"

# 把长指令写在文件里
codex exec - < prompts/nightly-check.md

# 多行指令用 heredoc
codex exec -s workspace-write - <<'EOF'
给 src/api/orders.ts 里的每个导出函数补上 JSDoc 注释：
- 说明参数和返回值
- 不要改动任何逻辑
- 完成后运行 npx tsc --noEmit 确认没有类型错误
EOF
```

::: tip 没有指令时会怎样
stdin 是终端（没有管道）又没给参数，exec 会提示 `No prompt provided` 并以退出码 1 结束；管道给了空内容也一样。脚本里不会因此卡住等输入。
:::

### 附带图片

`-i/--image` 可以附加一张或多张图片，适合「按设计稿实现」「解释这张报错截图」：

```bash
codex exec -s workspace-write -i design/login.png "按这张设计稿实现 src/pages/Login.vue，样式用项目里已有的 Tailwind 配置"
```

## 参数一览

以下是 `codex exec --help`（0.142.3）里的参数，按用途分组：

| 用途 | 参数 | 说明 |
|---|---|---|
| 模型 | `-m, --model` | 指定模型，例如 `gpt-5.5` |
| | `--oss`、`--local-provider` | 使用本地开源模型（lmstudio / ollama） |
| 权限 | `-s, --sandbox` | `read-only` / `workspace-write` / `danger-full-access` |
| | `--dangerously-bypass-approvals-and-sandbox` | 不要沙箱、不要审批。只能在本身已隔离的环境（一次性容器）里用 |
| | `--add-dir` | 额外允许写入的目录，可重复 |
| 目录 | `-C, --cd` | 指定工作根目录 |
| | `--skip-git-repo-check` | 允许在非 Git 目录里运行 |
| 配置 | `-c key=value` | 临时覆盖配置项，值按 TOML 解析 |
| | `-p, --profile` | 叠加 `$CODEX_HOME/<名字>.config.toml` |
| | `--enable` / `--disable` | 开关功能开关（feature flag） |
| | `--ignore-user-config` | 不读 `$CODEX_HOME/config.toml`（认证仍然用 CODEX_HOME 里的） |
| | `--ignore-rules` | 不加载 execpolicy 的 `.rules` 规则文件 |
| | `--strict-config` | 配置里有不认识的字段就报错 |
| 输出 | `--json` | 把事件以 JSONL 打到 stdout |
| | `-o, --output-last-message` | 把最后一条回复写到文件 |
| | `--output-schema` | 用 JSON Schema 约束最终回复的结构 |
| | `--color` | `always` / `never` / `auto` |
| 会话 | `--ephemeral` | 不把会话写到磁盘 |
| 其他 | `-i, --image` | 附加图片 |

::: warning --full-auto 已经废弃
很多旧教程写 `codex exec --full-auto`。在 0.142.3 里它还能用，但会打印警告「`--full-auto` is deprecated; use `--sandbox workspace-write` instead」；在官方仓库最新源码里这个参数已经删掉了。新脚本请直接写 `-s workspace-write`。另外，旧教程里的 `--stdin`、`--approval`、`--no-auto-approve`、`--output` 这些参数在 Codex CLI 里**都不存在**。
:::

### 沙箱怎么选

| 任务 | 推荐 | 原因 |
|---|---|---|
| 分析、审查、出报告 | 默认（只读） | 不需要写文件，最安全 |
| 改代码、跑测试 | `-s workspace-write` | 只能写工作目录和临时目录，默认不能联网 |
| 改代码且要装依赖 | `-s workspace-write -c sandbox_workspace_write.network_access=true` | 打开沙箱内的网络 |
| 一次性 Docker 容器里随便跑 | `--dangerously-bypass-approvals-and-sandbox` | 外层容器已经是隔离边界 |

workspace-write 下有一个容易踩的点：工作区里的 `.git` 和 `.codex` 目录始终是只读的。所以 Codex 自己执行 `git commit` 会失败（交互模式下会弹审批，exec 里直接失败）。在流水线里，**让 Codex 只改文件，提交和推送交给后面的脚本步骤**，这样权限也更清楚。沙箱细节见 [沙箱与审批](/codex/sandbox)。

## 处理输出

### 默认输出：人看的进度 + stdout 上的结果

不加 `--json` 时，执行过程（读了哪些文件、跑了什么命令）打到 **stderr**，最终回复打到 **stdout**。所以重定向 stdout 就能拿到干净的结果：

```bash
codex exec "总结最近 20 个提交做了什么，按功能分组" > summary.md
```

更稳妥的是 `-o`，它只把最后一条回复写进文件，不受其他输出干扰：

```bash
codex exec -o summary.md "总结最近 20 个提交做了什么，按功能分组"
```

### --json：给程序读的事件流

`--json` 让 stdout 变成 JSONL，每行一个事件。事件类型（来自源码 `exec_events.rs`）：

| 事件 `type` | 含义 |
|---|---|
| `thread.started` | 会话开始，带 `thread_id`，之后可以用它续接 |
| `turn.started` | 一轮开始 |
| `item.started` / `item.updated` / `item.completed` | 某个条目的开始、更新、完成 |
| `turn.completed` | 一轮完成，带 `usage`（输入、缓存、输出、推理 token 数） |
| `turn.failed` | 一轮失败，带 `error` |
| `error` | 出错 |

条目（`item`）的 `type` 有：`agent_message`（回复文字）、`reasoning`（推理摘要）、`command_execution`（执行的命令，含 `command`、`aggregated_output`、`exit_code`）、`file_change`（改了哪些文件）、`mcp_tool_call`、`web_search`、`todo_list`、`error` 等。

几个用 `jq` 处理的例子：

```bash
# 只取最终回复文字
codex exec --json "检查 README 里的安装步骤是否和 package.json 一致" \
  | jq -r 'select(.type=="item.completed" and .item.type=="agent_message") | .item.text'

# 统计这次用了多少 token
codex exec --json "解释 src/router.ts 的作用" \
  | jq 'select(.type=="turn.completed") | .usage'

# 列出 Codex 执行过的所有命令和退出码
codex exec --json -s workspace-write "运行测试并修复失败的用例" \
  | jq -r 'select(.type=="item.completed" and .item.type=="command_execution") | "\(.item.exit_code)\t\(.item.command)"'
```

### --output-schema：让结果变成固定结构的 JSON

如果结果要交给下一个程序处理（比如决定 CI 是否通过），用 JSON Schema 约束回复格式比在提示词里说「请输出 JSON」可靠得多。

```json
{
  "type": "object",
  "properties": {
    "risk": { "type": "string", "enum": ["low", "medium", "high"] },
    "summary": { "type": "string" },
    "issues": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "file": { "type": "string" },
          "line": { "type": "integer" },
          "problem": { "type": "string" }
        },
        "required": ["file", "line", "problem"],
        "additionalProperties": false
      }
    }
  },
  "required": ["risk", "summary", "issues"],
  "additionalProperties": false
}
```

```bash
codex exec --output-schema review.schema.json -o review.json \
  "审查当前分支相对 main 的改动，重点看安全和数据一致性问题"

# 高风险就让流水线失败
test "$(jq -r .risk review.json)" != "high"
```

::: tip Schema 写法
按 OpenAI 结构化输出的严格模式来写最稳：每个对象写 `"additionalProperties": false`，`properties` 里的字段全部列进 `required`；可选字段用 `["string", "null"]` 这样的联合类型表示。
:::

## 退出码

从源码看，exec 的退出码只有两种：

| 退出码 | 情况 |
|---|---|
| `0` | 任务正常完成 |
| `1` | 配置加载失败、没有指令、本轮失败或被中断、出现不再重试的错误等 |

注意：**退出码 0 只代表 Codex 正常跑完，不代表它把事情做对了**。比如它说「测试仍有 2 个失败」，退出码照样是 0。需要判断结果时，要么自己再跑一遍测试，要么用 `--output-schema` 让它给出结构化的结论再检查。

```bash
set -euo pipefail
codex exec -s workspace-write "修复 tests/test_price.py 里失败的用例，不要改测试本身"
pytest tests/test_price.py   # 用真实结果判断，而不是 Codex 的自述
```

## 续接会话：exec resume

exec 默认会把会话保存到 `~/.codex/sessions`，所以可以接着上一次继续：

```bash
# 第一步：只分析
codex exec "分析 src/billing 的结构，列出你认为需要重构的地方"

# 第二步：续接最近一次会话，在同一上下文里动手
codex exec resume --last -s workspace-write "按你刚才的第 1、2 条建议重构，完成后跑测试"

# 指定会话 ID（从 --json 的 thread.started 事件里拿）
codex exec resume 0199a7c1-xxxx "再补充单元测试"
```

`resume` 默认只在当前目录的会话里找最近一次，加 `--all` 可以跨目录。加了 `--ephemeral` 的运行不会落盘，自然也没法续接。

## 非交互代码审查

审查有专门的子命令，不用自己写「请审查改动」的提示词：

```bash
codex review --uncommitted                 # 审查暂存、未暂存和未跟踪的改动
codex review --base main                   # 审查当前分支相对 main 的改动
codex review --commit 3f2a9c1              # 审查某个提交引入的改动
codex review "审查当前分支相对 main 的改动，重点看 SQL 注入和权限校验"   # 自定义审查指令
```

`--uncommitted`、`--base`、`--commit` 三个预设目标和自定义指令是**互斥**的（源码里 `conflicts_with_all`），不能写成 `--base main "重点看……"`。想指定审查重点或输出语言，就只写自定义指令，在指令里说清楚审查范围；或者把「审查结论用中文写」这类长期要求放进 [AGENTS.md](/codex/agents-md)。

`codex exec review` 是同一个功能的 exec 形式，额外支持 `--json`、`-o` 等输出参数，放进流水线时更方便。

## 脚本化示例

### 示例 1：批量重构，每个文件一次独立运行

一次让 Codex 改几十个文件，容易漏改或改得不一致。按文件拆开，每次只给一个文件，失败了也只影响一个：

```bash
#!/usr/bin/env bash
set -uo pipefail

mkdir -p .codex-logs
failed=()

for f in $(git ls-files 'src/**/*.js'); do
  echo "==> $f"
  if ! codex exec -s workspace-write --ephemeral \
      -o ".codex-logs/$(echo "$f" | tr '/' '_').md" \
      "只修改 $f 这一个文件：把回调风格的异步代码改成 async/await，保持导出接口不变。不要修改其他文件。"; then
    failed+=("$f")
  fi
done

npm test || echo "测试未通过，请检查改动"
printf '失败的文件：%s\n' "${failed[@]:-无}"
git diff --stat
```

跑完先看 `git diff`，满意再提交；不满意 `git checkout -- .` 一键回退。

### 示例 2：根据提交记录生成变更日志

```bash
#!/usr/bin/env bash
set -euo pipefail
last_tag=$(git describe --tags --abbrev=0)

git log "$last_tag"..HEAD --pretty=format:'%h %s%n%b' \
  | codex exec --ephemeral -o CHANGELOG.draft.md \
    "下面是自 $last_tag 以来的提交记录。写一段中文变更日志：
     分「新功能」「修复」「其他」三节；每条一句话，写用户能感知到的变化；
     忽略纯格式化、依赖小版本升级这类提交；不要编造提交里没有的内容。"

echo "草稿已写入 CHANGELOG.draft.md，确认后合并到 CHANGELOG.md"
```

这里 Codex 只需要读 stdin，默认的只读沙箱就够了。

### 示例 3：提交前自动审查

放到 `.git/hooks/pre-push`（记得 `chmod +x`），推送前对改动做一次快速审查，发现高风险问题就拦下：

```bash
#!/usr/bin/env bash
set -euo pipefail
schema=$(mktemp)
cat > "$schema" <<'EOF'
{"type":"object","properties":{"block":{"type":"boolean"},"reason":{"type":"string"}},
 "required":["block","reason"],"additionalProperties":false}
EOF

result=$(git diff origin/main...HEAD | codex exec --ephemeral --output-schema "$schema" \
  "下面是即将推送的改动。只在发现明确的安全漏洞、密钥泄露或必然导致崩溃的错误时 block=true，风格问题不要拦。")

if [ "$(echo "$result" | jq -r .block)" = "true" ]; then
  echo "推送被拦下：$(echo "$result" | jq -r .reason)"
  exit 1
fi
```

## 在 GitHub Actions 里使用

### 准备认证：用 HiveGPT Key

CI 里没有浏览器，不能 `codex login`。用 HiveGPT 的 Key 时，推荐**自定义一个 provider，用 `env_key` 指向环境变量**：Key 只存在 GitHub Secrets 里，不落进任何文件。

1. 在 HiveGPT「API 密钥」页为 CI 单独创建一个 OpenAI 类分组（如「GPT-按量」）的 Key，方便单独限额和吊销。获取 Key 的完整步骤见 [接入 HiveGPT](/codex/hivegpt)。
2. 仓库 Settings → Secrets and variables → Actions，新建 secret `HIVEGPT_API_KEY`。
3. （可选）把「使用密钥」弹窗里下载的 `codex-models.json` 提交到仓库，例如 `.github/codex/codex-models.json`，这样 Codex 能认出 gpt-5.5 的上下文长度等信息。

工作流里在一个临时的 `CODEX_HOME` 中写出配置：

```yaml
# .github/workflows/codex-review.yml
name: Codex Review
on:
  pull_request:
    types: [opened, synchronize]

permissions:
  contents: read
  pull-requests: write

jobs:
  review:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    env:
      CODEX_HOME: ${{ runner.temp }}/codex-home
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # 审查需要 base 分支的历史

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Install Codex CLI
        run: npm install -g @openai/codex

      - name: Write Codex config
        run: |
          mkdir -p "$CODEX_HOME"
          cat > "$CODEX_HOME/config.toml" <<EOF
          model_provider = "hivegpt"
          model = "gpt-5.5"
          model_catalog_json = "$GITHUB_WORKSPACE/.github/codex/codex-models.json"

          [model_providers.hivegpt]
          name = "HiveGPT"
          base_url = "https://hivegpt.cn/v1"
          wire_api = "responses"
          env_key = "HIVEGPT_API_KEY"
          EOF

      - name: Run review
        env:
          HIVEGPT_API_KEY: ${{ secrets.HIVEGPT_API_KEY }}
        run: |
          codex exec review -o review.md \
            "审查当前分支相对 origin/${{ github.base_ref }} 的改动（git diff origin/${{ github.base_ref }}...HEAD）。用中文输出，按严重程度排序，每条给出文件和行号。"

      - name: Comment on PR
        env:
          GH_TOKEN: ${{ github.token }}
        run: gh pr comment ${{ github.event.pull_request.number }} --body-file review.md
```

几个设计点：

- `CODEX_HOME` 指向 runner 的临时目录，作业结束就清理，不会把配置和会话留在共享的 runner 上。
- 审查只读代码，沿用默认的只读沙箱。
- 发评论用的是 GitHub 自带的 `github.token`，Codex 本身拿不到它，也不需要。
- 没有提交 `codex-models.json` 时，删掉 `model_catalog_json` 那一行也能跑，只是 Codex 会把 gpt-5.5 当成未知模型、使用默认参数。

::: details 另一种写法：直接写 config.toml + auth.json
如果你想和本机保持完全一样的配置，也可以把「使用密钥」弹窗生成的 `config.toml` 存成仓库变量，`auth.json` 的内容 `{"OPENAI_API_KEY": "你的 Key"}` 存成 secret，在步骤里写出这两个文件：

```yaml
- name: Write Codex config
  env:
    CODEX_CONFIG: ${{ vars.CODEX_CONFIG_TOML }}
    CODEX_AUTH: ${{ secrets.CODEX_AUTH_JSON }}
  run: |
    mkdir -p "$CODEX_HOME"
    printf '%s' "$CODEX_CONFIG" > "$CODEX_HOME/config.toml"
    printf '%s' "$CODEX_AUTH" > "$CODEX_HOME/auth.json"
    chmod 600 "$CODEX_HOME/auth.json"
```

缺点是 Key 会以文件形式出现在 runner 磁盘上；`env_key` 方式更干净。
:::

### 让 Codex 修改代码并开 PR

需要写文件时，用 `workspace-write`，然后由工作流自己提交（前面说过，沙箱里 `.git` 是只读的）：

```yaml
      - name: Fix lint errors
        env:
          HIVEGPT_API_KEY: ${{ secrets.HIVEGPT_API_KEY }}
        run: |
          codex exec -s workspace-write \
            "运行 npm run lint，修复所有报错。只改源码，不要改 eslint 配置，不要加 eslint-disable 注释。"

      - name: Verify
        run: npm run lint && npm test

      - name: Open PR
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          git switch -c codex/lint-fix-${{ github.run_id }}
          git -c user.name="codex-bot" -c user.email="codex-bot@users.noreply.github.com" \
            commit -am "chore: fix lint errors (by Codex)"
          git push -u origin HEAD
          gh pr create --fill --base main
```

这个作业需要 `contents: write` 和 `pull-requests: write` 权限。「Verify」步骤是关键：Codex 说修好了不算，命令通过才算。

### 官方的 openai/codex-action

OpenAI 提供了官方 Action `openai/codex-action`，它负责安装 CLI、配置认证、以较安全的方式运行 `codex exec`（比如默认放弃 sudo 权限、限制谁能触发），并把最终回复作为步骤输出。它主要面向 OpenAI API Key，输入项和安全策略以它仓库 README 为准。用 HiveGPT Key 时，上面「自己安装 CLI + env_key」的写法更直接，也更容易排查问题。

### Linux runner 上的沙箱

Linux 上 Codex 用 bubblewrap 做文件系统沙箱，需要能创建用户命名空间（user namespace）。部分 CI 环境（某些容器、加了限制的内核）不允许，这时 Codex 会在启动时给出警告，沙箱里的命令会失败。可选的处理：

- 优先换成允许用户命名空间的 runner，或在自建 runner 上放开限制；
- 如果 Codex 本身就运行在一次性的、没有任何密钥挂载的容器里，可以用 `--dangerously-bypass-approvals-and-sandbox`，把容器当作隔离边界。不要在挂着生产凭据的机器上这样做。

## 在 GitLab CI 里使用

思路完全一样：Key 存成 CI/CD 变量（勾选 Masked 和 Protected），作业里写出配置后运行。

```yaml
# .gitlab-ci.yml
codex_review:
  image: node:22
  stage: test
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  variables:
    CODEX_HOME: "$CI_PROJECT_DIR/.codex-home"
    GIT_DEPTH: 0
  before_script:
    - npm install -g @openai/codex
    - mkdir -p "$CODEX_HOME"
    - |
      cat > "$CODEX_HOME/config.toml" <<EOF
      model_provider = "hivegpt"
      model = "gpt-5.5"

      [model_providers.hivegpt]
      name = "HiveGPT"
      base_url = "https://hivegpt.cn/v1"
      wire_api = "responses"
      env_key = "HIVEGPT_API_KEY"
      EOF
  script:
    - git fetch origin "$CI_MERGE_REQUEST_TARGET_BRANCH_NAME"
    - codex exec review -o review.md "审查当前分支相对 origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME 的改动，用中文输出，按严重程度排序"
  artifacts:
    when: always
    paths:
      - review.md
    expire_in: 7 days
```

`HIVEGPT_API_KEY` 在项目的 Settings → CI/CD → Variables 里设置，作业会自动拿到这个环境变量。

## CI 安全清单

| 风险 | 做法 |
|---|---|
| 密钥泄露 | Key 只放 Secrets / Masked 变量；用 `env_key` 而不是写进文件；CI 专用 Key，单独限额，泄露就吊销 |
| 提示词注入 | PR 内容是外部输入。不要对 fork 来的 PR 用 `pull_request_target` 运行能写仓库的 Codex 作业 |
| 权限过大 | 审查类作业保持只读沙箱；GitHub `permissions` 只给需要的那几项 |
| 改错代码 | Codex 只改文件，提交前跑 lint/测试；改动以 PR 形式提出，由人合并 |
| 费用失控 | 设 `timeout-minutes`；用 `rules` / `paths` 限制触发条件；给 Key 设额度 |
| 状态残留 | `CODEX_HOME` 用临时目录；不需要续接的任务加 `--ephemeral` |

::: warning 不要把 Key 交给模型
不要在提示词里写 Key，也不要让 Codex 执行 `env`、`printenv` 之类会把环境变量打印进日志的命令。按源码，Codex 默认把完整的环境变量传给它执行的命令；在 CI 里建议加上 `-c shell_environment_policy.ignore_default_excludes=false`，让它过滤掉名字里含 `KEY`、`SECRET`、`TOKEN` 的变量（不区分大小写）。这只影响 Codex 执行的命令，不影响它自己读取 `env_key` 去调用模型。
:::

## 用 TypeScript SDK 调用

想在自己的 Node.js 程序里驱动 Codex（内部工具、机器人、批处理服务），可以用官方 SDK `@openai/codex-sdk`。它会启动本机的 `codex` CLI，通过 JSONL 事件和它通信，所以机器上要先装好 `@openai/codex`。要求 Node.js 18+。

```bash
npm install @openai/codex-sdk
```

用 HiveGPT 时，通过 `config` 选项传入 provider 配置（SDK 会把它展开成 `--config` 参数），Key 从环境变量读：

```typescript
import { Codex } from "@openai/codex-sdk";

const codex = new Codex({
  config: {
    model_provider: "hivegpt",
    model_providers: {
      hivegpt: {
        name: "HiveGPT",
        base_url: "https://hivegpt.cn/v1",
        wire_api: "responses",
        env_key: "HIVEGPT_API_KEY", // 运行前 export HIVEGPT_API_KEY=...
      },
    },
  },
});

const thread = codex.startThread({
  model: "gpt-5.5",
  workingDirectory: "/path/to/repo",
  sandboxMode: "workspace-write",
});

const turn = await thread.run("运行测试，找出失败原因并修复");
console.log(turn.finalResponse); // 最终回复
console.log(turn.items);         // 过程中的命令、文件改动等

// 同一个 thread 上再次 run，就是在同一会话里继续
await thread.run("给刚才修复的地方补一个回归测试");
```

几个常用能力：

| 需求 | 写法 |
|---|---|
| 实时拿到过程事件 | `const { events } = await thread.runStreamed("...")`，再 `for await (const e of events)` |
| 结构化输出 | `thread.run("...", { outputSchema: schema })`，`schema` 是普通 JSON Schema 对象 |
| 附带图片 | `thread.run([{ type: "text", text: "..." }, { type: "local_image", path: "./ui.png" }])` |
| 续接旧会话 | `codex.resumeThread(threadId)`，会话保存在 `~/.codex/sessions` |
| 非 Git 目录 | `startThread({ skipGitRepoCheck: true })` |
| 取消 | `thread.run("...", { signal: abortController.signal })` |

`startThread` 还接受 `modelReasoningEffort`、`networkAccessEnabled`、`webSearchMode`、`approvalPolicy`、`additionalDirectories` 等选项，含义和 CLI 对应参数一致。另外，如果给 `new Codex()` 传了 `env`，子进程就**不再继承** `process.env`，记得把 `PATH` 和 Key 一起传进去。

官方仓库 `sdk/` 目录下还有 Python SDK，用法思路相同。

## 小结

- `codex exec` 跑完即退出：审批固定为 never，能做什么由 `-s` 沙箱决定，默认只读；`--full-auto` 已废弃，改用 `-s workspace-write`。
- 指令可以来自参数、stdin，或「参数 + stdin 附加材料」；结果用 `-o` 存文件，`--json` 输出事件流，`--output-schema` 拿结构化 JSON。
- 退出码只区分 0 和 1，0 不等于做对了，验收要靠测试和结构化结论。
- CI 里用 HiveGPT Key：自定义 provider + `env_key`，Key 放 Secrets，`CODEX_HOME` 用临时目录；Codex 只改文件，提交、推送和发评论交给工作流步骤。
- 在程序里调用用 `@openai/codex-sdk`，provider 配置通过 `config` 选项传入。

下一步：[Worktree 与并行任务](/codex/worktrees)
