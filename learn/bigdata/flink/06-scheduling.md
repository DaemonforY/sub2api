---
title: "Flink 源码导读 06：调度与容错 —— Failover、重启策略与自适应调度"
description: "调度器、Slot 分配、故障恢复策略和自适应调度（Adaptive Scheduler）的扩缩容。"
bigdata: "flink"
---

# Flink 源码导读 06：调度与容错 —— Failover、重启策略与自适应调度

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink/runtime`，`CFG` = `flink-core/src/main/java/org/apache/flink/configuration`
> 前置阅读：[01](/bigdata/flink/01-execution) 第 4 节（调度与部署）、[02](/bigdata/flink/02-checkpoint)、[03](/bigdata/flink/03-state-backend) 第 6 节（扩缩容时的状态重分配）
> 配套示例：`demos/src/main/java/study/scheduler/`，第 8 节的所有数据都是在本机实测的

## 0. 本篇要回答的问题

1. 一个 Task 抛出异常之后，JobManager 里发生了什么？
2. 为什么有的作业一个 Task 失败要重启全部 Task，有的只重启一个？**Pipelined Region** 是什么？
3. 默认的重启策略是什么？重启间隔是怎么算出来的？
4. 恢复时是如何回到最近一次 Checkpoint 的？
5. **AdaptiveScheduler** 如何根据可用资源自动扩缩容？
6. **AdaptiveBatchScheduler** 如何根据数据量决定并行度？

## 1. 三种调度器

| 调度器 | 适用场景 | 核心特点 | 如何启用 |
|---|---|---|---|
| **`DefaultScheduler`** | 流作业默认 | 并行度固定；按 Pipelined Region 调度和 failover | 默认 |
| **`AdaptiveScheduler`** | 流作业，资源弹性 | **先声明需要多少资源，拿到多少就用多少**；资源变化时自动扩缩容 | `jobmanager.scheduler: adaptive`（`CFG/JobManagerOptions.java:518`），或 `scheduler-mode: reactive`（`:561`） |
| **`AdaptiveBatchScheduler`** | 批作业默认（动态图） | 根据上游的**实际数据量**决定下游的并行度；支持推测执行 | 批作业自动选择 |

选择逻辑在 `DefaultSlotPoolServiceSchedulerFactory.getSchedulerType()`（`RT/jobmaster/DefaultSlotPoolServiceSchedulerFactory.java:236`）：
- 批作业：配置了 adaptive 或 reactive 时会被**改写**成 AdaptiveBatch（日志 `Adaptive Scheduler configured, but Batch job detected. Changing scheduler type to 'AdaptiveBatch'`）；否则，动态图用 AdaptiveBatch，静态图用 Default
- 流作业：reactive 模式 → Adaptive；否则看 `jobmanager.scheduler`，默认 Default

本篇前半部分讲 `DefaultScheduler` 的 failover，后半部分讲两种自适应调度器。

---

## 2. Failover 全流程（DefaultScheduler）

### 2.1 失败如何上报到调度器

```
Task 线程抛出异常
 └ Task.doRun() 的 catch 块：transitionState(RUNNING → FAILED)，记录 failureCause
   └ taskManagerActions.updateTaskExecutionState(...)            （01 篇 5.2 节的同一条路径）
     └ RPC → JobMaster.updateTaskExecutionState()                RT/jobmaster/JobMaster.java:537
       └ schedulerNG.updateTaskExecutionState(...)                :543
         └ SchedulerBase.updateTaskExecutionState()               RT/scheduler/SchedulerBase.java:808
            ├ executionGraph.updateState(...)                    更新 ExecutionGraph 中的状态机
            └ onTaskExecutionStateUpdate()                       :820 → onTaskFailed()（抽象方法，:843）
```

### 2.2 `DefaultScheduler`：决定怎么处理

`RT/scheduler/DefaultScheduler.java`

```java
protected void onTaskFailed(final Execution execution) {                        // :278
    final Throwable error = execution.getFailureInfo().get().getException().deserializeError(userCodeLoader);
    handleTaskFailure(execution, maybeTranslateToClusterDatasetException(error, ...));
}

protected void handleTaskFailure(...) {                                         // :289
    maybeRestartTasks(recordTaskFailure(failedExecution, error));
}

