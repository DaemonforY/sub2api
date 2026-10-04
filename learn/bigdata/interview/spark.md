---
title: Spark 面试题
description: 大数据开发岗常问的 Spark 原理与源码：Job/Stage/Task、调度、Shuffle、内存、存储、容错、Catalyst、代码生成、AQE 和排障，附参考答案和 AI 模拟面试。
---

# Spark 面试题

面试大数据开发，Spark 题一般分三层往下问：**使用层**（算子、缓存、调参会不会用）、**原理层**（Stage 怎么切、Shuffle 怎么走、内存怎么分）、**源码层**（是哪个类、哪段逻辑做的，有没有反直觉的细节）。只背到第一层能及格，能讲清第二层算合格，能随手说出第三层的例子，面试官才会相信你真读过、真调过。

下面的参考答案都以 Spark 4.2.0 为准，很多默认值和老资料不同，答题时可以主动点出版本。

## 题库

### 说说 Spark 中 Job、Stage、Task 的关系，Stage 是怎么按宽窄依赖切分的？

- **关系**：每次 Action 调用 `sc.runJob` 提交一个 Job；DAGScheduler 以 `ShuffleDependency` 为界切成 Stage，最后一个是 `ResultStage`，其余是 `ShuffleMapStage`；每个**需要计算的分区**对应一个 Task。
- **切分**：从最后一个 RDD 反向遍历依赖，窄依赖并入当前 Stage，宽依赖处切开；执行时先递归提交父 Stage，从前往后跑。
- **窄依赖**看父分区数据要不要拆开分发，`coalesce` 多对一也是窄依赖；分区器相同时 `reduceByKey`、`join` 不 Shuffle。
- **细节**：`take` 可能提交多个 Job，`sortByKey` 抽样也会触发 Job；Shuffle 输出已存在的 Stage 会被 skipped。

延伸阅读：[Job、Stage、Task 是什么关系](/bigdata/spark/02-job-stage-task)

### DAGScheduler 和 TaskScheduler 各负责什么？说说数据本地性、延迟调度和推测执行。

- **分工**：DAGScheduler 用单线程事件循环决定「跑什么」：切 Stage、找缺失分区、算位置偏好、广播 Task 二进制，打包成 TaskSet；TaskSchedulerImpl 决定「在哪跑、谁先跑」，每个 TaskSet 由一个 TaskSetManager 管理，按 FIFO（默认）或 FAIR 排序。
- **本地性**：`PROCESS_LOCAL` → `NODE_LOCAL` → `NO_PREF` → `RACK_LOCAL` → `ANY`。
- **延迟调度**：每级等 `spark.locality.wait`（默认 3 秒），等不到再降级。
- **推测执行**：默认关闭；4.2.0 要完成 90% 且慢于中位数 3 倍才推测，解决不了数据倾斜。

延伸阅读：[第 2 讲：调度体系](/bigdata/spark/lecture-02-scheduling)

### Spark 的 Shuffle 是怎么写和读的？SortShuffleManager 的三种 Writer 分别在什么条件下被选用？

- **写**：Hash Shuffle 已在 2.0 移除；Sort Shuffle 每个 Map Task 只写一个 `.data` 和一个 `.index`。
- **选 Writer**（`registerShuffle`）：无 map 端预聚合且分区数 ≤ 200 → `BypassMergeSortShuffleWriter`；序列化器支持重定位、无预聚合、分区数 ≤ 2^24 → `UnsafeShuffleWriter`，只排 8 字节指针；其余走 `SortShuffleWriter`，可预聚合、溢写后归并。
- **读**：向 `MapOutputTrackerMaster` 查位置，`ShuffleBlockFetcherIterator` 本地直读、远程拉取，在途不超过 48m，再按需聚合或排序。
- SQL 默认 200 个分区，正好走 Bypass。

延伸阅读：[第 3 讲：Shuffle](/bigdata/spark/lecture-03-shuffle)

### Spark 的统一内存管理是怎么划分 Executor 内存的？Execution 和 Storage 之间怎么互相借用？

