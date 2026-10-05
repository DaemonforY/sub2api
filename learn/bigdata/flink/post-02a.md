---
title: "Flink 2.x 源码精读（二·上）：背压一来，Checkpoint 从 15 毫秒变成 3 秒，时间花在了哪？"
description: "同一个作业，有背压时 Checkpoint 从 15 毫秒涨到 3 秒，但快照本身只花了不到 1 毫秒。顺着这个现象，读懂 Flink 2.3 Checkpoint 从触发、Barrier 对齐、快照到完成通知的全过程，并实测\"一次超时就全局重启\"。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-02a-cover.webp"}]]
---

# Flink 2.x 源码精读（二·上）：背压一来，Checkpoint 从 15 毫秒变成 3 秒，时间花在了哪？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02a-cover.webp" alt="Yui和Kai守望被背压拖慢的Checkpoint数据河流" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai守望被背压拖慢的Checkpoint数据河流<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 前置阅读：第一讲（下）的 Mailbox 模型

---

先看一组本机实测的数据。

同一个作业：Source 并行度 2，每秒产生 2000 条、每条约 1KB 的数据，按 key 计数，每 5 秒做一次 Checkpoint。我只改了一个地方：让计数算子每处理一条数据就 `sleep(5)` 毫秒。这样它每秒最多只能处理约 200 条，远远跟不上上游，于是产生了**稳定的背压**。

| | ckpt 1 | ckpt 2 | ckpt 3 | ckpt 4 | ckpt 5 | ckpt 6 |
|---|---|---|---|---|---|---|
| 无背压 | 44 ms | 15 ms | 14 ms | 19 ms | 16 ms | 29 ms |
| 有背压 | 442 ms | **2705 ms** | **3201 ms** | **3136 ms** | **3260 ms** | **3211 ms** |

（数据取自日志中 `Completed checkpoint N (... checkpointDuration=... ms)` 这一行。我前后跑过三轮，有背压时第 2 次以后的耗时都在 2.7~3.9 秒之间。）

Checkpoint 的大小**两组完全一样**，每次都是 5714 字节。状态没有变大，Checkpoint 却慢了 200 倍。

更麻烦的是：如果把 Checkpoint 的超时时间设成 1 秒，日志里会出现这样几行（本机实测，节选）：

```
Checkpoint 2 of job d3677c57... expired before completing.
FlinkRuntimeException: Exceeded checkpoint tolerable failure threshold. ...
JobMaster - 4 tasks will be restarted to recover from a global failure.
```

**一次 Checkpoint 超时，整个作业的 4 个 Task 全部重启。** 之后每隔几秒就重复一次。

这 3 秒究竟花在了哪里？为什么一次超时就要全部重启？这一篇，我们从头到尾走一遍 Checkpoint 的全过程，最后用 Web UI 同款的数据，把这 3 秒拆开来看。

本篇要回答：

1. Checkpoint 是谁、在什么时候触发的？为什么 JobManager **只通知 Source**？
2. Barrier 是怎么插进数据流的？做快照时为什么不需要暂停整个作业？
3. 下游有多个上游时，"对齐"到底在做什么？"阻塞通道"阻塞的是什么？
4. 什么时候才算 Checkpoint 完成？
5. 背压下慢掉的那 3 秒，究竟花在了哪一步？

---

## 一、Checkpoint 要解决什么问题

Flink 作业是分布在很多台机器上的很多个 Task。要做一次"全局一致"的快照，最笨的办法是：**让所有 Task 停下来，各自存一份状态，再一起继续**。但流作业不能停。

Flink 的做法来自一篇论文里的算法，叫 **ABS**（Asynchronous Barrier Snapshotting，异步屏障快照）。思路很简单：

- JobManager 让 Source 往数据流里插入一个特殊的标记，叫 **Barrier**，它和普通数据一起往下游流动，**不会超过前面的数据**；
- 一个算子从**所有**上游都收到 Barrier 之后，就给自己的状态做快照，然后把 Barrier 继续发给下游；
- 这样，每个算子的快照恰好对应同一个"切面"：**Barrier 之前的数据都处理完了，Barrier 之后的数据一条都还没处理**；
- 所有 Task 都报告完成后，这次 Checkpoint 才算成功。

【配图 1：Barrier 在数据流里流动的示意。一条从左到右的数据流，中间夹着一个醒目的 Barrier 标记，标记左边的数据标注"属于下一个 Checkpoint"，右边的数据标注"属于本次 Checkpoint"，下游有两个输入通道汇入一个算子】

