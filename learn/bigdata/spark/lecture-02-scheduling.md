---
title: "第 2 讲：调度体系"
description: "DAGScheduler 与 TaskScheduler：Stage 何时提交、多 Job 调度、本地性、失败重试和推测执行。"
bigdata: "spark"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-02-scheduling-cover.webp"}]]
---

# 第 2 讲：调度体系

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-02-scheduling-cover.webp" alt="调度三层协作运行任务" width="1200" height="800" loading="eager" /><figcaption>调度三层协作运行任务<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 文中所有配置默认值、类名、方法名、行号均已在 4.2.0 源码中核实。

## 0. 本讲要回答的问题

第 1 讲我们知道了：Action 触发 Job，Job 按宽依赖切成 Stage，每个分区一个 Task。
但还有一堆问题没回答：

1. 一个 Stage 什么时候才能提交？谁来决定？
2. 多个 Job 同时提交时，谁先跑？
3. 一个 Task 到底被分配到哪台机器、哪个 Executor？
4. Task 失败了怎么办？机器挂了怎么办？
5. 有一个 Task 特别慢，拖住了整个 Stage，怎么办？

这些就是**调度体系**要解决的事。

---

## 1. 全景：三层分工

```
            ┌──────────────────────────────────────────────┐
  Job ────▶ │ DAGScheduler（逻辑调度）                      │
            │   Stage 切分、Stage 依赖、Stage 失败重算       │
            └──────────────┬───────────────────────────────┘
                           │ submitTasks(TaskSet)   一个 Stage 的一次尝试 = 一个 TaskSet
            ┌──────────────▼───────────────────────────────┐
            │ TaskSchedulerImpl（物理调度）                  │
            │   调度池（FIFO/FAIR）、数据本地性、Task 重试、  │
            │   推测执行；每个 TaskSet 由一个 TaskSetManager 管理│
            └──────────────┬───────────────────────────────┘
                           │ resourceOffers(空闲资源) → TaskDescription
            ┌──────────────▼───────────────────────────────┐
            │ SchedulerBackend（通信层）                    │
            │   掌握有哪些 Executor、各有多少空闲核；         │
            │   把 Task 发给 Executor，接收状态汇报          │
            └──────────────────────────────────────────────┘
```

**一句话分工**：DAGScheduler 决定"**跑什么**"，TaskScheduler 决定"**在哪跑、谁先跑**"，SchedulerBackend 负责"**送过去**"。

---

## 2. DAGScheduler：逻辑调度

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-02-scheduling-1.webp" alt="事件队列决定Stage提交" width="960" height="640" loading="lazy" /><figcaption>事件队列决定Stage提交<span>AI 生成配图</span></figcaption></figure>

### 2.1 单线程事件循环（第 1 讲思考题的答案）

DAGScheduler 内部有一个名为 `dag-scheduler-event-loop` 的线程（`DAGScheduler.scala:3507`），所有外部请求都被包装成**事件**放进队列，由这一个线程按顺序处理。

4.2.0 中 `doOnReceive` 处理的事件有 20 多种，最核心的是这几类：

| 事件 | 来源 | 作用 |
|---|---|---|
| `JobSubmitted` | 用户线程调用 Action | 创建 ResultStage，开始提交 Stage |
| `CompletionEvent` | TaskScheduler，Task 结束时 | 更新 Stage 状态，可能触发下游 Stage 提交 |
| `ExecutorLost` / `ExecutorAdded` | SchedulerBackend | Executor 挂了要清理它上面的 Shuffle 输出 |
| `ResubmitFailedStages` | 自己（延迟发送） | 重新提交失败的 Stage |
| `JobCancelled` / `StageCancelled` 等 | 用户取消 | 取消 Job / Stage |
| `SpeculativeTaskSubmitted` | TaskScheduler | 推测任务提交通知 |

**为什么用单线程事件循环，而不是加锁？**

DAGScheduler 维护着大量相互关联的状态：

