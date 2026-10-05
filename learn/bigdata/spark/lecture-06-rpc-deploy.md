---
title: "第 6 讲：RPC 与部署"
description: "RPC 框架与部署模式：RpcEndpoint、Driver 与 Executor 的通信、动态资源分配。"
bigdata: "spark"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-06-rpc-deploy-cover.webp"}]]
---

# 第 6 讲：RPC 与部署

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-06-rpc-deploy-cover.webp" alt="驾驶员与执行者的通信部署全景" width="1200" height="800" loading="eager" /><figcaption>驾驶员与执行者的通信部署全景<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`
> 配置默认值、类名、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；RPC 端点和动态资源分配经过实验验证（仓库 `experiments/07-deploy/`，使用 `local-cluster` 模式，Executor 是真实的独立进程）。

## 0. 本讲要回答的问题

1. Driver 和 Executor 之间的那些消息（注册、发 Task、汇报状态、心跳）是怎么传递的？
2. 执行 `spark-submit` 之后，到底发生了什么？client 模式和 cluster 模式有什么区别？
3. 一个 Executor 是怎么在集群上启动、注册、开始干活的？
4. Executor 挂了，Driver 怎么知道？
5. 动态资源分配是怎么决定加多少、减多少 Executor 的？

---

## 1. Spark 的 RPC 框架

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-06-rpc-deploy-1.webp" alt="消息胶囊连接端点收发请求" width="960" height="640" loading="lazy" /><figcaption>消息胶囊连接端点收发请求<span>AI 生成配图</span></figcaption></figure>

### 1.1 演进：从 Akka 到自研

| 时间 | 变化 | 提交 |
|---|---|---|
| 早期 | Driver / Executor 之间用 **Akka**（Actor 框架）通信 | — |
| 2015-04，**Spark 1.4** 起 | 抽象出 Spark 自己的 RPC 接口，逐步替换对 Akka 的直接依赖 | `[SPARK-6602] Replace direct use of Akka with Spark RPC interface` |
| 2016-01，**Spark 2.0** | **彻底移除 Akka**，只剩基于 Netty 的实现 | `[SPARK-7997] Remove Akka from Spark Core and Streaming` |

为什么要去掉 Akka：用户程序自己也可能依赖 Akka，版本冲突很麻烦；而 Spark 需要的其实只是"在进程之间可靠地收发消息"，没必要背一整个 Actor 框架。

### 1.2 四个核心抽象

| 抽象 | 作用 | 对应源码 |
|---|---|---|
| **RpcEnv** | RPC 环境：负责网络通信、注册端点、查找端点。实现类是 `NettyRpcEnv` | `rpc/RpcEnv.scala`、`rpc/netty/NettyRpcEnv.scala` |
| **RpcEndpoint** | 一个"服务"，处理收到的消息 | `rpc/RpcEndpoint.scala` |
| **RpcEndpointRef** | 指向某个端点（本地或远程）的引用，用来给它发消息 | `rpc/RpcEndpointRef.scala` |
| **Dispatcher** | 把收到的消息分发给对应端点的收件箱（`Inbox`） | `rpc/netty/Dispatcher.scala` |

**端点的两种处理方法**（`RpcEndpoint.scala:69、77`）：

```scala
def receive: PartialFunction[Any, Unit]                          // 处理 send 发来的单向消息，不回复
def receiveAndReply(context: RpcCallContext): PartialFunction[Any, Unit]  // 处理 ask 发来的请求，要 context.reply(...)
```

**引用的两种发送方式**（`RpcEndpointRef.scala:45、64`）：

```scala
ref.send(msg)                 // 发了就不管（fire-and-forget）
ref.ask[T](msg, timeout)      // 发请求，返回 Future[T]；askSync 会阻塞等结果
```

> 第 1 讲调试时遇到的 `localEndpoint.send(ReviveOffers)`、第 2 讲里的 `StatusUpdate`，都是这套接口。

### 1.3 消息是怎么被处理的

```
远程消息 ──Netty──▶ NettyRpcEnv ──▶ Dispatcher ──▶ 端点 A 的 Inbox ──▶ 消息循环线程 ──▶ A.receive(...)
本地消息 ─────────────────────────▶ Dispatcher ──▶ 端点 B 的 Inbox ──▶ ...
发往远程的消息 ──▶ Outbox（按目标地址排队，等连接建好后发送）──Netty──▶ 远端
```

