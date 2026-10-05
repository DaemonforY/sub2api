---
title: "Flink 2.x 源码精读（三·下）：同一段代码，换成 RocksDB 后计数全变成了 0"
description: "一段在 HashMap 状态后端上运行正确的计数代码，换成 RocksDB 后，4 个 key 的计数全部是 0。顺着这个陷阱，读懂 Flink 2.3 中 RocksDB 的存储格式、增量 Checkpoint 每次到底上传了什么、扩缩容时状态怎么裁剪，以及 ForSt 和异步状态。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-03b-cover.webp"}]]
---

# Flink 2.x 源码精读（三·下）：同一段代码，换成 RocksDB 后计数全变成了 0

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03b-cover.webp" alt="Yui和Kai在状态存储世界中探索检查点与扩缩容" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在状态存储世界中探索检查点与扩缩容<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`，`RDB` = `flink-state-backends/flink-statebackend-rocksdb/src/main/java/org/apache/flink/state/rocksdb`
> 上篇：《扩容一次成功，改一个配置就再也恢复不了？读懂 KeyGroup》

---

上篇留了一道练习：计数用 `ValueState<long[]>` 保存，每来一条数据，直接执行 `value()[0]++` 修改数组，**不调用 `update()`**。

核心代码是这样的：

```java
long[] c = count.value();
if (c == null) {
    c = new long[] {0};
    count.update(c);       // 第一次：放进状态
}
c[0]++;                    // 直接修改拿到的数组，不调用 update()
```

4 个 key，每个 key 1000 条数据，并行度 1。本机实测，每个 key 的最终计数：

| 状态后端 | 调用 `update()` 吗 | key 0 | key 1 | key 2 | key 3 |
|---|---|---|---|---|---|
| HashMap | 否 | 1000 | 1000 | 1000 | 1000 |
| **RocksDB** | **否** | **0** | **0** | **0** | **0** |
| RocksDB | 是 | 1000 | 1000 | 1000 | 1000 |

**同一段代码，在 HashMap 上结果正确，换成 RocksDB 后，所有更新全部丢失，而且没有任何报错。**

这是从 HashMap 切换到 RocksDB 时最隐蔽的一类 bug。这一篇就从它开始，看懂 RocksDB 状态后端：

1. 一条状态在 RocksDB 里长什么样？为什么上面的代码会丢数据？
2. 增量 Checkpoint 每次**到底上传了什么**？我翻了一遍 Checkpoint 目录。
3. 旧的 Checkpoint 文件什么时候删？
4. 扩缩容时，RocksDB 的状态是怎么裁剪、合并的？
5. Flink 2.x 的 ForSt 和异步状态解决了什么问题？

---

## 一、一条状态在 RocksDB 里长什么样

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03b-1.webp" alt="Yui和Kai观察RocksDB中按KeyGroup连续排列的状态记录" width="960" height="640" loading="lazy" /><figcaption>Yui和Kai观察RocksDB中按KeyGroup连续排列的状态记录<span>AI 生成配图</span></figcaption></figure>

### 1.1 结构

- **每个子任务一个 RocksDB 实例**（`RDB/RocksDBKeyedStateBackend.java:268`）；
- **每个状态描述符一个 Column Family**，可以理解为同一个数据库里的一张独立的表（`:1022` 起的 `createOrUpdateInternalState()`，最终调用 `RDB/RocksDBOperationUtils.java:278`）。

RocksDB 是一个 key-value 存储，key 和 value 都是字节数组。Flink 写进去的 key，是三部分拼起来的：

```
┌──────────────┬──────────────────┬───────────────────────┐
│ KeyGroup 前缀 │ 序列化后的 key    │ 序列化后的 namespace   │  →  value：序列化后的状态值
└──────────────┴──────────────────┴───────────────────────┘
```

