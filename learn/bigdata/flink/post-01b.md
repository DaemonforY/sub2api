---
title: "Flink 2.x 源码精读（一·下）：为什么 Flink 的算子不需要加锁？"
description: "open、processElement、定时器、Checkpoint 快照、Checkpoint 完成通知，我在一个算子的 6 个回调里打印了线程名，结果全是同一个线程。顺着这个现象，读懂 Flink 2.3 从调度、部署到 Task 线程和 Mailbox 循环的全过程。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-01b-cover.webp"}]]
---

# Flink 2.x 源码精读（一·下）：为什么 Flink 的算子不需要加锁？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01b-cover.webp" alt="Yui和Kai在单线程数据流水线上协作理解Flink作业执行" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在单线程数据流水线上协作理解Flink作业执行<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`
> 上篇：《同一份 WordCount，为什么 IDE 里 3 个 Task，集群上只有 2 个？》

---

写过 Flink 作业的人，大概都写过这样的代码：在 `processElement` 里更新状态，在定时器 `onTimer` 里读状态，在 Checkpoint 时把状态快照下来。

这三件事触发的时机完全不同：数据什么时候来不知道，定时器什么时候到期不知道，Checkpoint 什么时候开始也不知道。按照多线程编程的常识，它们访问同一份状态，**应该要加锁**。

可是 Flink 的文档里从来没有让你加锁。

我做了一个实验：在一个算子的 6 个回调里，分别打印当前线程的名字。本机实测输出：

```
keyed-process.initializeState      thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
keyed-process.open                 thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
keyed-process.processElement       thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
keyed-process.snapshotState        thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
keyed-process.notifyCheckpointComplete thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
keyed-process.onTimer              thread = keyed-process -> chained-map -> sink: Writer (1/1)#0
```

**全是同一个线程。** 同一时刻只有一件事在执行，所以不需要锁。

这一篇就来回答，Flink 是怎么做到的。我们接着上篇，从 JobGraph 出发，一直走到这个线程里的 `processElement()`：

1. JobGraph 怎么展开成 ExecutionGraph？
2. JobManager 怎么申请资源、把 Task 部署到 TaskManager？
3. Task 线程启动以后做了什么？
4. **Mailbox 模型**是怎么让所有回调都在一个线程里执行的？
5. 一条数据在 Task 里经过了哪些方法，才到达你的 `processElement()`？

---

## 一、JobGraph → ExecutionGraph：按并行度展开

上篇讲到，JobGraph 是在 JobMaster 创建调度器时生成的（`RT/runtime/jobmaster/JobMaster.java:406`，`createScheduler()` 在 `:423`）。紧接着，调度器的父类在构造时就把 JobGraph 展开成了 ExecutionGraph：

```
SchedulerBase 构造函数                       RT/runtime/scheduler/SchedulerBase.java:255
 └ createAndRestoreExecutionGraph()           :407
    └ DefaultExecutionGraphBuilder.buildGraph()  RT/runtime/executiongraph/DefaultExecutionGraphBuilder.java:77
       ├ executionGraph.attachJobGraph(...)      :199
       └ executionGraph.enableCheckpointing(...) :312   ← CheckpointCoordinator 在这里创建
```

`attachJobGraph()`（`RT/runtime/executiongraph/DefaultExecutionGraph.java:867`）为每个 JobVertex 创建一个 `ExecutionJobVertex`，再**按并行度展开**成多个 `ExecutionVertex`。每个 `ExecutionVertex` 持有一个 `Execution`，代表一次执行尝试：Task 失败重启时，会创建新的 `Execution`，所以 Web UI 上 Task 名字的末尾有 `#0`、`#1` 这样的编号。

【配图 1：JobGraph 到 ExecutionGraph 的展开。左侧 3 个 JobVertex，右侧每个 JobVertex 展开成 N 个 ExecutionVertex，每个 ExecutionVertex 下面挂着一个 Execution（#0）】

如果作业是从 Checkpoint 或 Savepoint 恢复的，状态也是在这一步分配给各个并行实例的，第三讲（State Backend）会详细讲。

---

## 二、调度：以 Pipelined Region 为单位申请资源

### 2.1 先理解 JobManager 的线程模型

