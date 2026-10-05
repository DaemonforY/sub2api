---
title: "Flink 2.x 源码精读（六）：调度与容错，一个 Task 失败之后发生了什么"
description: "两个作业，同样让一个 Task 抛异常，一个重启了 4 个 Task，另一个只重启了 1 个。顺着这个现象，读懂 Flink 2.3 的 Failover 全流程、Pipelined Region、默认的指数退避重启策略，以及 AdaptiveScheduler 如何按资源自动扩缩容。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-06-cover.webp"}]]
---

# Flink 2.x 源码精读（六）：调度与容错，一个 Task 失败之后发生了什么

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-06-cover.webp" alt="Yui和Kai在故障调度控制室处理Task失败与恢复" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在故障调度控制室处理Task失败与恢复<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink/runtime`，`CFG` = `flink-core/src/main/java/org/apache/flink/configuration`
> 前置阅读：第一讲下篇（Pipelined Region 调度）、第二讲上篇（Checkpoint）、第三讲上篇（扩缩容时的状态重分配）

---

先看一个实验。

两个作业，并行度都是 2，每 2 秒做一次 Checkpoint。都让 `fail-map` 的子任务 0 处理 3000 条数据后抛一个异常，前两次执行各抛一次。

两个作业只有一处不同：

- **作业 A**：`fail-map` 后面有一个 `keyBy`；
- **作业 B**：没有 `keyBy`，所有算子都链在一起。

本机实测，JobMaster 的日志（节选，省略了日志级别、线程名和 Task 的执行 ID，完整日志在 `assets/logs/failover-keyby.log`、`failover-forward.log`）：

```text
-- 作业 A（keyBy）
18:14:05,965 Task - Source: source -> fail-map (1/2)#0 switched from RUNNING to FAILED
18:14:05,975 JobMaster - 4 tasks will be restarted to recover the failed task ...

-- 作业 B（全部链在一起）
18:14:34,090 Task - Source: source -> fail-map -> count -> Sink: Writer (1/2)#0 switched from RUNNING to FAILED
18:14:34,107 JobMaster - 1 tasks will be restarted to recover the failed task ...
```

**同样是一个 Task 失败，作业 A 重启了 4 个 Task，作业 B 只重启了 1 个。** 作业 B 里另一个子任务 `(2/2)`，从头到尾一直是 `#0`，没有被重启过。

第一讲下篇留过一个问题："一个 Task 失败，为什么有时只重启一部分？"这一篇来回答。我们会看到：

1. 一个 Task 抛出异常后，JobManager 里依次发生了什么？
2. 重启哪些 Task，是怎么决定的？**Pipelined Region** 是什么？
3. 默认的重启策略是什么？为什么两次重启的间隔分别是 1 秒和 1.5 秒？
4. 不开 Checkpoint 会怎样？
5. **AdaptiveScheduler** 怎么根据资源自动扩缩容？扩容为什么要等一个 Checkpoint？

---

## 一、全景：失败之后的四个问题

一个 Task 失败之后，调度器要依次回答四个问题：

| 问题 | 由谁回答 | 本文 |
|---|---|---|
| ① 能不能重启？等多久？ | `RestartBackoffTimeStrategy`（重启策略） | 第三节 |
| ② 重启哪些 Task？ | `FailoverStrategy`（默认按 region） | 第二节 |
| ③ 从哪里恢复状态？ | `CheckpointCoordinator` | 1.3 节 |
| ④ 重新部署 | `SchedulingStrategy` | 第一讲下篇 |

### 1.1 失败怎么报到调度器

```text
Task 线程抛出异常
 └ Task.doRun() 的 catch：状态 RUNNING → FAILED
   └ RPC → JobMaster.updateTaskExecutionState()         RT/jobmaster/JobMaster.java:537
     └ schedulerNG.updateTaskExecutionState(...)         :543
       └ SchedulerBase.updateTaskExecutionState()        RT/scheduler/SchedulerBase.java:808
         └ onTaskExecutionStateUpdate() → onTaskFailed() :813、:843
```

这条路径和第一讲下篇里 Task 报告"运行中"、"已完成"的路径是同一条，只是状态变成了 FAILED。

### 1.2 DefaultScheduler 怎么处理

`RT/scheduler/DefaultScheduler.java`：

```java
protected void onTaskFailed(final Execution execution) { ... }                  // :278
protected void handleTaskFailure(...) {                                          // :289
    maybeRestartTasks(recordTaskFailure(failedExecution, error));
}
protected FailureHandlingResult recordTaskFailure(...) {                         // :294
    ...
    notifyCoordinatorsAboutTaskFailure(failedExecution, error);                  // :298
    return executionFailureHandler.getFailureHandlingResult(...);                // :300
}
```

