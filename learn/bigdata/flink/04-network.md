---
title: "Flink 源码导读 04：网络栈与背压 —— 从 RecordWriter 到 Netty，再到 Credit 流控"
description: "网络栈：ResultPartition、InputGate、Credit-based 流控，以及背压是怎样产生和传递的。"
bigdata: "flink"
lesson: "f4"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/04-network-cover.webp"}]]
---

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/04-network-cover.webp" alt="网络栈与背压全景图" width="1200" height="800" loading="eager" /><figcaption>网络栈与背压全景图<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`NET` = `flink-runtime/src/main/java/org/apache/flink/runtime/io/network`，`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 前置阅读：[01](/bigdata/flink/01-execution) 第 6 节（mailbox 与 `processInput`）、[02](/bigdata/flink/02-checkpoint) 第 5 节（Barrier 对齐阻塞上游）
> 配套示例：`demos/src/main/java/study/network/BackpressureDemo.java`，第 8 节的所有数据都是在本机实测的

## 0. 本篇要回答的问题

1. 一条记录从上游算子发出后，经过了哪些数据结构才到达下游？
2. Network Buffer 是怎么分配的？一个 Task 能用多少个？
3. **Credit-based 流控**是怎么工作的？背压是怎么一层层传回 Source 的？
4. Web UI 上的 Busy / Idle / BackPressured 是怎么算出来的？怎么根据它们定位瓶颈？
5. Buffer Debloating 调整的是什么？为什么能缩短 Checkpoint 时间？

## 1. 全景图

```
上游 Task（Source: source (1/2)）                                     下游 Task（slow-map (2/2)）
─────────────────────────────                                         ──────────────────────
算子 output.collect()
 └ RecordWriterOutput
   └ RecordWriter.emit()           序列化：[4B 长度][数据]
     └ ChannelSelector 选下游        keyBy → KeyGroupStreamPartitioner
       └ BufferWritingResultPartition.emitRecord()
          └ 写进 BufferBuilder（从 LocalBufferPool 申请）
            └ PipelinedSubpartition[i] 的 buffers 队列
                   │
       同一个 TM ──┼──► LocalInputChannel：直接读上游 subpartition，不经过 Netty，不走 credit
                   │
       不同 TM ────┴──► Netty Server                          Netty Client
                        PartitionRequestQueue    BufferResponse    CreditBasedPartitionRequestClientHandler
                        (有 credit 才发)     ─────────────────►   └ RemoteInputChannel.onBuffer()
                                             ◄─────────────────      （收到后回收/申请 buffer，产生新 credit）
                                                 AddCredit
                                                                   SingleInputGate（gate 下所有 channel）
                                                                    └ CheckpointedInputGate（处理 Barrier）
                                                                      └ StreamTaskNetworkInput.emitNext()
                                                                        └ 反序列化 → processElement()
```

核心概念对照：

| 上游（生产者） | 下游（消费者） | 粒度 |
|---|---|---|
| `ResultPartition` | `InputGate`（`SingleInputGate`） | 一个 Task 对一个中间结果集（`IntermediateDataSet`） |
| `ResultSubpartition`（`PipelinedSubpartition`） | `InputChannel`（`Local`/`RemoteInputChannel`） | 一对一：上游第 i 个 subpartition ↔ 下游 subtask i 的一个 channel |

上游有 M 个并行度、下游有 N 个并行度时，keyBy 会产生 **M × N 条逻辑通道**。示例里 2 × 2 = 4 条：其中 2 条在同一个 TM 内，2 条要跨 TM。

---

## 2. 写入：记录怎么变成 Buffer

### 2.1 `RecordWriter`

`NET/api/writer/ChannelSelectorRecordWriter.java:54`

```java
public void emit(T record) throws IOException {
    emit(record, channelSelector.selectChannel(record));    // 选下游：keyBy 的话就是 03 篇的 KeyGroup 算法
}
```

`RecordWriter.emit(record, targetSubpartition)`（`NET/api/writer/RecordWriter.java:108`）：

```java
targetPartition.emitRecord(serializeRecord(serializer, record), targetSubpartition);
if (flushAlways) { flushAll(); }
```

**序列化格式**（`serializeRecord()`，`:146`）很简单：先空出 4 字节，写入数据，再回填长度。下游的 `SpillingAdaptiveSpanningRecordDeserializer` 就是按这个长度切分记录的，**一条记录可以跨越多个 buffer**。