**KeyGroup 放在最前面**（`RT/runtime/state/CompositeKeySerializationUtils.java:109`），而且按大端序写入。RocksDB 按 key 的字节顺序排序，所以**同一个 KeyGroup 的数据在物理上是连续的**。这一点在第四节扩缩容时至关重要。

前缀占几个字节，由 maxParallelism 决定（`:169`）：maxParallelism 不超过 128 时用 1 个字节，否则用 2 个字节。这就是上篇说"maxParallelism 不要设得过大"的原因之一：它会让每一条数据的 key 都变长。

### 1.2 为什么开头的代码会丢数据

`RocksDBValueState.value()`（`RDB/RocksDBValueState.java:79`）：

```java
byte[] valueBytes = backend.db.get(columnFamily, serializeCurrentKeyWithGroupAndNamespace());   // :82
// 然后把 valueBytes 反序列化成一个对象返回
```

**每次 `value()`，都是从 RocksDB 里读出字节，再反序列化成一个全新的对象。** 你修改的只是这个新对象，RocksDB 里存的字节一点没变。不调用 `update()`，修改就永远不会写回去。所以每次读到的都是第一次写进去的 `{0}`。

而 HashMap 状态后端恰好相反（上篇 4.1 节）：状态就是 JVM 堆里的一个对象，`value()` 返回的是**它本身的引用**。你直接修改它，就等于修改了状态，所以"碰巧"正确。

> 注意：HashMap 状态后端上的这个"碰巧正确"也并不可靠。上篇讲过，Checkpoint 期间 `get()` 会触发写时复制，换出一个新对象。本实验没有开启 Checkpoint，所以没有遇到这种情况。

**结论：不管用哪种状态后端，修改状态之后都要调用 `update()`。**

### 1.3 读写都要序列化

由 1.2 节可以推出 RocksDB 和 HashMap 的根本区别：**RocksDB 的每一次读写都要序列化和反序列化**。所以单次状态访问比 HashMap 慢得多，但状态的大小只受磁盘限制，不受 JVM 堆的限制。

RocksDB 自己用的内存（读缓存、写缓冲区）默认从 Flink 的托管内存里分配（`state.backend.rocksdb.memory.managed` 默认 `true`，`RDB/RocksDBOptions.java:123-125`）。每个算子能分到多少托管内存，是在 JobManager 生成 JobGraph 时算好的（`RT/streaming/api/graph/StreamingJobGraphGenerator.java:247` 调用的 `setManagedMemoryFraction()`）。

---

## 二、增量 Checkpoint：每次到底上传了什么

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03b-2.webp" alt="Yui和Kai只把新增SST文件送入远端检查点仓库" width="960" height="640" loading="lazy" /><figcaption>Yui和Kai只把新增SST文件送入远端检查点仓库<span>AI 生成配图</span></figcaption></figure>

### 2.1 先说一个容易忽略的默认值

**增量 Checkpoint 默认是关闭的。** 配置项 `execution.checkpointing.incremental` 的默认值是 `false`（`flink-core/.../configuration/CheckpointingOptions.java:133-135`）。不打开它，RocksDB 每次做的都是全量 Checkpoint。下面的实验是手动打开的。

### 2.2 同步阶段：硬链接

第二讲说过，快照分同步和异步两段。RocksDB 的同步阶段在 `RDB/snapshot/RocksDBSnapshotStrategyBase.java:151`，核心只有两行：

```java
Checkpoint checkpoint = Checkpoint.create(db);                              // :174
checkpoint.createCheckpoint(outputDirectory.getDirectory().toString());     // :175
```

这是 RocksDB 自带的"本地检查点"功能：把内存中的数据刷到磁盘，然后对所有 SST 数据文件做**硬链接**。SST 文件一旦写好就不会再修改，所以硬链接出来的就是一份一致的快照，几乎不花时间。

### 2.3 异步阶段：SST 能复用就复用

真正上传是在异步阶段（`RDB/snapshot/RocksIncrementalSnapshotStrategy.java:413` 起的 `createUploadFilePaths()`）：