protected FailureHandlingResult recordTaskFailure(...) {                        // :294
    setGlobalFailureCause(error, timestamp);
    notifyCoordinatorsAboutTaskFailure(failedExecution, error);                 // 通知 OperatorCoordinator（如 SourceCoordinator 回收 split）
    return executionFailureHandler.getFailureHandlingResult(failedExecution, error, timestamp);   // :300
}
```

**`ExecutionFailureHandler`**（`RT/executiongraph/failover/ExecutionFailureHandler.java`）把两个问题分开回答：

- **重启哪些 Task？** `failoverStrategy.getTasksNeedingRestart(...)`（`:122`）→ 第 3 节
- **能不能重启、等多久？** `handleFailure()`（`:177`）：

```java
if (isUnrecoverableError(cause)) {                          // :187 不可恢复的错误：直接让作业失败
    return FailureHandlingResult.unrecoverable(...);
}
boolean isNewAttempt = restartBackoffTimeStrategy.notifyFailure(cause);   // :197
if (restartBackoffTimeStrategy.canRestart()) {                            // :198
    if (isNewAttempt) { numberOfRestarts++; }
    return FailureHandlingResult.restartable(..., verticesToRestart,
            restartBackoffTimeStrategy.getBackoffTime(), ...);            // :209 → 第 4 节
} else {
    return FailureHandlingResult.unrecoverable(...,
            new JobException("Recovery is suppressed by " + restartBackoffTimeStrategy, cause), ...);
}
```

**不可恢复的错误**（`isUnrecoverableError()`，`:224`）：异常链上有被 `@ThrowableAnnotation(ThrowableType.NonRecoverableError)` 标注的异常类型。比如作业配置错误，重启多少次都没用，就直接失败，不再浪费重启次数。

日志里熟悉的 **`Recovery is suppressed by ...`** 就是在这里产生的：重启次数用完了。

### 2.3 重启：取消 → 等待 → 恢复状态 → 重新调度

`maybeRestartTasks()` → `restartTasksWithDelay()`（`DefaultScheduler.java:358`）：

```java
final Set<ExecutionVertexVersion> executionVertexVersions =
        executionVertexVersioner.recordVertexModifications(verticesToRestart).values();   // ① 记录版本号
log.info("{} tasks will be restarted to recover the failed task {}.", ...);            // ★ 实测日志
addVerticesToRestartPending(verticesToRestart);        // 作业状态：RUNNING → RESTARTING
final CompletableFuture<?> cancelFuture = cancelTasksAsync(verticesToRestart);          // ② :383 取消
delayExecutor.schedule(                                 // ③ :387 等待退避时间
        () -> cancelFuture.thenRunAsync(() -> restartTasks(executionVertexVersions, globalRecovery), ...),
        failureHandlingResult.getRestartDelayMS(), TimeUnit.MILLISECONDS);
```

`restartTasks()`（`:415`）：

```java
final Set<ExecutionVertexID> verticesToRestart =
        executionVertexVersioner.getUnmodifiedExecutionVertices(executionVertexVersions);   // ★ 版本号检查
if (verticesToRestart.isEmpty()) { return; }
removeVerticesFromRestartPending(verticesToRestart);
resetForNewExecutions(verticesToRestart);        // ④ :427 为每个 ExecutionVertex 创建新的 Execution（attemptNumber + 1）
restoreState(verticesToRestart, isGlobalRecovery);   // ⑤ :430 从 Checkpoint 恢复状态
schedulingStrategy.restartTasks(verticesToRestart);   // ⑥ :436 重新申请 slot 并部署（01 篇 4.8 节）
```

**`ExecutionVertexVersioner` 有什么用？** 等待退避的这段时间里，可能又有新的失败（比如下游 Task 也因为上游挂了而报错），引发新一轮重启。版本号保证**旧的重启计划不会覆盖新的**：如果某个 vertex 在等待期间又被别的 failover 修改过，旧计划就跳过它。

### 2.4 恢复状态

`SchedulerBase.restoreState()`（`RT/scheduler/SchedulerBase.java:462`）：

```java
checkpointCoordinator.abortPendingCheckpoints(
        new CheckpointException(CheckpointFailureReason.JOB_FAILOVER_REGION));   // 正在进行的 Checkpoint 全部作废
