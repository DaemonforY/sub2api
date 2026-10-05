---
title: "Flink 2.x 源码精读（三·上）：扩容一次成功，改一个配置就再也恢复不了？读懂 KeyGroup"
description: "同一个 Checkpoint，并行度从 2 改成 3，1000 个 key 一个不少地恢复了；maxParallelism 从 128 改成 256，作业直接起不来；而如果你从没设置过它，Flink 会悄悄沿用旧值。读懂 Flink 2.3 的 KeyGroup、HashMap 状态后端和扩缩容。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-03a-cover.webp"}]]
---

# Flink 2.x 源码精读（三·上）：扩容一次成功，改一个配置就再也恢复不了？读懂 KeyGroup

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03a-cover.webp" alt="Yui和Kai守护按KeyGroup分配并可扩缩容的状态流" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai守护按KeyGroup分配并可扩缩容的状态流<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 前置阅读：第二讲（Checkpoint 的同步、异步快照）

---

先看三次实验。

作业很简单：1000 个 key，每个 key 维护一个计数和一段 1KB 的数据，用 HashMap 状态后端，并行度 2，每 5 秒做一次 Checkpoint。跑 22 秒后取消，然后从最后一个 Checkpoint 恢复。

**实验一：并行度从 2 改成 3。** 本机实测输出：

```
>>> [count-per-key -> Sink: Writer (1/3)#0] maxParallelism=128 keyGroups=[0, 42] keys=359 restoredKeys=359 keysOutOfRange=0
>>> [count-per-key -> Sink: Writer (2/3)#0] maxParallelism=128 keyGroups=[43, 85] keys=311 restoredKeys=311 keysOutOfRange=0
>>> [count-per-key -> Sink: Writer (3/3)#0] maxParallelism=128 keyGroups=[86, 127] keys=330 restoredKeys=330 keysOutOfRange=0
```

359 + 311 + 330 = **1000**。每个 key 都在新的子任务上找回了自己的状态，一个不少，也没有重复。

**实验二：把 `pipeline.max-parallelism` 从 128 改成 256。** 作业直接起不来：

```
java.lang.IllegalStateException: Failed to rollback to checkpoint/savepoint ... Max parallelism mismatch between
checkpoint/savepoint state and new program. Cannot map operator 5b61d6019ab04da71a4e62533c1a8c40 with max parallelism 128
to new program with max parallelism 256. This indicates that the program has been changed in a non-compatible way ...
```

**实验三：反过来，用 maxParallelism=256 生成 Checkpoint，恢复时干脆不设置它。** 按并行度 2 推算，默认值应该是 128。结果作业正常恢复了，而且运行时实际生效的是：

```
>>> [count-per-key -> Sink: Writer (1/2)#0] maxParallelism=256 keyGroups=[0, 127] keys=494 restoredKeys=494 keysOutOfRange=0
>>> [count-per-key -> Sink: Writer (2/2)#0] maxParallelism=256 keyGroups=[128, 255] keys=506 restoredKeys=506 keysOutOfRange=0
```

**256，不是 128。** Flink 悄悄沿用了 Checkpoint 里的值。

为什么并行度可以随便改，maxParallelism 却不行？为什么不设置反而能恢复？这一篇，我们从"状态是怎么存的"开始，一路看到"扩缩容时状态是怎么切开的"：

1. KeyGroup 是什么？为什么需要它？
2. 你调用 `state.value()` 时，Flink 怎么知道"当前 key"是谁？
3. HashMap 状态后端怎么做到"做快照时不阻塞数据处理"？
4. 扩缩容时，状态是怎么重新分配的？
5. maxParallelism 到底能不能改？

---

## 一、先分清两个概念

| 概念 | 负责什么 | 选项 |
|---|---|---|
| **State Backend**（状态后端） | 作业**运行时**，状态放在哪里、怎么读写、怎么做快照 | `hashmap`（JVM 堆内存）、`rocksdb`（本地磁盘）、`forst`（远端存储 + 本地缓存） |
| **Checkpoint Storage**（Checkpoint 存储） | 快照**写到哪里** | JobManager 内存（仅用于测试）、文件系统（HDFS、S3、本地目录） |