### 2.2 `BufferWritingResultPartition.emitRecord()`

`NET/partition/BufferWritingResultPartition.java:165`

```java
BufferBuilder buffer = appendUnicastDataForNewRecord(record, targetSubpartition);   // 写进当前 buffer
while (record.hasRemaining()) {
    // full buffer, partial record
    finishUnicastBufferBuilder(targetSubpartition);                               // 写满了：封口
    buffer = appendUnicastDataForRecordContinuation(record, targetSubpartition);  // 申请新 buffer 继续写
}
if (buffer.isFull()) {
    // full buffer, full record
    finishUnicastBufferBuilder(targetSubpartition);
}
// partial buffer, full record   ← 大多数情况：buffer 还没写满，留着给下一条记录用
```

- 每个 subpartition 当前正在写的 buffer 是一个 `BufferBuilder`。第一次写入时就调用 `addToSubpartition()`（`:310`）把它对应的 `BufferConsumer` 挂进 subpartition 的队列，**边写边可读**（写和读共享同一块 `MemorySegment`，通过位置指针协调）
- 所以"buffer 写满"并不是发送的前提。那什么时候发？见 2.3 节

### 2.3 什么时候发出去？buffer 写满，或者 flush

`PipelinedSubpartition.isDataAvailableUnsafe()`（`NET/partition/PipelinedSubpartition.java:623-626`）：

```java
return !isBlocked && (flushRequested || getNumberOfFinishedBuffers() > 0);
```

满足以下任一条件，数据才能被读走：
1. 有**写满封口**的 buffer
2. 有人调用了 **`flush()`**（`:689`）：`flushRequested` 被置为 true，没写满的 buffer 也可以读

谁来调用 flush？**`OutputFlusher` 线程**（`RecordWriter.java:258`）：每隔 `execution.buffer-timeout.interval`（默认 **100ms**，`flink-core/.../ExecutionOptions.java:108-110`）调用一次 `flushAll()`。01 篇实测线程列表里的 `OutputFlusher for Source: datagen (1/2)` 就是它。

> 📊 这就是 Flink **延迟与吞吐的权衡**：buffer timeout 越小，延迟越低，但每个 buffer 装的数据越少，网络和 CPU 开销越大。设为 0 表示每条记录都 flush（`flushAlways`，`RecordWriter.java:90`）。

### 2.4 申请 buffer：软背压与硬背压

`requestNewBufferBuilderFromPool()`（`BufferWritingResultPartition.java:449`）：

```java
BufferBuilder bufferBuilder = bufferPool.requestBufferBuilder(targetSubpartition);   // 非阻塞
if (bufferBuilder != null) {
    return bufferBuilder;
}
hardBackPressuredTimeMsPerSecond.markStart();
bufferBuilder = bufferPool.requestBufferBuilderBlocking(targetSubpartition);         // ★ 阻塞 Task 线程
hardBackPressuredTimeMsPerSecond.markEnd();
```

背压有两种：

| | 发生在哪 | 表现 | 指标 |
|---|---|---|---|
| **软背压**（soft） | `StreamTask.processInput()` 发现 `recordWriter.isAvailable()` 为 false（`StreamTask.java:687`，01 篇 6.3 节） | Task 线程**不处理新数据**，但仍然可以处理 mailbox 里的邮件（Checkpoint、定时器等） | `softBackPressuredTimeMsPerSecond` |
| **硬背压**（hard） | 处理一条记录的**过程中**需要新 buffer 却申请不到（比如一条大记录跨了多个 buffer，或者 flatMap 一次输出很多条） | Task 线程**被阻塞**在 `requestBufferBuilderBlocking`，连邮件都处理不了，Checkpoint 也会被卡住 | `hardBackPressuredTimeMsPerSecond` |

UI 上的 `backPressuredTimeMsPerSecond` 是两者之和（`RT/runtime/metrics/groups/TaskIOMetricGroup.java:240-242`）。

**Overdraft buffer**（FLIP-227）用来缓解硬背压：池子用完时，允许每个 gate 从全局池里**额外透支**最多 20 个 buffer（`LocalBufferPool.requestOverdraftMemorySegmentFromGlobal()`，`NET/buffer/LocalBufferPool.java:450`），让正在处理的那条记录先写完，然后回到软背压状态。这样 Checkpoint 就不会被卡住。

---

## 3. Buffer 从哪来：内存与池

### 3.1 两级池

