---
title: "第 3 讲：Shuffle 原理"
description: "Shuffle 的写和读：三种 ShuffleWriter 的选择条件、溢写与合并、Shuffle 读取流程。"
bigdata: "spark"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-03-shuffle-cover.webp"}]]
---

# 第 3 讲：Shuffle 原理

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-03-shuffle-cover.webp" alt="Yui 和 Kai 在仓库中演示 Spark Shuffle 全流程" width="1200" height="800" loading="eager" /><figcaption>Yui 和 Kai 在仓库中演示 Spark Shuffle 全流程<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 文中配置默认值、类名、行号均在 4.2.0 源码中核实；版本演进时间来自 Git 提交历史；三种 Writer 的选择结论经过 10 个用例的实验验证。

## 0. 本讲要回答的问题

1. Shuffle 到底在干什么？为什么说它是 Spark 最昂贵的操作？
2. Hash Shuffle 为什么被淘汰？
3. 一个 Map Task 写出的 Shuffle 文件长什么样？
4. `SortShuffleManager` 里的三种 Writer，分别在什么条件下被选用？
5. Reduce 端是怎么把数据拉过来的？

---

## 1. Shuffle 是什么，为什么贵

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-03-shuffle-1.webp" alt="按 key 打散并分发数据，展示 Shuffle 的昂贵开销" width="960" height="640" loading="lazy" /><figcaption>按 key 打散并分发数据，展示 Shuffle 的昂贵开销<span>AI 生成配图</span></figcaption></figure>

第 1 讲说过：宽依赖意味着父分区的数据要**按 key 打散**，分发给多个子分区。这个"打散 + 分发"的过程就是 **Shuffle**。

```
       Map 端（上游 Stage 的 ShuffleMapTask）            Reduce 端（下游 Stage 的 Task）
  ┌────────────┐
  │ Map Task 0 │──写──▶ 按分区号分好组的本地文件 ─┐
  └────────────┘                                  ├──拉取分区 0 的数据──▶ Reduce Task 0
  ┌────────────┐                                  │
  │ Map Task 1 │──写──▶ 按分区号分好组的本地文件 ─┼──拉取分区 1 的数据──▶ Reduce Task 1
  └────────────┘                                  │
  ┌────────────┐                                  │
  │ Map Task 2 │──写──▶ 按分区号分好组的本地文件 ─┘
  └────────────┘
```

Shuffle 贵在它把几乎所有昂贵的操作都占全了：

| 开销 | 原因 |
|---|---|
| **序列化 / 反序列化** | 内存里的对象要变成字节才能写盘、传网络 |
| **磁盘 I/O** | Map 端的输出要**落盘**；内存不够时还要溢写（spill）临时文件 |
| **网络 I/O** | Reduce 端要从**所有** Map 端拉取属于自己的数据 |
| **排序 / 聚合** | 为了按分区组织数据，要排序；`reduceByKey` 还要做聚合 |
| **同步屏障** | 下游必须等上游**全部** Task 完成才能开始（Stage 边界） |

> 这也是为什么第 1 讲强调"能用窄依赖就别用宽依赖"，以及"分区器相同时 reduceByKey 不会 Shuffle"。

---

## 2. 演进史：从 Hash 到 Sort

以下时间节点均来自 Spark 的 Git 提交历史：

| 时间 | 变化 | 提交 |
|---|---|---|
| 早期 | **Hash Shuffle** 是默认实现 | — |
| 2014-09，**Spark 1.2** | **Sort Shuffle** 成为默认实现 | `[SPARK-3280] Made sort-based shuffle the default implementation` |
| 2015-05，**Spark 1.4** | 加入基于二进制数据排序的 **tungsten-sort** 路径 | `[SPARK-7081] Faster sort-based shuffle path using binary processing cache-aware sort` |
| 2016-04，**Spark 2.0** | **HashShuffleManager 被移除** | `[SPARK-14667] Remove HashShuffleManager` |
| 现在（4.2.0） | 只有 `SortShuffleManager`；配置值 `sort` 和 `tungsten-sort` 都指向它（`ShuffleManager.scala:113-114`） | — |

### Hash Shuffle 为什么被淘汰

Hash Shuffle 的做法很直接：**每个 Map Task 为每个 Reduce 分区各写一个文件**。

```
文件数 = Map Task 数 × Reduce 分区数
```

