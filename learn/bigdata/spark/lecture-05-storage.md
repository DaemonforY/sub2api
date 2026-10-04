---
title: "第 5 讲：存储体系"
description: "BlockManager 存储体系：存储级别、cache 的实现、广播变量的分块传输。"
bigdata: "spark"
---

# 第 5 讲：存储体系

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 配置默认值、类名、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；关键结论经过实验验证（仓库 `experiments/06-storage/`）。

## 0. 本讲要回答的问题

1. Spark 里的"块"（Block）都有哪些？谁在管理它们？
2. 调用 `rdd.cache()` 之后，数据存在哪、怎么被读出来？
3. 不同存储级别（`MEMORY_ONLY`、`MEMORY_ONLY_SER`……）差别有多大？
4. RDD 和 Dataset 的 `cache()` 一样吗？
5. 广播变量是怎么分发到成百上千个 Executor 上的？
6. 缓存、广播、Shuffle 文件，什么时候被清理？

---

## 1. 一切皆块（Block）

在 Spark 里，**缓存的 RDD 分区、Shuffle 文件、广播变量、大的 Task 结果**，统统都是"块"，由同一套存储体系管理。每种块都有自己的 `BlockId`（`BlockId.scala`）：

| BlockId | 命名格式 | 是什么 |
|---|---|---|
| `RDDBlockId` | `rdd_{rddId}_{分区号}` | 缓存的 RDD 分区 |
| `ShuffleDataBlockId` / `ShuffleIndexBlockId` | `shuffle_{shuffleId}_{mapId}_0.data` / `.index` | 第 3 讲的 Shuffle 文件 |
| `BroadcastBlockId` | `broadcast_{id}` / `broadcast_{id}_piece{i}` | 广播变量本体 / 广播变量的分块 |
| `TaskResultBlockId` | `taskresult_{taskId}` | 太大、不能直接随 RPC 返回的 Task 结果 |
| `StreamBlockId` | `input-{streamId}-{uniqueId}` | 流式接收的数据 |

> 4.x 里还出现了 `PythonWorkerLogBlockId` 等日志相关的块类型，说明连 Python Worker 的日志也被纳入了块存储，这是比较新的变化。

---

## 2. 架构：Master / Slave 结构

```
Driver                                       每个 Executor
┌───────────────────────────────┐            ┌──────────────────────────────────┐
│ BlockManagerMaster            │◀──RPC─────▶│ BlockManager                      │
│   BlockManagerMasterEndpoint  │  注册/汇报  │   ├── BlockInfoManager  块的元信息和读写锁│
│   记录：哪个块在哪些 Executor  │  块的增删   │   ├── MemoryStore       内存存储（第 4 讲）│
└───────────────────────────────┘            │   ├── DiskStore         磁盘存储          │
                                             │   │     └── DiskBlockManager 目录管理    │
                                             │   └── BlockTransferService 块的远程传输  │
                                             └──────────────────────────────────┘
```

- **每个 Executor（以及 Driver）都有一个 `BlockManager`**，负责本地块的存取
- **Driver 上的 `BlockManagerMaster`** 掌握全局视图：每个块存在哪些 Executor 上。Executor 存了新块、删了块，都会向它汇报（`tellMaster`）
- 读一个本地没有的块时，先问 Master "它在哪"，再去对应的 Executor 拉取

### 2.1 块的读写锁：BlockInfoManager

同一个块可能被多个 Task 同时读，也可能正在被写入或被淘汰。`BlockInfoManager` 为每个块维护**读写锁**：

- 可以有多个读者（`readerCount`），或者一个写者（`writerTask`）
- 第 4 讲里 `MemoryStore` 淘汰块时，会对候选块尝试获取**写锁**；**正在被读的块拿不到写锁，就不会被淘汰**

### 2.2 磁盘目录结构

`DiskBlockManager.getFile`（`DiskBlockManager.scala:95`）按块名的哈希值决定放在哪个目录：

```scala
val hash = Utils.nonNegativeHash(filename)
val dirId = hash % localDirs.length                              // 选哪个本地目录（spark.local.dir 可配多个）
val subDirId = (hash / localDirs.length) % subDirsPerLocalDir    // 选哪个子目录（默认 64 个）
```

最终路径（实验 E 的实测结果）：

```
<spark.local.dir>/blockmgr-<UUID>/<两位十六进制子目录>/<块名>
例：.../blockmgr-a3f0b505-dd2f-460f-8289-a5785450d62a/02/rdd_11_0
```

- 子目录数由 `spark.diskStore.subDirectories` 控制，默认 **64**；目录名用 `"%02x"` 格式化，所以是 `00` ~ `3f`
- **为什么要分子目录**：一个目录下文件太多时，文件系统的查找和创建都会变慢。用哈希把文件分散到 64 个子目录里
- **生产建议**：本地目录配置多块磁盘（逗号分隔），Shuffle 和缓存的磁盘 I/O 会被分散到多块盘上

