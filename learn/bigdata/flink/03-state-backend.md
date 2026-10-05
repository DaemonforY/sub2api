---
title: "Flink 源码导读 03：State Backend —— 状态怎么存、怎么快照、怎么扩缩容"
description: "HashMap 与 RocksDB 状态后端的存储方式、增量 Checkpoint、KeyGroup 与扩缩容时的状态重分布。"
bigdata: "flink"
lesson: "f3"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/03-state-backend-cover.webp"}]]
---

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/03-state-backend-cover.webp" alt="Yui和Kai展示状态后端全景" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai展示状态后端全景<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`，`RDB` = `flink-state-backends/flink-statebackend-rocksdb/src/main/java/org/apache/flink/state/rocksdb`，`FST` = `flink-state-backends/flink-statebackend-forst/src/main/java/org/apache/flink/state/forst`
> 前置阅读：[02 Checkpoint 全流程](/bigdata/flink/02-checkpoint)（4.4 节同步/异步快照）
> 配套示例：`demos/src/main/java/study/state/`，第 9 节的所有数据都是在本机实测的

## 0. 本篇要回答的问题

1. `getRuntimeContext().getState(...)` 拿到的对象背后是什么？`value()` 怎么知道"当前 key"是谁？
2. **KeyGroup** 是什么？为什么 `maxParallelism` 设定后就不能改？
3. HashMap 状态后端怎么做到"快照时不阻塞处理"？
4. RocksDB 里的一条状态长什么样？增量 Checkpoint 怎么复用文件？
5. 从 2 个并行度扩到 3 个时，状态是怎么切分和合并的？
6. Flink 2.x 的 ForSt 和异步状态（State V2）解决了什么问题？

## 1. 整体结构：两个正交的概念

| 概念 | 负责什么 | 实现 | 配置 |
|---|---|---|---|
| **State Backend** | 状态在 **Task 运行时**存放在哪、怎么读写、怎么做快照 | `hashmap`（JVM 堆）、`rocksdb`（本地磁盘）、`forst`（远端 DFS + 本地缓存） | `state.backend.type`（`flink-core/.../StateBackendOptions.java:47`） |
| **Checkpoint Storage** | 快照**写到哪** | `JobManagerCheckpointStorage`（JM 内存，仅测试用）、`FileSystemCheckpointStorage`（HDFS/S3/本地） | `execution.checkpointing.storage` / `execution.checkpointing.dir` |

从 1.13 起，这两个概念被拆开了。老版本的 `FsStateBackend`、`MemoryStateBackend` 实际上是"HashMap 状态后端 + 某种存储"的组合，2.x 中已经删除。

### 1.1 接口

`RT/runtime/state/StateBackend.java:81`

```java
<K> CheckpointableKeyedStateBackend<K> createKeyedStateBackend(...)       // :104  keyed state
OperatorStateBackend createOperatorStateBackend(...)                       // :151  operator state（如 Source 的 offset）
default boolean supportsAsyncKeyedStateBackend()                           // :136  2.x 新增：是否支持异步状态
default <K> AsyncKeyedStateBackend<K> createAsyncKeyedStateBackend(...)   // :120
```

**每个 subtask 的每个算子**都有自己的 keyed backend 和 operator backend 实例。keyed backend 只负责这个 subtask 所分配到的那一段 KeyGroup（第 2 节）。

### 1.2 加载

Task 初始化时调用 `StateBackendLoader.fromApplicationOrConfigOrDefault()`（`RT/streaming/runtime/tasks/StreamTask.java:1680` → `RT/runtime/state/StateBackendLoader.java:265`）。优先级是：代码里 `env.setStateBackend(...)` > 配置 `state.backend.type` > 默认 `hashmap`。

`loadStateBackendFromConfig()`（`:106`）按名字分派：`"hashmap"`（`:119`）直接 new；`"rocksdb"`（`:127`）和 `"forst"`（`:134`）通过**反射加载 Factory 类**（`:61`、`:65`）。因为它们在独立的 jar 里，`flink-runtime` 不直接依赖它们，这样 RocksDB/ForSt 的 native 库也不会被强制打进核心包。

## 2. KeyGroup：Keyed State 的最小分配单位

<figure class="ai-figure"><img src="/bigdata-img/flink/03-state-backend-1.webp" alt="KeyGroup像分段轨道承接不同key" width="960" height="640" loading="lazy" /><figcaption>KeyGroup像分段轨道承接不同key<span>AI 生成配图</span></figcaption></figure>

### 2.1 为什么需要 KeyGroup？

如果直接用 `hash(key) % parallelism` 来分区，并行度一变，**几乎所有 key 都要换 subtask**，状态就得按 key 逐条重新分发，代价极高。

Flink 的做法是加一个中间层：

```
key ──murmurHash──► KeyGroup（0 ~ maxParallelism-1，固定不变）──连续区间──► subtask
```

- key → KeyGroup 只依赖 `maxParallelism`，**和并行度无关**
- 每个 subtask 负责一段**连续的** KeyGroup 区间
- 扩缩容时只需要**重新划分区间**，状态以 KeyGroup 为单位整块迁移

### 2.2 源码

`RT/runtime/state/KeyGroupRangeAssignment.java`