if (isGlobalRecovery) {
    checkpointCoordinator.restoreLatestCheckpointedStateToAll(jobVerticesToRestore, true);   // :495
} else {
    checkpointCoordinator.restoreLatestCheckpointedStateToSubtasks(subtasksToRestore.keySet());   // :502
}
```

→ `CheckpointCoordinator.restoreLatestCheckpointedStateInternal()`（`RT/checkpoint/CheckpointCoordinator.java:1745`）：
1. `completedCheckpointStore.getLatestCheckpoint()`（`:1768`）：取出最近一次**完成**的 Checkpoint
2. 打印 `Restoring job {} from {}.`（`:1798`）：实测日志中的 `Restoring job ... from Checkpoint 2`
3. `new StateAssignmentOperation(...).assignStates()`（`:1812`、`:1819`）：把状态句柄分配给要重启的 subtask（03 篇 6.1 节，此时并行度没变，所以就是原样分配）

状态句柄随后放进 TDD 里，Task 在 `StreamTask.restore()`（01 篇 5.3 节）中真正把状态读回来。

**恢复的是最近一次完成的 Checkpoint，所以从这个 Checkpoint 到失败之间处理过的数据会被重新处理一遍**。Source 的位点也回退到那个 Checkpoint（Kafka offset 等），所以数据不会丢，只会重复计算；再配合两阶段提交的 Sink，对外就是精确一次（02 篇 7.3 节）。

---

## 3. ★ Pipelined Region：重启的最小单位

### 3.1 定义

**用 pipelined 边相连的 Task 构成一个 region**。pipelined 边意味着上下游必须**同时运行**（数据在内存里流过去，不落盘），所以一个挂了，另一个也无法继续。blocking 边（批作业的 shuffle 数据会落盘）则把 region 切开。

计算：`SchedulingPipelinedRegionComputeUtil.computePipelinedRegions()`（`RT/executiongraph/failover/SchedulingPipelinedRegionComputeUtil.java:46`）→ `buildRawRegions()` 把通过 `mustBePipelinedConsumed()` 的边（`:115`）相连的 vertex 合并起来 → `mergeRegionsOnCycles()`（`:69`）合并环。ExecutionGraph 构建时在 `DefaultExecutionTopology` 中计算好（`RT/scheduler/adapter/DefaultExecutionTopology.java:401`）。

### 3.2 流作业中的 region

**流作业中所有边都是 pipelined 的**，所以 region 的划分完全取决于**连接模式**：

```
拓扑 A：keyBy（all-to-all）                        拓扑 B：全部 forward（pointwise）

 Source(1) ──┬──► Count(1)                          Source(1) → Map(1) → Sink(1)    ← region 1
             ╳                                      Source(2) → Map(2) → Sink(2)    ← region 2
 Source(2) ──┴──► Count(2)

 所有 Task 都互相连通 → 只有 1 个 region               每个 subtask 各自独立 → 2 个 region
 任何一个失败 → 全部重启                               subtask 1 失败 → 只重启 region 1
