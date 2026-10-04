---
title: "第 7 讲：容错机制"
description: "容错机制：Task 重试、Executor 丢失、FetchFailed 与 Stage 重算、Checkpoint。"
bigdata: "spark"
---

# 第 7 讲：容错机制

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 配置默认值、类名、行号均在 4.2.0 源码中核实；关键结论经过实验验证（仓库 `experiments/08-fault-tolerance/`，含 `local` 和 `local-cluster` 两种模式）。

## 0. 本讲要回答的问题

前几讲零散地提到过很多容错机制，这一讲把它们串起来：

1. 分布式计算里会出哪些故障？Spark 分别怎么应对？
2. Task 失败了，重试几次？失败时的累加器更新算不算数？
3. Executor 挂了，它上面的 Shuffle 输出怎么办？Spark 一定会马上发现吗？
4. 为什么有的 Stage 重算会导致整个 Job 失败？
5. Checkpoint 和 cache 有什么区别？什么时候用 Checkpoint？
6. Driver 挂了怎么办？

---

## 1. 全景：故障类型与应对

| 故障 | 谁发现 | 怎么处理 | 上限 | 详见 |
|---|---|---|---|---|
| **Task 抛异常** | TaskSetManager | 重新调度这个 Task | `spark.task.maxFailures`，默认 4（`local[N]` 为 1） | 第 2 讲、本讲 2 节 |
| **Task 特别慢** | TaskSetManager | 推测执行：再启动一个副本 | 默认不开启 | 第 2 讲 |
| **某节点反复失败** | HealthTracker | 暂时排除这个节点 / Executor | 默认不开启 | 第 2 讲 |
| **Shuffle 数据丢失** | 下游 Task 报 FetchFailed | DAGScheduler 重算上游 Stage 的**丢失分区** | `spark.stage.maxConsecutiveAttempts`，默认 4 | 第 2 讲、本讲 3 节 |
| **Executor 丢失** | 心跳超时 / 集群管理器通知 | 重新调度它上面的 Task；注销它的 Shuffle 输出 | — | 第 6 讲、本讲 3 节 |
| **缓存块丢失** | 读缓存时发现没有 | 按**血缘**重算这个分区 | — | 第 5 讲 |
| **非确定性 Stage 被重算** | DAGScheduler | 回滚下游 Stage 全部重跑；回滚不了就让 Job 失败 | — | 本讲 4 节 |
| **Driver 失败** | 集群管理器 | 重启整个应用（如果配置了） | YARN `spark.yarn.maxAppAttempts` 等 | 本讲 6 节 |

**一切的基础是血缘（Lineage）**：RDD 记得自己是怎么从父 RDD 算出来的，所以任何一个分区丢了，都可以沿着依赖往上找到还存在的数据，重新算出来（第 1 讲）。窄依赖只需要重算一个父分区；宽依赖则要靠 Shuffle 输出，Shuffle 输出也丢了，就要重算上游 Stage。

---

## 2. Task 级别：重试

### 2.1 实验 A：重试几次

每个 Task 的前 2 次尝试都故意抛异常，共 2 个 Task：

| master | 失败次数 | 成功次数 | 结果 |
|---|---|---|---|
| `local[2]` | 2 | 0 | **Job 失败**（每个 Task 失败 1 次就放弃） |
| `local[2,3]` | 4 | 2 | **成功**（每个 Task 失败 2 次，第 3 次成功） |

这验证了第 2 讲的结论：`local[N]` 模式下 `MAX_LOCAL_TASK_FAILURES = 1`，**根本不会重试**。想在本地测试重试，要用 `local[N,F]`，F 是最大失败次数。集群模式下默认是 `spark.task.maxFailures = 4`。

### 2.2 失败尝试的累加器更新会被丢弃

同一个实验里，每处理一个元素就给累加器加 1：

```
local[2,3]：累加器 = 4
```

4 个元素，正好是**最后成功的那 2 次尝试**处理的数量，前面 4 次失败尝试的累加全部被丢弃了。

> ⚠️ 但这不代表累加器在任何情况下都精确。如果累加器是在**转换算子**（如 `map`）里更新的，而某个**已经成功**的 Task 因为 Stage 重算、推测执行等原因又被执行了一次，累加就可能被重复计入。只有在 **Action**（如 `foreach`）里更新的累加器，Spark 才保证每个 Task 的更新只算一次（官方文档 `docs/rdd-programming-guide.md:1541`）。第 5 讲的"缓存可见性"（`spark.rdd.cache.visibilityTracking.enabled`）也是为了解决类似的问题。

---

## 3. Executor 丢失：Shuffle 输出怎么办

