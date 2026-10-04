---
title: Flink 面试题
description: 实时计算岗常问的 Flink 原理：图转换与作业执行、Checkpoint 与 Barrier、端到端 exactly-once、状态后端、KeyGroup 扩缩容、背压、Watermark 与窗口、Table/SQL 和一道综合排障题，附参考答案和 AI 模拟面试。
---

# Flink 面试题

实时计算岗的面试，很少停留在「Flink 是流批一体的计算引擎」这种定义上。面试官想确认的是：你知道一条数据、一个 Barrier、一个 Watermark 在作业里是怎么走的；作业出问题时，你能从 Checkpoint 耗时、背压指标、状态大小里读出原因，并且说清楚每个配置背后的代价。

下面 10 道题都以 Flink 2.3.0 源码为准。不少网上资料还停留在 1.x，比如「对齐时把数据缓存到磁盘」「默认 fixed-delay 重启」，答题时能指出 2.x 的变化，会是明显的加分项。

## 题库

### 一个 DataStream 作业从提交到 Task 跑起来，要经过哪几次图的转换？各在哪里完成？

- **四张图**：Transformation 列表（API 调用只记账）→ StreamGraph（客户端 `StreamGraphGenerator`，一个算子一个节点）→ JobGraph（`StreamingJobGraphGenerator`，算子链在这一步决定）→ ExecutionGraph（按并行度展开成 `ExecutionVertex`，调度和容错的对象）。
- **2.x 变化**：本地和 Session 模式直接提交 StreamGraph，JobGraph 改在 JM 侧生成，方便批作业按运行时信息调整图。
- **算子链**：并行度相同、Forward 分区、同一 slot sharing group 等条件都满足才链接。
- **执行**：`Execution.deploy()` 把 TDD 发给 TaskExecutor，每个 Task 一个线程，Mailbox 串行处理数据、Checkpoint 和定时器，算子不用加锁。

延伸阅读：[Flink 源码导读 01：从 `env.execute()` 到 `processElement()` —— 作业执行全链路](/bigdata/flink/01-execution)

### Checkpoint 是怎么触发的？Barrier 对齐和非对齐 Checkpoint 有什么区别？

- **触发**：JM 的 `CheckpointCoordinator` 周期触发，先给 OperatorCoordinator 做快照，再只向 Source 发 RPC。Task 把它投递进 mailbox：先 `prepareSnapshotPreBarrier`，再广播 Barrier，再同步快照，异步上传后 ack 带回 StateHandle。
- **对齐**：先到 Barrier 的通道被阻塞（上游停止发送），所有通道到齐才快照。背压时 Barrier 排在 in-flight 数据后面，耗时变长。
- **非对齐**：Barrier 超车，被越过的数据作为 channel state 存进快照。快，但 Checkpoint 变大、恢复变慢，只支持 EXACTLY_ONCE，Savepoint 始终对齐。
- **生产**：配 `execution.checkpointing.aligned-checkpoint-timeout`，先对齐，超时再切非对齐。

延伸阅读：[Flink 源码导读 02：Checkpoint 全流程 —— 从触发、Barrier 对齐到完成通知](/bigdata/flink/02-checkpoint)

### Flink 怎么做到端到端 exactly-once？Sink 的两阶段提交分别发生在什么时候？

- **前提**：Source 的读取进度（split 状态）和算子状态在同一个 Checkpoint 里，Sink 用两阶段提交。
- **第一阶段**：`SinkWriterOperator.prepareSnapshotPreBarrier` 中 flush 并 `prepareCommit`，Committable 在 Barrier 之前发给 Committer，进入它的 Checkpoint 状态。
- **第二阶段**：Checkpoint 全局完成后，`CommitterOperator.notifyCheckpointComplete` 才真正 commit；在此之前 `read_committed` 的消费者看不到数据。
- **幂等**：完成通知不保证送达，恢复后会把状态里的 Committable 再提交一次，所以 commit 必须幂等，事务 ID 在重启前后唯一。
- **代价**：数据可见延迟约一个 Checkpoint 间隔；不开 Checkpoint 的无界作业永远不提交。

延伸阅读：[Flink 源码导读 09：Source / Sink 新架构 —— FLIP-27 与 Sink V2](/bigdata/flink/09-source-sink)

### HashMap 和 RocksDB 状态后端有什么区别？RocksDB 的增量 Checkpoint 是怎么做的？