```
TaskManager 进程
└ NetworkBufferPool（全局，一个 TM 一个）           NET/buffer/NetworkBufferPool.java
   │  总大小 = taskmanager.memory.network.fraction/min/max 决定的网络内存
   │  切成 32KB 的 MemorySegment（taskmanager.memory.segment-size，TaskManagerOptions.java:202-204）
   │  availableMemorySegments（:76）
   │
   ├ LocalBufferPool（每个 ResultPartition 一个）     NET/buffer/LocalBufferPool.java
   └ LocalBufferPool（每个 InputGate 一个）
```

`NetworkBufferPool.createBufferPool()`（`:453`）创建 LocalBufferPool，并通过 `redistributeBuffers()`（`:594`）在各个 LocalBufferPool 之间**动态重新分配**buffer：新的池子创建或旧的销毁时，按各自的需求重新划分。

### 3.2 ★ 2.x 的变化：每个通道用几个 buffer 已经写死在代码里了

Flink 1.x 时代有 4 个著名的配置项：`taskmanager.network.memory.buffers-per-channel`、`floating-buffers-per-gate`、`max-buffers-per-channel`、`max-overdraft-buffers-per-gate`。**这几个配置项在 2.x 中已经删除**，网上的调优文章大多还在讲它们。

2.3.0 中它们变成了硬编码的常量（`RT/runtime/taskmanager/NettyShuffleEnvironmentConfiguration.java`）：

| 含义 | 值 | 行号 |
|---|---|---|
| 每个输入 channel 的**独占** buffer | `buffersPerChannel = 2` | `:325` |
| 每个 InputGate 额外的**浮动** buffer | `extraBuffersPerGate = 8` | `:326` |
| 每个输出 subpartition 最多占用的 buffer | `maxBuffersPerChannel = 10` | `:332` |
| 每个 gate 最多的 overdraft buffer | `20` | `:401` |

（还剩 `taskmanager.network.memory.read-buffer.required-per-gate.max`，`flink-core/.../NettyShuffleEnvironmentOptions.java:133`，用于限制每个 gate 必需的 buffer 数。）

2.x 的调优思路也随之改变：**不再手动调 buffer 个数，而是开启 Buffer Debloating 自动调整 buffer 大小**（第 6 节）。

### 3.3 ★ 一个下游慢，整个上游 Task 都会被背压

`LocalBufferPool.requestMemorySegment()`（`:393`）：

```java
if (++subpartitionBuffersCount[targetChannel] == maxBuffersPerChannel) {    // :411
    unavailableSubpartitionsCount++;
}
```

`shouldBeAvailable()`（`:514-517`）：

```java
return !availableMemorySegments.isEmpty() && unavailableSubpartitionsCount == 0;
```

也就是说，**只要有任何一个 subpartition 占用的 buffer 达到 10 个，整个 LocalBufferPool 就变为不可用**，`RecordWriter.isAvailable()` 返回 false，上游 Task 停止处理数据，**哪怕发往其他下游的通道完全空闲**。

这是有意为之的设计：如果不做这个限制，一个慢的下游会把池子里所有的 buffer 都占满，而且 Task 线程无法预知下一条记录要发往哪个下游，所以只能整体暂停。这也是**数据倾斜**会拖垮整个作业的原因之一：一个热点 key 所在的下游变慢，所有上游都会被它拖住。

---

## 4. 读取：数据怎么进入下游

### 4.1 InputGate

`NET/partition/consumer/SingleInputGate.java`

- `inputChannelsWithData`（`:161`）：一个**有数据的 channel 队列**。channel 收到数据时调用 `notifyChannelNonEmpty()`（`:1210`）→ `queueChannel()`（`:1261`）把自己加入队列
- 下游 Task 读数据：01 篇的 `AbstractStreamTaskNetworkInput.emitNext()` → `CheckpointedInputGate.pollNext()`（02 篇 5.1 节，在这里处理 Barrier）→ `SingleInputGate.pollNext()`（`:866`）→ `getChannel()`（`:1330`）从队列里取出一个 channel → `channel.getNextBuffer()`

这个队列同时也是**优先级队列**（`PrioritizedDeque`）：02 篇的非对齐 Barrier 就是作为优先级元素插到队首的。

### 4.2 本地通道：`LocalInputChannel`

