---
title: 模型与推理强度
description: 在 Codex 里选择模型、设置推理强度、推理摘要和回复详略，按任务类型给出推荐组合，并说明使用 HiveGPT 和第三方模型时的注意事项。
---

# 模型与推理强度

同一个任务，换个模型或推理强度，结果可能差很多：简单的改名重构用最高推理强度，是在浪费时间和 token；排查并发 bug 用最低推理强度，它可能三轮都找不到原因。这一页讲 Codex 里和模型有关的几个旋钮：选哪个模型、推理强度怎么调、推理摘要和回复详略是什么，以及不同任务该怎么搭配。

## 在哪里选模型

有三种方式，作用范围不同：

| 方式 | 写法 | 作用范围 |
|---|---|---|
| 会话内切换 | `/model` | 立即生效，并保存为默认值 |
| 命令行参数 | `codex -m gpt-5.5` | 只对这一次运行有效 |
| 配置文件 | `model = "gpt-5.5"` | 长期默认 |

```bash
codex -m gpt-5.5                      # 交互模式指定模型
codex exec -m gpt-5.5 "补全单元测试"    # 非交互模式同样可用
```

```toml
# ~/.codex/config.toml
model = "gpt-5.5"
```

在会话里输入 `/model`，会先列出可选模型，再列出该模型支持的推理强度。选完从下一轮生效，同时写回配置文件，下次启动沿用。

### HiveGPT 下有哪些模型

通过 [接入 HiveGPT](/codex/hivegpt) 使用 Codex 时，当前默认模型是 **gpt-5.5**。可选模型由两样东西决定：

- 你的 Key 所在分组开放了哪些模型；
- 配置里的 `model_catalog_json = "~/.codex/codex-models.json"`，也就是「使用密钥」弹窗里下载的模型目录文件。`/model` 的列表就来自这里。

所以，**以 `/model` 里列出的为准**。HiveGPT 上线新模型后，重新下载一次模型目录文件即可在列表里看到。想看 Codex 当前加载的完整模型目录（含每个模型支持的推理强度、上下文窗口），可以运行：

```bash
codex debug models
```

::: tip 为什么需要模型目录
Codex 需要知道每个模型的上下文窗口、支持哪些推理强度、默认用什么设置，才能正确地压缩上下文、显示选项。官方登录时这些信息从 OpenAI 获取；通过自定义提供方接入时，就由 `model_catalog_json` 提供。缺了它，`/model` 可能列不出模型，或者推理强度选项不对。
:::

## 推理强度

推理模型在回答前会先「想」一段，`model_reasoning_effort` 决定想多久。

```toml
model_reasoning_effort = "medium"
```

可选值由模型自己声明。以 gpt-5.5 为例，Codex 内置目录里它支持四档，默认 `medium`：

| 推理强度 | 速度 | token 消耗 | 适合 |
|---|---|---|---|
| `low` | 最快 | 最少 | 改文案、重命名、格式调整、照着现成模式补代码 |
| `medium` | 适中 | 适中 | 日常开发的默认档：写功能、修一般的 bug、写测试 |
| `high` | 较慢 | 较多 | 跨文件重构、复杂业务逻辑、不熟悉的代码库 |
| `xhigh` | 最慢 | 最多 | 疑难 bug（并发、内存、偶发问题）、架构设计、安全审查 |

源码里推理强度的取值范围更大，还有 `none`、`minimal`、`max`、`ultra` 等，但**某个模型实际支持哪些，以 `/model` 里显示的为准**。写了模型不支持的值，Codex 会提示或回退。

几个经验：

- **先用 medium，不行再升**。多数任务 medium 就够；升档前先想想是不是任务描述不清楚，描述清楚往往比加推理强度更有效。
- **规划和执行可以分开**。`plan_mode_reasoning_effort` 单独设置计划模式（`/plan`）的推理强度，方案阶段多想一点，执行阶段用常规档：

  ```toml
  model_reasoning_effort = "medium"
  plan_mode_reasoning_effort = "high"
  ```

- **临时提升不要改默认值**。`/model` 会把选择存成默认值，攻坚完记得调回来；只用一次的话，用命令行更干净：

  ```bash
  codex -c 'model_reasoning_effort="xhigh"' "这个死锁在高并发下才出现，帮我找原因"
  ```

  或者建一个 profile：`~/.codex/deep.config.toml` 里写 `model_reasoning_effort = "xhigh"`，需要时 `codex -p deep`。见 [高级配置](/codex/config-advanced)。

## 推理摘要

模型的推理过程本身不直接展示，Codex 显示的是它的**摘要**，让你知道它在想什么、打算怎么做。

```toml
model_reasoning_summary = "auto"   # auto / concise / detailed / none
```

| 值 | 效果 |
|---|---|
| `auto` | 默认，由模型决定详略 |
| `concise` | 简短摘要 |
| `detailed` | 详细摘要，适合想跟上它思路、学习它怎么排查问题 |
| `none` | 不生成摘要 |

界面上不想看推理过程，用 `hide_agent_reasoning = true` 隐藏，这只影响显示，不影响模型推理本身。