1000 个 Map Task、1000 个 Reduce 分区，就是 **100 万个小文件**。这带来了：

- 文件系统元数据压力巨大，大量随机小 I/O
- 每个 Map Task 要同时打开 R 个文件，每个文件都要一块写缓冲区，**内存占用随 R 线性增长**

> 后来的"文件合并"（consolidation）优化让同一个 Executor 上的 Map Task 共用一组文件，缓解了问题，但没有根治。

### Sort Shuffle 的思路

**每个 Map Task 只写一个数据文件**：先按分区号排好序，再一次性写出；同时写一个**索引文件**，记录每个分区在数据文件里的起止位置。

```
文件数 = Map Task 数 × 2（数据文件 + 索引文件）
```

> 网上不少 Spark 原理文章还在详细讲 Hash Shuffle 的两种模式。了解历史有好处，但要清楚：**它在 Spark 2.0 就已经被移除了**。

---

## 3. Map 端输出：数据文件 + 索引文件

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-03-shuffle-2.webp" alt="数据文件配索引文件，Reduce 按偏移读取对应片段" width="960" height="640" loading="lazy" /><figcaption>数据文件配索引文件，Reduce 按偏移读取对应片段<span>AI 生成配图</span></figcaption></figure>

文件命名在 `BlockId.scala` 中定义：

```
shuffle_{shuffleId}_{mapId}_0.data     数据文件：所有分区的数据，按分区号顺序排列
shuffle_{shuffleId}_{mapId}_0.index    索引文件：每个分区在数据文件中的偏移量
```

> 最后的 `0` 是 `NOOP_REDUCE_ID`（`IndexShuffleBlockResolver.scala:676`）。因为所有分区写在一个文件里，这个位置不再代表具体的 reduce 分区号。

```
.data 文件：  [ 分区0的数据 | 分区1的数据 | 分区2的数据 | ... ]
.index 文件： [ 0, 偏移1, 偏移2, 偏移3, ... , 文件总长 ]
                 └─ 分区 i 的数据 = data[offset(i), offset(i+1))
```

Reduce Task 想要分区 i 的数据时，先读索引文件找到偏移，再从数据文件里读出对应的一段即可。

---

## 4. 三种 Writer：在什么条件下被选用

### 4.1 选择流程

在 `ShuffleDependency` 创建时，`SortShuffleManager.registerShuffle`（`SortShuffleManager.scala:92`）就决定了用哪种 Writer：

```
                         registerShuffle(dependency)
                                    │
           ① 不需要 map 端预聚合，且 分区数 ≤ 200？
              （spark.shuffle.sort.bypassMergeThreshold，默认 200）
               ├── 是 ──▶ BypassMergeSortShuffleHandle ──▶ BypassMergeSortShuffleWriter
               │
           ② 同时满足以下三条？
              · 序列化器支持"重定位"（supportsRelocationOfSerializedObjects）
              · 不需要 map 端预聚合
              · 分区数 ≤ 16,777,216（2^24）
               ├── 是 ──▶ SerializedShuffleHandle ──▶ UnsafeShuffleWriter（即 tungsten-sort）
               │
               └── 否 ──▶ BaseShuffleHandle ──▶ SortShuffleWriter
```

`getWriter`（`SortShuffleManager.scala:155-174`）按 Handle 的类型创建对应的 Writer。

### 4.2 BypassMergeSortShuffleWriter：不排序，直接分文件写

**做法**：为每个 reduce 分区打开一个临时文件，每条记录按分区号直接写进对应文件；最后把这些文件**按顺序拼接**成一个数据文件，再写索引文件。

`registerShuffle` 中的源码注释说得很清楚它的取舍：

- **优点**：不排序；也避免了"溢写后合并时要再序列化、反序列化一遍"的开销
- **缺点**：要同时打开很多文件，每个文件都要一块写缓冲区（`spark.shuffle.file.buffer`，默认 32k），内存占用更多

所以它只适合**分区数少**、**不需要预聚合**的场景，阈值默认 200。

### 4.3 UnsafeShuffleWriter（tungsten-sort）：只排 8 字节的"指针"

**做法**：

1. 每条记录一到达就**立刻序列化**，以二进制形式追加到内存页（memory page）里
2. 同时为每条记录生成一个 **8 字节的 `PackedRecordPointer`**，放进一个 long 数组
3. 排序时**只排这个 long 数组**（按分区号），而不是移动真正的数据
4. 内存不够就溢写；最后合并溢写文件时，可以**直接拼接二进制数据，无需反序列化**

