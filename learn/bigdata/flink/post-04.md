---
title: "Flink 2.x 源码精读（四）：一个下游慢，另一个下游也跟着停了"
description: "两个下游子任务，只让一个变慢，另一个本来每 5 秒能处理 3600 万条，结果只处理了几千条，全程空等。顺着这个现象，读懂 Flink 2.3 的网络缓冲区、Credit 流控、背压的传导链，以及 Buffer Debloating 为什么能把 Checkpoint 从 2.8 秒降到 1 秒。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-04-cover.webp"}]]
---

# Flink 2.x 源码精读（四）：一个下游慢，另一个下游也跟着停了

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-04-cover.webp" alt="Yui和Kai在数据流网络中观察慢下游拖住全链路的场景" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在数据流网络中观察慢下游拖住全链路的场景<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`NET` = `flink-runtime/src/main/java/org/apache/flink/runtime/io/network`，`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 前置阅读：第一讲下篇（`processInput()` 与 Web UI 三个指标）、第二讲（Barrier 排队）

---

先看一个实验。

一个 Source，交替发出两个 key：key 0 和 key 4。它们经过 `keyBy` 后，分别落在下游的两个子任务上，子任务 1 只处理 key 0，子任务 0 只处理 key 4。

两次运行只有一个区别：第二次让 key 0 的每条数据都 `sleep` 1 毫秒。本机实测，t=20 秒时每 5 秒处理的条数，以及从 REST 接口查到的指标：

| | 子任务 0（只处理 key 4，**从来不慢**） | 子任务 1（只处理 key 0） | Source |
|---|---|---|---|
| 两个 key 都不慢 | 34323011 条 | 34323252 条 | 几乎不被背压 |
| 只让 key 0 慢 | **4681 条**，`idle=1000` | 4035 条，`busy=999` | `backPressured=1000` |

子任务 0 的代码一行没改，它每 5 秒本来能处理三千多万条，现在只处理了四千多条，**剩下的时间全在空等**（`idle=1000` 表示每秒 1000 毫秒都是空闲的）。

**一个下游慢了，另一个下游也跟着停了。** 这就是数据倾斜会拖垮整个作业的原因。

这一篇，我们看数据是怎么在 Task 之间流动的：

1. 一条数据从上游算子发出，要经过哪些数据结构才能到达下游？
2. 网络缓冲区是怎么分配的？为什么一个慢的下游能拖住整个上游？
3. **Credit 流控**是怎么工作的？背压是怎么一级一级传回 Source 的？
4. Web UI 上的指标怎么看？怎么定位瓶颈？
5. Buffer Debloating 调整的是什么？为什么它能把 Checkpoint 从 2.8 秒降到 1 秒？

---

## 一、全景图

```
上游 Task                                                    下游 Task
算子 output.collect()
 └ RecordWriter.emit()              序列化：[4 字节长度][数据]
    └ 按分区器选择下游                keyBy 用的是第三讲的 KeyGroup 算法
       └ 写进缓冲区（BufferBuilder，从 LocalBufferPool 申请）
          └ 子分区（PipelinedSubpartition）的队列
                 │
   同一个 TM ────┼──► LocalInputChannel：直接读上游的队列，不经过网络
                 │
   不同 TM ──────┴──► Netty 服务端 ──BufferResponse──► Netty 客户端 → RemoteInputChannel
                       （有 credit 才发）  ◄──AddCredit──   （处理完腾出缓冲区，就发新的 credit）
                                                          SingleInputGate（汇总所有输入通道）
                                                           └ 处理 Barrier（第二讲）
                                                             └ 反序列化 → processElement()
```

上下游的概念是一一对应的：

| 上游 | 下游 | 说明 |
|---|---|---|
| `ResultPartition` | `InputGate` | 一个 Task 的一份输出 / 一份输入 |
| `ResultSubpartition`（子分区） | `InputChannel`（输入通道） | 上游第 i 个子分区，对应下游第 i 个子任务的一个输入通道 |

上游有 M 个并行度、下游有 N 个时，`keyBy` 会产生 **M × N 条逻辑通道**。

---

## 二、写入：一条数据怎么变成缓冲区

### 2.1 序列化

`RecordWriter.emit()`（`NET/api/writer/ChannelSelectorRecordWriter.java:54`）先用分区器选出目标子分区，然后调用 `NET/api/writer/RecordWriter.java:108`：

