---
title: "Flink 源码导读 01：从 `env.execute()` 到 `processElement()` —— 作业执行全链路"
description: "用户代码怎样变成 StreamGraph、JobGraph、ExecutionGraph，Task 怎样部署运行到 processElement()。"
bigdata: "flink"
---

# Flink 源码导读 01：从 `env.execute()` 到 `processElement()` —— 作业执行全链路

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过，其他版本行号会有偏移，按方法名搜索即可。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`

## 0. 本篇要回答的问题

1. 我写的 `map/keyBy/sum` 是怎样变成一张图的？这张图经历了哪几次变换？
2. 作业提交后，JobManager 里的哪些组件依次接手？
3. Task 是怎样被部署到 TaskManager 上并跑起来的？
4. 一条数据在 Task 内部是怎样被一路传到 `processElement()` 的？为什么 Flink 的算子不需要加锁？

## 1. 全景图

```
用户代码 (Transformation 列表)
   │ StreamGraphGenerator            ① 客户端
   ▼
StreamGraph   ──提交──►  MiniCluster / Dispatcher
                              │ JobMasterServiceLeadershipRunner → JobMaster
                              │ DefaultSchedulerFactory
                              ▼
                         JobGraph   (StreamingJobGraphGenerator：算子链在这里决定)  ② JM 侧
                              │ DefaultExecutionGraphBuilder
                              ▼
                       ExecutionGraph (按并行度展开)
                              │ DefaultScheduler → PipelinedRegionSchedulingStrategy
                              │ → DefaultExecutionDeployer → Execution.deploy()
                              ▼  RPC: submitTask(TaskDeploymentDescriptor)
                        TaskExecutor → Task (独立线程)                              ③ TM 侧
                              │ StreamTask.restore() / invoke()
                              ▼
                        MailboxProcessor 循环
                              │ processInput() → emitNext() → processElement()     ④ 数据处理
```

**四张图**是贯穿全篇的主线：

| 图 | 在哪生成 | 节点含义 | 关键特点 |
|---|---|---|---|
| Transformation 列表 | 用户调用 API 时 | 每个 API 调用 | 只是记录，不做任何计算 |
| StreamGraph | 客户端 `StreamGraphGenerator` | 一个算子一个 `StreamNode` | 逻辑图，边上带分区器 |
| JobGraph | **JM 侧**（2.x 新变化）`StreamingJobGraphGenerator` | 一条算子链一个 `JobVertex` | 算子链在这一步完成 |
| ExecutionGraph | JM 侧 `DefaultExecutionGraphBuilder` | 每个并行实例一个 `ExecutionVertex` | 调度、容错的真正对象 |

> ⚠️ **与老资料的区别**：Flink 1.x 在客户端生成 JobGraph 后再提交。2.x 中 `StreamGraph` 和 `JobGraph` 都实现了 `ExecutionPlan` 接口，本地模式和 Session 模式会**直接提交 StreamGraph**，由 JM 侧调度器生成 JobGraph。这是为了让批作业能根据运行时信息**动态调整图**（AdaptiveBatchScheduler，见 `StreamGraphOptimizer`）。

## 2. 准备：在 IDEA 里跑起来

1. IDEA 打开 `flink/pom.xml`，SDK 选 JDK 17，等待索引完成
2. 找到 `flink-examples/flink-examples-streaming/src/main/java/org/apache/flink/streaming/examples/wordcount/WordCount.java`，直接运行 `main`
3. 在 IDE 里运行时没有 CLI 上下文，`getExecutionEnvironment()` 会走到 `createLocalEnvironment()`（`RT/streaming/api/environment/StreamExecutionEnvironment.java:2197`）。于是：
   - `execution.target = local`（`LocalStreamEnvironment.java:61`）
   - 默认并行度 = CPU 核数（`StreamExecutionEnvironment.java:161`）
   - 会在**同一个 JVM** 里起一个 `MiniCluster`，JM 和 TM 都在里面，**所有断点都能打**

WordCount 的核心代码：

```java
text = env.fromData(WordCountData.WORDS)          // 源
counts = text.flatMap(new Tokenizer())            // 切词
             .keyBy(value -> value.f0)            // 按单词分区
             .sum(1);                             // 聚合，名字叫 "Keyed Aggregation"
