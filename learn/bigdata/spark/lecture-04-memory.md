---
title: "第 4 讲：内存管理"
description: "统一内存管理：执行内存与存储内存的划分、互相借用的规则，以及堆外内存。"
bigdata: "spark"
lesson: "e7"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-04-memory-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-04-memory-cover.webp" alt="Yui和Kai管理Executor内存城堡" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai管理Executor内存城堡<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 配置默认值、公式、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；内存划分和借用规则经过实验验证（仓库 `experiments/05-memory/`）。

## 0. 本讲要回答的问题

1. 我给 Executor 配了 4g 内存，Spark 真正能用来计算和缓存的有多少？
2. 计算用的内存（Execution）和缓存用的内存（Storage）怎么分？能不能互相借？
3. 一个 Executor 上同时跑多个 Task，它们怎么分内存？
4. 第 3 讲里反复出现的 `acquireMemory`、溢写，背后是谁在协调？
5. 堆外内存是什么，什么时候该用？
6. 遇到 OOM，该调哪个参数？

---

## 1. 从容器视角看：一个 Executor 占多少内存

在 YARN / K8s 上，一个 Executor 是一个容器。它向集群申请的总内存是（`ResourceProfile.scala:562`）：

```
容器总内存 = spark.executor.memory              JVM 堆内存（默认 1g）
          + memoryOverhead                    JVM 堆外的额外开销
          + spark.memory.offHeap.size         Spark 管理的堆外内存（仅当 offHeap.enabled=true）
          + spark.executor.pyspark.memory     Python worker 内存（仅 Python 应用且设置了该项）
```

其中 `memoryOverhead` 如果没有显式设置，按下面的公式计算（`ResourceProfile.calculateOverHeadMemory`）：

```
memoryOverhead = max(spark.executor.memoryOverheadFactor × executor.memory,  spark.executor.minMemoryOverhead)
               = max(0.1 × executor.memory, 384MB)                           ← 默认值
```

**overhead 里装的是什么**：JVM 自身的元空间、线程栈、直接内存（Netty 网络缓冲区）、本地库等，这些都不在 `-Xmx` 堆里，但一样占用容器的物理内存。

> 💡 "**Container killed by YARN for exceeding memory limits**" 这类报错，往往不是堆不够，而是 **overhead 不够**。这时该调大的是 `spark.executor.memoryOverhead`，而不是 `spark.executor.memory`。

---

## 2. 堆内存怎么划分：统一内存模型

### 2.1 划分公式

```
JVM 堆（Runtime.maxMemory，通常就是 spark.executor.memory）
 ├── 预留内存 Reserved：固定 300MB                 ← RESERVED_SYSTEM_MEMORY_BYTES
 └── 可用内存 usable = 堆 - 300MB
      ├── 统一内存 Unified = usable × 0.6          ← spark.memory.fraction
      │    ├── Storage 区域 = Unified × 0.5         ← spark.memory.storageFraction（是"软边界"，见第 3 节）
      │    └── Execution 区域 = Unified × 0.5
      └── 用户内存 User = usable × 0.4             ← 用户代码里的对象、Spark 内部的元数据等
```

源码在 `UnifiedMemoryManager.getMaxMemory`（`UnifiedMemoryManager.scala:462`）。

- **预留内存 300MB** 用来保证 Spark 自身运行。堆内存至少要是它的 1.5 倍，也就是 **450MB**，否则直接报错（错误类 `INVALID_DRIVER_MEMORY` / `INVALID_EXECUTOR_MEMORY`）
- 计算用的"系统内存"取的是 JVM 报告的 **`Runtime.getRuntime.maxMemory`**，不是配置里的数字。不同 GC 配置下它可能比 `-Xmx` 略小

### 2.2 实验验证

| 配置 | JVM maxMemory | 公式 `(堆 - 300MB) × 0.6` | 实际统一内存 | Storage 区域 |
|---|---|---|---|---|
| `--driver-memory 1g` | 1024.0 MB | 434.4 MB | **434.4 MB** | 217.2 MB |
| `--driver-memory 2g` | 2048.0 MB | 1048.8 MB | **1048.8 MB** | 524.4 MB |

**和公式完全一致。** 也就是说：配了 1g，真正给计算和缓存用的只有 434 MB，不到一半。

> local 模式下 Driver 兼任 Executor，所以这里用 `--driver-memory` 来控制。

### 2.3 演进史

| 时间 | 变化 | 提交 |
|---|---|---|
| Spark 1.6 之前 | **静态内存管理**：Storage、Execution 各占固定比例，互不借用 | — |
| 2015-10，**Spark 1.6** | 引入**统一内存管理**，成为默认；旧模型保留为 legacy 模式 | `[SPARK-10983] Unified memory manager` |
| 2019-01，**Spark 3.0** | **移除** `StaticMemoryManager` 和 `spark.memory.useLegacyMode` | `[SPARK-26539]` |
| 2025-07，**Spark 4.1** | 统一内存管理开始感知 RocksDB 等"非托管内存"（见第 8 节） | `[SPARK-53001]` |