读 JobManager 的代码之前，要先知道一件事：`Dispatcher`、`JobMaster`、`ResourceManager`、`TaskExecutor` 都继承自 `RpcEndpoint`（`flink-rpc/flink-rpc-core/.../rpc/RpcEndpoint.java`）。

每个 `RpcEndpoint` 都有一个**单线程的主线程执行器**（`getMainThreadExecutor()`，`:357`），它的 RPC 方法都在这个线程上串行执行，所以这些组件的内部状态也不需要加锁。耗时的操作会被扔到其他线程池里，完成后再通过 `CompletableFuture` 切回主线程。

所以 JobManager 的代码里到处是 `CompletableFuture` 的链式调用。读代码时，要时刻注意每个 lambda 是在哪个线程上执行的。

你会发现，**"一个组件一个线程、所有操作串行执行"** 这个思路，和后面 Task 的 Mailbox 模型是一样的。

### 2.2 开始调度

JobMaster 启动后，`onStart()`（`JobMaster.java:482`）→ `startJobExecution()`（`:1172`）→ `schedulerNG.startScheduling()`（`:1273`）。之后的调用链：

| 步骤 | 位置 | 做什么 |
|---|---|---|
| `SchedulerBase.startScheduling()` | `SchedulerBase.java:669` | 启动所有 OperatorCoordinator，比如 Source 的 `SourceCoordinator`（第八讲） |
| `DefaultScheduler.startSchedulingInternal()` | `RT/runtime/scheduler/DefaultScheduler.java:249` | |
| `PipelinedRegionSchedulingStrategy.startScheduling()` | `RT/runtime/scheduler/strategy/PipelinedRegionSchedulingStrategy.java:182` | 以 **Pipelined Region** 为单位调度 |
| `scheduleRegion()` | `:282` | 为一个 region 里的所有 Task 申请资源并部署 |
| `DefaultExecutionDeployer.allocateSlotsAndDeploy()` | `RT/runtime/scheduler/DefaultExecutionDeployer.java:90` | 先申请 slot（`:121`），**等所有 slot 都到位后再统一部署**（`:151`） |

**Pipelined Region** 是一组通过流水线方式交换数据的 Task，它们必须同时运行。流作业里的数据都是流水线式传递的，所以通常整个作业就是一个 region。它同时也是故障恢复的单位，第六讲会讲到"一个 Task 失败，为什么有时只重启一部分"。

**为什么要等所有 slot 都到位再部署？** 一个 region 里的 Task 必须同时运行。如果先部署一部分，剩下的却申请不到资源，已经部署的那部分只能空等，还白白占着资源。

### 2.3 slot 从哪里来

申请请求从 JobMaster 的 `SlotPool` 发给 `ResourceManager`，ResourceManager 再通知某个 TaskExecutor，由 TaskExecutor 把 slot **直接提供给 JobMaster**（`JobMaster.offerSlots()`，`:741`）。SlotPool 拿到 slot 后，分配给正在等待的请求。

在 IDE 里运行时，这些组件都在同一个 JVM 里，但它们之间走的仍然是同一套 RPC 接口。

---

## 三、部署：把 Task 发给 TaskManager

slot 全部到位后，对每个 Task 调用 `deployTaskSafe()`（`DefaultExecutionDeployer.java:320`），最终来到 `Execution.deploy()`（`RT/runtime/executiongraph/Execution.java:572`）：

1. 状态机从 `SCHEDULED` 变为 `DEPLOYING`；
2. 生成一个 **TaskDeploymentDescriptor**（简称 TDD）：里面有这个 Task 要运行的算子链配置、要读取哪些上游的数据、要往哪里写数据、要恢复哪些状态；
3. 通过 RPC 把 TDD 发给 TaskExecutor：`taskManagerGateway.submitTask(deploymentDescriptor, rpcTimeout)`（`:656`）。

TaskExecutor 收到后（`RT/runtime/taskexecutor/TaskExecutor.java:660`）：

```
校验 slot 确实分配给了这个作业
 → new Task(...)                     :837
 → taskSlotTable.addTask(task)       :877
 → task.startTaskThread()            :883   ← 每个 Task 一个独立线程
```

**每个 Task 一个线程。** 开头实验里看到的线程名 `keyed-process -> chained-map -> sink: Writer (1/1)#0`，就是这个线程的名字：算子链的名字，加上并行实例编号 `(1/1)`，再加上执行尝试编号 `#0`。

