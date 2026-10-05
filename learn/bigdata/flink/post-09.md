---
title: "Flink 2.x 源码精读（九·终）：从读者到贡献者"
description: "2026 年 9 月，Flink 主仓库 133 个 commit 中有 35 个声明使用了 AI 工具。社区的规则是什么？一个合格的 bug 修复长什么样？读完这个系列，有哪些可以动手的第一批任务？全系列收官。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-09-cover.webp"}]]
---

# Flink 2.x 源码精读（九·终）：从读者到贡献者

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-09-cover.webp" alt="Yui和Kai从源码阅读者走向Flink社区贡献者" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai从源码阅读者走向Flink社区贡献者<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**。本篇的社区数据来自本地仓库中的 `origin/master`（最新 commit 为 2026-09-24），规则引用自同一版本的 `AGENTS.md` 和 `.github/PULL_REQUEST_TEMPLATE.md`。**规则以官方文件和官网为准**，本文只是导读。
> 统计命令都在文中给出，你可以在自己的仓库里复现。

---

先看一组数据。

对 Flink 主仓库最近一年（2025-09-24 ～ 2026-09-24）的 commit 做统计，看 commit message 里有没有 `Generated-by:` 这一行。这是 Apache 软件基金会要求的 AI 辅助声明格式：

| 月份 | commit 数 | 带 `Generated-by:` 的 commit | 占比 |
|---|---|---|---|
| 2025-09-24 ～ 2026-03 | 587 | 0 | 0% |
| 2026-04 | 120 | 3 | 3% |
| 2026-05 | 124 | 2 | 2% |
| 2026-06 | 135 | 24 | 18% |
| 2026-07 | 176 | 36 | 20% |
| 2026-08 | 128 | 10 | 8% |
| 2026-09（截至 24 日） | 133 | 35 | **26%** |

> 统计口径：commit message 中某一行以 `Generated-by:` 开头。没有声明、但实际用了 AI 的 commit 统计不到，所以真实比例只会更高。表中各行合计 1403 个 commit。

**最近一个月，Flink 主仓库大约四分之一的 commit 公开声明使用了 AI 工具。** 116 行声明里，以 Claude 开头的有 99 行，其次是 OpenAI Codex（写法不一，合计 13 行）、GitHub Copilot CLI（3 行）等。

数字从 2026 年 4 月开始出现，并不是因为大家那时才开始用 AI，而是因为 **2026-04-17**，社区通过 FLINK-39477 在仓库里加入了 `AGENTS.md`，并在 PR 模板里加上了 AI 使用声明。

这一篇是全系列的最后一讲。我们来看：

1. 社区对 AI 辅助贡献的规则是什么？
2. 一个改动从想法到被合入，要走哪些步骤？
3. 一个合格的 bug 修复长什么样？（真实案例 + 红绿验证）
4. 读完这个系列，有哪些可以动手的第一批任务？

---

## 一、社区现状

同一时间段的整体数据：

| 指标 | 数值 |
|---|---|
| commit 总数 | **1403** |
| 不同的作者 | **169** |
| 以 `[FLINK-XXXX]` 开头的 commit | 1198（85%） |
| 以 `[hotfix]` 开头的 commit | 187（13%） |
| 带 `Generated-by:` 的 commit | **110**（共 116 行声明，有的 commit 声明了不止一个工具） |

复现命令（在你的 Flink 仓库里执行）：

```bash
git log origin/master --since=2025-09-24 --grep='^Generated-by:' --oneline | wc -l
```

两点观察：

- **169 位作者**，说明这是一个很开放的社区，远不是只有少数几个核心开发者在写代码；
- **AI 辅助已经很普遍，社区的态度是：接受，但要求透明，要求作者负责。** 下一节是具体规则。

---

## 二、AI 辅助贡献的规则