```java
targetPartition.emitRecord(serializeRecord(serializer, record), targetSubpartition);
```

序列化格式很简单（`:146`）：先写 4 个字节的长度，再写数据。**一条数据可以跨越多个缓冲区**，下游按长度把它拼回来。

### 2.2 写进缓冲区

`BufferWritingResultPartition.emitRecord()`（`NET/partition/BufferWritingResultPartition.java:165`）把序列化后的数据写进当前子分区正在用的缓冲区。写满了就封口，再申请一个新的继续写。

一个细节：缓冲区在**第一次写入时**就已经挂进了子分区的队列（`:310`），写和读共享同一块内存，**边写边可读**。所以"写满"并不是发送的前提。

### 2.3 什么时候发出去

`PipelinedSubpartition.isDataAvailableUnsafe()`（`NET/partition/PipelinedSubpartition.java:623-626`）：

```java
return !isBlocked && (flushRequested || getNumberOfFinishedBuffers() > 0);
```

满足任一条件，数据就可以被读走：
1. 有**写满封口**的缓冲区；
2. 有人调用了 **`flush()`**（`:689`），没写满的缓冲区也可以读。

谁来调用 `flush()`？每个 RecordWriter 都有一个 **OutputFlusher 线程**（`RecordWriter.java:258`），每隔 `execution.buffer-timeout.interval` 调用一次，**默认 100 毫秒**（`flink-core/.../configuration/ExecutionOptions.java:106-110`）。

这就是 Flink **延迟与吞吐的权衡**：间隔越小，延迟越低，但每个缓冲区装的数据越少，开销越大。

---

## 三、缓冲区从哪来：为什么一个慢的下游能拖住整个上游

### 3.1 两级缓冲池

```
TaskManager 进程
└ NetworkBufferPool：全局一个，切成一块块 32KB 的内存段
   ├ LocalBufferPool：每个 ResultPartition 一个（输出）
   └ LocalBufferPool：每个 InputGate 一个（输入）
```

内存段的大小由 `taskmanager.memory.segment-size` 决定，默认 **32KB**（`flink-core/.../configuration/TaskManagerOptions.java:202-204`）。

### 3.2 2.x 的变化：每个通道用几个缓冲区，已经写死了

Flink 1.x 有四个经典的调优参数：`taskmanager.network.memory.buffers-per-channel`、`floating-buffers-per-gate`、`max-buffers-per-channel`、`max-overdraft-buffers-per-gate`。网上大量的调优文章至今还在讲它们。

**在 2.3.0 里，这四个配置项都已经删除了**，源码中一次也搜不到。其中浮动缓冲区和独占缓冲区那两项，是 FLINK-33750（2024-09-18）删除的。它们变成了代码里的常量（`RT/runtime/taskmanager/NettyShuffleEnvironmentConfiguration.java`）：

| 含义 | 值 | 行号 |
|---|---|---|
| 每个输入通道的**独占**缓冲区 | 2 | `:325` |
| 每个 InputGate 额外的**浮动**缓冲区（gate 内各通道共享） | 8 | `:326` |
| 每个输出子分区**最多**占用的缓冲区 | 10 | `:332` |
| 每个 gate 最多可以**透支**的缓冲区 | 20 | `:401` |

> 注意：官方文档里仍有多处提到这些已删除的配置项。（勘误 2026-10-04：原写"搜到了 1~3 处"，按文件重新统计：`network_mem_tuning.md` 中英文各 7 行、`adaptive_batch.md` 各 1 行、`batch_shuffle.md` 各 2 行。）如果你按老文章去配置它们，不会生效，而且**不会报错**（实测见踩坑实验室 #21）。

2.x 的调优思路也随之改变：**不再手动调缓冲区的个数，而是开启 Buffer Debloating，自动调整缓冲区的大小**（第八节）。

### 3.3 关键规则：一个子分区满了，整个缓冲池就不可用

`LocalBufferPool.requestMemorySegment()`（`NET/buffer/LocalBufferPool.java:393`）：

```java
if (++subpartitionBuffersCount[targetChannel] == maxBuffersPerChannel) {    // :411
    unavailableSubpartitionsCount++;
}
```

判断缓冲池是否可用（`:514`）：

```java
return !availableMemorySegments.isEmpty() && unavailableSubpartitionsCount == 0;
```