- **共享消息循环**（`SharedMessageLoop`）：大多数端点共用一个名为 `dispatcher-event-loop` 的线程池。线程数默认 `max(2, 可用核数)`，可以用 `spark.rpc.netty.dispatcher.numThreads` 调整，也可以按角色分别设置（`spark.driver.rpc.netty.dispatcher.numThreads` / `spark.executor....`）（`MessageLoop.scala:115-120`）
- **独立消息循环**（`DedicatedMessageLoop`）：实现了 `IsolatedRpcEndpoint` 的端点有自己专用的线程池，不会被其他端点的消息堵住

**线程安全保证**：实现了 `ThreadSafeRpcEndpoint` 的端点，**同一时刻只处理一条消息**，前一条处理完才会处理下一条（`RpcEndpoint.scala:140` 起的注释）。所以端点内部的状态不需要加锁。

> 这和第 2 讲 DAGScheduler 的单线程事件循环是同一个思路：**用"串行处理消息"代替"加锁"**。

### 1.4 实验 A：Driver 上有哪些端点

在 `local-cluster` 模式下，用反射读出 Driver 的 Dispatcher 里注册的所有端点：

```
RpcEnv 实现：NettyRpcEnv
· AppClient                    与 Standalone Master 通信（申请 Executor）
· BlockManagerEndpoint1        Driver 自己的 BlockManager 的端点
· BlockManagerMaster           第 5 讲：全局块位置信息
· BlockManagerMasterHeartbeat  BlockManager 的心跳
· CoarseGrainedScheduler       第 2 讲：SchedulerBackend 的 Driver 端（Executor 注册、Task 下发、状态汇报）
· HeartbeatReceiver            接收 Executor 心跳，判断 Executor 是否存活
· MapOutputTracker             第 3 讲：Shuffle 输出位置
· OutputCommitCoordinator      协调 Task 的输出提交，避免推测执行的两个副本都提交结果
· endpoint-verifier            用于远程检查"某个名字的端点是否存在"
```

前面几讲的每个组件，在这里都对上了号：**它们之间就是靠这些 RPC 端点通信的**。

