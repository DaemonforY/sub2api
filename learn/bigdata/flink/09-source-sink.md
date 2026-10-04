---
title: "Flink 源码导读 09：Source / Sink 新架构 —— FLIP-27 与 Sink V2"
description: "FLIP-27 新 Source 架构（SplitEnumerator / SourceReader）与新版 Sink 的两阶段提交。"
bigdata: "flink"
---

# Flink 源码导读 09：Source / Sink 新架构 —— FLIP-27 与 Sink V2

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`CORE` = `flink-core/src/main/java/org/apache/flink`，`RT` = `flink-runtime/src/main/java/org/apache/flink`，`BASE` = `flink-connectors/flink-connector-base/src/main/java/org/apache/flink/connector/base`
> 前置阅读：[02](/bigdata/flink/02-checkpoint)（Checkpoint 的触发顺序、`prepareSnapshotPreBarrier`）、[06](/bigdata/flink/06-scheduling)（region failover）、[08](/bigdata/flink/08-watermark-window) 第 1.3 节和第 6 节
> 配套示例：`demos/src/main/java/study/connector/SourceSinkDemo.java`，第 6 节的所有输出都是在本机实测的

## 0. 本篇要回答的问题

1. 新 Source 为什么要拆成 **Enumerator（JM）+ Reader（TM）** 两部分？它们之间怎么通信？
2. split 分配出去之后作业失败了，这个 split 会不会丢？会不会被读两次？
3. `SourceReaderBase` 里的 fetcher 线程和 Task 线程是怎么配合的？背压怎么传到外部系统？
4. Sink V2 的 Writer 和 Committer 是怎么被"展开"成算子的？两阶段提交的两个阶段分别发生在哪一刻？
5. 为什么说 **Committer 必须幂等**？
6. 不开 Checkpoint 时，事务型 Sink 什么时候提交？

一张图先看全貌：

```
            JobManager                                             TaskManager（每个并行度一个 Task，全部 chain 在一起）
┌───────────────────────────────────┐                 ┌───────────────────────────────────────────────────────────────┐
│ SourceCoordinator（单线程）          │  OperatorEvent  │ SourceOperator                                                 │
│  └ SplitEnumerator                │◀──────────────▶│  └ SourceReader ── pollNext ──▶ Map ──▶ SinkWriterOperator      │
│     handleSplitRequest / addReader│   RequestSplit  │                                          └ SinkWriter           │
│     addSplitsBack / snapshotState │   AddSplit      │                                              prepareCommit()    │
│  └ SplitAssignmentTracker         │   ...           │                                                 │ committables │
│ OperatorCoordinatorHolder         │                 │                                          CommitterOperator      │
│  └ SubtaskGatewayImpl（Checkpoint   │                 │                                           └ Committer.commit()  │
│    期间把事件"关在门外"）             │                 │                                             （Checkpoint 完成后） │
└───────────────────────────────────┘                 └───────────────────────────────────────────────────────────────┘
```

---

## 1. Source API：三个角色

`CORE/api/connector/source/` 下的接口：

| 接口 | 运行在 | 核心方法 |
|---|---|---|
| `Source` | 工厂，两边都会用 | `getBoundedness()`（`Source.java:45`）、`createEnumerator`（`:55`）、`restoreEnumerator`（`:68`）、`createReader`（继承自 `SourceReaderFactory`）、两个序列化器（`:81`、`:89`） |
| `SplitEnumerator` | **JM**，每个 Source 一个 | `start`（`SplitEnumerator.java:42`）、`handleSplitRequest`（`:52`）、`addSplitsBack`（`:61`）、`addReader`（`:68`）、`snapshotState`（`:90`） |
| `SourceReader` | **TM**，每个并行度一个 | `start`（`SourceReader.java:60`）、`pollNext`（`:73`）、`snapshotState`（`:80`）、`isAvailable`（`:102`）、`addSplits`（`:111`）、`notifyNoMoreSplits`（`:120`） |
| `SourceSplit` | 两边都有 | 只有一个 `splitId()`；**split 对象本身就是"读到哪里了"的状态** |

老的 `SourceFunction` 把"发现分区"和"读数据"写在同一个 `run()` 循环里，每个并行度各自发现分区、各自决定读哪些，所以动态发现分区、批流统一、Watermark 对齐都很难做。FLIP-27 把**发现与分配**集中到 JM 上的 Enumerator 中，Reader 只负责读 Enumerator 分给它的 split。

分配有两种模式，都在 `SplitEnumeratorContext` 上：
- **推（push）**：Enumerator 主动调用 `assignSplits(...)`（`SplitEnumeratorContext.java:107`），例如 Kafka 发现新分区后直接分配。
- **拉（pull）**：Reader 调用 `SourceReaderContext.sendSplitRequest()`（`SourceReaderContext.java:55`），Enumerator 在 `handleSplitRequest` 中响应，例如 FileSource 读完一个文件再要下一个。它天然能做负载均衡：读得快的 Reader 会拿到更多 split。

本篇示例用的是"拉"模式。

---

## 2. JM 侧：SourceCoordinator

### 2.1 它从哪里来

`SourceOperatorFactory` 实现了 `CoordinatedOperatorFactory`，生成 JobGraph 时会把 `SourceCoordinatorProvider` 挂到 JobVertex 上。ExecutionGraph 构建时为它创建 `OperatorCoordinatorHolder`，调度器启动时再创建真正的 `SourceCoordinator`：

```java
// RT/runtime/source/coordinator/SourceCoordinatorProvider.java
public OperatorCoordinator getCoordinator(OperatorCoordinator.Context context) {       // :77
    final String coordinatorThreadName = "SourceCoordinator-" + operatorName;        // :78
    ...
    return new SourceCoordinator<>(...);                                              // :91
}
```

`SourceCoordinatorContext` 用这个线程工厂创建了一个**单线程**的 `ScheduledExecutorService`（`SourceCoordinatorContext.java:133`）。**Enumerator 的所有方法都在这个线程里执行**，所以 Enumerator 的实现不需要加锁。示例日志中，所有 `ENUM` 行的线程名都是 `SourceCoordinator-Source: range-source`。

