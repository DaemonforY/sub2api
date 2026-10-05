---
title: "Flink 2.x 源码精读（八·上）：Source，分片是怎么分下去、又怎么收回来的"
description: "split-2 在 Checkpoint 5 之后才分给 subtask 1，subtask 1 读了第一条就挂了。恢复后，split-2 被还给了 Enumerator，160 条记录一条不丢、一条不重。顺着这个现象，读懂 Flink 2.3 新 Source 架构：Enumerator 与 Reader、split 追踪、Coordinator 的\"关门\"机制，以及 fetcher 线程模型。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-08a-cover.webp"}]]
---

# Flink 2.x 源码精读（八·上）：Source，分片是怎么分下去、又怎么收回来的

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-08a-cover.webp" alt="Yui与Kai协作展示分片分配、检查点与故障恢复" width="1200" height="800" loading="eager" /><figcaption>Yui与Kai协作展示分片分配、检查点与故障恢复<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`CORE` = `flink-core/src/main/java/org/apache/flink`，`RT` = `flink-runtime/src/main/java/org/apache/flink`，`BASE` = `flink-connectors/flink-connector-base/src/main/java/org/apache/flink/connector/base`
> 前置阅读：第二讲上篇（Checkpoint 的触发顺序）、第六讲（region failover、`SourceCoordinator.subtaskReset`）、第七讲上篇（Split 级别的 Watermark）

---

先看一个实验。

一个 Source，4 个 split（分片），每个 40 条记录：split-0 是 `[0,40)`，split-1 是 `[40,80)`，依此类推。并行度 2，每秒一次 Checkpoint。Reader 读完一个 split，再向 Enumerator 要下一个。

我在记录 80（split-2 的第一条）上注入一次故障。本机实测（节选，完整日志在 `assets/logs/sourcesink-fail.log`）：

```text
[+ 4.57s] ENUM     ...  snapshotState(chk 5), pending=[split-2[80,120), split-3[120,160)]
[+ 4.60s] READER   ...  subtask 1 snapshotState(chk 5) = [split-1[76,80)]
[+ 4.93s] READER   ...  subtask 1 finished split-1 -> sendSplitRequest
[+ 4.93s] ENUM     ...  subtask 1 requests a split -> assign split-2[80,120)
[+ 4.93s] READER   ...  subtask 1 addSplits[split-2[80,120)]
[+ 5.01s] MAP      ...  !!! injected failure at record 80
[+ 6.05s] ENUM     ...  addSplitsBack([split-2[80,120)]) from failed subtask 1
[+ 6.07s] READER   ...  subtask 1 addSplits[split-1[76,80)]
...
[+ 6.40s] ENUM     ...  subtask 1 requests a split -> assign split-2[80,120)
...
[+ 9.78s] MAIN     ...  committed records: total=160 distinct=160 expected=160 duplicates=0
```

> 每行是示例打印的日志，`...` 处省略了线程名。`ENUM` 是 Enumerator，`READER` 是 Reader。

时间线是这样的：

1. Checkpoint 5 时，split-2 还在 Enumerator 手里（`pending=[split-2..., split-3...]`）；
2. **Checkpoint 5 之后**，split-2 才分给 subtask 1；
3. subtask 1 读到 split-2 的第一条就挂了，作业回滚到 Checkpoint 5。

问题是：**Checkpoint 5 里，Enumerator 的状态说 split-2 已经分出去了吗？subtask 1 的状态里有 split-2 吗？** 都没有。那 split-2 会不会丢？

结果是：恢复时，`addSplitsBack` 把 split-2 **还给了 Enumerator**，之后重新分配。最终 160 条记录，`duplicates=0`，一条不丢，一条不重。

这一篇，我们来看：

1. 新 Source 为什么要拆成 Enumerator 和 Reader？
2. split 是怎么被追踪的？为什么故障时能被找回来？
3. Checkpoint 时，Enumerator 和 Reader 的状态是怎么"对齐"的？
4. 真实的连接器里，Reader 怎么用一个阻塞的客户端读数据？

Sink 和端到端的 exactly-once 放到下篇。

---

## 一、三个角色

`CORE/api/connector/source/` 下的几个接口：

