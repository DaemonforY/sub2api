---
title: "第 10 讲：AQE、DataSource V2 与 Spark Connect"
description: "AQE 自适应执行、DataSource V2 接口和 Spark Connect 的源码实现。"
bigdata: "spark"
---

# 第 10 讲：AQE、DataSource V2 与 Spark Connect

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`adaptive/…` = `sql/core/src/main/scala/org/apache/spark/sql/execution/adaptive/`，`connector/…` = `sql/catalyst/src/main/java/org/apache/spark/sql/connector/`
> 配置默认值、类名、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；AQE 三大优化、DataSource V2 聚合下推、Spark Connect 客户端/服务端都经过实验验证（仓库 `experiments/11-aqe-dsv2-connect/`）。

## 0. 本讲要回答的问题

这一讲覆盖 Spark SQL 的三个"现代化"方向：

1. **AQE**：静态优化器靠估计，估错了怎么办？Spark 怎样在运行中根据真实数据**重新优化**？
2. **DataSource V2**：Iceberg、Paimon 这类外部数据源，是通过什么接口接入 Spark 的？
3. **Spark Connect**：客户端和 Spark 集群之间，为什么要隔一层 gRPC？它改变了什么？

---

# 一、AQE：自适应查询执行

## 1. 为什么需要 AQE

第 8 讲讲过，静态优化器和物理计划生成都依赖**统计信息的估计**。而估计常常不准：

- 过滤条件到底过滤掉了多少数据？没有收集统计信息时，Spark 基本不知道
- Join、聚合之后数据量变成多少？更难估计
- 数据是否倾斜？编译期完全不知道

估错的代价很大：本来可以广播的小表走了 SortMergeJoin；200 个 Shuffle 分区里大部分几乎是空的；一个倾斜分区拖慢整个 Stage……

**AQE 的思路**：**以 Shuffle 为界，把查询切成多个阶段，一个阶段执行完，就拿到了真实的数据量，再用真实数据重新优化剩下的部分。**

## 2. 演进

| 时间 | 变化 | 提交 |
|---|---|---|
| 2019-06，**Spark 3.0** | 新的自适应执行框架 | `[SPARK-23128] A new approach to do adaptive execution in Spark SQL` |
| 2020-01，**Spark 3.0** | 基于数据大小的倾斜分区优化 | `[SPARK-29544] optimize skewed partition based on data size` |
| 2020-12，**Spark 3.2** | **AQE 默认开启** | `[SPARK-33679] Enable spark.sql.adaptive.enabled by default` |

## 3. 工作原理

开启 AQE 时（`spark.sql.adaptive.enabled`，默认 `true`），第 8 讲的准备规则 `InsertAdaptiveSparkPlan` 会把整个物理计划包进一个 **`AdaptiveSparkPlanExec`**。它的执行过程：

```
1. 从计划里找出可以先执行的"查询阶段"（QueryStage）：
     · ShuffleQueryStageExec    每个 Shuffle 的 map 端
     · BroadcastQueryStageExec  每个广播
2. 提交这些阶段，等它们完成  →  拿到真实的统计信息（每个 Shuffle 分区多大、一共多少行）
3. 重新优化（reOptimize，AdaptiveSparkPlanExec.scala:805）：
     · 把已完成的阶段替换成 LogicalQueryStage，它带着真实的统计信息
     · 重新跑一遍 AQE 优化器 + 普通的物理计划生成（第 809 行，和第 8 讲用的是同一个 planner）
4. 代价比较（第 367 行）：新计划更好才采用
5. 重复 1～4，直到所有阶段都执行完
```

**AQE 的"代价"非常朴素**（`adaptive/simpleCosting.scala:44`，`SimpleCostEvaluator`）：**代价就是计划里 Shuffle 的个数**。新计划的 Shuffle 更少（或者一样多但计划不同），就采用新计划。可以通过 `spark.sql.adaptive.customCostEvaluatorClass` 换成自定义的代价评估器。