```java
// key → KeyGroup（:63 → :75）
public static int computeKeyGroupForKeyHash(int keyHash, int maxParallelism) {
    return MathUtils.murmurHash(keyHash) % maxParallelism;             // :76
}

// subtask i 负责的区间（:93）
int start = ((operatorIndex * maxParallelism + parallelism - 1) / parallelism);
int end   = ((operatorIndex + 1) * maxParallelism - 1) / parallelism;

// KeyGroup → subtask（:124）
return keyGroupId * parallelism / maxParallelism;

// 默认 maxParallelism（:137）：roundUpToPowerOfTwo(p * 1.5)，下限 128（:32），上限 32768（:35）
```

**同一套算法用在两个地方，保证它们一定一致**：
- **网络分区**：`keyBy` 用到的 `KeyGroupStreamPartitioner.selectChannel()` → `assignKeyToParallelOperator()`（`RT/streaming/runtime/partitioner/KeyGroupStreamPartitioner.java:63`），决定数据发给哪个 subtask
- **状态访问**：`AbstractKeyedStateBackend.setCurrentKey()`（`RT/runtime/state/AbstractKeyedStateBackend.java:258-262`）算出当前 key 的 KeyGroup，决定状态存到哪个分片

### 2.3 为什么 maxParallelism 不能改？

因为 `murmurHash(key) % maxParallelism` 的结果取决于 maxParallelism。**改了它，同一个 key 会落到不同的 KeyGroup**，旧 Checkpoint 中按 KeyGroup 组织的状态就对不上了。`StateAssignmentOperation.checkParallelismPreconditions()`（`RT/runtime/checkpoint/StateAssignmentOperation.java:730`）在恢复时会检查这一点，不匹配就直接报错。

**生产建议**：上线第一天就显式设置 `pipeline.max-parallelism`（`flink-core/.../PipelineOptions.java:191`），留足扩容空间。默认值由初始并行度推算（`SchedulerBase.java:327`），并行度 10 时默认只有 128，意味着以后最多只能扩到 128 个并行度。

### 2.4 🧪 实验：KeyGroupDemo（解开第 02 篇的疑问）

第 02 篇做背压实验时，16 个 key 在 2 个并行度下分成了 7:9。`KeyGroupDemo` 直接调用 `KeyGroupRangeAssignment` 来验证：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.state.KeyGroupDemo
```

实测输出（节选）：

```
key=0  -> keyGroup=94  -> subtask 1        key=8  -> keyGroup=15  -> subtask 0
key=1  -> keyGroup=86  -> subtask 1        key=9  -> keyGroup=51  -> subtask 0
key=2  -> keyGroup=127 -> subtask 1        key=10 -> keyGroup=67  -> subtask 1
key=3  -> keyGroup=113 -> subtask 1        key=11 -> keyGroup=102 -> subtask 1
key=4  -> keyGroup=7   -> subtask 0        key=12 -> keyGroup=50  -> subtask 0
key=5  -> keyGroup=126 -> subtask 1        key=13 -> keyGroup=1   -> subtask 0
key=6  -> keyGroup=18  -> subtask 0        key=14 -> keyGroup=27  -> subtask 0
key=7  -> keyGroup=113 -> subtask 1        key=15 -> keyGroup=70  -> subtask 1
subtask0 有 7 个 key，subtask1 有 9 个 key

parallelism=2: subtask0=[0, 63]  subtask1=[64, 127]
parallelism=3: subtask0=[0, 42]  subtask1=[43, 85]  subtask2=[86, 127]

key="user-42", maxParallelism=128 -> keyGroup=20
key="user-42", maxParallelism=256 -> keyGroup=148     ← 改了 maxParallelism，KeyGroup 就变了
```

几个观察：
- **key 很少时，数据倾斜几乎不可避免**：16 个 key 散到 128 个 KeyGroup 上，每个 subtask 分到几个全凭运气。key 3 和 key 7 甚至落进了**同一个** KeyGroup（113）
- 默认 maxParallelism：p=100 → 256，p=1000 → 2048，p=30000 → 32768

---

## 3. 状态访问路径：`getState()` 与 `value()`

### 3.1 获取状态对象

```
StreamingRuntimeContext.getState(descriptor)          RT/streaming/api/operators/StreamingRuntimeContext.java:204
└ DefaultKeyedStateStore.getState()                   RT/runtime/state/DefaultKeyedStateStore.java:85
  └ getPartitionedState()                             :142 → keyedStateBackend.getPartitionedState(...)（:150）
    └ AbstractKeyedStateBackend.getOrCreateKeyedState()   AbstractKeyedStateBackend.java:378
       ├ 先查缓存 keyValueStatesByName（:66、:387），同名状态只创建一次
       └ TtlStateFactory.createStateAndWrapWithTtlIfEnabled()   RT/runtime/state/ttl/TtlStateFactory.java:56
          ├ 没配 TTL → backend.createOrUpdateInternalState(...)       各状态后端自己实现
          └ 配了 TTL → 外面再包一层，如 TtlValueState（:145），值里额外存一个时间戳
```

你拿到的 `ValueState` 是一个**绑定到整个 subtask 的对象**，而不是绑定到某个 key。那 `value()` 怎么知道读哪个 key？

### 3.2 "当前 key" 的传递

回忆 01 篇 6.4 节：数据进入 keyed 算子之前，`recordProcessor` 会先调用 `setKeyContextElement`：

```
AbstractStreamOperator.setKeyContextElement1(record)     RT/streaming/api/operators/AbstractStreamOperator.java:569
└ setCurrentKey(keySelector.getKey(value))               :599
  └ StreamOperatorStateHandler.setCurrentKey()           RT/streaming/api/operators/StreamOperatorStateHandler.java:450
    └ AbstractKeyedStateBackend.setCurrentKey(newKey)     AbstractKeyedStateBackend.java:258
        keyContext.setCurrentKey(newKey);
        keyContext.setCurrentKeyGroupIndex(KeyGroupRangeAssignment.assignToKeyGroup(newKey, numberOfKeyGroups));
