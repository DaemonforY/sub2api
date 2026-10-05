---
title: "宽窄依赖：Stage 到底在哪里切开？"
description: "窄依赖和宽依赖的区别、Stage 在哪里被切开，用 10 组实验纠正 3 个常见误区。"
bigdata: "spark"
lesson: "e3"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/03-dependency-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/03-dependency-cover.webp" alt="窄依赖流水线，宽依赖分拣切分 Stage" width="1200" height="800" loading="eager" /><figcaption>窄依赖流水线，宽依赖分拣切分 Stage<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> X老师读源码 · 《Spark 源码通关》
> 基于 Spark **4.2.0**，文中结论均经过实验验证

---

"遇到宽依赖就切分 Stage。"

这句话几乎每个学 Spark 的人都会背。但如果我问你下面三个问题：

1. 窄依赖就是"父分区和子分区一对一"吗？
2. `reduceByKey`、`join` 一定是宽依赖、一定会 Shuffle 吗？
3. 把 `mapValues` 换成 `map`，结果一样，性能会有区别吗？

如果有一个答不上来，这篇文章值得读完。

## 一、源码里的依赖，一共就这几种

打开 `core/src/main/scala/org/apache/spark/Dependency.scala`，依赖的继承关系非常清晰：

```
Dependency（所有依赖的基类）
 ├─ NarrowDependency（窄依赖）
 │   ├─ OneToOneDependency     map、filter 等
 │   ├─ RangeDependency        union
 │   └─ （匿名子类）             coalesce 等
 └─ ShuffleDependency（宽依赖 / Shuffle 依赖）
```

源码里对窄依赖的注释是这样写的（`Dependency.scala:48`）：

> Base class for dependencies where each partition of the child RDD depends on a small number of partitions of the parent RDD. Narrow dependencies allow for pipelined execution.

翻译过来就是：**子 RDD 的每个分区只依赖父 RDD 的少数几个分区。窄依赖允许流水线执行。**

窄依赖的核心是一个方法：

```scala
def getParents(partitionId: Int): Seq[Int]
```

**给定子 RDD 的一个分区，返回它依赖父 RDD 的哪几个分区。**

而宽依赖（`ShuffleDependency`）没有这个方法，取而代之的是一个 `partitioner`（分区器）。因为宽依赖下，子分区的数据**来自父 RDD 的所有分区**：每个父分区的数据都要按 key 打散，分发到不同的子分区去，这个过程就是 **Shuffle**。

### 一句话区分

| | 窄依赖 | 宽依赖 |
|---|---|---|
| 本质 | 子分区只需要读父 RDD **确定的几个分区** | 父分区的数据要**按 key 打散**到多个子分区 |
| 执行方式 | 和父 RDD 在**同一个 Task 里**流水线计算 | 上游写 Shuffle 文件，下游再去读 |
| 对 Stage 的影响 | 不切分 | **切分** |

## 二、误区 1：窄依赖 = 一对一？

**不是。** 窄依赖有三种形态，只有一种是一对一。

我用 4 个分区的 RDD 做了实验：

**`map` + `filter`：OneToOneDependency，一对一**

```
最后一个RDD: MapPartitionsRDD, 依赖: OneToOneDependency
规划的Stage: S0(4t) | 实际执行: 1 个
```

`OneToOneDependency` 的实现只有一行（`Dependency.scala:267`）：子分区 i 就依赖父分区 i。

```scala
override def getParents(partitionId: Int): List[Int] = List(partitionId)
```

**`union`：RangeDependency，按区间对应**

```
最后一个RDD: UnionRDD, 依赖: RangeDependency,RangeDependency
规划的Stage: S1(8t) | 实际执行: 1 个
```

两个 4 分区的 RDD 做 `union`，结果有 8 个分区：前 4 个对应第一个 RDD，后 4 个对应第二个 RDD。**没有任何数据移动**，只是把分区"拼"在一起，所以 8 个 Task 一个 Stage 就够了。