> 对比第 8 讲：静态物理计划阶段"只取第一个候选，不比较代价"；AQE 则会在运行时比较新旧计划。

**两组规则**（`AdaptiveSparkPlanExec.scala` 第 85、137 行）：

| 规则组 | 作用于 | 主要规则 |
|---|---|---|
| **AQE 逻辑优化器**（`AQEOptimizer`） | 重新优化时的逻辑计划 | `AQEPropagateEmptyRelation`（真实结果为空就整片消除）、`DynamicJoinSelection`、`EliminateLimits`…… |
| **查询阶段优化规则**（`queryStageOptimizerRules`） | 物理计划 | `CoalesceShufflePartitions`（合并分区）、`OptimizeSkewedJoin`（倾斜 Join）、`OptimizeShuffleWithLocalRead`（本地读）、`OptimizeSkewInRebalancePartitions` |

## 4. 三大优化与实验

### 4.1 合并 Shuffle 分区

**问题**：`spark.sql.shuffle.partitions` 默认 200，对小数据来说，大部分分区几乎是空的，却每个都要启动一个 Task。

**实验 A**：10 万行数据，按 10 个 key 聚合：

```
AQE=false：各 Stage 的 Task 数 = 8, 200
AQE=true： 各 Stage 的 Task 数 = 8, 1
    +- AQEShuffleRead coalesced
```

AQE 看到 200 个 Shuffle 分区加起来也很小，就把它们**合并**成 1 个，下游只需 1 个 Task。

> ⚠️ 实践细节：`spark.sql.adaptive.coalescePartitions.parallelismFirst` 默认 `true`。这时合并**不以** `advisoryPartitionSizeInBytes`（64MB）为目标，而是优先保证并行度（第 1111 行起的配置说明）。源码注释原话：**在繁忙的集群上，建议设为 `false`**，避免产生大量很小的 Task。

### 4.2 动态切换 Join 策略

**问题**：一张表经过过滤后其实很小，但静态估计不知道，于是选了 SortMergeJoin（两边都要 Shuffle + 排序）。

**实验 B**：小表是 500 万行过滤到只剩 100 行：

```
执行前（初始计划）：  SortMergeJoin [k], [k2], Inner
执行后（最终计划）：  BroadcastHashJoin [k], [k2], Inner, BuildRight
```

**是谁换的？** 我一开始以为是 `DynamicJoinSelection` 这条规则，读了源码才发现**不是**。它的注释写得很清楚，只做三件事：

1. 某一侧**空分区比例很高**：加 `NO_BROADCAST_HASH` 提示，**避免**广播
2. 某一侧**每个分区都很小**：加 `PREFER_SHUFFLE_HASH` 提示，倾向 Shuffle Hash Join
3. 两者同时满足：加 `SHUFFLE_HASH` 提示

真正把 SortMergeJoin 换成广播的，是第 3 节的**重新优化**：小表那一侧的 Shuffle 阶段执行完后，`LogicalQueryStage` 报告了真实的大小（很小），**普通的 `JoinSelection` 策略**（第 8 讲）在重新生成物理计划时，发现它小于广播阈值，就选择了 `BroadcastHashJoin`；新计划少一次 Shuffle，代价更低，被采用。

> 大表那一侧的 Shuffle 往往已经执行完了，这时 `OptimizeShuffleWithLocalRead` 会让下游**直接读本地的 Shuffle 输出**，不再按分区跨网络拉取，尽量挽回已经付出的 Shuffle 代价。

### 4.3 处理倾斜 Join

**问题**：第 2 讲讲过，推测执行解决不了数据倾斜。一个 key 的数据特别多，那个分区的 Task 就特别慢。

**判断标准**（`OptimizeSkewedJoin.getSkewThreshold`，第 65 行）：

```scala
倾斜阈值 = max(skewedPartitionThresholdInBytes, 中位数 × skewedPartitionFactor)
        = max(256MB, 中位数 × 5)                        ← 默认值
```

分区大小**超过这个阈值**就被判定为倾斜（第 146、148 行：`leftSize > leftSkewThreshold`）。