`PackedRecordPointer` 的结构（`PackedRecordPointer.java:25`）：

```
[ 24 bit 分区号 ][ 13 bit 内存页号 ][ 27 bit 页内偏移 ]   = 64 bit = 8 字节
```

这一个设计解释了它的两个限制条件：

| 限制 | 原因 |
|---|---|
| **分区数 ≤ 2^24 = 16,777,216** | 分区号只有 24 位（`MAXIMUM_PARTITION_ID = (1 << 24) - 1`） |
| **单个内存页 ≤ 128 MB** | 页内偏移只有 27 位，2^27 字节 = 128 MB（`MAXIMUM_PAGE_SIZE_BYTES = 1 << 27`） |
| **不能做 map 端预聚合** | 数据已经序列化成二进制了，没法再按 key 合并 value |
| **序列化器必须支持"重定位"** | 排序和合并时会直接搬动、拼接序列化后的字节，要求每条记录的字节是独立的、可以挪位置的 |

**为什么快**：排序的对象只是一个紧凑的 long 数组，对 CPU 缓存非常友好；数据本身全程不需要反序列化。

### 4.4 SortShuffleWriter：通用路径

**做法**：用 `ExternalSorter` 在内存中缓存**反序列化的对象**：

- 需要预聚合（比如 `reduceByKey`）：用 `PartitionedAppendOnlyMap`，边插入边按 key 合并
- 不需要预聚合：用 `PartitionedPairBuffer`，只追加

内存不够时**溢写**到磁盘，最后把内存数据和所有溢写文件**归并**成一个数据文件。

**什么时候溢写**（`Spillable.maybeSpill`，`Spillable.scala:86`）：

```scala
// 每插入 32 条记录检查一次：当前占用 ≥ 阈值时，尝试再申请内存，把阈值扩大到当前占用的 2 倍
if (_elementsRead % 32 == 0 && currentMemory >= myMemoryThreshold) {
  val amountToRequest = 2 * currentMemory - myMemoryThreshold
  val granted = acquireMemory(amountToRequest)
  myMemoryThreshold += granted
  // 申请不到足够的内存 → 溢写
  currentMemory >= myMemoryThreshold
}
```

- 初始阈值 `spark.shuffle.spill.initialMemoryThreshold`，默认 **5 MB**
- 另外还有两个强制溢写条件：记录数超过 `spark.shuffle.spill.numElementsForceSpillThreshold`（默认 `Integer.MAX_VALUE`，即实际不限制），或内存占用超过 `spark.shuffle.spill.maxSizeInBytesForSpillThreshold`

> Spark UI 的 Stage 页面里，**Spill (Memory)** 和 **Spill (Disk)** 两列就是溢写量。溢写很多，通常意味着 Executor 内存紧张或者某些分区数据量过大（倾斜）。

### 4.5 实验：理论和实际一致吗？

我在 Spark 4.2.0 上做了 10 个用例，直接打印 `ShuffleDependency.shuffleHandle` 的类型（SQL 部分关闭了 AQE，便于观察执行计划）：

| # | 操作 | 序列化器 | 结果 |
|---|---|---|---|
| 1 | `reduceByKey(_+_, 3)` | — | `BaseShuffleHandle`（有预聚合） |
| 2 | `groupByKey(3)` | — | `BypassMergeSortShuffleHandle`（无预聚合，3 ≤ 200） |
| 3 | `groupByKey(300)`，key/value 都是 Int | 默认 Java | **`SerializedShuffleHandle`** ⭐ |
| 4 | `reduceByKey(_+_, 300)` | — | `BaseShuffleHandle`（有预聚合） |
| 6 | `groupByKey(300)`，value 是自定义类 | 默认 Java | **`BaseShuffleHandle`** ⭐ |
| 6' | 同上 | Kryo | `SerializedShuffleHandle` |
| 7 | SQL `repartition(300)` | `UnsafeRowSerializer` | `SerializedShuffleHandle` |
| 8 | SQL `repartition(200)` | `UnsafeRowSerializer` | **`BypassMergeSortShuffleHandle`** ⭐ |
| 9 | SQL `repartition(201)` | `UnsafeRowSerializer` | `SerializedShuffleHandle` |
| 10 | SQL `groupBy().count()`，默认分区数 200 | `UnsafeRowSerializer` | **`BypassMergeSortShuffleHandle`** ⭐ |