下面按时间顺序，分四个阶段看源码。

---

## 二、全景图：四个阶段、四类线程

```
 JobManager                                            TaskManager
 ──────────                                            ───────────
 ① [Checkpoint Timer 线程] 触发
    计算要触发谁、等谁
    先给 Coordinator 做快照
    RPC：只发给 Source ──────────────────────────►  ② [Source 的 Task 线程，在 Mailbox 里]
                                                       发 Barrier 给下游 ──────► ③ [下游 Task 线程]
                                                       同步快照                     Barrier 对齐
                                                       └► [AsyncOperations 线程]    同步快照 → 异步上传
                                                            异步上传状态                    │
 ④ [jobmanager-io 线程] ◄──────── ack ─────────────────────┴──────────────────────────────┘
    全部 ack → 写 _metadata → 完成
    RPC：通知完成 ──────────────────────────────►  所有 Task：notifyCheckpointComplete（在 Mailbox 里）
```

| 线程 | 做什么 | 依据 |
|---|---|---|
| `Checkpoint Timer` | 定时触发 Checkpoint、检测超时 | 创建于 `RT/runtime/executiongraph/DefaultExecutionGraph.java:519`；本机日志中 `Triggering checkpoint` 和 `expired before completing` 都打印在这个线程上 |
| Task 线程 | Barrier 对齐、同步快照、处理完成通知 | 全部在 **Mailbox** 里执行（第一讲下篇） |
| `AsyncOperations` | 把状态写到 Checkpoint 存储 | 线程池创建于 `RT/streaming/runtime/tasks/StreamTask.java:465` |
| `jobmanager-io-thread-*` | 汇总 ack、完成 Checkpoint | 本机日志中 `Completed checkpoint` 打印在这个线程上 |

---

## 三、阶段①：JobManager 触发

### 3.1 什么时候开始

上一讲说过，`CheckpointCoordinator` 是在构建 ExecutionGraph 时创建的（`DefaultExecutionGraph.java:523`）。但它不会马上开始工作。启动的条件在 `RT/runtime/checkpoint/ExecutionTrackingCheckpointCoordinatorDeActivator.java:80-84`：

```java
if (jobStatus == JobStatus.RUNNING && allTasksRunning()) {
    ...
    coordinator.startCheckpointScheduler();
}
```

**作业是 RUNNING 还不够，必须等所有 Task 都进入 RUNNING。** 道理很简单：只要还有 Task 没部署好，Barrier 就过不去，这次 Checkpoint 注定超时。

### 3.2 定时触发，但要先过一道关

`startCheckpointScheduler()`（`RT/runtime/checkpoint/CheckpointCoordinator.java:2032`）在 `Checkpoint Timer` 线程上注册一个定时任务 `ScheduledTrigger`（`:2178`）。每次到点，调用 `triggerCheckpoint()`（`:567`）。

但触发请求不会马上执行，而是先经过一个"调度员"`CheckpointRequestDecider`（`:620`），它负责执行以下几条限制：

| 配置项（`flink-core/.../configuration/CheckpointingOptions.java`） | 默认值 | 作用 |
|---|---|---|
| `execution.checkpointing.interval`（`:512`） | 无（不设置就不开启） | 触发间隔 |
| `execution.checkpointing.timeout`（`:410`） | **10 分钟** | 超过就判定失败 |
| `execution.checkpointing.max-concurrent-checkpoints`（`:417`） | **1** | 同时进行的 Checkpoint 个数 |
| `execution.checkpointing.min-pause`（`:427`） | **0** | 上一次完成到下一次触发之间，至少间隔多久 |
| `execution.checkpointing.tolerable-failed-checkpoints`（`:446`） | **0** | 允许连续失败几次（第七节会用到） |

### 3.3 触发流水线

`startTriggeringCheckpoint()`（`:624`）是 JobManager 上最核心的方法，它是一条 `CompletableFuture` 链：

| 步骤 | 行号 | 做什么 |
|---|---|---|
| ① 算出计划 | `:637` | 决定**触发谁**、**等谁的 ack**、**完成后通知谁** |
| ② 分配 ID | `:653` | `checkpointIdCounter.getAndIncrement()`。高可用模式下，这个计数器存在 ZooKeeper 或 Kubernetes 中，保证 JobManager 切换后 ID 不会倒退 |
| ③ 创建 PendingCheckpoint | `:895` | 同时注册超时任务 `CheckpointCanceller`（类定义在 `:2323`）。超时就是它判定的 |
| ④ Coordinator 先做快照 | `:703` | 比如 Source 的 `SourceCoordinator`，第八讲会讲它为什么要先做 |
| ⑤ 通知 Task | `:782` → `:830` | 发 RPC：`execution.triggerCheckpoint(...)`（`:858`） |