```java
if (fileName.endsWith(SST_FILE_SUFFIX)) {                                        // :421  是 SST 文件
    Optional<StreamStateHandle> uploaded = previousSnapshot.getUploaded(fileName);
    if (uploaded.isPresent() && ...couldReuseStateHandle(uploaded.get())) {
        sstFiles.add(HandleAndLocalPath.of(uploaded.get(), fileName));            // 以前传过：直接复用
    } else {
        sstFilePaths.add(filePath);                                               // 新文件：上传
    }
} else {
    miscFilePaths.add(filePath);                                                  // :430  其他文件：每次都上传
}
```

- **SST 文件**：按文件名判断是否上传过。RocksDB 的 SST 文件名是递增的编号，内容又不可变，所以同名就是同一份内容；
- **其他文件**（OPTIONS、MANIFEST、CURRENT）：每次都重新上传；
- "以前传过"的基准，要等 Checkpoint **确认完成**后才更新（`:167` 的 `notifyCheckpointComplete()`）。否则，如果复用了一次失败的 Checkpoint 上传的文件，就可能引用到已经被删掉的文件。

### 2.4 实测：翻一遍 Checkpoint 目录

示例作业：1000 个 key，每个 key 一个计数和一段 1KB 的数据，并行度 2，每 5 秒一次 Checkpoint，保留最近 3 个。运行 32 秒，完成了 7 次 Checkpoint，大小依次是（字节）：

```
141307 → 154461 → 167763 → 180996 → 152207 → 165468 → 178782
```

目录里保留了 chk-5、chk-6、chk-7。我统计了 `shared/` 下每个文件被哪几个保留的 Checkpoint 引用（在每个 Checkpoint 的 `_metadata` 里搜索文件名）：

| shared 文件 | 大小（字节） | 创建时间 | 被引用 |
|---|---|---|---|
| `b9d5336f…` | 35253 | 15:52:04（第 1 次 Checkpoint） | chk-5、chk-6、chk-7 |
| `cc269b74…` | 36714 | 15:52:04（第 1 次 Checkpoint） | chk-5、chk-6、chk-7 |
| `6ac88626…`、`e64cc5dd…` | 25845 | 15:52:24 | chk-5 |
| `2c2a8460…`、`c60eaae8…` | 25845 | 15:52:29 | chk-6 |
| `7e130e22…`、`f8aecfc6…` | 25845 | 15:52:34 | chk-7 |

光看这张表，很容易得出一个结论："每次 Checkpoint 只新上传两个 25KB 的 SST 文件，里面是这 5 秒内的计数更新。"**我自己最早的笔记就是这么写的，但这是错的。**

后面做扩缩容实验时，恢复日志里列出了每个文件在 RocksDB 本地对应的文件名。以子任务 0 在 chk-7 中的状态为例（本机实测，整理自日志）：

| 本地文件名 | 大小 | 存储方式 | 是什么 |
|---|---|---|---|
| `000017.sst` | 36714 B | 独立文件（`cc269b74…`） | 第 1 次 Checkpoint 上传后**一直被复用**的 SST，里面主要是那 1000 段 1KB 的数据 |
| `000021.sst` ~ `000024.sst` | 4.9~6.7 KB | **内联在 `_metadata` 里** | 每次新产生的小 SST，里面是计数的更新 |
| `OPTIONS-000015` | 25845 B | 独立文件（`f8aecfc6…`） | **RocksDB 的配置文件**，每次都重新上传 |
| `MANIFEST-000005`、`CURRENT` | 1561 B、16 B | 内联在 `_metadata` 里 | RocksDB 的元数据文件 |

**原来那些每次新增的 25845 字节文件，是 RocksDB 的 OPTIONS 配置文件，不是 SST。** 计数的更新确实在新的 SST 里，但这些 SST 太小了，没有单独写成文件，而是直接**内联**进了 `_metadata`。

