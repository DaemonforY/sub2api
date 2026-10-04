---
title: "Apache Spark 源码学习路线：从入门到 Committer"
description: "L0 预备到 L4 专家五个阶段：每阶段读哪些源码、做哪些练习、怎样参与社区。"
bigdata: "spark"
---

# Apache Spark 源码学习路线：从入门到 Committer

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

::: v-pre
> 基于本地源码 `opensource-codes/spark`（分支 `learn-v4.2.0`），文中路径均已在该版本中核实。
> 时间估算按 **每周 10–15 小时** 计算。

## 总览

| 阶段 | 时长 | 核心目标 | 社区参与 | 阶段出口标志 |
|---|---|---|---|---|
| L0 预备 | 2–4 周 | Scala/JVM/分布式基础 | 订阅邮件列表 | 能读懂 Spark 的 Scala 代码风格 |
| L1 入门 | 1–2 月 | 会用、会调试、跟通一个 Job | 看 JIRA、看 PR | 能在 IDE 断点跟踪 Job 从提交到执行的全过程 |
| L2 中级 | 3–4 月 | 吃透 Core：调度/Shuffle/内存/存储/RPC | 第一个 PR 合入 | 合入 3–5 个 PR（文档/测试/小 bug） |
| L3 高级 | 4–6 月 | 吃透 SQL：Catalyst/Codegen/AQE/DSv2 | 固定领域持续贡献、开始 Review | 合入 20+ 个有实质内容的 PR，参与 Review |
| L4 专家 | 12 月+ | 成为某个领域的 Owner | 设计讨论、SPIP、Release 投票 | 被提名为 Committer |

> 说实话：Committer 看的是 **长期、持续、高质量的贡献和社区信誉**，不是只看技术能力。一般需要 **1.5–3 年** 以上稳定投入。技术深度是门槛，Review、答疑、社区沟通才是真正的分水岭。

---

## L0 预备（2–4 周）

**要补的知识**
- **Scala 2.13**：case class、模式匹配、implicit、trait、隐式转换、集合 API。Catalyst 里到处是模式匹配（`transform { case ... => }`），必须熟悉。
  - 推荐：*Programming in Scala*（Odersky）前 20 章
- **JVM**：内存模型、GC、堆外内存（`Unsafe`）、类加载。Tungsten 和内存管理都依赖这些。
- **分布式基础**：MapReduce 论文、Spark RDD 论文（*Resilient Distributed Datasets*，NSDI'12）、Spark SQL 论文（SIGMOD'15）
- **数据库基础**：关系代数、查询优化（RBO/CBO）、Volcano 迭代模型、向量化执行。推荐 CMU 15-445 课程。

**动手**
- 订阅 `dev@spark.apache.org`（发空邮件到 `dev-subscribe@spark.apache.org`），开始每天扫一眼标题
- 注册 ASF JIRA 账号：https://issues.apache.org/jira/projects/SPARK

---

## L1 入门（1–2 个月）

### 目标
会用 RDD / DataFrame / SQL API，看懂 Spark UI，能在 IDE 里调试源码。

### 学习内容
1. **API 熟练度**：跑通 `examples/` 下的例子（`SparkPi`、`WordCount`、SQL 示例）
2. **Spark UI**：看懂 Jobs / Stages / Tasks / SQL 页签、DAG 可视化、Shuffle 读写量
3. **执行计划**：用 `df.explain("extended")` 看 Parsed → Analyzed → Optimized → Physical 四个计划
4. **整体架构**：Driver / Executor / Cluster Manager，Job → Stage → Task 的划分

### 源码阅读（先看主干，别陷进细节）
| 主题 | 入口文件 |
|---|---|
| 提交入口 | `core/src/main/scala/org/apache/spark/deploy/SparkSubmit.scala` |
| 核心上下文 | `core/src/main/scala/org/apache/spark/SparkContext.scala` |
| RDD 抽象 | `core/src/main/scala/org/apache/spark/rdd/RDD.scala`（重点看五大属性：partitions、compute、dependencies、partitioner、preferredLocations） |

### 实践作业
- [ ] 在 IDEA 中以 `local[2]` 运行 `SparkPi`，从 `rdd.reduce()` 开始打断点，一路跟到 `DAGScheduler.submitJob` → `TaskSchedulerImpl.submitTasks` → `Executor` 执行 Task，**画出调用链时序图**
- [ ] 写一个包含 `groupByKey` 的程序，在 UI 上找到 Stage 边界，并在源码中找到 `ShuffleDependency` 在哪里产生
- [ ] 写一篇博客：《一个 Spark Job 的一生》

