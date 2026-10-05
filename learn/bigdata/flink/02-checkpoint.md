---
title: "Flink 源码导读 02：Checkpoint 全流程 —— 从触发、Barrier 对齐到完成通知"
description: "Checkpoint 从触发、Barrier 对齐、快照到确认完成的全流程，以及非对齐 Checkpoint。"
bigdata: "flink"
lesson: "f2"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/02-checkpoint-cover.webp"}]]
---

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/02-checkpoint-cover.webp" alt="Yui和Kai引导Barrier完成Checkpoint全流程" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai引导Barrier完成Checkpoint全流程<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 前置阅读：[01 作业执行全链路](/bigdata/flink/01-execution)（Mailbox 模型、OperatorChain、网络输入这几节）
> 配套示例：`demos/`，第 10 节的所有数据都是用它在本机实测得到的

## 0. 本篇要回答的问题

1. Checkpoint 是谁、在什么时候触发的？为什么只发给 Source？
2. Barrier 是怎么插进数据流的？为什么快照和数据处理之间不需要加锁？
3. 下游收到多个上游的 Barrier 时，"对齐"到底是怎么实现的？"阻塞通道"是阻塞了什么？
4. 非对齐 Checkpoint 为什么快？代价是什么？
5. 什么时候才算 Checkpoint 完成？两阶段提交的 Sink 为什么要依赖 `notifyCheckpointComplete`？

## 1. 理论回顾：ABS 算法

Flink 的 Checkpoint 基于 **Asynchronous Barrier Snapshotting**（ABS，Chandy-Lamport 的变体）：

- JM 周期性地让 **Source** 往数据流里注入一个 **Barrier(n)**。Barrier 跟普通数据一起往下游流动，不会超车（对齐模式下）
- 一个算子从**所有**输入通道都收到 Barrier(n) 后，对自己的状态做快照，再把 Barrier(n) 继续往下游广播
- 这样每个算子的快照恰好对应"Barrier 之前的所有数据都处理完、之后的一条都没处理"这个一致的切面
- 所有 Task 都汇报完成后，JM 把这次 Checkpoint 标记为完成

三种模式的对比：

| 模式 | 多输入时的处理 | 语义 | 选择哪个 Handler（`InputProcessorUtil`） |
|---|---|---|---|
| AT_LEAST_ONCE | 不对齐，先到的通道继续处理 | 至少一次 | `CheckpointBarrierTracker`（`:129`） |
| EXACTLY_ONCE + 对齐 | 先到 Barrier 的通道**暂停消费**，等其他通道 | 精确一次 | `SingleCheckpointBarrierHandler.aligned()`（`:166`） |
| EXACTLY_ONCE + 非对齐 | Barrier **超车**，把被超过的 in-flight 数据一起存进快照 | 精确一次 | `SingleCheckpointBarrierHandler.alternating()`（`:156`） |

路径：`RT/streaming/runtime/io/checkpointing/InputProcessorUtil.java`

## 2. 全景图与线程模型

```
 JobManager                                           TaskManager
 ──────────                                           ───────────
 [Checkpoint Timer 线程]
  ScheduledTrigger.run()
  └ triggerCheckpoint()
    └ startTriggeringCheckpoint()
      ├ 计算 CheckpointPlan（要触发谁、等谁的 ack）
      ├ 创建 PendingCheckpoint + 超时 Canceller
      ├ OperatorCoordinator 快照（如 SourceCoordinator）
      └ triggerTasks() ── RPC triggerCheckpoint ──────►  [Source Task 线程 / mailbox]
                                                          performCheckpoint()
                                                          └ checkpointState()
                                                            ├ (1) prepareSnapshotPreBarrier
                                                            ├ (2) 广播 Barrier ─────► 下游 Task
                                                            ├ (5) 同步快照
                                                            └ 提交异步任务 ─► [async 快照线程池]
                                                                               上传状态文件
                          ◄── RPC acknowledgeCheckpoint ─────────────────────── ack
 [jobmanager-io 线程]
  receiveAcknowledgeMessage()                         [下游 Task 线程 / mailbox]
  └ 全部 ack → completePendingCheckpoint()             CheckpointedInputGate.pollNext()
    ├ 写 _metadata                                     └ Barrier 对齐 → performCheckpoint()（同上）
    ├ 存入 CompletedCheckpointStore                       ... 同样 ack 给 JM
    └ sendAcknowledgeMessages() ── RPC ──────────►    notifyCheckpointComplete()（mailbox）
                                                       └ 两阶段提交 Sink 在这里 commit
```

实测的线程名（第 10 节的日志里可以看到）：

| 线程 | 做什么 |
|---|---|
| `Checkpoint Timer` | 触发 Checkpoint、超时检测。单线程，在 `DefaultExecutionGraph.java:519` 创建 |
| `jobmanager-io-thread-*` | 处理 ack，完成 Checkpoint（写 metadata 可能是 IO 操作） |
| Task 线程，如 `count-per-key -> ... (1/2)#0` | 同步快照、Barrier 对齐、`notifyCheckpointComplete` 都在 **mailbox** 里执行 |
| `AsyncOperations-*` | 异步快照：把状态写到 Checkpoint 存储（线程池在 `StreamTask.java:465` 创建） |

---

## 3. 阶段一（JM）：触发 Checkpoint

### 3.1 CheckpointCoordinator 是什么时候创建和启动的？

