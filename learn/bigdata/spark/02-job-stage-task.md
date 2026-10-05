---
title: "面试拆解 01｜Job、Stage、Task 到底是什么关系？"
description: "用 7 个实验说清楚 Action、Job、Stage、Task 的对应关系，以及常见面试题怎么答。"
bigdata: "spark"
lesson: "e2"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/02-job-stage-task-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/02-job-stage-task-cover.webp" alt="动作触发作业并拆成阶段任务" width="1200" height="800" loading="eager" /><figcaption>动作触发作业并拆成阶段任务<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> X老师读源码 · 《面试拆解》第 1 期
> 基于 Spark **4.2.0**，文中结论均经过实验验证

---

## 面试官的问题

> "说说 Spark 里 Job、Stage、Task 的关系。"

这道题几乎是 Spark 面试的必考题。大多数人会这样回答：

> "一个 Action 产生一个 Job，Job 按宽依赖切分成多个 Stage，每个 Stage 里每个分区对应一个 Task。"

**这个回答没错，但只能拿及格分。**

因为面试官接下来很可能追问：

- 一个 Action **一定**只产生一个 Job 吗？
- Stage 是从前往后切，还是从后往前切？
- Task 的数量**一定**等于分区数吗？
- 同一个 RDD 执行两次 Action，Stage 会重新算吗？

这篇文章用 **7 个实验**，把这些问题一次讲透。

## 一、先把基本关系讲清楚

<figure class="ai-figure"><img src="/bigdata-img/spark/02-job-stage-task-1.webp" alt="宽依赖把作业切成不同阶段" width="960" height="640" loading="lazy" /><figcaption>宽依赖把作业切成不同阶段<span>AI 生成配图</span></figcaption></figure>

```
Application（一次 spark-submit，对应一个 SparkContext）
 └─ Job       由 Action 算子触发
     └─ Stage     在宽依赖（Shuffle）处切分
         └─ Task      每个分区一个，最小的执行单元
```

| 层级 | 由什么决定 | 谁负责 | 类型 |
|---|---|---|---|
| **Job** | Action 算子调用 `sc.runJob()` | — | — |
| **Stage** | 宽依赖（`ShuffleDependency`）的位置 | DAGScheduler | `ShuffleMapStage`、`ResultStage` |
| **Task** | Stage 中**需要计算的分区** | TaskScheduler 分配到 Executor 执行 | `ShuffleMapTask`、`ResultTask` |

- **Job**：在源码里，所有 Action 最终都会调用 `sc.runJob()`。比如 `count()` 就是一行代码：`sc.runJob(this, Utils.getIteratorSize _).sum`（`RDD.scala:1320`）
- **Stage**：最后一个 Stage 叫 `ResultStage`，负责产出 Action 的结果；其余都是 `ShuffleMapStage`，负责为下游写 Shuffle 数据
- **Task**：`ShuffleMapStage` 里跑的是 `ShuffleMapTask`，`ResultStage` 里跑的是 `ResultTask`

## 二、7 个实验，验证每一个结论

我写了一个 `SparkListener`，监听每个 Job 包含哪些 Stage、每个 Stage 有几个 Task、**实际执行了哪些 Stage**。测试数据是 8 个单词、4 个分区：

```scala
val words = sc.parallelize(Seq("a","b","c","a","b","a","d","e"), 4)
```

### 实验 1：只有窄依赖 → 1 个 Stage

```scala
words.map(_ * 2).filter(_ != "dd").count()
```

```
Job0: Stage0(4 tasks)
  -> 实际执行 Stage0: 4 tasks
```

`map` 和 `filter` 都是窄依赖，不切分 Stage。4 个分区 → 4 个 Task。这三个算子在同一个 Task 里**流水线式**执行，数据逐条流过 `map` → `filter` → `count`，不会产生中间结果集。

### 实验 2：一次 Shuffle → 2 个 Stage

```scala
val counts = words.map(w => (w, 1)).reduceByKey(_ + _, 3)
counts.collect()
```

```
Job1: Stage1(4 tasks, map) | Stage2(3 tasks, collect, parents=1)
  -> 实际执行 Stage1: 4 tasks
  -> 实际执行 Stage2: 3 tasks
```

- `reduceByKey` 产生宽依赖，在这里切成两个 Stage
- Stage1 是 `ShuffleMapStage`，有 **4** 个 Task（上游 4 个分区）
- Stage2 是 `ResultStage`，有 **3** 个 Task（`reduceByKey` 指定了 3 个分区）