上下游在同一个 TM 时，`LocalInputChannel` 直接调用 `partitionManager.createSubpartitionView()`（`NET/partition/consumer/LocalInputChannel.java:183`）拿到上游 subpartition 的视图，`getNextBuffer()`（`:272`）直接从上游队列里取 buffer：

- **零拷贝**：下游直接读取上游写入的那块内存
- **没有 credit**：背压是天然的。buffer 在下游处理完之前不会被回收，上游池子就会被慢慢耗尽（3.3 节的机制）

### 4.3 远程通道：`RemoteInputChannel`

跨 TM 时，数据要经过 Netty。下游用 `RemoteInputChannel` 接收：

1. **建立连接**：`requestSubpartitions()`（`NET/partition/consumer/RemoteInputChannel.java:211`）→ `connectionManager.createPartitionRequestClient()`（`:222`，同一对 TM 之间的连接会被复用）→ `requestSubpartition()`（`:229`）→ 发送 `PartitionRequest` 消息（`NET/netty/NettyPartitionRequestClient.java:128`）
2. **上游 Netty Server** 收到后：`PartitionRequestServerHandler.channelRead0()`（`NET/netty/PartitionRequestServerHandler.java:73`）→ `msgClazz == PartitionRequest.class`（`:80`）→ 为这个 channel 创建一个 `CreditBasedSequenceNumberingViewReader`，它包装了 subpartition 的读视图
3. 之后上游就可以通过 `BufferResponse` 消息把 buffer 发过来，**前提是有 credit**

---

## 5. ★ Credit-based 流控

<figure class="ai-figure"><img src="/bigdata-img/flink/04-network-1.webp" alt="信用令牌控制数据发送" width="960" height="640" loading="lazy" /><figcaption>信用令牌控制数据发送<span>AI 生成配图</span></figcaption></figure>

### 5.1 为什么需要 credit？

Flink 1.5 之前用的是 **TCP 自身的流控**：下游处理不过来时 TCP 窗口被填满，上游写不进 socket。问题在于，**同一对 TM 之间的多个逻辑通道共用一条 TCP 连接**（4.3 节第 1 步）。一个慢通道占满 TCP 窗口，会把同一连接上的其他通道也一起堵住（队头阻塞），Checkpoint Barrier 也过不去。

Credit-based 流控（FLINK-7282，1.5 引入）改为：**下游告诉上游"我还有几个空 buffer"（credit），上游只发这么多**。数据在发出之前就被限流，TCP 连接上永远不会积压，各个逻辑通道之间互不影响。

### 5.2 下游：credit 从哪来

`RemoteInputChannel`：

| 步骤 | 行号 | 做什么 |
|---|---|---|
| 初始化 | `:156`、`:201` | `initialCredit = 2`（独占 buffer 数），`bufferManager.requestExclusiveBuffers(initialCredit)` |
| 收到 buffer | `onBuffer()`，`:590` | 放进 `receivedBuffers` 队列，通知 gate 有数据（`notifyChannelNonEmpty`），然后读取消息里附带的 **backlog**（上游在这个 subpartition 里还积压了几个 buffer） |
| 根据 backlog 申请浮动 buffer | `onSenderBacklog()`，`:582-583` | `bufferManager.requestFloatingBuffers(backlog + initialCredit)`：从 gate 的浮动 buffer 里再要一些 |
| buffer 被处理完、回收 | `notifyBufferAvailable()`，`:452` | 空 buffer 数累加到 `unannouncedCredit`；**从 0 变成非 0 时**才调用 `notifyCreditAvailable()`（`:398`），避免每回收一个都发一条消息 |
| 发送 credit | `NettyPartitionRequestClient.notifyCreditAvailable()`，`:217` | 发送 `AddCreditMessage` → `NettyMessage.AddCredit`（`NettyMessage.java:761`），携带 `getAndResetUnannouncedCredit()`（`:514`）的值 |

**浮动 buffer 的意义**：独占 buffer 只有 2 个，保证每个 channel 至少能收数据；浮动 buffer 是 gate 内各个 channel **共享**的 8 个，按 backlog 分给积压多的 channel。这样，数据多的通道可以多拿，数据少的通道也不会占用太多。浮动 buffer 不够时，channel 会把自己注册为 buffer 监听者（`BufferManager.java:191`），等有 buffer 被回收时再分配给它。

### 5.3 上游：有 credit 才发

**记账**：`CreditBasedSequenceNumberingViewReader`（`NET/netty/CreditBasedSequenceNumberingViewReader.java`）