状态后端由配置项 `state.backend.type` 决定，**默认是 `hashmap`**（`flink-core/.../configuration/StateBackendOptions.java:46-49`）。代码里调用 `env.setStateBackend(...)` 的优先级更高（`RT/runtime/state/StateBackendLoader.java:265`）。

每个子任务的每个算子，都有自己独立的状态后端实例（接口见 `RT/runtime/state/StateBackend.java:81`）。这一篇先讲最简单的 HashMap 状态后端，RocksDB 和 ForSt 放到下篇。

---

## 二、KeyGroup：状态分配的最小单位

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03a-1.webp" alt="KeyGroup像分区岛屿，让状态随并行度重新分配" width="960" height="640" loading="lazy" /><figcaption>KeyGroup像分区岛屿，让状态随并行度重新分配<span>AI 生成配图</span></figcaption></figure>

### 2.1 为什么不能直接按 key 分？

`keyBy` 之后，每个 key 的数据都会发到固定的一个子任务上，它的状态也存在那个子任务里。

最直观的分法是 `hash(key) % 并行度`。但这样做有一个致命问题：**并行度一变，几乎所有 key 都要换到别的子任务上。** 比如从 2 改成 3，原来在子任务 0 的 key，现在可能该去子任务 1 或 2 了。状态就得按 key 一条一条地重新分发，代价极高。

Flink 的办法是在中间加一层：

```
key ──哈希──► KeyGroup（0 ~ maxParallelism-1，固定不变）──按连续区间──► 子任务
```

- key 属于哪个 KeyGroup，只取决于 **maxParallelism**，**和并行度无关**；
- 每个子任务负责一段**连续的** KeyGroup；
- 扩缩容时，只需要**重新划分区间**，状态以 KeyGroup 为单位**整块**搬迁。

【配图 1：KeyGroup 的两层映射。左边是一堆 key，中间是 128 个格子（KeyGroup），右边是子任务。上半部分：并行度 2，格子 0~63 归子任务 0，64~127 归子任务 1；下半部分：并行度 3，同样的 128 个格子被重新划分成 0~42、43~85、86~127 三段。key 到格子的连线上下两部分完全相同】

### 2.2 源码：三个公式

`RT/runtime/state/KeyGroupRangeAssignment.java`

```java
// ① key → KeyGroup（:63 → :75）
public static int computeKeyGroupForKeyHash(int keyHash, int maxParallelism) {
    return MathUtils.murmurHash(keyHash) % maxParallelism;              // :76
}

// ② 子任务 i 负责的 KeyGroup 区间（:93）
start = (i * maxParallelism + parallelism - 1) / parallelism;
end   = ((i + 1) * maxParallelism - 1) / parallelism;

// ③ KeyGroup → 子任务（:124）
return keyGroupId * parallelism / maxParallelism;
```

**同一套算法用在了两个地方**，所以它们一定是一致的：

- **数据发给谁**：`keyBy` 背后的 `KeyGroupStreamPartitioner`，用它决定一条数据发给哪个子任务（`RT/streaming/runtime/partitioner/KeyGroupStreamPartitioner.java:63`）；
- **状态存在哪**：状态后端用它算出当前 key 的 KeyGroup，决定状态放在哪个分片（`RT/runtime/state/AbstractKeyedStateBackend.java:258-262`）。

实验里的 `keysOutOfRange=0` 验证的就是这一点：每个子任务收到的 key，算出来的 KeyGroup 都落在它自己的区间里。

### 2.3 实测：16 个 key 怎么分

用 Flink 自己的函数算一下，16 个 key（0~15）在并行度 2、maxParallelism 128 时的分布（本机实测，节选）：

```
key=0  -> keyGroup=94  -> subtask 1
key=3  -> keyGroup=113 -> subtask 1
key=4  -> keyGroup=7   -> subtask 0
key=7  -> keyGroup=113 -> subtask 1
...
subtask0 有 7 个 key，subtask1 有 9 个 key
```

注意 key 3 和 key 7 落进了**同一个** KeyGroup（113）。key 很少的时候，分布不均几乎不可避免：16 个 key 散在 128 个 KeyGroup 上，每个子任务分到几个，全凭哈希的结果。