- **两个概念**：State Backend 管运行时状态放在哪，Checkpoint Storage 管快照写到哪。
- **HashMap**：状态在堆上，访问最快；快照靠 Copy-On-Write，同步阶段只浅拷贝桶数组，但只能全量快照，规模受堆限制。
- **RocksDB**：本地盘加托管内存，每次读写都要序列化；`value()` 拿到的是新对象，原地修改后必须 `update()`。
- **增量**：同步阶段做 RocksDB 原生 checkpoint（flush memtable、硬链接 SST），异步阶段只上传新 SST，旧 SST 按文件名复用，放在 `shared/`；只以已完成的 Checkpoint 为基准，共享文件何时删除由 JM 的 `SharedStateRegistry` 决定。

延伸阅读：[Flink 源码导读 03：State Backend —— 状态怎么存、怎么快照、怎么扩缩容](/bigdata/flink/03-state-backend)

### 什么是 KeyGroup？扩缩容时 keyed state 怎么重新分配？为什么最大并行度不能随便改？

- **KeyGroup**：key 经 murmurHash 对 maxParallelism 取模得到，每个 subtask 负责一段连续区间；`keyBy` 分区和状态访问用同一套算法。
- **扩缩容**：只重新划分区间，状态按 KeyGroup 整块迁移。JM 的 `StateAssignmentOperation` 对新旧区间求交集；RocksDB 恢复时用 `deleteRange` 裁掉区间外的 KeyGroup，多个来源再合并。
- **不能改**：maxParallelism 一变，同一个 key 落到不同 KeyGroup，旧状态对不上，恢复直接报错。默认值由初始并行度推算，下限 128，上线前就应显式设置 `pipeline.max-parallelism`。
- Operator State 没有 key，按均分、union、广播三种方式重分配。

延伸阅读：[Flink 源码导读 03：State Backend —— 状态怎么存、怎么快照、怎么扩缩容](/bigdata/flink/03-state-backend)

### Credit-based 流控是怎么工作的？背压是怎么一层层传回 Source 的，怎么定位瓶颈？

- **为什么**：同一对 TM 之间的多个逻辑通道共用一条 TCP 连接，靠 TCP 流控会互相堵塞。
- **credit**：下游 `RemoteInputChannel` 把空闲 buffer 数作为 credit 告诉上游（每个 channel 独占 2 个，再按 backlog 申请 gate 共享的浮动 buffer），上游有 credit 才发数据，事件不消耗 credit。
- **传导**：下游慢 → 不发 credit → 上游 subpartition 堆满 → `LocalBufferPool` 不可用 → `RecordWriter.isAvailable()` 为 false → 上游暂停，一直传到 Source，表现为 Kafka 消费 lag 上涨。
- **定位**：沿数据流找第一个 backPressured 接近 0、busy 很高的算子；参考 outPoolUsage、inPoolUsage，必要时拆开算子链。

延伸阅读：[Flink 源码导读 04：网络栈与背压 —— 从 RecordWriter 到 Netty，再到 Credit 流控](/bigdata/flink/04-network)

### Watermark 是怎么生成和传播的？为什么一个空闲分区会让窗口一直不触发？

- **生成**：`WatermarkGenerator` 在 `onEvent` 里只记最大时间戳，按 `pipeline.auto-watermark-interval`（默认 200ms）周期发出「最大时间戳 − 乱序容忍 − 1」。FLIP-27 Source 按 split 各自生成，再取最小值。
- **传播**：Watermark 和数据走同一条通道，会被背压挡住；多个输入时 `StatusWatermarkValve` 取所有 ACTIVE 通道的最小值；算子先触发 Timer，再向下游转发。
- **空闲分区**：没有数据的通道 Watermark 一直是 `Long.MIN_VALUE`，占住最小值，下游 Watermark 不再前进。用 `withIdleness` 把它标为 IDLE 移出计算，或让 Source 并行度不超过分区数；背压期间空闲计时会暂停。

延伸阅读：[Flink 源码导读 08：时间、Watermark 与窗口](/bigdata/flink/08-watermark-window)

### 事件时间窗口是怎么触发的？迟到数据有哪几种结局，允许迟到时间有什么副作用？

- **触发**：窗口对齐到 UTC 纪元，左闭右开。`EventTimeTrigger` 在 `window.maxTimestamp()` 注册 Timer，Watermark 越过时 FIRE；FIRE 不清状态，清理 Timer 在 maxTimestamp 加 `allowedLateness` 时清空窗口。
- **四种结局**：Watermark 未越过窗口，正常进入；已越过但在允许迟到范围内，进状态并立即再触发一次；超出且配了侧输出，进侧输出；否则静默丢弃，只增加 `numLateRecordsDropped`。
- **副作用**：同一个窗口会多次输出完整结果，追加写会重复，下游要按 upsert 写；窗口状态也保留得更久。
- 只要聚合结果时用 `aggregate`，`process` 或 evictor 会保存全部元素。