**`coalesce(2)`：多对一，但依然是窄依赖！**

```
最后一个RDD: CoalescedRDD, 依赖: CoalescedRDD$$anon$1
规划的Stage: S2(2t) | 实际执行: 1 个
coalesce 子分区0 依赖的父分区: ArraySeq(0, 1)
```

看最后一行：**子分区 0 依赖父分区 0 和 1**，这是"多对一"。但它依然是窄依赖，只有 1 个 Stage。

源码里，`CoalescedRDD` 用一个匿名的 `NarrowDependency` 实现了 `getParents`（`CoalescedRDD.scala:105`）：

```scala
Seq(new NarrowDependency(prev) {
  def getParents(id: Int): Seq[Int] =
    partitions(id).asInstanceOf[CoalescedRDDPartition].parentsIndices.toImmutableArraySeq
})
```

**关键在于：** 父分区 0 的数据**整个**交给子分区 0，不需要拆开分给多个子分区。所以子分区 0 的 Task 直接把父分区 0、1 的数据依次读进来就行，不需要 Shuffle。

> **判断窄依赖的正确方式**：不看"一对几"，看**父分区的数据是否需要被拆开、分发到多个子分区**。需要拆开，就是宽依赖。

## 三、Stage 到底在哪里切开？看源码

<figure class="ai-figure"><img src="/bigdata-img/spark/03-dependency-1.webp" alt="沿依赖回溯，遇到 Shuffle 就切 Stage" width="960" height="640" loading="lazy" /><figcaption>沿依赖回溯，遇到 Shuffle 就切 Stage<span>AI 生成配图</span></figcaption></figure>

DAGScheduler 切分 Stage 的核心逻辑在 `getShuffleDependenciesAndResourceProfiles` 方法里（`DAGScheduler.scala:801`），简化后就这几行：

```scala
traverseRDDGraph(rdd) { (toVisit, enqueue) =>
  toVisit.dependencies.foreach {
    case shuffleDep: ShuffleDependency[_, _, _] =>
      parents += shuffleDep        // 遇到宽依赖：记下来，不再往上走
    case dependency =>
      enqueue(dependency.rdd)      // 遇到窄依赖：继续往上游遍历
  }
}
```

**这就是"在宽依赖处切开"的真正含义**：

- 从当前 RDD 出发往上游走
- 遇到**窄依赖**，父 RDD 和自己属于同一个 Stage，继续往上走
- 遇到**宽依赖**，停下来，记录这个 Shuffle 依赖，它的上游会成为一个新的 Stage

这个方法的注释还特别说明了一点：如果 `A <-- B <-- C` 之间都是 Shuffle 依赖，从 C 调用时**只会返回 `B <-- C` 这一个**。更上游的 Stage，是在为 B 创建 Stage 时再递归找出来的。**一层一层往上切**。

### 小技巧：用 `toDebugString` 一眼看出 Stage 边界

```scala
rdd.toDebugString
```

```
(3) ShuffledRDD[47] at reduceByKey []
 +-(3) MapPartitionsRDD[46] at map []
    |  ShuffledRDD[45] at partitionBy []
    +-(4) MapPartitionsRDD[44] at map []
       |  ParallelCollectionRDD[43] at parallelize []
```

- 每个 **`+-`** 表示一个 **Shuffle 边界**，也就是 Stage 切开的地方
- 括号里的数字是分区数，也就是该 Stage 的 Task 数
- 同一缩进层级、用 `|` 连起来的 RDD，属于**同一个 Stage**

这个例子里有两个 `+-`，所以是 **3 个 Stage**。

## 四、误区 2：reduceByKey、join 一定是宽依赖？

<figure class="ai-figure"><img src="/bigdata-img/spark/03-dependency-2.webp" alt="保留分区器可本地聚合，丢失则触发 Shuffle" width="960" height="640" loading="lazy" /><figcaption>保留分区器可本地聚合，丢失则触发 Shuffle<span>AI 生成配图</span></figcaption></figure>