- **创建**：01 篇 4.7 节，`DefaultExecutionGraphBuilder` 调 `executionGraph.enableCheckpointing()` → `new CheckpointCoordinator(...)`（`RT/runtime/executiongraph/DefaultExecutionGraph.java:523`）
- **注册启停监听**：`createActivatorDeactivator()`（`RT/runtime/checkpoint/CheckpointCoordinator.java:2138`），注册为作业状态监听器（`DefaultExecutionGraph.java:556`）
- **启动时机**：`ExecutionTrackingCheckpointCoordinatorDeActivator.maybeStartOrStopCheckpointScheduler()`（`RT/runtime/checkpoint/ExecutionTrackingCheckpointCoordinatorDeActivator.java:80-84`）

```java
if (jobStatus == JobStatus.RUNNING && allTasksRunning()) {
    ...
    coordinator.startCheckpointScheduler();
}
```

> 💡 **作业 RUNNING 不等于所有 Task RUNNING。** 调度器要等**每个 Task 都进入 RUNNING** 之后才启动（FLIP-331）。原因是：只要有 Task 还没部署，Barrier 就传不过去，这次 Checkpoint 注定超时。实测日志里的 `Starting checkpoint scheduler` 就是这一行打印的。

### 3.2 周期触发

`startCheckpointScheduler()`（`CheckpointCoordinator.java:2032`）→ `scheduleTriggerWithDelay()`（`:2114`）往 `timer` 上注册一个 `ScheduledTrigger`（`:2178`）：

```java
// ScheduledTrigger.run()
nextCheckpointTriggeringRelativeTime += checkpointInterval;
currentPeriodicTriggerFuture = timer.schedule(this, ...);   // 先把下一次排上
...
triggerCheckpoint(checkpointProperties, null, true);        // 再触发这一次
```

`triggerCheckpoint()`（`:567`）不会马上执行，而是先经过 **`CheckpointRequestDecider`**（`:620` 的 `chooseRequestToExecute`）。它负责执行这几条限制：

| 配置 | Key（`flink-core/.../configuration/CheckpointingOptions.java`） | 作用 |
|---|---|---|
| 间隔 | `execution.checkpointing.interval`（`:512`） | 两次触发之间的间隔 |
| 最小停顿 | `execution.checkpointing.min-pause`（`:427`） | 上一次**完成**到下一次**触发**之间至少隔多久 |
| 最大并发 | `execution.checkpointing.max-concurrent-checkpoints`（`:417`） | 同时进行的 Checkpoint 个数 |
| 超时 | `execution.checkpointing.timeout`（`:410`） | 超时即失败 |

Savepoint 请求和周期性请求都经过这个 Decider 排队，所以你手动触发的 Savepoint 可能要等当前 Checkpoint 做完才开始。

### 3.3 `startTriggeringCheckpoint()`：一条异步流水线

`CheckpointCoordinator.java:624`。这是 JM 侧最核心的方法，它是一条 `CompletableFuture` 链，**注意每一步在哪个线程上执行**（`executor` 是 IO 线程池，`timer` 是 Checkpoint Timer 线程）：

| 步骤 | 行号 | 线程 | 做什么 |
|---|---|---|---|
| ① 计算 CheckpointPlan | `:637` | — | `DefaultCheckpointPlanCalculator`：决定**触发谁**（`tasksToTrigger`）、**等谁的 ack**（`tasksToWaitFor`）、**通知谁完成**（`tasksToCommitTo`） |
| ② 分配 Checkpoint ID | `:653` | executor | `checkpointIdCounter.getAndIncrement()`。HA 模式下这个计数器存在 ZK/K8s 中，保证 JM 切换后 ID 不会回退 |
| ③ 创建 PendingCheckpoint | `:895` | timer | 同时注册超时任务 `CheckpointCanceller`（`:936`，类定义在 `:2323`） |
| ④ 初始化存储位置 | — | executor | 比如创建 `chk-N` 目录 |
| ⑤ OperatorCoordinator 快照 | `:703` | timer | 先给 JM 上的 Coordinator（如 `SourceCoordinator`）做快照 |
| ⑥ Master Hook 快照 | `:730` / `:960` | timer | 外部系统的钩子（少见） |
| ⑦ 触发 Task | `:782` → `:830` | timer | `triggerCheckpointRequest` → `triggerTasks` |

**为什么 `tasksToTrigger` 只有 Source？** `DefaultCheckpointPlanCalculator.calculateWithAllTasksRunning()`（`RT/runtime/checkpoint/DefaultCheckpointPlanCalculator.java:158`）只把 source 类型的 vertex 放进 `tasksToTrigger`。非 Source 算子由 Barrier 驱动，**不需要** JM 直接触发。当部分 Task 已经 FINISHED 时（有界流），会走 `calculateAfterTasksFinished()`（`:181`），改为触发"上游都已结束的那些正在运行的 Task"。

**为什么 Coordinator 要先于 Task 做快照？** 以 `SourceCoordinator` 为例，它负责把 split 分配给 reader，分配是通过 OperatorEvent 发送的。Coordinator 快照完成后，会**关闭**发往各 subtask 的事件通道（`OperatorCoordinatorHolder.closeGateways`，`RT/runtime/operators/coordination/OperatorCoordinatorHolder.java:344`）。这期间新发的事件会暂存在 `SubtaskGatewayImpl.blockedEventsMap`（`SubtaskGatewayImpl.java:71`）里。等 subtask 在 mailbox 中完成同步快照后，回传 `AcknowledgeCheckpointEvent`（`RT/streaming/runtime/tasks/RegularOperatorChain.java:201` → `OperatorChain.sendAcknowledgeCheckpointEvent`），通道才重新打开（`OperatorCoordinatorHolder.java:220`）。这样可以保证 "Coordinator 已经分出去的 split" 和 "Reader 已经收到的 split" 在快照里是一致的，不会有 split 丢失或被分配两次。