也就是说，**只要有任何一个子分区占用的缓冲区达到 10 个，整个输出缓冲池就变成"不可用"**。上游 Task 随即停止处理新数据（第一讲下篇的 `processInput()`），哪怕发往其他下游的通道完全空闲。

这是有意为之的设计：如果不加这个限制，一个慢的下游会占满池子里所有的缓冲区；而 Task 在处理下一条数据之前，并不知道它要发往哪个下游，所以只能整体暂停。

这就是开头实验里看到的现象：慢的子任务 1 对应的子分区很快堆满了 10 个缓冲区，整个 Source 停了下来，快的子任务 0 也就收不到数据了。

> 说明：开头的实验中，两个 key 是交替出现的。即使没有这条规则，Source 每隔一条也要写一次慢 key，同样会被卡住。所以实验证明的是**现象**：慢的下游拖住了上游，快的下游跟着空等；**机制**则以上面这段源码为准。

### 3.4 顺便回答第二讲下篇的一个问题

第二讲下篇做非对齐 Checkpoint 实验时，被 Barrier 越过的数据，大部分记在两个 **Source** 名下（每个约 26~30 万字节），下游只有 6~12KB。为什么？

看上面那张表就明白了：**输出侧**每个子分区最多可以攒 10 个缓冲区；而**输入侧**每个通道只有 2 个独占缓冲区，外加整个 gate 共享的 8 个浮动缓冲区。有背压时，下游的输入缓冲区很快被占满，就不再向上游要数据（下一节的 credit），于是数据主要堆在上游的输出队列里。

---

## 四、Credit 流控：下游说能收多少，上游才发多少

<figure class="ai-figure"><img src="/bigdata-img/flink/post-04-1.webp" alt="下游用空缓冲区向上游发出信用额度控制数据发送" width="960" height="640" loading="lazy" /><figcaption>下游用空缓冲区向上游发出信用额度控制数据发送<span>AI 生成配图</span></figcaption></figure>

### 4.1 为什么需要 credit

同一对 TaskManager 之间的多条逻辑通道，**共用一条 TCP 连接**。如果靠 TCP 自身的流控，一条慢的通道塞满了 TCP 的窗口，同一条连接上的其他通道也会被一起堵住，Checkpoint Barrier 也过不去。

Credit 流控换了一个思路：**下游告诉上游"我还有几个空缓冲区"（这就是 credit），上游最多只发这么多。** 数据在发出去之前就被限流了，TCP 连接上永远不会积压，各条通道互不影响。

### 4.2 下游：credit 从哪里来

`NET/partition/consumer/RemoteInputChannel.java`：

| 步骤 | 行号 | 做什么 |
|---|---|---|
| 初始化 | `:156`、`:201` | 初始 credit = 2（独占缓冲区的个数），并申请这 2 个缓冲区 |
| 收到数据 | `:590`（`onBuffer()`） | 放进接收队列，同时读出消息里附带的 **backlog**：上游这个子分区还积压了几个缓冲区 |
| 按积压量要浮动缓冲区 | `:582`（`onSenderBacklog()`） | 上游积压得多，就从 gate 共享的浮动缓冲区里多要一些 |
| 数据处理完、缓冲区回收 | `:452`（`notifyBufferAvailable()`） | 把新的空缓冲区记为"还没通知的 credit"；**从 0 变成非 0 时**才通知上游（`:398`），避免每回收一个就发一条消息 |

**独占缓冲区**保证每个通道至少能收数据；**浮动缓冲区**按积压量分给最需要的通道。这样，数据多的通道可以多拿，数据少的通道也不会白占。

### 4.3 上游：有 credit 才发

上游为每个下游通道维护一个 credit 计数（`NET/netty/CreditBasedSequenceNumberingViewReader.java:81`）：收到 credit 就累加（`:149`），每发出一个**数据**缓冲区就减 1（`:258`）。

能不能发，由 `PipelinedSubpartition.getAvailabilityAndBacklog()` 决定（`PipelinedSubpartition.java:608-620`）：

```java
if (isCreditAvailable) {
    isAvailable = isDataAvailableUnsafe();               // 有 credit：有数据就能发
} else {
    isAvailable = getNextBufferTypeUnsafe().isEvent();   // 没 credit：只有队首是"事件"时才能发
}
```

**事件不需要 credit。** Checkpoint Barrier 这类事件，只要排到了队首，即使下游没有空缓冲区也能发出去。

