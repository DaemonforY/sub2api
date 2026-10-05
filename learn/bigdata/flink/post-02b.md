---
title: "Flink 2.x 源码精读（二·下）：非对齐 Checkpoint 把 3 秒压到 16 毫秒，多出来的 600KB 是什么？"
description: "同样的背压，开启非对齐 Checkpoint 后耗时从 3 秒降到 16 毫秒，Checkpoint 却从 5.7KB 涨到 600KB。拆开来看，这 600KB 几乎全部是被 Barrier 越过的、堆在 Source 输出缓冲区里的数据。读懂 Flink 2.3 非对齐 Checkpoint 的实现、代价和推荐配置。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-02b-cover.webp"}]]
---

# Flink 2.x 源码精读（二·下）：非对齐 Checkpoint 把 3 秒压到 16 毫秒，多出来的 600KB 是什么？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02b-cover.webp" alt="Yui和Kai观察非对齐Checkpoint越过数据并保存飞散数据" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai观察非对齐Checkpoint越过数据并保存飞散数据<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 上篇：《背压一来，Checkpoint 从 15 毫秒变成 3 秒，时间花在了哪？》

---

上篇的结论是：**背压下 Checkpoint 变慢，不是因为快照慢，而是 Barrier 排在积压的数据后面，在路上等了 3 秒。**

那么，能不能让 Barrier 不排队？

可以。只要在同一个作业上加一行配置，开启**非对齐 Checkpoint**（Unaligned Checkpoint）。本机实测，三种配置的对比如下（同一个作业、同样的背压，每 5 秒一次 Checkpoint，取第 2~6 次）：

| 配置 | 每次耗时 | 每次大小 |
|---|---|---|
| 对齐（默认） | 2705~3260 ms | 5714 B |
| **非对齐** | **16~17 ms** | **567~646 KB** |
| 先对齐，超过 1 秒切换为非对齐 | 1016~1147 ms | 309~419 KB |

耗时降到了原来的 1/200，大小却涨到了原来的 100 倍以上。**状态本身只有 5.7KB，多出来的六百多 KB 是什么？**

这一篇回答：

1. 非对齐 Checkpoint 是怎么让 Barrier "插队"的？
2. 多出来的这 600KB 究竟是什么，存在哪里？
3. "先对齐、超时再切换"是怎么实现的？为什么我推荐这种配置？
4. 非对齐 Checkpoint 有哪些代价和限制？

---

## 一、思路：不排队，但要把越过的数据记下来

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02b-1.webp" alt="Barrier越过数据并将未处理数据收入Checkpoint" width="960" height="640" loading="lazy" /><figcaption>Barrier越过数据并将未处理数据收入Checkpoint<span>AI 生成配图</span></figcaption></figure>

对齐 Checkpoint 慢，根本原因在于 Barrier 必须**排在数据后面**。在一条积压严重的通道里：

```
对齐：     [B][d5][d4][d3][d2][d1]  ──►  算子     B 要等 d1~d5 全部处理完，才轮到它
非对齐：   [d5][d4][d3][d2][d1][B]  ──►  算子     B 直接跳到最前面（队首在靠近算子的一侧）
```

但 Barrier 一旦越过 d1~d5，问题就来了：上篇讲过，Checkpoint 的切面是"Barrier 之前的数据都已处理，之后的都没处理"。现在 d1~d5 原本在 Barrier 前面，却还没有被处理。如果直接做快照，这几条数据就丢了。

**解决办法：把被 Barrier 越过的数据，也一起存进 Checkpoint。** 这部分数据叫 **channel state**（通道状态），也就是"还在路上的数据"（in-flight data）。

恢复时，先把 channel state 里的数据重新放回输入、输出通道，再继续处理。这样，结果和对齐模式完全一致，依然是精确一次（exactly-once）。

【配图 1：对齐与非对齐的对比。上半部分：Barrier 排在一串数据后面，旁边的时钟显示"等待 3 秒"；下半部分：Barrier 跳到最前面，被越过的数据被复制一份装进一个标着"channel state"的箱子，箱子进入 Checkpoint】

---

## 二、实现：两个地方要记录数据

<figure class="ai-figure"><img src="/bigdata-img/flink/post-02b-2.webp" alt="Barrier优先穿过输出队列并记录被越过的数据" width="960" height="640" loading="lazy" /><figcaption>Barrier优先穿过输出队列并记录被越过的数据<span>AI 生成配图</span></figcaption></figure>