**处理方式**：把倾斜分区**拆成多份**，每份和另一侧对应分区的**完整数据**做 Join（另一侧的这个分区被复制多份）。每一份的目标大小由 `targetSize`（第 75 行）决定：

```scala
目标大小 = max(advisoryPartitionSizeInBytes, 非倾斜分区的平均大小)    // advisoryPartitionSizeInBytes 默认 64MB
```

也就是说，`advisoryPartitionSizeInBytes` **不参与"是否倾斜"的判定**，只决定**拆成多大的块**。

**实验 C**：90% 的数据 key = 0（为了让小数据也能触发，把阈值调到了 1MB）：

```
skewJoin.enabled=false：各 Stage 的 Task 数 = 4, 4, 20, 1
    +- SortMergeJoin [k], [k], Inner
skewJoin.enabled=true： 各 Stage 的 Task 数 = 4, 4, 23, 1
    +- SortMergeJoin(skew=true) [k], [k], Inner
    :  +- AQEShuffleRead skewed
    +- AQEShuffleRead
```

Join 阶段的 Task 从 20 个变成 **23 个**：那个倾斜分区被拆成了 4 份（多出 3 个 Task），计划里标出了 `skew=true` 和 `AQEShuffleRead skewed`。

## 5. AQE 配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.sql.adaptive.enabled` | `true` | 3.2 起默认开启 |
| `spark.sql.adaptive.coalescePartitions.enabled` | `true` | 合并分区 |
| `spark.sql.adaptive.advisoryPartitionSizeInBytes` | `64MB` | 合并 / 拆分时的目标分区大小 |
| `spark.sql.adaptive.coalescePartitions.parallelismFirst` | `true` | 优先并行度；繁忙集群建议 `false` |
| `spark.sql.adaptive.coalescePartitions.minPartitionSize` | `1MB` | 合并后分区的最小大小 |
| `spark.sql.adaptive.coalescePartitions.initialPartitionNum` | 未设置（用 `spark.sql.shuffle.partitions`） | 合并前的初始分区数 |
| `spark.sql.adaptive.skewJoin.enabled` | `true` | 倾斜 Join 优化 |
| `spark.sql.adaptive.skewJoin.skewedPartitionFactor` | `5.0` | 倾斜倍数 |
| `spark.sql.adaptive.skewJoin.skewedPartitionThresholdInBytes` | `256MB` | 倾斜阈值 |
| `spark.sql.adaptive.localShuffleReader.enabled` | `true` | 本地读 Shuffle |
| `spark.sql.adaptive.autoBroadcastJoinThreshold` | 未设置（用 `spark.sql.autoBroadcastJoinThreshold`，10MB） | AQE 运行时的广播阈值 |

---

# 二、DataSource V2：外部数据源怎么接入 Spark

## 6. 为什么要 V2

旧的数据源 API（V1，`BaseRelation`、`PrunedFilteredScan` 等）的问题：

- **能下推的东西很有限**：基本只有列裁剪和简单的过滤条件
- **和 SQL 内部类型耦合**，接口不稳定
- **没有统一的写入语义**：事务、覆盖写、行级更新都难以表达
- **没有 Catalog 的概念**：不能像数据库那样管理"库、表"

| 时间 | 变化 | 依据 |
|---|---|---|
| 2017-09，**Spark 2.3** | DataSource V2 读取路径 | `[SPARK-15689] data source v2 read path` |
| **Spark 3.0** | **Catalog 插件**：`CatalogPlugin`、`TableCatalog` | 接口注释 `@since 3.0.0` |
| **Spark 3.2** | 聚合下推、运行时过滤 | `SupportsPushDownAggregates`、`SupportsRuntimeFiltering` `@since 3.2.0` |
| 🆕 **Spark 4.1** | **Join 下推**、Variant 字段提取下推 | `SupportsPushDownJoin`、`SupportsPushDownVariantExtractions` `@since 4.1.0` |

## 7. 接口全景