```scala
// DAGScheduler.scala:164-170
private[scheduler] val waitingStages = new HashSet[Stage]   // 等待父 Stage 完成
private[scheduler] val runningStages = new HashSet[Stage]   // 正在运行
private[scheduler] val failedStages  = new HashSet[Stage]   // 失败、等待重新提交
```

还有 `jobIdToActiveJob`、`stageIdToStage`、`shuffleIdToMapStage` 等映射。一个事件往往要同时修改其中好几个。如果多线程直接调用，就要非常精细地加锁，极易出现死锁或状态不一致。

**把所有状态变更收敛到一个线程**，就天然不存在并发问题，代码也更容易理解和测试。代价是 DAGScheduler 本身的处理能力受单线程限制，但它只做"决策"，不做"计算"，每个事件的处理通常很快，所以这个代价完全可以接受。

> 这是分布式系统里非常经典的设计模式（Actor 模型 / 事件驱动）。Spark 的 RPC 端点（`RpcEndpoint`）、很多组件都用了类似思路。

### 2.2 Stage 的生命周期

```
            submitStage(stage)
                   │
          有缺失的父 Stage？
          ├── 是 → 先递归 submitStage(父 Stage)，自己进入 waitingStages
          └── 否 → submitMissingTasks(stage)，进入 runningStages
                          │
                 所有 Task 成功？
                 ├── 是 → Stage 完成，检查 waitingStages 里的子 Stage 能否提交
                 └── 遇到 FetchFailed → 进入 failedStages，稍后重新提交
```

源码入口：`submitStage`（`DAGScheduler.scala:1540`）、`submitMissingTasks`（`DAGScheduler.scala:1635`）。

### 2.3 submitMissingTasks 做了什么

这是 DAGScheduler 把 Stage 变成 Task 的地方，做了四件事：

1. **找出需要计算的分区**：`stage.findMissingPartitions()`。Stage 重试时只算丢失的分区，不是全部重算
2. **计算每个分区的位置偏好**：`getPreferredLocs`，为后面的数据本地性调度做准备
3. **序列化并广播 Task 的"二进制"**：把 RDD 和计算函数序列化成 `taskBinaryBytes`（`DAGScheduler.scala:1740` 附近），再用 `sc.broadcast` 广播出去。每个 Task 只携带这个广播变量的引用，而不是各带一份
   - **为什么要广播？** 一个 Stage 可能有上万个 Task，它们的 RDD 和函数完全相同。如果每个 Task 都带一份，Driver 要序列化、发送上万次。广播之后，每个 Executor 只需拉取一次
4. **打包成 TaskSet 交给 TaskScheduler**（`DAGScheduler.scala:1821`）：

```scala
taskScheduler.submitTasks(new TaskSet(
  tasks.toArray, stage.id, stage.latestInfo.attemptNumber(), jobId, properties, ...))
```

注意第 4 个参数 `jobId`：它会成为 TaskSet 的 **`priority`**，在 FIFO 调度中决定谁先跑。

---

## 3. TaskScheduler：谁先跑？

### 3.1 TaskSetManager 和调度池

- 每个 TaskSet 都由一个 **TaskSetManager** 管理，它负责这一批 Task 的：本地性选择、失败计数、推测执行
- 所有 TaskSetManager 被组织成一棵**调度池（Pool）树**，根节点是 `rootPool`
- 每次有空闲资源时，TaskScheduler 调用 `rootPool.getSortedTaskSetQueue` 得到一个**排好序的 TaskSet 列表**，按顺序分配资源

排序规则由调度模式决定：`spark.scheduler.mode`，默认 **FIFO**。

### 3.2 FIFO：先来先服务

`SchedulingAlgorithm.scala:29`：

```scala
// 先比 priority（即 jobId），小的优先；相同再比 stageId，小的优先
var res = math.signum(priority1 - priority2)
if (res == 0) res = math.signum(stageId1 - stageId2)
```

**含义**：先提交的 Job 优先拿到所有资源；同一个 Job 内，Stage 编号小的优先。