**为什么只通知 Source？** 计算计划时（`RT/runtime/checkpoint/DefaultCheckpointPlanCalculator.java:158`），只有 Source 被放进"触发列表"。其他算子不需要 JobManager 通知，**Barrier 流到它们那里，就是通知**。

---

## 四、阶段②：Source 插入 Barrier

### 4.1 RPC 不直接做快照，而是投递一封邮件

Source 所在的 TaskManager 收到 RPC 后，最终调用到 `StreamTask.triggerCheckpointAsync()`（`RT/streaming/runtime/tasks/StreamTask.java:1305`），它把 Checkpoint 包装成一封邮件，投进 Task 的 Mailbox。

第一讲下篇说过，Mailbox 里的邮件和数据处理在**同一个线程上串行执行**。所以执行 Checkpoint 时，一定没有哪条数据正处理到一半，**不需要锁，也不需要暂停其他 Task**。

### 4.2 最关键的 6 步

邮件被执行后，经过 `performCheckpoint()`（`:1467`），来到整个流程中**最重要的方法**：`SubtaskCheckpointCoordinatorImpl.checkpointState()`（`RT/streaming/runtime/tasks/SubtaskCheckpointCoordinatorImpl.java:271`）。源码注释把它分成了 6 步：

| 步骤 | 行号 | 做什么 |
|---|---|---|
| Step (0) | `:301` | 记录这次的 Checkpoint ID；如果它已经被取消了，就通知下游后直接返回 |
| **Step (1)** | `:331` | `prepareSnapshotPreBarrier`：**Barrier 发出前**的最后机会，让算子把缓存的数据先发出去 |
| **Step (2)** | `:335` | **向下游广播 Barrier** |
| Step (3) | `:346` | 注册对齐超时定时器（下篇讲非对齐时用到） |
| Step (4) | `:349` | 非对齐模式下的准备工作（下篇） |
| **Step (5)** | `:355` | **做快照** |

两个值得注意的细节：

**先发 Barrier，后做快照。** 这样下游可以尽早开始自己的流程，和本 Task 的快照并行进行。因为这 6 步是在同一封邮件里连续执行的，中间不会插入任何数据处理，所以一致性不受影响。

**Step (1) 的典型用途**：两阶段提交（第八讲会详细讲），Sink 的 Writer 正是在这一步把"待提交的事务"发给下游的 Committer（`RT/streaming/runtime/operators/sink/SinkWriterOperator.java:188`）。它必须赶在 Barrier 前面，Committer 做快照时才能拿到本轮所有的事务。

### 4.3 快照分两段：同步和异步

Step (5) 会对链上的每个算子调用 `StreamOperatorStateHandler.snapshotState()`（`RT/streaming/api/operators/StreamOperatorStateHandler.java:178`），依次处理定时器（`:257`）、**你写的 `snapshotState()`**（`:261`）、Operator State（`:269`）和 Keyed State（`:295`）。

这时状态**还没有真正写到文件里**。状态后端把快照拆成两段（`RT/runtime/state/SnapshotStrategyRunner.java:70`）：

| 阶段 | 行号 | 在哪个线程 | 做什么 |
|---|---|---|---|
| **同步** | `:77` | Task 线程（**会阻塞数据处理**） | 只做"拍照"：比如 HashMap 状态后端用写时复制记下当前状态；RocksDB 状态后端做一次本地的轻量快照 |
| **异步** | `:80` | `AsyncOperations` 线程 | 把快照真正写到 Checkpoint 存储（比如 HDFS、S3） |

**Web UI 上的 Sync Duration 和 Async Duration，就是这两段各自的耗时。** 同步阶段要尽可能短，因为它会挡住数据处理；耗时的上传工作都放在异步阶段。两种状态后端的具体实现，第三讲会展开。

### 4.4 ack 里带回的是"地址"，不是状态

异步阶段完成后，Task 向 JobManager 发送 ack（`RT/runtime/state/TaskStateManagerImpl.java:158`）。ack 里带的是**状态句柄**，也就是"状态存在哪个文件的什么位置"，而不是状态数据本身，所以 ack 消息很小。

---

