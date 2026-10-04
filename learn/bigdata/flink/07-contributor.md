---
title: "Flink 源码导读 07：从读者到贡献者 —— 规范、流程、真实案例与第一批候选任务"
description: "从读源码到给 Flink 提 PR：社区数据、JIRA 与 PR 流程、适合新人的切入点。"
bigdata: "flink"
---

# Flink 源码导读 07：从读者到贡献者 —— 规范、流程、真实案例与第一批候选任务

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`），部分统计基于 `origin/master`（截至 2026-09-24）。
> 本篇引用的规则都来自仓库中的官方文件：`AGENTS.md`、`DEVELOPMENT.md`、`.github/PULL_REQUEST_TEMPLATE.md`、`.github/CONTRIBUTING.md`。**规则以官方文件和官网为准**，本文只是导读。
> 前置：学习路线 中"社区参与"一节

## 0. 本篇要回答的问题

1. Flink 社区现在的贡献者有多少？commit 长什么样？
2. 一个改动从想法到被合入，要经过哪些步骤？
3. 本地要跑哪些检查，才能保证 CI 通过？
4. 一个合格的 bug 修复长什么样？（真实案例 + **红绿验证**复现）
5. **用 AI 辅助写代码，社区的规则是什么？**
6. 读完前 6 篇，有没有可以动手的第一批任务？

## 1. 社区现状：用数据说话

对 `origin/master` 最近一年（2025-09-24 ~ 2026-09-24）的 commit 做统计：

| 指标 | 数值 |
|---|---|
| commit 总数 | **1403** |
| 不同的作者 | **169** |
| 带 JIRA 号的 `[FLINK-XXXX]` commit | 1198（**85%**） |
| `[hotfix]` commit | 187（**13%**） |
| 带 `Generated-by:`（AI 辅助声明）的 commit | **110**（勘误 2026-10-02：原写 116，那是声明的行数；有的 commit 声明了不止一个工具） |

按组件（`[FLINK-XXXX][component]` 中的 component）统计，排名前列的是：`table`（342）、`runtime`（95）、`tests`（81）、`python`（73）、`docs`（56）、`table-planner`（47）、`checkpoint`（34）、`network`（22）……

几点观察：
- **169 位作者**，说明这是一个非常开放的社区，远不是只有少数几个核心开发者
- **Table/SQL 是最活跃的方向**，而 checkpoint、network 这些"硬核"方向的 commit 少、人也少。这对想成为 committer 的人来说反而是机会：在一个人少的领域做深，更容易成为"遇到问题就会去问他的那个人"
- **AI 辅助已经很普遍**：110 个 commit 公开声明使用了 AI 工具（`AGENTS.md` 和 PR 模板的 AI 声明于 2026-04-17 由 FLINK-39477 加入，之后比例逐月上升，2026-09 约 26%，见 `content-plan/23-第9讲-公众号定稿.md`）。社区的态度是**接受，但要求透明、要求作者负责**（第 5 节）

复现这些数字的命令：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && git log origin/master --since=2025-09-24 --format='%s' | grep -oE '^\[(FLINK-[0-9]+|hotfix)\]' | sed -E 's/FLINK-[0-9]+/FLINK-XXXX/' | sort | uniq -c
```

## 2. 一个改动的完整生命周期

```
① 发现问题 / 想法
② 在 JIRA 搜索是否已经有人报过（issues.apache.org/jira/projects/FLINK）
③ 新建或认领 JIRA，描述问题、复现方式、打算怎么改 ──► 较大的改动：先在 dev@ 邮件列表讨论
│                                                     涉及 @Public/@PublicEvolving/@Experimental API：需要 FLIP + 投票
④ 等 committer 把 issue assign 给你（避免重复劳动）
⑤ 在自己的 fork 上建分支、写代码、写测试
⑥ 本地检查：spotless / checkstyle / 相关测试 / 架构测试（第 3 节）
⑦ 提 PR：标题 [FLINK-XXXX][component] ...，按模板填写
⑧ CI（GitHub Actions / Azure Pipelines）+ review，反复修改
⑨ committer 合入；如果是用户可见的变化，在 JIRA 的 Release Notes 字段中填写说明
```