counts.print().name("print-sink");              // Sink（旧版 SinkFunction，对应算子 StreamSink）
env.execute("WordCount");
```

> 💡 调试技巧：断点上右键 → 勾选 "Suspend: Thread" 而不是 "All"，否则一停全停，RPC 超时会让作业失败。也可以在 `MiniCluster` 相关的配置里调大 `pekko.ask.timeout`。

---

## 3. 第一段（客户端）：Transformation → StreamGraph

### 3.1 API 调用只是在"记账"

`flatMap`、`keyBy`、`sum` 本身不执行任何计算，只是 new 一个 `Transformation` 加到 `env.transformations` 列表里。`keyBy` 甚至不产生算子，只生成一个 `PartitionTransformation`（虚拟节点，带 `KeyGroupStreamPartitioner`）。

### 3.2 `execute()` 入口

| 步骤 | 位置 | 说明 |
|---|---|---|
| `execute(String)` | `StreamExecutionEnvironment.java:1838` | 调 `getStreamGraph()` 生成图 |
| `getStreamGraphGenerator(...).generate()` | `:2044` / `:2075` | |
| `execute(StreamGraph)` | `:1871` | 调 `executeAsync`，attached 模式下阻塞等结果 |
| `executeAsync(StreamGraph)` | `:1988` | 通过 SPI 找到 `PipelineExecutor` 并提交 |
| `getPipelineExecutor()` | `:2509` | 根据 `execution.target` 选 executor（local / remote / yarn / k8s） |

### 3.3 `StreamGraphGenerator.generate()`

`RT/streaming/api/graph/StreamGraphGenerator.java:253`

```java
for (Transformation<?> transformation : transformations) {
    transform(transformation);          // :463
}
```

`transform()` 做了三件事：
1. **去重**：`alreadyTransformed` 保证同一个 Transformation 只转换一次（`:464`）
2. **递归**：先转换上游（通过各个 translator 内部调用），保证父节点先入图
3. **按类型分派**：`translatorMap.get(transform.getClass())`，比如 `OneInputTransformation` → `OneInputTransformationTranslator`，最终调用 `streamGraph.addOperator()` / `addEdge()`

**边上的分区器是怎么定的？** `RT/streaming/api/graph/StreamGraph.java:912-920`：

```java
// If no partitioner was specified and the parallelism of upstream and downstream
// operator matches use forward partitioning, use rebalance otherwise.
if (partitioner == null && upstreamNode.getParallelism() == downstreamNode.getParallelism()) {
    partitioner = ... new ForwardPartitioner<>();
} else if (partitioner == null) {
    partitioner = new RebalancePartitioner<Object>();
}
```

记住这一条，下面判断算子链时会用到。

### 3.4 提交给 MiniCluster

| 步骤 | 位置 |
|---|---|
| `LocalExecutor.execute()` | `flink-clients/.../deployment/executors/LocalExecutor.java:83` |
| `PerJobMiniClusterFactory.submitJob()`：启动 MiniCluster 并提交 | `flink-clients/.../program/PerJobMiniClusterFactory.java:71` |
| `MiniCluster.submitJob(ExecutionPlan)`：克隆图、上传 jar 到 BlobServer，调 `dispatcherGateway.submitApplication(new SingleJobApplication(...))` | `RT/runtime/minicluster/MiniCluster.java:1099` |

> 2.x 新增了 **Application** 概念：一次提交先包装成 `SingleJobApplication`（`RT/runtime/application/SingleJobApplication.java:75`），它再调用 `Dispatcher.submitJob()`。

---

## 4. 第二段（JobManager）：Dispatcher → JobMaster → Scheduler

### 4.1 RPC 线程模型：先理解这个，JM 代码才读得懂

`Dispatcher`、`JobMaster`、`ResourceManager`、`TaskExecutor` 都继承自 `RpcEndpoint`（`flink-rpc/flink-rpc-core/.../rpc/RpcEndpoint.java`）。

- 每个 `RpcEndpoint` 有一个**单线程的主线程执行器**（`getMainThreadExecutor()`，`:357`），所有 RPC 方法都在这个线程上串行执行。所以这些组件的内部状态**不需要加锁**
- 耗时操作（IO、生成图）会丢到 ioExecutor 上执行，完成后再通过 `CompletableFuture...Async(..., mainThreadExecutor)` 切回主线程
- 所以 JM 侧代码里到处都是 `CompletableFuture` 链。**读代码时要盯住每个 lambda 在哪个线程上执行**

### 4.2 Dispatcher：接收作业

`RT/runtime/dispatcher/Dispatcher.java`

| 方法 | 行号 | 作用 |
|---|---|---|
| `submitJob` | `:835` | 检查作业是否已全局终止、是否重复提交 |
| `internalSubmitJob` | `:1270` | 等待同 ID 的旧作业结束 |
| `persistAndRunJob` | `:1333` | ① 持久化 ExecutionPlan（HA 下写 ZK/K8s ConfigMap）② 创建 runner |
| `createJobMasterRunner` | `:1342` | 通过工厂创建 `JobMasterServiceLeadershipRunner` |

### 4.3 选主 → 创建 JobMaster

每个作业都有自己的 leader 选举（HA 场景下同一作业可能有多个 JM 候选）：

1. `JobMasterServiceLeadershipRunner.start()` → 参与选举（`RT/runtime/jobmaster/JobMasterServiceLeadershipRunner.java:164`）
2. 当选后回调 `grantLeadership()`（`:243`）→ `createNewJobMasterServiceProcess()`（`:317`）
3. `DefaultJobMasterServiceProcess`（`RT/runtime/jobmaster/DefaultJobMasterServiceProcess.java:92`）
4. `DefaultJobMasterServiceFactory.internalCreateJobMasterService()`：`new JobMaster(...)`（`:112`），然后 `jobMaster.start()`（`:138`）

### 4.4 JobMaster 构造时就创建了调度器（重点）

`RT/runtime/jobmaster/JobMaster.java:406` → `createScheduler()`（`:423`）→ `DefaultSchedulerFactory.createInstance()`

**JobGraph 就是在这里生成的**：`RT/runtime/scheduler/DefaultSchedulerFactory.java:88-96`

```java
if (executionPlan instanceof JobGraph) {
    jobGraph = (JobGraph) executionPlan;
} else if (executionPlan instanceof StreamGraph) {
    jobGraph = ((StreamGraph) executionPlan).getJobGraph(userCodeLoader);  // ← 断点
}
```

`StreamGraph.getJobGraph()`（`RT/streaming/api/graph/StreamGraph.java:1195`）→ `StreamingJobGraphGenerator.createJobGraph()`

### 4.5 StreamGraph → JobGraph：算子链

`RT/streaming/api/graph/StreamingJobGraphGenerator.java:222`

```java
private JobGraph createJobGraph() {
    preValidate(...);
    setChaining();                         // ★ 核心：从源头 DFS，决定哪些算子合成一条链
    ...
    setPhysicalEdges(...);                 // 链之间的边 → JobEdge + IntermediateDataSet
    setSlotSharingAndCoLocation(...);      // Slot 共享组
    setManagedMemoryFraction(...);         // 托管内存（RocksDB、批算子）怎么分
    serializeOperatorCoordinatorsAndStreamConfig(...);  // 算子配置序列化进 JobVertex
    return jobGraph;
}
```

**两个算子能否链接**，看 `isChainable()`（`:1734`）和 `isChainableInput()`（`:1756`），全部满足才能链接：

1. 下游只有**一条输入边**
2. 全局没有禁用算子链（`streamGraph.isChainingEnabled()`，即没有调 `disableOperatorChaining()`）
3. 两个算子在**同一个 slot sharing group**
4. 两个算子的 `ChainingStrategy` 允许（`areOperatorsChainable`，`:1802`）：上游不是 `NEVER`；下游不是 `NEVER`/`HEAD`；`HEAD_WITH_SOURCES` 仅当上游是 Source
5. 两个算子的**并行度相同**（`:1866`）
6. 分区器是 `ForwardPartitioner`，并且不是 BATCH 交换模式（`arePartitionerAndExchangeModeChainable`，`:1786`）
7. 不是 union 的多个同类型输入

### 4.6 🧪 实战推演：WordCount 会切成几个 Task？

设 IDE 里默认并行度为 N（CPU 核数，比如 8）：

| 边 | 分区器 | 能否链接 | 原因 |
|---|---|---|---|
| Source → FlatMap | **Rebalance** | ❌ | `fromData` 固定 `setParallelism(1)`（`StreamExecutionEnvironment.java:843`），与 FlatMap 的 N 不同，StreamGraph 自动换成了 Rebalance（见 3.3） |
| FlatMap → Keyed Aggregation | **Hash**（keyBy） | ❌ | 不是 Forward |
| Keyed Aggregation → print-sink | Forward | ✅ | 并行度相同、Forward、同组 |

所以 JobGraph 有 **3 个 JobVertex**：`Source: in-memory-input`(1) → `Flat Map`(N) → `Keyed Aggregation -> print-sink`(N)。

**练习**：给 `flatMap` 后面加一个 `.setParallelism(1)`，在 `setChaining` 里打断点验证 Source 和 FlatMap 链接到了一起；Web UI 里也能看到节点合并了。

### 4.7 JobGraph → ExecutionGraph

调度器父类构造函数里完成：`RT/runtime/scheduler/SchedulerBase.java:255` → `createAndRestoreExecutionGraph()`（`:407`）→ `DefaultExecutionGraphFactory`（`:155`）→ `DefaultExecutionGraphBuilder.buildGraph()`（`RT/runtime/executiongraph/DefaultExecutionGraphBuilder.java:77`）

- `executionGraph.attachJobGraph(...)`（`:199`）→ `DefaultExecutionGraph.attachJobGraph`（`:867`）→ 每个 `JobVertex` 生成一个 `ExecutionJobVertex`，再按并行度展开成 N 个 `ExecutionVertex`，每个 `ExecutionVertex` 持有当前一次执行尝试 `Execution`
- `executionGraph.enableCheckpointing(...)`（`:312`）→ **`CheckpointCoordinator` 在这里创建**（下一篇的主角）
- 如果是从 Checkpoint/Savepoint 恢复，也在这一步把状态分配给各个 vertex

### 4.8 开始调度

`JobMaster.start()` → RPC 框架在主线程回调 `onStart()`（`JobMaster.java:482`）→ `startJobExecution()`（`:1172`）：

```java
shuffleMaster.registerJob(context);
startJobMasterServices();   // 启动心跳、连接 ResourceManager（:1202）、SlotPool
startScheduling();          // → schedulerNG.startScheduling()（:1273）
```

| 步骤 | 位置 | 说明 |
|---|---|---|
| `SchedulerBase.startScheduling()` | `SchedulerBase.java:669` | 注册作业指标，启动所有 `OperatorCoordinator`（如 `SourceCoordinator`） |
| `DefaultScheduler.startSchedulingInternal()` | `DefaultScheduler.java:249` | |
| `PipelinedRegionSchedulingStrategy.startScheduling()` | `RT/runtime/scheduler/strategy/...:182` | 以 **Pipelined Region** 为单位调度，流作业一般整张图就是一个 region |
| `scheduleRegion()` | `:282` | → `schedulerOperations.allocateSlotsAndDeploy()` |
| `DefaultScheduler.allocateSlotsAndDeploy()` | `DefaultScheduler.java:485` | |
| `DefaultExecutionDeployer.allocateSlotsAndDeploy()` | `RT/runtime/scheduler/DefaultExecutionDeployer.java:90` | ① `allocateSlotsFor` 申请 slot（`:121`）② `waitForAllSlotsAndDeploy`，**所有 slot 就位后才一起部署**（`:151`） |
| `deployTaskSafe()` | `:320` | → `executionOperations.deploy(execution)` |

**Slot 从哪来？** 申请请求经 `SlotPool` → `ResourceManager`，RM 让 TaskExecutor 把 slot 直接**提供**给 JobMaster（`JobMaster.offerSlots()`，`:741`），SlotPool 再把 slot 分给等待中的请求。在 MiniCluster 里这些组件都在同一个 JVM 中，但走的仍然是同一套 RPC 接口。

### 4.9 `Execution.deploy()`

`RT/runtime/executiongraph/Execution.java:572`

1. 状态机：`SCHEDULED → DEPLOYING`
2. 用 `TaskDeploymentDescriptorFactory` 生成 **TDD**（`TaskDeploymentDescriptor`）：包含 JobVertex 的配置（序列化的算子链）、要读哪些上游分区、要写哪些分区、要恢复的状态句柄
3. 在 futureExecutor 上异步调用 `taskManagerGateway.submitTask(tdd, timeout)`（`:656`）→ `RpcTaskManagerGateway` → RPC 到 `TaskExecutor`

---

## 5. 第三段（TaskManager）：Task 线程跑起来

### 5.1 `TaskExecutor.submitTask()`

`RT/runtime/taskexecutor/TaskExecutor.java:660`

- 校验 slot 确实分配给了这个作业
- 加载 TDD 里的大对象（可能放在 BlobServer 上）
- `new Task(...)`（`:837`）
- `taskSlotTable.addTask(task)`（`:877`）
- `task.startTaskThread()`（`:883`）→ **每个 Task 一个独立线程**

### 5.2 `Task.run()` → `doRun()`

`RT/runtime/taskmanager/Task.java:585`，是 TM 侧最重要的方法之一，按顺序做了：

1. 创建用户代码类加载器（下载 jar，**user-code classloader** 就在这里）
2. `setupPartitionsAndGates()`（`:676`）：初始化本 Task 的 `ResultPartition`（输出）和 `InputGate`（输入），并向网络栈注册
3. `loadAndInstantiateInvokable()`（`:756`）：反射实例化 invokable。流作业里是 `StreamTask` 的子类：
   - 源头 Task → `SourceOperatorStreamTask`
   - 单输入 → `OneInputStreamTask`
   - 双输入（join/connect）→ `TwoInputStreamTask`
4. `restoreAndInvoke()`（`:944`）

```java
transitionState(DEPLOYING, INITIALIZING);   // 汇报给 JM
invokable.restore();                        // 恢复状态、打开算子
transitionState(INITIALIZING, RUNNING);     // 汇报给 JM → Web UI 变绿
invokable.invoke();                         // 主循环，直到数据结束或被取消
```

状态变化通过 `taskManagerActions.updateTaskExecutionState()` 汇报，最终 RPC 到 `JobMaster.updateTaskExecutionState()`（`JobMaster.java:537`），驱动 ExecutionGraph 中的状态机。

### 5.3 `StreamTask.restore()` / `invoke()`

`RT/streaming/runtime/tasks/StreamTask.java`

**restoreInternal()**（`:792`）：
- 创建 `OperatorChain`（`:806`）：按 StreamConfig 把链上的算子实例化，并用 `ChainingOutput` 把它们串起来
- `init()`：子类初始化 input processor（比如 `OneInputStreamTask` 创建 `StreamOneInputProcessor`）
- `restoreStateAndGates()`：→ `operatorChain.initializeStateAndOpenOperators()`（`:876`）→ 各算子依次执行 `initializeState()`（恢复状态）和 `open()`（用户的 `RichFunction.open` 在这里被调用）
- 然后运行一次 mailbox 循环，直到所有 InputGate 恢复完毕（非对齐 Checkpoint 恢复 in-flight 数据要用到）

**invoke()**（`:945`）：

```java
if (!isRunning) restoreInternal();
scheduleBufferDebloater();
runMailboxLoop();        // ★ 主循环
afterInvoke();           // 数据结束：endInput、finish、等待最后一次 checkpoint
```

---

## 6. 第四段：Mailbox 线程模型与数据处理

### 6.1 为什么需要 Mailbox？

一个 Task 线程要处理这几类事情：**处理数据**、**触发定时器**、**执行 Checkpoint**、**处理 RPC 过来的通知**（比如 notifyCheckpointComplete）。早期版本用一把大锁（checkpointLock）让这些线程互斥，既容易出错又影响性能。

Mailbox 模型（FLINK-12477，1.9/1.10 引入）把这些都变成**同一个线程上串行执行的任务**：
- **默认动作（default action）**：处理输入数据，即 `StreamTask.processInput()`
- **邮件（mail）**：定时器、Checkpoint、异步回调等，其他线程通过 `MailboxExecutor.execute()` 把它们投递进邮箱

所以算子里的状态、定时器回调、`snapshotState()` 都在同一个线程上运行，**用户代码和算子实现都不需要加锁**。

### 6.2 主循环

`RT/streaming/runtime/tasks/mailbox/MailboxProcessor.java:214`

```java
while (isNextLoopPossible()) {
    processMail(localMailbox, false);                        // 1. 先把邮箱里的邮件处理完
    if (isNextLoopPossible()) {
        mailboxDefaultAction.runDefaultAction(controller);   // 2. 再处理一批数据
    }
}
```

默认动作在构造时注册：`new MailboxProcessor(this::processInput, ...)`（`StreamTask.java:421`）。

### 6.3 `StreamTask.processInput()`：处理数据，同时感知背压

`StreamTask.java:655`

```java
DataInputStatus status = inputProcessor.processInput();
switch (status) {
    case MORE_AVAILABLE: if (taskIsAvailable()) return;   // 还有数据，下一轮继续
    case END_OF_INPUT:   mailboxProcessor.suspend(); ...   // 输入全部结束
    ...
}
// 走到这里说明暂时不能处理数据：
if (!recordWriter.isAvailable())  → 下游没有空闲 buffer = 【背压】，记到 backPressuredTime
else if (!inputProcessor.isAvailable()) → 没有输入数据 = 【空闲】，记到 idleTime
// 然后挂起默认动作，等 future 完成再恢复，期间线程只处理邮件
```

**这就是 Web UI 上 Busy / Idle / BackPressured 三个指标的来源。**

### 6.4 一条记录的旅程（以 `Flat Map` 这个 Task 为例）

```
StreamOneInputProcessor.processInput()           RT/streaming/runtime/io/StreamOneInputProcessor.java:64
  └─ AbstractStreamTaskNetworkInput.emitNext()   RT/streaming/runtime/io/AbstractStreamTaskNetworkInput.java:152
       ├─ 当前 buffer 还有数据 → currentRecordDeserializer.getNextRecord()（:159）反序列化
       │    └─ processElement(element, output)（:210）
       │         ├─ 是 Record    → output.emitRecord()（:213）
       │         ├─ 是 Watermark → 交给 StatusWatermarkValve，对齐多个输入通道后再 emitWatermark（:251）
       │         └─ ...
       └─ buffer 读完了 → checkpointedInputGate.pollNext()（:178）取下一个 buffer 或事件
            └─ 是 CheckpointBarrier 等事件 → processEvent()（:260）→ Barrier 对齐（下一篇）