**不一定。** 这是最容易被忽略的一点。

看 `reduceByKey` 底层调用的 `combineByKeyWithClassTag`（`PairRDDFunctions.scala:92`）：

```scala
if (self.partitioner == Some(partitioner)) {
  self.mapPartitions(iter => { ... }, preservesPartitioning = true)   // 不 Shuffle！
} else {
  new ShuffledRDD[K, V, C](self, partitioner)                         // Shuffle
}
```

**如果数据已经按同样的分区器分好区了，就直接在每个分区内聚合，根本不需要 Shuffle。**

道理很简单：同一个 key 的数据已经在同一个分区里了，还打散它干什么？

### 实验对比

```scala
val p3 = new HashPartitioner(3)

// 普通的 reduceByKey
pairs.reduceByKey(_ + _, 3)
// 规划的Stage: S5(4t) S6(3t)  → 2 个 Stage

// 先 partitionBy，再用同一个分区器 reduceByKey
pairs.partitionBy(p3).reduceByKey(p3, _ + _)
// 最后一个RDD: MapPartitionsRDD, 依赖: OneToOneDependency
// 规划的Stage: S7(4t) S8(3t)  → 还是 2 个 Stage
```

第二种写法里有 `partitionBy` 和 `reduceByKey` **两个**按 key 分区的操作，但只有 **2 个** Stage，也就是**只 Shuffle 了一次**（`partitionBy` 那次）。看依赖：`reduceByKey` 产生的是 `OneToOneDependency`，窄依赖。

`join` 也一样。`join` 底层是 `CoGroupedRDD`，它会**对每个父 RDD 单独判断**（`CoGroupedRDD.scala:100`）：

```scala
rdds.map { rdd =>
  if (rdd.partitioner == Some(part)) {
    new OneToOneDependency(rdd)      // 分区器相同：窄依赖
  } else {
    new ShuffleDependency(...)       // 分区器不同：宽依赖
  }
}
```

实验：

```scala
// 普通 join
val a = pairs
a.join(a.mapValues(_ + 1), p3)
// 规划的Stage: S14(4t) S15(4t) S16(3t)  → 3 个 Stage，两边都要 Shuffle

// 预分区后 join
val a = pairs.partitionBy(p3)
a.join(a.mapValues(_ + 1), p3)
// 规划的Stage: S17(4t) S18(3t)  → 2 个 Stage，join 本身不 Shuffle
```

> **生产实践**：如果一个大 RDD 要和别的数据**多次 join**，可以先 `partitionBy` 再 `persist()`，之后每次 join 都不用再 Shuffle 这个大 RDD。

## 五、误区 3：`map` 和 `mapValues` 只是写法不同？

这是一个**真实的性能坑**。

```scala
// 6b：先分区，再 map，再 reduceByKey
pairs.partitionBy(p3).map(identity).reduceByKey(p3, _ + _)
// 规划的Stage: S9(4t) S10(3t) S11(3t)  → 3 个 Stage！

// 6c：先分区，再 mapValues，再 reduceByKey
pairs.partitionBy(p3).mapValues(_ + 1).reduceByKey(p3, _ + _)
// 规划的Stage: S12(4t) S13(3t)  → 2 个 Stage
```

`6b` 里的 `map(identity)` 什么都没改，**却多了一次 Shuffle**。

原因在 `MapPartitionsRDD.scala:52`：

```scala
override val partitioner = if (preservesPartitioning) firstParent[T].partitioner else None
```

- `map` 创建 `MapPartitionsRDD` 时，`preservesPartitioning` 是 `false`，**分区器直接丢了**
- `mapValues` 创建时传的是 `preservesPartitioning = true`（`PairRDDFunctions.scala:755`），**分区器保留**

为什么 `map` 要丢掉分区器？因为 `map` 可以修改 key。Spark 无法知道你的函数有没有改 key，key 一变，原来的分区就不对了，只能保守地认为"不再有序分区"。而 `mapValues` 从 API 上就保证了**只改 value，不改 key**。

