---
title: "L1 练习 1：跟踪 SparkPi —— 一个 Job 的一生"
description: "在 IDEA 里断点跟踪 SparkPi，从 rdd.reduce() 一路跟到 Executor 执行 Task 再回到 Driver。"
bigdata: "spark"
lesson: "e4"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/l1-sparkpi-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/l1-sparkpi-cover.webp" alt="Yui 和 Kai 追踪一个 Spark Job 的全流程" width="1200" height="800" loading="eager" /><figcaption>Yui 和 Kai 追踪一个 Spark Job 的全流程<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 目标：在 IDEA 中断点跟踪 `SparkPi`，从 `rdd.reduce()` 一路跟到 Executor 执行 Task，再跟回结果返回 Driver，最后画出时序图。
> 源码版本：`v4.2.0`，下文所有 `文件:行号` 均已核实。
> 路径缩写：`core/…` = `core/src/main/scala/org/apache/spark/`

---

## 第 0 步：先看"标准答案" —— 真实线程栈

<figure class="ai-figure"><img src="/bigdata-img/spark/l1-sparkpi-1.webp" alt="多线程通过队列和消息接力完成 Job" width="960" height="640" loading="lazy" /><figcaption>多线程通过队列和消息接力完成 Job<span>AI 生成配图</span></figcaption></figure>