接口都在 `connector/` 下，按职责分三块：

### 7.1 Catalog：管理"库、表"

```
CatalogPlugin（插件入口：spark.sql.catalog.<名字>=<实现类>）
 ├── TableCatalog          建表、删表、加载表
 ├── SupportsNamespaces    管理命名空间（库）
 ├── StagingTableCatalog   原子的 CREATE TABLE AS SELECT
 ├── ViewCatalog / FunctionCatalog
Table
 ├── SupportsRead   → newScanBuilder()
 └── SupportsWrite  → newWriteBuilder()
```

### 7.2 读：从 ScanBuilder 到 PartitionReader

```
ScanBuilder           ← 在这里接收下推（实现各种 SupportsPushDownXxx 接口）
  └─ build() → Scan
       └─ toBatch() → Batch
            ├─ planInputPartitions() → InputPartition[]       在 Driver 上切分任务
            └─ createReaderFactory() → PartitionReaderFactory
                 └─ createReader(partition) → PartitionReader  在 Executor 上真正读数据
```

可以实现的下推接口（4.2.0）：

| 接口 | 下推什么 |
|---|---|
| `SupportsPushDownRequiredColumns` | 列裁剪 |
| `SupportsPushDownFilters` | 过滤条件 |
| `SupportsPushDownAggregates` | 聚合（MIN、MAX、COUNT、SUM……） |
| `SupportsPushDownLimit` / `SupportsPushDownOffset` / `SupportsPushDownTopN` | LIMIT、OFFSET、TopN |
| `SupportsPushDownTableSample` | 采样 |
| 🆕 `SupportsPushDownJoin` | **Join**（4.1） |
| 🆕 `SupportsPushDownVariantExtractions` | Variant 类型的字段提取（4.1） |
| `SupportsRuntimeFiltering` | 运行时过滤（动态分区裁剪） |
| `SupportsReportStatistics` / `SupportsReportPartitioning` | 向 Spark 报告统计信息和分区方式 |

### 7.3 写

`WriteBuilder` → `Write` → `BatchWrite` → `DataWriterFactory` → `DataWriter`。每个 Task 写完提交一个 `WriterCommitMessage`，Driver 收齐后统一提交。这套"两阶段提交"是实现事务性写入的基础。行级操作（`DELETE`、`UPDATE`、`MERGE`）通过 `RowLevelOperation`、`SupportsDelta` 等接口支持。

## 8. 真实案例：Paimon 是怎么接入的

你本地的 Paimon 源码（`paimon/paimon-spark/`）就是一个完整的 DataSource V2 实现：

- 按 Spark 版本分了多个模块：`paimon-spark-3.2` ~ `paimon-spark-3.5`、`paimon-spark-4.0`、`paimon-spark-4.1`，公共代码在 `paimon-spark-common`
- **Catalog**：`SparkCatalog` 继承 `SparkBaseCatalog`，后者 **`implements TableCatalog, SupportsNamespaces, ProcedureCatalog...`**（`paimon-spark-common/src/main/java/org/apache/paimon/spark/catalog/SparkBaseCatalog.java:42`）
- **表**：`SparkTable`、`PaimonSparkTableBase` 实现了 `SupportsRead`
- **下推**：Paimon 的 ScanBuilder 实现了 `SupportsPushDownRequiredColumns`、`SupportsPushDownAggregates`、`SupportsPushDownLimit`、`SupportsPushDownTopN`，甚至已经实现了 **4.1 新增的 `SupportsPushDownVariantExtractions`**

使用方式就是一行配置：

```
spark.sql.catalog.paimon=org.apache.paimon.spark.SparkCatalog
```

> 💡 **一条很好的学习路径**：对照着 Spark 的 `connector/` 接口，去读 Paimon（或 Iceberg）的实现，能把"接口为什么这样设计"理解得非常透。数据湖与 Spark 的集成，也是一个很好的贡献方向。

## 9. 实验 D：V1 和 V2 的差别