**三个反直觉的发现：**

**① 没配 Kryo，也可能用上 Kryo（用例 3）**

`JavaSerializer` 不支持重定位（基类 `Serializer.scala:98` 默认返回 `false`），按理说用例 3 应该走 `SortShuffleWriter`。但实际走的是 tungsten-sort。

原因在 `SerializerManager.getSerializer`（`SerializerManager.scala:102`）：

```scala
if (canUseKryo(keyClassTag) && canUseKryo(valueClassTag)) kryoSerializer else defaultSerializer
```

**只要 key 和 value 都是基本类型、基本类型数组或 String，Spark 会自动用 Kryo 做 Shuffle 序列化**，不管 `spark.serializer` 配的是什么。

**② 自定义类型 + 默认序列化器，用不上 tungsten-sort（用例 6）**

value 换成自定义的 case class 后，就回到了 `JavaSerializer`，只能走通用的 `SortShuffleWriter`。换成 Kryo 后才能走 tungsten-sort。这是 RDD 程序里推荐配置 Kryo 的原因之一。

> Kryo 支持重定位还有一个前提：`autoReset` 开启（默认开启）。否则 Kryo 可能在流里写对象引用而不是完整字节，破坏可重定位性（`KryoSerializer.scala:265` 的注释）。

**③ Spark SQL 默认走的是 Bypass，不是 tungsten-sort（用例 8、10）**

SQL 的 Shuffle 用 `UnsafeRowSerializer`（支持重定位），也不做 RDD 层面的 map 端预聚合（聚合在算子里完成）。但是 **`spark.sql.shuffle.partitions` 默认是 200，正好等于 bypass 阈值 200**，所以第 ① 个条件先满足了，走的是 `BypassMergeSortShuffleWriter`。分区数调到 201，才会换成 tungsten-sort。

> 实验代码见仓库 `experiments/04-shuffle-writer/`。

---

## 5. Reduce 端：数据怎么拉过来

### 5.1 先问"数据在哪"

每个 Map Task 完成后，**输出位置和每个分区的大小**（`MapStatus`）会作为 Task 结果返回 Driver，由 DAGScheduler 在处理 `CompletionEvent` 时登记到 **`MapOutputTrackerMaster`**（`DAGScheduler.scala:2358` 调用 `registerMapOutput`；`MapOutputTrackerMaster` 定义在 `MapOutputTracker.scala:708`）。

Reduce Task 启动时，通过 Executor 上的 `MapOutputTrackerWorker` 向 Driver 查询：分区 i 的数据分布在哪些 Executor 上，各有多大（`getMapSizesByExecutorId`）。

> 这也是第 2 讲里 FetchFailed 处理时要"注销丢失的 Shuffle 输出"的原因：注销的就是 MapOutputTracker 里的这些记录。

### 5.2 拉取：ShuffleBlockFetcherIterator

`BlockStoreShuffleReader`（`BlockStoreShuffleReader.scala:73`）创建 `ShuffleBlockFetcherIterator`，它会：

- **本地的块直接读**，远程的块通过网络拉取
- **边拉边处理**：拉回来一块就交给下游迭代器处理，不必等所有数据都到齐
- **控制并发量**：同时在途的数据量不超过 `spark.reducer.maxSizeInFlight`（默认 **48m**）

### 5.3 聚合与排序

拉回来的数据再根据算子的需要处理（`BlockStoreShuffleReader.scala:116` 起）：

| 情况 | 处理方式 |
|---|---|
| 需要排序（如 `sortByKey`） | 用 `ExternalSorter` 排序（需要聚合时边排边聚合） |
| 只需要聚合（如 `reduceByKey`） | 用 `Aggregator` 聚合：map 端已预聚合过，就 `combineCombinersByKey`，否则 `combineValuesByKey`。底层都是 `ExternalAppendOnlyMap`，内存不够同样会溢写 |
| 都不需要（如 `repartition`） | 直接返回 |

---

## 6. 外部 Shuffle 服务与 Push-based Shuffle

**问题**：Shuffle 文件是 Executor 写在本地磁盘上的，由 Executor 自己对外提供读取服务。如果 Executor 被回收了（比如开启了动态资源分配），它写的 Shuffle 文件就没人提供了，下游会 FetchFailed。