OneInputStreamTask.StreamTaskNetworkOutput.emitRecord()   RT/streaming/runtime/tasks/OneInputStreamTask.java:249
  └─ recordProcessor.accept(record)    // 由 RecordProcessorUtils.getRecordProcessor(operator) 生成（:245）
       └─ operator.setKeyContextElement(record)  // keyed 算子：先切换当前 key
       └─ operator.processElement(record)        // ★ 到达！
            └─ StreamFlatMap → userFunction.flatMap(value, collector)
                 └─ output.collect(...)
```

**算子的输出去哪？** 取决于 `OperatorChain` 给它配的 `Output`：

| Output 类型 | 位置 | 场景 |
|---|---|---|
| `ChainingOutput` | `RT/streaming/runtime/tasks/ChainingOutput.java:73` | 链内的下一个算子：`pushToOperator()` **直接方法调用**，不序列化，不经过网络 |
| `RecordWriterOutput` | `RT/streaming/runtime/io/RecordWriterOutput.java:137` | 链的末尾：`recordWriter.emit()` → 按分区器选择 channel → 序列化写入 `BufferBuilder` → 写满后交给 `ResultPartition` → Netty 发给下游 Task |
| `BroadcastingOutputCollector` | 同目录 | 一个算子有多个下游 |

所以在 `Keyed Aggregation -> Sink` 这个 Task 里，`StreamGroupedReduceOperator.processElement`（`sum` 被翻译成 reduce，见 `ReduceTransformationTranslator`）算出结果后，通过 `ChainingOutput` **直接调用** `StreamSink.processElement`，是同一个调用栈。

### 6.5 源头 Task 有什么不同？

`Source: in-memory-input` 这个 Task 没有网络输入，它的输入是 `StreamTaskSourceInput`：

`StreamTaskSourceInput.emitNext()`（`RT/streaming/runtime/io/StreamTaskSourceInput.java:61`）→ `SourceOperator.emitNext()`（`RT/streaming/api/operators/SourceOperator.java:516`）→ `sourceReader.pollNext(output)`（`:535`）

这就是 **FLIP-27 新 Source 架构**：`SourceReader` 运行在 Task 线程中，通过 `pollNext` 被 mailbox 循环驱动；数据分片（split）由 JM 侧的 `SourceCoordinator` 负责发现和分配。旧的 `SourceFunction.run()` 会独占线程，必须另起一个线程并加锁，这正是它被废弃的原因之一。

---

## 7. 断点清单（按执行顺序）

把下面这些位置全部打上断点，运行一次 WordCount，按 F9 依次跳过，就能完整走一遍链路：

| # | 线程 | 位置 | 看什么 |
|---|---|---|---|
| 1 | main | `StreamExecutionEnvironment.java:1838` `execute` | `transformations` 列表 |
| 2 | main | `StreamGraphGenerator.java:463` `transform` | 每种 Transformation 如何变成 StreamNode |
| 3 | main | `StreamGraph.java:912` 分区器选择 | Source→FlatMap 为什么是 Rebalance |
| 4 | main | `MiniCluster.java:1099` `submitJob` | 提交的是 StreamGraph |
| 5 | Dispatcher 主线程 | `Dispatcher.java:1333` `persistAndRunJob` | |
| 6 | JM 线程 | `DefaultSchedulerFactory.java:92` | StreamGraph → JobGraph |
| 7 | JM 线程 | `StreamingJobGraphGenerator.java:1756` `isChainableInput` | 算子链判定 |
| 8 | JM 线程 | `DefaultExecutionGraph.java:867` `attachJobGraph` | 按并行度展开 |
| 9 | JobMaster 主线程 | `JobMaster.java:1172` `startJobExecution` | |
| 10 | JobMaster 主线程 | `PipelinedRegionSchedulingStrategy.java:282` `scheduleRegion` | |
| 11 | JobMaster 主线程 | `DefaultExecutionDeployer.java:151` `waitForAllSlotsAndDeploy` | |
| 12 | future 线程 | `Execution.java:656` `submitTask` | TDD 里有什么 |
| 13 | TaskExecutor 主线程 | `TaskExecutor.java:837` `new Task` | |
| 14 | Task 线程 | `Task.java:944` `restoreAndInvoke` | 状态迁移 |
| 15 | Task 线程 | `StreamTask.java:806` 创建 OperatorChain | 链上有哪些算子 |
| 16 | Task 线程 | `MailboxProcessor.java:227` 主循环 | |
| 17 | Task 线程 | `AbstractStreamTaskNetworkInput.java:170` | 反序列化出的第一条记录 |
| 18 | Task 线程 | `StreamMap.java:36` / 你的 `Tokenizer.flatMap` | **终点** |

> 线程名在 IDEA 的 Debugger → Threads 面板可以看到：Task 线程的名字形如 `Flat Map (3/8)#0`，JobMaster 主线程形如 `flink-pekko.actor.default-dispatcher-*`（MiniCluster 本地 RPC 时也可能直接在调用方线程执行）。