```

之后 `value()` 就从 `keyContext` 里取出当前 key 和 KeyGroup。所以：
- 在 `processElement` 之外（比如自己起的线程里）访问 keyed state，读到的是**上一条记录的 key**，结果是错的
- 每条记录都要计算一次 murmurHash，这是 keyed 算子固定的 CPU 开销

---

## 4. HashMap 状态后端

### 4.1 数据结构：每个 KeyGroup 一张 Map

```
HeapKeyedStateBackend                         RT/runtime/state/heap/HeapKeyedStateBackend.java
└ registeredKVStates: Map<状态名, StateTable>  :131   每个 StateDescriptor 一张 StateTable
   └ StateTable                                RT/runtime/state/heap/StateTable.java
      └ keyGroupedStateMaps: StateMap[]        :79    数组下标 = keyGroup - 本 subtask 起始 keyGroup
         └ CopyOnWriteStateMap                 每个 KeyGroup 一个，(key, namespace) → state
```

`HeapValueState.value()`（`RT/runtime/state/heap/HeapValueState.java:71`）→ `stateTable.get(currentNamespace)`（`StateTable.java:145`）：

```java
return get(keyContext.getCurrentKey(), keyContext.getCurrentKeyGroupIndex(), namespace);
// → getMapForKeyGroup(keyGroupIndex)（:325）→ keyGroupedStateMaps[keyGroupIndex - keyGroupOffset]（:335）
//   → CopyOnWriteStateMap.get(key, namespace)
```

**namespace** 是什么？普通 keyed state 的 namespace 是 `VoidNamespace`；**窗口状态**的 namespace 是窗口本身，所以同一个 key 在不同窗口里的状态互不干扰。

**为什么按 KeyGroup 分片？** 快照时可以按 KeyGroup 顺序写出，并记录每个 KeyGroup 在文件中的偏移（`KeyGroupRangeOffsets`，`HeapSnapshotStrategy.java:178`）。扩缩容时，新的 subtask 只需要 seek 到自己负责的那几段来读，不必读整个文件。

### 4.2 ★ Copy-On-Write：快照不阻塞处理的秘密

第 02 篇说过，HashMap 后端的同步阶段"只做一个 copy-on-write 快照，很快"。它是怎么做到的？

`RT/runtime/state/heap/CopyOnWriteStateMap.java`，关键是**两个版本号**：

| 字段 | 行号 | 含义 |
|---|---|---|
| `stateMapVersion` | `:177` | 当前 map 的版本，每做一次快照就 +1 |
| `highestRequiredSnapshotVersion` | `:180` | 还没释放的快照中最新的那个版本 |
| 每个 entry 上的 `entryVersion` / `stateVersion` | — | 这个 entry（或它的值）最后一次被修改时的 map 版本 |

**同步阶段**（`snapshotMapArrays()`，`:476`）：

```java
++stateMapVersion;                                        // :486
highestRequiredSnapshotVersion = stateMapVersion;         // :493
snapshotVersions.add(highestRequiredSnapshotVersion);     // :494
// 然后只复制哈希表的"桶数组"（浅拷贝，O(桶数)），不复制任何 entry
```

**之后的读写**（`get()`，`:268`；`putAndGetOld()`，`:323`）：

```java
if (e.stateVersion < requiredVersion) {         // 这个值是快照之前写的，快照还要用
    if (e.entryVersion < requiredVersion) {
        e = handleChainedEntryCopyOnWrite(...);  // 连 entry 链一起复制一份
    }
    e.stateVersion = stateMapVersion;
    e.state = getStateSerializer().copy(e.state);   // 复制值：新版本给 Task 用，旧版本留给快照
}
```

注意 **`get()` 也会触发复制**。因为用户可能拿到对象后原地修改（比如 `list.add(x)`），如果不复制，就会改到快照里的数据。

**异步阶段**：快照线程遍历浅拷贝出来的旧桶数组，旧 entry 不会再被修改，可以安全地序列化（`HeapSnapshotStrategy.asyncSnapshot()`，`:97` → `writeStateInKeyGroup()`，`:172`）。写完后 `releaseSnapshot()`（`:457`）降低 `highestRequiredSnapshotVersion`，之后的修改就不用再复制了。

**代价**：
- 快照期间被修改的 key 会**多占一份内存**。状态大、更新频繁时，Checkpoint 期间堆内存会明显上涨
- 快照是**全量**的：每次都要序列化全部状态（HashMap 后端不支持增量 Checkpoint）
- 状态必须全部放进堆内存，GC 压力随之增大

---

## 5. RocksDB 状态后端

### 5.1 一条状态在 RocksDB 里长什么样？

- **每个 StateDescriptor 对应一个 Column Family**：`createOrUpdateInternalState()`（`RDB/RocksDBKeyedStateBackend.java:1022`）→ `tryRegisterKvStateInformation()` → `RocksDBOperationUtils.createColumnFamily()`（`RDB/RocksDBOperationUtils.java:278`）
- 整个 subtask 共享**一个 RocksDB 实例**（`db`，`RocksDBKeyedStateBackend.java:268`）

**RocksDB 的 key 是一个复合的字节串**（`RT/runtime/state/SerializedCompositeKeyBuilder.java`）：

```
┌──────────────────┬──────────────────┬───────────────────────┐
│ KeyGroup 前缀     │ 序列化后的 key    │ 序列化后的 namespace   │  → value: 序列化后的状态值
│ 1 或 2 字节       │                  │                       │
└──────────────────┴──────────────────┴───────────────────────┘
```

- KeyGroup 前缀：`CompositeKeySerializationUtils.writeKeyGroup()`（`RT/runtime/state/CompositeKeySerializationUtils.java:109`）按**大端序**写入。字节数由 `computeRequiredBytesInKeyGroupPrefix()`（`:169`）决定：maxParallelism ≤ 128 用 1 字节，否则用 2 字节
- **KeyGroup 放在最前面**是整个设计的关键：RocksDB 按 key 的字节序排序，所以同一个 KeyGroup 的数据在物理上是**连续的**。扩缩容时，可以用 `deleteRange` 一次性删掉不属于自己的一整段（5.4 节）

`RocksDBValueState.value()`（`RDB/RocksDBValueState.java:79`）：

```java
byte[] valueBytes = backend.db.get(columnFamily, serializeCurrentKeyWithGroupAndNamespace());   // :82
```

`setCurrentKey()` 时就已经把 KeyGroup 和 key 序列化进了共享的 `sharedRocksKeyBuilder`（`RocksDBKeyedStateBackend.java:261`、`:523`），每次访问只需要追加 namespace。

**RocksDB 与 Heap 的关键区别**：
- **每次读写都要序列化/反序列化**，所以单次访问比 Heap 慢得多，但状态大小只受磁盘限制
- **`value()` 拿到的是一个新对象**，原地修改它不会写回去，必须调用 `update()`。Heap 后端恰好相反，原地修改"碰巧"能生效。这是从 Heap 切到 RocksDB 时最常见的 bug

### 5.2 内存管理

RocksDB 的 block cache 和 memtable 默认从 Flink 的 **Managed Memory** 里分配。每个 slot 内的所有 RocksDB 实例共享一个 `LRUCache` 和一个 `WriteBufferManager`，总量不会超过 slot 的托管内存（`RDB/RocksDBMemoryControllerUtils.java:47` `allocateRocksDBSharedResources`）。这就是 01 篇 4.5 节 `setManagedMemoryFraction` 分配给算子的那部分内存。

### 5.3 ★ 增量 Checkpoint

开启 `execution.checkpointing.incremental: true`（`flink-core/.../CheckpointingOptions.java:133`）后：

**同步阶段**：`RocksDBSnapshotStrategyBase.syncPrepareResources()`（`RDB/snapshot/RocksDBSnapshotStrategyBase.java:151`）→ `takeDBNativeCheckpoint()`（`:161`、`:170`）：

```java
Checkpoint checkpoint = Checkpoint.create(db);                          // :174
checkpoint.createCheckpoint(outputDirectory.getDirectory().toString()); // :175
```

这是 RocksDB 原生的 Checkpoint：flush memtable，然后对所有 SST 文件做**硬链接**。SST 文件本身是不可变的，所以硬链接出来的就是一份一致的快照，耗时通常只有几毫秒到几十毫秒。

**异步阶段**：`RocksIncrementalSnapshotStrategy`（`RDB/snapshot/RocksIncrementalSnapshotStrategy.java`）

```java
// createUploadFilePaths()（:413）
if (fileName.endsWith(SST_FILE_SUFFIX)) {
    Optional<StreamStateHandle> uploaded = previousSnapshot.getUploaded(fileName);   // :422
    if (uploaded.isPresent() && ...couldReuseStateHandle(uploaded.get())) {
        sstFiles.add(HandleAndLocalPath.of(uploaded.get(), fileName));   // 以前传过，直接复用句柄
    } else {
        sstFilePaths.add(filePath);                                      // 新文件，需要上传
    }
} else {
    miscFilePaths.add(filePath);   // MANIFEST、OPTIONS 等，每次都上传
}
```

- **"以前传过"是按 SST 文件名判断的**。RocksDB 的 SST 文件名是单调递增的编号，且内容不可变，所以同名即同内容
- 记录"已上传文件"的是 `uploadedSstFiles`（`:85`），只有 Checkpoint **确认完成**之后，才会被当作复用的基准（`notifyCheckpointComplete()`，`:167`）。否则，如果一个失败的 Checkpoint 上传的文件被复用了，就可能引用到已经被删掉的文件
- 上传的 SST 文件放在 Checkpoint 目录的 **`shared/`** 下，因为它们会被多个 Checkpoint 共享

### 5.4 JM 侧：共享文件什么时候删？

多个 Checkpoint 共享同一个 SST 文件，所以不能在删除旧 Checkpoint 时顺手把它的文件也删掉。管理这件事的是 **`SharedStateRegistry`**（`RT/runtime/state/SharedStateRegistryImpl.java`）：

1. 收到 ack 时注册共享状态：`CheckpointCoordinator.java:1240` → `OperatorSubtaskState.registerSharedStates()` → `registerReference()`（`SharedStateRegistryImpl.java:81`），记录每个文件**最后一次被哪个 Checkpoint 使用**（`lastUsedCheckpointID`，`:313`）
2. 旧 Checkpoint 被淘汰时：`AbstractCompleteCheckpointStore.unregisterUnusedState()`（`RT/runtime/checkpoint/AbstractCompleteCheckpointStore.java:56-57`）找出仍保留的 Checkpoint 中 ID 最小的那个 → `unregisterUnusedState(lowestCheckpointID)`（`SharedStateRegistryImpl.java:159`）：`lastUsedCheckpointID` 比它还小的文件，就没有任何保留中的 Checkpoint 在用了，异步删除（`scheduleAsyncDelete`，`:243`）

第 9.2 节会用真实的 `shared/` 目录验证这个过程。

### 5.5 小文件内联

小于 `execution.checkpointing.data-inline-threshold`（默认 20KB，`CheckpointingOptions.java:240`）的状态文件不会单独写成文件，而是**直接内联进 `_metadata`**。所以实验中 `chk-N/` 目录下只有一个 `_metadata`，而且它越来越大：MANIFEST 等小文件都内联在里面。

---

## 6. 扩缩容：状态怎么重新分配？

<figure class="ai-figure"><img src="/bigdata-img/flink/03-state-backend-2.webp" alt="扩缩容时按KeyGroup区间搬迁状态" width="960" height="640" loading="lazy" /><figcaption>扩缩容时按KeyGroup区间搬迁状态<span>AI 生成配图</span></figcaption></figure>

### 6.1 JM 侧：按 KeyGroup 区间求交集

作业从 Checkpoint 恢复时，01 篇 4.7 节创建 ExecutionGraph 的过程中会调用 **`StateAssignmentOperation.assignStates()`**（`RT/runtime/checkpoint/StateAssignmentOperation.java:105`）：

**Keyed State**：
1. `createKeyGroupPartitions()`（`:710`）：按新的并行度，为每个新 subtask 算出 KeyGroup 区间
2. `reDistributeKeyedStates()`（`:312`）→ `getManagedKeyedStateHandles()`（`:614`）→ `extractIntersectingState()`（`:679`）：对每个新 subtask，遍历**所有旧 subtask** 的状态句柄，调用 `keyedStateHandle.getIntersection(新区间)`
   - Heap 的 `KeyGroupsStateHandle.getIntersection()`（`RT/runtime/state/KeyGroupsStateHandle.java:104`）：根据偏移表，只保留相交 KeyGroup 的偏移。恢复时 seek 过去读
   - RocksDB 的增量句柄（`AbstractIncrementalStateHandle.java:101`）：**无法按 KeyGroup 切分 SST 文件**，所以只是把句柄的区间标记为交集，裁剪工作留给 TM 恢复时去做（6.2 节）

**Operator State**（如 Kafka Source 的 partition offset）没有 key，按分发模式处理（`RoundRobinOperatorStateRepartitioner.repartitionState()`，`RT/runtime/checkpoint/RoundRobinOperatorStateRepartitioner.java:54`）：

| 模式 | API | 扩缩容时 |
|---|---|---|
| `SPLIT_DISTRIBUTE` | `getListState()` | 把所有 subtask 的列表拼起来，再均匀切分 |
| `UNION` | `getUnionListState()` | 每个 subtask 都拿到**全部**元素，自己挑需要的 |
| `BROADCAST` | `getBroadcastState()` | 每个 subtask 拿到一份完整副本 |

### 6.2 TM 侧：RocksDB 恢复时的裁剪与合并

`RocksDBIncrementalRestoreOperation.restoreFromLocalState()`（`RDB/restore/RocksDBIncrementalRestoreOperation.java:319`）：

```java
if (localKeyedStateHandles.size() == 1) {
    // This happens if we don't rescale and for some scale out scenarios.
    initBaseDBFromSingleStateHandle(localKeyedStateHandles.get(0));
} else {
    // This happens for all scale ins and some scale outs.
    restoreFromMultipleStateHandles(localKeyedStateHandles);     // :422
}
```

- **只有一个来源**：下载 SST 后直接打开，再调用 `clipDBWithKeyGroupRange()`（`RDB/RocksDBIncrementalCheckpointUtils.java:134`），用 `deleteRange` 删掉区间两端多出来的 KeyGroup。因为 KeyGroup 是 key 的前缀，一段 KeyGroup 对应一个连续的字节区间，删除非常快
- **多个来源**：挑一个作为**基础 DB** 打开并裁剪，再把其他来源中属于自己区间的数据导入进来。如果开启了 `useIngestDbRestoreMode`（`:143`），会用 RocksDB 的 ingest 机制直接导入 SST 文件（`mergeStateHandlesWithClipAndIngest()`，`:467`），比逐条复制快得多

第 9.3 节有真实的日志。

---

## 7. ForSt 与异步状态（State V2）：Flink 2.x 的存算分离

### 7.1 为什么需要？

在云原生部署中，本地盘的 RocksDB 有几个痛点：
- 状态大小受限于本地盘，扩缩容和恢复时要**下载全部 SST**，大状态作业恢复需要十几分钟甚至更久
- Checkpoint 要把 SST 从本地盘**上传**到 DFS，占用网络和 IO

**ForSt**（FLIP-423 系列）是 RocksDB 的一个分支，它的**主存储直接放在 DFS**（`state.backend.forst.primary-dir`，`FST/ForStOptions.java:58`），本地盘只作为缓存（`cache.dir`、`cache.size-based-limit`，`:82`、`:93`）。于是 Checkpoint 基本不需要上传数据，恢复时也不需要下载，理论上可以做到秒级恢复和扩缩容。

新问题随之而来：**读 DFS 的延迟是毫秒级的**。如果还按同步的方式 `value()` 一条条去读，Task 线程大部分时间都会卡在 IO 上，吞吐量会急剧下降。

### 7.2 异步状态 API

所以 2.x 引入了 **State V2**（`flink-core-api/src/main/java/org/apache/flink/api/common/state/v2/`）：

```java
StateFuture<T> asyncValue();              // ValueState.java:49
StateFuture<Void> asyncUpdate(T value);   // :60
T value();                                // :73  同步版本仍然保留
```

使用方式（官方示例 `flink-examples/.../statemachine/StateMachineExample.java:166`、`:218`）：

```java
events.keyBy(Event::sourceAddress)
      .enableAsyncState()                  // KeyedStream.java:1099：开启异步处理
      .flatMap(new StateMachineMapper());  // 内部使用 v2 的 ValueState