为什么会内联？配置项 `execution.checkpointing.data-inline-threshold`，默认 **20KB**（`CheckpointingOptions.java:240-242`）：小于它的状态数据，不单独写文件，直接放进 `_metadata`。这也解释了另一个现象：三个 Checkpoint 的 `_metadata` 分别是 31725、45230、58788 字节，每次增长约 13.5KB，正好是两个子任务各新增一个约 6.6KB 的小 SST。

**所以准确的结论是**：
1. 第 1 次 Checkpoint 上传的大 SST，被之后的每一次 Checkpoint 复用，只传了一次；
2. 之后每次只新增几 KB 的小 SST（内联在 `_metadata` 里），外加每次都要重传的 OPTIONS 等文件；
3. 这类细节，只看 Checkpoint 目录是看不出来的，要结合 RocksDB 本地的文件名才能分辨。

再回头看那张 `shared/` 文件表：为什么每个 25845 字节的文件只被一个 Checkpoint 引用？因为它们是 OPTIONS 文件，属于每次都要重新上传的部分（2.3 节），每个 Checkpoint 都有自己的一份。而小 SST 因为内联在 `_metadata` 里，根本不会出现在 `shared/` 目录中，它们其实是会被后续 Checkpoint 继续引用的：chk-7 里同时引用着 `000021.sst` 到 `000024.sst` 四个小 SST。

> 思考题：Checkpoint 的大小在第 5 次时从 180996 字节降到了 152207 字节。结合 RocksDB 的 compaction（把多个小 SST 合并成新文件），你能解释这次下降吗？可以对比 chk-4 和 chk-5 引用的 SST 列表来验证。

---

## 三、共享文件什么时候删

多个 Checkpoint 共享同一个 SST 文件，所以删除旧 Checkpoint 时，不能顺手把它引用的文件也删掉。管理这件事的是 JobManager 上的 **`SharedStateRegistry`**（`RT/runtime/state/SharedStateRegistryImpl.java`）：

1. 收到 Task 的 ack 时，登记每个共享文件（`RT/runtime/checkpoint/CheckpointCoordinator.java:1240` → `SharedStateRegistryImpl.java:81` 的 `registerReference()`），并记下它**最后一次被哪个 Checkpoint 使用**（`lastUsedCheckpointID`，`:313`）；
2. 旧 Checkpoint 被淘汰时，找出仍保留的 Checkpoint 中编号最小的那个（`RT/runtime/checkpoint/AbstractCompleteCheckpointStore.java:56` 起），调用 `unregisterUnusedState(最小编号)`（`SharedStateRegistryImpl.java:159`）：`lastUsedCheckpointID` 比它还小的文件，已经没有任何保留中的 Checkpoint 在用了，异步删除（`:243`）。

实验里保留的是 chk-5、chk-6、chk-7，所以只被 chk-2 到 chk-4 用过的文件都已经被删掉了，而第 1 次上传的两个大 SST，因为一直被引用，至今还在。

---

## 四、扩缩容：RocksDB 怎么切分状态

上篇讲过，JobManager 按 KeyGroup 区间求交集，决定每个新子任务要从哪些旧子任务那里取状态。HashMap 状态后端可以按偏移只读自己需要的部分，但 **RocksDB 的 SST 文件没法按 KeyGroup 切开**。所以 RocksDB 的做法是：先把整个文件拿过来，再删掉不属于自己的部分。

### 4.1 两条路径

`RDB/restore/RocksDBIncrementalRestoreOperation.java:319` 的 `restoreFromLocalState()`：

- **只有一个来源**：打开这个数据库，然后调用 `clipDBWithKeyGroupRange()`（`RDB/RocksDBIncrementalCheckpointUtils.java:134`），用 RocksDB 的 `deleteRange` 删掉区间两端多出来的 KeyGroup。因为 KeyGroup 是 key 的前缀（1.1 节），一段 KeyGroup 就是一段连续的字节区间，删除非常快；
- **有多个来源**（`:422` 的 `restoreFromMultipleStateHandles()`）：挑一个作为基础数据库打开并裁剪，再把其他来源中属于自己区间的数据合并进来。如果开启了 `state.backend.rocksdb.use-ingest-db-restore-mode`（默认 `false`，`RDB/RocksDBConfigurableOptions.java:316-318`），会用 RocksDB 的 ingest 机制直接导入整个 SST 文件（`:467`），比逐条复制快得多。