### 3.4 `triggerTasks()`：发 RPC

`CheckpointCoordinator.java:830`

```java
final CheckpointOptions checkpointOptions = CheckpointOptions.forConfig(
        type, location, isExactlyOnceMode, unalignedCheckpointsEnabled, alignedCheckpointTimeout);
for (Execution execution : checkpoint.getCheckpointPlan().getTasksToTrigger()) {
    acks.add(execution.triggerCheckpoint(checkpointId, timestamp, checkpointOptions));   // :858
}
```

`CheckpointOptions` 会随 Barrier 一路传给所有下游，里面带着：Checkpoint 类型（Checkpoint / Savepoint / 全量）、存储位置、是否非对齐、对齐超时。

`Execution.triggerCheckpoint()`（`RT/runtime/executiongraph/Execution.java:1070`）→ `taskManagerGateway.triggerCheckpoint()`（`:1096`）→ RPC

---

## 4. 阶段二（Source Task）：注入 Barrier 并做快照

### 4.1 RPC 进入 Task，投递到 mailbox

| 步骤 | 位置 |
|---|---|
| `TaskExecutor.triggerCheckpoint()` | `RT/runtime/taskexecutor/TaskExecutor.java:1084` → `task.triggerCheckpointBarrier()`（`:1098`） |
| `Task.triggerCheckpointBarrier()` | `RT/runtime/taskmanager/Task.java:1388` → `invokable.triggerCheckpointAsync()` |
| `SourceOperatorStreamTask.triggerCheckpointAsync()` | `RT/streaming/runtime/tasks/SourceOperatorStreamTask.java:139`，外部触发型 Source 会走特殊逻辑，普通 Source 最终调用 `super.triggerCheckpointAsync()` |
| **`StreamTask.triggerCheckpointAsync()`** | `RT/streaming/runtime/tasks/StreamTask.java:1305` |

```java
MailboxExecutor.MailOptions mailOptions =
        CheckpointOptions.AlignmentType.UNALIGNED == checkpointOptions.getAlignment()
                ? MailboxExecutor.MailOptions.urgent()     // 非对齐：插队到邮箱最前面
                : MailboxExecutor.MailOptions.options();
mainMailboxExecutor.execute(mailOptions, () -> { ... triggerCheckpointAsyncInMailbox(...) ... });
```

**关键点**：RPC 线程**不直接**做快照，而是往 Task 的 mailbox 投递一封邮件。回忆 01 篇第 6 节：mailbox 里的邮件和数据处理在**同一个线程上串行执行**，所以快照发生时，一定不会有哪条记录正处理到一半，也就**不需要锁**。这是引入 Mailbox 模型之后，Checkpoint 实现最大的变化：早期版本需要在 checkpointLock 下做快照。

### 4.2 `performCheckpoint()`

`triggerCheckpointAsyncInMailbox()`（`StreamTask.java:1344`）→ `performCheckpoint()`（`:1467`）→ `subtaskCheckpointCoordinator.checkpointState(...)`（`:1493`）

如果 Task 已经不在运行，就广播一个 `CancelCheckpointMarker` 给下游，免得下游一直在等这个 Barrier。

### 4.3 ★ `SubtaskCheckpointCoordinatorImpl.checkpointState()`

`RT/streaming/runtime/tasks/SubtaskCheckpointCoordinatorImpl.java:271`，是整篇**最重要的方法**。源码注释把它分成了 6 步：

```java
// All of the following steps happen as an atomic step from the perspective of barriers and
// records/watermarks/timers/callbacks.
```

| 步骤 | 行号 | 做什么 |
|---|---|---|
| Step (0) | `:301` | 记录 `lastCheckpointId`；如果这个 Checkpoint 已经被通知取消，就广播 `CancelCheckpointMarker` 后直接返回 |
| **Step (1)** | `:331` | `operatorChain.prepareSnapshotPreBarrier(id)`：Barrier 发出**之前**的最后机会，让算子把缓存的数据刷出去 |
| **Step (2)** | `:335` | `operatorChain.broadcastEvent(checkpointBarrier, options.isUnalignedCheckpoint())`：**向下游广播 Barrier** |
| Step (3) | `:346` | 注册对齐超时定时器（对齐 → 非对齐切换用，见第 7 节） |
| Step (4) | `:349` | 非对齐模式：结束输出侧 in-flight 数据的收集 |
| **Step (5)** | `:355` | `takeSnapshotSync()`（`:724`）同步快照，成功后 `finishAndReportAsync()`（`:689`）把异步部分交给线程池 |

**为什么先发 Barrier、再做快照？** 这样下游可以尽早开始对齐，与本 Task 的快照并行进行。只要这 6 步在同一次 mailbox 动作里完成，中间就不会插入任何数据处理，一致性就有保证。

**Step (1) 的典型用法**：`SinkWriterOperator.prepareSnapshotPreBarrier()`（`RT/streaming/runtime/operators/sink/SinkWriterOperator.java:188`）

```java
sinkWriter.flush(false);
emitCommittables(checkpointId);    // 把本轮待提交的事务句柄发给下游的 CommitterOperator
```

Committable 要在 Barrier **之前**发出，才能保证下游 Committer 在收到 Barrier 做快照时，已经拿到了本轮所有的 Committable。

### 4.4 同步快照：`takeSnapshotSync()`