但第二讲上篇讲过：对齐的 Barrier 不能超车，它**排在数据后面**。前面的数据缓冲区要消耗 credit，没有 credit 就发不出去，Barrier 也只能跟着等。**这就是"背压下 Barrier 在路上排队 3 秒"在网络层的根源。**

真正的发送循环在 `NET/netty/PartitionRequestQueue.java`：只有可发送的通道才会进入发送队列（`:104-111`）；发送时从队列里取出一个通道，读一个缓冲区，包装成 `BufferResponse`，**顺带附上积压量**（`:339`）；收到下游的 credit 后，通道重新变为可发送（`:165`）。

### 4.4 背压的完整传导链

把前面几节串起来：

```
① 下游算子处理慢
② 下游的输入缓冲区迟迟不被回收 → 没有空缓冲区 → 不再给上游发 credit
   （本地通道没有 credit 这一层，缓冲区直接留在上游的队列里不被回收）
③ 上游没有 credit → 数据缓冲区发不出去，堆在子分区的队列里
④ 某个子分区堆到 10 个缓冲区 → 整个输出缓冲池变为不可用（3.3 节）
⑤ 上游 Task 的 processInput() 发现输出不可用 → 暂停处理新数据，只处理邮件 → 计入背压时间
⑥ 如果上游是 Source，它就不再从外部系统拉取数据
```

如果 Source 读的是 Kafka，Flink 不拉取，Kafka 的消费延迟（lag）就会上涨。**背压最终体现为消费延迟。**

---

## 五、软背压与硬背压

第一讲下篇说过，Web UI 上的 BackPressured 是"软背压"和"硬背压"之和（`RT/runtime/metrics/groups/TaskIOMetricGroup.java:240-243`）。它们的区别：

| | 什么时候发生 | 后果 |
|---|---|---|
| **软背压** | 处理下一条数据**之前**，发现输出缓冲池不可用 | Task 线程暂停处理数据，但**仍然能处理邮件**：Checkpoint、定时器照常进行 |
| **硬背压** | 处理一条数据的**过程中**，需要新的缓冲区却申请不到（比如一条很大的数据跨了好几个缓冲区，或者 flatMap 一次输出很多条） | Task 线程**被阻塞**在申请缓冲区上（`BufferWritingResultPartition.java:449` 起），连邮件都处理不了，Checkpoint 也会被卡住 |

为了缓解硬背压，Flink 允许在缓冲池耗尽时，每个 gate 从全局池里**透支**最多 20 个缓冲区（`LocalBufferPool.java:450`），让正在处理的那条数据先写完，然后回到软背压状态。

---

## 六、看懂指标，定位瓶颈

### 6.1 三个时间指标

每个 Task 的每一秒，被划分成三部分：

| 指标 | 含义 |
|---|---|
| `idleTimeMsPerSecond` | 没有输入数据可处理 |
| `backPressuredTimeMsPerSecond` | 输出被阻塞（软 + 硬） |
| `busyTimeMsPerSecond` | 真正在处理数据 |

注意：**busy 不是测出来的，而是算出来的**：`1000 - min(idle + backPressured, 1000)`（`TaskIOMetricGroup.java:281-283`）。所以作业刚启动、计时指标还没积累起数据时，busy 会显示为 1000。我做开头的实验时，前两轮的指标里所有子任务都是 `busy=1000`，到第 20 秒才显示出真实的 `idle=1000`。

### 6.2 定位方法：找"第一个不被背压、但很忙的算子"

用一个更典型的例子：Source 并行度 2、每秒共产生 2 万条数据，经过 `keyBy` 交给一个每条数据 `sleep` 1 毫秒的 `slow-map`（每个并行度每秒最多处理约 1000 条）。本机实测，稳定后的指标（各子任务的平均值）：

```
Source: source        backPressured=1000 busy=0   idle=9   outPoolUsage=1
slow-map -> Sink      backPressured=0    busy=998 idle=2   inPoolUsage=1 exclusiveUsage=1 floatingUsage=1
```

沿着数据流往下看：
- **Source**：`backPressured=1000`，它一直在等下游，不是它的问题；
- **slow-map**：`backPressured=0`、`busy≈1000`，自己不被背压，但一直在忙。**它就是瓶颈。**