---

## 四、Task 线程启动以后

Task 线程的入口是 `Task.run()`，它调用 `doRun()`（`RT/runtime/taskmanager/Task.java:585`）。这是 TaskManager 上最重要的方法之一，按顺序做了这些事：

1. 创建用户代码类加载器（下载你的 jar 包）；
2. `setupPartitionsAndGates()`（`:676`）：初始化这个 Task 的输出（`ResultPartition`）和输入（`InputGate`），并注册到网络栈（第四讲）；
3. `loadAndInstantiateInvokable()`（`:756`）：用反射创建真正执行逻辑的对象。流作业里是 `StreamTask` 的某个子类：Source 用 `SourceOperatorStreamTask`，单输入用 `OneInputStreamTask`，双输入（比如 join）用 `TwoInputStreamTask`；
4. `restoreAndInvoke()`（`:944`）：

```java
transitionState(DEPLOYING, INITIALIZING);   // 汇报给 JobManager
invokable.restore();                        // 恢复状态、打开算子
transitionState(INITIALIZING, RUNNING);     // 汇报给 JobManager → Web UI 变成 RUNNING
invokable.invoke();                         // 主循环，直到数据处理完或被取消
```

状态的每一次变化，都会通过 RPC 汇报给 `JobMaster.updateTaskExecutionState()`（`JobMaster.java:537`），驱动 ExecutionGraph 里的状态机。

### 4.1 `restore()`：把算子链组装起来

`StreamTask.restoreInternal()`（`RT/streaming/runtime/tasks/StreamTask.java:792`）：

- 创建 `OperatorChain`（`:806`）：按配置把链上的算子逐个实例化，并用"链内输出"把它们串起来（第六节会看到它在运行时的样子）；
- 依次对每个算子调用 `initializeState()`（恢复状态）和 `open()`（`:876`）。你写的 `RichFunction.open()` 就是在这里被调用的。

开头实验输出的前两行，`initializeState` 和 `open`，就发生在这一步。

### 4.2 `invoke()`：进入主循环

`StreamTask.invoke()`（`:945`）的核心只有一行：`runMailboxLoop()`（最终调用到 `:1022` 的 `mailboxProcessor.runMailboxLoop()`）。

**这就是 Task 线程的一生：恢复，然后进入 Mailbox 循环，直到数据结束。**

---

## 五、Mailbox：为什么不需要加锁

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01b-1.webp" alt="Mailbox循环让数据、定时器和快照在同一Task线程有序执行" width="960" height="640" loading="lazy" /><figcaption>Mailbox循环让数据、定时器和快照在同一Task线程有序执行<span>AI 生成配图</span></figcaption></figure>

### 5.1 一个线程要做的所有事

一个 Task 线程，要处理好几类事情：

- **处理数据**：这是它的本职工作；
- **触发定时器**：比如窗口到点了、`onTimer` 要执行了；
- **执行 Checkpoint**：做快照；
- **处理外部通知**：比如 JobManager 发来"Checkpoint 已完成"。

早期的 Flink 用一把大锁（checkpoint lock）让这些线程互斥：处理数据的线程、定时器线程、Checkpoint 线程，谁要动状态都得先拿锁。这样既容易出错，又影响性能。

**Mailbox 模型的做法是：所有这些事情，都放到同一个线程里排队执行。**

### 5.2 主循环

`RT/streaming/runtime/tasks/mailbox/MailboxProcessor.java:214`：

```java
while (isNextLoopPossible()) {
    // The blocking `processMail` call will not return until default action is available.
    processMail(localMailbox, false);                 // 1. 先处理邮箱里的"邮件"
    if (isNextLoopPossible()) {
        mailboxDefaultAction.runDefaultAction(        // 2. 再处理一批数据
                mailboxController);
    }
}
```

两类工作：

| | 是什么 | 例子 |
|---|---|---|
| **默认动作**（default action） | 处理输入数据 | `StreamTask.processInput()`，在构造 `MailboxProcessor` 时注册（`StreamTask.java:421`） |
| **邮件**（mail） | 其他所有事情 | 定时器回调、Checkpoint、Checkpoint 完成通知、异步操作的回调 |