**结论：两个 Stage 的 Task 数可以不一样，各自由该 Stage 的分区数决定。**

### 实验 3：同一个 RDD 再执行一次 Action → Stage 被跳过 ⭐

```scala
counts.count()   // counts 就是实验 2 里的那个 RDD
```

```
Job2: Stage3(4 tasks, map) | Stage4(3 tasks, count, parents=3)
  -> 实际执行 Stage4: 3 tasks          ← Stage3 没有执行！
```

**这是面试加分点。**

新的 Action 产生了新的 Job，Job 里也规划了两个 Stage。但 Stage3 **实际上没有运行**，因为实验 2 已经把这次 Shuffle 的数据写到磁盘了，Spark 直接复用。在 Spark UI 上，这类 Stage 会显示为 **skipped**。

源码里，`DAGScheduler.getMissingParentStages`（`DAGScheduler.scala:837`）在找"缺失的父 Stage"时，遇到宽依赖会检查：

```scala
// DAGScheduler.scala:856
if (!mapStage.isAvailable || !mapStage.shuffleDep.shuffleMergeFinalized) {
  missing += mapStage
}
```

而 `isAvailable` 的定义很直白：**所有分区的 Shuffle 输出都在**（`ShuffleMapStage.scala:89`）：

```scala
def isAvailable: Boolean = numAvailableOutputs == numPartitions
```

> 所以面试时可以补一句：**Shuffle 的输出本身就是一种"隐式缓存"**。这也是为什么有时候你没调用 `cache()`，第二次 Action 却快了很多。

### 实验 4：两次 Shuffle → 3 个 Stage（且复用前面的）

```scala
counts.map { case (w, c) => (c, w) }.groupByKey(2).collect()
```

```
Job3: Stage5(4 tasks) | Stage6(3 tasks, parents=5) | Stage7(2 tasks, parents=6)
  -> 实际执行 Stage6: 3 tasks
  -> 实际执行 Stage7: 2 tasks          ← Stage5 又被跳过了
```

两次宽依赖（`reduceByKey`、`groupByKey`）切出 3 个 Stage。最上游的 Stage5 就是 `counts` 那次 Shuffle，依然被复用、跳过。

### 实验 5：`take(1)` → 只有 1 个 Task ⭐

```scala
words.take(1)
```

```
Job4: Stage8(1 tasks, take)
```

`words` 明明有 4 个分区，**为什么只有 1 个 Task？**

因为 `take` 很"聪明"：它**先只扫描 1 个分区**，数据够了就直接返回；不够，再按倍数扩大扫描范围，**每扩大一次就提交一个新 Job**。源码在 `RDD.scala` 的 `take` 方法里：

```scala
// RDD.scala:1516
val res = sc.runJob(this, (it: Iterator[T]) => it.take(left).toArray, p)
```

注意第三个参数 `p`：它指定了**这次只计算哪些分区**。扩大的倍数由 `spark.rdd.limit.scaleUpFactor` 控制，默认是 4。

**所以：**
1. **Task 数不一定等于 RDD 的分区数**，而是等于 `runJob` 时**要计算的分区数**
2. **一个 Action 可能产生多个 Job**：`take(n)` 在前几个分区数据不够时，会连续提交多个 Job

### 实验 6：`sortByKey` 是转换算子，却触发了 Job ⭐⭐

```scala
words.map(w => (w, 1)).sortByKey()   // 注意：没有调用任何 Action！
```

```
Job5: Stage9(4 tasks, sortByKey)
```

**这是最反直觉的一个。** 按照"转换算子是惰性的"这条规则，这里不应该有任何 Job。

原因在于 `sortByKey` 要用 `RangePartitioner`（范围分区器）：为了让排序后每个分区的数据量大致均衡，它需要**先知道 key 的分布**，确定每个分区的边界。怎么知道分布？**先对数据抽样**。

`RangePartitioner` 在构造时就会计算分区边界 `rangeBounds`（`Partitioner.scala:198`），里面调用了 `sketch` 方法抽样，而 `sketch` 最后执行的是一个 `.collect()`（`Partitioner.scala:345`）。**`collect()` 就是 Action，所以会提交一个 Job。**

> 面试时如果能说出这个例子，基本可以证明你是真的读过源码，而不是只背过结论。

### 实验 7：`join` → 一个 Stage 有两个父 Stage

```scala
words.map((_, 1)).join(words.map((_, 2)), 2).count()
```

```
Job6: Stage10(4 tasks) | Stage11(4 tasks) | Stage12(2 tasks, parents=10,11)
```