辅助指标也印证了第四节的传导链：Source 的**输出池**用满了（`outPoolUsage=1`），slow-map 的**输入池**也用满了（`inPoolUsage=1`，独占和浮动缓冲区都满了）。

> 注意：如果瓶颈算子和它的下游**链在同一个 Task 里**，指标反映的是整个 Task。可以临时调用 `disableChaining()` 把它们拆开，再看具体是哪个算子。

---

## 七、背压与 Checkpoint：为什么 Barrier 要排队这么久

还是 6.2 节那个作业，每 5 秒做一次对齐 Checkpoint。本机实测，第 2~9 次 Checkpoint 的耗时：

```
3618ms  2701ms  2742ms  2761ms  2973ms  2912ms  2839ms  2784ms
```

和第二讲上篇一样，都在 3 秒左右。原因也一样：**Barrier 排在所有缓冲区里的数据后面，要等它们全部被处理完。** 缓冲区越多、越大，Barrier 前面排的数据就越多，等得就越久。

那能不能让缓冲区里的数据少一点？

---

## 八、Buffer Debloating：让缓冲区里的数据刚好够处理 1 秒

<figure class="ai-figure"><img src="/bigdata-img/flink/post-04-2.webp" alt="动态缩小缓冲区让检查点屏障快速穿过数据流" width="960" height="640" loading="lazy" /><figcaption>动态缩小缓冲区让检查点屏障快速穿过数据流<span>AI 生成配图</span></figcaption></figure>

### 8.1 思路

开启 `taskmanager.network.memory.buffer-debloat.enabled`（**默认关闭**，`TaskManagerOptions.java:492-494`）后，Flink 会根据下游**实际的处理速度**，动态调整每个缓冲区装多少数据，让在途的数据量刚好等于"一段目标时间内能处理完的量"。

### 8.2 实现

1. **定时计算**：每隔 `buffer-debloat.period`（默认 200 毫秒，`:464-466`），Task 对每个 InputGate 触发一次计算（`RT/streaming/runtime/tasks/StreamTask.java:968`、`:1000` → `NET/partition/consumer/SingleInputGate.java:514`）；
2. **算出新的大小**（`RT/runtime/throughput/BufferDebloater.java:99`）：

   ```java
   long desiredTotalBufferSizeInBytes = (currentThroughput * targetTotalTime) / MILLIS_IN_SECOND;
   int newSize = bufferSizeEMA.calculateBufferSize(desiredTotalBufferSizeInBytes, actualBuffersInUse);
   ```

   - `currentThroughput`：实际的消费速度（字节/秒）；
   - `targetTotalTime`：`buffer-debloat.target`，**默认 1 秒**（`TaskManagerOptions.java:483-485`）；
   - 新的大小 = 期望的总字节数 ÷ 正在使用的缓冲区个数，再做一次平滑；变化不到 25% 就不更新，避免频繁抖动（`:504-506`）；
3. **通知上游**：`SingleInputGate.announceBufferSize()`（`:505`），远程通道通过一条网络消息，本地通道直接设置，最终到达上游的子分区（`PipelinedSubpartition.java:646`）；
4. **上游按新的大小写**：缓冲区写到新的大小就封口（`BufferWritingResultPartition.java:366`，`buffer.trim(...)`）。

注意：**缓冲区的个数没有变，内存块也还是 32KB，变的是每个缓冲区实际装多少数据。**

### 8.3 实测

同一个作业，开启 Debloating 后（本机实测）：

```
slow-map -> Sink   ... debloatedBufferSize=11162 estimatedTimeToConsumeBuffersMs=1070.50
```

Checkpoint 耗时（第 1~9 次）：

| | #1 | #2 | #3 | #4 | #5 | #6 | #7 | #8 | #9 |
|---|---|---|---|---|---|---|---|---|---|
| 不开启 | 276 | 3618 | 2701 | 2742 | 2761 | 2973 | 2912 | 2839 | 2784 |
| 开启 | 112 | 606 | 962 | 1004 | 985 | 994 | 999 | 981 | 1004 |

（单位：毫秒）