Enumerator 在 `start()` 中创建（`SourceCoordinator.java:235`、`:252`）；从 Checkpoint 恢复时则在 `resetToCheckpoint()` 中调用 `source.restoreEnumerator(...)`（`:492`、`:524`）。

### 2.2 事件分发

Reader 发来的所有事件都进入同一个入口，放到 Coordinator 线程中执行：

```java
public void handleEventFromOperator(int subtask, int attemptNumber, OperatorEvent event) {   // :325
    runInEventLoop(() -> {
        if (event instanceof RequestSplitEvent)          handleRequestSplitEvent(...);     // → enumerator.handleSplitRequest (:660)
        else if (event instanceof SourceEventWrapper)    handleSourceEvent(...);           // 自定义事件
        else if (event instanceof ReaderRegistrationEvent) handleReaderRegistrationEvent(...); // → enumerator.addReader (:702)
        else if (event instanceof ReportedWatermarkEvent)  handleReportedWatermark(...);     // 08 篇第 6 节的 Watermark 对齐
        ...
```

Enumerator 调用 `context.assignSplits(...)` 时（`SourceCoordinatorContext.java:288`）：
1. 检查目标 subtask 已经注册过（没注册就抛 `IllegalArgumentException`）。
2. **`assignmentTracker.recordSplitAssignment(assignment)`**（`:305`）：记下"这些 split 是在上一个 Checkpoint 之后分出去的"。
3. 把 split 序列化成 `AddSplitEvent`（`:631`），通过 RPC 发给 Task。

### 2.3 `SplitAssignmentTracker`：split 不丢的关键

`RT/runtime/source/coordinator/SplitAssignmentTracker.java` 只有两个数据结构：

```java
SortedMap<Long, Map<Integer, LinkedHashSet<SplitT>>> assignmentsByCheckpointId;   // :47  已经归入某个 Checkpoint 的分配
Map<Integer, LinkedHashSet<SplitT>> uncheckpointedAssignments;                     // :50  上一个 Checkpoint 之后的新分配

void onCheckpoint(long checkpointId) {                                            // :63  Coordinator 做快照时
    assignmentsByCheckpointId.put(checkpointId, uncheckpointedAssignments);
    uncheckpointedAssignments = new HashMap<>();
}
void onCheckpointComplete(long checkpointId) {                                    // :96  Checkpoint 完成：之前的记录都可以扔了
    assignmentsByCheckpointId.entrySet().removeIf(entry -> entry.getKey() <= checkpointId);
}
List<SplitT> getAndRemoveUncheckpointedAssignment(int subtaskId, long restoredCheckpointId) {   // :120
    // 取出所有"晚于 restoredCheckpointId"的分配，再加上 uncheckpointedAssignments
}
```

某个 subtask 失败并回滚到 Checkpoint N 时，`SourceCoordinator.subtaskReset()`（`:372`）调用上面的 `getAndRemoveUncheckpointedAssignment(subtaskId, N)`，把结果通过 **`enumerator.addSplitsBack(...)`** 还给 Enumerator（`:389`）。

为什么只还"N 之后分配的"？因为 **N 之前分配的 split 已经在 Reader 的状态里了**：它们会随着 Reader 的状态一起恢复，由 Reader 自己接着读。N 之后分配的 split 在 Checkpoint N 中没有任何记录，如果不还给 Enumerator，就会丢失。这就回答了开头的第 2 个问题。**每个 split 在任意 Checkpoint 中，要么在 Enumerator 的状态里，要么在某个 Reader 的状态里，而且只在一处。**

### 2.4 Coordinator 的 Checkpoint：先于 Barrier，并且"关门"

02 篇讲过：`CheckpointCoordinator` 先触发所有 OperatorCoordinator 的快照（`CheckpointCoordinator.java:702-703`），**全部完成后**才向 Source Task 注入 Barrier（`:791`）。

这里有一个竞争问题：Coordinator 在 Checkpoint N 的快照中已经把 split-X 记为"已分配给 subtask 1"，但对应的 `AddSplitEvent` 如果在 Barrier N **之后**才到达 subtask 1，subtask 1 的 Reader 状态中就没有 split-X，split-X 就丢了。

`OperatorCoordinatorHolder`（`RT/runtime/operators/coordination/OperatorCoordinatorHolder.java`）用"关门"来解决。类注释 `:70-90` 写得很清楚，实现如下：

```java
private void checkpointCoordinatorInternal(long checkpointId, CompletableFuture<byte[]> result) {   // :307
    coordinatorCheckpoint.handleAsync((success, failure) -> {
        ... closeGateways(checkpointId) ...                           // :319 快照完成：关门
        completeCheckpointOnceEventsAreDone(checkpointId, result, success);   // :360 等门关上之前发出的事件都送达
    });
    subtaskGatewayMap.forEach((subtask, gateway) -> gateway.markForCheckpoint(checkpointId));   // :335
    coordinator.checkpointCoordinator(checkpointId, coordinatorCheckpoint);
}
```

- **关门之后**，Coordinator 发出的事件会被 `SubtaskGatewayImpl` 暂存起来（`SubtaskGatewayImpl.java:147-148`），因为它们属于 Checkpoint N 之后的阶段。
- **关门之前**发出、但还没确认送达的事件，由 `completeCheckpointOnceEventsAreDone` 等待，全部送达后才算 Coordinator 快照完成。如果其中某个事件送达失败，这个 Checkpoint 直接失败（`:390-395`）。
- **开门**：Task 在同步阶段做完所有算子的快照后，发送 `AcknowledgeCheckpointEvent`（`RegularOperatorChain.java:201` → `OperatorChain.java:968-980`）。Holder 收到后，打开这个 subtask 的门，并发出暂存的事件（`OperatorCoordinatorHolder.java:217-221`）。Checkpoint 完成或中止时，也会打开所有门（`:253-262`）。

所以任何一个 `AddSplitEvent`，要么在 Barrier 之前到达（split 进入 Reader 状态），要么在 Barrier 之后到达（split 在 Coordinator 快照中属于"N 之后的分配"），不会落到两者之间的缝隙里。

---

## 3. TM 侧：SourceOperator 与 SourceReaderBase

### 3.1 `SourceOperator`：Reader 的宿主

`RT/streaming/api/operators/SourceOperator.java`