### 4.2 实测：从 2 个并行度恢复成 3 个

从上面的 chk-7 恢复，并行度改为 3。三个子任务恢复的 key 数分别是 359、311、330，合计 1000，和上篇 HashMap 的结果完全相同（因为 maxParallelism 一样，KeyGroup 的划分就一样）。

恢复日志显示，三个子任务走了不同的路径（本机实测，整理自日志）：

| 新子任务 | 目标区间 | 来源 | 恢复方式 | 删除的区间（日志中的 boundaries） |
|---|---|---|---|---|
| 1/3 | [0, 42] | 只有旧子任务 0 [0, 63] | 打开基础数据库后裁剪 | `[[43], [64]]`：删掉 [43, 64) |
| 3/3 | [86, 127] | 只有旧子任务 1 [64, 127] | 打开基础数据库后裁剪 | `[[64], [86]]`：删掉 [64, 86) |
| 2/3 | [43, 85] | 旧子任务 0 **和** 1 | "restore backend ... from multiple state handles"：打开一个并裁剪，再合并另一个 | `[[86], [-128]]`：删掉 [86, 128) |

边界是 KeyGroup 前缀的**字节值**。`-128` 是 128 用有符号字节表示的结果（maxParallelism 为 128 时前缀只有 1 个字节），意思是"一直删到最后"。日志里还能看到 `Use IngestDB=false`，也就是走的默认的合并方式。

---

## 五、ForSt 与异步状态：Flink 2.x 的存算分离

### 5.1 为什么需要

在云上部署时，把状态放在本地盘的 RocksDB 有几个痛点：
- 状态大小受本地盘限制；
- 扩缩容和故障恢复时，要先把 SST 文件**下载**到本地，状态大的作业可能要很久；
- 每次 Checkpoint 都要把新的 SST **上传**到远端存储。

**ForSt** 是 RocksDB 的一个分支，它的思路是把**主存储直接放在远端存储**上，本地盘只当缓存。主存储目录由 `state.backend.forst.primary-dir` 控制，**默认和 Checkpoint 目录相同**（`flink-state-backends/flink-statebackend-forst/.../ForStOptions.java:58-67`）。这样，Checkpoint 时数据已经在远端了，基本不需要再上传；恢复时也不需要下载。

但新的问题随之而来：**读远端存储有毫秒级的延迟。** 如果还像以前那样，在 Task 线程里同步地一条一条读，Task 线程大部分时间都会卡在等待 IO 上。

### 5.2 异步状态 API

所以 Flink 2.x 引入了异步状态 API（State V2，`flink-core-api/.../api/common/state/v2/`）：

```java
StateFuture<T> asyncValue();              // ValueState.java:49
StateFuture<Void> asyncUpdate(T value);   // :60
T value();                                // :73  同步版本仍然保留
```

使用时，要在 `keyBy` 之后调用 `enableAsyncState()`（`RT/streaming/api/datastream/KeyedStream.java:1099`）。开启后，算子会被换成异步版本，比如 `process()` 会创建 `AsyncKeyedProcessOperator`，而不是 `KeyedProcessOperator`（`KeyedStream.java:362-365`）。示例中的写法：

```java
count.asyncValue().thenAccept(old -> {
    onKey(key, old);
    count.asyncUpdate(old == null ? 1 : old + 1);
});
```

### 5.3 异步了，同一个 key 的顺序怎么保证

最大的难点是：**同一个 key 的数据必须按顺序处理。** 否则两条数据同时执行"读计数 → 加 1 → 写回"，就会丢掉一次更新。