| 接口 | 运行在哪 | 核心方法 |
|---|---|---|
| `Source` | 工厂，两边都用 | `createEnumerator`（`Source.java:55`）、`restoreEnumerator`（`:68`）、`createReader`、两个序列化器 |
| `SplitEnumerator` | **JobManager**，每个 Source 一个 | `handleSplitRequest`（`SplitEnumerator.java:52`）、`addSplitsBack`（`:61`）、`addReader`（`:68`）、`snapshotState`（`:90`） |
| `SourceReader` | **TaskManager**，每个并行度一个 | `pollNext`（`SourceReader.java:73`）、`snapshotState`（`:80`）、`addSplits`（`:111`）、`notifyNoMoreSplits`（`:120`） |
| `SourceSplit` | 两边都有 | 只有一个 `splitId()`；**split 对象本身就是"读到哪了"的状态** |

**为什么要拆开？** 老的 `SourceFunction` 把"发现分区"和"读数据"写在同一个 `run()` 循环里，每个并行度各自发现分区、各自决定读哪些。动态发现新分区、批流统一、Watermark 对齐，都很难做。

FLIP-27 把**发现和分配**集中到 JobManager 上的 Enumerator，Reader 只读分给它的 split。

分配有两种方式（都在 `SplitEnumeratorContext` 上）：

- **推**：Enumerator 主动调用 `assignSplits(...)`（`SplitEnumeratorContext.java:107`），比如发现新分区后直接分配；
- **拉**：Reader 调用 `sendSplitRequest()`（`SourceReaderContext.java:55`），Enumerator 在 `handleSplitRequest` 里响应。读得快的 Reader 会拿到更多 split，天然能做负载均衡。

本篇的示例用的是"拉"。



---

## 二、JobManager 侧：SourceCoordinator

### 2.1 单线程

Enumerator 跑在 `SourceCoordinator` 里。线程名是 `"SourceCoordinator-" + operatorName`（`RT/runtime/source/coordinator/SourceCoordinatorProvider.java:78`），线程池只有 1 个线程（`SourceCoordinatorContext.java:133`，`Executors.newScheduledThreadPool(1, ...)`）。

**Enumerator 的所有方法都在这一个线程里执行**，所以实现 Enumerator 时不需要加锁。开头日志里所有 `ENUM` 行，线程名都是 `SourceCoordinator-Source: range-source`。

Reader 发来的所有事件都走同一个入口 `handleEventFromOperator()`（`RT/runtime/source/coordinator/SourceCoordinator.java:325`），放到这个线程里处理：请求 split 交给 `enumerator.handleSplitRequest`（`:660`），Reader 注册交给 `enumerator.addReader`（`:702`），第七讲上篇的 Watermark 对齐上报也走这里。

### 2.2 分配时记一笔账

Enumerator 调用 `context.assignSplits(...)` 时（`SourceCoordinatorContext.java:288`）：

1. 检查目标 subtask 已经注册过；
2. **`assignmentTracker.recordSplitAssignment(assignment)`**（`:305`）：记下"这些 split 是在上一个 Checkpoint 之后分出去的"；
3. 把 split 序列化成 `AddSplitEvent`（`:631`），通过 RPC 发给 Task。

第 2 步就是 split 不会丢的关键。

---

## 三、split 追踪：故障时怎么找回来

### 3.1 SplitAssignmentTracker

`RT/runtime/source/coordinator/SplitAssignmentTracker.java` 只有两个数据结构：

```java
SortedMap<Long, Map<Integer, LinkedHashSet<SplitT>>> assignmentsByCheckpointId;   // :47 已经归入某个 Checkpoint 的分配
Map<Integer, LinkedHashSet<SplitT>> uncheckpointedAssignments;                     // :50 上一个 Checkpoint 之后的新分配
```

三个时机：

| 时机 | 方法 | 做什么 |
|---|---|---|
| Coordinator 做快照 | `onCheckpoint`（`:63`，由 `SourceCoordinator.java:446` 调用） | 把 `uncheckpointedAssignments` 归到这个 Checkpoint 名下，再清空 |
| Checkpoint 完成 | `onCheckpointComplete`（`:96`） | 这个 Checkpoint 及之前的记录都可以扔了 |
| 某个 subtask 回滚到 Checkpoint N | `getAndRemoveUncheckpointedAssignment(subtaskId, N)`（`:120`） | 取出这个 subtask 在 N **之后**拿到的所有 split |

最后一个方法的实现（`:120-135`）：

```java
for (final Map.Entry<Long, Map<Integer, LinkedHashSet<SplitT>>> entry :
        assignmentsByCheckpointId.entrySet()) {
    if (entry.getKey() > restoredCheckpointId) {
        removeFromAssignment(subtaskId, entry.getValue(), splits);
    }
}
removeFromAssignment(subtaskId, uncheckpointedAssignments, splits);
return splits;
```