```java
public void open() {                                                     // :416
    ...
    final List<SplitT> splits = CollectionUtil.iterableToList(readerState.get());   // :444 恢复出来的 split
    if (!splits.isEmpty() && !supportsSplitReassignmentOnRecovery) {
        sourceReader.addSplits(splits);                                  // :451 直接交还给自己的 Reader
    }
    registerReader(...);                                                 // :455 发 ReaderRegistrationEvent
    sourceReader.start();                                                // :459
    eventTimeLogic.startPeriodicWatermarkEmits();                        // 08 篇
}

public DataInputStatus emitNext(DataOutput<OUT> output) {                // :516 Mailbox 默认动作每次调用一次
    ...
    do {
        status = sourceReader.pollNext(currentMainOutput);               // :535
    } while (status == InputStatus.MORE_AVAILABLE
            && canEmitBatchOfRecords.check()                             // :537 Mailbox 里有别的事就让出
            && !shouldWaitForAlignment());
    return convertToInternalStatus(status);
}

public void snapshotState(StateSnapshotContext context) {                // :661
    readerState.update(sourceReader.snapshotState(checkpointId));        // :669 Reader 状态 = 它手上所有 split 的当前进度
}
```

split 状态存在一个名为 `"SourceReaderState"` 的 **operator state**（`ListState<byte[]>`，`:114`）中。它是 operator state 而不是 keyed state，扩缩容时按 round-robin 重新分配（03 篇第 6 节）。

`handleOperatorEvent`（`:719`）收到 `AddSplitEvent` 后，先为新 split 创建独立的 Watermark 输出（08 篇第 1.3 节），再调用 `sourceReader.addSplits(...)`（`:743-757`）。这个方法在 **Task 的 Mailbox 线程**中执行，所以 `SourceReader` 的所有方法都在同一个线程里调用，这也是 `SourceReader` 不需要加锁的原因。

**Flink 2.2 起的新变化**（勘误 2026-10-02：原写为 2.3，release notes 显示 FLINK-38564 在 2.2 引入，见 `docs/content/release-notes/flink-2.2.md:175-183`）：实现了 `SupportsSplitReassignmentOnRecovery` 的 Source（FLINK-38564，FLIP-537），恢复时**不再**把 split 直接交还给原来的 Reader，而是随 `ReaderRegistrationEvent` 上报给 Enumerator（`:455`），由 Enumerator 统一重新分配。这样 Enumerator 就能掌握全局的 split 分布。

### 3.2 `SourceReaderBase`：线程模型

`pollNext` 在 Mailbox 线程中执行，**不能阻塞**，否则 Checkpoint、Timer 都无法处理（01 篇 Mailbox）。但是大多数外部客户端（Kafka Consumer、HDFS 输入流）的读取接口是阻塞的。`flink-connector-base` 提供的 `SourceReaderBase` 用"fetcher 线程 + 有界队列"解决这个问题：

```
  Fetcher 线程（"Source Data Fetcher for ..."，BASE/source/reader/fetcher/SplitFetcherManager.java:64）
  ┌─────────────────────────────────────────────┐
  │ SplitFetcher.run() → FetchTask.run()         │
  │   lastRecords = splitReader.fetch();   阻塞读 │  FetchTask.java:58
  │   elementsQueue.put(fetcherIndex, lastRecords)│  :64 队列满就阻塞 → 不再向外部系统拉数据
  └─────────────────────┬───────────────────────┘
                        │ FutureCompletingBlockingQueue（容量 source.reader.element.queue.capacity，默认 2 批）
                        ▼
  Task 线程（Mailbox）
  ┌─────────────────────────────────────────────┐
  │ SourceReaderBase.pollNext()                  │  BASE/source/reader/SourceReaderBase.java:177
  │   recordsWithSplitId = getNextFetch(output)  │  :181 非阻塞地取一批
  │   recordEmitter.emitRecord(record, output,    │  :202 发出记录，同时更新 split 状态
  │                            splitState)       │
  │ isAvailable() = elementsQueue.getAvailabilityFuture()   :341-344 队列空时让 Mailbox 挂起等待
  └─────────────────────────────────────────────┘
```

- 队列容量：`SourceReaderOptions.ELEMENT_QUEUE_CAPACITY`，默认 **2**（`BASE/source/reader/SourceReaderOptions.java:36-39`）。队列中的元素是 `RecordsWithSplitIds`，也就是**一批**记录，不是一条。
- **背压传递到外部系统**：下游慢 → Task 线程 `pollNext` 变慢 → 队列满 → fetcher 线程阻塞在 `put` 上 → 不再调用 `fetch()` → Kafka 的消费 lag 开始增长。
- **split 状态由 Task 线程维护**：`emitRecord` 在发出记录的同时更新 `splitState`（比如 Kafka 的 offset），`snapshotState` 把 `splitStates` 转回 split 对象（`:348-350`）。fetcher 线程只管读，不碰状态，所以 Checkpoint 时不需要和 fetcher 线程同步。代价是：队列里还没发出的记录不在状态中，恢复后会从状态中记录的 offset 重新读取，这正是我们想要的效果。
- `SingleThreadMultiplexSourceReaderBase`：一个 fetcher 线程读所有 split（Kafka 就是这样）；`SplitFetcherManager` 也支持每个 split 一个线程（线程池见 `SplitFetcherManager.java:157`）。

---

## 4. Sink V2：Writer + Committer

### 4.1 API：按能力组合接口

`CORE/api/connector/sink2/` 中的接口，一个 Sink 实现哪些接口，就说明它具备哪些能力：

| 接口 | 能力 | 关键方法 |
|---|---|---|
| `Sink` | 最基础：只有 Writer | `createWriter(WriterInitContext)`（`Sink.java:47`） |
| `SinkWriter` | 写数据 | `write`（`SinkWriter.java:41`）、`flush(endOfInput)`（`:47`）、`writeWatermark`（`:57`） |
| `SupportsWriterState` + `StatefulSinkWriter` | Writer 有自己的状态（比如文件写到一半） | `restoreWriter`（`SupportsWriterState.java:49`）、`snapshotState`（`StatefulSinkWriter.java:38`） |
| `SupportsCommitter` + `CommittingSinkWriter` | **两阶段提交** | `prepareCommit()`（`CommittingSinkWriter.java:38`）、`createCommitter`（`SupportsCommitter.java:51`）、`getCommittableSerializer`（`:54`） |
| `Committer` | 提交 | `commit(Collection<CommitRequest>)`（`Committer.java:46`） |
| `SupportsPreWriteTopology` / `SupportsPreCommitTopology` / `SupportsPostCommitTopology` | 在 Writer 前、Committer 前、Committer 后插入自定义拓扑 | 位于 `RT/streaming/api/connector/sink2/` |

