---
title: "Flink 2.x 源码精读（八·下）：Sink 与端到端 exactly-once"
description: "每秒 10 条，Checkpoint 每 2 秒一次，用默认配置的 FileSink 写文件，第一个文件直到第 120 秒才出现，最早的数据等了 120 秒。换成 OnCheckpointRollingPolicy，最多 2.1 秒。顺着这个现象，读懂 Flink 2.3 Sink V2 的两阶段提交与端到端 exactly-once。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-08b-cover.webp"}]]
---

# Flink 2.x 源码精读（八·下）：Sink 与端到端 exactly-once

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-08b-cover.webp" alt="Yui和Kai协作完成流式写入与事务提交" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai协作完成流式写入与事务提交<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`CORE` = `flink-core/src/main/java/org/apache/flink`，`RT` = `flink-runtime/src/main/java/org/apache/flink`，`FILES` = `flink-connectors/flink-connector-files/src/main/java/org/apache/flink/connector/file/sink`，`FSC` = `flink-connectors/flink-file-sink-common/src/main/java/org/apache/flink/streaming/api/functions/sink/filesystem`
> 前置阅读：第八讲上篇（Source）、第二讲上篇（Checkpoint 的触发顺序）；踩坑实验室 #06、#07、#08

---

先看一个实验。

用 Flink 自带的 FileSink 写文件，并行度 1，每秒 10 条，**Checkpoint 每 2 秒一次**。每条数据带上它的生成时间。另起一个线程每 200ms 扫一次输出目录，记录每个正式文件第一次出现的时刻，算出每条数据从生成到"能被看到"等了多久。

两次运行只差一行配置：

```java
// 版本 A：什么都不设
FileSink.forRowFormat(new Path(OUT), new SimpleStringEncoder<String>("UTF-8")).build();

// 版本 B：加一个滚动策略
FileSink.forRowFormat(new Path(OUT), new SimpleStringEncoder<String>("UTF-8"))
        .withRollingPolicy(OnCheckpointRollingPolicy.build())
        .build();
```

**版本 A**（本机实测，运行 150 秒，完整日志在 `assets/logs/filesink-default.log`）：

```text
>>> t=  0.2s  committed files=0, records visible=0, hidden (in-progress/pending) files=1
>>> t= 10.2s  committed files=0, records visible=0, hidden (in-progress/pending) files=1
...
>>> t=110.6s  committed files=0, records visible=0, hidden (in-progress/pending) files=1
>>> t=120.4s  new committed file part-85129103-6e69-4e40-b823-eb93d57a05d2-0: 1201 records, visible after 0.2s ~ 120.2s
...
>>> mode=default: records visible=1201, latency min=0.2s median=60.2s max=120.2s
```

**版本 B**（运行 30 秒，`assets/logs/filesink-oncheckpoint.log`）：

```text
>>> t=  2.2s  new committed file part-97ae4380-79b5-4a23-91d6-affd9e16a1c7-0: 20 records, visible after 0.1s ~ 2.0s
>>> t=  4.3s  new committed file part-97ae4380-79b5-4a23-91d6-affd9e16a1c7-1: 20 records, visible after 0.2s ~ 2.0s
>>> t=  6.3s  new committed file part-97ae4380-79b5-4a23-91d6-affd9e16a1c7-2: 20 records, visible after 0.2s ~ 2.1s
...
>>> mode=oncheckpoint: records visible=281, latency min=0.0s median=1.1s max=2.1s
```

**版本 A：Checkpoint 做了约 60 次，前 120 秒一个正式文件都没有，最早的数据等了 120 秒。版本 B：每 2 秒一个文件，最多等 2.1 秒。**

这一篇，我们来看：

1. Sink V2 的两阶段提交，两个阶段分别发生在哪一刻？
2. 为什么 Checkpoint 做了，文件却没提交？
3. Source 的进度和 Sink 的事务，是怎么在同一个 Checkpoint 里对上的？这就是端到端 exactly-once。
4. 恢复、重试、不开 Checkpoint、作业结束，这几种情况下分别会怎样？

---

## 一、Sink V2：按能力组合接口

`CORE/api/connector/sink2/` 下的接口，一个 Sink 实现了哪些，就说明它有哪些能力：