`SubtaskCheckpointCoordinatorImpl.java:724` → `operatorChain.snapshotState()`（`RT/streaming/runtime/tasks/RegularOperatorChain.java:180`）对链上**每个算子**调 `buildOperatorSnapshotFutures()`（`:204`）→ `AbstractStreamOperator.snapshotState()`（`RT/streaming/api/operators/AbstractStreamOperator.java:421`）→ **`StreamOperatorStateHandler.snapshotState()`**（`RT/streaming/api/operators/StreamOperatorStateHandler.java:178`）：

| 顺序 | 行号 | 内容 |
|---|---|---|
| 1 | `:257` | 定时器：`timeServiceManager.snapshotToRawKeyedState()` |
| 2 | `:261` | **用户的 `snapshotState()`**：`CheckpointedFunction.snapshotState()` 就是在这里被调用的 |
| 3 | `:269` | Operator State：`operatorStateBackend.snapshot(...)` |
| 4 | `:295` | Keyed State：`keyedStateBackend.snapshot(...)` |

3 和 4 返回的都是 `RunnableFuture`，**此时还没有真正写文件**。以 Keyed State 为例，状态后端通过 `SnapshotStrategyRunner.snapshot()`（`RT/runtime/state/SnapshotStrategyRunner.java:70`）把快照拆成两段：

| 阶段 | 行号 | 在哪个线程 | HashMap 状态后端 | RocksDB 状态后端 |
|---|---|---|---|---|
| **同步**：`syncPrepareResources` | `:77` | Task 线程（**会阻塞数据处理**） | 对 `StateTable` 做 copy-on-write 快照（很快） | 触发 RocksDB native checkpoint：flush memtable，对 SST 文件做硬链接 |
| **异步**：`asyncSnapshot` | `:80` | AsyncOperations 线程 | 把快照序列化后写入 Checkpoint 存储 | 把**新增的** SST 文件上传（增量 Checkpoint） |

具体实现见 `HeapSnapshotStrategy`，以及 RocksDB 的 `RocksDBSnapshotStrategyBase.syncPrepareResources()`（`flink-state-backends/flink-statebackend-rocksdb/.../snapshot/RocksDBSnapshotStrategyBase.java:151`，其中 `takeDBNativeCheckpoint` 在 `:161`）和 `RocksIncrementalSnapshotStrategy`，第 03 篇会展开讲。

> 📊 Web UI 上 Checkpoint 详情里的 **Sync Duration** 和 **Async Duration** 就是这两段各自的耗时。Sync 很长通常说明 RocksDB flush 慢；Async 很长通常说明状态大，或者上传到存储慢。

### 4.5 异步阶段与 ack

`finishAndReportAsync()`（`SubtaskCheckpointCoordinatorImpl.java:689`）创建一个 `AsyncCheckpointRunnable`（`:698`），提交到 `asyncOperationsThreadPool`（`:716`）。

`AsyncCheckpointRunnable.run()`（`RT/streaming/runtime/tasks/AsyncCheckpointRunnable.java:109`）：
1. `finalizeNonFinishedSnapshots()`（`:177`）：逐个执行上面那些 `RunnableFuture`，得到状态句柄（StateHandle，也就是"状态存在哪个文件的哪个位置"）
2. `reportCompletedSnapshotStates()`（`:218`）→ `TaskStateManagerImpl.reportTaskStateSnapshots()`（`RT/runtime/state/TaskStateManagerImpl.java:148`）：
   - 如果开了本地恢复，还会在本地存一份（`:156`），重启时可以直接从本地盘恢复
   - `checkpointResponder.acknowledgeCheckpoint(...)`（`:158`）→ `RpcCheckpointResponder`（`RT/runtime/taskexecutor/rpc/RpcCheckpointResponder.java:50`）→ RPC 到 JM

**ack 里带回的是 StateHandle（文件路径 + 偏移），不是状态数据本身**，所以 ack 消息很小。

---

## 5. 阶段三（下游 Task）：Barrier 对齐

<figure class="ai-figure"><img src="/bigdata-img/flink/02-checkpoint-1.webp" alt="下游等待所有输入Barrier后再快照" width="960" height="640" loading="lazy" /><figcaption>下游等待所有输入Barrier后再快照<span>AI 生成配图</span></figcaption></figure>

### 5.1 Barrier 在哪里被拦截？

回忆 01 篇 6.4 节：`AbstractStreamTaskNetworkInput.emitNext()` 读完一个 buffer 后调用 `checkpointedInputGate.pollNext()`。**Barrier 在这里就被处理掉了**，不会进入算子。

`RT/streaming/runtime/io/checkpointing/CheckpointedInputGate.java`

```java
pollNext()                              // :150
└ handleEvent(bufferOrEvent)            // :178
   ├ CheckpointBarrier       → barrierHandler.processBarrier(...)              // :182
   ├ CancelCheckpointMarker → barrierHandler.processCancellationBarrier(...)  // :184
   ├ EndOfPartition         → barrierHandler.processEndOfPartition(...)       // :190
   └ EventAnnouncement      → barrierHandler.processBarrierAnnouncement(...)  // :199
```

### 5.2 `SingleCheckpointBarrierHandler.processBarrier()`

`RT/streaming/runtime/io/checkpointing/SingleCheckpointBarrierHandler.java:214`