## 五、阶段③：下游 Task 对齐 Barrier

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02a-1.webp" alt="多条数据河流等待Barrier全部到齐后继续前进" width="960" height="640" loading="lazy" /><figcaption>多条数据河流等待Barrier全部到齐后继续前进<span>AI 生成配图</span></figcaption></figure>

### 5.1 Barrier 在哪里被拦下

第一讲下篇看过一条数据进入 Task 的路径。Barrier 走的是同一条路，但它在**进入算子之前**就被拦下了：`CheckpointedInputGate` 读到 Barrier 时，交给 `barrierHandler.processBarrier(...)`（`RT/streaming/runtime/io/checkpointing/CheckpointedInputGate.java:182`）。

### 5.2 对齐：等所有上游的 Barrier 都到齐

一个 Task 往往有多个输入通道，比如 keyBy 之后，每个下游 Task 都要接收所有上游 Task 的数据。对齐的逻辑写成了一个小小的状态机（`SingleCheckpointBarrierHandler.aligned()`，`RT/streaming/runtime/io/checkpointing/SingleCheckpointBarrierHandler.java:147`）：

```
  等待第一个 Barrier ──收到第 1 个 Barrier──► 收集 Barrier 中
        ▲                阻塞这个通道          │ 每收到一个 Barrier：阻塞这个通道
        │                                     │
        └──── 所有通道都到齐：做快照、解除所有阻塞 ────┘
```

核心代码在 `RT/streaming/runtime/io/checkpointing/AbstractAlignedBarrierHandlerState.java`：收到 Barrier 就阻塞这个通道（`:62`）；所有通道都到齐后，触发快照并解除阻塞（`:72` 起的 `triggerGlobalCheckpoint`）。快照走的是和 Source 完全相同的 6 步：发 Barrier 给自己的下游、做快照、ack。

**为什么先到的通道要阻塞？** 如果不阻塞，先到 Barrier 的那个通道会继续送来"Barrier 之后"的数据，它们会被算进本次快照，切面就不一致了。

### 5.3 "阻塞通道"到底阻塞了什么

很多资料说："对齐时，会把先到 Barrier 的那个通道后续的数据缓存起来。"这是 **Flink 早期版本**的做法。现在的实现完全不同：**让上游停止发送**。

1. 对齐用的 Barrier，类型是 `ALIGNED_CHECKPOINT_BARRIER`（`RT/runtime/io/network/buffer/Buffer.java:298`），它带有"阻塞上游"的属性；
2. 上游的输出队列把这个 Barrier 交给网络层之后，**把自己标记为阻塞**，不再向这个下游通道发送任何数据（`RT/runtime/io/network/partition/PipelinedSubpartition.java:534-535`）：

   ```java
   if (buffer.getDataType().isBlockingUpstream()) {
       isBlocked = true;
   }
   ```
3. 下游对齐完成后，解除阻塞（`RT/streaming/runtime/io/checkpointing/ChannelState.java:74`），通过网络通知上游，上游再恢复发送（`PipelinedSubpartition.java:567`）。

所以对齐期间，被挡住的数据**留在上游的输出缓冲区里**，下游不需要额外的内存。

---

## 六、阶段④：完成与通知

### 6.1 收齐 ack

所有 ack 最终来到 `CheckpointCoordinator.receiveAcknowledgeMessage()`（`:1204`）。每来一个 ack，就从"还没确认的 Task"里划掉一个（`:1248` → `RT/runtime/checkpoint/PendingCheckpoint.java:385`）。全部划完，就调用 `completePendingCheckpoint()`（`:1359`）：

1. **写 `_metadata` 文件**（`PendingCheckpoint.java:339`）。它记录了所有算子的状态句柄，恢复时读的就是它。**这个文件写完，Checkpoint 才算真正落盘**；
2. 存入已完成 Checkpoint 列表，并删掉超出保留数量的旧 Checkpoint（`:1504`）；
3. 通知所有 Task：Checkpoint 完成了（`:1570`，逐个调用 `:1579`）。

### 6.2 完成通知

Task 收到通知后，同样是投递一封邮件（`StreamTask.java:1560`），最终依次调用链上每个算子的 `notifyCheckpointComplete()`（`:1612`）。

**两阶段提交的"第二阶段"就是在这里完成的。** 它有两个推论，第八讲和后面的踩坑实验室会分别实测：
- 通知可能丢失，作业恢复后会再提交一次，所以提交必须幂等；
- 不开 Checkpoint，就永远等不到这个通知，事务型 Sink 就永远不会提交。

---