其他线程**不能直接执行**这些事情，只能把它们包装成一封"邮件"投进邮箱，由 Task 线程在循环里取出来执行。

【配图 2：Mailbox 模型示意。中间一个 Task 线程在循环里交替执行"处理邮件"和"处理数据"；左边几个其他线程（定时器线程、RPC 线程）把任务包装成信封投进邮箱，但不能直接碰算子的状态】

### 5.3 证据：定时器和 Checkpoint 都变成了邮件

**定时器。** 每个 StreamTask 都有一个专门的定时器线程，名字叫 `Time Trigger for <Task 名>`（`StreamTask.java:494`）。但定时器到期时，这个线程**并不直接执行你的 `onTimer`**，而是把它包装成一封邮件投进邮箱：

```java
// StreamTask.java:1905-1914
ProcessingTimeCallback deferCallbackToMailbox(MailboxExecutor mailboxExecutor, ProcessingTimeCallback callback) {
    return timestamp -> {
        mailboxExecutor.execute(
                () -> invokeProcessingTimeCallback(callback, timestamp),
                "Timer callback for %s @ %d", callback, timestamp);
    };
}
```

所以实验里的 `onTimer`，打印出的是 Task 线程的名字，而不是 `Time Trigger for ...`。

**Checkpoint 完成通知。** JobManager 通过 RPC 通知"Checkpoint 已完成"时，`notifyCheckpointCompleteAsync()`（`:1560`）同样只是投递一封邮件，而且用的是最高优先级（`:1593-1598`，`TaskMailbox.MAX_PRIORITY`）。

**Checkpoint 本身。** 对 Source Task 来说，Checkpoint 由 JobManager 通过 RPC 触发，`triggerCheckpointAsync()`（`:1305`）也是把任务交给邮箱执行（`:1315`）。对下游的 Task 来说，Checkpoint 是由数据流里的 Barrier 触发的，而读取 Barrier 本来就发生在 Task 线程处理输入的过程中。这部分是下一讲的主角。

**所以：算子的所有回调，都在 Task 线程上串行执行。你写的代码、Flink 自己的算子实现，都不需要加锁。**

> 这也引出一个推论：**任何一个回调执行得太慢，都会拖住其他所有事情。** 比如在 `processElement` 里同步调用一个很慢的外部接口，这段时间里定时器无法触发，Checkpoint 也无法进行，最终表现为 Checkpoint 超时。

---

## 六、一条数据的旅程

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01b-2.webp" alt="一条数据从Mailbox默认动作进入算子并沿调用链完成处理" width="960" height="640" loading="lazy" /><figcaption>一条数据从Mailbox默认动作进入算子并沿调用链完成处理<span>AI 生成配图</span></figcaption></figure>

现在，看一条数据在 Task 线程里究竟经过了哪些方法。

我在实验里 `chained-map` 的 `map()` 方法中打印了调用栈（只保留了 Flink 运行时和本示例的帧）。下面是本机实测输出，**从下往上读**就是一条数据的旅程：

```
    at MailboxThreadDemo$ChainedMap.map(MailboxThreadDemo.java:116)
    at MailboxThreadDemo$ChainedMap.map(MailboxThreadDemo.java:110)
    at StreamMap.processElement(StreamMap.java:37)                          ⑦ 下游算子
    at CopyingChainingOutput.pushToOperator(CopyingChainingOutput.java:75)  ⑥ 链内传递
    at CopyingChainingOutput.collect(CopyingChainingOutput.java:50)
    at CopyingChainingOutput.collect(CopyingChainingOutput.java:29)
    at TimestampedCollector.collect(TimestampedCollector.java:53)
    at MailboxThreadDemo$Keyed.processElement(MailboxThreadDemo.java:86)    ⑤ 你的代码
    at MailboxThreadDemo$Keyed.processElement(MailboxThreadDemo.java:70)
    at KeyedProcessOperator.processElement(KeyedProcessOperator.java:87)    ④ 算子
    at RecordProcessorUtils.lambda$getRecordProcessor$0(RecordProcessorUtils.java:64)
    at OneInputStreamTask$StreamTaskNetworkOutput.emitRecord(OneInputStreamTask.java:251)
    at AbstractStreamTaskNetworkInput.processElement(AbstractStreamTaskNetworkInput.java:213)
    at AbstractStreamTaskNetworkInput.emitNext(AbstractStreamTaskNetworkInput.java:170)  ③ 反序列化
    at StreamOneInputProcessor.processInput(StreamOneInputProcessor.java:65)
    at StreamTask.processInput(StreamTask.java:656)                         ② 默认动作
    at MailboxProcessor.runMailboxLoop(MailboxProcessor.java:231)           ① Mailbox 循环
    at StreamTask.runMailboxLoop(StreamTask.java:1022)
    at StreamTask.invoke(StreamTask.java:959)
    at Task.runWithSystemExitMonitoring(Task.java:987)
    at Task.restoreAndInvoke(Task.java:969)
    at Task.doRun(Task.java:774)
    at Task.run(Task.java:579)
```