```java
if (currentCheckpointId > barrierId || (currentCheckpointId == barrierId && !isCheckpointPending())) {
    // 过期的 Barrier（比如这个 Checkpoint 已经被取消了）：恢复该通道，直接忽略
    ...
    return;
}
checkNewCheckpoint(barrier);     // :335 —— 第一次见到这个 ID：记下要等几个通道（targetChannelCount）
markCheckpointAlignedAndTransformState(channelInfo, barrier,
        state -> state.barrierReceived(context, channelInfo, barrier, !isRpcTriggered));
```

`markCheckpointAlignedAndTransformState()` 负责：记录对齐开始和结束时间（就是 UI 上的 **Alignment Duration**），然后把事件交给**状态机**。

### 5.3 对齐状态机

对齐逻辑写成了一个状态机，每个状态是一个类（都在 `checkpointing/` 目录下）：

**纯对齐模式**（`aligned()`，`SingleCheckpointBarrierHandler.java:147`）：

```
  ┌──────────────────────┐   收到第 1 个 barrier   ┌───────────────────┐
  │ WaitingForFirstBarrier│ ───────────────────────►│ CollectingBarriers │
  └──────────────────────┘   阻塞该通道             └───────────────────┘
            ▲                                        │ 每收到一个 barrier：阻塞该通道
            │   triggerGlobalCheckpoint()            │
            └────────────────────────────────────────┘ 所有通道都到齐
                 + unblockAllChannels()
```

核心代码在 `AbstractAlignedBarrierHandlerState`（`AbstractAlignedBarrierHandlerState.java:53`）：

```java
if (markChannelBlocked) {
    state.blockChannel(channelInfo);                       // :62
}
if (controller.allBarriersReceived()) {
    return triggerGlobalCheckpoint(controller, checkpointBarrier);   // → :72
}
```

```java
protected WaitingForFirstBarrier triggerGlobalCheckpoint(...) {
    controller.triggerGlobalCheckpoint(checkpointBarrier);   // 做快照
    state.unblockAllChannels();                             // 恢复所有通道
    return new WaitingForFirstBarrier(state.getInputs());
}
```

`triggerGlobalCheckpoint` → `CheckpointBarrierHandler.notifyCheckpoint()`（`CheckpointBarrierHandler.java:125`）→ `toNotifyOnCheckpoint.triggerCheckpointOnBarrier(...)`（`:147`）→ **`StreamTask.triggerCheckpointOnBarrier()`**（`StreamTask.java:1430`）→ `performCheckpoint()`。之后的流程与 Source 完全相同（4.2 节起）：发 Barrier 给自己的下游、做快照、ack。

### 5.4 ★ "阻塞通道"到底阻塞了什么？

很多资料说"对齐时把先到 Barrier 的通道的数据缓存起来"，这是 **Flink 早期版本的实现**（`BarrierBuffer` 会把数据缓存到磁盘）。现在的实现完全不同：**让上游停止发送**。

1. 对齐 Barrier 的 buffer 类型是 `ALIGNED_CHECKPOINT_BARRIER`（`RT/runtime/io/network/buffer/Buffer.java:298`），它的 `isBlockingUpstream` 属性为 true
2. 上游的 `PipelinedSubpartition` 把这个 buffer 交给网络层之后，**自己把自己标记为阻塞**（`RT/runtime/io/network/partition/PipelinedSubpartition.java:534-536`）：

   ```java
   if (buffer.getDataType().isBlockingUpstream()) {
       isBlocked = true;     // 之后不再向这个下游通道发送任何数据
   }
   ```
3. 下游对齐完成后调用 `ChannelState.unblockAllChannels()`（`ChannelState.java:74`）→ `resumeConsumption()` → 通过网络通知上游 → `PipelinedSubpartition.resumeConsumption()`（`:567`）把 `isBlocked` 改回 false

所以对齐期间**被阻塞的数据还留在上游的输出 buffer 里**，下游不需要额外的内存。代价是：上游的 buffer 写满后会产生**背压**，在有背压的作业里，对齐时间就是 "Barrier 排在 in-flight 数据后面、等前面的数据处理完" 的时间。第 10 节的实验会量化这一点。

---

## 6. 非对齐 Checkpoint（Unaligned Checkpoint，FLIP-76）

<figure class="ai-figure"><img src="/bigdata-img/flink/02-checkpoint-2.webp" alt="非对齐Checkpoint让Barrier超车并保存途中数据" width="960" height="640" loading="lazy" /><figcaption>非对齐Checkpoint让Barrier超车并保存途中数据<span>AI 生成配图</span></figcaption></figure>

### 6.1 思路

对齐慢的根本原因是：Barrier 必须**排队**，等前面的 in-flight 数据都处理完。非对齐的做法是让 Barrier **超车**，把被它超过的数据一起存进快照：

```
对齐：      [d5][d4][d3][d2][d1][B]  ──►  算子     B 要等 d1..d5 处理完才能被处理
非对齐：    [B][d5][d4][d3][d2][d1]  ──►  算子     B 直接到队首；d1..d5 作为 channel state 存进快照
```

恢复时，先把 channel state 里的数据重新灌回输入输出通道，再继续处理，结果与对齐模式完全一致。

### 6.2 输出侧：Barrier 超车 + 持久化被超过的 buffer

在 Step (2) 中，`broadcastEvent(barrier, isPriorityEvent=true)`（`RT/runtime/io/network/api/writer/RecordWriter.java:129`）把 Barrier 作为**优先级事件**（`PRIORITIZED_EVENT_BUFFER`，`Buffer.java:292`）加入 subpartition：

`PipelinedSubpartition.processPriorityBuffer()`（`PipelinedSubpartition.java:218`）