| 接口 | 能力 | 关键方法 |
|---|---|---|
| `Sink` | 最基础：只有 Writer | `createWriter`（`Sink.java:47`） |
| `SupportsWriterState` + `StatefulSinkWriter` | Writer 有自己的状态（比如文件写到一半） | `restoreWriter`、`snapshotState` |
| `SupportsCommitter` + `CommittingSinkWriter` | **两阶段提交** | `prepareCommit()`（`CommittingSinkWriter.java:38`）、`createCommitter`（`SupportsCommitter.java:51`） |
| `Committer` | 提交 | `commit(...)`（`Committer.java:46`） |

**Committable** 是 Writer 交给 Committer 的"待提交凭证"：Kafka 里是事务 ID 加 producer 信息，FileSink 里是"已经写完、等待改名的文件"。它会进入 Checkpoint，所以必须能序列化。

`sinkTo(sink)` 翻译成 StreamGraph 时，由 `SinkTransformationTranslator`（`RT/streaming/runtime/translators/SinkTransformationTranslator.java:139` 的 `expand()`）按实现的接口展开成两个算子：`Writer`（`SinkWriterOperator`）和 `Committer`（`CommitterOperator`）。默认两者并行度相同、forward 连接，所以会链在一起。上篇示例的 Task 名字就是：

```text
Source: range-source -> Map -> Sink: Writer -> Sink: Committer
```

---

## 二、两阶段提交

<figure class="ai-figure"><img src="/bigdata-img/flink/post-08b-1.webp" alt="两阶段提交中写入、交出与提交分离" width="960" height="640" loading="lazy" /><figcaption>两阶段提交中写入、交出与提交分离<span>AI 生成配图</span></figcaption></figure>

### 2.1 第一阶段：prepareCommit，在 Barrier 之前

`RT/streaming/runtime/operators/sink/SinkWriterOperator.java`：

```java
public void prepareSnapshotPreBarrier(long checkpointId) throws Exception {   // :188
    super.prepareSnapshotPreBarrier(checkpointId);
    if (!endOfInput) {
        sinkWriter.flush(false);
        emitCommittables(checkpointId);                                       // :192
    }
}
```

`prepareSnapshotPreBarrier` 是第二讲上篇讲过的"Barrier 往下游发之前"的那一步。Writer 在这里调用 `prepareCommit()`，把这个 Checkpoint 的 committable 交出去，发给下游的 Committer。

所以 Checkpoint N 的 committable 一定在 Barrier N **之前**到达 Committer，并进入 Committer 在 Checkpoint N 里的状态（`CommitterOperator.java:144`，`snapshotState` 把所有还没提交的 committable 存进状态）。

### 2.2 第二阶段：commit，在 Checkpoint 完成之后

`RT/streaming/runtime/operators/sink/CommitterOperator.java`：

```java
public void notifyCheckpointComplete(long checkpointId) throws Exception {   // :159
    ...
    commitAndEmitCheckpoints(Math.max(lastCompletedCheckpointId, checkpointId));
}
```

**只有 Checkpoint N 全局完成后，才提交 N 及之前的 committable**（`commitAndEmitCheckpoints`，`:164`）。在此之前，外部系统里的数据处于"已写入、未提交"的状态：Kafka 的 `read_committed` 消费者看不到它们，FileSink 的文件是以 `.` 开头的隐藏文件。



### 2.3 那么版本 A 为什么不提交？

关键在于：**`prepareCommit` 交出什么，是 Writer 自己决定的。**

FileSink 的 Writer 在 `prepareCommit` 时，只交出"已经关闭的文件"。正在写的文件要不要关闭，由滚动策略决定（`FILES/writer/FileWriterBucket.java:196-203`）：

```java
List<FileSinkCommittable> prepareCommit(boolean endOfInput) throws IOException {
    if (inProgressPart != null
            && (rollingPolicy.shouldRollOnCheckpoint(inProgressPart) || endOfInput)) {
        ...
        closePartFile();
    }
    ...
}
```

两种策略的 `shouldRollOnCheckpoint`：

```java
// 版本 A：DefaultRollingPolicy（Row 格式的默认值，FileSink.java:364）
public boolean shouldRollOnCheckpoint(PartFileInfo<BucketID> partFileState) throws IOException {
    return partFileState.getSize() > partSize;          // FSC/rollingpolicies/DefaultRollingPolicy.java:72-74，partSize 默认 128MB
}

// 版本 B：OnCheckpointRollingPolicy（继承自 CheckpointRollingPolicy）
public boolean shouldRollOnCheckpoint(PartFileInfo<BucketID> partFileState) {
    return true;                                        // FSC/rollingpolicies/CheckpointRollingPolicy.java:30-32
}
```