currentState.asyncValue().thenAccept(state -> {
    ...
    currentState.asyncUpdate(nextState);
});
```

开启后，算子会被替换成异步版本：`KeyedStream.process()` 在 `isEnableAsyncState()` 时 new 一个 `AsyncKeyedProcessOperator`，而不是 `KeyedProcessOperator`（`RT/streaming/api/datastream/KeyedStream.java:360-366`）。

### 7.3 ★ AsyncExecutionController（AEC）：如何在"异步"下保持语义正确？

核心难点：**同一个 key 的记录必须按顺序处理**。否则两条记录并发执行"读计数 → +1 → 写回"，就会丢失更新。

`RT/runtime/asyncprocessing/AsyncExecutionController.java`（算子在 `AbstractAsyncStateStreamOperator.java:86` 中创建它的子类 `StateExecutionController`）：

**1. 每条记录一个上下文**：`setAsyncKeyedContextElement()`（`RT/runtime/asyncprocessing/operators/AbstractAsyncKeyOrderedStreamOperator.java:135`）→ `buildContext(record, key)`（AEC `:214`）创建 `RecordContext`，它替代了 3.2 节同步模式中的 "当前 key"。回调执行时会先恢复对应记录的上下文，所以回调里访问状态，访问的仍然是那条记录的 key。

**2. 按 key 占位**：`handleRequest()`（`:304`）

```java
seizeCapacity(allowOverdraft);           // Step 1: 在途记录数超限就先等（:384，maxInFlightRecordNum :88）
if (tryOccupyKey(currentContext)) {      // Step 3: 这个 key 当前没有别的记录在处理？（:293）
    insertActiveBuffer(request);         //   → 进入活跃队列，可以执行（:342）
} else {
    insertBlockingBuffer(request);       //   → 同 key 有记录在处理，先排队（:354）
}
triggerIfNeeded(false);                  // Step 4: 攒够一批（batchSize，:67）就发给执行器
```

`KeyAccountingUnit.occupy()`/`release()`（`KeyAccountingUnit.java:47`/`:52`）记录"哪些 key 正在被处理"。**不同 key 的记录可以并发访问状态，同一个 key 的记录严格串行。**

**3. 批量执行**：`triggerIfNeeded()` 把一批请求交给 `asyncExecutor.executeBatchRequests()`（`:373`）。ForSt 的实现是 `ForStStateExecutor.executeBatchRequests()`（`FST/ForStStateExecutor.java:149`），它把读请求分发给**读 IO 线程池**（`:62`），写请求交给**协调/写线程**（`:59`、`:65`）。批量处理能把多次 DFS 访问合并起来，摊薄延迟。

**4. 回调回到 Task 线程**：`CallbackRunnerWrapper`（AEC `:166`）把 `thenAccept` 中的回调**投递回 mailbox**。所以回调和 `processElement` 仍然在同一个 Task 线程上执行，**用户代码依然不需要加锁**（第 9.4 节的线程名可以印证）。

**5. 与 Checkpoint 的配合**：做快照之前必须把在途的请求都处理完（`drainInflightRecords()`，`:429`），否则快照里会缺少那些"读了但还没写回"的更新。

相关配置（`flink-core/.../ExecutionOptions.java`）：`execution.async-state.total-buffer-size`（`:198`，在途记录上限）、`execution.async-state.active-buffer-size`（`:218`，批大小）、`execution.async-state.active-buffer-timeout`（`:236`）。

---

## 8. 选型总结

| | HashMap | RocksDB | ForSt |
|---|---|---|---|
| 状态放在哪 | JVM 堆 | 本地磁盘 + 托管内存 | DFS（主）+ 本地缓存 |
| 访问开销 | 最低（对象引用） | 每次都要序列化 | 序列化 + 可能的远程 IO |
| 状态规模 | 受限于堆内存 | 受限于本地盘 | 受限于 DFS |
| 快照方式 | 全量，COW 实现异步 | 支持增量（硬链接 + 上传新 SST） | 支持增量，数据已在 DFS 上，几乎不用上传 |
| 恢复和扩缩容 | 读取全量 | 下载 SST + 裁剪 | 基本不需要下载 |
| API | V1 同步 | V1 同步 | 建议 V2 异步 |
| 适合场景 | 小状态、低延迟 | 大状态的主流选择 | 云原生、超大状态、需要快速扩缩容 |

---

## 9. 实验

示例：`StateBackendDemo.java`。1000 个 key，每个 key 一个计数加一个 1KB payload，状态总量约 1MB；5000 条/秒，每 5 秒做一次 Checkpoint，保留最近 3 个。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.state.StateBackendDemo rocksdb 2 32
```