```java
buffers.addPriorityElement(...);            // 插到队列最前面
// 把排在它后面、还没发出去的数据 buffer 全部复制一份
while (iterator.hasNext()) { ... inflightBuffers.add(bc.build()); }
channelStateWriter.addOutputData(barrier.getId(), subpartitionInfo, ..., inflightBuffers);   // :241
```

### 6.3 输入侧：持久化 Barrier 到齐前收到的数据

下游收到**第一个**非对齐 Barrier 时（`AlternatingWaitingForFirstBarrierUnaligned.barrierReceived()`，`AlternatingWaitingForFirstBarrierUnaligned.java:59`）：

```java
CheckpointBarrier unalignedBarrier = checkpointBarrier.asUnaligned();
controller.initInputsCheckpoint(unalignedBarrier);
for (CheckpointableInput input : channelState.getInputs()) {
    input.checkpointStarted(unalignedBarrier);      // 所有输入通道开始记录 in-flight 数据
}
controller.triggerGlobalCheckpoint(unalignedBarrier);   // ★ 不等其他通道，立刻做快照
```

之后，**其他通道**在它们的 Barrier 到达之前收到的 buffer，都会被 `ChannelStatePersister` 持久化（`RemoteInputChannel.checkpointStarted()`，`RT/runtime/io/network/partition/consumer/RemoteInputChannel.java:714`；`channelStatePersister.maybePersist(buffer)`，`:647`）。所有通道的 Barrier 都到齐后，停止记录。

输入和输出两侧的 channel state 最终都由 `ChannelStateWriterImpl`（`RT/runtime/checkpoint/channel/`）异步写入 Checkpoint 存储，并计入 Checkpoint 大小（`OperatorSubtaskState.java:144-161`：`streamChannelStates()` 被包含在 `stateSize` 里）。

### 6.4 对齐超时：先对齐，超时后切换到非对齐

开启非对齐后，Handler 实际用的是 **alternating** 状态机。配置了 `execution.checkpointing.aligned-checkpoint-timeout`（`CheckpointingOptions.java:565`）时：
- 先按对齐方式进行（`AlternatingWaitingForFirstBarrier` → `AlternatingCollectingBarriers`）
- 超时后调用 `alignedCheckpointTimeout()`，切换成非对齐（`AlternatingCollectingBarriers.java:41` → `AlternatingCollectingBarriersUnaligned`）
- 发送端也有对应的定时器：Step (3) 注册的 `registerAlignmentTimer`，超时后把输出队列里还没发出的对齐 Barrier 改成优先级 Barrier（`PipelinedSubpartition.java:337` 附近，`findInflightBuffersAndMakeBarrierToPriority`）

这样没有背压时走对齐（Checkpoint 小），背压严重时自动切换成非对齐（Checkpoint 快）。**生产环境推荐这样配置。**

### 6.5 代价与限制

- Checkpoint 变大：in-flight 数据越多越大（实测见第 10 节）
- 恢复变慢：要先把 channel state 回放一遍
- 只支持 EXACTLY_ONCE（`InputProcessorUtil.java:119-123`：AT_LEAST_ONCE 下开启会直接抛异常）
- Savepoint 永远是对齐的
- 部分场景会被**强制对齐**（`FORCED_ALIGNED`，如 pointwise 连接），见 `SubtaskCheckpointCoordinatorImpl.java:326` 的处理

---

## 7. 阶段四（JM）：收集 ack、完成 Checkpoint

### 7.1 ack 到达 CheckpointCoordinator

| 步骤 | 位置 |
|---|---|
| `JobMaster.acknowledgeCheckpoint()` | `RT/runtime/jobmaster/JobMaster.java:621` |
| `SchedulerBase.acknowledgeCheckpoint()` | `RT/runtime/scheduler/SchedulerBase.java:1043` |
| `ExecutionGraphHandler.acknowledgeCheckpoint()` | `RT/runtime/scheduler/ExecutionGraphHandler.java:100`，**切换到 ioExecutor 线程**（`:136`），不占用 JobMaster 主线程 |
| **`CheckpointCoordinator.receiveAcknowledgeMessage()`** | `CheckpointCoordinator.java:1204` |

```java
switch (checkpoint.acknowledgeTask(...)) {      // :1248 → PendingCheckpoint.java:385
    case SUCCESS:
        if (checkpoint.isFullyAcknowledged()) {   // notYetAcknowledgedTasks 为空
            completePendingCheckpoint(checkpoint);
        }
        break;
    case DUPLICATE: ...   // 重复 ack
    case UNKNOWN: ...     // 不认识的 Task（比如已经 failover 了）
    case DISCARDED: ...   // Checkpoint 已经被取消，要把 ack 带来的状态文件删掉
}
```

### 7.2 `completePendingCheckpoint()`

`CheckpointCoordinator.java:1359`

| 步骤 | 位置 | 做什么 |
|---|---|---|
| `finalizeCheckpoint()` | `:1464` → `PendingCheckpoint.finalizeCheckpoint()`（`PendingCheckpoint.java:317`） | **写 `_metadata` 文件**（`:339` `Checkpoints.storeCheckpointMetadata`）。写完这个文件，Checkpoint 才算真正落盘 |
| `addCompletedCheckpointToStoreAndSubsumeOldest()` | `:1504` | 存入 `CompletedCheckpointStore`（HA 下还会写 ZK/K8s），并删除超出保留数量的旧 Checkpoint |
| `reportCompletedCheckpoint()` | — | `failureManager.handleCheckpointSuccess()`，清零连续失败计数 |
| `scheduleTriggerRequest()` | `finally` 块中 | 如果有排队中的请求（比如被 min-pause 拦住的），这时可以放行了 |
| `cleanupAfterCompletedCheckpoint()` | `:1415` | 丢弃更早的 PendingCheckpoint，然后通知所有 Task |