```java
private int numCreditsAvailable;                         // :81

public void addCredit(int creditDeltas) { ... }          // :149  收到 AddCredit 时累加

public AvailabilityWithBacklog getAvailabilityAndBacklog() {
    return subpartitionView.getAvailabilityAndBacklog(numCreditsAvailable > 0);   // :193
}

public BufferAndAvailability getNextBuffer() {           // :255
    ...
    if (next.buffer().isBuffer() && --numCreditsAvailable < 0) {                   // :258
        throw new IllegalStateException("no credit available");
    }
}
```

**★ 判断能否发送**：`PipelinedSubpartition.getAvailabilityAndBacklog()`（`NET/partition/PipelinedSubpartition.java:608`）

```java
if (isCreditAvailable) {
    isAvailable = isDataAvailableUnsafe();               // 有 credit：有可发的数据就行（2.3 节）
} else {
    isAvailable = getNextBufferTypeUnsafe().isEvent();   // 没 credit：只有队首是【事件】时才能发
}
```

**事件不需要 credit**：只有数据 buffer 会消耗 credit（上面的 `isBuffer()` 判断）。所以 Checkpoint Barrier、EndOfPartition 这类事件，只要排到了队首，即使下游没有空 buffer 也能发出去。但对齐 Barrier 排在数据**后面**时，还是得等前面的数据发完，这就是 02 篇里"背压导致对齐慢"的根源。

**发送循环**：`PartitionRequestQueue`（`NET/netty/PartitionRequestQueue.java`）

- subpartition 有新数据时调用 `notifyReaderNonEmpty()`（`:84`，由 `CreditBasedSequenceNumberingViewReader.notifyDataAvailable()` 在 `:295` 调用）
- `enqueueAvailableReader()`（`:104`）：**先检查可用性（含 credit），不可用就不入队**（`:111`）
- `writeAndFlushNextMessageIfPossible()`（`:297`）：Netty 的 channel 不可写时直接返回（`:298`）；否则从可用 reader 队列里取一个（`:310`），读一个 buffer（`:319`），封装成 `BufferResponse`（`:339`，**顺带附上 backlog**），发送。还有更多数据就把 reader 放回队尾（`:334-335`），各个 channel 轮流发送，保证公平
- 收到 `AddCredit`：`PartitionRequestServerHandler`（`:112-115`）→ `addCreditOrResumeConsumption()`（`PartitionRequestQueue.java:165`）→ 累加 credit，reader 重新变为可用就入队

### 5.4 背压的完整传导链

把第 2 ~ 5 节串起来，以示例中的 `source → keyBy → slow-map` 为例：

```
① slow-map 处理慢（每条 sleep 1ms）
② slow-map 的 InputGate 中，buffer 迟迟不被回收 → 没有空 buffer → 不再发 AddCredit
   （本地通道：没有 credit 这一层，buffer 直接留在上游队列里不被回收）
③ 上游 reader 的 numCreditsAvailable = 0 → 数据 buffer 发不出去，堆积在 PipelinedSubpartition 里
④ 某个 subpartition 堆积到 10 个 buffer（maxBuffersPerChannel）→ LocalBufferPool 变为不可用
⑤ Source 的 RecordWriter.isAvailable() = false
⑥ StreamTask.processInput() 暂停默认动作，只处理邮件 → 记入 softBackPressuredTime
⑦ Source 不再从 SourceReader 拉取数据（FLIP-27 的 Source 由 mailbox 驱动，自然就停了）
```

如果 Source 前面还有 Kafka，那么 Flink 不拉取，Kafka 的消费 lag 就会上涨，**背压最终体现为消费延迟**。

---

## 6. Buffer Debloating（FLIP-183）

<figure class="ai-figure"><img src="/bigdata-img/flink/04-network-2.webp" alt="缩小缓冲减少排队延迟" width="960" height="640" loading="lazy" /><figcaption>缩小缓冲减少排队延迟<span>AI 生成配图</span></figcaption></figure>

### 6.1 问题

背压时，上下游之间**所有的 buffer 都是满的**。第 8 节的示例中，每个下游 subtask 大约有 (2 × 2 + 8) 个输入 buffer，加上上游 2 个 subpartition 各最多 10 个输出 buffer，每个 32KB，合计约 1MB 的 in-flight 数据。下游每秒只能处理约 1000 条 × 270 字节 ≈ 270KB，所以一个对齐 Barrier 要**排队 3 秒以上**才能被处理到。