**Committable** 是 Writer 交给 Committer 的"待提交凭证"：Kafka 中是事务 ID 加 producer 信息，FileSink 中是"已写完、待改名的文件"（`FileSinkCommittable`）。它本身必须可以序列化，因为它会进入 Checkpoint。

### 4.2 展开成算子：`SinkTransformationTranslator`

`sinkTo(sink)` 只生成一个 `SinkTransformation`，翻译成 StreamGraph 时才由 `SinkExpander.expand()`（`RT/streaming/runtime/translators/SinkTransformationTranslator.java:139`）按实现的接口展开：

```
[PreWrite 拓扑] → "Writer"（SinkWriterOperator） → [PreCommit 拓扑] → "Committer"（CommitterOperator） → [PostCommit 拓扑]
                   addWriter  :310-324                                    addCommittingTopology :260-306
```

算子名就是常量 `ConfigConstants.WRITER_NAME = "Writer"` 和 `COMMITTER_NAME = "Committer"`（`CORE/configuration/ConfigConstants.java:121-123`）。没有实现 `SupportsCommitter` 的 Sink 只有 Writer。默认情况下 Committer 与 Writer 并行度相同、forward 连接，所以它们会 **chain 在一起**。示例中整条链的名字是 `Source: range-source -> Map -> Sink: Writer -> Sink: Committer`。

### 4.3 第一阶段：`prepareCommit`，发生在 Barrier 之前

`RT/streaming/runtime/operators/sink/SinkWriterOperator.java`

```java
public void prepareSnapshotPreBarrier(long checkpointId) {                // :188
    super.prepareSnapshotPreBarrier(checkpointId);
    if (!endOfInput) {
        sinkWriter.flush(false);
        emitCommittables(checkpointId);                                   // :192
    }
}

private void emitCommittables(long checkpointId) {                        // :215
    Collection<CommT> committables = ((CommittingSinkWriter) sinkWriter).prepareCommit();   // :226
    emit(indexOfThisSubtask, numberOfParallelSubtasks, checkpointId, committables);
}
private void emit(...) {                                                  // :251
    output.collect(new CommittableSummary<>(subtask, parallelism, checkpointId, n, ...));   // 先发一个摘要："这个 Checkpoint 我有 n 个"
    for (CommT c : committables) {
        output.collect(new CommittableWithLineage<>(c, checkpointId, subtask));             // 再逐个发出
    }
}
```

`prepareSnapshotPreBarrier` 是 02 篇 `checkpointState` 六个步骤中的第 1 步，**先于 Barrier 向下游广播**执行。因此 Checkpoint N 的 committable 一定在 Barrier N **之前**到达 Committer，并进入 Committer 在 Checkpoint N 中的状态。示例日志中也能看到：同一个 subtask 的 `WRITER prepareCommit` 总是打印在 `READER snapshotState` 之前。

### 4.4 第二阶段：`commit`，发生在 Checkpoint 完成之后

`RT/streaming/runtime/operators/sink/CommitterOperator.java`

```java
public void processElement(StreamRecord<CommittableMessage<CommT>> element) {    // :210
    committableCollector.addMessage(element.getValue());        // 按 checkpointId 归类暂存
}
public void snapshotState(StateSnapshotContext context) {                        // :144
    committableCollectorState.update(Collections.singletonList(committableCollector.copy()));   // 未提交的全部进状态
}
public void notifyCheckpointComplete(long checkpointId) {                        // :159
    commitAndEmitCheckpoints(Math.max(lastCompletedCheckpointId, checkpointId));
}
private void commitAndEmitCheckpoints(long checkpointId) {                       // :164
    for (manager : committableCollector.getCheckpointCommittablesUpTo(checkpointId)) {   // ≤ checkpointId 的都提交
        commitAndEmit(manager);                                  // → committer.commit(...)
        committableCollector.remove(manager);
    }
}
```

**只有 Checkpoint N 全局完成后，才提交 N 及之前的 committable。** 在此之前，外部系统中的数据处于"已写入、未提交"的状态：Kafka 的 `read_committed` 消费者看不到它们，FileSink 的文件还叫 `.inprogress` 或 pending。

### 4.5 为什么 Committer 必须幂等

`CommitterOperator.initializeState()`：

```java
if (checkpointId.isPresent()) {                                                  // :135 从 Checkpoint 恢复
    committableCollectorState.get().forEach(cc -> committableCollector.merge(cc));
    lastCompletedCheckpointId = checkpointId.getAsLong();
    // try to re-commit recovered transactions as quickly as possible
    commitAndEmitCheckpoints(lastCompletedCheckpointId);                         // :139
}
```

从 Checkpoint N 恢复时，状态中保存的是"N 的快照时刻还没提交"的 committable，其中就包括 N 自己的。但是 `notifyCheckpointComplete(N)` 很可能在故障之前**已经执行过了**，Flink 无法知道外部系统中是否真的提交成功，所以恢复后**一律再提交一次**。

结论：
- `commit` 必须能识别"这个事务已经提交过"，然后跳过，并调用 `request.signalAlreadyCommitted()`。KafkaSink、FileSink 都是这样实现的。FileSink 调用的是 `commitAfterRecovery()`（`flink-connectors/flink-connector-files/.../sink/committer/FileCommitter.java:61-62`），它会检查目标文件是否已经存在。
- 事务 ID 必须**在重启前后都唯一**，通常由"前缀 + subtask + checkpointId"拼成。否则重启后新事务和旧事务同名，幂等检查反而会把新数据当成"已提交"跳过，造成**数据丢失**。

第 6.3 节会用一个"不幂等"的 Committer 实际演示重复提交。

### 4.6 重试：`retryLater()` 并不会"稍后"重试