`ExecutionFailureHandler`（`RT/executiongraph/failover/ExecutionFailureHandler.java`）把问题 ① 和 ② 分开回答：

- 重启哪些 Task：`failoverStrategy.getTasksNeedingRestart(...)`（`:122`）
- 能不能重启：`handleFailure()`（`:177-222`）

```java
if (isUnrecoverableError(cause)) {                                    // :187
    return FailureHandlingResult.unrecoverable(...,
            new JobException("The failure is not recoverable", cause), ...);
}
boolean isNewAttempt = restartBackoffTimeStrategy.notifyFailure(cause);   // :197
if (restartBackoffTimeStrategy.canRestart()) {                            // :198
    return FailureHandlingResult.restartable(..., verticesToRestart,
            restartBackoffTimeStrategy.getBackoffTime(), ...);            // :209
} else {
    return FailureHandlingResult.unrecoverable(...,
            new JobException("Recovery is suppressed by " + restartBackoffTimeStrategy, cause), ...);  // :216
}
```

两个常见的报错信息都出自这里：

- **`The failure is not recoverable`**：异常链上有被 `@ThrowableAnnotation(ThrowableType.NonRecoverableError)` 标注的异常（`isUnrecoverableError`，`:224-229`）。这类错误重启多少次都没用，直接让作业失败。
- **`Recovery is suppressed by ...`**：重启策略不允许再重启了。第四节会看到它。

### 1.3 重启：取消 → 等待 → 恢复状态 → 重新部署

`restartTasksWithDelay()`（`DefaultScheduler.java:358-395`）：

```java
// ① 给每个要重启的 vertex 记一个版本号
executionVertexVersioner.recordVertexModifications(verticesToRestart)
log.info("{} tasks will be restarted to recover the failed task {}.", ...);   // :376 开头实验的那行日志
addVerticesToRestartPending(verticesToRestart);                               // 作业状态 → RESTARTING
cancelTasksAsync(verticesToRestart);                                          // ② :383 取消
delayExecutor.schedule(() -> ... restartTasks(...), restartDelayMS, ...);     // ③ :387 等退避时间
```

`restartTasks()`（`:415`）：

```java
verticesToRestart = executionVertexVersioner.getUnmodifiedExecutionVertices(...);  // :419 版本号检查
resetForNewExecutions(verticesToRestart);      // ④ :427 新建 Execution，attemptNumber + 1
restoreState(verticesToRestart, ...);          // ⑤ :430 恢复状态
schedulingStrategy.restartTasks(verticesToRestart);  // ⑥ :436 重新部署
```

**版本号有什么用？** 在等待退避的这段时间里，可能又有新的失败（比如下游因为上游挂了也报错），触发新一轮重启。版本号保证旧的重启计划不会覆盖新的：某个 vertex 在等待期间被别的 failover 改过，旧计划就跳过它。

**`#0`、`#1`、`#2` 就是 attemptNumber**，线程名里直接能看到。开头实验里的 `(1/2)#0` 失败后，重启出来的就是 `(1/2)#1`。

### 1.4 恢复状态

`SchedulerBase.restoreState()`（`RT/scheduler/SchedulerBase.java:462`）：

1. 作废所有正在进行的 Checkpoint（`:489`，原因是 `JOB_FAILOVER_REGION`）；
2. 局部恢复时调用 `restoreLatestCheckpointedStateToSubtasks`（`:502`），全局恢复时调用 `restoreLatestCheckpointedStateToAll`（`:495`）；
3. 最终到 `CheckpointCoordinator.restoreLatestCheckpointedStateInternal()`（`RT/checkpoint/CheckpointCoordinator.java:1745`）：取出最近一次**完成**的 Checkpoint（`:1768`），打印 `Restoring job {} from {}.`（`:1798`），用 `StateAssignmentOperation` 把状态分配给各个 subtask（`:1812-1819`）。

`:502` 后面有一段注释很有意思：CheckpointCoordinator 并不知道这是一次"局部"恢复，它总是把状态分配给**所有** subtask，只不过对还在运行的那些 subtask 没有效果。所以通知 OperatorCoordinator "某个 subtask 被重置了"这件事，只能由调度器来做。