参数：`<hashmap|rocksdb|forst> <并行度> <运行秒数> [恢复路径]`。每个 subtask 每 10 秒打印一次：负责的 KeyGroup 区间、见过的 key 数、其中从状态里恢复出来的 key 数，以及落在区间之外的 key 数（应该永远是 0）。

### 9.1 三种状态后端的 Checkpoint 大小（实测）

| 后端 | 稳定后的 Checkpoint 大小 | 说明 |
|---|---|---|
| hashmap | 1,058,762 bytes | 全量快照，1000 × (1KB payload + 计数)，默认不压缩（`execution.checkpointing.snapshot-compression` 默认为 false，`ExecutionOptions.java:90`） |
| rocksdb | 141KB ~ 181KB | SST 文件默认用 Snappy 压缩（`state.backend.rocksdb.compression.per.level`，`RDB/RocksDBConfigurableOptions.java:171`），而 payload 全是字符 `x`，压缩率极高 |
| forst | 148KB ~ 189KB | 与 RocksDB 相近 |

两个后端里，1000 个 key 的分布都是 subtask0（KeyGroup [0,63]）510 个，subtask1（[64,127]）490 个。

### 9.2 增量 Checkpoint：`shared/` 目录里的文件复用

RocksDB 跑 32 秒，完成了 7 次 Checkpoint，保留 chk-5、chk-6、chk-7。查看 `shared/` 里每个文件被哪些 Checkpoint 引用（在 `_metadata` 里搜文件名）：