**版本 A 下，文件不到 128MB，Checkpoint 时就不关闭，`prepareCommit` 什么也不交，Committer 也就没东西可提交。** Checkpoint 照常完成，只是和这个文件无关。

那它什么时候关？`DefaultRollingPolicy` 还有按处理时间滚动的条件（`DefaultRollingPolicy.java:83-87`）：文件创建满 60 秒（`DEFAULT_ROLLOVER_INTERVAL`，`:50`），或者 60 秒没有新数据（`DEFAULT_INACTIVITY_INTERVAL`，`:48`）。而检查这个条件的定时器，默认**每 60 秒**才跑一次（`FILES/FileSink.java:297`，`DEFAULT_BUCKET_CHECK_INTERVAL`）。

### 2.4 为什么是 120 秒，不是 60 秒

检查定时器在 Writer 初始化时就注册了（`FILES/writer/FileWriter.java:175`），之后每次触发再注册下一次（`:267-279`）：

```java
private void registerNextBucketInspectionTimer() {
    final long nextInspectionTime =
            processingTimeService.getCurrentProcessingTime() + bucketCheckInterval;
    processingTimeService.registerTimer(nextInspectionTime, this);
}
```

而文件的创建时间，是第一条数据写进来的那一刻，比定时器注册**稍晚**。所以按代码推断：

1. 第 60 秒左右，第一次检查：文件的"年龄"还差一点不满 60 秒，不滚动；
2. 第 120 秒左右，第二次检查：满了，关闭文件；
3. 下一个 Checkpoint：`prepareCommit` 交出这个文件，Checkpoint 完成后改名为正式文件。

实验结果和这个推断吻合：第一个正式文件出现在 120.4 秒，里面有 1201 条，最早一条等了 120.2 秒。

> 这个"差一点"是按代码推断的，没有打印定时器和文件创建的精确时间。如果第一条数据和定时器注册几乎同时，有可能在第 60 秒就滚动。不同的机器、不同的启动速度，结果可能不一样。

**结论**：用 Row 格式的 FileSink 时，如果数据量不大、又希望下游尽快看到，要么用 `OnCheckpointRollingPolicy`，要么把 `DefaultRollingPolicy` 的滚动时间和 `withBucketCheckInterval` 调小。代价是小文件更多。

> Bulk 格式（Parquet 等）默认就是 `OnCheckpointRollingPolicy`（`FileSink.java:546`），没有这个问题。



---

## 三、端到端 exactly-once：Source 的进度与 Sink 的事务对上了

<figure class="ai-figure"><img src="/bigdata-img/flink/post-08b-2.webp" alt="Source进度与Sink事务在同一检查点对齐" width="960" height="640" loading="lazy" /><figcaption>Source进度与Sink事务在同一检查点对齐<span>AI 生成配图</span></figcaption></figure>

### 3.1 同一个 Checkpoint 里

回到上篇的 `SourceSinkDemo`。它的 Sink 是一个自己实现的两阶段提交 Sink：Writer 把数据攒成一个"事务"，`prepareCommit` 时交出，事务 ID 是 `s{subtask}-chk{n}`。

本机实测 `normal` 模式（节选，`assets/logs/sourcesink-normal.log`）：

```text
[+ 3.31s] ENUM     ...  snapshotState(chk 3), pending=[split-2[80,120), split-3[120,160)]
[+ 3.33s] WRITER   ...  subtask 1 prepareCommit -> s1-chk3 n=10 [2..11]
[+ 3.33s] READER   ...  subtask 1 snapshotState(chk 3) = [split-0[12,40)]
[+ 3.41s] COMMIT   ...  s1-chk3 n=10 [2..11] (retries=0)
```

同一个 subtask，同一个 Checkpoint 3：

- Sink 交出的事务是 `[2..11]`；
- Source 的状态是 `split-0[12,40)`，也就是"下一条要读 12"；
- 事务在约 80ms 后、Checkpoint 3 完成时提交。

**Source 读到哪里，Sink 就提交到哪里，两者在同一个 Checkpoint 里严丝合缝。** 这就是端到端 exactly-once 的核心。

注意顺序：`WRITER prepareCommit` 打印在 `READER snapshotState` **之前**。Writer 和 Reader 链在同一个 Task 里，`prepareSnapshotPreBarrier` 在算子做快照之前执行（2.1 节）。

### 3.2 一条记录的完整旅程

把第八讲上篇、本篇和第二讲串起来，一条记录要经过这些步骤，才算"恰好一次"地到达外部系统：