对 Source 来说，这个通知最终到 `SourceCoordinator.subtaskReset()`（`RT/source/coordinator/SourceCoordinator.java:372-393`）：把那个 subtask 在上一个 Checkpoint 之后才分到的 split 收回来，还给 SplitEnumerator（`:389`，`addSplitsBack`）。**所以局部重启时，split 不会丢，也不会被两个 subtask 同时读。**

> **恢复的是最近一次完成的 Checkpoint，所以从那个 Checkpoint 到失败之间处理过的数据，会被重新处理一遍。** 开头实验里，`count` 恢复出来的计数（比如 `count=1786`）就是 Checkpoint 里的值，而不是失败那一刻的值。要做到对外精确一次，还需要两阶段提交的 Sink（第二讲上篇、踩坑实验室 #06）。

---

## 二、Pipelined Region：重启的最小单位

### 2.1 什么是 region

**用 pipelined 边连起来的 Task，组成一个 region。** pipelined 边的意思是：数据在内存里直接流给下游，不落盘。所以上下游必须同时运行，一个挂了，另一个也没法继续。

批作业里常见的 blocking 边（上游把数据全部写到磁盘，下游再读）会把 region 切开。

计算在 `SchedulingPipelinedRegionComputeUtil.computePipelinedRegions()`（`RT/executiongraph/failover/SchedulingPipelinedRegionComputeUtil.java:46-62`）：先把必须以流水线方式消费的边（`mustBePipelinedConsumed()`）连起来的 vertex 合并成初始的 region，再用 Tarjan 算法合并成环的 region（`mergeRegionsOnCycles`，`:69`）。ExecutionGraph 构建时就算好了（`RT/scheduler/adapter/DefaultExecutionTopology.java:401`）。

### 2.2 流作业里，region 由连接方式决定

**流作业里所有的边都是 pipelined 的**，所以 region 怎么划分，完全取决于上下游怎么连：

```text
作业 A：有 keyBy（all-to-all）                    作业 B：全部 forward（一对一）

 Source(1) ──┬──► Count(1)                        Source(1) → Map(1) → Sink(1)   ← region 1
             ╳                                    Source(2) → Map(2) → Sink(2)   ← region 2
 Source(2) ──┴──► Count(2)

 所有 Task 互相连通 → 只有 1 个 region              每个 subtask 各自独立 → 2 个 region
 任何一个失败 → 4 个 Task 全部重启                  subtask 1 失败 → 只重启 region 1
```



这就是开头实验的答案。

**现实中的影响**：大多数流作业都有 `keyBy`、`rebalance` 这类 all-to-all 的边，所以**整个作业就是一个 region，一个 Task 失败，所有 Task 都要重启**。只有每一步都是一对一连接的作业（比如并行度一致的 Kafka → map/filter → Kafka），才能享受只重启一部分的好处。

### 2.3 RestartPipelinedRegionFailoverStrategy

默认的 failover 策略是 `region`（`jobmanager.execution.failover-strategy`，`CFG/JobManagerOptions.java:284`），另一个可选值是 `full`（每次重启全部）。

`getTasksNeedingRestart()`（`RT/executiongraph/failover/RestartPipelinedRegionFailoverStrategy.java:110-150`）调用 `getRegionsToRestart()`（`:158`）。按注释，要重启的 region 由三条规则决定：

1. 失败的 Task 所在的 region，一定要重启；
2. 某个要重启的 region 需要读的上游数据**已经不可用了**（比如异常是 `PartitionException`），产生这份数据的上游 region 也要重启；
3. 一个 region 要重启，它的下游 region 也要重启（它们读的数据要重新产生）。

最后，还处于 `CREATED` 状态（从来没部署过）的 Task 不用重启（`:136`）。

### 2.4 实测对比

`FailoverDemo keyby` 和 `FailoverDemo forward` 的输出（本机实测，节选，省略了日志级别、线程名和 ID，完整日志在 `assets/logs/failover-keyby.log`、`failover-forward.log`）：