一条数据从上游 Task 到下游 Task，会经过两个地方：**上游的输出队列**，以及**下游的输入通道**。Barrier 在这两个地方都可能越过数据，所以两边都要记录。

### 2.1 输出侧：Barrier 插到队首，复制后面的数据

开启非对齐后，上篇讲过的 Step (2)"向下游广播 Barrier"，会把 Barrier 作为**优先级事件**发出：`broadcastEvent(barrier, isPriorityEvent)`（`RT/runtime/io/network/api/writer/RecordWriter.java:129`）。这种 Buffer 的类型是 `PRIORITIZED_EVENT_BUFFER`（`RT/runtime/io/network/buffer/Buffer.java:292`）。

上游的输出队列收到它后（`RT/runtime/io/network/partition/PipelinedSubpartition.java:218` 的 `processPriorityBuffer`）：

```java
buffers.addPriorityElement(...);                 // :219  插到队列最前面
...
while (iterator.hasNext()) {                     // :231  遍历排在它后面、还没发出去的数据
    ...
    inflightBuffers.add(bc.build());             // :236  复制一份
}
channelStateWriter.addOutputData(barrier.getId(), subpartitionInfo, ..., inflightBuffers);   // :241
```

**Barrier 插到队首，排在它后面的数据被复制一份，交给 `ChannelStateWriter` 写进 Checkpoint。** 原来的数据照常发给下游，不受影响。

### 2.2 输入侧：第一个 Barrier 到达就立即做快照

下游 Task 收到**第一个**非对齐 Barrier 时，不再等其他通道（`RT/streaming/runtime/io/checkpointing/AlternatingWaitingForFirstBarrierUnaligned.java:59`）：

1. 通知所有输入通道："从现在开始，记录收到的数据"；
2. **立即做快照**，同时把 Barrier 发给自己的下游。

此后，其他通道在**它们的** Barrier 到达之前收到的数据，都会被记录下来（`RT/runtime/io/network/partition/consumer/RemoteInputChannel.java:714` 的 `checkpointStarted`，以及 `:647` 的 `channelStatePersister.maybePersist(buffer)`）。等所有通道的 Barrier 都到齐，就停止记录。

两侧记录下来的数据，都由 `ChannelStateWriter` 异步写入 Checkpoint 存储，并且**计入 Checkpoint 的大小**：Task 状态的大小，是算子状态加上通道状态（`RT/runtime/checkpoint/OperatorSubtaskState.java:149-161`）。

### 2.3 Source 的 Checkpoint 也要"插队"

还有一个细节。Source 的 Checkpoint 是通过 RPC 触发、以邮件的形式进入 Mailbox 的（上篇第四节）。对于非对齐 Checkpoint，这封邮件会被标记为**紧急**，插到邮箱的最前面（`RT/streaming/runtime/tasks/StreamTask.java:1309-1312`）：

```java
MailboxExecutor.MailOptions mailOptions =
        CheckpointOptions.AlignmentType.UNALIGNED == checkpointOptions.getAlignment()
                ? MailboxExecutor.MailOptions.urgent()
                : MailboxExecutor.MailOptions.options();
```

非对齐的目标就是"尽快"，所以从邮箱开始，就一路插队。

---

## 三、拆开那 600KB

和上篇一样，我在实验结束前通过 REST 接口取出了最近一次 Checkpoint 的分项数据，这次多加了两列：**persisted**（这个子任务持久化了多少通道数据）和 **unaligned**（这个子任务是否以非对齐方式完成）。

**非对齐 + 背压**（本机实测，第 6 次 Checkpoint；REST 接口给出的总耗时是 10ms，日志里的 16ms 还包含 JobManager 的收尾工作；大小 585832B）：

```
vertex                   sub   start_delay  alignment     sync    async   end_to_end  persisted  unaligned
Source: datagen          0             3ms        0ms      0ms      5ms          9ms    298273B       true
Source: datagen          1             3ms        0ms      0ms      5ms         10ms    263065B       true
count-per-key -> checkpo 0             3ms        0ms      0ms      5ms         10ms      6255B       true
count-per-key -> checkpo 1             8ms        0ms      0ms      0ms         10ms     12541B       true
```

四行的 persisted 加起来约 580KB，和这次 Checkpoint 的总大小 585832 字节基本吻合，剩下的就是原本那几 KB 的状态。

再看分布：

- **两个 Source 各持久化了 26~30 万字节**，占了绝大部分；
- **下游的计数算子只有 6~12KB**。