### 3.1 实验 B：杀掉一个 Executor

在 `local-cluster[2,1,1024]`（2 个独立的 Executor 进程）上：

```
[1] 第一次 count：
    Stage 0（attempt 0）: map，执行了 4 个 Task
    Stage 1（attempt 0）: count，执行了 2 个 Task
[2] 杀掉 Executor 0，它上面的 Shuffle 输出随之丢失
[3] 第二次 count：
    Stage 3（attempt 0）: count，执行了 2 个 Task     ← reduce 先跑，读 Shuffle 时报 FetchFailed
    Stage 2（attempt 1）: map，执行了 1 个 Task      ← 只重算丢失的分区
    Stage 3（attempt 1）: count，执行了 2 个 Task     ← reduce 重试
```

两个关键点：

1. **只重算丢失的分区**：map Stage 原本有 4 个 Task，重算时只跑了丢失的那部分（这次是 1 个，另一次运行是 3 个，取决于被杀的 Executor 上恰好有几个 map 输出）。没丢的输出被直接复用
2. **顺序是"先读、后发现、再重算"**：reduce Stage 先启动，读数据时才发现缺失（FetchFailed），然后才回头重算 map Stage

### 3.2 为什么没有马上发现？两条发现路径

我最初以为：Executor 一被杀，Driver 就会立刻注销它的 Shuffle 输出，第二次 count 应该**先**重算 map Stage。实验结果和预期不符，于是去读源码。

DAGScheduler 这边：只要收到 `ExecutorLost`，**一定会**注销 Shuffle 输出（`DAGScheduler.handleExecutorLost`，`DAGScheduler.scala:3079`）：

```scala
// 没有外部 Shuffle 服务时，fileLost = true
val fileLost = !sc.shuffleDriverComponents.supportsReliableStorage() &&
  (workerHost.isDefined || !env.blockManager.externalShuffleServiceEnabled)
```

但问题在上游。`TaskSchedulerImpl.executorLost`（`TaskSchedulerImpl.scala:1006`）：

```scala
if (executorIdToRunningTaskIds.contains(executorId)) {   // 这个 Executor 上还有正在运行的 Task？
  ...
  failedExecutor = Some(executorId)
}
...
if (failedExecutor.isDefined) {
  dagScheduler.executorLost(failedExecutor.get, reason)   // 只有这时才通知 DAGScheduler
}
```

**只有当丢失的 Executor 上还有正在运行的 Task 时，才会通知 DAGScheduler。** 实验里被杀的 Executor 是空闲的，所以 DAGScheduler 并不知道，它的 Shuffle 输出也没被注销。

于是存在两条路径：

| 情况 | 发现方式 | 处理 |
|---|---|---|
| Executor 丢失时**有 Task 在运行** | **主动**：立即通知 DAGScheduler | 注销它的 Shuffle 输出；它上面的 Task 重新调度 |
| Executor 丢失时**是空闲的** | **被动**：等下游读 Shuffle 报 FetchFailed | FetchFailed 处理流程（第 2 讲）：注销丢失的输出，重算上游 Stage |

两条路径的**最终结果一样**：只重算丢失的部分。区别只在于发现的时机，被动发现会多浪费一轮下游 Task 的尝试。

> 这正是"实验先于结论"的价值：只读一处源码，很容易得出"Executor 丢失时 Shuffle 输出会立即被注销"的结论，实际情况多了一个前提条件。

### 3.3 外部 Shuffle 服务的作用

如果开启了外部 Shuffle 服务（`spark.shuffle.service.enabled=true`），Shuffle 文件由节点上独立的服务提供，Executor 丢失**不会**导致 Shuffle 输出丢失（上面代码里 `fileLost = false`，除非整个 Worker 节点都丢了）。这也是第 6 讲动态资源分配需要它的原因。

---

## 4. 非确定性 Stage：一个隐蔽的正确性问题

### 4.1 问题

RDD 的输出有三种"确定性等级"（`RDD.scala:2205` 起的注释，`DeterministicLevel`）：

| 等级 | 含义 |
|---|---|
| `DETERMINATE` | 重算后，数据和顺序都完全一样 |
| `UNORDERED` | 重算后，数据一样，但顺序可能不同 |
| `INDETERMINATE` | 重算后，**数据本身都可能不同** |

**为什么会出问题**：设想一个 Shuffle Map Stage 的输出是非确定的，比如 `repartition` 按轮询把数据分给下游分区，而它的输入顺序每次可能不同。

1. 下游 Stage 的分区 0、1 已经读了**第一次**计算的结果，算完了
2. 上游的某个分区丢了，被**重算**，这次分给下游的数据**变了**
3. 下游分区 2、3 读到的是**第二次**的结果