```text
-- 作业 A（keyby）
18:14:05,965 Task - Source: source -> fail-map (1/2)#0 switched from RUNNING to FAILED
18:14:05,975 JobMaster - 4 tasks will be restarted to recover the failed task ...
18:14:07,029 CheckpointCoordinator - Restoring job ... from Checkpoint 2.
>>> [after-keyby -> count -> Sink: Writer (1/2)#1] attempt=1 initializeState: restored=true count=1786
>>> [after-keyby -> count -> Sink: Writer (2/2)#1] attempt=1 initializeState: restored=true count=2300
18:14:10,046 Task - Source: source -> fail-map (1/2)#1 switched from RUNNING to FAILED
18:14:10,048 JobMaster - 4 tasks will be restarted to recover the failed task ...
18:14:11,569 CheckpointCoordinator - Restoring job ... from Checkpoint 3.
>>> [after-keyby -> count -> Sink: Writer (1/2)#2] attempt=2 initializeState: restored=true count=3550
>>> [after-keyby -> count -> Sink: Writer (2/2)#2] attempt=2 initializeState: restored=true count=4566

-- 作业 B（forward）
18:14:34,090 Task - Source: source -> fail-map -> count -> Sink: Writer (1/2)#0 switched from RUNNING to FAILED
18:14:34,107 JobMaster - 1 tasks will be restarted to recover the failed task ...
18:14:35,119 CheckpointCoordinator - Restoring job ... from Checkpoint 3.
>>> [Source: source -> fail-map -> count -> Sink: Writer (1/2)#1] attempt=1 initializeState: restored=true count=2032
18:14:38,134 Task - Source: source -> fail-map -> count -> Sink: Writer (1/2)#1 switched from RUNNING to FAILED
18:14:38,137 JobMaster - 1 tasks will be restarted to recover the failed task ...
18:14:39,640 CheckpointCoordinator - Restoring job ... from Checkpoint 5.
>>> [Source: source -> fail-map -> count -> Sink: Writer (1/2)#2] attempt=2 initializeState: restored=true count=4045
```

> 为了便于阅读，日志中的作业 ID、ExecutionAttemptID 和 Checkpoint 的路径信息用 `...` 代替了，没有改动其他内容。

| | 作业 A（keyby） | 作业 B（forward） |
|---|---|---|
| region 个数 | 1 | 2 |
| 每次重启的 Task 数 | **4** | **1** |
| 没出错的 `count (2/2)` | `#0` → `#1` → `#2`，跟着重启了两次 | 一直是 `#0`，没有重新初始化 |
| 第 1 次：失败 → 开始恢复 | 1.064 秒 | 1.029 秒 |
| 第 2 次：失败 → 开始恢复 | 1.523 秒 | 1.506 秒 |

最后两行的间隔，是下一节的内容。

---

## 三、重启策略：默认是指数退避

### 3.1 默认值

重启策略由 `RestartBackoffTimeStrategyFactoryLoader` 加载（`RT/executiongraph/failover/RestartBackoffTimeStrategyFactoryLoader.java`）：

1. 先看作业配置里的 `restart-strategy.type`，再看集群配置（`:62-64`）；
2. 都没配，用默认值（`getDefaultRestartStrategyFactory`，`:94-121`）：
   - **开了 Checkpoint → `exponential-delay`（指数退避）**（`:97`）
   - **没开 Checkpoint → 不重启**（`NoRestartBackoffTimeStrategy`，`:118`）

> 很多资料里写"默认是 fixed-delay，重启 `Integer.MAX_VALUE` 次"，那是很早以前的行为。在 2.3.0 里，开了 Checkpoint 的默认策略是**指数退避**，并且**不限次数**（见下表最后一行）。

默认参数（`CFG/RestartStrategyOptions.java`）：

| 参数 `restart-strategy.exponential-delay.*` | 默认值 | 行号 |
|---|---|---|
| `initial-backoff` | 1 秒 | `:229` |
| `max-backoff` | 1 分钟 | `:242` |
| `backoff-multiplier` | 1.5 | `:257` |
| `reset-backoff-threshold` | 1 小时 | `:273` |
| `jitter-factor` | 0.1 | `:288` |
| `attempts-before-reset-backoff` | `Integer.MAX_VALUE` | `:303` |

### 3.2 退避时间怎么算

`ExponentialDelayRestartBackoffTimeStrategy`（`RT/executiongraph/failover/ExponentialDelayRestartBackoffTimeStrategy.java`）：

```java
public boolean notifyFailure(Throwable cause) {                 // :112
    long now = clock.absoluteTimeMillis();
    // Merge multiple failures into one attempt if there are tasks will be restarted later.
    if (now <= nextRestartTimestamp) {
        return false;
    }
    if ((now - nextRestartTimestamp) >= resetBackoffThresholdMS) {
        setInitialBackoff();                                    // 距上次重启超过 1 小时：重新从 1 秒开始
    }
    nextRestartTimestamp = now + calculateActualBackoffTime();
    currentRestartAttempt++;
    return true;
}

private long calculateActualBackoffTime() {                     // :183
    long currentBackoffTime =
            (long) (initialBackoffMS * Math.pow(backoffMultiplier, currentRestartAttempt));
    return Math.max(
            initialBackoffMS,
            Math.min(
                    currentBackoffTime + calculateJitterBackoffMS(currentBackoffTime),
                    maxBackoffMS));
}
```