### 7.3 `notifyCheckpointComplete`：通知 Task

`sendAcknowledgeMessages()`（`CheckpointCoordinator.java:1570`）对 `tasksToCommitTo` 中的每个 Task 调用 `ee.notifyCheckpointOnComplete(...)`（`:1579`）

→ `Execution.notifyCheckpointOnComplete()`（`Execution.java:1015`）→ RPC
→ `TaskExecutor.confirmCheckpoint()`（`TaskExecutor.java:1116`）→ `Task.notifyCheckpointComplete()`（`Task.java:1473`）
→ `StreamTask.notifyCheckpointCompleteAsync()`（`StreamTask.java:1560`）：**再次投递到 mailbox**
→ `notifyCheckpointComplete()`（`:1612`）→ `subtaskCheckpointCoordinator.notifyCheckpointComplete()` → 链上每个算子的 `notifyCheckpointComplete()`

**两阶段提交就是在这里完成的**：`CommitterOperator.notifyCheckpointComplete()`（`RT/streaming/runtime/operators/sink/CommitterOperator.java:159`）真正提交事务（比如 Kafka 事务的 commit）。

> ⚠️ **`notifyCheckpointComplete` 不保证一定会送达**：RPC 可能丢失，TM 可能在收到通知前挂掉。所以 Flink 的约定是：Committable 本身会存进 Checkpoint 状态，作业恢复时重新提交一遍。这就要求 **commit 操作必须幂等**。自己实现 Sink 时一定要注意这一点。

### 7.4 失败路径

| 情况 | 位置 |
|---|---|
| 超时 | `CheckpointCanceller`（`:2323`）→ `abortPendingCheckpoint(...)` |
| Task 拒绝（比如 Task 正在关闭） | Task 发 `declineCheckpoint` → `receiveDeclineMessage()`（`:1123`） |
| 失败计数 | `CheckpointFailureManager.handleCheckpointException()`（`RT/runtime/checkpoint/CheckpointFailureManager.java:93`）→ `checkFailureCounter()`（`:217`）：连续失败次数超过 `execution.checkpointing.tolerable-failed-checkpoints`（`CheckpointingOptions.java:446`）后，**让整个作业失败** |
| 取消后通知下游 | 广播 `CancelCheckpointMarker`，下游 `processCancellationBarrier`（`SingleCheckpointBarrierHandler.java:356`）解除阻塞 |

---

## 8. Checkpoint 目录长什么样？

示例作业结束后，查看 `/tmp/flink-ckpt-demo/<jobId>/`：

```
chk-6/                 # 最新一次 Checkpoint（默认只保留 1 个：execution.checkpointing.num-retained）
  _metadata            # 元数据：各算子的 StateHandle 列表。恢复时就是读这个文件
  <uuid>               # 状态数据文件（小状态可能直接内联进 _metadata）
shared/                # 增量 Checkpoint 中多个 Checkpoint 共享的文件（RocksDB 的 SST）
taskowned/             # TM 独占的文件（比如 changelog）
```

- 默认情况下，作业**取消**时 Checkpoint 会被删除。示例里设置了 `execution.checkpointing.externalized-checkpoint-retention = RETAIN_ON_CANCELLATION`，所以能在作业结束后查看
- 从 Checkpoint 恢复：`flink run -s /tmp/flink-ckpt-demo/<jobId>/chk-6 ...`

---

## 9. 断点清单（按执行顺序）

用 IDEA 打开 `flink-notes/demos`，运行 `CheckpointDemo`，程序参数填 `slow`。断点**务必设为 Suspend: Thread**（原因见 01 篇），否则 Checkpoint 会超时。

| # | 线程 | 位置 | 看什么 |
|---|---|---|---|
| 1 | pekko dispatcher | `ExecutionTrackingCheckpointCoordinatorDeActivator.java:84` | 调度器什么时候启动 |
| 2 | Checkpoint Timer | `CheckpointCoordinator.java:624` `startTriggeringCheckpoint` | |
| 3 | Checkpoint Timer | `DefaultCheckpointPlanCalculator.java:158` | `tasksToTrigger` 里只有 source |
| 4 | Checkpoint Timer | `CheckpointCoordinator.java:858` | `checkpointOptions` 的内容 |
| 5 | Source Task 线程 | `StreamTask.java:1344` `triggerCheckpointAsyncInMailbox` | 注意线程名：它就是 Task 线程本身 |
| 6 | Source Task 线程 | `SubtaskCheckpointCoordinatorImpl.java:344` 广播 Barrier | |
| 7 | 下游 Task 线程 | `SingleCheckpointBarrierHandler.java:214` `processBarrier` | `channelInfo`：从哪个通道来的 |
| 8 | 下游 Task 线程 | `AbstractAlignedBarrierHandlerState.java:62` `blockChannel` | 第一个 Barrier 到达后阻塞该通道 |
| 9 | 下游 Task 线程 | `StreamOperatorStateHandler.java:295` keyed state 快照 | |
| 10 | AsyncOperations | `AsyncCheckpointRunnable.java:109` | 异步上传 |
| 11 | jobmanager-io | `CheckpointCoordinator.java:1248` `acknowledgeTask` | 看 `notYetAcknowledgedTasks` 逐渐变少 |
| 12 | jobmanager-io | `PendingCheckpoint.java:339` 写 `_metadata` | |
| 13 | 下游 Task 线程 | 示例中 `CheckpointAwareMap.notifyCheckpointComplete` | 两阶段提交在这个时机 commit |