`AGENTS.md` 的 "AI-assisted contributions" 一节（`AGENTS.md:300-306`）和 PR 模板的最后一节（`.github/PULL_REQUEST_TEMPLATE.md:78-93`），遵循 [ASF Generative Tooling Guidance](https://www.apache.org/legal/generative-tooling.html)。要点：

1. **必须声明**：在 PR 模板里勾选 "Was generative AI tooling used to co-author this PR?"，并填写 `Generated-by` 这一行；
2. **commit message 里加上** `Generated-by: <工具名和版本>`；
3. **不要把 AI 写成共同作者**：禁止用 `Co-Authored-By: <AI>`。原文是 "agents are assistants, not authors"（`:304`，"Never" 一节 `:339` 又强调了一次）；
4. **你要为质量负全责**：必须能讲清楚设计、代码和测试，能调试，能对 review 意见做出实质性的回应；
5. **低质量的 AI PR 会被直接关闭，不做 review**。`AGENTS.md` 列举了几种典型表现：大段没人审阅过的文字、只有框架没有实际行为的代码、没有真正测到改动的测试、注水的 commit message。

PR 模板的注释里也写得很直白：

> （大意）无论用了什么工具，你都要为这个 PR 里每一处改动的质量和正确性负责。低质量的 AI 生成 PR 会被关闭。

### 2.1 一个值得注意的细节

很多 AI 编程工具默认会在 commit 里加一行 `Co-Authored-By`，写这个系列时用的工具也是这样。

在同一时间段里，我统计到 **11 个 commit 把 Claude 写成了 `Co-Authored-By`**：2 个在规则发布之前，1 个在发布当天，8 个在发布之后。这 11 个 commit 都没有 `Generated-by`。

```bash
git log origin/master --since=2025-09-24 -i -E --grep='^Co-authored-by:.*noreply@anthropic' --oneline | wc -l
```

> 这里只给统计数字，不讨论具体是哪些 commit。commit message 在合入时可能经过修改，这个数字只能说明：**工具的默认行为惯性很强**。

**给 Flink 提交代码时，以项目自己的规则为准：用 `Generated-by`，不用 `Co-Authored-By`。**

### 2.2 `AGENTS.md` 是写给谁的

`AGENTS.md` 的标题是 "Flink AI Agent Instructions"（`:20`），它首先是写给 AI 编程工具读的：大多数 AI 编程工具会自动读取仓库根目录下的这类文件。

但它对人同样有用。它把散落在官网、Wiki、邮件列表里的规则集中到了一处：构建命令、测试规范、commit 格式、哪些改动"必须先问"（"Ask first"，`:322-329`）、哪些事"绝对不要做"（"Never"，`:331-343`）。**第一次给 Flink 提交代码之前，值得从头到尾读一遍。**

`origin/master` 上的版本在 7、8 月又加入了 "Code Review Guidelines" 一节（`:308-318`）：比如测试中避免在重试循环之外使用 `Thread.sleep`、新功能如果由配置项控制，要在 INFO 日志中打印它的实际取值。这些也是 review 别人代码时的好清单。

---

## 三、一个改动的完整生命周期

<figure class="ai-figure"><img src="/bigdata-img/flink/post-09-2.webp" alt="Yui和Kai沿着贡献流程从问题发现走向社区合并" width="960" height="640" loading="lazy" /><figcaption>Yui和Kai沿着贡献流程从问题发现走向社区合并<span>AI 生成配图</span></figcaption></figure>

```text
① 发现问题 / 想法
② 在 JIRA 搜索是否已经有人报过（issues.apache.org/jira/projects/FLINK）
③ 新建或认领 JIRA，描述问题、复现方式、打算怎么改
   较大的改动：先在 dev@ 邮件列表讨论
   涉及 @Public / @PublicEvolving / @Experimental API：需要 FLIP + 投票
④ 等 committer 把 issue assign 给你
⑤ 在自己的 fork 上建分支、写代码、写测试
⑥ 本地检查：spotless、相关测试、架构测试
⑦ 提 PR：标题 [FLINK-XXXX][component] ...，按模板填写
⑧ CI + review，反复修改
⑨ committer 合入；用户可见的变化，在 JIRA 的 Release Notes 字段里写说明
```

几条容易被忽视的规则（`AGENTS.md` "Commits and PRs" `:279-298` 和 "Boundaries" `:320-343`）：

- **每个 PR 只解决一个问题**；清理、重构和功能改动要分成**不同的 commit**；
- 永远推送到**自己的 fork**，不要直接推到 `apache/flink`；
- 只有错别字可以不建 JIRA，用 `[hotfix][component] ...`；
- 以下改动**必须先问**：修改公共 API 的稳定性注解、大范围跨模块重构、新增依赖、修改序列化格式、修改 Checkpoint/Savepoint 的行为、可能影响热路径性能的改动；
- 以下事情**绝对不要做**：绕过 checkstyle、新功能使用 Java 序列化、在连接器中使用旧的 `SourceFunction`/`SinkFunction`、把 AI 写成 `Co-Authored-By`……

本地常用的命令（`AGENTS.md` "Commands"）：

| 目的 | 命令 |
|---|---|
| 快速编译 | `./mvnw clean install -DskipTests -Dfast -Pskip-webui-build -T1C` |
| 单个测试类 | `./mvnw -pl flink-core-api -Dtest=MemorySizeTest test` |
| 格式化（每次修改后都要跑） | `./mvnw spotless:apply` |

> 本系列第一讲之前编译 Flink 时，我曾以为 `-Pskip-webui-build` 不影响使用，结果 Web UI 打不开（404）。如果你需要看 Web UI（比如第一讲的实验），就不要加这个参数。

---

## 四、真实案例：FLINK-38932 与红绿验证

<figure class="ai-figure"><img src="/bigdata-img/flink/post-09-1.webp" alt="Yui和Kai验证处理时间定时器的红绿测试流程" width="960" height="640" loading="lazy" /><figcaption>Yui和Kai验证处理时间定时器的红绿测试流程<span>AI 生成配图</span></figcaption></figure>

### 4.1 为什么选它

```text
d30cf4d9ede Zhanghao Chen 2026-01-29
[FLINK-38932][runtime] Fix incorrect scheduled timestamp in ProcessingTimeCallback with scheduleWithFixedDelay (#27429)

 .../runtime/tasks/SystemProcessingTimeService.java | 17 +++--
 .../tasks/SystemProcessingTimeServiceTest.java     | 80 +++++++++++++++++++++-
 2 files changed, 89 insertions(+), 8 deletions(-)
```

- 改的是第一讲下篇讲过的 Mailbox 里的**处理时间定时器**；
- 改动很小：17 行生产代码 + 80 行测试，很适合作为第一个 PR 的参照；
- 2.3.0 已经包含了这个修复，可以在本地完整复现红绿验证。

### 4.2 问题与修复

`SystemProcessingTimeService` 有两种周期调度：

- `scheduleAtFixedRate`：固定**频率**，第 n 次的计划时间 = 初始时间 + n × period；
- `scheduleWithFixedDelay`：固定**间隔**，下一次的计划时间 = **上一次执行结束的时间** + period。

修复前，两者共用一套计算：`nextTimestamp += period`。所以 `scheduleWithFixedDelay` 传给回调的时间戳是错的：没有把回调本身的执行时间算进去。

修复后（`flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java:327-328`）：

```java
nextTimestamp =
        fixedDelay ? System.currentTimeMillis() + period : nextTimestamp + period;
```

测试（`flink-runtime/src/test/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeServiceTest.java`）做了两件事：

1. **加强已有的** `testScheduleAtFixedRate`（`:51`）：原来只检查"触发了多次"，现在还检查每次拿到的时间戳；
2. **新增** `testScheduleAtFixedDelay`（`:100`）：回调里 sleep 一段时间模拟耗时，断言下一次的时间戳接近"上一次执行结束的时间 + period"。

两个测试都加了 `@Timeout`（`:49`、`:98`），避免出错时卡住 CI；用 AssertJ 的 `isCloseTo` 容忍定时器的误差。

### 4.3 红绿验证

`AGENTS.md` 的测试规范里有一条（`:275`）：

> **Red-green verification:** For bug fixes, verify that new tests actually fail without the fix before confirming they pass with it.

意思是：修 bug 时，要先确认新测试在**没有修复**时会失败，再确认它在修复后通过。我在本地完整做了一遍（日志在 `assets/logs/redgreen-green.log`、`redgreen-red.log`）。

**绿**：在当前代码（已包含修复）上运行测试：

```bash
./mvnw -B -pl flink-runtime -Dfast -Dtest=SystemProcessingTimeServiceTest -Dsurefire.failIfNoSpecifiedTests=false test
```

```text
[INFO] Tests run: 8, Failures: 0, Errors: 0, Skipped: 0
[INFO] BUILD SUCCESS
```

**红**：**只**撤掉修复中生产代码的部分，保留新测试，再跑一次：

```bash
git show d30cf4d9ede -- flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java | git apply -R
```

```text
[ERROR] Tests run: 8, Failures: 1, Errors: 0, Skipped: 0
[ERROR] org.apache.flink.streaming.runtime.tasks.SystemProcessingTimeServiceTest.testScheduleAtFixedDelay -- Time elapsed: 0.067 s <<< FAILURE!
  Expecting actual:
    1790942092638L
  to be close to:
    1790942092651L
  by less than 10L but difference was 13L.
```

**恢复**：

```bash
git checkout -- flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java
```

三点解读：

- **只有 `testScheduleAtFixedDelay` 失败**，`testScheduleAtFixedRate` 仍然通过：新测试精确地命中了 bug，没有误伤其他逻辑；
- 失败信息说得很清楚：修复前，回调拿到的时间戳比预期**早了 13ms**，大致是回调里 sleep 的时间加上调度误差（之前跑过一次是 14ms，误差和机器的调度有关）；
- 如果一个"修复"的测试，在去掉修复后**依然能通过**，说明它根本没测到问题。这正是 `AGENTS.md` 所说的"没有真正测到改动的测试"。

| 方面 | 这个 PR 的做法 |
|---|---|
| 标题 | `[FLINK-38932][runtime]` + 一句话说清修的是什么、在什么场景下 |
| 范围 | 只修这一个问题，没有顺手做无关的清理 |
| 改动方式 | 最小改动：加一个字段、传一个参数，不重新设计整个类 |
| 测试 | 修 bug 的同时，也加强了原来过于宽松的测试 |
| 可验证性 | 测试能稳定复现 bug，并且有超时保护 |

---

## 五、第一批候选任务：这个系列发现的问题

写这个系列时，我遇到并核实过下面这些问题。我在本地的 `origin/master`（2026-09-24）上又确认了一遍，**它们都还在**。

> **动手之前请务必**：① 在最新的 `master` 上再确认一次；② 在 JIRA 上搜索是否已经有人报过（我做过关键词搜索，但不能保证一定没有）；③ 先开 JIRA 描述问题，和社区确认方向后再写代码。**这些都只是我的观察，不代表社区的结论。**

| # | 问题 | 现状 | 难度 / 性质 | 对应讲 |
|---|---|---|---|---|
| 1 | 文档仍在介绍 2.0 已删除的网络配置项（`buffers-per-channel` 等） | `network_mem_tuning.md`（中英文各 7 处）、`adaptive_batch.md`、`batch_shuffle.md` 仍在提；配置类里已经搜不到这些 key | **推荐作为第一个任务**：只改文档，但要真正理解第四讲的内容才能改好 | 第四讲 |
| 2 | 2.0 release notes 的"已删除配置项"清单里有 `state.backend.type`，但它在 2.x 里仍然是正在使用的主 key | `docs/content/release-notes/flink-2.0.md:1593` | 需要先向社区确认 | 第三讲 |
| 3 | `table.optimizer.agg-phase-strategy` 的配置说明没提到"流模式下需要先开 MiniBatch" | `OptimizerConfigOptions.java` 的说明里没有 MiniBatch 相关字样；调优文档里写了；`EXPLAIN PLAN_ADVICE` 会给出建议（踩坑实验室 #16），但普通 explain 和日志里没有提示 | 文档 / 配置说明改进 | 第五讲上篇 |
| 4 | `Committer.CommitRequest.retryLater()` 的 Javadoc 没说明重试是立即进行、没有退避 | `Committer.java:82-87` | Javadoc 改进 | 踩坑 #08、第八讲下篇 |
| 5 | REST 聚合指标接口：请求的多个指标里只要有一个不存在，整个结果就是空的 | `AbstractAggregatingMetricsHandler.java:242` 直接返回空列表，而"不是数值"时只是跳过（`:231`） | 会改变 REST API 行为，属于"先讨论" | 第四讲 |

另外两个方向，适合长期做：

- **中文文档同步**：`docs/content.zh/` 里有不少页面落后于英文版；
- **Review 别人的 PR**：`.github/workflows/community-review.yml` 会给**非 committer 做过 review** 的 PR 打上 `community-reviewed` 标签。社区有意让非 committer 的 review 被看见，这正是成为 committer 需要积累的贡献之一。

---

## 六、把这个系列学到的，变成贡献

| 讲 | 你已经掌握的 | 可以做的贡献 |
|---|---|---|
| 第一讲 | 作业执行链路、算子链、Mailbox | 回答"为什么算子没有链在一起""作业一直卡在 CREATED" |
| 第二讲 | Checkpoint、对齐与非对齐 | 分析 Checkpoint 超时，review checkpoint 相关的 PR |
| 第三讲 | 状态后端、KeyGroup、扩缩容 | 排查状态膨胀、恢复慢；候选任务 2 |
| 第四讲 | 网络栈与背压 | 候选任务 1、5；分析背压问题 |
| 第五讲 | Table / SQL | SQL 是最活跃的方向；候选任务 3 |
| 第六讲 | 调度与容错 | 分析 failover，参与 AdaptiveScheduler 的讨论 |
| 第七讲 | Watermark 与窗口 | 回答"窗口不触发""数据被判迟到"这类问题 |
| 第八讲 | Source / Sink | 候选任务 4；连接器仓库（Kafka、JDBC 等）里有更多机会 |

一条现实可行的节奏：

1. **第 1 个月**：配好开发环境，完成一个文档类任务（比如候选任务 1），完整走一遍 JIRA → PR → review → 合入；
2. **第 2～3 个月**：每周 review 1～2 个自己主攻领域的 PR；在 `user-zh@` 邮件列表回答问题；
3. **第 4～6 个月**：在主攻领域修一个真正的 bug：最小改动 + 严格的测试 + 红绿验证；
4. **之后**：参与 FLIP 讨论、发版验证投票，承担一个有分量的特性或重构。

---

## 七、关于这个系列本身

这个系列从第 0 讲到第 9 讲的长文、21 期踩坑实验室、所有示例代码，都是在 AI（Claude）的辅助下完成的。在这个过程中，AI 也犯过不少错，比如：

- 把一个 RocksDB 的 OPTIONS 配置文件，当成了增量 Checkpoint 上传的 SST 文件；
- 在实验输出里"补"了一个真实日志中并不存在的字段；
- 把"Timer 回调（事件时间和处理时间都会）设置当前 key，Checkpoint 回调不会"说成了"Checkpoint 回调也会设置"；
- 把一个 2.2 引入的特性写成了 2.3；
- 上一讲还把"exactly-once 的延迟最多约一个 Checkpoint 间隔"写成了"至少一个"。

这些都是在**重跑实验、逐行核对源码**时发现并改正的，每一篇末尾的勘误也记录了这些问题。

这和 Flink 社区的规则是同一个道理：**AI 可以帮你更快地读代码、查资料、写初稿，但能不能讲清楚这段代码为什么这样写，能不能用实验证明它，才是你自己的能力。**

如果你也打算用 AI 辅助学习源码或者写技术文章，我的建议是：

1. 每一个 `文件:行号` 都自己打开看一眼；
2. 每一段"实测输出"都自己跑一遍；
3. 推断和事实分开写，推断要明确标出来。

---

## 写在最后

到这里，**Flink 2.x 源码精读**第一季就结束了。

| 讲 | 主题 |
|---|---|
| 一 | 作业是怎么跑起来的：Transformation → StreamGraph → JobGraph → ExecutionGraph，Mailbox |
| 二 | Checkpoint：从触发到完成，对齐与非对齐 |
| 三 | 状态后端与扩缩容：KeyGroup、HashMap、RocksDB、增量 Checkpoint |
| 四 | 网络栈与背压：Credit 流控、Buffer Debloating |
| 五 | Table / SQL：从 SQL 到 Transformation，Changelog |
| 六 | 调度与容错：Pipelined Region、指数退避、AdaptiveScheduler |
| 七 | Watermark 与窗口 |
| 八 | Source / Sink 与端到端 exactly-once |
| 九 | 从读者到贡献者 |

所有示例代码都在 `flink-notes/demos` 里，运行方式见 `run.sh` 开头的注释。

**接下来**：

- 我会开始做第一个社区贡献，过程会写成一个新的系列《我给 Apache Flink 提 PR》。**它不按周更新，跟着真实进展写**：开 JIRA、讨论、写代码、被 review、修改、合入（或者被拒），每一步都如实记录；
- 第二季会接着讲 **Flink + Paimon 流式湖仓**，见下面的预告。

> **第二季预告：《Paimon 源码精读》**
>
> 第一季讲的是"数据怎么流"，第二季讲"数据流完之后存在哪、怎么被改、怎么被读"。Apache Paimon 是一个湖仓存储，和 Flink 配合得很紧，第一季的不少内容会在第二季里换个场景再出现：
>
> | 第二季 | 主题 | 回扣第一季 |
> |---|---|---|
> | 第 1～2 讲 | 一张图看懂 paimon-core；一条数据的写入之旅 | |
> | 第 3～4 讲 | LSM 合并怎么选文件、怎么执行 | 第三讲下篇：RocksDB 也是 LSM 树 |
> | 第 5 讲 | 并发提交：乐观锁与冲突检测 | |
> | 第 6～7 讲 | 读路径、删除向量 | |
> | 第 8 讲 | Lookup Changelog：`-U` `+U` 是怎么在存储里算出来的 | 第五讲下篇、踩坑 #17 |
> | 第 9 讲 | Flink 写入：两阶段提交，提交的结果是一个新快照 | 第八讲下篇 |
> | 第 10 讲 | Flink 流读：Enumerator 发现新快照、切成 split | 第八讲上篇 |
> | 第 11～12 讲 | Spark MERGE INTO、REST Catalog | |
>
> 形式和第一季一样：每讲一篇源码长文，配套《Paimon 踩坑实验室》短内容；每个结论都有实验输出或源码行号，实验代码公开。第二季的实验用 **Flink 2.2**（Paimon 目前支持到 Flink 2.2，还不支持本季讲的 2.3），源码基于开讲时 Paimon 的最新正式版。


**留一个问题**：你给开源项目提过 PR 吗？第一次提交时，最大的障碍是什么？
:::