按默认参数算：

| 第几次重启 | 基础时间 | 加上 ±10% 抖动 | 最后取 `max(1 秒, ...)` | 实测（失败 → 开始恢复） |
|---|---|---|---|---|
| 1 | 1000 × 1.5⁰ = 1000ms | 900～1100ms | **1000～1100ms** | 1.064 秒、1.029 秒 |
| 2 | 1000 × 1.5¹ = 1500ms | 1350～1650ms | 1350～1650ms | 1.523 秒、1.506 秒 |
| …… | …… | …… | 最多 1 分钟 | |

> 实测值是从"Task 失败"这行日志到"Restoring job"这行日志的间隔，除了退避时间，还包含取消 Task 等几毫秒的开销，所以只能说和计算结果吻合，不是精确的退避时间。

注意第一行：因为外面套了一层 `Math.max(initialBackoffMS, ...)`，**往下的抖动会被截掉，第一次重启至少等 1 秒**，实际在 1.0～1.1 秒之间。

三个细节：

1. **退避期内的失败会合并**（`return false`）：一台 TaskManager 宕机，可能有几十个 Task 同时失败，它们只算一次重启，退避时间也只增长一次。这就是 1.2 节里 `isNewAttempt` 的含义。
2. **抖动**：很多作业依赖同一个出了故障的外部系统（比如同一个 Kafka 集群）时，避免它们在同一时刻一起重启，把外部系统再打垮一次。
3. **长时间稳定后重置**：距离上次重启超过 1 小时，退避时间回到 1 秒。

### 3.3 全局失败

不属于某个具体 Task 的失败（比如 Checkpoint 连续失败超过 `tolerable-failed-checkpoints`，第二讲上篇；或者 OperatorCoordinator 出错），走的是 `handleGlobalFailure()`（`DefaultScheduler.java:337`）。全局失败**总是重启所有 Task**（`getGlobalFailureHandlingResult`，`ExecutionFailureHandler.java:135-145`，把拓扑中所有 vertex 都放进重启列表），也就用不上 region 了。

---

## 四、不开 Checkpoint：一次失败，作业就结束了

把作业 A 的 `enableCheckpointing` 去掉（`FailoverDemo keyby nockpt`），其他不变。本机实测：

```text
18:13:45,069 JobMaster - Using restart back off time strategy NoRestartBackoffTimeStrategy for failover-demo-keyby (...).
...
>>> [Source: source -> fail-map (1/2)#0] attempt=0: throwing an exception on purpose
>>> cause: ExecutionException: org.apache.flink.runtime.client.JobExecutionException: Job execution failed.
>>> cause: JobExecutionException: Job execution failed.
>>> cause: JobException: Recovery is suppressed by NoRestartBackoffTimeStrategy
>>> cause: RuntimeException: boom (attempt 0)
```

> `>>> cause` 是示例逐层打印的异常链，作业 ID 用 `...` 代替。完整日志在 `assets/logs/failover-keyby-nockpt.log`。

作业启动时，日志就已经说明了用的是 `NoRestartBackoffTimeStrategy`。第一次失败后，异常链上出现了 `Recovery is suppressed by NoRestartBackoffTimeStrategy`，作业直接失败，没有任何重试。

这对**本地调试**影响最大：很多人在 IDE 里写 demo 时不开 Checkpoint，代码里一个偶发的异常就会让作业直接退出。想要失败后自动重启，要么开启 Checkpoint，要么显式配置 `restart-strategy.type`。

---

## 五、AdaptiveScheduler：拿到多少资源就用多少

### 5.1 三种调度器

| 调度器 | 适用场景 | 特点 | 启用方式 |
|---|---|---|---|
| `DefaultScheduler` | 流作业默认 | 并行度固定；按 region 调度和重启 | 默认 |
| `AdaptiveScheduler` | 流作业，资源会变化 | 拿到多少资源就用多少，资源变化时自动扩缩容 | `jobmanager.scheduler: adaptive`（`CFG/JobManagerOptions.java:518`）或 `scheduler-mode: reactive`（`:561`） |
| `AdaptiveBatchScheduler` | 批作业默认 | 按上游实际产出的数据量决定下游并行度 | 批作业自动选择 |