为什么大头在 Source？因为有背压时，下游处理不过来，数据主要**堆在上游的输出队列里**，等着下游腾出空间。下游的输入通道只能接收有限的数据（由 credit 流控决定，第四讲会讲），所以在那里被越过的数据很少。Barrier 插队时，越过的主要是 Source 输出队列里的那几百 KB。

**所以多出来的 600KB，就是被 Barrier 越过的、还堆在 Source 输出缓冲区里的数据。** 每条数据约 1KB，大约是五六百条。背压越重、缓冲区越大，这部分就越大。

---

## 四、折中方案：先对齐，超时再切换

非对齐 Checkpoint 快，但 Checkpoint 变大了，恢复时也要多回放一部分数据。有没有两全的办法？

Flink 提供了一个配置：`execution.checkpointing.aligned-checkpoint-timeout`（`flink-core/.../configuration/CheckpointingOptions.java:565`）。开启非对齐后再设置它，Checkpoint 会**先按对齐的方式进行，超过这个时间还没对齐完，再切换为非对齐**。

它的**默认值是 0**（`:567`），源码里的说明是"If timeout is 0, checkpoints will always start unaligned"，也就是一开始就走非对齐。

### 4.1 怎么切换

切换在两端都有：

**下游**：输入侧使用的是一个"可切换"的状态机（`SingleCheckpointBarrierHandler.alternating()`，在 `RT/streaming/runtime/io/checkpointing/InputProcessorUtil.java:155-164` 中选择）。先按对齐的方式收集 Barrier，超时后调用 `alignedCheckpointTimeout()`（`AlternatingCollectingBarriers.java:41`），切换为非对齐。

**上游**：Source 发出的是一种"可超时"的对齐 Barrier，同时登记一个待完成的"输出侧通道数据"（`PipelinedSubpartition.java:273-280`）。上篇讲过的 Step (3) 注册了一个定时器（`RT/streaming/runtime/tasks/SubtaskCheckpointCoordinatorImpl.java:346-347`，实现在 `:380-405`）。定时器到期时，如果 Barrier 还没发出去，就把它**改成优先级 Barrier、插到队首**，并把被它越过的数据收集起来（`PipelinedSubpartition.java:329-356`）。

### 4.2 实测

**先对齐、1 秒后切换 + 背压**（本机实测，第 6 次 Checkpoint，总耗时 1126ms，大小 408857B）：

```
vertex                   sub   start_delay  alignment     sync    async   end_to_end  persisted  unaligned
Source: datagen          0             2ms        0ms      0ms   1081ms       1126ms    197517B       true
Source: datagen          1             2ms        0ms      0ms   1080ms       1125ms    196781B       true
count-per-key -> checkpo 0           153ms        0ms      0ms    932ms       1125ms         0B      false
count-per-key -> checkpo 1          1024ms       25ms     14ms     37ms       1125ms      8861B       true
```

这张表信息量很大：

1. **同一次 Checkpoint 里，有的子任务对齐完成，有的切换成了非对齐。** 计数算子的子任务 0，Barrier 在 153ms 就到齐了，没有超时，`unaligned=false`，没有持久化任何通道数据；子任务 1 的 start_delay 是 1024ms，正好是 1 秒超时之后，`unaligned=true`。
2. **Source 的 async 是 1080ms。** 这不是在上传状态，而是在**等那 1 秒超时**：Source 的异步阶段必须等"输出侧通道数据"准备好（`RT/streaming/api/operators/OperatorSnapshotFinalizer.java:68-69`），而它要到超时、Barrier 被改成优先级之后才能确定。
3. **耗时被限制在约 1 秒，大小约 40 万字节**，介于纯对齐和纯非对齐之间。为什么持久化的数据变少了？我的理解是：超时之前的 1 秒里，Barrier 一直在队列里往前挪，排在它前面的数据被下游处理掉了一部分，超时插队时越过的数据也就少了。这是根据源码逻辑做的推断，没有单独做实验验证。

**这就是我推荐的生产配置：开启非对齐，同时设置一个对齐超时。** 没有背压时，Checkpoint 走对齐，又小又快；背压严重时，自动切换为非对齐，耗时有上限。超时设多少，取决于你能接受的 Checkpoint 耗时。

```java
env.getCheckpointConfig().enableUnalignedCheckpoints();
env.getCheckpointConfig().setAlignedCheckpointTimeout(Duration.ofSeconds(1));
```

对应的配置项是 `execution.checkpointing.unaligned.enabled`（`CheckpointingOptions.java:539`，默认 `false`）和 `execution.checkpointing.aligned-checkpoint-timeout`。

---

## 五、代价与限制