| 步骤 | 发生在 | 如果此时失败 |
|---|---|---|
| ① Reader 读出，Source 进度前进 | Task 线程 | 回滚到上一个 Checkpoint，Source 从旧进度重读 |
| ② Writer 写进当前事务（外部不可见） | Task 线程 | 事务随 Writer 丢弃，记录会被重读、重写 |
| ③ Checkpoint N：`prepareCommit` 交出事务 | Barrier 之前 | 同上 |
| ④ Checkpoint N：Source 进度、待提交的事务一起进状态 | 快照 | Checkpoint N 不完成，回滚到 N-1 |
| ⑤ Checkpoint N 完成 → `commit` | `notifyCheckpointComplete` | **提交结果不在任何 Checkpoint 里** → 第四节 |

第 ⑤ 步是整个链条里最微妙的一步：Checkpoint N 已经完成了，提交却发生在它**之后**。如果提交到一半作业挂了，从 Checkpoint N 恢复时，Flink 不知道这个事务到底提交了没有。



---

## 四、四种情况

### 4.1 恢复：再提交一次，所以 Committer 必须幂等

`CommitterOperator.initializeState()`（`CommitterOperator.java:120`）：

```java
// try to re-commit recovered transactions as quickly as possible
commitAndEmitCheckpoints(lastCompletedCheckpointId);                           // :138-139
```

从 Checkpoint N 恢复时，状态里保存的是"N 的快照时刻还没提交"的事务，其中就包括 N 自己的。Flink **一律再提交一次**。

所以：

- `commit` 必须能识别"这个事务已经提交过"，跳过它，并调用 `request.signalAlreadyCommitted()`（`Committer.java:100`）；
- 事务 ID 在重启前后必须**唯一**，通常带上 Checkpoint 编号。

上篇的故障实验里就有这一行：

```text
[+ 6.06s] COMMIT   ...  s1-chk5 n=12 [64..75] already committed -> signalAlreadyCommitted
```

不幂等会怎样，踩坑实验室 #06 实测过：去掉幂等检查后，同样的故障，`duplicates=13`。

Flink 自带的 FileSink 是幂等的：恢复后提交时调用 `commitAfterRecovery()`（`FILES/committer/FileCommitter.java:61-62`），会先检查文件状态，已经改过名就跳过。

### 4.2 提交失败：`retryLater()` 并不会"稍后"

`commit` 里可以调用 `request.retryLater()`（`Committer.java:87`）请求重试。但重试是在同一次 `notifyCheckpointComplete` 里**立即**进行的（`RT/streaming/runtime/operators/sink/committables/CheckpointCommittableManagerImpl.java:149` 的 `for` 循环），中间没有任何等待，最多 `sink.committer.retries` 次（默认 10，`CORE/configuration/SinkOptions.java:35`），用完就让作业失败。

踩坑实验室 #08 实测过：11 次调用，几毫秒内全部用完。如果外部系统需要时间恢复，要在 `commit()` 内部自己退避，或者直接让作业失败、交给重启策略（第六讲）。

### 4.3 不开 Checkpoint：无界作业永远不提交

两个阶段都挂在 Checkpoint 上：`prepareCommit` 在 Checkpoint 时调用，`commit` 在 Checkpoint 完成后调用。唯一的例外是输入结束：

```java
// CommitterOperator
public void endInput() throws Exception {                          // :151
    if (!isCheckpointingEnabled || isBatchMode) {
        commitAndEmitCheckpoints(Long.MAX_VALUE);                  // :154 没有 Checkpoint 可等，直接全部提交
    }
}
```

**无界的流作业不开 Checkpoint，事务型 Sink 一次都不会提交。** 踩坑实验室 #07 实测过：写入 160 条，提交 0 条。

### 4.4 作业结束：Final Checkpoint

开了 Checkpoint 的流作业，有界输入结束时，Writer 把最后一批数据归入"下一个" Checkpoint（`SinkWriterOperator.java:206-211`，`emitCommittables(lastKnownCheckpointId + 1)`），然后等 **Final Checkpoint** 完成后提交。这依赖 `execution.checkpointing.checkpoints-after-tasks-finish`，默认是 true（`CORE/configuration/CheckpointingOptions.java:615-617`，FLIP-147）。

上篇的 `normal` 模式（正文没有贴出，日志见 `assets/logs/sourcesink-normal.log` 最后一行 `total=160 distinct=160 expected=160 duplicates=0`），最后一批就是这样提交的。