我在 Job 运行中途抓了一次线程栈（完整内容见 [expected-output.txt](https://github.com/DaemonforY/spark-source-notes/blob/main/experiments/01-thread-stacks/expected-output.txt)），精简后如下：

```
===== THREAD: main =====                                   ← 用户线程，在这里阻塞等结果
  at ThreadUtils$.awaitReady(ThreadUtils.scala:417)
  at DAGScheduler.runJob(DAGScheduler.scala:1052)          ← 阻塞点
  at SparkContext.runJob(SparkContext.scala:2496)
  at RDD.$anonfun$reduce$1(RDD.scala:1164)
  at RDD.reduce(RDD.scala:1146)

===== THREAD: dag-scheduler-event-loop =====               ← 调度线程，此刻空闲（已处理完提交事件）
  at EventLoop$$anon$1.run(EventLoop.scala:48)

===== THREAD: Executor task launch worker for task 0.0 in stage 0.0 (TID 0) =====
  at <你的 map 函数>                                         ← 用户代码在 Executor 线程执行
  at RDD.$anonfun$reduce$2(RDD.scala:1150)                 ← reducePartition
  at ResultTask.runTask(ResultTask.scala:93)
  at Task.run(Task.scala:147)
  at Executor$TaskRunner.run(Executor.scala:897)
```

**核心认知**：一个 Job 横跨**多个线程**，线程之间通过**事件队列 / RPC 消息 / 线程池**交接。
所以单看任何一个线程栈都是断的，**必须在交接点两侧都打断点**。本练习的关键就是找到这些交接点。

---

## 第 1 步：配置 IDEA（一次性）

1. **安装 Scala 插件**：IDEA → Settings → Plugins → Marketplace 搜 `Scala`（JetBrains 官方）→ Install → 重启
2. **导入项目**：File → Open → 选 `opensource-codes/spark/pom.xml` → Open as Project
3. **设置 JDK**：File → Project Structure → Project → SDK 选 **17**（temurin-17.0.19），Language level 17
4. **Maven 设置**：Settings → Build Tools → Maven
   - User settings file 勾 Override，选 `spark/aliyun-settings.xml`（加速依赖下载）
   - Importing → JDK for importer 选 17
5. **Maven Profiles**：右侧 Maven 面板 → Profiles，勾选 `hive`、`hive-thriftserver`
6. 等待索引完成（第一次要 10–20 分钟，右下角有进度条）

> 常见问题：若 IDEA 提示部分 generated sources 缺失，因为我们已经用 Maven 完整编译过，一般不影响阅读和调试。

## 第 2 步：配置远程调试

**为什么用远程调试，而不是直接在 IDEA 里 Run SparkPi？**
直接 Run 会遇到 `provided` 依赖缺失、Java 17 模块 `--add-opens` 参数缺失等一堆问题；
用 `spark-submit` 启动再让 IDEA 连上去，**和真实运行方式完全一致**，零配置坑。

1. IDEA → Run → Edit Configurations → `+` → **Remote JVM Debug**
2. Name: `SparkPi Debug`，Host: `localhost`，Port: `5005`，Module classpath 选 `spark-core_2.13`
3. 保存

## 第 3 步：打断点

> ⚠️ **断点设置**：右键断点 → Suspend 选 **Thread**（而不是 All）。
> 否则一个断点会冻结所有线程，可能导致其他线程超时，也看不清多线程的交接。

按**执行顺序**排列，🔀 表示线程交接点：

### 阶段 A：提交 Job（main 线程）

| # | 位置 | 观察什么 |
|---|---|---|
| A1 | `examples/src/main/scala/org/apache/spark/examples/SparkPi.scala:38` | `.reduce(_ + _)`：action 算子触发 Job，之前的 `parallelize`、`map` 都只是构建 RDD 血缘（惰性求值） |
| A2 | `core/rdd/RDD.scala:1164` | `sc.runJob(this, reducePartition, mergeResult)`：看两个闭包，`reducePartition` 在 **Executor** 上执行（分区内归约），`mergeResult` 在 **Driver** 上执行（汇总各分区结果） |
| A3 | `core/SparkContext.scala:2496` | `dagScheduler.runJob(...)`：注意上一行的 `clean(func)`，即 ClosureCleaner，确保闭包可序列化 |
| A4 | `core/scheduler/DAGScheduler.scala:1022` | 🔀 **交接点 1**：`eventProcessLoop.post(JobSubmitted(...))`，把事件放进队列，交给 `dag-scheduler-event-loop` 线程。注意上一行创建了 `JobWaiter` |
| A5 | `core/scheduler/DAGScheduler.scala:1052` | `ThreadUtils.awaitReady(waiter.completionFuture, ...)`：main 线程**在这里阻塞**，直到所有 Task 完成 |

### 阶段 B：切分 Stage、提交 Task（dag-scheduler-event-loop 线程）

| # | 位置 | 观察什么 |
|---|---|---|
| B1 | `core/scheduler/DAGScheduler.scala:3527` | `doOnReceive` 的 `case JobSubmitted`：事件循环收到事件 |
| B2 | `core/scheduler/DAGScheduler.scala:1400` | `handleJobSubmitted`：调用 `createResultStage`（704 行）创建最终 Stage |
| B3 | `core/scheduler/DAGScheduler.scala:728` | `getOrCreateParentStages`：SparkPi 没有 Shuffle，所以**没有父 Stage**，整个 Job 只有 1 个 ResultStage |
| B4 | `core/scheduler/DAGScheduler.scala:1540` | `submitStage`：递归地先提交缺失的父 Stage（`getMissingParentStages`，837 行），这里为空 |
| B5 | `core/scheduler/DAGScheduler.scala:1635` | `submitMissingTasks`：为每个分区创建一个 `ResultTask`（slices=2，所以 2 个 Task），并序列化 RDD 和闭包，广播出去 |
| B6 | `core/scheduler/DAGScheduler.scala:1821` | `taskScheduler.submitTasks(new TaskSet(...))`：从 DAGScheduler（**逻辑调度**）交给 TaskScheduler（**物理调度**） |
| B7 | `core/scheduler/TaskSchedulerImpl.scala:243` | `submitTasks`：创建 `TaskSetManager`，加入调度池（FIFO/FAIR） |
| B8 | `core/scheduler/TaskSchedulerImpl.scala:284` | `backend.reviveOffers()` |
| B9 | `core/scheduler/local/LocalSchedulerBackend.scala:171` | 🔀 **交接点 2**：`localEndpoint.send(ReviveOffers)`，发 RPC 消息，交给 RPC 的 dispatcher 线程 |

### 阶段 C：分配资源、启动 Task（RPC dispatcher 线程）

| # | 位置 | 观察什么 |
|---|---|---|
| C1 | `core/scheduler/local/LocalSchedulerBackend.scala:71` | `case ReviveOffers`：Endpoint 收到消息 |
| C2 | `core/scheduler/local/LocalSchedulerBackend.scala:101` | `scheduler.resourceOffers(offers, true)`：用空闲 CPU 核（`WorkerOffer`）去匹配待运行的 Task，这里涉及 **Locality（数据本地性）** 调度，入口在 `TaskSchedulerImpl.scala:512` |
| C3 | `core/scheduler/local/LocalSchedulerBackend.scala:103` | 🔀 **交接点 3**：`executor.launchTask(...)`，把 Task 丢进 Executor 的线程池 |

### 阶段 D：执行 Task（Executor task launch worker 线程）

| # | 位置 | 观察什么 |
|---|---|---|
| D1 | `core/executor/Executor.scala:551` | `launchTask`：创建 `TaskRunner` 并提交到线程池 |
| D2 | `core/executor/Executor.scala:806` | `TaskRunner.run`：反序列化 Task，830 行向 Driver 汇报 `RUNNING` 状态 |
| D3 | `core/executor/Executor.scala:888` | `task.run(...)` |
| D4 | `core/scheduler/ResultTask.scala:93` | `func(context, rdd.iterator(partition, context))`：**最关键的一行**，`func` 就是 A2 里的 `reducePartition` |
| D5 | `core/rdd/RDD.scala:334` | `iterator`：先查缓存 / Checkpoint，没有就调用 `compute` |
| D6 | `core/rdd/MapPartitionsRDD.scala:56` | `compute`：执行你的 `map` 函数，并调用父 RDD 的 `iterator` |
| D7 | `core/rdd/ParallelCollectionRDD.scala:101` | `compute`：最底层数据源，返回分区内的数据迭代器 |

> 💡 在 D6、D7 两个 `compute` 上观察：RDD 的计算是 **从下游向上游"拉"** 的（迭代器嵌套），数据是 **从上游向下游"流"** 的。这就是 **Pipeline（流水线）**：同一 Stage 内多个窄依赖算子被串成一个迭代器链，不会落盘、也不会产生中间集合。

### 阶段 E：结果返回（跨越多个线程回到 main）

| # | 位置 | 观察什么 |
|---|---|---|
| E1 | `core/executor/Executor.scala:1029` | `execBackend.statusUpdate(taskId, TaskState.FINISHED, serializedResult)`：汇报结果 |
| E2 | `core/scheduler/local/LocalSchedulerBackend.scala:74` | `case StatusUpdate`：Driver 端收到状态更新，同时释放 CPU 核，再次 `reviveOffers` |
| E3 | `core/scheduler/TaskSchedulerImpl.scala:808` | 🔀 **交接点 4**：`taskResultGetter.enqueueSuccessfulTask(...)`，交给 `task-result-getter` 线程池去反序列化结果（大结果可能要从 BlockManager 远程拉取） |
| E4 | `core/scheduler/TaskSetManager.scala:812` | `handleSuccessfulTask`：标记 Task 成功，929 行调用 `dagScheduler.taskEnded` |
| E5 | `core/scheduler/DAGScheduler.scala:419` | 🔀 **交接点 5**：`eventProcessLoop.post(CompletionEvent(...))`，又回到 DAGScheduler 事件循环 |
| E6 | `core/scheduler/DAGScheduler.scala:2203` | `handleTaskCompletion`，2319 行 `job.listener.taskSucceeded(...)` |
| E7 | `core/scheduler/JobWaiter.scala:66` | `resultHandler(index, result)`：这就是 A2 里的 `mergeResult`！所有 Task 完成后 `completionFuture` 完成 |
| E8 | 回到 A5 `DAGScheduler.scala:1052` | main 线程被唤醒，`reduce` 返回结果，打印 Pi |

## 第 4 步：开始调试

1. 终端运行：
   ```bash
   SPARK_HOME=<你的 Spark 源码目录> ./setup/debug-sparkpi.sh
   ```
   看到 `Listening for transport dt_socket at address: 5005` 后，JVM 会停住等你连接
2. IDEA 中选择 `SparkPi Debug` 点 Debug 🐞
3. 程序会停在 A1。按 **F9（Resume）** 跳到下一个断点，在断点之间用 **F7（Step Into）/ F8（Step Over）** 细看
4. **多用 Debug 面板左侧的 Threads 视图**：每到一个 🔀 交接点，注意观察断点停在了哪个线程

> 调试脚本里把 `spark.network.timeout` 和心跳间隔调得很大，所以你停在断点上慢慢看不会超时。

---

## 第 5 步：时序图（先自己画，再对照）

<figure class="ai-figure"><img src="/bigdata-img/spark/l1-sparkpi-2.webp" alt="Job 时序像接力赛一样跨线程推进" width="960" height="640" loading="lazy" /><figcaption>Job 时序像接力赛一样跨线程推进<span>AI 生成配图</span></figcaption></figure>
:::

<Mermaid code="c2VxdWVuY2VEaWFncmFtCiAgICBhdXRvbnVtYmVyCiAgICBwYXJ0aWNpcGFudCBNIGFzIG1haW4g57q/56iLPGJyLz4o55So5oi35Luj56CBKQogICAgcGFydGljaXBhbnQgRCBhcyBkYWctc2NoZWR1bGVyLWV2ZW50LWxvb3A8YnIvPihEQUdTY2hlZHVsZXIpCiAgICBwYXJ0aWNpcGFudCBUIGFzIFRhc2tTY2hlZHVsZXJJbXBsCiAgICBwYXJ0aWNpcGFudCBCIGFzIExvY2FsRW5kcG9pbnQ8YnIvPihSUEMgZGlzcGF0Y2hlcikKICAgIHBhcnRpY2lwYW50IEUgYXMgRXhlY3V0b3Ig57q/56iL5rGgPGJyLz4oVGFza1J1bm5lcikKICAgIHBhcnRpY2lwYW50IFIgYXMgdGFzay1yZXN1bHQtZ2V0dGVyCgogICAgTS0+Pk06IHJkZC5yZWR1Y2UoKSDihpIgc2MucnVuSm9iKCkKICAgIE0tPj5EOiBwb3N0KEpvYlN1Ym1pdHRlZCkg8J+UgAogICAgTm90ZSBvdmVyIE06IGF3YWl0UmVhZHkoKSDpmLvloZ4KICAgIEQtPj5EOiBoYW5kbGVKb2JTdWJtaXR0ZWQ8YnIvPmNyZWF0ZVJlc3VsdFN0YWdlIOKGkiBzdWJtaXRTdGFnZQogICAgRC0+PkQ6IHN1Ym1pdE1pc3NpbmdUYXNrczxici8+KOavj+S4quWIhuWMuuS4gOS4qiBSZXN1bHRUYXNrKQogICAgRC0+PlQ6IHN1Ym1pdFRhc2tzKFRhc2tTZXQpCiAgICBULT4+VDog5Yib5bu6IFRhc2tTZXRNYW5hZ2VyIOWKoOWFpeiwg+W6puaxoAogICAgVC0+PkI6IHJldml2ZU9mZmVycyDihpIgc2VuZChSZXZpdmVPZmZlcnMpIPCflIAKICAgIEItPj5UOiByZXNvdXJjZU9mZmVycyjnqbrpl7LmoLgpCiAgICBULS0+PkI6IOWIhumFjeWlveeahCBUYXNrRGVzY3JpcHRpb24KICAgIEItPj5FOiBleGVjdXRvci5sYXVuY2hUYXNrIPCflIAKICAgIEUtPj5FOiBUYXNrUnVubmVyLnJ1biDihpIgVGFzay5ydW48YnIvPuKGkiBSZXN1bHRUYXNrLnJ1blRhc2s8YnIvPuKGkiByZGQuaXRlcmF0b3Ig4oaSIGNvbXB1dGUgKHBpcGVsaW5lKQogICAgRS0+PkI6IHN0YXR1c1VwZGF0ZShGSU5JU0hFRCwg57uT5p6cKQogICAgQi0+PlQ6IHN0YXR1c1VwZGF0ZQogICAgVC0+PlI6IGVucXVldWVTdWNjZXNzZnVsVGFzayDwn5SACiAgICBSLT4+VDogaGFuZGxlU3VjY2Vzc2Z1bFRhc2sKICAgIFQtPj5EOiB0YXNrRW5kZWQg4oaSIHBvc3QoQ29tcGxldGlvbkV2ZW50KSDwn5SACiAgICBELT4+RDogaGFuZGxlVGFza0NvbXBsZXRpb248YnIvPuKGkiBKb2JXYWl0ZXIudGFza1N1Y2NlZWRlZDxici8+4oaSIG1lcmdlUmVzdWx0CiAgICBELS0+Pk06IGNvbXBsZXRpb25GdXR1cmUg5a6M5oiQ77yM5ZSk6YaSCiAgICBNLT4+TTog5omT5Y2wIFBpIGlzIHJvdWdobHkgMy4xNC4uLgo=" />
::: v-pre
---

## 第 6 步：思考题（带着问题调试，答案在源码里）

1. **为什么 DAGScheduler 用单线程事件循环，而不是直接加锁调用？**（提示：看 `DAGSchedulerEventProcessLoop`，思考状态一致性）
2. `reducePartition` 和 `mergeResult` 分别在哪个 JVM / 线程执行？如果换成 `collect()`，结果太大会怎样？（提示：`spark.driver.maxResultSize`，以及 E3 里 `IndirectTaskResult` 的处理）
3. 在 B5 `submitMissingTasks` 里，Task 的二进制内容（RDD + 闭包）是怎么发给 Executor 的？为什么要 **broadcast** 而不是每个 Task 里都带一份？
4. 把 `slices` 参数改成 4，Task 数量、Stage 数量分别怎么变？
5. **进阶**：把 SparkPi 改成先 `map(i => (i % 10, 1)).reduceByKey(_ + _).collect()`，重新跟踪 B3、B4。现在有几个 Stage？`ShuffleMapTask` 和 `ResultTask` 有什么区别？

## 完成标准

- [ ] 所有断点都至少命中过一次，并能说出当时在哪个线程
- [ ] 能不看文档说出 5 个 🔀 线程交接点
- [ ] 自己画出时序图（可以参考上面的，但要自己理解后重画）
- [ ] 回答上面 5 道思考题中的至少 3 道
- [ ] 写一篇博客：《一个 Spark Job 的一生》
:::