**问题**：一个大 Job 会把资源全部占满，后提交的小 Job（比如一个交互式查询）只能干等。

### 3.3 FAIR：公平调度

`SchedulingAlgorithm.scala:43`，比较规则按顺序：

1. **"饥饿"的优先**：`runningTasks < minShare` 的调度池叫"needy"（没拿到最低保障份额），needy 的排在前面
2. 都 needy：比 `runningTasks / minShare`，比值小的优先（离保障份额差得越多越优先）
3. 都不 needy：比 `runningTasks / weight`，比值小的优先（按权重分配）
4. 还相同：按名字排序

**怎么用**：

```scala
// 1. 开启公平调度
spark.scheduler.mode=FAIR
// 2. （可选）用 XML 文件定义调度池及其 weight、minShare
spark.scheduler.allocation.file=/path/to/fairscheduler.xml
// 3. 在提交 Job 的线程里指定调度池
sc.setLocalProperty("spark.scheduler.pool", "interactive")
```

> ⚠️ **常见误解**：FAIR 调度是**同一个 Application 内部、多个 Job 之间**的公平，不是多个 Application 之间的公平。多个 Application 之间的资源分配，由 YARN / K8s 这类集群管理器负责。
>
> 典型场景：一个常驻的 Spark 服务（比如 Thrift Server、Spark Connect Server）同时服务多个用户的查询。

### 3.4 资源分配的完整流程：resourceOffers

SchedulerBackend 会在这些时机把空闲资源"报价"（offer）给 TaskScheduler：

- 有新的 TaskSet 提交（`submitTasks` 里调用 `backend.reviveOffers()`）
- 有 Task 结束、释放了 CPU 核
- 有新 Executor 注册
- **定时**：集群模式下默认每 **1 秒**一次（`spark.scheduler.revive.interval`，未设置时为 1000ms，见 `CoarseGrainedSchedulerBackend.scala:160`）

`TaskSchedulerImpl.resourceOffers`（`TaskSchedulerImpl.scala:512`）的核心逻辑：

```
1. 过滤掉被排除的节点 / Executor
2. 随机打乱 offers（shuffleOffers）   ← 避免 Task 总是堆在前几个 Executor 上
3. 取得排好序的 TaskSet 列表（FIFO / FAIR）
4. for 每个 TaskSet（按优先级）:
     for 每个本地性级别（从最好到最差，但不超过当前允许的级别）:
       尽量把 Task 分配到满足这个本地性级别的 offer 上
5. 返回分配结果，SchedulerBackend 把 Task 发给对应的 Executor
```

注意第 4 步的两层循环：**先满足高优先级 TaskSet，并且尽量用好的本地性**。这就引出了下一节。

---

## 4. 数据本地性与延迟调度

### 4.1 五个本地性级别

`TaskLocality.scala:25`，**从好到差**：

| 级别 | 含义 | 典型场景 |
|---|---|---|
| `PROCESS_LOCAL` | 数据就在**同一个 Executor 进程**的内存里 | 读 cache 过的 RDD 分区 |
| `NODE_LOCAL` | 数据在**同一台机器**上，但不在这个进程里 | HDFS 数据块在本机磁盘；同机其他 Executor 缓存了数据 |
| `NO_PREF` | 数据**没有位置偏好**，在哪都一样 | 从外部数据库读数据 |
| `RACK_LOCAL` | 数据在**同一个机架**的其他机器上 | 需要跨机器，但不跨机架 |
| `ANY` | 任意位置 | 跨机架读取 |

> 移动计算比移动数据便宜：一个 Task 的序列化体积通常只有几 KB，而它要处理的数据可能有上百 MB。

### 4.2 延迟调度（Delay Scheduling）

**问题**：一个 Task 最想去 A 机器（数据在那），但 A 机器现在没空闲核，B 机器有空。是立刻在 B 上跑，还是等一等 A？

Spark 的答案是：**等一会儿，等不到再降级**。