buffer 越多、越大，吞吐缓冲越好，但 **in-flight 数据越多，Checkpoint 对齐越慢，非对齐 Checkpoint 越大**。

### 6.2 思路：让 in-flight 数据量刚好等于"1 秒能处理完的量"

开启 `taskmanager.network.memory.buffer-debloat.enabled: true`（`flink-core/.../TaskManagerOptions.java:492`）后：

1. **定时触发**：`StreamTask.scheduleBufferDebloater()`（`RT/streaming/runtime/tasks/StreamTask.java:968`）按 `buffer-debloat.period`（`TaskManagerOptions.java:464`）注册定时器 → `debloat()`（`:1000`）→ 每个 InputGate 调用 `triggerDebloating()`（`SingleInputGate.java:514`）
2. **计算新的 buffer 大小**：`BufferDebloater.recalculateBufferSize()`（`RT/runtime/throughput/BufferDebloater.java:99`）

   ```java
   long desiredTotalBufferSizeInBytes = (currentThroughput * targetTotalTime) / MILLIS_IN_SECOND;
   int newSize = bufferSizeEMA.calculateBufferSize(desiredTotalBufferSizeInBytes, actualBuffersInUse);
   ```

   - `currentThroughput`：`ThroughputCalculator` 统计的**实际消费速率**（字节/秒）
   - `targetTotalTime`：`buffer-debloat.target`，默认 **1 秒**（`TaskManagerOptions.java:483-485`）
   - 新大小 = 期望总字节数 ÷ 在用的 buffer 数，再做一次指数移动平均（EMA）平滑
   - 变化小于阈值（`buffer-debloat.threshold-percentages`，`:504`）时不更新（`skipUpdate()`），避免频繁抖动
3. **通知上游**：`SingleInputGate.announceBufferSize()`（`:505`）→ 每个 channel：
   - 远程通道：发送 `NewBufferSize` 消息（`NettyMessage.java:920`）→ 上游 `PartitionRequestServerHandler`（`:126-129`）→ `PartitionRequestQueue.notifyNewBufferSize()`（`:186`）
   - 本地通道：直接设置
   - 最终到达 `PipelinedSubpartition.bufferSize()`（`:646`）
4. **上游按新大小写 buffer**：`BufferWritingResultPartition.addToSubpartition()` 把 BufferBuilder **截短**（`buffer.trim(...)`，`BufferWritingResultPartition.java:366`）。内存块仍然是 32KB，但只写到新的大小就封口

注意：**buffer 的个数没变，变的是每个 buffer 实际装多少数据**。所以它能减少 in-flight 数据量，又不需要重新分配内存。

---

## 7. 背压指标与瓶颈定位

### 7.1 三个时间指标

每个 Task 每秒被划分成三部分（单位：ms/s）：

| 指标 | 含义 | 来源 |
|---|---|---|
| `idleTimeMsPerSecond` | 没有输入数据可处理 | `StreamTask.processInput()` 中 `!inputProcessor.isAvailable()`（01 篇 6.3 节） |
| `backPressuredTimeMsPerSecond` | 输出被阻塞（软 + 硬） | 2.4 节 |
| `busyTimeMsPerSecond` | 真正在干活 | **不是测出来的**，而是 `1000 - idle - backPressured`（`TaskIOMetricGroup.java:281-283`） |

`idle` 和 `backPressured` 用的都是 `TimerGauge`（`RT/runtime/metrics/TimerGauge.java:37`），会在一个时间窗口内累计测量值（默认 60 秒，`:39`）。

### 7.2 定位瓶颈的方法

**沿着数据流找"第一个不被背压、但很忙的算子"**，它就是瓶颈：

```
Source          backPressured≈1000   ← 被背压：不是它的问题，它在等下游
  ↓
slow-map        backPressured≈0, busy≈1000   ← ★ 瓶颈：自己不被背压，但一直在忙
  ↓
Sink            idle 高
```

辅助指标：
- `Shuffle.Netty.Output.Buffers.outPoolUsage` ≈ 1：上游输出池满了，说明下游消费不过来
- `Shuffle.Netty.Input.Buffers.inPoolUsage` ≈ 1：下游输入池满了，说明它自己处理不过来
- 两者都高：瓶颈在这个 Task 或者更下游；上游 outPool 满但下游 inPool 不满：可能是网络本身的问题