下游各分区读到的是"两个版本"的上游数据拼起来的，结果可能**少数据或重复数据**，而且不会报错。

### 4.2 Spark 的处理：回滚，回滚不了就失败

`DAGScheduler.rollbackSucceedingStages`（`DAGScheduler.scala:1989`）的注释：

> If a map stage is non-deterministic, the map tasks of the stage may return different result when re-try. To make sure data correctness, we need to clean up shuffles to make sure succeeding stages will be resubmitted and re-try all the tasks...

也就是：**非确定的上游 Stage 被重算时，下游 Stage 要全部重跑**，不能只重跑一部分。

但有一种情况回滚不了（`filterAndAbortUnrollbackableStages`，`DAGScheduler.scala:2702` 起）：

```scala
case resultStage: ResultStage if resultStage.activeJob.isDefined =>
  if (numMissingPartitions < resultStage.numTasks) {
    // ResultStage 已经有分区完成了（结果可能已经返回给用户或写出去了）
    abortStage(resultStage, generateErrorMessage(resultStage), None)
  }
```

**ResultStage 已经有部分 Task 完成**时，结果可能已经交给用户或者写到外部了，无法撤回，Spark 只能**让 Job 失败**，错误信息是：

> A shuffle map stage with indeterminate output was failed and retried. However, Spark cannot rollback the ... to re-process the input data, and has to fail this job. Please eliminate the indeterminacy by **checkpointing the RDD before repartition** and try again.

### 4.3 怎么避免

- **RDD**：在非确定的操作（如 `repartition`）之前 **checkpoint**，让它的输入固定下来
- **Spark SQL**：`spark.sql.execution.sortBeforeRepartition` 默认 `true`，会在 `repartition` 之前**插入一次局部排序**，让输出顺序确定（代价是多一次排序）
- 尽量不要在会被重算的代码里使用随机数、当前时间等非确定的值

---

## 5. Checkpoint：切断血缘

### 5.1 为什么需要

- **血缘太长**：迭代算法（如 PageRank、机器学习）每轮都在上一轮的基础上加几个转换，跑 100 轮血缘就有几百层。一旦早期的数据丢了，重算代价极高；血缘过长还可能让任务序列化、DAG 分析本身都变慢
- **非确定的上游**：见第 4 节
- **昂贵的宽依赖**：上游有复杂的 Join，不希望 Executor 丢失后重算

### 5.2 两种 Checkpoint

| | 可靠 Checkpoint | 本地 Checkpoint |
|---|---|---|
| API | `rdd.checkpoint()` | `rdd.localCheckpoint()` |
| 存在哪 | `sc.setCheckpointDir(...)` 指定的目录，**生产上应该是 HDFS 等可靠存储** | Executor 本地（借用缓存层） |
| Executor 丢失后 | 数据还在 | **数据丢失，而且血缘已经切断了，Job 无法恢复** |
| 适合 | 需要真正容错 | 只想截断血缘、追求速度（如 GraphX 定期截断） |

> 源码注释（`RDD.scala:1700` 起）特别提醒：本地 Checkpoint **不能和动态资源分配一起安全使用**，因为动态分配会移除 Executor，连同上面的数据一起。如果必须一起用，要把 `spark.dynamicAllocation.cachedExecutorIdleTimeout` 设得很大。

### 5.3 实验 C：Checkpoint 做了什么

一个有 50 次 `map` 的 RDD：

```
[直接 checkpoint]
  血缘长度：52 层 → checkpoint 后 2 层，父依赖变成 ReliableCheckpointRDD
  一次 count() 触发了 2 个 Job；源头数据被计算了 200 次（共 100 条）
[先 cache 再 checkpoint]
  血缘长度：52 层 → checkpoint 后 3 层，父依赖变成 ReliableCheckpointRDD
  一次 count() 触发了 2 个 Job；源头数据被计算了 100 次（共 100 条）
```

1. **血缘被切断**：52 层变成 2 层，父依赖直接变成了从文件读数据的 `ReliableCheckpointRDD`
2. **一次 Action 触发了 2 个 Job**：Checkpoint 不是在 Action 计算的同时顺便写出的，而是在 Action **结束后**（`SparkContext.runJob` 的最后一行 `rdd.doCheckpoint()`，`SparkContext.scala:2498`）**再提交一个 Job** 把数据写到 Checkpoint 目录
3. **所以要先 cache**：不 cache，第二个 Job 要把整个血缘**重新计算一遍**（源头数据算了 200 次）；先 cache，第二个 Job 直接读缓存（100 次）。源码注释原话：*"It is strongly recommended that this RDD is persisted in memory, otherwise saving it on a file will require recomputation."*