> ⚠️ **本地目录从哪来**（`Utils.getConfiguredLocalDirs`）：**运行在 YARN 容器里时，用的是 YARN 分配的本地目录，`spark.local.dir` 会被忽略**（要在 YARN 的 `yarn.nodemanager.local-dirs` 里配置）；其他情况下依次取环境变量 `SPARK_EXECUTOR_DIRS`、`SPARK_LOCAL_DIRS`、配置 `spark.local.dir`，都没有时用 JVM 的 `java.io.tmpdir`。

---

## 3. 一个缓存块的一生：从 cache() 到读取

### 3.1 读取路径

第 1 讲的调试练习里，Task 最终会调用 `RDD.iterator`。如果这个 RDD 被缓存过，会走到 `getOrCompute`（`RDD.scala:381`）：

```
RDD.iterator(分区)
 └─ storageLevel != NONE？
     └─ getOrCompute
         └─ BlockManager.getOrElseUpdateRDDBlock(rdd_{id}_{分区})      BlockManager.scala:1409
             ├─ get(blockId)                                          BlockManager.scala:1347
             │   ├─ getLocalValues   本地有？→ 直接返回（缓存命中）
             │   └─ getRemoteValues  问 Master 在哪，去别的 Executor 拉 → 返回
             └─ 都没有 → 调用 computeOrReadCheckpoint 计算，
                         边计算边写入 MemoryStore / DiskStore（doPutIterator），再返回
```

**缓存是惰性的**：`rdd.cache()` 本身什么都不做，只是给 RDD 打上存储级别的标记。第一次 Action 计算到这个分区时，才真正写入缓存。

### 3.2 实验 C：缓存命中

```
第 1 次 count：map 函数执行了 1000 次（计算并写入缓存）
第 2 次 count：map 函数执行了 0 次（直接读缓存块 rdd_8_0 ~ rdd_8_3）
```

### 3.3 新机制：缓存可见性（3.5+）

`getOrElseUpdate` 里有一个参数 `isCacheVisible`。对应的配置是 `spark.rdd.cache.visibilityTracking.enabled`（**3.5.0 引入，默认 `false`**）。

**它解决的问题**：一个 Task 计算并缓存了某个分区，但这个 Task 最后失败了（或者是被杀掉的推测执行副本）。如果其他 Task 直接复用这个缓存块，**累加器（Accumulator）的值就会不一致**：计算时累加了，Task 却失败了。

开启后，缓存块只有在**生成它的某个 Task 成功完成后**才"可见"，才能被复用。

---

## 4. 存储级别

### 4.1 五个维度

`StorageLevel(useDisk, useMemory, useOffHeap, deserialized, replication)`，4.2.0 预定义了这些级别（`common/utils/.../StorageLevel.scala:149-161`）：

| 级别 | 磁盘 | 内存 | 堆外 | 反序列化存储 | 副本 |
|---|---|---|---|---|---|
| `NONE` | | | | | 1 |
| `DISK_ONLY` / `_2` / `_3` | ✅ | | | | 1 / 2 / 3 |
| `MEMORY_ONLY` / `_2` | | ✅ | | ✅ | 1 / 2 |
| `MEMORY_ONLY_SER` / `_2` | | ✅ | | | 1 / 2 |
| `MEMORY_AND_DISK` / `_2` | ✅ | ✅ | | ✅ | 1 / 2 |
| `MEMORY_AND_DISK_SER` / `_2` | ✅ | ✅ | | | 1 / 2 |
| `OFF_HEAP` | ✅ | ✅ | ✅ | | 1 |

- **反序列化（deserialized）**：存 Java 对象本身，读的时候不用反序列化，最快；但 Java 对象有对象头、指针等开销，**最占内存**
- **序列化（`_SER`）**：存字节数组，省内存，但读的时候要花 CPU 反序列化
- **`_2`、`_3`**：在其他 Executor 上存副本，某个 Executor 挂了缓存还在；代价是成倍的存储和网络开销

### 4.2 实验 A：差距有多大

同一份数据（100 万个 `User(id, name, city)` 对象，4 个分区）：

| 存储级别 | 内存 | 磁盘 |
|---|---|---|
| `MEMORY_ONLY` | **87.7 MB** | — |
| `MEMORY_ONLY_SER` | **39.0 MB** | — |
| `DISK_ONLY` | — | 39.0 MB |

**反序列化存储占用的内存是序列化存储的 2.25 倍。** 这里用的还是默认的 Java 序列化；换成 Kryo 通常会更小。

> 缓存时选哪个序列化器，规则和第 3 讲一样：`SerializerManager.getSerializer(ct, autoPick)` 遇到基本类型或 String 会自动用 Kryo，自定义类型用 `spark.serializer`（默认 Java 序列化）。