### 3.2 谁来调用

第六讲讲过，局部恢复时，调度器会通知 OperatorCoordinator "某个 subtask 被重置了"。对 Source 来说，最终到 `SourceCoordinator.subtaskReset()`（`SourceCoordinator.java:372`）：

```java
final List<SplitT> splitsToAddBack =
        context.getAndRemoveUncheckpointedAssignment(subtaskId, checkpointId);
...
enumerator.addSplitsBack(splitsToAddBack, subtaskId);                       // :389
```

对应开头日志里的 `addSplitsBack([split-2[80,120)]) from failed subtask 1`。

### 3.3 为什么只还"N 之后分配的"

因为 **N 之前分配的 split，已经在 Reader 的状态里了**。它们会随着 Reader 的状态一起恢复，由 Reader 自己接着读。

开头日志里也能看到：

```text
[+ 4.60s] READER   ...  subtask 1 snapshotState(chk 5) = [split-1[76,80)]
...
[+ 6.07s] READER   ...  subtask 1 addSplits[split-1[76,80)]
```

Checkpoint 5 时，subtask 1 手上是 `split-1[76,80)`（下一条要读 76）。恢复后，subtask 1 拿回的就是 `split-1[76,80)`，接着读完 76～79，再去要新的 split。这一步由 `SourceOperator.open()` 完成（`RT/streaming/api/operators/SourceOperator.java:443-451`）：从状态里读出 split，直接交给自己的 Reader。

于是有一个不变式：**每个 split 在任意一个 Checkpoint 里，要么在 Enumerator 的状态里，要么在某个 Reader 的状态里，而且只在一处。** N 之后才分出去的 split，在 Checkpoint N 里属于 Enumerator（`pending` 里还有它），所以恢复时要还给 Enumerator。



### 3.4 Flink 2.2 起的另一种做法

实现了 `SupportsSplitReassignmentOnRecovery` 的 Source（`CORE/api/connector/source/SupportsSplitReassignmentOnRecovery.java:29`），恢复时**不再**把 split 直接交还给原来的 Reader（`SourceOperator.java:445` 的判断），而是在注册时上报给 Enumerator（`:455`），由 Enumerator 统一重新分配。

这是 Flink 2.2 引入的（FLINK-38564，`docs/content/release-notes/flink-2.2.md:175-183`，标题是"Balanced splits assignment"）：以前 Enumerator 不知道 split 在运行时的实际分布，没法保证分配均匀；有了这个接口，Enumerator 能掌握全局的 split 分布。

> 本篇的示例没有实现这个接口，走的是默认路径。

---

## 四、Checkpoint 时：Coordinator 先做，并且"关门"

### 4.1 顺序

第二讲上篇讲过：`CheckpointCoordinator` 先触发所有 OperatorCoordinator 的快照（`RT/runtime/checkpoint/CheckpointCoordinator.java:702-703`），**全部完成后**，才向 Source Task 发送 Barrier（`:791`，`triggerTasks`）。

开头的日志也是这个顺序：`ENUM snapshotState(chk 5)` 在 4.57s，`READER snapshotState(chk 5)` 在 4.60s。

### 4.2 一个缝隙

但这里有一个竞争：

1. Coordinator 在 Checkpoint N 的快照里，已经把 split-X 记为"分配给 subtask 1"（不在 `pending` 里了）；
2. 对应的 `AddSplitEvent` 还在路上，**在 Barrier N 之后**才到达 subtask 1；
3. subtask 1 在 Checkpoint N 的状态里没有 split-X。

两边都没有 split-X。如果从 Checkpoint N 恢复，split-X 就丢了。

### 4.3 关门

`OperatorCoordinatorHolder`（`RT/runtime/operators/coordination/OperatorCoordinatorHolder.java`）用"关门"解决这个问题。类注释（`:67-86`）写得很清楚，实现在 `checkpointCoordinatorInternal()`（`:307`）：

```java
coordinatorCheckpoint.handleAsync((success, failure) -> {
    ...
    closeGateways(checkpointId);                                         // :319 快照完成：关门
    completeCheckpointOnceEventsAreDone(checkpointId, result, success);  // :320 等门关上之前发出的事件都送达
});
subtaskGatewayMap.forEach((subtask, gateway) -> gateway.markForCheckpoint(checkpointId));   // :335
coordinator.checkpointCoordinator(checkpointId, coordinatorCheckpoint);
```