### 1.5 RPC 相关配置

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.network.timeout` | `120s` | 网络交互的总超时，很多超时配置都回退到它 |
| `spark.rpc.askTimeout` | 未设置时用 `spark.network.timeout` | `ask` 请求的超时（`RpcUtils.askRpcTimeout`） |
| `spark.rpc.message.maxSize` | `128`（MB） | 单条 RPC 消息的大小上限 |
| `spark.rpc.netty.dispatcher.numThreads` | 未设置时为 `max(2, 可用核数)` | 共享消息循环的线程数 |

---

## 2. 从 spark-submit 到 SparkContext

### 2.1 spark-submit 做了什么

`SparkSubmit.doSubmit` → `prepareSubmitEnvironment`（`deploy/SparkSubmit.scala:243`）→ `runMain`（第 964 行）。

核心是 `prepareSubmitEnvironment` 决定**要启动哪个主类（childMainClass）**：

| 部署模式 | 启动的主类 | Driver 在哪 |
|---|---|---|
| **client 模式**（任何集群管理器） | **你自己的 main 类**（第 727 行） | **提交命令所在的机器**，就在 spark-submit 这个 JVM 里 |
| YARN cluster 模式 | `org.apache.spark.deploy.yarn.YarnClusterApplication` | 集群里的 YARN ApplicationMaster 容器中 |
| K8s cluster 模式 | `org.apache.spark.deploy.k8s.submit.KubernetesClientApplication` | 集群里的 Driver Pod 中 |
| Standalone cluster 模式 | `ClientApp` 或 `RestSubmissionClientApp` | 集群里的某个 Worker 上 |

（常量定义在 `SparkSubmit.scala:1126-1131`）

**client 和 cluster 模式怎么选**：

- **client**：Driver 在本地，日志、`println` 直接打在终端上，适合**交互式使用和调试**（`spark-shell` 只能用 client 模式）。缺点是提交机器要一直在线，并且要和集群网络互通
- **cluster**：Driver 跑在集群里，提交完就可以断开，适合**生产作业**

> ⚠️ **Mesos 已经不支持了**：`[SPARK-44442] Remove Mesos support`（2023-09），**Spark 4.0.0 起移除**。4.2.0 的 `resource-managers/` 目录下只剩 `yarn` 和 `kubernetes`。网上还把 Mesos 列为可选集群管理器的资料都过时了。

### 2.2 SparkContext 的初始化顺序

`SparkContext` 构造时按这个顺序创建核心组件（`SparkContext.scala`）：

```
1. createSparkEnv          第 501 行   RpcEnv、BlockManager、MapOutputTracker、序列化器、内存管理器……
2. HeartbeatReceiver       第 590 行   注册心跳接收端点（必须在 createTaskScheduler 之前：Executor 在构造时就要查找它，见源码注释 SPARK-6640）
3. createTaskScheduler     第 600 行   根据 master URL 创建 TaskScheduler + SchedulerBackend（第 2 讲）
4. new DAGScheduler        第 603 行
5. taskScheduler.start()               SchedulerBackend 开始向集群管理器申请 Executor
6. ExecutorAllocationManager 第 679 行  如果开启了动态资源分配
```

> 这里就是第 2 讲"三层调度"在代码里诞生的地方。

---

## 3. Executor 的一生

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-06-rpc-deploy-2.webp" alt="执行者启动注册后才接任务" width="960" height="640" loading="lazy" /><figcaption>执行者启动注册后才接任务<span>AI 生成配图</span></figcaption></figure>

### 3.1 启动与注册

```
SchedulerBackend 向集群管理器申请资源
   ↓（YARN：ApplicationMaster + YarnAllocator 申请容器；K8s：ExecutorPodsAllocator 创建 Pod；Standalone：Master 让 Worker 启动）
集群管理器在某台机器上启动一个 JVM，主类是 CoarseGrainedExecutorBackend
   （YARN 上是 YarnCoarseGrainedExecutorBackend，K8s 上是 KubernetesExecutorBackend）
   ↓
CoarseGrainedExecutorBackend.onStart：
   向 Driver 的 "CoarseGrainedScheduler" 端点 ask RegisterExecutor(...)      CoarseGrainedExecutorBackend.scala:106
   ↓
Driver 端 CoarseGrainedSchedulerBackend 收到 RegisterExecutor：              CoarseGrainedSchedulerBackend.scala:249
   记录到 executorDataMap，回复 true，发出 SparkListenerExecutorAdded 事件
   （此时还不会给它派 Task）
   ↓
Executor 端收到 RegisteredExecutor：                                       CoarseGrainedExecutorBackend.scala:168
   创建真正干活的 Executor 对象（线程池、BlockManager 等）
   创建完成后，再给 Driver 发一条 LaunchedExecutor                           第 173 行
   ↓
Driver 端收到 LaunchedExecutor：                                           CoarseGrainedSchedulerBackend.scala:233
   把这个 Executor 的空闲核数设为总核数，调用 makeOffers(executorId) 开始派 Task
   ↓