| shared 文件 | 大小 | 创建时间 | 被引用 |
|---|---|---|---|
| `4ca541cf…` | 36,671 | 18:53:00（第 1 次 Checkpoint） | chk-5、chk-6、chk-7 |
| `e57b267b…` | 35,220 | 18:53:00（第 1 次 Checkpoint） | chk-5、chk-6、chk-7 |
| `1f20d0e2…`、`358efab6…` | 25,845 | 18:53:20 | chk-5 |
| `54d84508…`、`b11d736e…` | 25,845 | 18:53:25 | chk-6 |
| `0977a608…`、`9c68a708…` | 25,845 | 18:53:30 | chk-7 |

解读：
1. **第 1 次 Checkpoint 上传的两个 SST**（两个 subtask 各一个，包含全部 payload）被后面**每一次** Checkpoint 复用，只上传了一次
2. ~~每次 Checkpoint 只**新上传**两个约 25KB 的 SST（两个 subtask 各一个），里面是这 5 秒内计数的更新~~
   **勘误（2026-10-02）**：这些 25845 字节的文件是 RocksDB 的 **OPTIONS 配置文件**（恢复日志中的本地文件名为 `OPTIONS-000015`），每次 Checkpoint 都会重新上传。计数的更新在新的小 SST 里（约 5~7KB），因为小于 `data-inline-threshold`（20KB）而内联在 `_metadata` 中，所以不会出现在 `shared/` 目录。详见 `content-plan/14-第3讲下-公众号定稿.md` 第二节。