> **生产实践**：对键值对 RDD，**只修改 value 时，用 `mapValues` / `flatMapValues`，不要用 `map`**。

## 六、常用算子速查表

| 算子 | 依赖类型 | 说明 |
|---|---|---|
| `map`、`filter`、`flatMap`、`mapPartitions` | 窄（OneToOne） | `map` 等会丢失分区器 |
| `mapValues`、`flatMapValues` | 窄（OneToOne） | **保留分区器** |
| `union` | 窄（Range） | 分区直接拼接 |
| `coalesce(n)`（默认 `shuffle = false`） | 窄（多对一） | 只能减少分区数；要求更多分区时保持原分区数不变 |
| `repartition(n)` | **宽** | 等价于 `coalesce(n, shuffle = true)`（`RDD.scala:489`） |
| `reduceByKey`、`groupByKey`、`aggregateByKey`、`combineByKey` | **通常宽** | 父 RDD 已有相同分区器时为窄 |
| `join`、`cogroup` | **对每个父 RDD 分别判断** | 分区器相同的一侧为窄 |
| `distinct` | **通常宽** | 底层是 `map` + `reduceByKey`；已分区且分区数不变时在分区内去重 |
| `partitionBy` | **通常宽** | 分区器相同时直接返回自身（`PairRDDFunctions.scala:532`） |
| `sortByKey` | **宽** | 还会额外触发一个抽样 Job（见上一期面试拆解） |

## 七、面试怎么答

> **问：什么是宽依赖和窄依赖？Stage 怎么划分？**
>
> 窄依赖是指子 RDD 的每个分区只依赖父 RDD 中确定的少数几个分区，父分区的数据不需要拆开分发，可以和父 RDD 在同一个 Task 里流水线计算。它有一对一（map、filter）、按区间（union）、多对一（coalesce）几种形式。
>
> 宽依赖也就是 ShuffleDependency，父分区的数据需要按 key 重新分发到多个子分区，必须经过 Shuffle。
>
> DAGScheduler 从最后一个 RDD 开始往上游遍历依赖：遇到窄依赖就继续往上，属于同一个 Stage；遇到 ShuffleDependency 就停下来，在这里切分，上游成为新的 Stage。
>
> **加分点**：reduceByKey、join 不一定是宽依赖。如果父 RDD 已经有相同的分区器，Spark 会直接用窄依赖，避免 Shuffle。所以对键值 RDD 只改 value 时应该用 mapValues 而不是 map，因为 map 会丢掉分区器，导致后续多一次 Shuffle。

## 速记卡

```
┌──────────────────────────────────────────┐
│     宽窄依赖速记卡（Spark 4.2.0）          │
├──────────────────────────────────────────┤
│ 窄依赖：父分区数据不用拆开 → 同 Stage 流水线│
│   · 一对一  map / filter                  │
│   · 区间   union                          │
│   · 多对一  coalesce                       │
│ 宽依赖：按 key 打散到多个子分区 → 切 Stage  │
├──────────────────────────────────────────┤
│ ⭐ 窄依赖 ≠ 一对一（coalesce 是多对一）     │
│ ⭐ 分区器相同 → reduceByKey/join 不 Shuffle │
│ ⭐ map 丢分区器，mapValues 保留            │
│ ⭐ toDebugString 里的 +- 就是 Stage 边界    │
└──────────────────────────────────────────┘
```

---

**思考题**：

`coalesce(n)` 默认不 Shuffle，但只能**减少**分区数。如果想把 2 个分区扩到 10 个，必须用 `coalesce(10, shuffle = true)` 或者 `repartition(10)`。

为什么"增加分区"就一定要 Shuffle？结合窄依赖的定义想一想。

> 本文实验代码已放在 GitHub：https://github.com/DaemonforY/spark-source-notes，复制到 `spark-shell` 里就能复现。

—— X老师读源码
:::