**几条容易被忽视的规则**（来自 `AGENTS.md` 的 "Commits and PRs" 和 "Boundaries" 两节）：
- **每个 PR 只解决一个问题**；清理和重构要和功能改动分成**不同的 commit**
- 永远推送到**自己的 fork**，不要直接推到 `apache/flink`
- 提交之前先 **rebase** 到目标分支的最新代码
- 以下改动**必须先问**（"Ask first"）：修改公共 API 的稳定性注解、大范围的跨模块重构、新增依赖、修改序列化格式、修改 checkpoint/savepoint 的行为、可能影响热路径性能的改动
- 以下事情**绝对不要做**（"Never"）：用 `CHECKSTYLE:OFF` 或 `suppressions.xml` 绕过检查；新功能使用 Java 序列化；在 connector 中使用旧的 `SourceFunction`/`SinkFunction`；修改 `Parser.jj`；在 commit 中把 AI 写成 `Co-Authored-By`

**JIRA 和 PR 的对应关系**：PR 标题是 `[FLINK-XXXX][component] Title`；只有 JavaDoc 或文档中的错别字可以不建 JIRA，用 `[hotfix][docs] ...`。

## 3. 本地开发工作流

### 3.1 环境

`AGENTS.md` "Prerequisites" 一节：**Java 11、17（默认）或 21**，但**所有模块都只能用 Java 11 的语法**（record、sealed class、模式匹配这些 Java 17 语法，只允许在 `flink-tests-java17` 模块中使用）。Maven 用仓库自带的 `./mvnw`。

IDE 配置按 `DEVELOPMENT.md` 操作："Required Plugins"（`:89`）需要安装 Save Actions、Checkstyle-IDEA 和 **google-java-format**（`:100-102`，要求特定版本）；"Code Formatting"（`:109`）配置 Java 和 Scala 的格式化。还有一个实用技巧（`:27`）：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && git config blame.ignoreRevsFile .git-blame-ignore-revs
```

这样 `git blame` 会跳过大规模格式化之类的 commit，直接看到真正修改逻辑的那次提交。

### 3.2 常用命令（来自 `AGENTS.md` 的 "Commands"）

| 目的 | 命令 |
|---|---|
| 快速编译（本系列一直在用） | `./mvnw clean install -DskipTests -Dfast -Pskip-webui-build -T1C` |
| 单个模块带测试 | `./mvnw clean verify -pl flink-core-api` |
| **单个测试类** | `./mvnw -pl flink-core-api -Dtest=MemorySizeTest test` |
| **单个测试方法** | `./mvnw -pl flink-core-api -Dtest=MemorySizeTest#testParseBytes test` |
| **格式化代码**（每次修改后都要跑） | `./mvnw spotless:apply` |
| 检查格式 | `./mvnw spotless:check` |
| Checkstyle | `./mvnw checkstyle:check -T1C`（配置在 `tools/maven/checkstyle.xml`） |

注意：`flink-core`、`flink-runtime` 等模块**不在 checkstyle 的强制检查范围内**，但仍然要遵守同样的规范。

本机实测：`./mvnw spotless:check -pl flink-core-api` 约 **6 秒**；`./mvnw -pl flink-runtime -Dfast -Dtest=SystemProcessingTimeServiceTest test` 约 **17 秒**（模块之前已经编译过）。改一个小地方时，**只跑相关的测试就够了**，完整的 `clean verify` 交给 CI。

### 3.3 测试规范（`AGENTS.md` "Testing Standards"）

- **JUnit 5 + AssertJ**；新代码中**不要**再用 JUnit 4 或 Hamcrest
- 尽量使用真实的测试实现，而不是 Mockito mock（Flink 有大量现成的 `Testing*` 类，比如 `TestingSchedulingTopology`，先搜一下再写）
- 集成测试的类名以 **`ITCase`** 结尾
- 测试的目录结构与源码一一对应
- **★ 红绿验证**：修 bug 时，要**确认新测试在没有修复的情况下会失败**，然后才能确认它在修复后通过。第 4 节会完整演示一遍

### 3.4 架构测试

`flink-architecture-tests/` 用 ArchUnit 检查模块边界（比如 API 模块不能依赖实现模块）。如果你的改动引入了新的违规，README 的要求是：**首先想办法改代码来避免违规**；如果认为规则本身有误，就开 JIRA 讨论；只有在确实无法避免时，才按流程设置 `freeze.refreeze=true` 重新冻结违规记录。

### 3.5 CI