- **划分**：堆先扣 300MB 预留，剩余部分乘 `spark.memory.fraction`（0.6）得统一内存，其中 `spark.memory.storageFraction`（0.5）是 Storage 保底区域，其余 40% 是用户内存。
- **借用不对称**：Storage 只能借 Execution 的空闲内存；Execution 可以驱逐缓存块收回借出的部分，但只收回到 Storage 区域边界。原因是执行内存被拿走 Task 会失败，缓存丢了还能重算。
- **多 Task**：每个 Task 至少 1/2N、最多 1/N。
- 容器还要加 overhead，「exceeding memory limits」应调 `spark.executor.memoryOverhead`。

延伸阅读：[第 4 讲：内存管理](/bigdata/spark/lecture-04-memory)

### 调用 rdd.cache() 之后数据存在哪、怎么被读出来？广播变量又是怎么分发到各个 Executor 的？

- **存储**：缓存分区、Shuffle 文件、广播都是块，由各节点的 `BlockManager` 管理，Driver 上的 `BlockManagerMaster` 记录块在哪。
- **cache**：惰性，只打存储级别标记；第一次 Action 时 `getOrCompute` 先查本地、再远程拉取，都没有就计算并写入。`RDD.cache()` 是 `MEMORY_ONLY`，放不下就重算；`Dataset.cache()` 默认 `MEMORY_AND_DISK`。内存按 LRU 淘汰。
- **广播**：`TorrentBroadcast` 把值序列化压缩后切成 4m 的块；Executor 随机顺序拉取，拉到一块就登记，自己也成为数据源，分摊 Driver 压力。

延伸阅读：[第 5 讲：存储体系](/bigdata/spark/lecture-05-storage)

### Task 失败、Executor 丢失和 FetchFailed 分别怎么处理？为什么非确定性 Stage 被重算可能让整个 Job 失败？

- **Task 失败**：TaskSetManager 重试，同一个 Task 失败达 `spark.task.maxFailures`（4）次 Job 失败；`local[N]` 不重试。
- **FetchFailed**：上游输出丢了，DAGScheduler 注销丢失输出，延迟 200ms 合并后重提上游 Stage，只算丢失分区；不计入 Task 失败次数，上限 `spark.stage.maxConsecutiveAttempts`（4）。
- **Executor 丢失**：上面有运行中的 Task 才立即通知 DAGScheduler，空闲的要等下游 FetchFailed 才发现。
- **非确定性**：重算后数据可能变，下游须全部重跑；ResultStage 已有分区完成就只能让 Job 失败，应在 `repartition` 前 checkpoint。

延伸阅读：[第 7 讲：容错机制](/bigdata/spark/lecture-07-fault-tolerance)

### 一条 SQL 在 Catalyst 中会经历哪几个阶段？举例说说真正生效的优化规则。

- **解析**：ANTLR 语法加 `AstBuilder` 生成未解析逻辑计划；DataFrame API 跳过这步。
- **分析**：Analyzer 通过 Catalog 绑定表、列、函数，列名写错在 `CheckAnalysis` 报错。
- **优化**：`RuleExecutor` 按批次执行规则，`Once` 或 `FixedPoint`（直到不动点，默认最多 100 轮）。常见生效规则：`ConstantFolding`、`PushDownPredicates`、`InferFiltersFromConstraints`、`ColumnPruning`。
- **物理计划**：`SparkPlanner` 只取第一个候选，Join 靠 `JoinSelection` 启发式（小于 10MB 广播）；`EnsureRequirements` 插入 `Exchange`。

延伸阅读：[第 8 讲：Spark SQL 与 Catalyst](/bigdata/spark/lecture-08-catalyst)

### 什么是全阶段代码生成？它比火山模型快在哪里，什么情况下会放弃？

- **火山模型的问题**：每一行在每个算子上都有一次 `next()` 虚调用，表达式树递归 `eval()`，CPU 难以优化。
- **全阶段代码生成**：`CollapseCodegenStages` 把连续算子合并成一个 `WholeStageCodegenExec`（计划里的 `*(n)`），`produce` 自顶向下、`consume` 自底向上，把逻辑拼进一个循环，中间值是局部变量，用 Janino 编译；`Exchange` 是天然边界。实验中关掉它慢约 17 倍。
- **放弃**：字段数超 `spark.sql.codegen.maxFields`（100）、方法字节码超 `spark.sql.codegen.hugeMethodLimit`（65535）、编译失败时回退解释执行。