- 每个本地性级别都有一个等待时间，默认都是 **3 秒**（`spark.locality.wait`；也可以用 `spark.locality.wait.process` / `.node` / `.rack` 分别设置）
- TaskSetManager 记录当前允许的本地性级别（`currentLocalityIndex`）。如果在当前级别**等待超时**还没分到资源，就**降一级**（`getAllowedLocalityLevel`，`TaskSetManager.scala:621`）
- **优化**：如果某个级别上已经没有待调度的 Task 了，就不必等，直接跳到下一级（源码注释里提到的 SPARK-4939）

**什么时候重置等待计时？**（4.x 的新行为）

- **新逻辑（默认）**：只有当 TaskSet 在拿到"全量资源报价"时（集群范围的 `makeOffers`，而不是单个 Executor 释放资源时的局部报价），**没有因为延迟调度而拒绝任何资源**，才重置计时
- **旧逻辑**：只要成功启动了一个 Task 就重置。可以用 `spark.locality.wait.legacyResetOnTaskLaunch=true` 恢复

> 旧逻辑的问题：只要偶尔有一个 Task 能以好的本地性启动，计时就一直被重置，TaskSet 永远不会降级，导致大量空闲资源被白白浪费。（参见 `TaskSchedulerImpl.scala` 第 71–81 行的类注释）

### 4.3 调优建议

在 Spark UI 的 Stage 页面，每个 Task 都显示了 **Locality Level**：

- **大量 Task 是 `ANY` / `RACK_LOCAL`，且读的是 HDFS**：可能是 Executor 分布和数据分布不匹配；可以适当调大 `spark.locality.wait`
- **Task 很短（几百毫秒），却经常空等 3 秒**：等待的代价比本地性收益大，可以调小 `spark.locality.wait`，甚至设为 0
- **数据在对象存储（S3、OSS）上**：本来就没有本地性可言，设成 0 往往更好

---

## 5. 失败与重试

### 5.1 Task 失败

- 由 TaskSetManager 的 `handleFailedTask`（`TaskSetManager.scala:951`）处理
- 同一个 Task 失败次数达到 **`spark.task.maxFailures`（默认 4）** 时，整个 TaskSet 被中止，Job 失败
- 注意是"**任何一个 Task 失败了 N 次**"，不是"所有 Task 加起来失败 N 次"

⚠️ **local 模式的坑**：

```scala
// SparkContext.scala
val MAX_LOCAL_TASK_FAILURES = 1
case "local"              => new TaskSchedulerImpl(sc, MAX_LOCAL_TASK_FAILURES, isLocal = true)
case LOCAL_N_REGEX(threads) => new TaskSchedulerImpl(sc, MAX_LOCAL_TASK_FAILURES, isLocal = true)
```

`local` 和 `local[N]` 模式下，**Task 失败 1 次就直接让 Job 失败，不会重试**。如果想在本地测试重试逻辑，要用 `local[N, F]` 的形式，比如 `local[4, 3]` 表示 4 个线程、最多失败 3 次。

### 5.2 FetchFailed：Stage 级别的重算

如果下游 Task 读 Shuffle 数据时，发现上游的 Shuffle 文件拿不到了（比如上游所在的 Executor 挂了），会报 **FetchFailed**。

这和普通的 Task 失败**完全不同**：

| | 普通 Task 失败 | FetchFailed |
|---|---|---|
| 问题在哪 | 这个 Task 自己 | **上游 Stage 的输出丢了** |
| 谁处理 | TaskSetManager 重试这个 Task | **DAGScheduler** 重新提交上游 Stage |
| 是否计入 `spark.task.maxFailures` | 是 | **否** |
| 上限 | `spark.task.maxFailures`（默认 4） | `spark.stage.maxConsecutiveAttempts`（默认 4） |

处理流程（`DAGScheduler.scala:2395` 的 `case FetchFailed`）：