> 网上还有不少文章在讲 `spark.storage.memoryFraction`、`spark.shuffle.memoryFraction` 这些静态内存模型的参数。静态内存管理器在 Spark 3.0 已被移除，**在 4.2.0 的源码里已经完全找不到这两个参数，设置了也不会生效**。

---

## 3. Storage 和 Execution 怎么互相借：不对称的借用规则

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-04-memory-1.webp" alt="Storage与Execution软边界互相借用" width="960" height="640" loading="lazy" /><figcaption>Storage与Execution软边界互相借用<span>AI 生成配图</span></figcaption></figure>

统一内存模型的核心是：**Storage 和 Execution 之间的边界是"软"的**，空闲时可以互相借用。但两个方向的规则**不对称**。

### 3.1 Storage 借 Execution：只能借空闲的

`UnifiedMemoryManager.acquireStorageMemory`（`UnifiedMemoryManager.scala:206`）：

```scala
if (numBytes > storagePool.memoryFree) {
  // Storage 池不够，从 Execution 池借**空闲**的内存
  val memoryBorrowedFromExecution = Math.min(executionPool.memoryFree, numBytes - storagePool.memoryFree)
  executionPool.decrementPoolSize(memoryBorrowedFromExecution)
  storagePool.incrementPoolSize(memoryBorrowedFromExecution)
}
```

- **只能借空闲的**，不能把 Execution 正在用的内存抢过来
- 借完还不够：在 Storage 内部按 LRU 淘汰其他缓存块（见第 6 节）；再不够，这个块就缓存失败

### 3.2 Execution 借 Storage：可以"收回"，但只收回到边界

`maybeGrowExecutionPool`（`UnifiedMemoryManager.scala:160`）：

```scala
// 可以从 Storage 收回的内存 = max(Storage 的空闲内存, Storage 超出自己区域的部分)
val memoryReclaimableFromStorage = math.max(
  storagePool.memoryFree,
  storagePool.poolSize - storageRegionSize)
// 必要时会驱逐缓存块来腾出空间（freeSpaceToShrinkPool）
```

- Execution 可以拿走 Storage 的**空闲内存**
- 如果 Storage 之前借了 Execution 的内存（池子超出了 Storage 区域），Execution 可以**驱逐缓存块，把借出去的收回来**
- **但最多只收回到 Storage 区域的边界**（`storageFraction` 那条线），区域以内的缓存是受保护的

### 3.3 为什么不对称

| | Execution 的内存 | Storage 的内存 |
|---|---|---|
| 里面是什么 | 正在进行的计算的中间数据（排序缓冲、哈希表） | 缓存的数据块 |
| 被拿走的后果 | 计算无法继续，Task 失败 | 缓存丢了，**可以重算或从磁盘读** |
| 规则 | **不能被 Storage 抢** | **可以被 Execution 收回**（但有保底区域） |

保底区域的意义：防止一个计算密集的 Job 把所有缓存都挤掉，导致缓存反复失效、反复重算。

### 3.4 实验验证（1g 堆，统一内存 434.4 MB，Storage 区域 217.2 MB）

```
[B1] 缓存 38 个 8MB 分区：Storage 已用 304.0 MB（Storage 区域只有 217.2 MB）
[B2] 大聚合之后：缓存剩 27 个分区，Storage 已用 216.0 MB
     该聚合的峰值 Execution 内存 223.4 MB，溢写 446.8 MB
```

逐行解读：

1. **B1**：缓存了 304 MB，超出 Storage 区域 87 MB。这多出来的部分是**从 Execution 借的空闲内存**（规则 3.1）
2. **B2**：一个大聚合来要内存，Execution 驱逐了 11 个缓存块，把 Storage 压回到 **216 MB**，正好落在 217.2 MB 的边界以内（规则 3.2）
3. Execution 拿到了约 434 - 216 ≈ 218 MB（峰值 223.4 MB，与之吻合）后，**就不再继续驱逐缓存，转而溢写到磁盘**（446.8 MB）—— 区域内的缓存受到了保护

---

## 4. 多个 Task 怎么分 Execution 内存：1/2N 到 1/N

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-04-memory-2.webp" alt="多个Task按公平份额争用执行内存" width="960" height="640" loading="lazy" /><figcaption>多个Task按公平份额争用执行内存<span>AI 生成配图</span></figcaption></figure>

一个 Executor 有多个核，就会同时跑多个 Task，它们共享同一个 Execution 池。

`ExecutionMemoryPool` 的规则（`ExecutionMemoryPool.scala:33` 的类注释、第 128-129 行）：