解读：
1. 缓冲区从 32KB 缩到了约 **11KB**（`debloatedBufferSize`），预估"把缓冲区里的数据处理完"需要约 **1070 毫秒**，正好接近默认目标 1 秒；
2. Checkpoint 耗时随之稳定在 **1 秒左右**，因为 Barrier 前面只剩下约 1 秒的数据；
3. 它是**逐步收敛**的：第 2 次 606ms，第 3 次 962ms，之后稳定。它不是"尽量缩小"，而是让在途数据量**逼近 1 秒的处理量**；
4. 缓冲区变小后，每秒收到的缓冲区个数从约 2.7 个升到了约 11.7 个，而处理的数据条数没有变。这正是"个数不变、每个装得更少"的结果。

**Debloating 没有消除背压**，Source 依然是 `backPressured=1000`，它只是减少了在途的数据量。所以它和第二讲下篇的非对齐 Checkpoint 是互补的：Debloating 让 Barrier 前面排的数据变少，非对齐让 Barrier 直接跳过这些数据。

---

## 九、自己动手

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/network/` 下：

```bash
# 开头的实验：一个下游慢，另一个下游会怎样（各运行 20 秒）
./run.sh study.network.SkewDemo even
./run.sh study.network.SkewDemo skew

# 第六~八节：背压指标与 Buffer Debloating（各运行 42 秒，每 5 秒打印一次指标）
./run.sh study.network.BackpressureDemo 42
./run.sh study.network.BackpressureDemo 42 debloat
```

`BackpressureDemo` 启动了 2 个 TaskManager，所以 4 条逻辑通道里，2 条在同一个 TaskManager 内（本地通道），2 条要跨 TaskManager（远程通道 + credit）。

**断点清单**（Suspend 改成 Thread）：

| # | 线程 | 位置 | 看什么 |
|---|---|---|---|
| 1 | 上游 Task | `BufferWritingResultPartition.java:165` | 数据写进哪个子分区 |
| 2 | 上游 Task | `LocalBufferPool.java:411` | 某个子分区占满 10 个缓冲区 |
| 3 | OutputFlusher | `PipelinedSubpartition.java:689` | 每 100 毫秒 flush 一次 |
| 4 | Netty 服务端 | `PipelinedSubpartition.java:612` | 没有 credit 时，只有事件能发 |
| 5 | Netty 客户端 | `RemoteInputChannel.java:590` | 上游附带的积压量 |
| 6 | 下游 Task | `RemoteInputChannel.java:452` | credit 从 0 变成非 0 |
| 7 | Netty 服务端 | `PartitionRequestQueue.java:165` | 上游收到 credit |
| 8 | 下游 Task（debloat） | `BufferDebloater.java:99` | 吞吐量和新的缓冲区大小 |

---

## 十、课后练习

1. **buffer timeout**：去掉 `slow-map` 里的 `sleep`，分别加上 `env.setBufferTimeout(0)` 和 `env.setBufferTimeout(-1)`，对比每秒收到的缓冲区个数。`-1` 表示什么？（读 `RecordWriter.java:89-100`）
2. **倾斜实验进阶**：把开头实验的 Source 改成先连续发 1000 条 key 4，再发 1 条 key 0，如此循环。快的子任务还会被拖住吗？结合 3.3 节的规则解释。
3. **非对齐 + Debloating**：在第二讲的 `CheckpointDemo` 里同时开启非对齐 Checkpoint 和 Debloating，非对齐 Checkpoint 的**大小**会怎么变？为什么？
4. **思考**：为什么下游只在 credit 从 0 变成非 0 时才通知上游？如果每回收一个缓冲区就发一次，会怎样？

---

## 写在最后

这一讲的核心可以用一句话概括：**数据能不能发，由下游说了算（credit）；上游能不能继续处理，由最慢的那个下游说了算（10 个缓冲区的限制）。**

这也让前面几讲的几个现象都有了网络层的解释：
- 第二讲上篇：背压下 Barrier 要排队 3 秒，因为它排在等待 credit 的数据后面；
- 第二讲下篇：被越过的数据大多在 Source 那边，因为输出侧能攒的缓冲区比输入侧多；
- 本篇开头：一个慢的下游，让另一个下游也跟着停了。

下一讲，我们离开 DataStream，看 **Table / SQL**：一条 SQL 是怎么变成第一讲那张 Transformation 图的？为什么同一个 `GROUP BY`，有时候会输出 `-U` 和 `+U` 两条结果？


**留一个问题**：你遇到过"一个热点 key 拖垮整个作业"的情况吗？最后是怎么解决的？

下一讲：**Flink 2.x 源码精读（五）：Table / SQL，从 SQL 到 Transformation**
:::