`.github/workflows/ci.yml` 是 GitHub Actions 的主流程，`template.pre-compile-checks.yml` 负责编译前的检查。PR 模板还建议在自己的 fork 上配置 Azure Pipelines（见模板中的链接）。

## 4. ★ 真实案例：FLINK-38932（红绿验证复现）

### 4.1 选这个案例的原因

- 改的是 01 篇讲过的 mailbox 模型中的**处理时间定时器**（`SystemProcessingTimeService`）
- 改动很小：**17 行代码 + 80 行测试**，很适合作为第一个 PR 的参照标准
- 2.3.0 已经包含了这个修复，可以在本地完整复现红绿验证

```
d30cf4d9ede  Zhanghao Chen  2026-01-29
[FLINK-38932][runtime] Fix incorrect scheduled timestamp in ProcessingTimeCallback with scheduleWithFixedDelay (#27429)

 .../runtime/tasks/SystemProcessingTimeService.java | 17 +++--
 .../tasks/SystemProcessingTimeServiceTest.java     | 80 +++++++++++++++++++++-
 2 files changed, 89 insertions(+), 8 deletions(-)
```

### 4.2 问题

`SystemProcessingTimeService` 有两种周期调度：
- `scheduleAtFixedRate`：固定**频率**，第 n 次的计划时间 = 初始时间 + n × period
- `scheduleWithFixedDelay`：固定**间隔**，下一次的计划时间 = **上一次执行结束的时间** + period

两者共用一个 `ScheduledTask`，每次执行完都这样计算下一次的时间戳：

```java
nextTimestamp += period;       // 修复前：两种模式都按固定频率计算
```

所以 `scheduleWithFixedDelay` 传给回调的 `timestamp` 是**错的**：它没有把回调本身的执行时间算进去。回调执行得越慢，偏差就越大。

### 4.3 修复

`flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java`（2.3.0 中的位置）：
- 给 `ScheduledTask` 增加一个 `fixedDelay` 字段，从 `scheduleRepeatedly()` 一路传进来
- 计算下一次时间戳时区分两种模式（`:327-328`）：

```java
nextTimestamp =
        fixedDelay ? System.currentTimeMillis() + period : nextTimestamp + period;
```

### 4.4 测试

`SystemProcessingTimeServiceTest` 做了两件事：
1. **加强已有的测试** `testScheduleAtFixedRate`：原来只检查"触发了多次"，现在还检查每次回调拿到的 `timestamp` 是否接近 `initialTimestamp + n × period`
2. **新增测试** `testScheduleAtFixedDelay`（2.3.0 中在 `:100`）：回调中 `Thread.sleep(executionDelay)` 模拟耗时，然后断言下一次的 `timestamp` 接近**上一次执行结束的时间 + period**

两个测试都用了 `@Timeout`，避免出错时卡住 CI；用 AssertJ 的 `isCloseTo(..., Offset.offset(period))` 容忍定时器的误差。这些都是值得学习的写法。

### 4.5 🧪 在本地复现红绿验证

**绿**：在当前代码（已包含修复）上运行测试：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && ./mvnw -s ../maven-settings-aliyun.xml -B -q -pl flink-runtime -Dfast -Dtest=SystemProcessingTimeServiceTest -Dsurefire.failIfNoSpecifiedTests=false test
```

```
Tests run: 8, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 1.287 s
```

**红**：**只**反向应用修复中生产代码（main 目录）的部分，保留新测试，再运行一次：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && git show d30cf4d9ede -- flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java | git apply -R
```

```
Tests run: 8, Failures: 1, Errors: 0, Skipped: 0 <<< FAILURE!
SystemProcessingTimeServiceTest.testScheduleAtFixedDelay <<< FAILURE!
  Expecting actual:
    1790261013239L
  to be close to:
    1790261013253L
  by less than 10L but difference was 14L.
```