第二讲上篇的思考题里提到过"16 个 key 分成 7 个和 9 个"，这就是它的来历。

再看同一个 key 在不同 maxParallelism 下的结果：

```
key="user-42", maxParallelism=128 -> keyGroup=20
key="user-42", maxParallelism=256 -> keyGroup=148
```

**maxParallelism 一变，同一个 key 就落进了不同的 KeyGroup。** 这是第六节的伏笔。

### 2.4 默认的 maxParallelism

如果你不设置，Flink 会根据并行度推算一个默认值（`KeyGroupRangeAssignment.java:137` 起）：并行度乘以 1.5，再向上取到 2 的幂；下限 128（`:32`），上限 32768（`:35`）。本机实测：

| 并行度 | 默认 maxParallelism |
|---|---|
| 1、2、10 | 128 |
| 100 | 256 |
| 200 | 512 |
| 1000 | 2048 |
| 30000 | 32768 |

**maxParallelism 是并行度的上限。** 并行度不能超过它，因为每个子任务至少要分到一个 KeyGroup。

---

## 三、`state.value()` 怎么知道"当前 key"

### 3.1 你拿到的状态对象，不属于任何一个 key

在 `open()` 里调用 `getRuntimeContext().getState(...)`（`RT/streaming/api/operators/StreamingRuntimeContext.java:204`），最终会来到 `AbstractKeyedStateBackend.getOrCreateKeyedState()`（`AbstractKeyedStateBackend.java:378`）。它先查一个按名字缓存的表（`:387`），同名的状态只创建一次。

你拿到的这个 `ValueState` 对象，是整个子任务共享的，**它不绑定任何一个 key**。那 `value()` 读的是哪个 key 的值？

### 3.2 每条数据进来前，先切换"当前 key"

第一讲下篇看过一条数据的调用栈，其中有一步是"把当前 key 切换成这条记录的 key"。具体路径是：

```
AbstractStreamOperator.setKeyContextElement1(record)        RT/streaming/api/operators/AbstractStreamOperator.java:569
└ setCurrentKey(keySelector.getKey(value))                  :599
  └ StreamOperatorStateHandler.setCurrentKey()              RT/streaming/api/operators/StreamOperatorStateHandler.java:450
    └ AbstractKeyedStateBackend.setCurrentKey(newKey)        AbstractKeyedStateBackend.java:258
        记下当前 key，并算出它的 KeyGroup                       :262
```

之后你调用 `value()`，状态后端就用这个"当前 key"和"当前 KeyGroup"去找数据。

两个推论：

- **在 `processElement` 之外访问 keyed state 要格外小心。** 比如你自己起了一个线程去读状态，那时的"当前 key"是上一条记录留下的，读到的不是你想要的 key。Flink 的定时器在触发前，会先把当前 key 切换成定时器所属的 key（事件时间见 `RT/streaming/api/operators/InternalTimerServiceImpl.java:338`，处理时间见 `:301`），所以在 `onTimer` 里访问状态是安全的；而自己起的线程没有这一步。
- **每条数据都要算一次哈希。** 这是 keyed 算子固定的 CPU 开销。

---

## 四、HashMap 状态后端

### 4.1 数据结构：每个 KeyGroup 一张表

```
HeapKeyedStateBackend                            RT/runtime/state/heap/HeapKeyedStateBackend.java
└ registeredKVStates: 状态名 → StateTable          :131   每个状态描述符一张 StateTable
   └ StateTable                                   RT/runtime/state/heap/StateTable.java
      └ keyGroupedStateMaps: StateMap[]           :79    数组下标对应本子任务负责的每个 KeyGroup
         └ CopyOnWriteStateMap                    每个 KeyGroup 一张哈希表：(key, namespace) → 状态值
```

`HeapValueState.value()`（`RT/runtime/state/heap/HeapValueState.java:71`）调用 `stateTable.get(namespace)`（`StateTable.java:145`）：先按当前 KeyGroup 找到对应的那张表，再按 key 和 namespace 查值。