> 注意：瓶颈算子和它下游**链在一起**（在同一个 Task 里）时，指标反映的是整个 Task。可以临时调用 `disableChaining()` 把它拆开，再看具体是哪个算子。

---

## 8. 实验

### 8.1 示例作业

`BackpressureDemo.java`：

```
source(2 并行度，共 20000 条/秒，每条约 270B) ──keyBy──► slow-map(每条 sleep 1ms) → sink
```

- MiniCluster 启动 **2 个 TaskManager**（`minicluster.number-of-taskmanagers`，`TaskManagerOptions.java:744-745`），每个 1 个 slot。所以 4 条逻辑通道中，2 条在同一个 TM 内（`LocalInputChannel`），2 条跨 TM（`RemoteInputChannel` + Netty + credit）
- REST 端口固定为 18081，主线程每 5 秒通过 REST 接口拉取一次指标
- 每 5 秒做一次对齐 Checkpoint
- 下游处理能力约 1000 条/秒/并行度，输入约 10000 条/秒/并行度，所以会产生**稳定的背压**

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.network.BackpressureDemo 42
```

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.network.BackpressureDemo 42 debloat
```

运行期间也可以直接用 curl 查询 REST 接口，比如 `http://localhost:18081/jobs/<jobId>/vertices/<vertexId>/subtasks/metrics`。

### 8.2 实测：稳定后的指标（各 subtask 平均值）

**不开 Debloating**：

```
Source: source        backPressured=1000 busy=0 idle=6 outPoolUsage=1
slow-map -> Sink      backPressured=0 busy=1000 idle=0 inPoolUsage=1 exclusiveUsage=1 floatingUsage=1
                      bufsInLocal/s=2.47 bufsInRemote/s=2.64
latest checkpoint #9: end_to_end_duration=3083ms
```

**开启 Debloating**：

```
Source: source        backPressured=1000 busy=0 idle=6 outPoolUsage=1
slow-map -> Sink      backPressured=0 busy=1000 idle=0 inPoolUsage=0.95 exclusiveUsage=1 floatingUsage=0.94
                      bufsInLocal/s=10.56 bufsInRemote/s=10.71
                      debloatedBufferSize=10641.50 estimatedTimeToConsumeBuffersMs=981
latest checkpoint #9: end_to_end_duration=1128ms
```

Checkpoint 耗时变化（两次运行）：

| Checkpoint | #1 | #2 | #3 | #4 | #5 | #6 | #7 | #8 | #9 |
|---|---|---|---|---|---|---|---|---|---|
| 不开 Debloating | 157ms | 3432ms | 3302ms | 3131ms | 3206ms | 3117ms | 3220ms | 3142ms | 3083ms |
| 开启 Debloating | 113ms | 665ms | 900ms | 910ms | 1091ms | 1112ms | 1109ms | 1149ms | 1128ms |

### 8.3 解读

1. **瓶颈定位**：Source `backPressured=1000`，slow-map `backPressured=0, busy=1000`，完全符合 7.2 节的判断方法。Source 的 `outPoolUsage=1`、slow-map 的 `inPoolUsage=1`，也印证了 5.4 节的传导链：下游输入 buffer 全满 → 不发 credit → 上游输出 buffer 全满
2. **本地通道和远程通道各占一半**：`bufsInLocal/s` 与 `bufsInRemote/s` 基本相等。这两个值是速率类指标，刚启动时会逐渐爬升，看比例即可
3. **Checkpoint 耗时从约 3.2 秒降到约 1.1 秒**：
   - 不开 Debloating：6.1 节估算约 1MB 的 in-flight 数据 ÷ 约 270KB/s 的处理速度 ≈ 3.8 秒，与实测的 3.1 ~ 3.4 秒同一量级
   - 开启后：buffer 从 32KB 缩到约 **10.6KB**（`debloatedBufferSize`），`estimatedTimeToConsumeBuffersMs` ≈ **981ms**，正好接近目标值 1 秒（6.2 节）。Checkpoint 耗时随之稳定在 1.1 秒左右
4. **buffer 变小后，buffer 的数量速率上升了约 4 倍**（`bufsIn/s` 从约 2.5 升到约 10.6），而记录的吞吐量没有变化（仍然受限于 slow-map），这正是"个数不变、每个装得更少"的结果
5. **Debloating 是渐进收敛的**：debloatedBufferSize 从 6660 → 8374 → 10641 逐步稳定下来，Checkpoint 耗时也是从 665ms 逐渐变为 1.1 秒左右。这说明它不是简单地"尽量缩小"，而是让 in-flight 数据量**逼近 1 秒的处理量**