延伸阅读：[第 9 讲：代码生成与 Tungsten](/bigdata/spark/lecture-09-codegen)

### AQE 是怎么工作的？说说它的三大优化。

- **原理**：以 Shuffle 为界切成查询阶段，一个阶段执行完拿到真实统计，`reOptimize` 重新跑优化器和物理计划，以 Shuffle 个数为代价，新计划更好才采用。3.2 起默认开启。
- **合并分区**：`CoalesceShufflePartitions` 把很小的分区合并；`parallelismFirst` 默认 `true`，繁忙集群建议改为 `false`。
- **切换 Join**：重新规划时 `JoinSelection` 发现一侧低于广播阈值，把 SortMergeJoin 换成 BroadcastHashJoin；`DynamicJoinSelection` 只加提示。
- **倾斜 Join**：超过 max(256MB, 中位数 × 5) 判为倾斜，拆成多块，另一侧对应分区复制多份。

延伸阅读：[第 10 讲：AQE、DSv2 与 Spark Connect](/bigdata/spark/lecture-10-aqe-dsv2)

### 线上一个 Spark 作业总卡在某个 Stage 的少数几个 Task 上，最后 Executor OOM 或被 YARN 杀掉，你会怎么排查和处理？

- **先定位**：Stage 页看 Task 耗时和数据量分布、Spill、Peak Execution Memory，Executors 页看 GC，判断是数据倾斜还是整体内存不足。
- **看报错**：`Java heap space` 多是单个 Task 数据太多；`exceeding memory limits` 是 overhead 不够，调 `spark.executor.memoryOverhead`；Driver OOM 查大 `collect()`。
- **处理**：SQL 依靠 AQE 倾斜 Join 拆分大分区；增加分区数；减少 `spark.executor.cores`，让每个 Task 分到更多执行内存；小表广播，省掉 Shuffle。
- 推测执行解决不了倾斜，别指望它。

延伸阅读：[第 4 讲：内存管理](/bigdata/spark/lecture-04-memory)

## 答题思路

Spark 题最容易答成两种样子：一句话背定义，或者把源码细节一股脑倒出来。比较稳的结构是三步：

1. **先结论**：一两句话直接回答。「Stage 在宽依赖处切开，从后往前切、从前往后跑。」
2. **再原理**：解释为什么是这样。「Shuffle 要求上游全部完成，才能交给下游，所以必须分 Stage；从最后一个 RDD 往回找，才知道哪些数据是真正需要的。」
3. **再源码细节与实践**：给一个能证明你读过源码或调过线上作业的点。「比如 `sortByKey` 是转换算子，但 `RangePartitioner` 抽样会触发一个 Job；线上遇到被 YARN 杀，我们先调的是 overhead，不是堆。」

第三步最拉开差距。注意版本：推测执行的默认阈值、AQE 默认开启、ANSI 模式默认开启，都是近几个版本才变的，能说出「在 4.2.0 里……」会加分。不确定的细节就说「我记得是……，具体要看源码」，不要硬编参数名。

## 模拟面试

准备好了就开始一场：5 道题从上面的题库随机抽，逐题作答，每题都会得到打分、点评和要点。用自己的话回答，不会的题可以直接写「不会」，看完要点再来一场。

<MockInterview topic="spark" />

## 小结

- Spark 面试按「使用层 → 原理层 → 源码层」逐层追问，每道题都至少答到原理层。
- Core 部分的主线是：Job/Stage/Task → 调度 → Shuffle → 内存与存储 → 容错；SQL 部分的主线是：Catalyst → 代码生成 → AQE。
- 回答按「结论 → 原理 → 源码细节与实践」组织，排障题先定位、再分类、再给手段。

想系统补课，按顺序读 [Apache Spark 源码学习](/bigdata/spark/) 的各篇笔记。