1. 把当前 Stage 和丢失输出的上游 ShuffleMapStage 标记为失败，放入 `failedStages`
2. 注销丢失的 Shuffle 输出（这样上游重算时只算缺失的分区）
3. 延迟 **200ms**（`RESUBMIT_TIMEOUT`）后发送 `ResubmitFailedStages` 事件，重新提交

> 为什么要延迟 200ms？一台机器挂了，往往会让一批 Task 几乎同时报 FetchFailed。源码里只有**第一次**失败会安排重新提交（`noResubmitEnqueued` 判断，`DAGScheduler.scala:2485`），之后陆续到达的失败只是把 Stage 加入 `failedStages`。稍等 200ms，这些失败就会被合并成**一次**重新提交。

### 5.3 失败节点排除（Exclude On Failure）

如果某台机器硬盘有问题，Task 在上面反复失败，重试到这台机器上也没用。

`HealthTracker` 可以把频繁失败的 Executor / 节点暂时**排除**出调度范围。**默认不开启**，通过 `spark.excludeOnFailure.enabled=true` 开启。

> 历史上这个功能叫 blacklist，Spark 3.1 起改名为 exclude。

---

## 6. 推测执行（Speculation）

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-02-scheduling-2.webp" alt="慢任务被复制赛跑" width="960" height="640" loading="lazy" /><figcaption>慢任务被复制赛跑<span>AI 生成配图</span></figcaption></figure>

### 6.1 解决什么问题

一个 Stage 有 1000 个 Task，999 个都在 1 分钟内跑完了，有 1 个跑了 10 分钟还没完。整个 Stage 要等它。

如果原因是**那台机器有问题**（磁盘慢、负载高），换一台机器重跑这个 Task 可能 1 分钟就完了。这就是推测执行：**给慢 Task 再启动一个副本，谁先完成用谁的结果，另一个被杀掉**。

### 6.2 判断规则（4.2.0）

**默认不开启**，需要 `spark.speculation=true`。开启后，TaskScheduler 每 **100ms**（`spark.speculation.interval`）检查一次（`TaskSetManager.checkSpeculatableTasks`，`TaskSetManager.scala:1300`）：

```scala
// 1. 已完成的 Task 达到一定比例，才开始判断
minFinishedForSpeculation = max(floor(speculationQuantile * numTasks), 1)

// 2. 运行时间超过阈值的 Task，就是"慢 Task"
threshold = max(speculationMultiplier * 已完成 Task 的中位数耗时, minTimeToSpeculation)
```

| 配置 | 4.2.0 默认值 | 含义 |
|---|---|---|
| `spark.speculation` | `false` | 是否开启 |
| `spark.speculation.interval` | `100`（ms） | 检查间隔 |
| `spark.speculation.quantile` | **`0.9`** | 完成 90% 的 Task 后才开始推测 |
| `spark.speculation.multiplier` | **`3`** | 运行时间超过中位数 3 倍才算慢 |
| `spark.speculation.minTaskRuntime` | `100`（ms） | 最短运行时间门槛 |
| `spark.speculation.efficiency.enabled` | `true` | 结合数据处理速率判断，只推测真正低效的 Task |

> ⚠️ **注意版本差异**：很多文章、书里写的默认值是 `quantile = 0.75`、`multiplier = 1.5`，那是**较早版本**的默认值。**4.2.0 中分别是 0.9 和 3**，推测执行变得更保守了。

### 6.3 什么时候不该用

- **数据倾斜**：慢的原因是这个 Task 的数据量本来就大，换机器重跑一样慢，只会浪费资源。推测执行**解决不了数据倾斜**
- **输出不幂等**：如果 Task 会往外部系统写数据（比如直接写数据库），两个副本可能写两份
- **资源紧张的集群**：副本会占用额外资源

---

## 7. SchedulerBackend：通信层