两个 RDD 都需要按 key 重新分区，产生两个 `ShuffleMapStage`，它们**没有依赖关系，可以并行执行**；下游的 `ResultStage` 有两个父 Stage。

**所以 Stage 之间构成的是一个 DAG（有向无环图），不是一条直线。**

> 补充：如果 `join` 的两个 RDD 已经用**同一个分区器**分好区了，就不需要 Shuffle，`join` 是窄依赖，Stage 数会变少。

## 三、追问：Stage 是从前往后切，还是从后往前切？

<figure class="ai-figure"><img src="/bigdata-img/spark/02-job-stage-task-2.webp" alt="阶段从结果端回溯切分再顺序执行" width="960" height="640" loading="lazy" /><figcaption>阶段从结果端回溯切分再顺序执行<span>AI 生成配图</span></figcaption></figure>

**从后往前切，从前往后执行。**

1. **切分**：DAGScheduler 拿到的是 Action 所在的**最后一个 RDD**，先为它创建 `ResultStage`（`createResultStage`，`DAGScheduler.scala:704`），然后沿着依赖**往上游遍历**（`traverseRDDGraph`，`DAGScheduler.scala:740`）：
   - 遇到**窄依赖**：继续往上找，属于同一个 Stage
   - 遇到**宽依赖**：在这里切开，为上游创建一个 `ShuffleMapStage`
2. **提交**：`submitStage`（`DAGScheduler.scala:1540`）先检查有没有缺失的父 Stage，**有就先递归提交父 Stage**，自己进入等待；父 Stage 全部完成后，才提交自己

**为什么要从后往前？**
因为只有最后一个 RDD 是确定的（Action 作用在它上面），从它出发反向遍历，能找出"**为了得到结果，到底需要哪些数据**"，不相关的分支根本不会被访问到。

## 四、满分回答模板

> **基本关系**：
> 一个 Application 包含多个 Job，每次 Action 调用 `sc.runJob` 会提交 Job；DAGScheduler 以宽依赖为边界，把 Job 切分成多个 Stage，最后一个是 ResultStage，其余是 ShuffleMapStage；每个 Stage 中每个需要计算的分区对应一个 Task，由 TaskScheduler 分配到 Executor 上执行。
>
> **切分方式**：
> Stage 是从最后一个 RDD 开始**反向**遍历依赖来切分的，遇到窄依赖合并到当前 Stage，遇到宽依赖就切开；执行时则是先递归提交父 Stage，**从前往后**执行。Stage 之间构成 DAG，比如 join 的下游 Stage 会有两个可以并行的父 Stage。
>
> **几个细节**：
> 1. 一个 Action 不一定只产生一个 Job：`take` 可能分多次提交 Job；`sortByKey` 虽然是转换算子，但 `RangePartitioner` 抽样时会调用 `collect`，也会触发 Job。
> 2. Task 数等于**需要计算的分区数**，不一定等于 RDD 的分区数，比如 `take(1)` 可能只启动 1 个 Task。
> 3. 如果上游 Shuffle 的输出已经存在，对应的 Stage 会被**跳过**（UI 上显示 skipped），这相当于一种隐式缓存。

## 五、速记卡

```
┌─────────────────────────────────────────────┐
│     Job / Stage / Task 速记卡（Spark 4.2.0）    │
├─────────────────────────────────────────────┤
│ Job   ← Action 调用 sc.runJob               │
│ Stage ← 宽依赖切分（从后往前切，从前往后跑）  │
│ Task  ← 每个"需要计算的分区"一个             │
├─────────────────────────────────────────────┤
│ ⭐ 一个 Action ≠ 一定一个 Job                │
│    take 可能多次提交；sortByKey 抽样会触发    │
│ ⭐ Task 数 ≠ 一定等于分区数                  │
│    take(1) 只算 1 个分区                     │
│ ⭐ Shuffle 输出已存在 → Stage skipped        │
│ ⭐ Stage 之间是 DAG，join 有两个父 Stage      │
└─────────────────────────────────────────────┘
```

---

**思考题**：

在 Spark SQL 里执行一条带 `JOIN` 和 `GROUP BY` 的 SQL，在 Spark UI 上经常能看到**多个 Job**，比你预想的要多。这是为什么？（提示：和 AQE 自适应查询执行、广播 Join 有关）


> 本文实验代码已放在 GitHub：https://github.com/DaemonforY/spark-source-notes，复制到 `spark-shell` 里就能复现。

—— X老师读源码
:::