### 4.3 实验 B：RDD 和 Dataset 的 cache() 默认级别不同

```
RDD.cache()     -> MEMORY_ONLY       （RDD.scala:200，persist() = persist(MEMORY_ONLY)）
Dataset.cache() -> MEMORY_AND_DISK   （由 spark.sql.defaultCacheStorageLevel 决定）
设置 spark.sql.defaultCacheStorageLevel=MEMORY_ONLY 后 Dataset.cache() -> MEMORY_ONLY
```

- **RDD**：写死的 `MEMORY_ONLY`。内存放不下的分区**不缓存**，下次用到时**重算**
- **Dataset / DataFrame**：默认 `MEMORY_AND_DISK`，放不下的写到磁盘
- 🆕 `spark.sql.defaultCacheStorageLevel` 是 **4.0.0 新增**的配置，同时影响 `dataset.cache()` 和 `catalog.cacheTable()` 等

> 另外，Dataset 的缓存不是直接缓存行对象，而是转换成**列式格式**（`InMemoryRelation`）存储，通常比 RDD 缓存更省内存。这部分在 Spark SQL 的几讲里再细说。

### 4.4 怎么选

| 场景 | 建议 |
|---|---|
| 数据不大，内存充足，追求速度 | `MEMORY_ONLY`（RDD 默认） |
| 内存紧张 | `MEMORY_ONLY_SER`，最好配合 Kryo |
| 重算代价很高（比如前面有复杂的 Join） | `MEMORY_AND_DISK`，放不下的落盘，避免重算 |
| 对可用性要求高，Executor 经常被抢占 | `_2` 副本 |
| **只用一次的数据** | **别缓存**。缓存本身有写入开销，还挤占 Execution 内存（第 4 讲） |

---

## 5. 广播变量：TorrentBroadcast

### 5.1 为什么需要广播

一个 Task 用到了一个 100MB 的字典，一个 Stage 有 1000 个 Task：

- **不广播**：这个字典会被序列化进每个 Task 的闭包，Driver 要发 1000 次，共 100GB
- **广播**：每个 Executor 只拉取一次，同一个 Executor 上的所有 Task 共享

> 第 2 讲里，DAGScheduler 也是用广播把 Task 的二进制（RDD + 函数）发出去的。

### 5.2 演进：从 HTTP 到 Torrent

| 时间 | 变化 | 提交 |
|---|---|---|
| 早期 | `HttpBroadcast`：所有 Executor 都从 Driver 上的 HTTP 服务器下载 | — |
| 2015-12，**Spark 2.0** | **HttpBroadcast 被移除**，只剩 `TorrentBroadcast` | `[SPARK-12588] Remove HttpBroadcast in Spark 2.0.` |

HttpBroadcast 的问题：Driver 是唯一的数据源，1000 个 Executor 同时下载，Driver 的网卡就成了瓶颈。

### 5.3 TorrentBroadcast 怎么工作

**Driver 端：写**（`writeBlocks`，`TorrentBroadcast.scala:139`）

1. 把完整的值存一份到本地 BlockManager（`broadcast_{id}`），供 Driver 自己使用
2. 把值序列化、压缩（`spark.broadcast.compress`，默认 `true`），**切成多块**，每块大小 `spark.broadcast.blockSize`（默认 **4m**）
3. 每一块存为 `broadcast_{id}_piece{i}`，并**告诉 Master**（`tellMaster = true`，第 172 行）

**Executor 端：读**（`readBlocks`，`TorrentBroadcast.scala:189`）

1. 把块的编号**随机打乱**（`Random.shuffle`，第 195 行）
2. 逐块获取：**先查本地，没有再远程拉取**
   - 远程拉取时，可能从 Driver 拉，也可能从**已经拉到这一块的其他 Executor** 拉
3. 每拉到一块，**自己也存一份，并告诉 Master**（第 218 行）：从此**自己也成为这一块的数据源**
4. 所有块都到齐后，拼接、解压、反序列化，得到完整的值，缓存在本地

```
                  Driver（拥有全部块）
                 ╱       │       ╲
        piece0 ╱  piece2 │        ╲ piece1
             ╱           │         ╲
      Executor A ◀─piece1─ Executor B ─piece0─▶ Executor C
     （拉完后也成为      （拉完后也成为        （从 A、B 那里
       数据源）            数据源）             拉取剩下的块）
```

**和 BitTorrent 一样**：下载的人越多，数据源也越多，Driver 的压力被分摊了。**随机顺序**是为了让不同 Executor 先拉不同的块，尽快形成多个数据源，而不是大家都挤着从 Driver 拉第 0 块。

### 5.4 实验 D：分块