- **关门之后**，Coordinator 发出的事件先被 `SubtaskGatewayImpl` 暂存起来（`blockedEventsMap`，`SubtaskGatewayImpl.java:147-148`），因为它们属于 Checkpoint N 之后；
- **关门之前**发出、但还没确认送达的事件，由 `completeCheckpointOnceEventsAreDone` 等待，全部送达后，Coordinator 的快照才算完成；
- **开门**：Task 收到 Barrier、做完快照后，发回一个 `AcknowledgeCheckpointEvent`；Holder 收到后打开这个 subtask 的门，发出暂存的事件（`OperatorCoordinatorHolder.java:217-221`）；
- **如果 Task 在这期间失败了**，暂存的事件直接丢弃。类注释（`:82-84`）的解释是：从 Coordinator 的角度看，这些事件是在这个 subtask 最近一个完成的 Checkpoint 之后发给它的，本来就该当作丢失处理；第三节的 `addSplitsBack` 会把相应的 split 找回来。

所以任何一个 `AddSplitEvent`，要么在 Barrier 之前到达（split 进入 Reader 的状态），要么在开门之后到达（split 在 Coordinator 的快照里属于"N 之后的分配"）。**不会落进两者之间的缝隙。**

> 这一节只读了代码，示例的日志里没有专门观察"关门"的时刻。

---

## 五、TaskManager 侧：Reader 怎么读

### 5.1 SourceOperator：Reader 的宿主

`SourceOperator`（`RT/streaming/api/operators/SourceOperator.java`）负责把 Reader 接进 Task：

```java
public DataInputStatus emitNext(DataOutput<OUT> output) throws Exception {   // :516 Mailbox 默认动作，每次调用一次
    ...
    do {
        status = sourceReader.pollNext(currentMainOutput);                    // :535
    } while (status == InputStatus.MORE_AVAILABLE && canEmitBatchOfRecords.check() && ...);
    ...
}
```

- split 状态存在一个名为 `"SourceReaderState"` 的 **operator state** 里（`:115`）。它不是 keyed state，扩缩容时按 round-robin 重新分配（第三讲上篇）。
- 收到 `AddSplitEvent`（`handleOperatorEvent`，`:719`）后，先为新 split 创建独立的 Watermark 输出（第七讲上篇），再调用 `sourceReader.addSplits(...)`。
- 这些都在 **Task 的 Mailbox 线程**里执行（第一讲下篇），所以 `SourceReader` 的所有方法都在同一个线程里调用，也不需要加锁。

### 5.2 SourceReaderBase：阻塞的客户端怎么办

`pollNext` 在 Mailbox 线程里执行，**不能阻塞**，否则 Checkpoint、Timer 都没法处理。但大多数外部客户端（Kafka Consumer、HDFS 输入流）的读取接口是阻塞的。

`flink-connector-base` 里的 `SourceReaderBase` 用"fetcher 线程 + 有界队列"解决这个问题：

```text
  Fetcher 线程（"Source Data Fetcher for ..."，BASE/source/reader/fetcher/SplitFetcherManager.java:64）
  ┌───────────────────────────────────────────────┐
  │ lastRecords = splitReader.fetch();   阻塞读     │  FetchTask.java:58
  │ elementsQueue.put(fetcherIndex, lastRecords)  │  :64 队列满就阻塞 → 不再向外部系统拉数据
  └──────────────────────┬────────────────────────┘
                         │ 有界队列（source.reader.element.queue.capacity，默认 2 批）
                         ▼
  Task 线程（Mailbox）
  ┌───────────────────────────────────────────────┐
  │ SourceReaderBase.pollNext()                    │  BASE/source/reader/SourceReaderBase.java:177
  │   getNextFetch(output)          非阻塞地取一批   │  :181
  │   recordEmitter.emitRecord(...) 发出记录，更新进度│  :202
  │ isAvailable()：队列空时让 Mailbox 挂起等待        │  :341
  └───────────────────────────────────────────────┘
```

（`FetchTask.java` 在 `BASE/source/reader/fetcher/` 下；队列容量的默认值见 `BASE/source/reader/SourceReaderOptions.java:36-39`。）

由此可以推出三点：

1. **背压一直传到外部系统**：下游慢 → Task 线程 `pollNext` 变慢 → 队列满 → fetcher 线程阻塞在 `put` 上 → 不再调用 `fetch()` → 外部系统里的积压（比如 Kafka 的消费 lag）开始增长。
2. **进度由 Task 线程维护**：`emitRecord` 在发出记录的同时更新 split 的进度，`snapshotState` 把进度转回 split 对象（`SourceReaderBase.java:348`）。fetcher 线程只管读、不碰状态，所以 Checkpoint 时不需要和 fetcher 线程同步。
3. **队列里还没发出的数据不在状态里**：恢复后，会从状态里记录的进度重新读。这正是我们想要的：已经读到队列、但还没发给下游的数据，不应该算"读过了"。