**恢复**：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && git checkout -- flink-runtime/src/main/java/org/apache/flink/streaming/runtime/tasks/SystemProcessingTimeService.java
```

解读：
- 只有 `testScheduleAtFixedDelay` 失败，`testScheduleAtFixedRate` 仍然通过：**新测试精确地命中了 bug，没有误伤其他逻辑**。这就是好测试的标准
- 失败信息说得很清楚：修复前，回调拿到的时间戳比预期**早了 14ms**，大约就是回调中 sleep 的 10ms 加上调度误差。修复前的代码根本没有考虑回调的执行时间
- 如果一个"修复"的测试在去掉修复后**依然能通过**，说明这个测试根本没有测到问题。`AGENTS.md` 专门要求红绿验证，就是为了防止这种情况

### 4.6 可以借鉴的地方

| 方面 | 这个 PR 的做法 |
|---|---|
| 标题 | `[FLINK-38932][runtime]` + 一句话说清楚：修的是什么、在什么场景下 |
| 范围 | 只修这一个问题，没有顺手做无关的清理 |
| 改动方式 | 最小改动：加一个字段、传一个参数，不重新设计整个类 |
| 测试 | 修复了 bug 的同时，也**加强了**原来过于宽松的测试 |
| 可验证性 | 测试可以稳定复现 bug，并且有超时保护 |

---

## 5. ★ AI 辅助贡献：社区的规则

本系列的 7 篇导读、所有示例代码，都是在 AI（Claude）的辅助下完成的。如果你也打算用 AI 工具给 Flink 写代码，**下面这些规则必须遵守**。它们来自 `AGENTS.md` 的 "AI-assisted contributions" 一节，以及 PR 模板的最后一节，并遵循 [ASF Generative Tooling Guidance](https://www.apache.org/legal/generative-tooling.html)：

1. **必须声明**：在 PR 模板中勾选 "Was generative AI tooling used to co-author this PR?"，并取消 `Generated-by: [Tool Name and Version]` 这一行的注释
2. **commit message 中加上** `Generated-by: <工具名和版本>` 这个 trailer
3. **不要把 AI 写成共同作者**：禁止使用 `Co-Authored-By: <AI>`。原文是 "agents are assistants, not authors"
4. **你要为质量负全责**：你必须能够讲清楚设计、代码和测试，能够调试它们，并且能对 review 意见做出**实质性的**回应
5. **低质量的 AI PR 会被直接关闭**：AGENTS.md 列举了几种典型表现：大段没人审阅过的文字、只有框架没有实际行为的代码、没有真正测到改动的测试、注水的 commit message

> ⚠️ 请注意：很多 AI 编程工具默认会在 commit 中加上 `Co-Authored-By` 这一行。**给 Flink 提交代码时，必须把它换成 `Generated-by`**。项目自己的规则优先。

**这对学习的启示**：AI 可以帮你更快地读代码、查资料、写初稿，但**能不能讲清楚这段代码为什么这样写，才是你自己的能力**。本系列每一篇都附带了断点清单和课后练习，就是希望你亲自跟着调试一遍，而不是只读文字。前几篇的"踩坑记录"也说明了：AI 同样会犯错（zsh 分词、错误的配置结论、对 print sink 的误判），**实测验证**是不能省的一步。

---

## 6. ★ 第一批候选任务：本系列发现的问题

下面这些问题是写作前 6 篇时**实际遇到并核实过的**。在动手之前，请务必：
1. 在 `origin/master` 上**再确认一次**，问题是否仍然存在
2. **在 JIRA 上搜索**，是否已经有人报过（我做过关键词搜索，但不能保证一定没有）
3. **先开 JIRA 描述问题**，与社区确认方向后再写代码

### 6.1 文档仍在描述 2.x 已删除的网络配置项（推荐作为第一个任务）

- **事实**：2.0 删除了 `taskmanager.network.memory.buffers-per-channel`、`floating-buffers-per-gate`、`max-buffers-per-channel`、`max-overdraft-buffers-per-gate`（2.0 发布说明 `docs/content/release-notes/flink-2.0.md:1546` 的 "List of removed configuration options" 一节；在 `release-2.0.0` 的 `flink-core` 配置类中也已经搜不到这些 key）。源头是 **FLINK-35461**（Deprecate Some Runtime Configuration for Flink 2.0）。2.3.0 中它们变成了硬编码的常量（04 篇 3.2 节）
- **问题**：以下用户文档至今（包括 `origin/master`）仍在讲这些配置项：
  - `docs/content/docs/deployment/memory/network_mem_tuning.md`（例如 `:102`、`:124`、`:171`）
  - `docs/content/docs/deployment/adaptive_batch.md`（`:86`）
  - `docs/content/docs/ops/batch/batch_shuffle.md`
  - 以及 `docs/content.zh/` 下对应的三个中文文件
- **为什么适合作为第一个任务**：不涉及代码逻辑；需要真正理解 04 篇的内容（buffer 模型、Debloating）才能改好；中英文两个版本都要改，还能锻炼中文文档的维护能力
- **注意**：这不是错别字，需要**先建 JIRA**（component 选 `docs` 或 `Runtime / Network`）。改写时要说明这些配置项已被删除，以及 2.x 推荐的调优方式（Buffer Debloating）

### 6.2 2.0 发布说明中"已删除配置项"清单可能有误（需要先确认）

- **事实**：`flink-2.0.md` "List of removed configuration options" 中列出了 `state.backend.type`，但它在 `release-2.0.0` 和 2.3.0 中**仍然存在，而且是正在使用的主 key**（`flink-core/.../StateBackendOptions.java:46-50`，它的废弃别名是 `state.backend`）。03 篇的示例也正在使用它
- **可能的原因**：清单是自动生成的，或者本来想写的是某个废弃别名。这需要**在 JIRA 或 dev@ 邮件列表上向社区确认**，而不是直接改
- 可以顺手检查一下这个清单中还有没有其他仍然存在的配置项

### 6.3 REST 聚合指标接口：一个指标不存在，整个结果就是空的（需要先讨论）

- **事实**：`AbstractAggregatingMetricsHandler.getAggregatedMetricValues()`（`flink-runtime/.../rest/handler/job/metrics/AbstractAggregatingMetricsHandler.java`）中：请求的某个指标**不是数值**时 `continue`，跳过这一个，其余照常返回；而**不存在**时直接 `return ... emptyList()`（`:242`），整个请求的结果都变成空
- **影响**：04 篇写监控代码时就踩了这个坑：Debloat 相关的指标只有在开启 Debloating 时才会注册，一起请求时，其他指标也全都拿不到了
- **现状**：`AggregatingMetricsHandlerTestBase` 中没有测试覆盖"多个指标中有一个不存在"的情况，所以这个行为并没有被测试固定下来
- **需要讨论的**：这是有意的设计（比如为了让调用方注意到指标名写错了），还是疏漏？修改它会改变 REST API 的行为，属于 "Ask first" 的范畴。**先开 JIRA 讨论**，达成一致后再动手，并补上测试

### 6.4 MiniCluster：动态加入的 TaskManager 无法通信（优先级较低）

- **事实**：`MiniCluster.useLocalCommunication()`（`flink-runtime/.../minicluster/MiniCluster.java:795-796`）按**配置的 TM 数量是否为 1** 来决定是否只用本地通信。以 1 个 TM 启动、再调用 `startTaskManager()` 时，TM 之间无法交换数据，报错 `Local execution does not support shuffle connection`（06 篇 8.2 节）
- **定位**：MiniCluster 主要用于测试，这个方法也是 `protected` 的，可以覆盖。所以这更像是一个**可用性改进**（比如在 `startTaskManager()` 的 JavaDoc 中说明这一点，或者在这种情况下给出更明确的报错），而不是严重的 bug。同样需要先讨论

### 6.5 JIRA 上现成的 `starter` 任务

截至 2026-09-24，JIRA 上有 **63 个未解决、带 `starter` 标签**的 issue（JQL：`project = FLINK AND labels = starter AND resolution = Unresolved`）。其中有两个正好对应本系列的内容：

| Issue | 标题（节选） | 对应篇目 |
|---|---|---|
| FLINK-39998 | RocksDBStateDownloader loses parallel download failures, surfacing a misleading ClosedCha… | 03 篇 6.2 节（RocksDB 恢复时下载 SST） |
| FLINK-36317 | Populate the ArchivedExecutionGraph with CheckpointStatsSnapshot data if in WaitingForReso… | 06 篇 5.2 节（AdaptiveScheduler 的 WaitingForResources 状态） |

动手之前，先在 JIRA 上确认它们**还没有被 assign 给别人**，也没有正在进行中的 PR；然后在 issue 下留言，说明你打算怎么做，请求 assign。

### 6.6 持续性的贡献方向

- **中文文档翻译和同步**：`docs/content.zh/` 中有不少页面落后于英文版，6.1 节就是一个例子
- **Review 别人的 PR**：`.github/workflows/community-review.yml` 每天运行 3 次，会给**非 committer 做过 review 的 PR** 打上 `community-reviewed` / `community-reviewed-LGTM` 标签。社区是**有意让非 committer 的 review 被看见**的，这正是成为 committer 需要积累的贡献之一
- **回答邮件列表上的用户问题**（`user@`、`user-zh@`）：前 6 篇的内容足以回答很多关于 Checkpoint 超时、背压、状态膨胀、failover 的问题

---

## 7. 从贡献者到 Committer

回到学习路线的最后一个阶段，把前 6 篇的内容和社区贡献对应起来：

| 你已经掌握的 | 可以做的贡献 |
|---|---|
| 01 作业执行链路 | 回答"为什么算子没有链在一起""作业一直卡在 CREATED"这类问题 |
| 02 Checkpoint | 分析 Checkpoint 超时，review checkpoint 组件的 PR |
| 03 状态后端 | 排查状态膨胀、扩缩容恢复慢；参与 ForSt 和异步状态相关的工作 |
| 04 网络与背压 | 修复 6.1 节的文档；分析背压问题 |
| 05 Table/SQL | SQL 是最活跃的方向（第 1 节：342 个 commit），新增内置函数有现成的模式可循（`AGENTS.md` "Adding a new SQL built-in function"） |
| 06 调度与容错 | 分析 failover 问题，参与 AdaptiveScheduler 相关的讨论 |

一条现实可行的节奏：
1. **第 1 个月**：配好开发环境，完成 6.1 节的文档修复，走完一遍完整的 JIRA → PR → review → 合入流程
2. **第 2 ~ 3 个月**：每周 review 1 ~ 2 个自己主攻领域的 PR；在 `user-zh@` 回答问题；挑选带 `starter` 标签的 JIRA
3. **第 4 ~ 6 个月**：在主攻领域修复真正的 bug（像第 4 节的案例那样：最小改动 + 严格的测试 + 红绿验证）
4. **之后**：参与 FLIP 的讨论，参与发版验证投票，承担一个有分量的特性或重构

## 8. 课后练习

1. **读懂一个 PR**：从第 1 节 `git log` 的输出中，在自己的主攻领域挑一个 `[FLINK-XXXX]` 的 bug 修复，按第 4 节的方法完整复盘：问题是什么、修复改了什么、测试是怎么写的，并在本地做一次红绿验证
2. **把 JIRA 写出来**：为 6.1 节的问题写一份 JIRA 描述草稿（英文）：问题现象、受影响的文件、2.x 中的实际情况、建议的改法。**先不要提交**，写完后对照 PR 模板检查一遍
3. **本地检查**：对 `flink-runtime` 跑一次 `./mvnw spotless:check -pl flink-runtime`，再看看 `tools/maven/checkstyle.xml` 中都检查了哪些规则
4. **写一个测试**：为 6.3 节的行为补一个测试（先不改实现，只把**现在的**行为固定下来）。思考：如果社区决定修改这个行为，你的测试应该怎样改？
5. **思考题**：为什么 `AGENTS.md` 要求 "Separate cleanup/refactoring from functional changes into distinct commits"？从 reviewer 和日后 `git bisect`、`git revert` 的角度分别想一想

## 9. 系列回顾

| 篇 | 主题 | 配套示例 |
|---|---|---|
| [01](/bigdata/flink/01-execution) | 从 `execute()` 到 `processElement()` | WordCount |
| [02](/bigdata/flink/02-checkpoint) | Checkpoint 全流程 | `CheckpointDemo` |
| [03](/bigdata/flink/03-state-backend) | 状态后端与扩缩容 | `KeyGroupDemo`、`StateBackendDemo` |
| [04](/bigdata/flink/04-network) | 网络栈与背压 | `BackpressureDemo` |
| [05](/bigdata/flink/05-table-sql) | Table / SQL | `SqlDemo` |
| [06](/bigdata/flink/06-scheduling) | 调度与容错 | `FailoverDemo`、`AdaptiveSchedulerDemo` |
| 07 | 从读者到贡献者 | FLINK-38932 红绿验证 |
| [08](/bigdata/flink/08-watermark-window) | 时间、Watermark 与窗口 | `WindowDemo` |
| [09](/bigdata/flink/09-source-sink) | Source / Sink 新架构 | `SourceSinkDemo` |

所有示例都在 `demos/` 中，运行方式见 `demos/run.sh` 开头的注释。
:::