| | 说明 | 依据 |
|---|---|---|
| **Checkpoint 变大** | 多出来的就是 in-flight 数据，背压越重越大。本例中比状态本身大了 100 倍 | 第三节实测 |
| **恢复变慢** | 恢复时要先把通道数据回放一遍，再继续处理 | 源码结论，本期没有实测恢复耗时 |
| **只支持 EXACTLY_ONCE** | 在 AT_LEAST_ONCE 模式下开启会直接抛异常："Cannot use unaligned checkpoints with AT_LEAST_ONCE checkpointing mode" | `InputProcessorUtil.java:119-124` |
| **Savepoint 永远是对齐的** | "Savepoints can not be unaligned" | `RT/runtime/checkpoint/CheckpointOptions.java:78` |
| **部分连接方式会被强制对齐** | 比如 pointwise 连接，Barrier 的对齐类型会被标记为 `FORCED_ALIGNED` | `SubtaskCheckpointCoordinatorImpl.java:324-329` |

还有一点很重要：**非对齐 Checkpoint 只是让 Checkpoint 不再被背压拖慢，并没有解决背压本身。** 作业依然处理不过来，数据依然在堆积。它是止痛药，不是解药。真正的解决办法是找到处理慢的算子，或者增加资源，第四讲会讲怎么找。

---

## 六、自己动手

实验代码和上篇是同一个：`flink-notes/demos/src/main/java/study/checkpoint/CheckpointDemo.java`，每次运行 30 秒：

```bash
./run.sh study.checkpoint.CheckpointDemo slow             # 对齐 + 背压
./run.sh study.checkpoint.CheckpointDemo slow unaligned   # 非对齐 + 背压
./run.sh study.checkpoint.CheckpointDemo slow mixed       # 先对齐，超过 1 秒切换为非对齐
```

**断点清单**（Suspend 记得改成 Thread）：

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `StreamTask.java:1309` | 非对齐时邮件被标记为紧急 |
| 2 | `PipelinedSubpartition.java:219` | Barrier 插到输出队列最前面 |
| 3 | `PipelinedSubpartition.java:241` | 被越过的数据交给 ChannelStateWriter |
| 4 | `AlternatingWaitingForFirstBarrierUnaligned.java:59` | 下游收到第一个 Barrier 就做快照 |
| 5 | `RemoteInputChannel.java:647` | 输入侧记录 in-flight 数据 |
| 6 | `PipelinedSubpartition.java:329` | mixed 模式下，超时后把 Barrier 改成优先级 |
| 7 | `AlternatingCollectingBarriers.java:41` | 下游超时后切换为非对齐 |

---

## 七、课后练习

1. **调整超时**：把 mixed 模式的对齐超时从 1 秒改成 3 秒，Checkpoint 耗时和大小会怎么变？如果改成 5 秒呢？（提示：上篇实测，纯对齐时需要 2.7~3.3 秒）
2. **减轻背压**：把实验里的 `sleep(5)` 改成 `sleep(1)`，非对齐 Checkpoint 的大小会怎么变？为什么？
3. **读代码**：第三节说，有背压时数据主要堆在上游的输出队列里。下游输入通道最多能接收多少数据，是由什么决定的？（提示：第四讲会讲的 credit 流控，以及 `RemoteInputChannel` 里的 exclusive buffer 和 floating buffer）
4. **思考**：既然非对齐这么快，为什么 Savepoint 不允许非对齐？（提示：Savepoint 常用于升级作业、修改并行度，想想恢复时 channel state 要还原到哪里）

---

## 写在最后

第二讲到这里就结束了。

| 上篇：对齐 | 下篇：非对齐 |
|---|---|
| Barrier 排在数据后面，不超车 | Barrier 插到队首，越过数据 |
| 先到的通道暂停接收，等所有 Barrier 到齐 | 第一个 Barrier 到达就做快照 |
| Checkpoint 小，但背压下很慢 | 背压下也很快，但要额外存储被越过的数据 |
| 一次超时就可能全局重启（默认配置） | 推荐：开启非对齐 + 设置对齐超时 |

下一讲，我们把目光从"什么时候做快照"转向"快照里存的是什么"：**State Backend**。HashMap 和 RocksDB 两种状态后端的数据结构有什么不同？改了并行度，状态是怎么切分、重新分配的？Flink 2.x 的存算分离状态后端 ForSt 又是什么？


**留一个问题**：你的生产作业开启了非对齐 Checkpoint 吗？对齐超时设的是多少？

下一讲：**Flink 2.x 源码精读（三）：State Backend 与扩缩容**
:::