`CommitRequest` 提供了 `retryLater()`、`signalFailedWithKnownReason()` 等方法（`Committer.java` 中的 `CommitRequest` 接口）。重试的实现在 `CheckpointCommittableManagerImpl.commit()`（`RT/streaming/runtime/operators/sink/committables/CheckpointCommittableManagerImpl.java:145-160`）：

```java
for (int retry = 0; !requests.isEmpty() && retry <= maxRetries; retry++) {
    committer.commit(Collections.unmodifiableCollection(requests));
    requests = requests.stream().filter(r -> !r.isFinished()).collect(Collectors.toList());
}
if (!requests.isEmpty()) {
    throw new IOException("Failed to commit %s committables after %s retries: %s");
}
```

它会在 `notifyCheckpointComplete` 内**同步地、连续地**重试，中间**没有任何退避**。最多重试 `sink.committer.retries` 次，默认 **10** 次（`CORE/configuration/SinkOptions.java:35-37`），之后抛出异常，作业失败重启。

这是 FLINK-36455（`bc0f241b867`）有意做的改动。提交说明中的理由是：`notifyCheckpointComplete` 的约定要求 RPC 返回时所有事务都已经提交，所以不能再像以前那样异步重试。对 Connector 开发者来说，这意味着：**如果外部系统需要等待才能恢复，应该在 `commit()` 内部自己退避，或者直接让作业失败。**

### 4.7 结束时的提交与 Final Checkpoint

```java
// SinkWriterOperator
public void endInput() {                                        // :206
    endOfInput = true;
    sinkWriter.flush(true);
    emitCommittables(lastKnownCheckpointId + 1);                // :211 最后一批数据，归入"下一个"Checkpoint
}
// CommitterOperator
public void endInput() {                                        // :151
    if (!isCheckpointingEnabled || isBatchMode) {
        commitAndEmitCheckpoints(Long.MAX_VALUE);               // :154 没有 Checkpoint 可等，直接全部提交
    }
}
```

- **开了 Checkpoint 的流作业**：有界输入结束后，最后一批 committable 要等 **Final Checkpoint** 完成后才提交。`execution.checkpointing.checkpoints-after-tasks-finish` 默认是 true（`CheckpointingOptions.java:614-617`，FLIP-147）。
- **没开 Checkpoint**：只在 `endInput` 时提交一次。**无界的流作业如果不开 Checkpoint，事务型 Sink 永远不会提交。** 第 6.4 节的实测可以看到，整个作业只在结束时提交了一次。
- **批作业**：同样在 `endInput` 时提交。

---

## 5. 真实 Connector 中的对应关系

| 概念 | FileSource / FileSink（本仓库） | Kafka（独立仓库 `apache/flink-connector-kafka`） |
|---|---|---|
| Split | 文件的一段 `FileSourceSplit`（路径 + offset） | 一个 TopicPartition + 起始 offset |
| 分配方式 | 拉：读完一段再要下一段 | 推：发现分区后直接分配 |
| Reader 线程 | `SourceReaderBase` + fetcher 线程 | `SingleThreadMultiplexSourceReaderBase`，一个 KafkaConsumer |
| Committable | 已关闭、待改名的文件 | 事务 ID + producer 信息 |
| prepareCommit | 按滚动策略关闭 part 文件（`FileWriterBucket.java:196-201`） | `flush` 后返回当前事务，并开启下一个事务 |
| commit | 改名：pending → 正式文件（`FileCommitter.java:56-66`） | `commitTransaction()` |
| 幂等 | `commitAfterRecovery()` 检查文件是否已存在 | 用相同事务 ID 恢复 producer，已提交的事务再次提交是 no-op |

> **补充（2026-10-02）**：Row 格式的 FileSink 默认使用 `DefaultRollingPolicy`，它在 Checkpoint 时只有文件超过 128MB 才滚动（`flink-file-sink-common/.../rollingpolicies/DefaultRollingPolicy.java:72-74`），按处理时间滚动要等 60 秒，检查周期也是 60 秒（`FileSink.java:297`）。`demos/.../connector/FileSinkLatencyDemo.java` 实测：每秒 10 条、Checkpoint 2 秒，第一个正式文件在第 120 秒才出现；换成 `OnCheckpointRollingPolicy` 后最多 2.1 秒。详见 `content-plan/22-第8讲下-公众号定稿.md` 第二节。

从源码仓库的角度看，**Flink 主仓库中的 connector 只剩 files、datagen 和 base**（`ls flink-connectors`）。Kafka、JDBC、Elasticsearch 等都在独立仓库中，有各自的发版节奏。07 篇说过，这些仓库的 committer 更少、review 更慢，但也更容易找到还没人做的事。

---

## 6. 动手实验

示例：`SourceSinkDemo.java`，大约 400 行，**包含一个完整的 FLIP-27 Source 和一个两阶段提交的 Sink**，没有依赖任何 connector 基础类。
- Source：4 个 split，每个 40 条记录（split-0 是 `[0,40)`，依此类推），Reader 每 80ms 读一条，按"拉"模式请求 split。
- Sink：Writer 把记录攒在一个"事务"中，`prepareCommit` 时交出。事务 ID 为 `s{subtask}-chk{n}`，其中 n 从"恢复的 Checkpoint + 1"开始递增。Committer 做幂等检查，并在一个全局 Map 中记下每条记录被提交的次数，用来校验 exactly-once。
- 并行度 2，Checkpoint 间隔 1s，整条链全部 forward，所以每个并行度是一个独立的 pipelined region。