选择逻辑在 `DefaultSlotPoolServiceSchedulerFactory.getSchedulerType()`（`RT/jobmaster/DefaultSlotPoolServiceSchedulerFactory.java:236`）。批作业即使配置了 adaptive，也会被改成 AdaptiveBatch（`:245` 的日志 `Changing scheduler type to 'AdaptiveBatch'`）。

`DefaultScheduler` 的并行度是固定的：作业要 4 个 slot，就得凑齐 4 个。在 Kubernetes 这类资源会伸缩的环境里，我们更希望：资源不够时先跑起来；资源多了自动扩容；节点被回收时自动缩容，继续跑。这就是 AdaptiveScheduler（FLIP-160）要解决的问题，Reactive Mode（FLIP-159）是在它基础上"总是用满集群资源"的模式。

### 5.2 状态机

`RT/scheduler/adaptive/` 下，每个状态是一个类：

```text
Created → WaitingForResources → CreatingExecutionGraph → Executing → Finished / Canceling / Failing ...
                ▲                       ▲                    │
                │                       └──── Restarting ◄───┤ 失败、或者要扩缩容
                └──────── 资源不够 ─────────────────────────────┘
```

所有状态迁移都经过 `transitionToState()`（`RT/scheduler/adaptive/AdaptiveScheduler.java:1735`），它会打一行 DEBUG 日志 `Transition from state {} to {}.`（`:1747`）。

### 5.3 实测：声明并行度 4，集群只有 2 个 slot

`AdaptiveSchedulerDemo`：作业声明并行度 4，但集群一开始只有 1 个 TaskManager（2 个 slot）。第 15 秒加入第二个 TaskManager，第 35 秒停掉一个。为了便于观察，扩缩容后的冷却时间调成了 5 秒（默认 30 秒），运行中的资源稳定等待时间调成了 3 秒（默认 60 秒），每 2 秒做一次 Checkpoint。

本机实测（节选，省略了日志级别、线程名和 ID，完整日志在 `assets/logs/adaptive-scheduler.log`）：

```text
18:14:58,973 Transition from state Created to WaitingForResources.
>>> t=2s: CREATED map=4, Source: source=4                                ← 声明的并行度，还没开始跑（t=5s～10s 相同）
18:15:09,065 Transition from state WaitingForResources to CreatingExecutionGraph.   ← 等了 10 秒
18:15:09,110 Transition from state CreatingExecutionGraph to Executing.
>>> t=12s: RUNNING Source: source=2, map=2                               ← 凑不齐 4 个，先用 2 个跑

>>> t=15s: starting a 2nd TaskManager (+2 slots)
18:15:15,195 CheckpointCoordinator - Completed checkpoint 4 ...
18:15:15,199 Transition from state Executing to Restarting.
18:15:15,211 Transition from state Restarting to CreatingExecutionGraph.
18:15:15,213 CheckpointCoordinator - Restoring job ... from Checkpoint 4 ...
18:15:15,219 Transition from state CreatingExecutionGraph to Executing.
>>> t=17s: RUNNING Source: source=4, map=4                               ← 自动扩容到 4

>>> t=35s: terminating TaskManager #0 (-2 slots)
18:15:33,254 CheckpointCoordinator - Completed checkpoint 14 ...         ← TaskManager 被停掉之前，最后一个完成的 Checkpoint
18:15:34,057 Task - map -> Sink: Writer (3/4)#0 ... switched from RUNNING to FAILED
18:15:34,073 Transition from state Executing to Restarting.
18:15:35,088 Transition from state Restarting to WaitingForResources.     ← 1 秒退避
18:15:45,090 Transition from state WaitingForResources to CreatingExecutionGraph.   ← 又等了 10 秒
18:15:45,092 CheckpointCoordinator - Restoring job ... from Checkpoint 14 ...
18:15:45,093 Transition from state CreatingExecutionGraph to Executing.
>>> t=47s: RUNNING Source: source=2, map=2                               ← 自动缩容到 2，继续跑
```

> `>>>` 行是示例每 2.5 秒采样一次的作业状态和各算子的并行度；其他行是 Flink 的日志，去掉了线程名和日志级别。

三段分别来看。

**① 提交：先等 10 秒，再用现有资源跑起来。** 拿不到期望的 4 个 slot，进入 `WaitingForResources`，等资源稳定下来。这个时间是 `jobmanager.adaptive-scheduler.submission.resource-stabilization-timeout`，默认 **10 秒**（`CFG/JobManagerOptions.java:776`）。最长等多久由 `submission.resource-wait-timeout` 决定，默认 5 分钟（`:694`）。

**② 扩容：等到下一个 Checkpoint 完成，马上切换。** 注意这三行的时间：