3. chk-2 ~ chk-4 用过的文件已经**被删除**了：它们的 `lastUsedCheckpointID` 小于保留中最小的 chk-5（5.4 节）
4. `_metadata` 大小依次是 31KB → 45KB → 59KB，越来越大，因为小于 20KB 的 SST 和 MANIFEST 都内联在里面（5.5 节）

### 9.3 扩缩容：2 → 3 个并行度

从上面的 chk-7 恢复，并行度改为 3：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && KEEP_CKPT=1 ./run.sh study.state.StateBackendDemo rocksdb 3 22 /tmp/flink-state-demo/<jobId>/chk-7
```

（`KEEP_CKPT=1` 让 `run.sh` 不要清空 Checkpoint 目录。）

实测输出：

```
Restoring job ... from Savepoint 7 @ 0 ... located at file:/tmp/flink-state-demo/.../chk-7.
[count-per-key (1/3)] keyGroups=[0, 42]   keys=359 restoredKeys=359 keysOutOfRange=0
[count-per-key (2/3)] keyGroups=[43, 85]  keys=311 restoredKeys=311 keysOutOfRange=0
[count-per-key (3/3)] keyGroups=[86, 127] keys=330 restoredKeys=330 keysOutOfRange=0
Completed checkpoint 8 ...      ← Checkpoint ID 接着恢复点继续递增
```

**359 + 311 + 330 = 1000**：每个 key 都在它的新 subtask 上找到了原来的状态，没有丢失，也没有重复。

打开 `RocksDBIncrementalCheckpointUtils` 和 `RocksDBIncrementalRestoreOperation` 的 INFO 日志（`demos/src/main/resources/log4j2.properties` 中已配置），可以看到三个 subtask 走了不同的路径：

| 新 subtask | 目标区间 | 来源（旧区间） | 恢复方式 | range delete 的边界 |
|---|---|---|---|---|
| 1/3 | [0, 42] | 只有旧 sub0 [0,63] | 打开基础 DB 后裁剪 | `[[43], [64]]`：删掉 [43, 64) |
| 3/3 | [86, 127] | 只有旧 sub1 [64,127] | 打开基础 DB 后裁剪 | `[[64], [86]]`：删掉 [64, 86) |
| 2/3 | [43, 85] | 旧 sub0 **和** 旧 sub1 | "restore ... from multiple state handles"：打开一个作为基础 DB 并裁剪，再把另一个的数据合并进来 | `[[86], [-128]]`：删掉 [86, 128) |

边界是 KeyGroup 前缀的**字节值**。`-128` 就是 128 用有符号 byte 表示的结果（maxParallelism=128 时前缀只有 1 字节），代表"一直删到末尾"。

### 9.4 ForSt + 异步状态的线程模型

`forst` 模式会自动调用 `enableAsyncState()`，并改用 V2 API。运行中抓一次线程栈（`jstack`），与状态相关的线程有：

```
2 ForSt-StateExecutor-Coordinator-And-Write-thread   ← 每个 subtask 1 个：协调批次、执行写
6 ForSt-StateExecutor-read-IO-thread                  ← 每个 subtask 3 个：并发读
8 Flink-ForStStateDataTransfer-thread                 ← Checkpoint/恢复时的文件传输
1 asyncRequestBuffer-timeout-scheduler-thread         ← 批次攒不满时按超时触发
```

同时，回调中打印的线程名是 `count-per-key -> Sink: Writer (1/2)#0`，也就是 **Task 线程本身**。这印证了 7.3 节第 4 点：IO 在后台线程池执行，回调回到 mailbox 执行。