```
N = 当前活跃的 Task 数
每个 Task 至少能拿到   1 / 2N    （拿不到会等待，而不是立刻溢写）
每个 Task 最多只能拿   1 / N     （超过就要溢写）
```

```scala
val maxMemoryPerTask = maxPoolSize / numActiveTasks
val minMemoryPerTask = poolSize / (2 * numActiveTasks)
```

- N 是动态变化的：Task 启动、结束都会触发重新计算
- **推论**：Executor 的核数越多，每个 Task 能用的内存越少。如果单个 Task 需要很多内存（比如大分区的聚合），**减少 `spark.executor.cores`** 和**增加内存**有相似的效果

---

## 5. Task 内部：TaskMemoryManager 和 MemoryConsumer

一个 Task 里也可能有多个"吃内存"的组件，比如一个 `ExternalSorter` 加一个 `ExternalAppendOnlyMap`。它们都继承 **`MemoryConsumer`**，由每个 Task 独有的 **`TaskMemoryManager`** 统一协调。

`TaskMemoryManager.acquireExecutionMemory`（`TaskMemoryManager.java:159`）的流程：

```
1. 向 MemoryManager 申请内存
2. 不够？→ 先让同一个 Task 里的【其他】Consumer 溢写，腾出内存
     · 选择策略：优先选"占用内存 ≥ 还缺的量"的 Consumer 里占用最小的那个
     · 目的：减少溢写次数和小溢写文件，又不多溢写不必要的数据
3. 还不够？→ 让【申请者自己】溢写
```

> 第 3 讲 `Spillable.maybeSpill` 里的 `acquireMemory`，就是走的这条路径。溢写的根本原因，是在这里申请不到内存了。

---

## 6. Storage 内部：MemoryStore 与 LRU 淘汰

缓存块存放在 `MemoryStore` 里，底层是一个**按访问顺序排列**的 `LinkedHashMap`（`MemoryStore.scala:93`）：

```scala
private val entries = new LinkedHashMap[BlockId, MemoryEntry[_]](32, 0.75f, true)  // 第三个参数 true = accessOrder
```

所以淘汰时，**最久没被访问的块最先被淘汰**（LRU）。

淘汰时有一个重要的保护规则（`evictBlocksToFreeSpace`，`MemoryStore.scala:472`）：

```scala
// 不淘汰和"正要放进来的块"属于同一个 RDD 的块
entry.memoryMode == memoryMode && (rddToAdd.isEmpty || rddToAdd != getRddId(blockId))
```

**为什么**：如果缓存 RDD A 的第 10 个分区时，把 A 的第 1 个分区挤出去，那缓存 A 就变成了"边缓存边丢"，永远缓存不全。

**被淘汰的块去哪了**：

- `MEMORY_ONLY`：直接丢弃，下次用到时**重算**
- `MEMORY_AND_DISK`：写到磁盘，下次**从磁盘读**

> 还有一个细节：缓存一个分区时，Spark 并不知道它展开后有多大，所以会一边迭代一边申请"展开内存"（unroll memory），初始申请 `spark.storage.unrollMemoryThreshold`（默认 **1MB**），不够再逐步加，确认放得下才真正缓存。

---

## 7. 堆外内存与 Tungsten

### 7.1 堆内 vs 堆外

| | 堆内（ON_HEAP，默认） | 堆外（OFF_HEAP） |
|---|---|---|
| 开启方式 | 默认 | `spark.memory.offHeap.enabled=true` 且 `spark.memory.offHeap.size > 0`（默认 `false` / `0`） |
| 内存在哪 | JVM 堆里的 `long[]` 数组 | 通过 `Unsafe` 直接分配的本地内存 |
| GC | 受 GC 管理，大堆可能停顿明显 | **不受 GC 管理** |
| 大小计算 | 有 Java 对象头开销，估算不精确 | 按字节精确管理 |
| 代价 | — | 要额外配置；**计入容器内存**（第 1 节公式）；用错了容易造成本地内存泄漏 |

开启后，Spark 会额外建一组堆外的 Storage / Execution 池，Tungsten 的数据结构（第 3 讲的 `UnsafeShuffleWriter`、SQL 的 `UnsafeRow` 等）会改用堆外内存（`MemoryManager.tungstenMemoryMode`，`MemoryManager.scala:228`）。

### 7.2 Tungsten 的地址编码

不管堆内还是堆外，Tungsten 都用一个 **64 位的 long** 表示一个内存地址（`TaskMemoryManager.java` 第 61-77 行）：

```
[ 13 bit 页号 ][ 51 bit 页内偏移 ]
```

- 每个 Task 最多 **2^13 = 8192 个页**（`PAGE_TABLE_SIZE`）
- 堆内模式下，一个页是一个 `long[]`，最大 `((2^31 - 1) × 8)` 字节，约 **16 GB**（`MAXIMUM_PAGE_SIZE_BYTES`）