同一份 Parquet 数据，执行 `SELECT max(id), min(id), count(*)`，开启 `spark.sql.parquet.aggregatePushdown=true`：

```
V1（默认）：
  FileScan parquet [id] ...
    PushedFilters: []
V2（把 parquet 从 spark.sql.sources.useV1SourceList 中移除）：
  BatchScan parquet ...
    PushedAggregation: [MAX(id), MIN(id), COUNT(*)]
```

- **V1** 的 `FileScan`：要把 `id` 列整列读出来，再由 Spark 计算 max、min、count
- **V2** 的 `BatchScan`：聚合被**下推给了 Parquet**。Parquet 文件的元数据里本来就存着每个数据块的最大值、最小值和行数，**直接读元数据就能得到答案，不用读数据本身**

> ⚠️ **一个容易误解的事实**：`spark.sql.sources.useV1SourceList` 的默认值是 `avro,csv,json,kafka,orc,parquet,text`。也就是说，**到 4.2.0 为止，内置的文件数据源默认仍然走 V1**。DataSource V2 目前主要服务于 Iceberg、Paimon、Delta 等**外部数据源**，以及各种数据库连接器。

---

# 三、Spark Connect：客户端与服务端分离

## 10. 为什么需要 Spark Connect

经典模式下，**你的应用程序就是 Driver**：

- 应用和 Spark 运行在**同一个 JVM** 里，依赖冲突（比如 Jackson、Guava 版本）是常见噩梦
- 一个用户的代码把 Driver 搞崩，同一个 Driver 上的其他会话全部受影响
- 想从 Go、Rust、IDE 插件、Notebook 里用 Spark，都得先有一个完整的 Spark Driver 环境
- 升级 Spark 要求所有客户端一起升级

**Spark Connect 的思路**：把客户端和 Driver **分开**，中间用 **gRPC + protobuf** 通信。客户端是一个**很薄的库**，只负责构造查询计划、接收结果。

- 3.4.0 引入（protobuf 定义最早的提交在 2022-10，首次出现于 v3.4.0）
- 🆕 4.0.0 新增 `spark.api.mode`（`classic` / `connect`）：经典模式的应用设成 `connect` 后，会**自动在本地启动一个专用的 Connect 服务**，然后改用 Connect API 运行
- 默认端口 **15002**（`ConnectCommon.CONNECT_GRPC_BINDING_PORT`）

## 11. 架构

```
客户端（很薄：sql/connect/client/jvm、PySpark Connect……）
  │  DataFrame API 调用 → 构造 protobuf 格式的"未解析计划"
  │
  │  gRPC（sql/connect/common/src/main/protobuf/spark/connect/base.proto）
  │    ExecutePlan      执行，返回 流式的 Arrow 批数据
  │    AnalyzePlan      取 schema、explain 等
  │    Config / AddArtifacts（上传 jar、UDF）/ Interrupt / ReattachExecute（断线重连）
  ▼
服务端（运行在 Driver 里：SparkConnectService）
  │  SparkConnectPlanner.transformRelation：protobuf → Catalyst 逻辑计划
  ▼
  第 8～9 讲的整条流程：分析 → 优化 → 物理计划 → 代码生成 → 执行
```

**关键点**：客户端只负责**描述**查询，不做解析、不做分析，更不执行。所有的"理解"和"计算"都在服务端。

## 12. 实验 E：亲手跑一次

在本机启动一个**只监听 127.0.0.1** 的 Connect 服务（`local[2]`），然后在**另一个** `local[1]` 的经典 spark-shell 里，用代码创建 Connect 客户端会话连过去：