**namespace 是什么？** 普通 keyed state 的 namespace 是一个固定的空值。**窗口状态**的 namespace 是窗口本身，所以同一个 key 在不同窗口里的状态互不干扰（第七讲会用到）。

**为什么要按 KeyGroup 分表？** 做快照时，可以按 KeyGroup 的顺序依次写出，并记下每个 KeyGroup 在文件里的起始位置（`RT/runtime/state/heap/HeapSnapshotStrategy.java:178`）。扩缩容时，新的子任务只需要跳到自己负责的那几段去读，不用读整个文件。这一点在第五节会用到。

### 4.2 写时复制：快照为什么不阻塞数据处理

第二讲说过，快照的同步阶段在 Task 线程上执行，会挡住数据处理，所以要尽可能快。HashMap 状态后端的同步阶段很快，秘密在 `RT/runtime/state/heap/CopyOnWriteStateMap.java` 里的**两个版本号**：

| 字段 | 行号 | 含义 |
|---|---|---|
| `stateMapVersion` | `:177` | 这张表当前的版本，每做一次快照就加 1 |
| `highestRequiredSnapshotVersion` | `:180` | 还没完成的快照里，最新的那个版本 |
| 每个条目上的版本号 | — | 这个条目最后一次被修改时，表的版本 |

**同步阶段**（`snapshotMapArrays()`，`:476`）：把版本号加 1（`:486`），记下"有一个快照需要这个版本"（`:493-494`），然后**只复制哈希表的桶数组**。这是浅拷贝，只复制引用，不复制任何数据，所以很快。

**之后的读写**（以 `get()` 为例，`:268`）：

```java
if (e.stateVersion < requiredVersion) {               // :281  这个值是快照之前写的，快照还要用它
    if (e.entryVersion < requiredVersion) {
        e = handleChainedEntryCopyOnWrite(...);        // :284  连同链表上的条目一起复制
    }
    e.stateVersion = stateMapVersion;
    e.state = getStateSerializer().copy(e.state);      // :287  复制一份值：新的给 Task 用，旧的留给快照
}
return e.state;
```

注意，**连 `get()` 也会触发复制**。因为你拿到对象后可能会直接修改它（比如往一个 List 里加元素），如果不复制，就会改到快照里的数据。

**异步阶段**：快照线程遍历那份旧的桶数组。旧的条目不会再被修改，可以放心地序列化（`HeapSnapshotStrategy.java:97` → `:172`）。写完后调用 `releaseSnapshot()`（`CopyOnWriteStateMap.java:457`），之后的修改就不用再复制了。

**代价**：
- 快照期间被修改的 key，会**多占一份内存**。状态大、更新频繁时，Checkpoint 期间堆内存会明显上涨；
- 快照是**全量**的，每次都要把全部状态序列化一遍；
- 所有状态都在 JVM 堆里，状态越大，GC 压力越大。

所以 HashMap 状态后端适合**小状态、低延迟**的场景。状态大了，就该考虑下篇的 RocksDB。

---

## 五、扩缩容：状态是怎么切开的

<figure class="ai-figure"><img src="/bigdata-img/flink/post-03a-2.webp" alt="扩缩容时不同子任务按KeyGroup区间接收状态" width="960" height="640" loading="lazy" /><figcaption>扩缩容时不同子任务按KeyGroup区间接收状态<span>AI 生成配图</span></figcaption></figure>

### 5.1 JobManager：按 KeyGroup 区间求交集

作业从 Checkpoint 恢复时，在构建 ExecutionGraph 的过程中（第一讲下篇），会调用 `StateAssignmentOperation.assignStates()`（`RT/runtime/checkpoint/StateAssignmentOperation.java:105`）分配状态：

1. **按新的并行度划分区间**：`createKeyGroupPartitions()`（`:710`），用的就是 2.2 节的公式 ②；
2. **求交集**：`reDistributeKeyedStates()`（`:312`）→ `extractIntersectingState()`（`:679`）。对每个新的子任务，遍历**所有旧子任务**的状态句柄，取出和自己区间相交的那部分。

以实验一为例，并行度从 2 改成 3：