负责这件事的是 `AsyncExecutionController`（`RT/runtime/asyncprocessing/AsyncExecutionController.java`）。它的处理方式（`:304` 起的 `handleRequest()`）：

1. 在途的数据太多，就先等一等（`:384`，上限见 `:88`）；
2. **检查这个 key 现在有没有别的数据正在处理**（`:293` 的 `tryOccupyKey()`，背后是 `KeyAccountingUnit.occupy()`，`RT/runtime/asyncprocessing/KeyAccountingUnit.java:47`）：
   - 没有：放进可执行队列（`:342`）；
   - 有：先排队，等前面那条处理完（`:354`）；
3. 攒够一批，交给状态后端批量执行（`:373`）。

**不同 key 的数据可以并发访问状态，同一个 key 的数据严格串行。**

更重要的是第 4 点：**回调回到 Task 线程执行。** `AsyncExecutionController` 用一个包装器把 `thenAccept` 里的回调投递回 Mailbox（`:166`）。所以回调和 `processElement` 仍然在同一个 Task 线程上执行，第一讲下篇的结论依然成立：**用户代码不需要加锁**。

做 Checkpoint 之前，还要先把在途的请求都处理完（`:429` 的 `drainInflightRecords()`），否则快照里会缺少那些"读了但还没写回"的更新。

### 5.4 实测：线程是怎么分工的

用 ForSt 运行示例（自动开启异步状态），运行中用 `jstack` 抓一次线程栈，统计和状态相关的线程（本机实测）：

```
   2 ForSt-StateExecutor-Coordinator-And-Write-thread   ← 每个子任务 1 个：协调批次、执行写入
   6 ForSt-StateExecutor-read-IO-thread                  ← 每个子任务 3 个：并发读取
   8 Flink-ForStStateDataTransfer-thread                 ← Checkpoint、恢复时传输文件
   1 asyncRequestBuffer-timeout-scheduler-thread         ← 一批攒不满时，按超时触发
```

（读线程和写线程定义在 `flink-state-backends/flink-statebackend-forst/.../ForStStateExecutor.java:59-62`。）

同时，示例在回调里打印的线程名是 `count-per-key -> Sink: Writer (1/2)#0`，也就是 **Task 线程本身**。IO 在后台线程池里执行，回调回到 Mailbox 执行，和 5.3 节一致。

> 说明：本实验的 Checkpoint 目录在本地 `/tmp` 下，ForSt 的主存储默认也在这里，相当于用本地文件系统**模拟**远端存储，没有测到真实远端存储的延迟。

---

## 六、怎么选

三种状态后端在同一个示例（1000 个 key，每个 key 约 1KB）上的 Checkpoint 大小（本机实测，稳定后）：

| | HashMap | RocksDB | ForSt |
|---|---|---|---|
| Checkpoint 大小 | 1058762 B（全量） | 141~181 KB（增量） | 148~189 KB（增量） |
| 状态放在哪 | JVM 堆 | 本地磁盘 + 托管内存 | 远端存储（主）+ 本地缓存 |
| 每次访问 | 直接拿对象引用 | 每次都要序列化 | 序列化 + 可能的远程 IO |
| 状态规模 | 受堆内存限制 | 受本地盘限制 | 受远端存储限制 |
| 修改状态 | 原地修改"碰巧"能生效（不要依赖） | **必须调用 `update()`** | 必须调用 `asyncUpdate()` |
| 推荐 API | 同步 | 同步 | 异步（State V2） |
| 适合 | 小状态、低延迟 | 大状态的主流选择 | 云上部署、超大状态、需要快速扩缩容 |

RocksDB 和 ForSt 的 Checkpoint 比 HashMap 小得多，除了增量的原因，还因为 SST 文件默认用 Snappy 压缩（`state.backend.rocksdb.compression.per.level`，`RDB/RocksDBConfigurableOptions.java:171-174`），而示例中的数据全是同一个字符，压缩率极高。真实业务数据的压缩效果不会这么好。