延伸阅读：[Flink 源码导读 08：时间、Watermark 与窗口](/bigdata/flink/08-watermark-window)

### 一条 Flink SQL 是怎么变成 Transformation 的？Group Aggregate 的 changelog、MiniBatch 和两阶段聚合是怎么回事？

- **流程**：Calcite 解析出 SqlNode → 校验 → 转成 RelNode → `FlinkStreamProgram` 分阶段优化（HEP 规则加 Volcano 代价）→ 物理 RelNode → 可序列化的 ExecNode → Transformation。表达式和聚合走代码生成，在 TM 上用 Janino 编译。
- **changelog**：聚合输出 +I、-U、+U、-D；是否发 -U 由下游需求决定（sink 的 `getChangelogMode()`、下游聚合），与主键无关。
- **MiniBatch**：攒一批，每个 key 每批只读写一次状态。
- **两阶段聚合**：先 Local 再 Global，缓解热点 key；必须开 MiniBatch，聚合函数要支持 merge。COUNT DISTINCT 热点要用 `table.optimizer.distinct-agg.split.enabled`。

延伸阅读：[Flink 源码导读 05：Table / SQL —— 一条 SQL 如何变成 Transformation](/bigdata/flink/05-table-sql)

### 线上作业 Checkpoint 频繁超时，同时有背压、状态不断增长，你会怎么排查和处理？

- **看耗时拆分**：Alignment Duration 长，是背压让 Barrier 排队；Sync Duration 长，多半是 RocksDB flush 慢；Async Duration 长，是状态大或上传慢。
- **治背压**：按 busy / backPressured 找瓶颈算子，查数据倾斜（一个热点下游会拖住所有上游），加并行度或用两阶段聚合；开启 `taskmanager.network.memory.buffer-debloat.enabled`，配置对齐超时切非对齐。
- **治状态**：开 RocksDB 增量 Checkpoint；SQL 配 `table.exec.state.ttl`；窗口用 `aggregate` 代替 `process`；检查空闲分区导致窗口不触发、状态只进不出。
- **兜底**：`execution.checkpointing.tolerable-failed-checkpoints` 决定连续失败多少次让作业失败重启。

延伸阅读：[Flink 源码导读 04：网络栈与背压 —— 从 RecordWriter 到 Netty，再到 Credit 流控](/bigdata/flink/04-network)

## 答题思路

Flink 的原理题，面试官往往会顺着你的回答一路追问：「Barrier 怎么插进去的？」「对齐时阻塞的是什么？」「那背压时为什么慢？」所以回答要成链条，而不是一堆名词。比较稳的结构是三步：

1. **先讲机制**：一两句话说清楚这件事是怎么发生的。「对齐 Checkpoint 时，先收到 Barrier 的通道会被阻塞，等所有通道到齐才做快照。」
2. **再给证据**：点出关键的类、配置或指标，说明你是真看过或真调过。「阻塞是让上游停止发送，数据还留在上游的输出 buffer 里，不是缓存到下游。」
3. **最后落到生产**：代价是什么、出了问题看哪个指标、你会怎么配。「所以背压重的作业对齐时间长，我们会配对齐超时，让它自动切到非对齐。」

几个容易拉开差距的点：知道 2.x 和老资料的区别（JobGraph 在 JM 侧生成、网络 buffer 个数的配置项已删除、开启 Checkpoint 后默认指数退避重启）；能把不同模块串起来（背压 → Barrier 排队 → Checkpoint 超时；空闲分区 → Watermark 不前进 → 窗口状态只进不出）；说到 exactly-once 时，不忘提 Committer 幂等。没把握的地方直接说「这块我没读过源码，我的理解是……」，比硬编更可信。

每题后面的延伸阅读都指向 [Apache Flink 源码学习](/bigdata/flink/) 里对应的一篇，里面有源码位置和本机实测数据，答题时引用实测现象会很有说服力。

## 模拟面试

准备好了就开始一场：5 道题从上面的题库随机抽，逐题作答，每题都会得到 0–10 分的打分、点评和要点。用自己的话回答，不会的题可以直接写「不会」，看完要点再来一场。

<MockInterview topic="flink" />

## 小结

- 先把主线讲通：四张图、Mailbox 单线程模型，是理解 Checkpoint、背压、Watermark 的共同基础。
- Checkpoint、状态、网络三块要能串起来：背压让 Barrier 排队，状态大小决定同步和异步耗时，KeyGroup 决定扩缩容怎么迁移状态。
- exactly-once 记住两阶段的时机和 Committer 幂等；时间语义记住「Watermark 取最小值」和迟到数据的四种结局。
- 综合排障题按「看指标 → 定位原因 → 给出配置和代价」回答，能举实测数据更好。