第七讲上篇说过，`SourceReaderBase` 默认为每个 split 调用 `createOutputForSplit`（`:471`），所以基于它的 Source 都有 Split 级别的 Watermark。

> 本篇的示例没有基于 `SourceReaderBase` 实现，这一节的线程模型只读了代码，没有做实验。

---

## 六、自己动手

示例代码在 `flink-notes/demos`：

```bash
./run.sh study.connector.SourceSinkDemo normal
```

```bash
./run.sh study.connector.SourceSinkDemo fail
```

`SourceSinkDemo` 大约 400 行，**包含一个完整的 FLIP-27 Source**（拉模式，4 个 split），没有依赖任何连接器基础类，可以作为自己写 Source 的起点。

split-2 可能分给任何一个 subtask，取决于谁先请求。多跑几次，故障可能发生在 subtask 0 上，这时日志里是 `addSplitsBack([split-2[80,120)]) from failed subtask 0`，结果同样是 `duplicates=0`。

**推荐的断点**：

| 断点 | 看什么 |
|---|---|
| `SourceCoordinatorContext.java:305` | 每一次分配被记下来 |
| `SplitAssignmentTracker.java:63` | Checkpoint 时归档 |
| `SourceCoordinator.java:389` | 故障后哪些 split 被还给 Enumerator |
| `OperatorCoordinatorHolder.java:319` | 关门 |
| `SourceOperator.java:451` | 恢复时 split 直接交还给 Reader |
| `SourceOperator.java:535` | 每次 `pollNext` |

---

## 七、课后练习

1. **推模式**：把 `SourceSinkDemo` 的 Enumerator 改成在 `addReader` 时一次性把所有 split 推给 Reader。注入同样的故障，`addSplitsBack` 还会出现吗？为什么？
2. **故障时机**：把故障注入在 split-0 的中间（比如记录 20），恢复后 Reader 拿到的是哪个 split、从哪里开始读？
3. **队列容量**：基于 `SourceReaderBase` 的 Source，把 `source.reader.element.queue.capacity` 调大，对内存和背压各有什么影响？
4. **思考**：为什么 Coordinator 的快照要在 Barrier 之前完成，而不是之后？如果反过来，4.2 节的缝隙会变成什么样？

---

## 写在最后

这一讲的核心可以用一句话概括：**每个 split 在任意一个 Checkpoint 里，要么在 Enumerator 的状态里，要么在某个 Reader 的状态里，而且只在一处。**

为了保证这一点，Flink 做了三件事：

- 分配时记账（`SplitAssignmentTracker`），故障时把 Checkpoint 之后分出去的 split 还给 Enumerator；
- Checkpoint 时先做 Coordinator 的快照，再发 Barrier；
- 用"关门"堵住事件在路上的那个缝隙。

回到开头：split-2 在 Checkpoint 5 之后才分给 subtask 1，所以在 Checkpoint 5 里，它还属于 Enumerator。subtask 1 失败后，split-2 被还给 Enumerator，重新分配。160 条记录，一条不丢，一条不重。

不过，"Source 不丢不重"还不等于"结果不丢不重"。Source 回滚了，Sink 已经写出去的数据怎么办？下篇我们看 **Sink V2 的两阶段提交**，把踩坑实验室 #06、#07、#08 串成完整的端到端 exactly-once。

> **第二季预告**：第一季讲完之后，第二季讲 **Apache Paimon**。其中第 10 讲"Flink 流读"会用到这一讲的全部概念：Paimon 的流读 Source 同样是 FLIP-27 的 Enumerator + Reader，Enumerator（`ContinuousFileSplitEnumerator`）不断发现新的快照，把里面的数据文件切成 split（`FileStoreSourceSplit`）分给 Reader。到时候你会看到，"split 只在一处"这条规则在湖仓里是怎么落地的。（类名基于 Paimon master @ `d15d250cf`，第二季会按当时的正式版复核；第二季的实验用 Flink 2.2，Paimon 目前支持到 Flink 2.2。）


**留一个问题**：你自己写过 Source 吗？用的是老的 `SourceFunction` 还是 FLIP-27？迁移时遇到过什么问题？

下一讲：**Flink 2.x 源码精读（八·下）：Sink 与端到端 exactly-once**
:::