---

## 七、自己动手

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/state/` 下：

```bash
# 开头的原地修改陷阱
./run.sh study.state.InPlaceUpdateDemo hashmap
./run.sh study.state.InPlaceUpdateDemo rocksdb
./run.sh study.state.InPlaceUpdateDemo rocksdb update

# 增量 Checkpoint：运行 32 秒，然后查看 /tmp/flink-state-demo/<jobId>/shared/
./run.sh study.state.StateBackendDemo rocksdb 2 32

# 扩缩容：从上面的 chk-7 恢复成 3 个并行度
KEEP_CKPT=1 ./run.sh study.state.StateBackendDemo rocksdb 3 22 /tmp/flink-state-demo/<jobId>/chk-7

# ForSt + 异步状态
./run.sh study.state.StateBackendDemo forst 2 32
```

检查 `_metadata` 里引用了哪些文件时，注意它是二进制文件，要用 `LC_ALL=C grep -a`，否则在中文环境下可能匹配不到。我第一次统计时，所有文件都显示"没有被引用"，就是这个原因。

**断点清单**：

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `CompositeKeySerializationUtils.java:109` | KeyGroup 前缀怎么写进 key |
| 2 | `RocksDBValueState.java:82` | 每次 `value()` 都是一次 `db.get` |
| 3 | `RocksDBSnapshotStrategyBase.java:175` | 本地检查点（硬链接） |
| 4 | `RocksIncrementalSnapshotStrategy.java:421` | 哪些文件复用，哪些重传 |
| 5 | `SharedStateRegistryImpl.java:159` | 哪些共享文件被删除 |
| 6 | `RocksDBIncrementalCheckpointUtils.java:134`（带恢复路径、改并行度） | 按 KeyGroup 裁剪 |
| 7 | `AsyncExecutionController.java:293`（forst） | 同一个 key 的请求怎么排队 |

---

## 八、课后练习

1. **开启 Checkpoint 再试一次**：给 `InPlaceUpdateDemo` 开启每秒一次的 Checkpoint，用 HashMap 状态后端运行（不调用 `update()`）。结果还是 1000 吗？结合上篇 4.2 节的写时复制分析。
2. **关闭增量 Checkpoint**：把示例中的增量 Checkpoint 关掉，再用 RocksDB 运行，Checkpoint 大小和 `shared/` 目录会怎么变？
3. **缩容**：从并行度 2 的 RocksDB Checkpoint 恢复成并行度 1。源码注释里说"This happens for all scale ins"，所有缩容都会走"多个来源"的路径，为什么？
4. **思考**：为什么"以前传过的 SST"这个基准，必须等到 Checkpoint 确认完成后才更新？如果在上传完就更新，会出现什么问题？（提示：第二讲上篇讲过，Checkpoint 可能失败，完成通知也可能丢失）

---

## 写在最后

第三讲到这里结束了。

| 上篇 | 下篇 |
|---|---|
| 状态按 KeyGroup 分配，maxParallelism 一旦确定就不能显式修改 | RocksDB 的 key 以 KeyGroup 开头，同一个 KeyGroup 的数据物理连续 |
| HashMap：写时复制，快照不阻塞处理 | RocksDB：硬链接 + 只上传新的 SST |
| 扩缩容：按区间求交集，按偏移读取 | 扩缩容：拿来整个文件，再用 deleteRange 裁剪 |
| | ForSt：主存储在远端，配合异步状态 API 使用 |

下一讲，我们看数据是怎么在 Task 之间流动的：**网络栈与背压**。前两讲里反复出现的"背压"，到底是怎么一级一级传到上游的？第二讲下篇里，为什么数据主要堆在 Source 的输出缓冲区里？


**留一个问题**：你的生产作业用的是哪种状态后端？增量 Checkpoint 打开了吗？

下一讲：**Flink 2.x 源码精读（四）：网络栈与背压**
:::