```
广播 10MB 随机字节，blockSize=4m：被切成 3 块（broadcast_5_piece0 ...）
Executor 端读到的长度：10485760, 10485760
```

10MB ÷ 4MB 向上取整 = 3 块。（用随机字节是为了不让压缩改变大小。）

### 5.5 一个细节：Driver 端用软引用持有广播值

`TorrentBroadcast` 里的 `_value` 是一个 `Reference`（第 72 行）。源码注释说明：Driver 端用 **`SoftReference`** 持有广播值，内存紧张时可以被 GC 回收，需要时再从 BlockManager 重新读出来；Spark 内部的某些广播（`serializedOnly = true`）则用 **`WeakReference`**，回收得更积极。

> 推论：**不要在 Driver 端修改广播变量的值**并指望 Executor 看到。广播变量是只读的，Executor 拿到的是创建时那一刻的序列化快照。

---

## 6. 清理：什么时候删除这些块

| 方式 | 说明 |
|---|---|
| **手动** | `rdd.unpersist()`、`broadcast.unpersist()` / `destroy()` |
| **自动：ContextCleaner** | Driver 上的 `ContextCleaner` 用 `WeakReference` 跟踪 RDD、Shuffle、广播变量等对象。当用户代码里**不再有任何引用**指向它们、并且被 GC 回收后，ContextCleaner 会通知各个 Executor 删除对应的块。由 `spark.cleaner.referenceTracking` 控制，默认 `true` |
| **淘汰** | 内存不够时，`MemoryStore` 按 LRU 淘汰缓存块（第 4 讲） |
| **应用结束** | 删除 `blockmgr-*` 目录（外部 Shuffle 服务管理的 Shuffle 文件除外） |

> ⚠️ ContextCleaner 依赖 **Driver 端的 GC** 才能发现"对象已经不用了"。如果 Driver 堆很大、很少发生 GC，已经不用的 Shuffle 文件和缓存就会迟迟得不到清理。
>
> 为此 ContextCleaner 会**定期主动触发一次 `System.gc()`**（`ContextCleaner.scala:132`），间隔由 `spark.cleaner.periodicGC.interval` 控制，**默认 30 分钟**。所以在常驻应用里，最坏情况下清理会滞后约 30 分钟；如果磁盘空间增长很快，可以适当调小这个间隔，或者在代码里主动 `unpersist()`。

---

## 7. 常用配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.local.dir` | 未设置时用 `java.io.tmpdir` | 本地存储目录，可配多个（逗号分隔）；**YARN 上会被忽略** |
| `spark.diskStore.subDirectories` | `64` | 每个本地目录下的子目录数 |
| `spark.broadcast.blockSize` | `4m` | 广播变量分块大小 |
| `spark.broadcast.compress` | `true` | 压缩广播变量 |
| `spark.broadcast.checksum` | `true` | 广播块校验和 |
| `spark.sql.defaultCacheStorageLevel` | `MEMORY_AND_DISK` | Dataset 默认缓存级别（4.0.0+） |
| `spark.rdd.cache.visibilityTracking.enabled` | `false` | 缓存可见性追踪（3.5.0+） |
| `spark.storage.replication.proactive` | `true` | 副本丢失时主动补副本 |
| `spark.cleaner.referenceTracking` | `true` | 启用 ContextCleaner 自动清理 |
| `spark.cleaner.periodicGC.interval` | `30min` | ContextCleaner 定期触发 GC 的间隔 |
| `spark.storage.decommission.enabled` | `false` | 节点下线时迁移块（`BlockManagerDecommissioner`） |

---

## 自测题

1. 列举 3 种不同的块，并说出它们的命名格式。
2. 调用 `rdd.cache()` 之后马上调用 `rdd.unpersist()`，中间没有任何 Action，会发生什么？
3. 一个 Task 要读缓存块 `rdd_5_3`，本地没有，它是怎么找到并拿到这个块的？
4. 同一份数据用 `MEMORY_ONLY` 和 `MEMORY_ONLY_SER` 缓存，大小可能差几倍？各自的代价是什么？
5. RDD 和 Dataset 的 `cache()` 默认存储级别分别是什么？内存放不下时各自会怎样？
6. TorrentBroadcast 为什么要把块的读取顺序随机打乱？
7. 为什么 Executor 拉到一个广播块后，要"告诉 Master"？
8. 一个常驻的 Spark 应用磁盘占用一直在涨，可能和存储体系的哪个机制有关？可以怎么缓解？
9. 在 YARN 上设置了 `spark.local.dir`，为什么不生效？

## 下一讲预告

第 6 讲：**RPC 与部署**。Driver 和 Executor 之间的那些消息是怎么传递的？`spark-submit` 之后发生了什么？YARN 和 K8s 上的 Executor 是怎么启动的？动态资源分配是怎么工作的？
:::