### 5.4 RDD 与 Dataset 的区别

| | RDD | Dataset |
|---|---|---|
| 默认时机 | **惰性**：第一次 Action 结束后才写 | **立即**：`checkpoint()` 默认 `eager = true`，调用时就执行 |
| 切断的是 | RDD 血缘 | **逻辑计划**（源码注释：迭代算法中逻辑计划可能"指数级增长"） |

> ⚠️ RDD 的 checkpoint 是在 Action **之后**才写，源码注释还提醒：写出的数据可能和 Action 用到的数据**不完全一样**（因为非确定性和重试）。如果需要确定的快照，应该先 cache，再用一个 Action 显式触发。

### 5.5 cache 和 checkpoint 怎么选

| | cache / persist | checkpoint |
|---|---|---|
| 血缘 | **保留** | **切断** |
| 存在哪 | Executor 的内存 / 磁盘 | 可靠存储（HDFS 等） |
| Executor 丢失后 | 丢失，按血缘重算 | 不受影响 |
| 应用结束后 | 删除 | 保留（需要自己清理；或开启 `spark.cleaner.referenceTracking.cleanCheckpoints`，默认 `false`，让 RDD 不再被引用时自动删除） |
| 代价 | 小 | 写可靠存储的 I/O，外加一个额外的 Job |

**常见组合**：`rdd.cache(); rdd.checkpoint(); rdd.count()`：先缓存避免重算，再 checkpoint 切断血缘，最后用一个 Action 触发。

---

## 6. Driver 容错

Driver 保存着整个应用的调度状态（DAGScheduler、TaskScheduler、BlockManagerMaster 的元数据），**Driver 挂了，所有内存中的状态都会丢失**，一般只能**重新运行整个应用**。各集群管理器提供的是"自动重启"：

| 集群管理器 | 机制 |
|---|---|
| YARN（cluster 模式） | ApplicationMaster 失败后重新尝试，次数由 `spark.yarn.maxAppAttempts` 控制（未设置时使用 YARN 自身的 AM 最大尝试次数配置） |
| Standalone（cluster 模式） | 提交时加 `--supervise`，Driver 异常退出后自动重启 |
| Kubernetes | 依赖 Pod 的重启策略或上层的作业控制器 |

> 自动重启只是"从头再跑一次"。**Structured Streaming** 能做到从故障点恢复，靠的是它自己的 checkpoint 机制（记录偏移量和状态），这和本讲的 RDD Checkpoint 是两套东西。

---

## 7. 配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.task.maxFailures` | `4` | 单个 Task 最多失败次数（`local[N]` 固定为 1，`local[N,F]` 为 F） |
| `spark.stage.maxConsecutiveAttempts` | `4` | Stage 因 FetchFailed 最多连续重试次数 |
| `spark.shuffle.service.enabled` | `false` | 外部 Shuffle 服务 |
| `spark.sql.execution.sortBeforeRepartition` | `true` | SQL repartition 前插入局部排序，避免非确定性 |
| `spark.rdd.cache.visibilityTracking.enabled` | `false` | 缓存可见性追踪（第 5 讲） |
| `spark.yarn.maxAppAttempts` | 未设置（用 YARN 的配置） | YARN 上应用的最大尝试次数 |

---

## 自测题

1. 在 `local[4]` 模式下，一个 Task 抛异常会重试吗？如果想测试重试逻辑，master 应该怎么写？
2. 一个 Task 失败了 2 次后成功，它在失败尝试里对累加器的更新会被计入吗？
3. 在 `map` 里更新累加器，结果一定准确吗？为什么？
4. 一个空闲的 Executor 被杀掉，Driver 会立即注销它的 Shuffle 输出吗？最终会怎么发现并处理？
5. 开启外部 Shuffle 服务后，Executor 丢失还会导致 Shuffle 输出丢失吗？
6. 什么是"非确定性 Stage"？它被重算时为什么要让下游 Stage 全部重跑？
7. 什么情况下 Spark 会因为非确定性 Stage 直接让 Job 失败？应该怎么避免？
8. 为什么调用 `rdd.checkpoint()` 之前推荐先 `cache()`？实验中两种做法的计算次数分别是多少？
9. 本地 Checkpoint 和可靠 Checkpoint 有什么区别？本地 Checkpoint 为什么不适合和动态资源分配一起用？
10. Driver 挂了，Spark 能从故障点继续执行吗？

## 下一讲预告

第 8 讲进入 **Spark SQL**：一条 SQL 是怎么变成 RDD 的？Catalyst 优化器的四个阶段（解析、分析、优化、物理计划）分别在做什么？
:::