逐层解读：

| 编号 | 发生了什么 |
|---|---|
| ① | Task 线程在 Mailbox 循环里，执行"默认动作" |
| ② | 默认动作就是 `processInput()`：处理一批输入数据 |
| ③ | 从网络缓冲区里读出字节，反序列化成一条记录（`AbstractStreamTaskNetworkInput.emitNext`）。如果读到的是 Watermark 或 Checkpoint Barrier，会走别的分支（第七讲、第二讲） |
| ④ | 把记录交给链头的算子。对 keyed 算子，这里还会先把"当前 key"切换成这条记录的 key，这样你在 `processElement` 里访问的状态，自动就是这个 key 的 |
| ⑤ | **你的 `processElement` 被调用了** |
| ⑥ | 你调用 `out.collect()` 输出一条数据。因为下一个算子和你在同一条链上，`collect` 直接调用了下游算子的方法 |
| ⑦ | 下游算子 `chained-map` 的 `map()` 被调用 |

注意 ⑤ 到 ⑦：**上游算子输出一条数据，就是在同一个调用栈里直接调用了下游算子。** 没有队列，没有线程切换，也没有经过网络。这就是上篇讲的算子链省下来的开销。

### 6.1 一个细节：链内传递也会拷贝一次对象

调用栈里出现的是 `CopyingChainingOutput`，而不是 `ChainingOutput`。两者的区别由配置项 `pipeline.object-reuse` 决定，**默认是 `false`**（`flink-core/.../configuration/PipelineOptions.java:201-204`）：

- 不开启对象复用时（默认），使用 `CopyingChainingOutput`，每传一条记录，都会用序列化器**拷贝一份对象**再交给下游（`RT/streaming/runtime/tasks/CopyingChainingOutput.java:74`）。这样即使下游修改了这个对象，也不会影响上游手里的那一份。
- 开启对象复用后，使用 `ChainingOutput`，直接把同一个对象传下去，省掉了拷贝。代价是：你不能在 `collect()` 之后继续修改或缓存这个对象。

选择逻辑在 `RT/streaming/runtime/tasks/OperatorChain.java:690-697`。

所以更准确的说法是：**链内传递不需要序列化成字节、不经过网络，但默认会做一次对象拷贝。**

### 6.2 链的末尾：交给网络

如果下一个算子不在同一条链上（比如中间隔着一个 keyBy），链末尾的算子用的是 `RecordWriterOutput`：按分区器选出目标通道，把记录序列化写进缓冲区，缓冲区写满后交给网络层发出去。这是第四讲（网络栈与背压）的内容。

---

## 七、顺便看懂 Web UI 上的三个指标

Web UI 上每个 Task 都有三个指标：**Busy**、**Idle**、**BackPressured**。它们的来源就在默认动作 `processInput()` 里（`StreamTask.java:655-705`）。

处理完一批数据后，如果暂时不能继续处理，Flink 会判断原因：

```java
if (!recordWriter.isAvailable()) {                    // :687 下游没有空闲的缓冲区
    timer = new GaugePeriodTimer(ioMetrics.getSoftBackPressuredTimePerSecond());   // → 背压
    resumeFuture = recordWriter.getAvailableFuture();
} else if (!inputProcessor.isAvailable()) {           // :690 没有输入数据
    timer = new GaugePeriodTimer(ioMetrics.getIdleTimeMsPerSecond());              // → 空闲
    resumeFuture = inputProcessor.getAvailableFuture();
}
...
controller.suspendDefaultAction(timer)                // :704 暂停默认动作，等条件满足再恢复
```