> 对比第 3 讲的 `PackedRecordPointer`：它要把**分区号**也塞进 8 字节，所以页号仍是 13 位，但页内偏移被压缩到 27 位，单页只能 128 MB。两者的思路是一样的：**用一个 long 装下"在哪一页、页内哪个位置"**，这样排序时只需要排 long。

---

## 8. 新机制：非托管内存（4.1+）

Structured Streaming 的 RocksDB 状态存储、一些本地库，会**自己管理内存**，不经过 Spark 的统一内存管理。以前 Spark 完全不知道它们用了多少，容易出现"Spark 以为还有内存，实际已经被 RocksDB 占满"的情况。

`[SPARK-53001]`（2025-07）让统一内存管理开始**感知这部分非托管内存**（`UnifiedMemoryManager.scala:70` 起的注释）：

- 开启后，Storage 和 Execution 计算可用内存时都会**扣除非托管内存的用量**（`computeMaxExecutionPoolSize`、`acquireStorageMemory` 里的 `getUnmanagedMemoryUsed`）
- 配置：`spark.memory.unmanagedMemoryPollingInterval`，**4.1.0 引入**，默认 **`0s`（不开启轮询）**

> 这是一个比较新的机制，中文资料很少。如果你想往 Structured Streaming / RocksDB 方向深入，这里是一个不错的切入点。

---

## 9. OOM 排查地图

| 现象 | 通常的原因 | 优先尝试 |
|---|---|---|
| Executor 报 `java.lang.OutOfMemoryError: Java heap space` | 单个 Task 处理的数据太多（倾斜、分区太大）；用户代码里创建了大对象 | 增加分区数 / 处理倾斜；减少 `executor.cores`；增加 `executor.memory` |
| **Container killed … exceeding memory limits** | 堆外开销超出 overhead：Netty 缓冲、本地库、Python worker | 调大 `spark.executor.memoryOverhead` |
| Driver OOM | `collect()` 拉回太多数据；广播的变量太大；太多分区 / Task 的元数据 | 避免大 `collect`；`spark.driver.maxResultSize`；增加 `driver.memory` |
| 大量溢写、任务很慢 | Execution 内存紧张 | 减少每个 Task 的数据量；调整 `spark.memory.fraction`（谨慎） |
| 缓存命中率低、反复重算 | Storage 不够，缓存被淘汰 | 用 `MEMORY_AND_DISK`；序列化缓存（`MEMORY_ONLY_SER`）；只缓存真正复用的数据 |

> 先看 Spark UI 的 **Executors** 页面（Storage Memory、GC Time）和 **Stage** 页面（Spill、Peak Execution Memory），再决定调哪个参数。

---

## 10. 常用配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.executor.memory` | `1g` | Executor 堆内存 |
| `spark.executor.memoryOverheadFactor` | `0.1` | overhead 系数 |
| `spark.executor.minMemoryOverhead` | `384m` | overhead 下限 |
| `spark.memory.fraction` | `0.6` | 统一内存占（堆 - 300MB）的比例 |
| `spark.memory.storageFraction` | `0.5` | Storage 保底区域占统一内存的比例 |
| `spark.memory.offHeap.enabled` | `false` | 是否使用堆外内存 |
| `spark.memory.offHeap.size` | `0` | 堆外内存大小 |
| `spark.storage.unrollMemoryThreshold` | `1MB` | 展开缓存块时的初始申请量 |
| `spark.memory.unmanagedMemoryPollingInterval` | `0s` | 非托管内存轮询间隔（4.1.0+，0 表示不开启） |

---

## 自测题

1. `spark.executor.memory=4g`，在默认配置下统一内存是多少？Storage 区域是多少？
2. YARN 上 Executor 被杀，报错 "exceeding memory limits"。应该优先调大哪个参数？为什么？
3. Storage 已经占满了 Storage 区域，Execution 还有很多空闲。这时再缓存一个块，会发生什么？
4. Execution 需要更多内存，而 Storage 区域内还有很多缓存块。Execution 能驱逐它们吗？为什么这样设计？
5. 一个 Executor 有 8 个核，同时跑 8 个 Task，每个 Task 最多能拿到多少 Execution 内存？
6. 为什么缓存 RDD A 时，Spark 不会淘汰 A 自己的其他分区？
7. 堆外内存有哪些好处？开启它之后，容器申请的内存会怎么变化？
8. 网上有文章教你调 `spark.shuffle.memoryFraction`，在 Spark 4.2.0 里有用吗？

## 下一讲预告

第 5 讲：**存储体系**。BlockManager 是怎么管理缓存块、Shuffle 块和广播变量的？广播变量是怎么分发到每个 Executor 上的？
:::