```

大多数真实的流作业都有 keyBy、rebalance 这类 all-to-all 边，所以**整张图就是一个 region，一个 Task 失败就要重启全部 Task**。只有"embarrassingly parallel"的作业（比如纯 ETL：Kafka → map/filter → Kafka，并行度都一样）才能享受细粒度 failover 的好处。

### 3.3 `RestartPipelinedRegionFailoverStrategy`

默认的 failover 策略是 `region`（`jobmanager.execution.failover-strategy`，`CFG/JobManagerOptions.java:284`），另一个可选值是 `full`（`RestartAllFailoverStrategy`）。

`getTasksNeedingRestart()`（`RT/executiongraph/failover/RestartPipelinedRegionFailoverStrategy.java:110`）→ `getRegionsToRestart()`（`:158`），从失败的 region 出发做一次 BFS：

1. 失败的 region 本身要重启
2. 如果它需要消费的上游结果分区**已经不可用了**（比如批作业中上游的 blocking 数据丢了，或者异常本身就是 `PartitionException`），上游的 **producer region** 也要重启
3. 它的下游 **consumer region** 也要重启，因为它们消费的数据要重新产生

最后，状态还是 `CREATED` 的 vertex（还没被调度过）不需要重启（`:136`）。第 8.1 节的实验会验证拓扑 A 和拓扑 B 的区别。

---

## 4. 重启策略

### 4.1 默认值

`RestartBackoffTimeStrategyFactoryLoader`（`RT/executiongraph/failover/RestartBackoffTimeStrategyFactoryLoader.java`）：

- 显式配置了 `restart-strategy.type`（`CFG/RestartStrategyOptions.java:115-116`）就按配置来（`:74` 的 switch）
- **没有配置时**（`getDefaultRestartStrategyFactory()`，`:95`）：
  - **开启了 Checkpoint → `exponential-delay`**（`:97`），使用默认参数
  - **没有开启 Checkpoint → 不重启**（`NoRestartBackoffTimeStrategy`，`:118`）

> ⚠️ 很多老资料说"默认是 fixed-delay，重启 `Integer.MAX_VALUE` 次"，那是很早以前的行为。在 2.3.0 中，开启 Checkpoint 后的默认策略是**指数退避**，并且**不限次数**（第 4.2 节的 `attempts-before-reset-backoff` 默认是 `Integer.MAX_VALUE`）。

可选的策略：`disable`/`none`、`fixed-delay`、`failure-rate`、`exponential-delay`，分别对应同一目录下的 `NoRestartBackoffTimeStrategy`、`FixedDelayRestartBackoffTimeStrategy`、`FailureRateRestartBackoffTimeStrategy`、`ExponentialDelayRestartBackoffTimeStrategy`。

### 4.2 指数退避

`ExponentialDelayRestartBackoffTimeStrategy`（`RT/executiongraph/failover/ExponentialDelayRestartBackoffTimeStrategy.java`），默认参数（`CFG/RestartStrategyOptions.java`）：

| 参数 | 默认值 | 行号 |
|---|---|---|
| `restart-strategy.exponential-delay.initial-backoff` | 1 秒 | `:228-231` |
| `...max-backoff` | 1 分钟 | `:241-244` |
| `...backoff-multiplier` | 1.5 | `:254-259` |
| `...reset-backoff-threshold` | 1 小时 | `:270-275` |
| `...jitter-factor` | 0.1 | `:287-290` |
| `...attempts-before-reset-backoff` | `Integer.MAX_VALUE` | `:302-305` |

核心逻辑：

```java
public boolean notifyFailure(Throwable cause) {                  // :112
    long now = clock.absoluteTimeMillis();
    if (now <= nextRestartTimestamp) {
        return false;                          // ★ 还在上一次的退避期内：算作同一次失败，不增加重启计数
    }
    if ((now - nextRestartTimestamp) >= resetBackoffThresholdMS) {
        setInitialBackoff();                   // 距离上次重启已经很久（默认 1 小时）：退避时间重置为初始值
    }
    nextRestartTimestamp = now + calculateActualBackoffTime();   // 退避时间 × 1.5，再加 ±10% 抖动
    currentRestartAttempt++;
    return true;
}

public boolean canRestart() {                                    // :99
    return currentRestartAttempt <= attemptsBeforeResetBackoff;
}
```

- **退避期内的失败被合并**（`return false`）：一个 TM 宕机可能导致几十个 Task 同时失败，它们只算一次重启，退避时间也只增长一次。这就是 2.2 节中 `isNewAttempt` 的含义
- **抖动（jitter）**：大量作业共享一个出了故障的外部系统（比如同一个 Kafka 集群）时，避免它们在同一时刻集体重启

第 8.1 节实测的两次重启间隔约为 1.0 秒和 1.44 ~ 1.65 秒，正好是 1 秒 × 1.5 ± 10%。

### 4.3 全局失败

`handleGlobalFailure()`（`DefaultScheduler.java:337`）处理**不属于某个 Task** 的失败，比如：
- 连续失败的 Checkpoint 超过了 `execution.checkpointing.tolerable-failed-checkpoints`（02 篇 7.4 节）
- OperatorCoordinator 出错
- 状态恢复失败（`restartTasks()` 中 `restoreState` 抛出异常，`:430` 附近 → `handleGlobalFailure(t)`）

全局失败**总是重启所有 Task**（`getGlobalFailureHandlingResult()`，`ExecutionFailureHandler.java:135`），并调用 `restoreLatestCheckpointedStateToAll`。

---

## 5. AdaptiveScheduler：拿到多少资源就用多少

### 5.1 解决的问题

`DefaultScheduler` 的并行度是固定的：作业要 4 个 slot，集群只有 2 个，就**一直等下去**，直到超时失败。在 Kubernetes 这类弹性环境中，我们更希望：
- 资源不够时，**先用现有资源跑起来**
- 资源增加时（HPA 扩容了 TM），**自动扩大并行度**
- 资源减少时（节点被回收），**自动缩小并行度**继续运行，而不是一直失败重试

这就是 **AdaptiveScheduler**（FLIP-160），以及基于它的 **Reactive Mode**（FLIP-159：作业总是使用集群中的全部资源）。

### 5.2 状态机

`RT/scheduler/adaptive/` 目录下，每个状态都是一个类：

```
Created ──► WaitingForResources ──► CreatingExecutionGraph ──► Executing ──┬──► Finished
               ▲         ▲                    ▲                  │         ├──► Canceling
               │         │                    │                  │         ├──► Failing
               │         └──── Restarting ◄───┼──────────────────┤         └──► StopWithSavepoint
               │                    │         │                  │
               │                    └─────────┘  资源已就绪        │ 失败 / 需要扩缩容
               └──── 资源还没到位 ─────────────────────────────────┘