之后 Executor 不断收到 LaunchTask(data) → executor.launchTask(...)          CoarseGrainedExecutorBackend.scala:181
Task 结束 → 发送 StatusUpdate 给 Driver                                     Driver 端第 168 行
```

> **为什么要多一次 `LaunchedExecutor` 握手**：注册成功不代表 Executor 已经能干活了，它还要创建线程池、初始化 BlockManager。等它准备好再告诉 Driver，Driver 才开始派 Task，避免 Task 发过去却没人接。
>
> 第 1 讲用的 `LocalSchedulerBackend` 把这些步骤都省略了：Driver 和 Executor 在同一个 JVM，直接调用方法。集群模式下，每一步都是一次 RPC。

### 3.2 心跳：Driver 怎么知道 Executor 还活着

- Executor 每隔 `spark.executor.heartbeatInterval`（默认 **10s**）向 Driver 的 `HeartbeatReceiver` 发一次心跳，顺带汇报正在运行的 Task 的指标（累加器、内存用量等）
- `HeartbeatReceiver` 定期检查（`spark.network.timeoutInterval`，默认 60s），如果某个 Executor 超过 **`spark.network.timeout`（默认 120s）** 没有心跳，就认为它已经丢失（`HeartbeatReceiver.scala:84-88`），通知 TaskScheduler 和 DAGScheduler 处理（第 2 讲的 `ExecutorLost`）

> 💡 第 1 讲的调试脚本里把 `spark.network.timeout` 和心跳间隔调得很大，就是为了防止停在断点上时被 Driver 判定为"Executor 已死"。

### 3.3 实验 C：Executor 真的是独立进程

```
Driver 进程：<pid>@<host>
Task 运行所在进程：<pid1>@<host>
Task 运行所在进程：<pid2>@<host>
Task 运行所在进程：<pid3>@<host>
Task 运行所在进程：<pid4>@<host>
```

`local-cluster[4,1,1024]` 启动了 4 个 Worker，每个 Worker 拉起一个 Executor 进程。16 个 Task 分布在 **4 个不同的 JVM 进程**里，和 Driver 进程也不同。

---

## 4. 动态资源分配

### 4.1 解决什么问题

静态分配（`--num-executors 100`）的问题：作业刚开始只需要 10 个 Executor，或者最后只剩几个慢 Task 时，剩下的 Executor 全在**空转占资源**；而作业高峰期又可能不够用。

**动态资源分配**：根据积压的 Task 自动**加** Executor，根据空闲情况自动**减** Executor。由 `ExecutorAllocationManager` 实现，默认**不开启**（`spark.dynamicAllocation.enabled=false`）。

### 4.2 扩容：指数增长

1. 有 Task 积压（等待调度）持续 `schedulerBacklogTimeout`（默认 **1s**），开始申请
2. 之后积压还在，每隔 `sustainedSchedulerBacklogTimeout`（默认同上，**1s**）再申请一轮
3. **每轮申请的数量翻倍**：1、2、4、8……（`ExecutorAllocationManager.scala:459`：`numExecutorsToAdd * 2`）
4. 上限是"**当前需要的最多 Executor 数**"（第 307 行）：

```scala
maxNeeded = ceil(运行中和等待中的 Task 数 × executorAllocationRatio / 每个 Executor 能同时跑的 Task 数)
```

再受 `maxExecutors`（默认不限）约束。

**为什么指数增长**：开始时不知道到底需要多少，先少量试探，避免一次申请过多；如果积压一直在，就快速放大，几轮之内就能达到需要的规模。

### 4.3 实验 D：观察目标 Executor 数

16 个 Task（每个睡 3 秒），每个 Executor 1 个核，`initialExecutors=0`：

```
[ 1.0s] 目标 Executor 数 = 0
[ 2.3s] 目标 Executor 数 = 1     +1
[ 3.3s] 目标 Executor 数 = 3     +2
[ 4.4s] 目标 Executor 数 = 7     +4
[ 5.4s] 目标 Executor 数 = 15    +8
[ 6.4s] 目标 Executor 数 = 16    封顶：16 个 Task ÷ 每个 Executor 1 核 = 16
...（Task 陆续完成，目标随之下降）
[48.6s] 目标 Executor 数 = 0
```

- **1、2、4、8 的翻倍**，大约每 1 秒一轮，与默认的 `sustainedSchedulerBacklogTimeout = 1s` 吻合
- 封顶在 16，正是上面 `maxNeeded` 公式的结果
- 实际只启动了 **4 个** Executor：这个本地集群总共只有 4 个 Worker，**目标数只是 Driver 的"期望"，能拿到多少取决于集群资源**

> 实验里还能看到：从申请到 Executor 真正注册上来，在这台机器上用了 30 秒左右。这说明动态扩容是有延迟的，**对很短的作业，动态分配不一定比静态分配快**。

### 4.4 缩容：空闲超时

- Executor 空闲超过 `executorIdleTimeout`（默认 **60s**）就会被移除
- **缓存了数据的 Executor** 用 `cachedExecutorIdleTimeout`，默认**无限长**：不会因为空闲而移除，否则缓存就丢了

实验 E（`executorIdleTimeout` 设为 5s）：Job 结束后，4 个 Executor 在空闲超时后被 Driver 陆续移除（`Executor killed by driver`），最终回到 0 个。

### 4.5 关键前提：Shuffle 数据怎么办

**问题**：Executor 被移除了，它写在本地磁盘上的 Shuffle 文件也就没人提供了（第 3 讲第 6 节）。下游来读，就会 FetchFailed。

所以开启动态分配时，**必须满足下面任一条件**，否则直接报错（`ExecutorAllocationManager.validateSettings`，第 212 行起）：

| 方案 | 配置 | 原理 |
|---|---|---|
| ① 外部 Shuffle 服务 | `spark.shuffle.service.enabled=true` | 节点上有独立的服务提供 Shuffle 文件，Executor 退出也不影响 |
| ② **Shuffle 追踪** | `spark.dynamicAllocation.shuffleTracking.enabled` | **还有 Shuffle 数据被需要的 Executor 不移除**，直到这些数据不再被使用（`shuffleTracking.timeout` 默认无限） |
| ③ 下线时迁移 Shuffle 块 | `spark.decommission.enabled` + `spark.storage.decommission.shuffleBlocks.enabled` | 移除前先把 Shuffle 块迁移到别的 Executor |
| ④ 可靠存储插件（实验性） | `spark.shuffle.sort.io.plugin.class` | 自定义的 ShuffleDataIO 把 Shuffle 数据写到可靠存储 |

> ✅ **4.2.0 中 `shuffleTracking.enabled` 默认是 `true`**，所以即使没有外部 Shuffle 服务（比如 K8s 上），开启动态分配也能直接工作。代价是：持有 Shuffle 数据的 Executor 可能迟迟不能被回收。

### 4.6 配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.dynamicAllocation.enabled` | `false` | 是否开启 |
| `spark.dynamicAllocation.minExecutors` | `0` | 下限 |
| `spark.dynamicAllocation.maxExecutors` | `Int.MaxValue` | 上限（生产环境**一定要设**，否则一个大作业可能占满集群） |
| `spark.dynamicAllocation.initialExecutors` | 同 `minExecutors` | 初始数量 |
| `spark.dynamicAllocation.schedulerBacklogTimeout` | `1s` | 积压多久开始申请 |
| `spark.dynamicAllocation.sustainedSchedulerBacklogTimeout` | 同上 | 持续积压时每轮申请的间隔 |
| `spark.dynamicAllocation.executorIdleTimeout` | `60s` | 空闲多久移除 |
| `spark.dynamicAllocation.cachedExecutorIdleTimeout` | 无限 | 有缓存的 Executor 空闲多久移除 |
| `spark.dynamicAllocation.executorAllocationRatio` | `1.0` | 调小可以少申请一些（比如 0.5 表示只按一半的需求申请） |
| `spark.dynamicAllocation.shuffleTracking.enabled` | `true` | Shuffle 追踪 |