---

## 五、自己动手

示例代码在 `flink-notes/demos`（本篇在 pom 里新增了 `flink-connector-files` 依赖）：

```bash
./run.sh study.connector.FileSinkLatencyDemo default 150
```

```bash
./run.sh study.connector.FileSinkLatencyDemo oncheckpoint 30
```

```bash
./run.sh study.connector.SourceSinkDemo normal
```

```bash
./run.sh study.connector.SourceSinkDemo fail-naive
```

运行 `FileSinkLatencyDemo` 时，可以另开一个终端看输出目录 `/tmp/flink-filesink-demo`：版本 A 的前 120 秒里，只有一个以 `.part-` 开头、以 `.inprogress.xxx` 结尾的隐藏文件在不断变大。

**推荐的断点**：

| 断点 | 看什么 |
|---|---|
| `SinkWriterOperator.java:192` | 第一阶段：交出 committable |
| `FileWriterBucket.java:196` | 版本 A / B 的 `shouldRollOnCheckpoint` 结果 |
| `FileWriterBucket.java:233` | 按处理时间滚动的检查 |
| `CommitterOperator.java:159` | 第二阶段：Checkpoint 完成后提交 |
| `CommitterOperator.java:139` | 恢复时再提交一次 |

---

## 六、课后练习

1. **调小检查周期**：版本 A 加上 `.withBucketCheckInterval(1000)`，再把 `DefaultRollingPolicy` 的 `withRolloverInterval` 设成 10 秒，数据要多久才能看到？
2. **大文件**：把版本 A 的数据速率调到每秒几万条，文件超过 128MB 后，`shouldRollOnCheckpoint` 会返回什么？
3. **Final Checkpoint**：把 `checkpoints-after-tasks-finish` 设成 false，跑 `SourceSinkDemo normal`，最后一批数据还能提交吗？
4. **思考**：为什么 Flink 不在 Checkpoint 完成**之前**提交？如果在 `prepareCommit` 时就直接提交，会出现什么问题？

---

## 写在最后

这一讲的核心可以用一句话概括：**两阶段提交把"写"和"提交"分开：写随时进行，交出在 Barrier 之前，提交在 Checkpoint 完成之后；Source 的进度和 Sink 的事务在同一个 Checkpoint 里对齐，恢复时再提交一次，所以 Committer 必须幂等。**

回到开头：Checkpoint 每 2 秒做一次，但 FileSink 的 Writer 在 Checkpoint 时没有交出任何东西，因为默认的滚动策略不在 Checkpoint 时关闭文件。**Checkpoint 只决定"能提交什么"的时机，Writer 决定"有什么可以提交"。**

**实际使用时的建议**：

- 用 Row 格式的 FileSink，要关注滚动策略，否则数据可能要一两分钟后才能看到；
- 即使滚动策略配置正确，下游看到数据也要等到下一个 Checkpoint 完成：最长约一个 Checkpoint 间隔，平均约一半（版本 B 实测中位数 1.1 秒、最大 2.1 秒）。exactly-once 的代价，就是这段延迟；
- 自己实现 Committer：事务 ID 要带 Checkpoint 编号，`commit` 要幂等，重试要自己退避；
- 事务型 Sink 一定要开 Checkpoint。

到这里，第八讲（Source / Sink）就结束了。

> **第二季预告**：第二季讲 **Apache Paimon**，第 9 讲"Flink 写入"正好接着这一讲。一个有意思的细节：Paimon 写入 Flink 的默认路径没有直接实现 Sink V2 接口，而是自己搭了一组算子（`FlinkSink`），但两个阶段的位置和这一讲一样：Barrier 之前交出待提交的内容（`PrepareCommitOperator.prepareSnapshotPreBarrier`），Checkpoint 完成后再提交（`CommitterOperator.notifyCheckpointComplete`）。和 FileSink 把文件改名不同，Paimon 每次提交的结果是表的一个新**快照**。从源码看（`CommitterOperator` 的注释），不开 Checkpoint 时它也只在输入结束时提交，和这一讲 4.3 节的结论一致；这一点第二季会实测。（类名基于 Paimon master @ `d15d250cf`，第二季会按当时的正式版复核；第二季的实验用 Flink 2.2，Paimon 目前支持到 Flink 2.2。）


**留一个问题**：你的作业里，下游看到数据的延迟大概是多少？有没有因为 exactly-once 的延迟，选择过 at-least-once？
:::