- 输出发不出去 → 计入**背压**时间；
- 没有输入可读 → 计入**空闲**时间；
- 其余时间就是在**忙**着处理数据。

暂停默认动作期间，线程并没有闲着，它仍然会处理邮件，所以背压时定时器和 Checkpoint 依然能执行。

Web UI 上的 BackPressured，是这里的"软背压"时间，加上写数据时申请缓冲区被阻塞的"硬背压"时间（`RT/runtime/metrics/groups/TaskIOMetricGroup.java:240-243`）。后者的细节在第四讲。

---

## 八、自己动手

### 1. 运行本篇的实验

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/exec/MailboxThreadDemo.java`：

```bash
./run.sh study.exec.MailboxThreadDemo
```

拓扑是 `gen → upstream-map → keyBy → keyed-process → chained-map → sink`，并行度 1，Checkpoint 间隔 1 秒。你会看到，`upstream-map` 和 Source 链在一起，运行在另一个线程 `Source: gen -> upstream-map (1/1)#0` 上；而 `keyed-process` 的所有回调，都在它自己那条链的线程上。

### 2. 断点清单

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `DefaultExecutionGraph.java:867` | JobGraph 按并行度展开 |
| 2 | `PipelinedRegionSchedulingStrategy.java:282` | 以 region 为单位调度 |
| 3 | `DefaultExecutionDeployer.java:151` | 等所有 slot 到位后统一部署 |
| 4 | `Execution.java:656` | 发给 TaskExecutor 的 TDD 里有什么 |
| 5 | `TaskExecutor.java:883` | Task 线程启动 |
| 6 | `Task.java:944` | 状态迁移：DEPLOYING → INITIALIZING → RUNNING |
| 7 | `StreamTask.java:806` | 算子链被组装起来 |
| 8 | `MailboxProcessor.java:227` | Mailbox 主循环 |
| 9 | `StreamTask.java:1908` | 定时器被投递成邮件 |
| 10 | 你的 `processElement` | **终点**。在 IDEA 的 Debugger 面板里看调用栈，和第六节对照 |

记得把断点的 Suspend 改成 **Thread**。

---

## 九、课后练习

1. **线程名**：把实验里 `keyed-process` 和 `chained-map` 之间加一个 `.rebalance()`，再运行一次。`chained-map.map` 打印出的线程名会变成什么？调用栈会怎么变？
2. **慢回调**：在 `processElement` 里加一个 `Thread.sleep(2000)`，观察 `onTimer` 和 `notifyCheckpointComplete` 的打印时间是否被推迟。结合第五节解释原因。
3. **对象复用**：用 `env.getConfig().enableObjectReuse()` 开启对象复用（实验里加参数即可：`./run.sh study.exec.MailboxThreadDemo reuse`），再看调用栈，确认 `CopyingChainingOutput` 变成了 `ChainingOutput`。开启之后，你在自己的代码里要注意什么？
4. **状态机**：在 `Task.transitionState` 上打断点，画出 Task 从 `CREATED` 到 `FINISHED` 经历的所有状态，对照 `ExecutionState` 枚举。

---

## 写在最后

第一讲到这里就结束了。我们从一行 `env.execute()` 出发，看到了作业图的四次变形、算子链的 7 条规则、调度和部署的过程，最后在 Mailbox 循环里找到了"为什么不需要加锁"的答案。

| 上篇 | 下篇 |
|---|---|
| Transformation → StreamGraph → JobGraph | JobGraph → ExecutionGraph → 部署 → Task 线程 |
| keyBy 是边，不是算子 | 一个 Task 一个线程，所有回调串行执行 |
| 算子链的 7 条规则 | 链内传递是直接方法调用，默认拷贝一次对象 |

下一讲，我们进入 Flink 最核心的机制：**Checkpoint**。Barrier 是怎么从 Source 一路流到 Sink 的？对齐的时候发生了什么？非对齐 Checkpoint 又是怎么"插队"的？


**留一个问题**：你有没有遇到过"Checkpoint 总是超时"，最后发现是某个算子处理得太慢？

下一讲：**Flink 2.x 源码精读（二）：Checkpoint 全流程**
:::