```text
18:15:15,195 Completed checkpoint 4
18:15:15,199 Transition from state Executing to Restarting.     ← 4 毫秒后
18:15:15,213 Restoring job ... from Checkpoint 4
```

这不是巧合。`Executing` 状态在 **Checkpoint 完成**时触发扩缩容判断（`RT/scheduler/adaptive/Executing.java:280-281`，`onCompletedCheckpoint → triggerPotentialRescale`），然后从这个**刚刚完成的** Checkpoint 恢复。这样，需要重新处理的数据几乎为零。

如果 Checkpoint 一直不成功怎么办？连续失败 2 次（`jobmanager.adaptive-scheduler.rescale-trigger.max-checkpoint-failures`，默认 2）也会触发（`Executing.java:285-290`）；还有一个兜底的最长等待时间 `rescale-trigger.max-delay`，按配置说明，开启 Checkpoint 时默认是 Checkpoint 间隔 ×（上面那个次数 + 1）（`CFG/JobManagerOptions.java:734-746`）。

决定"要不要扩"的是 `DefaultStateTransitionManager`（`RT/scheduler/adaptive/DefaultStateTransitionManager.java`），它自己也是一个状态机（`:45-62` 的注释）：

```text
Cooldown → Idling → Stabilizing → Stabilized → 触发切换
```

- **Cooldown**：刚扩缩容完，先冷却，避免来回抖动（`executing.cooldown-after-rescaling`，默认 30 秒，`CFG/JobManagerOptions.java:589`）；
- **Stabilizing**：资源变了，再观察一会儿（`executing.resource-stabilization-timeout`，默认 60 秒，`:600`）。在这个阶段，**如果已经拿到了期望的全部资源，收到触发信号就立即切换**（`DefaultStateTransitionManager.java:376-405`），不用等满稳定时间；
- **Stabilized**：稳定期结束，只要资源"够用"（能比现在的并行度更高），收到触发信号就切换（`:433-444`）。

本实验里，加入第二个 TaskManager 后正好凑齐了期望的 4 个 slot，所以没有等满 3 秒的稳定时间，下一个 Checkpoint 一完成就扩容了。

> 以上对 Cooldown 与扩容时机的先后关系，是根据代码和这一次的日志推断的，示例没有单独打印 `DefaultStateTransitionManager` 的阶段切换日志。

**③ 缩容：先当作一次失败，再等资源。** 停掉一个 TaskManager，对调度器来说是**一次失败**，不是一次主动的缩容：

1. 上面的 Task 失败，进入 `Restarting`，按重启策略退避约 1 秒；
2. 进入 `WaitingForResources`，又等了 10 秒，看丢掉的资源会不会回来（这里用的也是 `submission.resource-stabilization-timeout`，`AdaptiveScheduler.java:1275-1283`）；
3. 用剩下的 2 个 slot 重新启动，从 **Checkpoint 14** 恢复。

Checkpoint 14 在 `18:15:33,254` 完成，TaskManager 在 `18:15:34,057` 前后被停掉，这之间约 0.8 秒的数据会被重新处理。

| | 扩容 | 缩容 |
|---|---|---|
| 起因 | 新资源到达 | TaskManager 丢失，Task 失败 |
| 什么时候切换 | 下一个 Checkpoint 完成时 | 失败后立即进入 Restarting |
| 从哪个 Checkpoint 恢复 | **刚刚完成的那个** | 失败前最后一个完成的 |
| 这次实验中，作业没有在运行的时间 | 约 20ms（`Executing → Restarting → Executing`） | 约 11 秒（1 秒退避 + 10 秒等待） |

> 最后一行的"约 20ms"只是调度器状态切换的时间，不包括 Task 重新部署、恢复状态的时间。这个作业的状态非常小（546 字节），真实作业的状态越大，恢复越慢。

### 5.4 扩缩容就是一次重启

不管扩还是缩，最后都要经过 `Restarting`：取消所有 Task，按新的并行度重新创建 ExecutionGraph，从 Checkpoint 恢复，状态按 KeyGroup 重新分配（第三讲上篇）。

另外，AdaptiveScheduler 处理失败的方法 `howToHandleFailure()`（`AdaptiveScheduler.java:1654`）同样先检查不可恢复的错误（`:1665`），再问重启策略，用的是同一套 `RestartBackoffTimeStrategy`。但它**每次都重启整个作业**，没有 region 的概念。

---

## 六、AdaptiveBatchScheduler：按数据量决定并行度