---

## 8. 课后练习

1. **算子链**：在 WordCount 里调用 `env.disableOperatorChaining()`，Web UI 上会有几个节点？再试试只给 `sum` 设置 `.slotSharingGroup("g2")`，又会怎样？结合 4.5 的规则解释
2. **状态迁移**：在 `Task.transitionState` 打断点，画出 Task 从 `CREATED` 到 `FINISHED` 的状态图，对照 `ExecutionState` 枚举
3. **背压实验**：在 Sink 前加一个 `map` 里 `Thread.sleep(10)`，在 `StreamTask.java:687` 附近观察 `recordWriter.isAvailable()` 什么时候变成 false
4. **读测试**：读 `flink-streaming-java/src/test/java/org/apache/flink/streaming/api/graph/StreamingJobGraphGeneratorTest.java`，挑三个测试用例，说明它们验证的是哪条链接规则
5. **思考题**：为什么 2.x 要把 JobGraph 的生成挪到 JM 侧？提示：读 `AdaptiveGraphManager`、`StreamGraphOptimizer` 和 `AdaptiveBatchScheduler`，再到 FLIP 列表里找"StreamGraph 提交"和"自适应批执行"相关的提案

## 9. 下一篇预告

**02：Checkpoint 全流程**，从 `CheckpointCoordinator` 触发开始，经过 Source 注入 Barrier、`SingleCheckpointBarrierHandler` 对齐、`SubtaskCheckpointCoordinatorImpl` 做快照，到 ack 汇总和 `notifyCheckpointComplete`，并对比对齐和非对齐两种模式。
:::