```
E1 经典模式的 spark：    org.apache.spark.sql.classic.SparkSession
E1 Connect 客户端会话：  org.apache.spark.sql.connect.SparkSession
E2 通过 Connect 执行查询：sum(0..99) = 4950
E3 在 Connect 会话上访问 sparkContext：失败 → SparkUnsupportedOperationException:
   [UNSUPPORTED_CONNECT_FEATURE.SESSION_SPARK_CONTEXT] Feature is not supported in Spark Connect: Access to the SparkContext.
E4 Connect 客户端发出的未解析计划（protobuf 的文本形式，节选）：
   root {
     common { plan_id: 4 }
     filter {
       input {
         common { plan_id: 3 }
         range { start: 0  end: 100  step: 1 }
       }
       condition {
         expression_string { expression: "id > 50" }
         ...
E5 服务端执行时的物理计划：
   *(1) Filter (id#8L > 50)
   +- *(1) Range (0, 100, step=1, splits=2)
```

逐条解读：

- **E1**：4.x 里有两套 `SparkSession` 实现，`classic` 和 `connect`，接口相同（都来自 `sql/api`，第 8 讲提过）
- **E3**：**Connect 客户端没有 SparkContext**。这是两种模式最本质的区别，也意味着依赖 `sparkContext`、RDD API 的老代码**不能直接迁移到 Connect**
- **E4**：客户端发出去的计划里，过滤条件还只是一个字符串 `"id > 50"`。**客户端根本不解析 SQL 表达式**，这些都交给服务端的 Catalyst
- **E5**：物理计划里 `splits=2`，而客户端所在的 spark-shell 是 `local[1]`，`2` 来自**服务端的 `local[2]`**。这证明查询真的是在服务端执行的

> 补充：用 `spark-shell --remote sc://host:15002` 可以直接启动 Connect 版的 REPL（基于 Ammonite，jar 在构建产物的 `jars/connect-repl/` 目录下）。它需要一个真实的终端，所以实验里改用了在经典 spark-shell 中创建客户端会话的方式。

---

## 13. 三个方向的共同点

| | 解决的问题 | 核心思路 |
|---|---|---|
| **AQE** | 编译期的估计不准 | 推迟决策：先执行一部分，用**真实数据**再优化 |
| **DataSource V2** | 外部系统能力用不上 | 开放接口：让数据源**尽可能多地承担计算**（下推） |
| **Spark Connect** | 客户端和引擎耦合太紧 | 解耦：客户端只**描述**查询，引擎集中执行 |

---

## 自测题

1. AQE 以什么为边界切分查询阶段？每个阶段结束后，它拿到了哪些新信息？
2. AQE 判断新计划是否"更好"的标准是什么？和第 8 讲的静态物理计划阶段有什么不同？
3. 实验 A 中，AQE 为什么把 200 个分区合并成了 1 个？`parallelismFirst` 默认值在什么场景下应该改？
4. 实验 B 中，SortMergeJoin 变成 BroadcastHashJoin，是哪条规则做的？`DynamicJoinSelection` 又是做什么的？
5. 一个分区要多大才会被判定为倾斜？倾斜分区会被拆成多大的块？`advisoryPartitionSizeInBytes` 参与倾斜判定吗？
6. DataSource V2 的读取流程中，哪一步在 Driver 上执行，哪一步在 Executor 上执行？下推接口在哪一层实现？
7. 实验 D 中，V2 的聚合下推为什么可以"不读数据就得到答案"？
8. 4.2.0 中，读 Parquet 默认走 V1 还是 V2？
9. 想在 Spark 里使用 Paimon，需要配置什么？Paimon 实现了 DataSource V2 的哪些接口？
10. Spark Connect 客户端发给服务端的是 SQL 字符串、未解析计划，还是物理计划？在 Connect 会话上能用 `sparkContext` 吗？

---

## 理论部分完结

至此，第 1～10 讲覆盖了学习路线 L1～L3 的全部理论：

| 部分 | 讲次 |
|---|---|
| Spark Core | 1 架构 · 2 调度 · 3 Shuffle · 4 内存 · 5 存储 · 6 RPC 与部署 · 7 容错 |
| Spark SQL | 8 Catalyst · 9 代码生成与 Tungsten · 10 AQE、DataSource V2、Spark Connect |

下一步可以回到实战：L1 的断点调试练习、给社区提第一个 PR（第 3 讲发现的注释错误），以及按学习路线选定一个方向深入。
:::