批作业的上下游之间通常是 blocking 的：上游全部跑完，把数据写到磁盘，下游才开始。这给了调度器一个机会：**等上游跑完，看它实际产出了多少数据，再决定下游用多大的并行度。**

- 每当有 Task 完成（`onTaskFinished()`，`RT/scheduler/adaptivebatch/AdaptiveBatchScheduler.java:404`），就尝试初始化后续的 vertex；
- 并行度由 `DefaultVertexParallelismAndInputInfosDecider.decideParallelismAndInputInfosForVertex()`（`RT/scheduler/adaptivebatch/DefaultVertexParallelismAndInputInfosDecider.java:103`）决定：大致是输入数据量 ÷ `execution.batch.adaptive.auto-parallelism.avg-data-volume-per-task`（默认 16MB），再限制在 `min-parallelism`（默认 1）和 `max-parallelism`（默认 128）之间（`CFG/BatchExecutionOptions.java:46`、`:62`、`:79`）；
- 推测执行（`execution.batch.speculative.enabled`，默认 false，`CFG/BatchExecutionOptions.java:123`）：发现某个 Task 明显比同批的其他 Task 慢，就在别的机器上再跑一份，谁先跑完用谁的结果。

> 本篇没有对批作业做实验，这一节只是介绍，具体行为请以你自己的测试为准。

---

## 七、自己动手

示例代码在 `flink-notes/demos`：

```bash
./run.sh study.scheduler.FailoverDemo keyby
```

```bash
./run.sh study.scheduler.FailoverDemo forward
```

```bash
./run.sh study.scheduler.FailoverDemo keyby nockpt
```

```bash
./run.sh study.scheduler.AdaptiveSchedulerDemo
```

**推荐的断点**（用 `FailoverDemo keyby`）：

| 断点 | 看什么 |
|---|---|
| `SchedulerBase.java:808` | 失败状态的上报 |
| `RestartPipelinedRegionFailoverStrategy.java:110` | 失败的 region 包含哪些 vertex |
| `ExponentialDelayRestartBackoffTimeStrategy.java:112` | 退避时间、`isNewAttempt` |
| `DefaultScheduler.java:387` | 退避延迟 |
| `DefaultScheduler.java:419` | 版本号检查后剩下哪些 vertex |
| `CheckpointCoordinator.java:1768` | 用哪个 Checkpoint 恢复 |
| `Executing.java:281`（`AdaptiveSchedulerDemo`） | 扩容是不是由 Checkpoint 完成触发的 |

---

## 八、课后练习

1. **不可恢复的错误**：定义一个异常类，加上 `@ThrowableAnnotation(ThrowableType.NonRecoverableError)`，在 `FailingMap` 里抛出它，日志里会出现什么？作业还会重启吗？
2. **重启次数用完**：设置 `restart-strategy.type: fixed-delay`、`restart-strategy.fixed-delay.attempts: 1`，让 `FailingMap` 失败两次，找到 `Recovery is suppressed by` 这一行，它后面打印的策略信息是什么？
3. **full 策略**：在 forward 作业里设置 `jobmanager.execution.failover-strategy: full`，每次重启的 Task 数会变成多少？
4. **扩容与 Checkpoint**：把 `AdaptiveSchedulerDemo` 的 Checkpoint 间隔改成 20 秒，扩容会推迟多久？和 `rescale-trigger.max-delay` 有什么关系？
5. **思考**：为什么 AdaptiveScheduler 不支持按 region 重启？（提示：它每次都按新的并行度重新创建 ExecutionGraph。）

---

## 写在最后

这一讲的核心可以用一句话概括：**一个 Task 失败后，重启策略决定能不能重启、等多久，region 决定重启哪些 Task，Checkpoint 决定从哪里恢复。**

回到开头：作业 A 有 `keyBy`，所有 Task 连成一个 region，所以一个失败就要重启 4 个；作业 B 全是一对一连接，每个 subtask 是一个独立的 region，所以只重启 1 个。

**实际使用时的建议**：

- 有 `keyBy` 的流作业，一个 Task 失败通常意味着整个作业重启，恢复时间主要取决于状态大小；
- 2.x 开了 Checkpoint 的默认重启策略是**指数退避、不限次数**。如果你需要"失败几次就报警、停止"，要显式配置；
- 不开 Checkpoint 时，默认不重启，本地调试时要注意；
- 用 AdaptiveScheduler 时，扩容会等下一个 Checkpoint 完成，Checkpoint 间隔越长，扩容越慢。


**留一个问题**：你的生产作业用的是什么重启策略？有没有遇到过"重启风暴"？
:::