## 回复详略

`model_verbosity` 控制模型最终回复写多长：

```toml
model_verbosity = "low"   # low / medium / high
```

`low` 时回复干脆，改完代码只说结果和要点；`high` 时会详细解释每一处改动。gpt-5.5 在内置目录里默认是 `low`。学习阶段想多看解释，可以调到 `medium` 或 `high`；熟练之后 `low` 更省 token 也更好读。

## 按任务推荐的组合

| 任务 | 推理强度 | 推理摘要 | 回复详略 | 备注 |
|---|---|---|---|---|
| 改文案、样式微调、重命名 | `low` | `auto` | `low` | 追求速度 |
| 日常功能开发、写测试 | `medium` | `auto` | `low` | 默认就是它 |
| 学习陌生项目、让它讲解代码 | `medium` | `detailed` | `high` | 配合 `-s read-only` |
| 跨模块重构 | `high` | `auto` | `medium` | 先 `/plan` 对齐方案 |
| 疑难 bug、性能问题 | `xhigh` | `detailed` | `medium` | 给足复现步骤和日志 |
| 代码审阅 `/review` | `high` | `auto` | `medium` | 可用 `review_model` 单独指定模型 |
| CI 里的批量小任务 | `low` 或 `medium` | `none` | `low` | 省 token、跑得快 |

`review_model` 让 `/review` 和 `codex review` 使用单独的模型，主会话不受影响：

```toml
review_model = "gpt-5.5"
```

## 速度、成本和质量怎么权衡

按量计费时（例如 HiveGPT 的「GPT-按量」分组），费用主要由 token 决定，而推理 token 也计入输出。影响花费的因素，按影响大小大致是：

1. **上下文长度**。每一轮请求都会带上当前会话的历史，会话越长每轮越贵。换任务就 `/new`，长任务适时 `/compact`。
2. **推理强度**。档位越高，思考 token 越多，耗时也越长。
3. **无效的来回**。任务描述含糊导致返工，比任何参数都费钱。把目标、约束、验收标准一次说清楚，见 [提示词最佳实践](/codex/prompting)。
4. **回复详略**。影响相对小，但积少成多。

一个实用的做法：日常保持 `medium` + `low` 详略，只在明确卡住时提升推理强度；每完成一个独立任务就 `/new`。消耗明细可以在会话里用 `/status` 看本次 token，或在 HiveGPT 的「使用记录」里看每次调用的扣费。

有的模型还提供不同的服务档位（例如更快但更贵的优先处理），对应配置项 `service_tier`，会话里 `/model` 下方也可能出现切换命令。是否可用、怎么计费，以服务方说明为准。

## 上下文窗口与自动压缩

Codex 会根据模型目录里的上下文窗口大小，在快满时自动压缩历史。gpt-5.5 在内置目录里的上下文窗口是 272K token。相关配置：

```toml
model_auto_compact_token_limit = 200000   # 达到多少 token 时自动压缩
model_context_window = 272000             # 一般不用写，模型目录里没有时才手动指定
```

压缩会丢失细节。如果发现 Codex 在长会话后期「忘了」之前的约定，把关键约定写进 `AGENTS.md`，它每次都会读，不受压缩影响。

## 使用第三方模型的注意事项

Codex 可以通过 `model_providers` 接入其他兼容 OpenAI Responses 接口的模型，具体步骤见 [接入 DeepSeek 等其他模型](/codex/providers)。换用第三方模型时要留意：

- **协议**：当前 Codex 只支持 Responses 接口（`wire_api = "responses"`），只提供 Chat Completions 的服务需要对方或中间层做转换。
- **推理参数**：`model_reasoning_effort`、`model_verbosity`、`model_reasoning_summary` 是 OpenAI 模型的参数，第三方模型可能不支持或含义不同，按对方文档设置；不支持时不要写，避免请求报错。
- **模型目录**：没有对应的目录信息时，Codex 不知道上下文窗口，可以用 `model_context_window` 手动指定，否则自动压缩的时机可能不准。
- **工具调用能力**：Codex 高度依赖模型的工具调用和长上下文能力，弱一些的模型可能频繁调用失败、改错文件。先用小任务试，确认靠谱再用到正式项目。
- **内置功能**：网页搜索等依赖 OpenAI 服务端的功能，在第三方提供方上通常不可用。

## 小结

- 选模型：`/model`（会存为默认）、`-m`（只这一次）、`model =`（长期默认）；HiveGPT 下可选模型以 `/model` 列表为准，默认 gpt-5.5。
- 推理强度先用 `medium`，卡住再升到 `high` / `xhigh`；可选档位由模型决定，gpt-5.5 支持 low 到 xhigh。
- `model_reasoning_summary` 管推理摘要，`model_verbosity` 管回复长短，`plan_mode_reasoning_effort` 单独管计划模式。
- 花费主要来自上下文长度和推理强度：换任务 `/new`，长任务 `/compact`，临时攻坚用命令行或 profile。
- 第三方模型只支持 Responses 协议，推理类参数按对方文档设置。

下一步：[提示词最佳实践](/codex/prompting)