### 社区
- 每周浏览 JIRA 上 `starter` 标签的 issue，看看别人的 PR 是怎么写、怎么被 Review 的
- 读 https://spark.apache.org/contributing.html（**必读**）

---

## L2 中级（3–4 个月）：吃透 Spark Core

### 模块与源码

| 模块 | 关键源码 | 要回答的问题 |
|---|---|---|
| **调度** | `scheduler/DAGScheduler.scala`、`scheduler/TaskSchedulerImpl.scala`、`TaskSetManager.scala` | Stage 怎么切分？失败重试和推测执行怎么做？Locality 怎么调度？ |
| **Shuffle** | `shuffle/sort/SortShuffleManager.scala`，Java 部分 `core/src/main/java/org/apache/spark/shuffle/sort/`（`UnsafeShuffleWriter`、`BypassMergeSortShuffleWriter`、`ShuffleExternalSorter`） | 三种 Writer 分别在什么条件下被选用？Spill 怎么触发？ |
| **内存管理** | `memory/UnifiedMemoryManager.scala`、`TaskMemoryManager`（Java） | Execution 和 Storage 内存怎么互相借用？堆外内存怎么管理？ |
| **存储** | `storage/BlockManager.scala`、`BlockManagerMaster` | Cache/Broadcast/Shuffle 数据怎么存、怎么取？ |
| **RPC** | `rpc/netty/NettyRpcEnv.scala`、`common/network-common/` | Driver 和 Executor 之间怎么通信？ |
| **资源/部署** | `deploy/`、`resource-managers/kubernetes`、`resource-managers/yarn` | 动态资源分配（`ExecutorAllocationManager`）怎么工作？ |

所有 Core 路径前缀：`core/src/main/scala/org/apache/spark/`

### 实践作业
- [ ] 画出 `DAGScheduler` 的事件循环（`DAGSchedulerEventProcessLoop`）状态图
- [ ] 人为制造 Executor 失败 / FetchFailed，跟踪 Stage 重算流程
- [ ] 分析一个数据倾斜场景的 Shuffle 源码表现
- [ ] **学会跑单测**：开发时用 sbt 比 Maven 快得多：
  ```bash
  ./build/sbt "core/testOnly org.apache.spark.scheduler.DAGSchedulerSuite"
  ```
  Spark 的测试（`*Suite.scala`）是最好的文档，读源码时对照着看

### 社区：第一批 PR（本阶段关键里程碑）
按难度递进：
1. **文档修正**（`docs/` 目录）：错别字、过时说明
2. **补测试**：给覆盖不足的地方补单测
3. **小 Bug 修复**：从 JIRA 找 `starter` / `easyfix` 标签或自己在读代码时发现的问题
4. **报错信息改进**：Spark 近年在做统一错误类（`common/utils/src/main/resources/error/error-conditions.json`），这类 PR 容易上手

**PR 规范**
- 先在 JIRA 建 issue 或认领已有 issue，拿到 `SPARK-XXXXX` 编号
- PR 标题格式：`[SPARK-XXXXX][CORE] 简短描述`（组件标签：CORE / SQL / SS / CONNECT / PYTHON / K8S / DOCS 等）
- 按 PR 模板填写：What / Why / Does this PR introduce any user-facing change / How was this tested
- 提交前运行 `./dev/lint-scala`、`./dev/scalastyle`
- 在自己的 fork 上开启 GitHub Actions，保证 CI 通过

---

## L3 高级（4–6 个月）：吃透 Spark SQL

Spark SQL 是 Spark 最活跃、代码量最大、Committer 最多的领域，**是冲击 Committer 的主战场**。

### 一条 SQL 的完整链路

```
SQL 文本
  → Parser (ANTLR)          sql/api/src/main/antlr4/.../SqlBaseParser.g4
  → Unresolved LogicalPlan
  → Analyzer                sql/catalyst/.../analysis/Analyzer.scala
  → Resolved LogicalPlan
  → Optimizer               sql/catalyst/.../optimizer/Optimizer.scala
  → Optimized LogicalPlan
  → Planner (Strategies)    sql/core/.../execution/SparkStrategies.scala
  → SparkPlan
  → AQE / Codegen           sql/core/.../execution/adaptive/AdaptiveSparkPlanExec.scala
                            sql/core/.../execution/WholeStageCodegenExec.scala
  → RDD[InternalRow]
总控：sql/core/src/main/scala/org/apache/spark/sql/execution/QueryExecution.scala
```