| 新子任务 | 新区间 | 旧子任务 0 [0, 63] | 旧子任务 1 [64, 127] |
|---|---|---|---|
| 1/3 | [0, 42] | ✅ 取 [0, 42] | — |
| 2/3 | [43, 85] | ✅ 取 [43, 63] | ✅ 取 [64, 85] |
| 3/3 | [86, 127] | — | ✅ 取 [86, 127] |

中间那个新子任务，要从两个旧子任务那里各取一段。

HashMap 状态后端的状态句柄，求交集的方法是 `KeyGroupsStateHandle.getIntersection()`（`RT/runtime/state/KeyGroupsStateHandle.java:104`）：根据 4.1 节记下的偏移表，只保留相交的那些 KeyGroup 的位置。恢复时，TaskManager 按这些位置跳过去读就行。

### 5.2 Operator State：没有 key，怎么分？

除了 keyed state，还有一种**没有 key** 的状态，叫 operator state。最典型的是 Source 记录的读取位置，比如 Kafka 每个分区读到了哪个 offset。它不能按 KeyGroup 分，Flink 提供了三种方式（`RT/runtime/checkpoint/RoundRobinOperatorStateRepartitioner.java:54`）：

| 方式 | 获取方法 | 扩缩容时 |
|---|---|---|
| 均分 | `getListState()` | 把所有子任务的列表拼起来，再平均切给新的子任务 |
| 全量 | `getUnionListState()` | 每个子任务都拿到**全部**元素，自己挑需要的 |
| 广播 | `getBroadcastState()` | 每个子任务都拿到一份完整的副本 |

---

## 六、回到开头：maxParallelism 到底能不能改

### 6.1 显式改了，就恢复不了

2.3 节说过：maxParallelism 一变，同一个 key 就会落进不同的 KeyGroup。而 Checkpoint 里的状态，是**按旧的 KeyGroup 组织的**。如果允许恢复，新的子任务会去错误的 KeyGroup 里找状态，结果就是状态"丢了"。

所以恢复时有一道检查（`RT/runtime/checkpoint/Checkpoints.java:175-192`）：

```java
if (executionJobVertex.getMaxParallelism() == operatorState.getMaxParallelism()
        || executionJobVertex.canRescaleMaxParallelism(operatorState.getMaxParallelism())) {
    operatorStates.put(...);                    // 一致，或者允许调整：通过
} else {
    throw new IllegalStateException("... Max parallelism mismatch between checkpoint/savepoint state and new program ...");
}
```

实验二抛出的就是这个异常。

### 6.2 没有显式设置，会沿用旧值

注意判断条件里的第二部分：`canRescaleMaxParallelism()`。什么时候允许调整？答案在 `RT/runtime/scheduler/SchedulerBase.java:357-377`：

```java
int maxParallelism = vertex.getMaxParallelism();
// if no max parallelism was configured by the user, we calculate and set a default
if (maxParallelism == JobVertex.MAX_PARALLELISM_DEFAULT) {
    maxParallelism = defaultMaxParallelismFunc.apply(vertex);
    autoConfigured = true;                 // ← 自动推算出来的
} else {
    autoConfigured = false;                // ← 用户显式设置的
}
...
// Allow rescaling if the max parallelism was not set explicitly by the user
(newMax) -> autoConfigured ? Optional.empty() : Optional.of("Cannot override a configured max parallelism.")
```

- **你没有显式设置**：Flink 推算出来的默认值，可以被 Checkpoint 里的值**覆盖**。实验三就是这种情况：推算值是 128，但 Checkpoint 里是 256，最终生效的是 256。
- **你显式设置了**：就不允许覆盖，两边不一致就报错。实验二就是这种情况。

### 6.3 这意味着什么

把两点合在一起，会得到一个容易被忽略的结论：

**如果你从没设置过 maxParallelism，它的值就锁定在作业第一次启动时推算出来的那个数上。** 比如作业上线时并行度是 10，默认 maxParallelism 是 128。之后不管你怎么从 Checkpoint 恢复，它都会一直沿用 128。哪天业务量涨了，想把并行度调到 200，是做不到的，因为并行度不能超过 maxParallelism。到那时再想改，就只能丢掉状态重新开始。