## 七、回到开头：3 秒花在了哪

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02a-2.webp" alt="背压让Barrier长时间排队而快照瞬间完成" width="960" height="640" loading="lazy" /><figcaption>背压让Barrier长时间排队而快照瞬间完成<span>AI 生成配图</span></figcaption></figure>

### 7.1 拆开来看

Web UI 的 Checkpoints → Details 页面里，每个子任务都有一组分项耗时。我在实验结束前，通过 REST 接口取出了最近一次 Checkpoint 的数据（和 Web UI 显示的是同一份）：

**无背压**（本机实测，第 7 次 Checkpoint，总耗时 35ms）：

```
vertex                   sub   start_delay  alignment     sync    async   end_to_end
Source: datagen          0            22ms        0ms      1ms      2ms         34ms
Source: datagen          1            22ms        0ms      1ms      2ms         35ms
count-per-key -> checkpo 0            24ms        0ms      0ms      4ms         35ms
count-per-key -> checkpo 1            25ms        0ms      0ms      7ms         35ms
```

**有背压**（本机实测，与开头表格是同一次运行的第 6 次 Checkpoint；REST 接口给出的总耗时是 3203ms，日志里的 3211ms 还包含了 JobManager 最后写元数据等收尾工作）：

```
vertex                   sub   start_delay  alignment     sync    async   end_to_end
Source: datagen          0             3ms        0ms      0ms      0ms          5ms
Source: datagen          1             3ms        0ms      0ms      0ms          5ms
count-per-key -> checkpo 0           150ms        0ms      0ms      1ms        154ms
count-per-key -> checkpo 1          3201ms        0ms      0ms      0ms       3203ms
```

答案一目了然：**同步快照和异步快照都只花了 0~1 毫秒。** 3 秒几乎全部记在了 **start_delay** 上。

这两列的定义在 `RT/streaming/runtime/io/checkpointing/CheckpointBarrierHandler.java:167-177`：

- **start_delay**：**第一个** Barrier 到达这个子任务的时刻，减去 Checkpoint 创建的时刻。也就是 Barrier 在路上走了多久；
- **alignment**：从第一个 Barrier 到达，到**最后一个** Barrier 到达，中间等了多久。

**有背压时，上游和网络缓冲区里堆满了还没处理的数据。Barrier 不会超车，只能排在这些数据后面，等下游一条一条处理完，才轮到它。** 下游每秒只能处理约 200 条，几百条数据排在 Barrier 前面，Barrier 就要等上好几秒。

我又跑了一次，这次的分项是这样的（本机实测，第 6 次 Checkpoint，总耗时 3295ms）：

```
count-per-key -> checkpo 0             2ms      127ms      0ms      1ms        132ms
count-per-key -> checkpo 1          3292ms        0ms      0ms      0ms       3295ms
```

子任务 0 的第一个 Barrier 很快就到了，但还要等另一个通道的 Barrier，这段时间就记在了 alignment 上。所以，**排队等待的时间到底记在 start_delay 还是 alignment 上，取决于哪个通道的 Barrier 先到；而整个 Checkpoint 的耗时，由最慢的那个子任务决定。**

> ⚠️ 不少资料（包括我自己最早的笔记）把背压下慢掉的这部分时间统称为"对齐时间"。从上面的数据看，这并不准确。排查时要同时看 Start Delay 和 Alignment Duration 两列。

**结论：背压下 Checkpoint 变慢，不是因为快照慢，而是 Barrier 在排队。** 状态再小也没用。要解决它，要么解决背压本身（第四讲），要么让 Barrier 不再排队，这就是下篇要讲的**非对齐 Checkpoint**。

> 思考题：上面两个下游子任务的差距非常大（150ms 和 3201ms）。我用 Flink 的 key 分配函数算了一下，16 个 key 被分成了 7 个和 9 个。这能不能完全解释这个差距？还有哪些因素？

### 7.2 为什么一次超时，整个作业就重启了

开头那个实验，我把 Checkpoint 超时设成了 1 秒。第 2 次 Checkpoint 需要 3 秒左右，于是超时了。之后发生的事情，由 `CheckpointFailureManager` 决定（`RT/runtime/checkpoint/CheckpointFailureManager.java`）：