> 🐛 **踩坑记录**：第一次运行 forst 模式时，作业已经取消、`main` 也执行完了，JVM 却一直不退出。用 `jstack` 查看，发现有两个**没有 Java 栈、状态为 RUNNABLE 的非 daemon 线程**：它们是 ForSt 通过 JNI 挂到 JVM 上的原生线程，所以 JVM 无法正常退出。只有在本地进程里跑 MiniCluster 才会遇到这个问题（真实集群的 TM 退出时会调用 `System.exit`）。示例在 `main` 末尾加了 `System.exit(0)` 来规避。

---

## 10. 断点清单

用 IDEA 运行 `StateBackendDemo`，参数 `rocksdb 2 60`（第 7 组参数用 `forst 2 60`）。

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `StateBackendLoader.java:127` | 反射加载 RocksDB 工厂 |
| 2 | `AbstractKeyedStateBackend.java:378` `getOrCreateKeyedState` | 状态对象创建，TTL 包装 |
| 3 | `AbstractKeyedStateBackend.java:262` | 每条记录算一次 KeyGroup |
| 4 | `SerializedCompositeKeyBuilder.java:184` `serializeKeyGroupAndKey` | RocksDB key 的字节布局 |
| 5 | `RocksDBValueState.java:82` | 真正的 `db.get` |
| 6 | `RocksDBSnapshotStrategyBase.java:174` | 原生 Checkpoint（硬链接） |
| 7 | `RocksIncrementalSnapshotStrategy.java:422` | 哪些 SST 被复用，哪些要上传 |
| 8 | `SharedStateRegistryImpl.java:159` `unregisterUnusedState` | 哪些共享文件被删除 |
| 9 | `StateAssignmentOperation.java:679`（带恢复路径运行） | 状态句柄如何按区间求交集 |
| 10 | `RocksDBIncrementalCheckpointUtils.java:134`（带恢复路径、改变并行度） | 裁剪 KeyGroup |
| 11 | `AsyncExecutionController.java:304`（forst） | 同 key 请求如何排队 |
| 12 | `CopyOnWriteStateMap.java:268`（hashmap，在 Checkpoint 期间命中） | COW 复制 |

## 11. 课后练习

1. **Heap 的原地修改陷阱**：把 `SyncCountPerKey` 中的计数改成 `ValueState<long[]>`，用 `value()[0]++` 原地修改、**不调用** `update()`。分别用 hashmap 和 rocksdb 运行，观察计数是否增长，并用 4.2 节和 5.1 节解释
2. **maxParallelism 不兼容**：用 `pipeline.max-parallelism=128` 生成一个 Checkpoint，再改成 256 并从这个 Checkpoint 恢复，看看报什么错，在源码里找到抛出异常的位置
3. **缩容**：从 2 个并行度的 Checkpoint 恢复成 1 个并行度，查看日志。为什么源码注释说 "all scale ins" 都会走 `restoreFromMultipleStateHandles`？
4. **TTL**：给 `count` 加上 `StateTtlConfig`，读 `TtlValueState` 和 RocksDB 的 compaction filter（`RDB/ttl/` 目录），回答：过期数据是读的时候删，还是 compaction 的时候删？
5. **思考题**：为什么 RocksDB 增量 Checkpoint 必须等 `notifyCheckpointComplete` 之后，才能把这次上传的 SST 当作复用的基准？如果不等，会出现什么问题？（提示：第 02 篇 7.3 节说过，这个通知不保证送达）

## 12. 下一篇预告

**04：网络栈与背压**。从 `RecordWriter` 到 Netty 再到 `InputChannel`，Buffer 是怎么申请和回收的；Credit-based 流控如何把背压一层层传回上游；Buffer Debloating 调整的是什么；以及第 02 篇里"对齐时阻塞上游"在网络层到底是怎么实现的。
:::