### 6.1 正常运行（`normal`）

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.connector.SourceSinkDemo normal
```

本机实测（线程名截断为 40 个字符，中间省略了部分行）：

```
[+ 1.37s] ENUM     SourceCoordinator-Source: range-source   start, pending=[split-0[0,40), split-1[40,80), split-2[80,120), split-3[120,160)]
[+ 1.52s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 0 createWriter (restoredCheckpointId=OptionalLong.empty)
[+ 1.52s] READER   Source: range-source -> Map -> Sink: Wri subtask 0 start -> sendSplitRequest
[+ 1.52s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 start -> sendSplitRequest
[+ 1.52s] ENUM     SourceCoordinator-Source: range-source   addReader(subtask 1)
[+ 1.52s] ENUM     SourceCoordinator-Source: range-source   addReader(subtask 0)
[+ 1.52s] ENUM     SourceCoordinator-Source: range-source   subtask 0 requests a split -> assign split-0[0,40)
[+ 1.53s] ENUM     SourceCoordinator-Source: range-source   subtask 1 requests a split -> assign split-1[40,80)
[+ 1.53s] READER   Source: range-source -> Map -> Sink: Wri subtask 0 addSplits[split-0[0,40)]
[+ 1.53s] ENUM     SourceCoordinator-Source: range-source   snapshotState(chk 1), pending=[split-2[80,120), split-3[120,160)]
...
[+ 2.54s] ENUM     SourceCoordinator-Source: range-source   snapshotState(chk 3), pending=[split-2[80,120), split-3[120,160)]
[+ 2.59s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 0 prepareCommit -> s0-chk3 n=10 [2..11]
[+ 2.59s] READER   Source: range-source -> Map -> Sink: Wri subtask 0 snapshotState(chk 3) = [split-0[12,40)]
[+ 2.68s] COMMIT   Source: range-source -> Map -> Sink: Wri s0-chk3 n=10 [2..11] (retries=0)
...
[+ 5.09s] READER   Source: range-source -> Map -> Sink: Wri subtask 0 finished split-0 -> sendSplitRequest
[+ 5.09s] ENUM     SourceCoordinator-Source: range-source   subtask 0 requests a split -> assign split-3[120,160)
...
[+ 8.58s] ENUM     SourceCoordinator-Source: range-source   subtask 1 requests a split -> signalNoMoreSplits
[+ 8.66s] ENUM     SourceCoordinator-Source: range-source   subtask 0 requests a split -> signalNoMoreSplits
[+ 8.68s] MAIN     main                                     committed records: total=160 distinct=160 expected=160 duplicates=0
```

对照源码：
- **线程**：`ENUM` 行全部在 `SourceCoordinator-...` 线程（2.1 节）；`READER`、`WRITER`、`COMMIT` 全部在同一个 Task 线程（3.1 节、4.2 节）。
- **拉模式**：Reader 先注册（`addReader`），再请求 split。一个 split 读完才请求下一个。
- **顺序**：每个 Checkpoint 中，`WRITER prepareCommit` 都在 `READER snapshotState` **之前**（4.3 节），`COMMIT` 在大约 90ms 之后，也就是 Checkpoint 完成、`notifyCheckpointComplete` 到达的时候（4.4 节）。
- **状态中的进度**：`snapshotState(chk 3) = [split-0[12,40)]` 表示下一条要读的是 12；而 `prepareCommit` 交出的事务 `s0-chk3` 正好是 `[2..11]`。**Source 的进度和 Sink 的事务在同一个 Checkpoint 中严丝合缝地对上了**，这就是端到端 exactly-once 的全部奥秘。
- **结束**：split 分完后返回 `signalNoMoreSplits`；Reader 返回 `END_OF_INPUT` 后触发 Final Checkpoint，提交最后一批，共 160 条，无重复。

### 6.2 故障恢复（`fail`）

在 split-2 的第一条记录（80）上抛一次异常。此时 split-2 **刚刚分配下去，还没有进入任何 Checkpoint**。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.connector.SourceSinkDemo fail
```

本机实测（截取故障前后）：

```
[+ 4.59s] ENUM     SourceCoordinator-Source: range-source   snapshotState(chk 5), pending=[split-2[80,120), split-3[120,160)]
[+ 4.60s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 1 prepareCommit -> s1-chk5 n=11 [63..73]
[+ 4.60s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 snapshotState(chk 5) = [split-1[74,80)]
[+ 4.69s] COMMIT   Source: range-source -> Map -> Sink: Wri s1-chk5 n=11 [63..73] (retries=0)
[+ 5.14s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 finished split-1 -> sendSplitRequest
[+ 5.14s] ENUM     SourceCoordinator-Source: range-source   subtask 1 requests a split -> assign split-2[80,120)
[+ 5.14s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 addSplits[split-2[80,120)]
[+ 5.14s] ENUM     SourceCoordinator-Source: range-source   subtask 0 requests a split -> assign split-3[120,160)
[+ 5.23s] MAP      Source: range-source -> Map -> Sink: Wri !!! injected failure at record 80
[+ 6.26s] ENUM     SourceCoordinator-Source: range-source   addSplitsBack([split-2[80,120)]) from failed subtask 1
[+ 6.27s] COMMIT   Source: range-source -> Map -> Sink: Wri s1-chk5 n=11 [63..73] already committed -> signalAlreadyCommitted
[+ 6.27s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 1 createWriter (restoredCheckpointId=OptionalLong[5])
[+ 6.27s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 addSplits[split-1[74,80)]
[+ 6.27s] ENUM     SourceCoordinator-Source: range-source   addReader(subtask 1)
[+ 6.27s] ENUM     SourceCoordinator-Source: range-source   snapshotState(chk 6), pending=[split-2[80,120)]
[+ 6.29s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 0 prepareCommit -> s0-chk6 n=19 [34..132]
[+ 6.35s] WRITER   Source: range-source -> Map -> Sink: Wri subtask 1 prepareCommit -> s1-chk6 n=1 [74..74]
[+ 6.79s] READER   Source: range-source -> Map -> Sink: Wri subtask 1 finished split-1 -> sendSplitRequest
[+ 6.79s] ENUM     SourceCoordinator-Source: range-source   subtask 1 requests a split -> assign split-2[80,120)
...
[+10.36s] MAIN     main                                     committed records: total=160 distinct=160 expected=160 duplicates=0
```

JobMaster 日志：`1 tasks will be restarted to recover the failed task ...`，`Restoring job ... from Checkpoint 5`。

逐条对照：

| 现象 | 源码 |
|---|---|
| 只重启了 1 个 Task，subtask 0 继续读 split-3（`s0-chk6` 中包含 120 以后的记录） | region failover（06 篇），两个并行度是两个独立的 region |
| **`addSplitsBack([split-2[80,120)])`** | split-2 是 chk 5 之后分配的，`getAndRemoveUncheckpointedAssignment(1, 5)` 把它找了回来（2.3 节） |
| 恢复后 Reader 收到的是 **`split-1[74,80)`**，而不是 split-2 | split-1 在 chk 5 时属于 Reader 的状态（`snapshotState(chk 5) = [split-1[74,80)]`），由 `SourceOperator.open()` 直接交还给 Reader（`SourceOperator.java:451`） |
| 记录 74~79 被重新读了一遍 | 它们在故障前已经写进了 Writer 的事务，但这个事务还没 `prepareCommit`，故障后随 Writer 一起丢弃。重读是正确的 |
| **`s1-chk5 ... already committed`** | `CommitterOperator.initializeState` 把 chk 5 的 committable 又提交了一次（`CommitterOperator.java:139`，4.5 节），幂等检查挡住了它 |
| 重启后第一个事务叫 `s1-chk6` | Writer 的事务号从 `restoredCheckpointId + 1 = 6` 开始 |
| 最终 160 条，无重复 | 端到端 exactly-once |

split-2 有可能被分配给另一个 subtask，这取决于谁先请求。我连续跑了两次，第二次故障发生在 subtask 0 上，同样是 `addSplitsBack([split-2[80,120)]) from failed subtask 0`，结果也是 `duplicates=0`。

### 6.3 反例：不幂等的 Committer（`fail-naive`）

把幂等检查去掉，事务号每次从 0 开始（这就是我第一次写示例时的写法）：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.connector.SourceSinkDemo fail-naive
```

本机实测：

```
[+ 1.51s] WRITER   ... subtask 1 prepareCommit -> s1-txn0 n=1 [40..40]
[+ 1.59s] COMMIT   ... s1-txn0 n=1 [40..40] (retries=0)
...
[+ 4.51s] WRITER   ... subtask 1 prepareCommit -> s1-txn4 n=12 [63..74]
[+ 4.60s] COMMIT   ... s1-txn4 n=12 [63..74] (retries=0)
[+ 5.05s] MAP      ... !!! injected failure at record 80
[+ 6.09s] ENUM     ... addSplitsBack([split-2[80,120)]) from failed subtask 1
[+ 6.10s] COMMIT   ... s1-txn4 n=12 [63..74] (retries=0)          ← 同一个事务第二次提交
[+ 6.10s] WRITER   ... subtask 1 createWriter (restoredCheckpointId=OptionalLong[5])
[+ 6.10s] READER   ... subtask 1 addSplits[split-1[75,80)]
[+ 6.19s] WRITER   ... subtask 1 prepareCommit -> s1-txn0 n=1 [75..75]      ← 与 1.51s 的 s1-txn0 重名
...
[+10.09s] WRITER   ... subtask 1 prepareCommit -> s1-txn4 n=10 [110..119]   ← 与 4.51s 的 s1-txn4 重名
[+10.11s] MAIN     main   committed records: total=172 distinct=160 expected=160 duplicates=12
```

- `s1-txn4` 被提交了两次，所以**重复了 12 条**，也就是这个事务的大小。这正是 4.5 节 `:139` 的"恢复后再提交"。
- 恢复后事务号又从 0 开始，`s1-txn0` 和 `s1-txn4` 都与故障前的事务**重名**。这次没有造成数据丢失，只是因为 naive 模式根本不做幂等检查；如果加上"按 ID 去重"，这两批新数据反而会被当成已提交而**丢掉**（4.5 节的第二个结论）。
- 我第一次写示例时，故障点放在记录 60，也就是 split-1 读到一半的位置。当时 `addSplitsBack([])` 是**空的**，因为 split-1 已经在上一个 Checkpoint 的 Reader 状态里，不需要还给 Enumerator。重复条数则是 10 条（那一次被重复提交的事务有 10 条）。

### 6.4 不开 Checkpoint（`nockpt`）

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.connector.SourceSinkDemo nockpt
```

本机实测：

```
[+ 1.54s] WRITER   ... subtask 0 createWriter (restoredCheckpointId=OptionalLong.empty)
[+ 8.64s] WRITER   ... subtask 0 prepareCommit -> s0-chk1 n=80 [40..119]
[+ 8.64s] WRITER   ... subtask 1 prepareCommit -> s1-chk1 n=80 [0..159]
[+ 8.64s] COMMIT   ... s0-chk1 n=80 [40..119] (retries=0)
[+ 8.64s] COMMIT   ... s1-chk1 n=80 [0..159] (retries=0)
[+ 8.66s] MAIN     main   committed records: total=160 distinct=160 expected=160 duplicates=0
```

作业运行了 7 秒，这期间**一次提交都没有**。直到输入结束，`SinkWriterOperator.endInput` 用 `lastKnownCheckpointId + 1 = 1` 交出全部数据，`CommitterOperator.endInput` 才立即提交（4.7 节）。如果这是一个无界作业，数据会永远停留在"未提交"状态。

### 6.5 读社区的测试

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && ./mvnw -s ../maven-settings-aliyun.xml -B -q -pl flink-runtime -Dfast -Dtest=SplitAssignmentTrackerTest -Dsurefire.failIfNoSpecifiedTests=false test
```

本机实测：6 个测试全部通过，耗时约 12 秒。推荐先读 `testGetAndRemoveSplitsAfterSomeCheckpoint`（`flink-runtime/src/test/.../source/coordinator/SplitAssignmentTrackerTest.java:158`），它就是 6.2 节的最小复现。

其他值得读的测试：
- `SourceCoordinatorTest`（同一目录）：不启动集群，直接驱动 Coordinator 的各个回调。
- `flink-connectors/flink-connector-base/src/test/.../SourceReaderBaseTest.java`：fetcher 线程与队列。
- `flink-streaming-java/src/test/.../operators/sink/SinkV2CommitterOperatorTest.java`、`SinkV2SinkWriterOperatorTest.java`：和 08 篇的窗口测试一样，**算子的代码在 `flink-runtime`，测试却在 `flink-streaming-java`**。

---

## 7. 断点清单

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `SourceCoordinator.java:325` | 所有来自 Reader 的事件 |
| 2 | `SourceCoordinatorContext.java:305` | 每一次 split 分配被记录 |
| 3 | `SplitAssignmentTracker.java:63` | Coordinator 快照时，"未归档的分配"归入 Checkpoint |
| 4 | `SourceCoordinator.java:389` | 故障后 split 被还给 Enumerator |
| 5 | `OperatorCoordinatorHolder.java:319` | 关门 |
| 6 | `OperatorCoordinatorHolder.java:217` | 收到 Task 的确认，开门 |
| 7 | `SourceOperator.java:451` | 恢复出来的 split 交还给 Reader |
| 8 | `SourceOperator.java:743` | 收到新 split |
| 9 | `FetchTask.java:64` | fetcher 线程往队列里放数据，队列满时阻塞在这里（看背压） |
| 10 | `SinkWriterOperator.java:188` | 第一阶段：prepareCommit |
| 11 | `CommitterOperator.java:159` | 第二阶段：commit |
| 12 | `CommitterOperator.java:139` | 恢复后的再提交 |
| 13 | `CheckpointCommittableManagerImpl.java:149` | 提交重试循环 |

建议在 IDE 中调试示例的 `fail` 模式，在断点 4 和断点 12 处停下，对照 6.2 节的表格。

---

## 8. 课后练习

1. **推导题**：6.2 节中，如果故障发生在 subtask 1 读完 split-2 的前 10 条、并且 chk 6 已经完成之后，`addSplitsBack` 会收到什么？Reader 恢复时会收到什么？修改 `MaybeFail` 中的记录号验证你的推导。
2. **push 模式**：把 `RangeEnumerator` 改成在 `addReader` 中直接用 `assignSplit` 把所有 split 轮流分给 Reader，Reader 不再调用 `sendSplitRequest`。故障恢复的行为有什么不同？
3. **扩缩容**：开启 `KEEP_CKPT=1`，用并行度 2 跑出一个 Checkpoint，再用并行度 3 恢复（03 篇的方法）。`"SourceReaderState"` 这个 operator state 是怎么在 3 个 Reader 之间分配的？如果 split 数少于并行度会怎样？
4. **读源码**：`FileWriterBucket.java:196-201` 中，行式格式（`forRowFormat`）和批量格式（`forBulkFormat`，如 Parquet）的滚动策略有什么不同？为什么批量格式**必须**在 Checkpoint 时滚动文件？
5. **读 FLIP**：读 FLIP-27 和 FLIP-191（Sink V2 拓扑扩展），对照本篇 4.1 节，说说 `SupportsPreCommitTopology` 解决了什么问题。提示：小文件合并。
6. **进阶**：`SupportsSplitReassignmentOnRecovery`（FLIP-537，3.1 节）在 2.3 中是新加入的。读 `git show 7b0a8827ffc --stat`，看看改了哪些类。如果你是 Kafka connector 的维护者，接入这个接口要做哪些事？

---

## 9. 踩坑记录

| 现象 | 原因 | 解决 |
|---|---|---|
| 示例第一版的 `fail` 模式出现 `duplicates=10`（`fail-naive` 模式复现为 12） | Committer 不幂等，恢复后重复提交了上一个 Checkpoint 的事务（6.3 节） | 事务 ID 带上 checkpointId，并在提交前检查是否已经提交过 |
| 第一次注入故障时，`addSplitsBack([])` 是空的 | 故障点所在的 split 已经进了 Reader 的状态 | 把故障点放到"刚分配、还没 Checkpoint"的 split 上（记录 80） |
| 流作业用了 Kafka exactly-once Sink，下游一直读不到数据 | 没开 Checkpoint，或者 Checkpoint 一直失败，事务永远不提交（6.4 节） | 开启 Checkpoint；检查 Checkpoint 是否成功 |
| exactly-once Sink 下游数据有"Checkpoint 间隔"那么大的延迟 | 两阶段提交的必然结果：数据在 Checkpoint 完成后才可见 | 缩短 Checkpoint 间隔，或者接受 at-least-once |
| Committer 遇到外部系统短暂不可用，作业很快就失败了 | `retryLater()` 是同步的立即重试，最多 10 次，没有退避（4.6 节） | 在 `commit()` 内部自己实现退避，或调大 `sink.committer.retries` |
| 在 `pollNext` 中阻塞读取，Checkpoint 超时 | `pollNext` 在 Mailbox 线程中执行（3.2 节） | 使用 `SourceReaderBase` + `SplitReader`，把阻塞调用放进 fetcher 线程 |
| 在 Enumerator 中自己起线程调用 `context.assignSplits`，出现并发问题 | Enumerator 应该只在 Coordinator 线程中运行 | 用 `context.callAsync(...)` 做异步发现，结果回调会回到 Coordinator 线程 |

---

## 10. 小结

- **Source = Enumerator（JM，单线程）+ Reader（TM，Mailbox 线程）**，两者通过 OperatorEvent 通信。split 对象本身就是读取进度。
- **split 不丢不重**：每个 split 在任意一个 Checkpoint 中，要么在 Enumerator 的状态里，要么在某个 Reader 的状态里。`SplitAssignmentTracker` 负责把"Checkpoint 之后才分配的"找回来还给 Enumerator；网关的关门与开门保证分配事件不会落进 Barrier 前后的缝隙里。
- **SourceReaderBase**：阻塞读取放在 fetcher 线程中，通过一个容量为 2 批的有界队列与 Task 线程解耦，背压会一直传递到外部系统。
- **Sink V2 的两阶段提交**：`prepareCommit` 在 Barrier **之前**执行，`commit` 在 Checkpoint **完成之后**执行，committable 在两者之间存放在 Committer 的状态中。
- **Committer 必须幂等**，事务 ID 在重启前后必须唯一。不开 Checkpoint 的无界作业永远不会提交。

至此，学习路线阶段 2、阶段 3 列出的所有子系统都已经讲完：执行链路（01）、Checkpoint（02）、State（03）、网络（04）、SQL（05）、调度（06）、时间与窗口（08）、Source/Sink（09），再加上贡献流程（07）。接下来建议回到 07 篇第 6 节，从候选任务中挑一个动手。本篇 4.6 节"`retryLater()` 的 Javadoc 没有说明它其实是立即重试"也可以作为一个候选：先在 master 上确认代码与文档现状，再上 JIRA 搜索是否已经有人讨论过。
:::