1. 超时属于会被计数的失败原因（`CHECKPOINT_EXPIRED`，`:249-258`），连续失败计数加 1；
2. 计数超过 `tolerable-failed-checkpoints`，就抛出 `Exceeded checkpoint tolerable failure threshold`，让作业失败（`:204-212`）；
3. 而这个配置的**默认值是 0**（`CheckpointingOptions.java:446`）。也就是说，**默认情况下，只要有一次 Checkpoint 超时，作业就会失败**，然后由重启策略接管。本机日志显示的是"4 tasks will be restarted to recover from a global failure"，所有 Task 全部重启。

并不是所有失败都会被计数。比如 Task 还没准备好、作业正在关闭这类情况，会被忽略（`:224-247`）。

**生产环境的建议**：
- Checkpoint 超时不要设得太紧，先看清楚正常情况下耗时是多少；
- 根据业务能接受的程度，把 `tolerable-failed-checkpoints` 设成一个大于 0 的值，避免偶尔一次超时就让整个作业重启；
- 但不要无限放宽：Checkpoint 长期失败，意味着出故障时要回退到很久以前的状态，重放大量数据。

---

## 八、自己动手

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/checkpoint/CheckpointDemo.java`，每次运行 30 秒：

```bash
./run.sh study.checkpoint.CheckpointDemo                  # 无背压
./run.sh study.checkpoint.CheckpointDemo slow             # 有背压
./run.sh study.checkpoint.CheckpointDemo slow timeout     # 有背压 + 超时 1 秒
```

结束前会打印最近一次 Checkpoint 的分项耗时，就是第七节里的那张表。

**断点清单**（断点的 Suspend 一定要改成 Thread，否则 Checkpoint 会超时）：

| # | 线程 | 位置 | 看什么 |
|---|---|---|---|
| 1 | pekko 调度线程 | `ExecutionTrackingCheckpointCoordinatorDeActivator.java:84` | 什么时候开始调度 |
| 2 | Checkpoint Timer | `CheckpointCoordinator.java:624` | 触发流水线 |
| 3 | Checkpoint Timer | `DefaultCheckpointPlanCalculator.java:158` | 触发列表里只有 Source |
| 4 | Source 的 Task 线程 | `StreamTask.java:1344` | 线程名就是 Task 线程本身 |
| 5 | Source 的 Task 线程 | `SubtaskCheckpointCoordinatorImpl.java:335` | 广播 Barrier |
| 6 | 下游 Task 线程 | `SingleCheckpointBarrierHandler.java:214` | Barrier 从哪个通道来 |
| 7 | 下游 Task 线程 | `AbstractAlignedBarrierHandlerState.java:62` | 阻塞通道 |
| 8 | AsyncOperations | `AsyncCheckpointRunnable.java:109` | 异步上传 |
| 9 | jobmanager-io | `CheckpointCoordinator.java:1248` | 未确认的 Task 越来越少 |
| 10 | jobmanager-io | `PendingCheckpoint.java:339` | 写 `_metadata` |

---

## 九、课后练习

1. **推导**：把实验里的 `sleep(5)` 改成 `sleep(1)`，背压会减轻。先预测有背压时 Checkpoint 的耗时会变成多少，再运行验证。
2. **读代码**：Step (2) 先发 Barrier，Step (5) 再做快照。如果 Step (5) 失败了，下游已经收到 Barrier，会发生什么？（提示：顺着 `declineCheckpoint` 往下找）
3. **找默认值**：`execution.checkpointing.num-retained`（保留几个已完成的 Checkpoint）的默认值是多少？它在 `CheckpointingOptions.java` 的哪一行？
4. **思考**：既然对齐只是让上游暂停发送，那么在**只有一个输入通道**的 Task 上，对齐会阻塞什么吗？Web UI 上它的 alignment 会是多少？

---

## 写在最后

这一篇我们把 Checkpoint 拆成了四个阶段：JobManager 只触发 Source；Source 先发 Barrier、再做快照；下游等所有 Barrier 到齐再快照；JobManager 收齐 ack、写完 `_metadata` 才算完成。

最后的实验告诉我们：**背压下 Checkpoint 变慢，慢的是 Barrier 在排队，而不是快照本身。**

下一篇，我们看 Flink 是怎么让 Barrier "插队"的：**非对齐 Checkpoint**。同样的背压，它能把耗时从 3 秒压回十几毫秒，代价是每次多存了 600KB 左右的数据。这 600KB 是什么？什么时候该用、什么时候不该用？


**留一个问题**：你的作业遇到过 Checkpoint 超时吗？当时 Start Delay 和 Alignment Duration 分别是多少？

下一篇：**Flink 2.x 源码精读（二·下）：非对齐 Checkpoint，Barrier 是怎么插队的？**
:::