| 机制 | 配置（默认值） | 作用 |
|---|---|---|
| **外部 Shuffle 服务** | `spark.shuffle.service.enabled`（`false`） | 每个节点上运行一个独立于 Executor 的服务来提供 Shuffle 文件，Executor 退出后文件仍可读取 |
| **Push-based Shuffle** | `spark.shuffle.push.enabled`（`false`） | Map 端主动把数据块推送到远端合并，把大量小块合并成大块，减少 Reduce 端的随机小 I/O |

> 这部分和资源管理关系密切，第 6 讲（部署与动态资源分配）会再展开。

---

## 7. 常用配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.shuffle.manager` | `sort` | 只有 Sort Shuffle 了 |
| `spark.shuffle.sort.bypassMergeThreshold` | `200` | Bypass 路径的分区数上限 |
| `spark.sql.shuffle.partitions` | `200` | SQL 默认 Shuffle 分区数（**正好等于上一行**） |
| `spark.shuffle.file.buffer` | `32k` | 每个 Shuffle 文件输出流的缓冲区 |
| `spark.shuffle.compress` | `true` | 压缩 Map 端输出 |
| `spark.shuffle.spill.compress` | `true` | 压缩溢写文件 |
| `spark.shuffle.spill.initialMemoryThreshold` | `5 MB` | 溢写判断的初始内存阈值 |
| `spark.reducer.maxSizeInFlight` | `48m` | Reduce 端同时在途的拉取数据量上限 |
| `spark.shuffle.checksum.enabled` | `true` | 为 Shuffle 数据计算校验和，便于诊断数据损坏 |
| `spark.shuffle.service.enabled` | `false` | 外部 Shuffle 服务 |
| `spark.shuffle.push.enabled` | `false` | Push-based Shuffle |

---

## 8. 彩蛋：你的第一个 PR 候选

读 `PackedRecordPointer.java` 时，我发现第 27 行的注释是这样写的：

```java
 * This implies that the maximum addressable page size is 2^27 bits = 128 megabytes, assuming that
 * our offsets in pages are not 8-byte-word-aligned.
```

**"2^27 bits = 128 megabytes" 单位写错了**：页内偏移是按**字节**寻址的（注释后半句也说了"不是按 8 字节对齐的"），2^27 **字节**才等于 128 MB；2^27 **位**只有 16 MB。代码里对应的常量也确实是字节：`MAXIMUM_PAGE_SIZE_BYTES = 1 << 27; // 128 megabytes`。

截至我写这篇讲义时，**最新的 master 分支里这处注释仍然没改**。

这是一个非常适合作为**第一个 PR** 的小问题：改动小、风险低、容易被接受，又能完整走一遍贡献流程（fork → 建分支 → 改代码 → 提 PR → 根据 Review 修改）。

**提交前要做的**：

1. 先在 GitHub 的 apache/spark PR 列表和 JIRA 里搜一下，确认没人已经提过
2. 读一遍 https://spark.apache.org/contributing.html，按照社区对小改动（如拼写、注释修正）的 PR 规范来写标题和描述
3. 基于最新的 master 修改，不要基于 4.2.0 的学习分支

> 这一步走完，《Committer 之路 01：我给 Apache Spark 提的第一个 PR》的素材就有了。

---

## 自测题

1. Hash Shuffle 的文件数是多少？Sort Shuffle 呢？为什么前者会成为瓶颈？
2. 一个 Map Task 的 Shuffle 输出由哪两个文件组成？Reduce Task 怎么找到属于自己的那段数据？
3. 为什么 `reduceByKey` 一定走不了 Bypass 和 tungsten-sort 路径？
4. tungsten-sort 为什么要求分区数 ≤ 16,777,216？
5. RDD 的 key 和 value 都是 String，没有配置 Kryo，Shuffle 时用的是哪个序列化器？
6. Spark SQL 在默认配置下执行一个 `GROUP BY`，Map 端用的是哪种 Writer？如果把 `spark.sql.shuffle.partitions` 改成 400 呢？
7. 为什么开启动态资源分配时，往往需要外部 Shuffle 服务？

## 下一讲预告

第 4 讲：**内存管理**。Executor 的内存是怎么划分的？Execution 内存和 Storage 内存怎么互相借用？上面反复出现的 `acquireMemory`，背后是谁在分配？
:::