### 深入专题
| 专题 | 重点 |
|---|---|
| **Catalyst 框架** | `TreeNode`、`transform`/`transformUp`、`Rule`、`RuleExecutor`、`Batch` 和 `FixedPoint` |
| **表达式与 Codegen** | `Expression.eval` 和 `doGenCode` 的对应；`expressions/codegen/CodeGenerator.scala`；用 `df.queryExecution.debug.codegen()` 看生成的代码 |
| **Tungsten** | `UnsafeRow` 内存布局、`UnsafeExternalSorter` |
| **Join** | BroadcastHash / ShuffledHash / SortMerge 三种 Join 的选择逻辑（`JoinSelection`） |
| **AQE** | 运行时合并分区、倾斜 Join 优化、动态切换 Join 策略 |
| **CBO / 统计信息** | `Statistics`、Join Reorder |
| **DataSource V2** | `sql/catalyst/.../connector/` 下的接口，理解 Iceberg/Paimon 等如何对接 Spark（你本地已有 paimon 源码，可以对照看） |
| **Spark Connect** | `sql/connect/`（client/server/common），Spark 4.x 的重点方向 |
| **Structured Streaming** | `MicroBatchExecution`、State Store、Watermark |
| **声明式管道（Declarative Pipelines）** | `sql/pipelines/`，4.x 新方向，代码较新，适合早期切入 |

### 实践作业
- [ ] **自己写一条优化规则**：通过 `SparkSessionExtensions` 注入一条自定义 Optimizer 规则，并写单测
- [ ] **自己实现一个内置函数**：包括 `eval`、`doGenCode`、函数注册、SQL 测试。这是 SQL 贡献最常见的模式之一
- [ ] 用 `SQLQueryTestSuite` 的 golden file 测试（`sql/core/src/test/resources/sql-tests/`）给自己的函数加测试
- [ ] 写一个简单的 DSv2 数据源
- [ ] 系列博客：《Catalyst 源码剖析》

### 社区：从"贡献者"到"熟面孔"
- **选定 1–2 个领域深耕**（例如 SQL 函数、AQE、Connect、Streaming、Pipelines），不要什么都做一点
- 做**有实质内容**的 PR：功能改进、性能优化、复杂 Bug 修复
- **开始 Review 别人的 PR**：在你熟悉的领域留下有价值的 Review 意见。这是 Committer 最看重的能力之一
- 在 JIRA 上帮忙复现、分类 Bug
- 在 user@ 邮件列表 / StackOverflow 回答问题

---

## L4 专家 → Committer（12 个月+）

### 技术层面
- 成为某个模块事实上的 **Owner**：别人遇到这个模块的问题会 @ 你
- 主导一个中大型特性：写设计文档，必要时提 **SPIP**（Spark Project Improvement Proposal），在 dev@ 发起讨论
- 做性能工作：用 TPC-DS 基准（`sql/core/src/test/scala/.../benchmark/`）验证优化效果
- 理解兼容性和发布流程：API 稳定性、迁移指南（`docs/sql-migration-guide.md`）、废弃策略

### 社区层面（决定性因素）
- **持续 Review**：比写代码更重要。Committer 的核心职责就是守护代码质量
- 参与 dev@ 讨论：设计讨论、版本规划
- **Release 验证**：在每个 RC 投票时下载、验证、回复投票（非约束性投票人人可以投）
- 帮助新人：指导他们的第一个 PR
- 线下影响力：Meetup、Data+AI Summit、技术博客

### Committer 是怎么产生的
- 由 **PMC 成员**提名并投票，不能自己申请
- 考量点（见 contributing 页面和 ASF 惯例）：
  1. 持续的高质量代码贡献
  2. 对项目某个领域的深入理解
  3. **Review 和社区协作**的质量
  4. 与社区的沟通方式：尊重、耐心、建设性
- 建议：与活跃在你领域的 Committer 建立联系，让他们熟悉你的工作

---

## 学习方法建议

1. **测试驱动读源码**：每个核心类都有对应的 `*Suite.scala`，从测试入手最高效
2. **Git 考古**：`git log -p --follow <file>` 和 `git blame`，看一个功能是怎么演进的；对照 PR 里的讨论，理解"为什么这么设计"
3. **读 JIRA 和 PR 的讨论**：比读代码更能学到设计取舍
4. **输出倒逼输入**：每个阶段都写博客或画图
5. **跟上 master**：定期 `git fetch origin && git log origin/master --oneline -50`，了解社区当前在做什么

## 参考资源
- 官方贡献指南：https://spark.apache.org/contributing.html
- 开发者工具：https://spark.apache.org/developer-tools.html
- JIRA：https://issues.apache.org/jira/projects/SPARK
- GitHub：https://github.com/apache/spark
- 邮件列表存档：https://lists.apache.org/list.html?dev@spark.apache.org
- 书籍：*Spark: The Definitive Guide*（入门）、*High Performance Spark*（进阶）
:::