```

状态迁移都通过 `AdaptiveScheduler.transitionToState()`（`RT/scheduler/adaptive/AdaptiveScheduler.java:1735`）完成，它会打印 DEBUG 日志 `Transition from state {} to {}.`（`:1747`）。各个 `goToXxx()` 方法定义在 `:1263`（WaitingForResources）、`:1299`（Executing）、`:1355`（Restarting）、`:1427`（Failing）、`:1489`（CreatingExecutionGraph）等位置。

### 5.3 关键决策

**资源够不够？**
- `hasDesiredResources()`（`:1182`）：够不够**声明的**并行度
- `hasSufficientResources()`（`:1207`）：能不能按**最小并行度**跑起来（`slotAllocator.determineParallelism(...)`，`:1209`）
- `determineParallelism()`（`:1227`）：`SlotSharingSlotAllocator`（`RT/scheduler/adaptive/allocator/SlotSharingSlotAllocator.java`）根据可用的 slot 数计算每个 vertex 的实际并行度

**WaitingForResources**：作业提交时，拿到期望的资源就立刻开始；否则等待资源稳定下来（`jobmanager.adaptive-scheduler.submission.resource-stabilization-timeout`，默认 **10 秒**，`CFG/JobManagerOptions.java:776`），然后用现有资源先跑起来；最多等 `submission.resource-wait-timeout`（默认 5 分钟，`:694`）。

**Executing 中何时触发扩缩容？** 新的 slot 到达时，`newResourcesAvailable()`（`AdaptiveScheduler.java:753`）通知当前状态。`Executing` 把决策交给 **`DefaultStateTransitionManager`**（`RT/scheduler/adaptive/DefaultStateTransitionManager.java:70`），它本身也是一个状态机（`:45-69` 的注释）：

```
Cooldown ──► Idling ──► Stabilizing ──► Stabilized ──► Transitioning（真正触发重启扩缩容）
```

- **Cooldown**：刚扩缩容完，冷却一段时间，避免抖动（`jobmanager.adaptive-scheduler.executing.cooldown-after-rescaling`，默认 **30 秒**，`CFG/JobManagerOptions.java:588-589`）
- **Stabilizing**：资源发生变化后，再等一会，看资源是否还在继续变化（`executing.resource-stabilization-timeout`，默认 **60 秒**，`:599-600`）。比如 K8s 一次扩出 10 个 Pod，是陆续就绪的，没必要每来一个就重启一次
- 判断是否值得扩缩容：`Executing.hasSufficientResources()` / `hasDesiredResources()`（`RT/scheduler/adaptive/Executing.java:146`、`:151`）都要求 `parallelismChanged()`（`:155`）为真，并且资源满足要求

**扩缩容 = 一次重启**：进入 `Restarting` 状态，取消所有 Task，然后按新的并行度重新创建 ExecutionGraph，**从最近的 Checkpoint 恢复**，状态按 KeyGroup 重新分配（03 篇 6.1、6.2 节）。所以扩缩容是有代价的：会中断处理，状态大的话恢复也慢。这也是 03 篇第 7 节 ForSt 存算分离想要解决的问题。

**AdaptiveScheduler 中的失败处理**：`howToHandleFailure()`（`AdaptiveScheduler.java:1654`）同样先检查不可恢复的错误（`:1665`），再询问重启策略（`:1670-1672`），和 DefaultScheduler 用的是同一套 `RestartBackoffTimeStrategy`。但它**总是重启整个作业**，没有 region 的概念。

---

## 6. AdaptiveBatchScheduler：按数据量决定并行度

批作业的上下游之间通常是 **blocking** 的数据交换（上游全部跑完，把数据写到磁盘，下游才开始），这就给了调度器一个机会：**等上游跑完，看看它实际产出了多少数据，再决定下游用多大的并行度**。

- 01 篇说过，2.x 的 JobGraph 在 JM 侧生成。对批作业来说，这一点更进一步：JobGraph 可以**边运行边生成**（`onNewJobVerticesAdded()`，`RT/scheduler/adaptivebatch/AdaptiveBatchScheduler.java:261`，配合 `StreamGraphOptimizer`、`AdaptiveGraphManager`）
- 每当有 Task 完成（`onTaskFinished()`，`:404`），就尝试初始化后续的 vertex（`initializeVerticesIfPossible()`，`:309`、`:424`）
- **决定并行度**：`DefaultVertexParallelismAndInputInfosDecider.decideParallelismAndInputInfosForVertex()`（`RT/scheduler/adaptivebatch/DefaultVertexParallelismAndInputInfosDecider.java:103`）根据输入数据量 ÷ `execution.batch.adaptive.auto-parallelism.avg-data-volume-per-task`（默认 **16MB**）计算并行度，再限制在 `min-parallelism`（默认 1）和 `max-parallelism`（默认 128）之间（`CFG/BatchExecutionOptions.java`）。它还会**决定每个下游 subtask 读取上游的哪些分区**，以缓解数据倾斜
- **推测执行**（`execution.batch.speculative.enabled`，默认 false）：`DefaultSpeculativeExecutionHandler`（`RT/scheduler/adaptivebatch/DefaultSpeculativeExecutionHandler.java`）发现某个 Task 明显比同批的其他 Task 慢（慢节点），就在别的机器上**再跑一份**，谁先跑完就用谁的结果

---

## 7. 断点清单

用 IDEA 运行 `FailoverDemo`，参数 `keyby`（第 9 ~ 12 行用 `AdaptiveSchedulerDemo`）。

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `SchedulerBase.java:808` | 失败状态的上报 |
| 2 | `DefaultScheduler.java:278` `onTaskFailed` | 反序列化出来的异常 |
| 3 | `RestartPipelinedRegionFailoverStrategy.java:110` | 失败的 region 包含哪些 vertex |
| 4 | `ExponentialDelayRestartBackoffTimeStrategy.java:112` | 退避时间的计算，`isNewAttempt` |
| 5 | `DefaultScheduler.java:387` | 退避延迟 |
| 6 | `DefaultScheduler.java:415` `restartTasks` | 版本号检查后剩下哪些 vertex |
| 7 | `CheckpointCoordinator.java:1768` | 用哪一个 Checkpoint 恢复 |
| 8 | 示例中 `CountingMap.initializeState` | `attemptNumber`、恢复出来的计数 |
| 9 | `AdaptiveScheduler.java:753` `newResourcesAvailable` | 新 TM 的 slot 到达 |
| 10 | `DefaultStateTransitionManager.java` 中各个 Phase 的切换 | 冷却、稳定 |
| 11 | `AdaptiveScheduler.java:1227` `determineParallelism` | 按可用 slot 算出的并行度 |
| 12 | `AdaptiveScheduler.java:1735` `transitionToState` | 每次状态迁移 |

---

## 8. 实验

### 8.1 Failover：一个 region vs 多个 region

`FailoverDemo.java`：并行度 2，每 2 秒做一次 Checkpoint。`fail-map` 让 subtask 0 **在第 0 次和第 1 次执行时**（`attemptNumber < 2`）各处理 3000 条后抛出异常；`count` 用 operator state 记录处理过的条数，并在 `initializeState` 时打印出来。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.scheduler.FailoverDemo keyby
```

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.scheduler.FailoverDemo forward
```

**拓扑 A（keyby）**：`Source → fail-map ─keyBy─► after-keyby → count → Sink`，一共 4 个 Task（两条算子链 × 2 个并行度）

```
22:25:30,381 Task - Source: source -> fail-map (1/2)#0 switched from RUNNING to FAILED
22:25:30,392 JobMaster - 4 tasks will be restarted to recover the failed task ...
22:25:31,408 CheckpointCoordinator - Restoring job ... from Checkpoint 2
>>> [after-keyby -> count -> Sink: Writer (2/2)#1] attempt=1 initializeState: restored=true count=2298
>>> [after-keyby -> count -> Sink: Writer (1/2)#1] attempt=1 initializeState: restored=true count=1786
22:25:34,426 Task - Source: source -> fail-map (1/2)#1 switched from RUNNING to FAILED
22:25:34,428 JobMaster - 4 tasks will be restarted to recover the failed task ...
22:25:35,866 CheckpointCoordinator - Restoring job ... from Checkpoint 3
>>> [after-keyby -> count -> Sink: Writer (2/2)#2] attempt=2 initializeState: restored=true count=4566
>>> [after-keyby -> count -> Sink: Writer (1/2)#2] attempt=2 initializeState: restored=true count=3547
```

**拓扑 B（forward）**：`Source → fail-map → count → Sink` 全部链在一起，一共 2 个 Task

```
22:25:58,583 Task - Source: source -> fail-map -> count -> Sink: Writer (1/2)#0 switched from RUNNING to FAILED
22:25:58,598 JobMaster - 1 tasks will be restarted to recover the failed task ...
22:25:59,613 CheckpointCoordinator - Restoring job ... from Checkpoint 3
>>> [Source: source -> fail-map -> count -> Sink: Writer (1/2)#1] attempt=1 initializeState: restored=true count=2036
22:26:02,630 Task - ... (1/2)#1 switched from RUNNING to FAILED
22:26:02,633 JobMaster - 1 tasks will be restarted to recover the failed task ...
22:26:04,284 CheckpointCoordinator - Restoring job ... from Checkpoint 5
>>> [Source: source -> fail-map -> count -> Sink: Writer (1/2)#2] attempt=2 initializeState: restored=true count=4057
```

| | 拓扑 A（keyby） | 拓扑 B（forward） |
|---|---|---|
| region 数 | 1 | 2 |
| 每次重启的 Task 数 | **4** | **1** |
| 没有失败的 subtask 是否被重启 | 是：`count (2/2)` 从 `#0` → `#1` → `#2` | **否**：`(2/2)` 始终是 `#0`，从未重新初始化 |
| 第 1 次重启延迟（失败 → Restoring） | 1.02 秒 | 1.02 秒 |
| 第 2 次重启延迟 | 1.44 秒 | 1.65 秒 |

解读：
1. **region 决定了重启范围**（3.2 节）。keyBy 把所有 Task 连成了一个 region，下游 `count` 明明没有出错，也被连带重启了
2. **重启延迟符合指数退避**（4.2 节）：第 1 次约 1 秒，第 2 次约 1.5 秒（×1.5），两次实测 1.44 秒和 1.65 秒，差别来自 ±10% 的抖动
3. **状态从 Checkpoint 恢复**：attempt 1 的计数是从 Checkpoint 中恢复出来的（比如 2298），而不是从 0 开始。失败前、最后一个 Checkpoint 之后处理的那部分数据会被**重新处理**
4. `#0`、`#1`、`#2` 就是 **attemptNumber**，线程名里直接可以看到，排查问题时很有用

### 8.2 AdaptiveScheduler：扩容与缩容

`AdaptiveSchedulerDemo.java`：直接使用 `MiniCluster` API，作业**声明并行度 4**，但集群一开始只有 **1 个 TM（2 个 slot）**。冷却时间调成 5 秒，资源稳定等待调成 3 秒。第 15 秒加入第 2 个 TM，第 35 秒停掉一个 TM。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.scheduler.AdaptiveSchedulerDemo
```

实测（每 2.5 秒采样一次作业状态和各 vertex 的并行度，并打开 `AdaptiveScheduler` 的 DEBUG 日志）：

```
>>> t=0s ~ t=10s: CREATED map=4, Source: source=4             ← 声明的并行度，还没开始跑
22:30:10,752 Transition from state Created to WaitingForResources.
22:30:20,857 Transition from state WaitingForResources to CreatingExecutionGraph.    ← 等了 10 秒
22:30:20,921 Transition from state CreatingExecutionGraph to Executing.
>>> t=12s: RUNNING Source: source=2, map=2                      ← ★ 凑不齐 4 个 slot，先用 2 个跑
>>> t=15s: starting a 2nd TaskManager (+2 slots)
22:30:27,059 Transition from state Executing to Restarting.
22:30:27,072 Transition from state Restarting to CreatingExecutionGraph.
22:30:27,081 Transition from state CreatingExecutionGraph to Executing.
>>> t=17s: RUNNING Source: source=4, map=4                      ← ★ 自动扩容到 4
>>> t=35s: terminating TaskManager #0 (-2 slots)
22:30:45,915 Transition from state Executing to Restarting.     ← TM 丢失：Task 失败
22:30:46,927 Transition from state Restarting to WaitingForResources.      ← 1 秒退避
22:30:56,941 Transition from state WaitingForResources to CreatingExecutionGraph.   ← 又等了 10 秒
22:30:56,948 Transition from state CreatingExecutionGraph to Executing.
>>> t=47s: RUNNING Source: source=2, map=2                      ← ★ 自动缩容到 2，继续运行
```

解读：
1. **提交时等待了 10 秒**：这就是 `submission.resource-stabilization-timeout` 的默认值（5.3 节）。拿不到期望的 4 个 slot，就等资源稳定下来，然后按现有的 2 个 slot 启动
2. **扩容路径**：Executing → Restarting → CreatingExecutionGraph → Executing，**跳过了 WaitingForResources**，因为资源已经到位了
   - **补充（2026-10-02 重跑）**：扩容是在 **Checkpoint 完成时**触发的（`RT/scheduler/adaptive/Executing.java:280-281`，`onCompletedCheckpoint → triggerPotentialRescale`）。重跑日志中 `Completed checkpoint 4` 在 18:15:15,195，4 毫秒后进入 Restarting，并且从 Checkpoint 4 恢复。详见 `content-plan/18-第6讲-公众号定稿.md` 5.3 节
3. **缩容路径**：TM 丢失对调度器来说是一次**失败**（而不是主动的扩缩容），所以先按重启策略退避 1 秒；然后进入 WaitingForResources，再等 10 秒，看看丢失的资源会不会回来；最后才按剩下的 2 个 slot 重新启动
4. 对比：如果用 DefaultScheduler 跑这个作业，它在只有 2 个 slot 时**根本不会启动**，TM 丢失后也会一直重试，直到重新凑齐 4 个 slot

> 🐛 **踩坑记录**：第一次运行时，扩容后作业一直卡在 RESTARTING，并不停地打印 `UnsupportedOperationException: Local execution does not support shuffle connection`。原因是 `MiniCluster.useLocalCommunication()`（`RT/minicluster/MiniCluster.java:795-796`）按**启动时配置的 TM 数量**决定是否只用本地通信：配置的是 1 个 TM，就用 `LocalConnectionManager`，后来通过 `startTaskManager()` 加进来的 TM 也继承了这个设置，TM 之间就无法交换数据。示例里用一个匿名子类覆盖这个方法，强制使用 Netty。
>
> 另外，Adaptive Scheduler 的状态迁移日志是 **DEBUG** 级别的，而 DefaultScheduler 的 `N tasks will be restarted` 用的是 **JobMaster 的 logger**（`RpcEndpoint` 中的 `LoggerFactory.getLogger(getClass())`），`Execution` 的状态迁移日志用的是 **`ExecutionGraph` 的 logger**（`DefaultExecutionGraph.java:138`）。想在日志里看到这些信息，要在 log4j 中配置正确的 logger 名字（示例的 `log4j2.properties` 中已经配置好了）。

---

## 9. 课后练习

1. **不可恢复的错误**：定义一个异常类，加上 `@ThrowableAnnotation(ThrowableType.NonRecoverableError)`，在 `FailingMap` 中抛出它，观察作业是否不再重启而直接失败，以及日志中的 `The failure is not recoverable`
2. **重启次数用完**：设置 `restart-strategy.type: fixed-delay`、`restart-strategy.fixed-delay.attempts: 1`，让 `FailingMap` 失败 2 次，找到 `Recovery is suppressed by` 这行日志
3. **不开 Checkpoint**：去掉 `enableCheckpointing`，观察第一次失败后作业是否直接失败（4.1 节：此时默认不重启）
4. **full 策略**：在 forward 拓扑中设置 `jobmanager.execution.failover-strategy: full`，确认重启的 Task 数变成了 2
5. **Reactive Mode**：把 AdaptiveSchedulerDemo 改成 `scheduler-mode: reactive`，并把作业的并行度设为 1，观察它在资源增加时是否自动扩大到 slot 总数。思考：reactive 模式下，作业声明的并行度还有什么意义？
6. **思考题**：为什么 AdaptiveScheduler 不支持 region failover？（提示：它每次都会按新的并行度重新创建 ExecutionGraph）

## 10. 下一篇预告

**07：从读者到贡献者**。前 6 篇从使用者和读者的角度剖析了 Flink 的核心模块，下一篇转向"如何参与社区"：挑选第一个 JIRA、搭建开发环境、代码规范与 spotless、跑测试（单测、ITCase、架构测试）、写一个合格的 PR，并从 Git 历史中挑一个真实的 bug 修复（比如本系列涉及的某个模块），完整复盘一遍"发现问题 → 定位 → 修复 → 测试 → review"的过程。
:::