---

## 自测题

1. `send` 和 `ask` 有什么区别？分别对应端点的哪个处理方法？
2. 为什么实现了 `ThreadSafeRpcEndpoint` 的端点内部不需要加锁？这和 DAGScheduler 的设计有什么共同点？
3. 实验 A 列出的端点里，哪些在前几讲出现过？各自负责什么？
4. 用 `spark-submit --deploy-mode cluster` 提交到 YARN，你的 main 方法在哪台机器上执行？client 模式呢？
5. 为什么 `HeartbeatReceiver` 要在 TaskScheduler 之前创建？
6. 一个 Executor 进程被 `kill -9` 了，Driver 最慢多久能发现？由哪个配置决定？
7. 动态资源分配为什么每轮申请的数量要翻倍，而不是一次申请到位？
8. 实验中目标 Executor 数到了 16，为什么实际只启动了 4 个？
9. 在 K8s 上没有外部 Shuffle 服务，开启动态资源分配会报错吗？为什么？
10. 网上有资料说 Spark 支持 Mesos 作为集群管理器，在 4.2.0 里还成立吗？

## 下一讲预告

第 7 讲：**容错机制**。把前几讲散落的容错知识串起来：Task 重试、Stage 重算、Executor 丢失、Driver 失败，以及 Checkpoint 是怎么切断血缘的。
:::