> 🐛 **踩坑记录**：写监控代码时，REST 接口 `/subtasks/metrics?get=a,b,c` 一直返回 `[]`。调试后发现，**只要 get 参数里有一个指标不存在，整个请求就返回空数组**，而不会跳过那个指标。Debloat 相关的指标只在开启 Debloating 时才注册，而且名字里带着 gate 编号（`Shuffle.Netty.Input.0.debloatedBufferSize`，见 `SingleInputGateFactory.java:190`、`:213`）。示例改成了先列出这个算子实际有哪些指标，再只查询存在的那些。
>
> 另外，作业刚启动时 Source 会先显示 `busy=1000`，大约 15 秒后才变为 `backPressured=1000`。这是因为 busy 是由 `1000 - idle - backPressured` 推算出来的，计时指标还没积累到数据时，busy 就会显示为 1000。

---

## 9. 断点清单

用 IDEA 运行 `BackpressureDemo`，参数填 `600`（跑 10 分钟方便调试）。**务必使用 Suspend: Thread**。

| # | 线程 | 位置 | 看什么 |
|---|---|---|---|
| 1 | Source Task | `BufferWritingResultPartition.java:165` `emitRecord` | 记录写进哪个 subpartition |
| 2 | Source Task | `LocalBufferPool.java:411` | subpartition 占用的 buffer 数达到 10 时 |
| 3 | Source Task | `StreamTask.java:687` | `recordWriter.isAvailable()` 变为 false：软背压开始 |
| 4 | OutputFlusher | `RecordWriter.java:160` `flushAll` | 每 100ms 一次 |
| 5 | Netty server 线程 | `PartitionRequestQueue.java:319` | 有 credit 时读取 buffer |
| 6 | Netty server 线程 | `PipelinedSubpartition.java:608` | `isCreditAvailable` 为 false 时，只有事件能发 |
| 7 | Netty client 线程 | `RemoteInputChannel.java:590` `onBuffer` | backlog 的值 |
| 8 | 下游 Task | `RemoteInputChannel.java:452` `notifyBufferAvailable` | credit 从 0 变成非 0 时 |
| 9 | Netty server 线程 | `PartitionRequestQueue.java:165` `addCreditOrResumeConsumption` | 上游收到 credit |
| 10 | 下游 Task（debloat 模式） | `BufferDebloater.java:99` | 吞吐量、新的 buffer 大小 |

## 10. 课后练习

1. **buffer timeout**：把 `env.setBufferTimeout(0)` 和 `env.setBufferTimeout(-1)` 分别加到示例里（先去掉 slow-map 的 sleep），比较 `bufsIn/s` 与延迟。`-1` 表示什么？读 `RecordWriter.java:89-100`
2. **倾斜实验**：把 `keyBy(t -> t.f0 % 128)` 改成 `keyBy(t -> t.f0 % 128 == 0 ? 0L : t.f0 % 128)`，让一部分数据集中到同一个 key 上，并把 sleep 只加在这个 key 上。观察 3.3 节"一个下游慢，拖住整个上游"的现象
3. **硬背压**：让 Source 后面的 flatMap 一次输出 1000 条记录，看看 `hardBackPressuredTimeMsPerSecond` 什么时候不为 0，以及 overdraft buffer 起了什么作用（在 `LocalBufferPool.java:450` 打断点）
4. **非对齐 + Debloating**：在示例中同时开启非对齐 Checkpoint 和 Debloating，对比 02 篇的数据，说明 Debloating 对非对齐 Checkpoint 的**大小**有什么影响
5. **思考题**：为什么 `RemoteInputChannel.notifyBufferAvailable()` 只在 `unannouncedCredit` 从 0 变成非 0 时才发送 AddCredit？如果每回收一个 buffer 就发一次，会怎样？

## 11. 下一篇预告

**05：Table / SQL**。一条 SQL 从 `executeSql()` 开始，经过 Calcite 解析与校验、逻辑优化（`FlinkStreamProgram` 中的各个优化阶段）、物理计划、ExecNode，再到代码生成，最后变成 01 篇中那张 Transformation 图。重点剖析 Group Aggregate 的 changelog 语义，以及 MiniBatch 和 Two-Phase 聚合优化。
:::