---

## 10. 实验：对齐 vs 非对齐

### 10.1 示例作业

`demos/src/main/java/study/checkpoint/CheckpointDemo.java`，拓扑：

```
datagen(2 并行度，共 2000 条/秒，每条约 1KB) ─hash─► count-per-key(ValueState) → checkpoint-aware-map → discard-sink
```

- Checkpoint 间隔 5 秒，存储为 `file:///tmp/flink-ckpt-demo`，运行 30 秒后自动取消
- `slow` 参数：每条记录 `Thread.sleep(5)`，每个并行度最多处理约 200 条/秒，远低于约 1000 条/秒的输入，于是产生**稳定的背压**
- `unaligned` 参数：开启非对齐 Checkpoint

运行方式：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.checkpoint.CheckpointDemo slow
```

### 10.2 实测结果（本机，Flink 2.3.0，JDK 17）

取日志中 `Completed checkpoint N (... bytes, checkpointDuration=... ms)`：

| 模式 | ckpt 1 | ckpt 2 | ckpt 3 | ckpt 4 | ckpt 5 | ckpt 6 |
|---|---|---|---|---|---|---|
| 对齐，无背压 | 41ms / 5.7KB | 17ms / 5.7KB | 20ms / 5.7KB | 19ms / 5.7KB | 16ms / 5.7KB | 16ms / 5.7KB |
| 对齐 + 背压 | 359ms / 5.7KB | **3029ms** / 5.7KB | **3861ms** / 5.7KB | **3877ms** / 5.7KB | **3907ms** / 5.7KB | **3903ms** / 5.7KB |
| 非对齐 + 背压 | 43ms / 91KB | 26ms / **694KB** | 22ms / **688KB** | 26ms / **590KB** | 17ms / **614KB** | 21ms / **625KB** |

（两次重复运行的结果基本一致）

**解读**：
1. **对齐 + 背压**：耗时从 ~20ms 涨到 ~3.9s，但大小不变。这 3.9 秒就是 Barrier 排在满载的网络 buffer 后面等待的时间（5.4 节）。背压越重、buffer 越多，等得越久。在生产环境中，这就是"背压导致 Checkpoint 超时"的根本原因
2. **非对齐 + 背压**：耗时回到 ~20ms，代价是每次多存了约 600KB 的 in-flight 数据（channel state）。600KB 大致就是网络 buffer 里积压的数据量：约 600 条 × 1KB
3. **Keyed State 本身只有 5.7KB**（16 个 key 的计数 + source split 状态），在这个例子里，in-flight 数据比状态本身大了 100 倍

### 10.3 自己动手

- 加上 `env.getCheckpointConfig().setAlignedCheckpointTimeout(Duration.ofSeconds(1))`，观察 Checkpoint 耗时是否稳定在约 1 秒，大小是否介于两者之间
- 把 `Thread.sleep(5)` 改成 `Thread.sleep(1)`，观察背压消失后，对齐 Checkpoint 是否也变快了
- 设置 `taskmanager.network.memory.buffer-debloat.enabled: true`（Buffer Debloating，FLIP-183），观察对齐耗时的变化。思考：它为什么能缩短对齐时间？

> 🐛 **踩坑记录**：写这组实验时，我在 zsh 的 for 循环里用 `java ... $mode` 传参，结果 "非对齐" 那一组的数据一直不对。原因是 **zsh 默认不对未加引号的变量分词**，`"slow unaligned"` 被当成一个参数传了进去，程序以为没有参数。要写成 `${=mode}`，或者直接用 bash。数据和预期对不上时，先怀疑实验本身。

---

## 11. 课后练习

1. **读测试**：`flink-runtime/src/test/java/org/apache/flink/streaming/runtime/io/checkpointing/` 下有对齐和非对齐 Handler 的单元测试，挑一个测试，画出它构造的 Barrier 到达顺序，以及每一步的状态机状态
2. **思考题**：为什么 `triggerCheckpointAsync` 对非对齐 Checkpoint 使用 `MailOptions.urgent()`，而对齐 Checkpoint 不用？（提示：非对齐的目标是"尽快"，那对齐的 Source 为什么不需要插队？）
3. **思考题**：Step (2) 发 Barrier 在 Step (5) 快照之前。如果 Step (5) 失败了，下游已经收到了 Barrier，会发生什么？（提示：看 `cleanup()` 和 `declineCheckpoint` 的流程）
4. **Savepoint**：跟一遍 `flink savepoint <jobId>` 的路径：`Dispatcher.triggerSavepoint` → ... → `CheckpointCoordinator.triggerSavepoint`。它和周期性 Checkpoint 有哪些区别？（提示：`CheckpointProperties`、`SavepointType`、强制对齐）
5. **有界流的最后一次 Checkpoint**：`execution.checkpointing.checkpoints-after-tasks-finish` 是做什么的？为什么两阶段提交的 Sink 需要它？读一读 `DefaultCheckpointPlanCalculator.calculateAfterTasksFinished()`

## 12. 下一篇预告

**03：State Backend**。HashMap 与 RocksDB 状态后端的存储结构、`KeyGroup` 与 rescale 时的状态重分配、RocksDB 增量 Checkpoint 的 `shared/` 目录是怎么复用 SST 文件的，以及 Flink 2.x 存算分离的 ForSt 状态后端和异步状态访问（State V2）。
:::