**生产建议**：
- **作业上线的第一天，就显式设置 `pipeline.max-parallelism`**（`flink-core/.../configuration/PipelineOptions.java:190`），按未来可能的最大并行度留出余量；
- 设置之后**不要再改**；
- 也不要设得过大。KeyGroup 越多，每个子任务要维护的分片越多，下篇会看到，它还会影响 RocksDB 里每条数据的 key 长度。

---

## 七、自己动手

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/state/` 下：

```bash
# 2.3 节：key 的分布、默认 maxParallelism
./run.sh study.state.KeyGroupDemo

# 实验一：并行度 2 运行 22 秒，再从 Checkpoint 恢复成并行度 3
./run.sh study.state.StateBackendDemo hashmap 2 22
KEEP_CKPT=1 ./run.sh study.state.StateBackendDemo hashmap 3 22 /tmp/flink-state-demo/<jobId>/chk-5

# 实验二：恢复时把 maxParallelism 改成 256
KEEP_CKPT=1 ./run.sh -Dmaxp=256 study.state.StateBackendDemo hashmap 2 22 /tmp/flink-state-demo/<jobId>/chk-5

# 实验三：用 256 生成 Checkpoint，恢复时不设置（-Dmaxp=-1）
./run.sh -Dmaxp=256 study.state.StateBackendDemo hashmap 2 22
KEEP_CKPT=1 ./run.sh -Dmaxp=-1 study.state.StateBackendDemo hashmap 2 22 /tmp/flink-state-demo/<jobId>/chk-5
```

（`KEEP_CKPT=1` 让脚本不要清空 Checkpoint 目录。`<jobId>` 换成上一次运行生成的目录名。）

**断点清单**：

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `AbstractKeyedStateBackend.java:262` | 每条数据算一次 KeyGroup |
| 2 | `AbstractKeyedStateBackend.java:378` | 状态对象的创建 |
| 3 | `StateTable.java:145` | 按 KeyGroup 找到对应的表 |
| 4 | `CopyOnWriteStateMap.java:476` | 快照的同步阶段：只复制桶数组 |
| 5 | `CopyOnWriteStateMap.java:281`（在 Checkpoint 期间命中） | 写时复制 |
| 6 | `StateAssignmentOperation.java:679`（带恢复路径运行） | 按区间求交集 |
| 7 | `Checkpoints.java:175`（实验二） | maxParallelism 检查 |
| 8 | `SchedulerBase.java:360`（实验三） | 默认值是不是"自动推算"的 |

---

## 八、课后练习

1. **缩容**：从并行度 2 的 Checkpoint 恢复成并行度 1，每个子任务会恢复多少个 key？对照 5.1 节的表格推导。
2. **原地修改**：把计数改成 `ValueState<long[]>`，用 `value()[0]++` 直接修改数组、**不调用** `update()`。用 HashMap 状态后端运行，计数会增长吗？为什么？（下篇会揭晓换成 RocksDB 之后的结果）
3. **推导**：并行度 100 时默认的 maxParallelism 是 256。如果一个作业以并行度 100 上线，之后想扩容到 300，能做到吗？结合 6.2 节解释。
4. **读代码**：`getUnionListState()` 在扩缩容时会把全部元素发给每个子任务。如果有 1000 个子任务、每个子任务存 1000 个元素，恢复时会有什么问题？

---

## 写在最后

这一篇的核心只有一句话：**状态不是按 key 分配的，而是按 KeyGroup 分配的。** KeyGroup 的数量由 maxParallelism 决定，一旦确定就不能改；并行度只是决定"一个子任务负责几个 KeyGroup"，所以可以随便调整。

下一篇，我们看状态大到 JVM 堆放不下时该怎么办：**RocksDB 状态后端**。一条状态在 RocksDB 里长什么样？增量 Checkpoint 是怎么做到"每次只上传几十 KB"的？扩缩容时它又是怎么切分状态的？最后再看看 Flink 2.x 的存算分离状态后端 ForSt。


**留一个问题**：你的生产作业显式设置了 `pipeline.max-parallelism` 吗？

下一篇：**Flink 2.x 源码精读（三·下）：RocksDB、增量 Checkpoint 与 ForSt**
:::