| 实现 | 场景 |
|---|---|
| `LocalSchedulerBackend` | local 模式，Driver 和 Executor 在同一个 JVM |
| `CoarseGrainedSchedulerBackend` | 集群模式的基类，管理所有 Executor 的注册、空闲核数、Task 下发 |
| `StandaloneSchedulerBackend` | Spark 自带的 Standalone 集群 |
| `YarnClientSchedulerBackend` / `YarnClusterSchedulerBackend` | YARN |
| `KubernetesClusterSchedulerBackend` | Kubernetes |

"**Coarse-Grained（粗粒度）**"的意思是：Executor 是**长期运行的进程**，一次申请、多次复用，Task 只是在已有的 Executor 里以**线程**方式运行，而不是每个 Task 单独启动一个进程。（开启动态资源分配后，Executor 的数量会随负载增减，第 6 讲细说。）

`CoarseGrainedSchedulerBackend` 里有两种报价方式：

- `makeOffers()`：把**所有** Executor 的空闲资源一起报价，`isAllFreeResources = true`（`CoarseGrainedSchedulerBackend.scala:380`）
- `makeOffers(executorId)`：某个 Task 结束时，只报价**这一个** Executor 的资源，`isAllFreeResources = false`（第 415 行）

这正好对应 4.2 节里"全量资源报价"的概念。

---

## 8. 串起来：一个 Task 从生到死

```
用户线程          DAGScheduler           TaskSchedulerImpl          SchedulerBackend      Executor
   │  Action         │                         │                          │                  │
   ├──JobSubmitted──▶│ 切 Stage                 │                          │                  │
   │                 │ submitStage              │                          │                  │
   │                 │ submitMissingTasks       │                          │                  │
   │                 │  · 找缺失分区             │                          │                  │
   │                 │  · 算位置偏好             │                          │                  │
   │                 │  · 广播 taskBinary        │                          │                  │
   │                 ├──submitTasks(TaskSet)──▶ │ 建 TaskSetManager         │                  │
   │                 │                          │ 加入调度池                │                  │
   │                 │                          ├──reviveOffers──────────▶ │                  │
   │                 │                          │ ◀──resourceOffers(空闲核)─┤                  │
   │                 │                          │ 排序(FIFO/FAIR)           │                  │
   │                 │                          │ 按本地性分配(延迟调度)     │                  │
   │                 │                          ├──TaskDescription───────▶ ├──LaunchTask────▶ │ 执行
   │                 │                          │                          │ ◀──StatusUpdate──┤
   │                 │                          │ ◀──statusUpdate──────────┤                  │
   │                 │                          │ 成功/失败重试/推测         │                  │
   │                 │ ◀──CompletionEvent───────┤                          │                  │
   │                 │ Stage 完成？提交子 Stage   │                          │                  │
   │ ◀──Job 完成──────┤                          │                          │                  │
```

---

## 自测题

1. DAGScheduler 为什么用单线程事件循环？这样设计有什么代价？
2. 为什么 Task 的 RDD 和函数要通过广播发送，而不是直接放在每个 Task 里？
3. FIFO 调度下，两个 Job 同时提交，谁先跑？在 FIFO 下，一个大 Job 会如何影响后面的小 Job？
4. FAIR 调度能解决多个 Spark Application 之间抢资源的问题吗？为什么？
5. 一个 Task 的位置偏好是 A 机器，A 机器当前没有空闲核。在默认配置下，Spark 会怎么做？
6. 为什么 FetchFailed 不计入 `spark.task.maxFailures`？
7. 在 `local[4]` 模式下，一个 Task 抛异常会重试吗？如果想测试重试，应该怎么设置？
8. 开启推测执行能解决数据倾斜吗？为什么？
9. 在 4.2.0 的默认配置下，一个 1000 个 Task 的 Stage，至少要完成多少个 Task 才会开始推测？慢到什么程度才会被推测？

## 下一讲预告

第 3 讲：**Shuffle 原理**。Stage 之间的数据到底是怎么传的？为什么 Hash Shuffle 被淘汰了？`SortShuffleManager` 里的三种 Writer 分别在什么情况下被选用？
:::
