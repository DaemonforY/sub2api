---
title: "Flink 2.x 源码精读（一·上）：同一份 WordCount，为什么 IDE 里 3 个 Task，集群上只有 2 个？"
description: "同一份 WordCount，IDE 里跑出 3 个 Task，集群上只有 2 个。顺着这个现象，读懂 Flink 2.3 从 API 调用到 StreamGraph、再到 JobGraph 和算子链的全过程，每一步都给出源码位置和本机实测输出。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-01a-cover.webp"}]]
---

# Flink 2.x 源码精读（一·上）：同一份 WordCount，为什么 IDE 里 3 个 Task，集群上只有 2 个？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01a-cover.webp" alt="从 API 到算子链的作业图演化" width="1200" height="800" loading="eager" /><figcaption>从 API 到算子链的作业图演化<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`RT` = `flink-runtime/src/main/java/org/apache/flink`

---

上一讲最后，我们在本机的 Flink 集群上跑了自带的 WordCount，Web UI 上显示的作业是这样的：

【配图 1：Web UI 中 WordCount 的作业图，两个节点：`Source: in-memory-input -> tokenizer` 和 `counter -> Sink: print-sink`，中间是 HASH】

**两个节点。**

可是同一份代码，如果在 IDEA 里直接运行 `main` 方法，生成的却是**三个**节点。我写了一个小程序，用和 WordCount 完全相同的拓扑生成作业图，在本机打印出来：

```
========== parallelism = 1 ==========
③ JobGraph：2 个 JobVertex（= Web UI 上的节点数）
   [Source: in-memory-input -> tokenizer]  parallelism=1
   [counter -> Sink: print-sink]  parallelism=1

========== parallelism = 默认（15） ==========
③ JobGraph：3 个 JobVertex（= Web UI 上的节点数）
   [Source: in-memory-input]  parallelism=1
   [tokenizer]  parallelism=15
   [counter -> Sink: print-sink]  parallelism=15
```

代码一个字都没改，Task 的个数却变了。

这不是 bug。读完这一篇，你会知道：

1. 你写的 `flatMap().keyBy().sum()`，在调用的那一刻**什么都没有执行**，它们只是在"记账"；
2. 这本账会经历**四次变形**：Transformation 列表 → StreamGraph → JobGraph → ExecutionGraph；
3. 哪些算子会被**链**在一起、放进同一个 Task，是由 **7 条规则**决定的，上面的现象就是其中一条在起作用；
4. Flink 2.x 改了一件事：**JobGraph 不再在客户端生成**，老资料里的这部分流程已经过时了。

---

## 一、先看全景：四张图

一个 Flink 作业从你写下代码，到真正在集群上运行，中间会依次生成四张"图"：

| 图 | 在哪里生成 | 一个节点代表什么 | 一句话 |
|---|---|---|---|
| **Transformation 列表** | 调用 API 时 | 一次 API 调用 | 只是记账，不做任何计算 |
| **StreamGraph** | 客户端 | 一个算子 | 逻辑图，每条边上标着数据怎么分发 |
| **JobGraph** | **JobManager**（2.x 的变化） | 一条**算子链** | 决定切成几个 Task |
| **ExecutionGraph** | JobManager | 一个**并行实例** | 调度和容错真正操作的对象 |

【配图 2：四张图的演变示意。从左到右四列，用 WordCount 的例子画出每一层的节点：Transformation 3 个 → StreamGraph 4 个节点、3 条边（REBALANCE / HASH / FORWARD）→ JobGraph 3 个节点（虚线框表示链）→ ExecutionGraph 按并行度展开】

这一篇讲前三张图；ExecutionGraph 以及之后的调度、部署、Task 运行，放在下篇。

---

## 二、API 调用只是在"记账"

先看 WordCount 的核心代码（`flink-examples/flink-examples-streaming/.../wordcount/WordCount.java`，为了阅读方便，省略了注释和读文件的分支）：

```java
text = env.fromData(WordCountData.WORDS).name("in-memory-input");
counts = text.flatMap(new Tokenizer()).name("tokenizer")
             .keyBy(value -> value.f0)
             .sum(1).name("counter");
counts.print().name("print-sink");
env.execute("WordCount");
```

一个很容易产生的误解是：调用 `flatMap` 时，Flink 就开始切词了。

**实际上，在 `execute()` 之前，一条数据都没有处理。** 每次 API 调用，只是创建一个 `Transformation` 对象，描述"这里要做什么"。其中一部分会被登记到环境的一个列表里：

```java
// RT/streaming/api/environment/StreamExecutionEnvironment.java:2131
public void addOperator(Transformation<?> transformation) {
    this.transformations.add(transformation);
}
```

我在调用 `execute()` 之前，把这个列表打印了出来（`env.getTransformations()`，`:2456`）：

```
① Transformation 列表（env.getTransformations()，execute 之前）：
   OneInputTransformation       tokenizer
   ReduceTransformation         counter
   LegacySinkTransformation     print-sink
```

**只有 3 项。** 有两个东西不在里面：

- **Source 不在。** `fromData` 只是返回一个带着 `SourceTransformation` 的 DataStream，并没有调用 `addOperator`。它会在后面被找到：每个 Transformation 都记着自己的上游，从 `tokenizer` 往上一找，就找到了 Source。
- **keyBy 不在。** `keyBy` 只创建了一个 `PartitionTransformation`（`RT/streaming/api/datastream/KeyedStream.java:134`），它**不是算子**，只是在两个算子之间标注"数据要按 key 重新分发"。后面它会变成边上的 `HASH`。

真正登记进列表的，是 `flatMap`（`RT/streaming/api/datastream/DataStream.java:844`）、`sum`（`KeyedStream.java:737`，`sum` 会被翻译成一个 reduce）和 `print`（`DataStream.java:1047`）。

**所以 `keyBy` 是"边"，不是"点"。** 这一点很重要，它直接决定了后面的 Task 怎么切。

---

## 三、`execute()`：把账本整理成 StreamGraph

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01a-1.webp" alt="execute 将转换列表整理成 StreamGraph" width="960" height="640" loading="lazy" /><figcaption>execute 将转换列表整理成 StreamGraph<span>AI 生成配图</span></figcaption></figure>

`execute()` 的入口在 `StreamExecutionEnvironment.java:1838`。它做的第一件事，就是把 Transformation 列表交给 `StreamGraphGenerator`，生成 StreamGraph（`:2044`）。

`StreamGraphGenerator.generate()`（`RT/streaming/api/graph/StreamGraphGenerator.java:253`）遍历列表，对每个 Transformation 调用 `transform()`（`:463`）。`transform()` 做三件事：

1. **去重**：同一个 Transformation 只转换一次（`:464`）。
2. **先上游，后自己**：转换时会先递归转换它的输入，所以 Source 就是在这一步被找到并加入图里的。
3. **按类型分派**：每种 Transformation 都有对应的翻译器，最终调用 `streamGraph.addOperator()` 加节点、`addEdge()` 加边。`PartitionTransformation` 不会生成真正的算子节点，只生成一个"虚拟分区节点"（`StreamGraph.addVirtualPartitionNode()`），连边时它会被解析掉，变成下一条边上的分区器。

### 边上的分区器是怎么定的？

这是本篇最关键的一段代码，`RT/streaming/api/graph/StreamGraph.java:912-920`：

```java
// If no partitioner was specified and the parallelism of upstream and downstream
// operator matches use forward partitioning, use rebalance otherwise.
if (partitioner == null
        && upstreamNode.getParallelism() == downstreamNode.getParallelism()) {
    partitioner = dynamic ? new ForwardForUnspecifiedPartitioner<>() : new ForwardPartitioner<>();
} else if (partitioner == null) {
    partitioner = new RebalancePartitioner<Object>();
}
```

翻译成一句话：**如果你没有指定数据怎么分发，并行度相同就用 FORWARD（一对一直连），不同就用 REBALANCE（轮询打散）。**

本机打印出的 StreamGraph（默认并行度 15）：

```
② StreamGraph：节点 + 边上的分区器
   Source: in-memory-input(p=1) --REBALANCE--> tokenizer(p=15)
   tokenizer(p=15) --HASH--> counter(p=15)
   counter(p=15) --FORWARD--> Sink: print-sink(p=15)
```

三条边，三种分区器：
- `Source → tokenizer` 是 **REBALANCE**：因为 `fromData` 的并行度被**固定为 1**（`StreamExecutionEnvironment.java:843`，`.setParallelism(1)`），而 tokenizer 是 15，并行度不同；
- `tokenizer → counter` 是 **HASH**：这就是 `keyBy` 留下的；
- `counter → print-sink` 是 **FORWARD**：没有指定分区方式，两边并行度都是 15。

**请记住第一条边。** 开头那个"3 个还是 2 个"的谜题，答案就藏在这里。

---

## 四、提交：2.x 交出去的是 StreamGraph

StreamGraph 生成之后，`executeAsync()`（`:1988`）根据 `execution.target` 找到对应的执行器（`getPipelineExecutor()`，`:2509`），把作业交出去。

**这里是 Flink 2.x 和 1.x 的一个重要区别。**

在 1.x 里，客户端会先把 StreamGraph 转换成 JobGraph，再提交 JobGraph。很多老文章讲的都是这个流程。

而在 2.3.0 里，不管是 IDE 里的本地模式，还是提交到 Session 集群，客户端交出去的都是 **StreamGraph**：

```java
// flink-clients/.../deployment/executors/LocalExecutor.java:97-101（本地模式）
final StreamGraph streamGraph = PipelineExecutorUtils.getStreamGraph(pipeline, configuration);
... .submitJob(streamGraph, userCodeClassloader)

// flink-clients/.../deployment/executors/AbstractSessionClusterExecutor.java:93-106（Session 集群）
StreamGraph streamGraph = PipelineExecutorUtils.getStreamGraph(pipeline, configuration);
... .submitJob(streamGraph)
```

JobManager 侧接收的类型也相应变成了一个接口 `ExecutionPlan`（`RT/streaming/api/graph/ExecutionPlan.java:49`），StreamGraph 和 JobGraph 都实现了它。比如 MiniCluster 的提交入口是 `submitJob(ExecutionPlan executionPlan)`（`RT/runtime/minicluster/MiniCluster.java:1099`）。

这个改动来自 **FLINK-36065**（提交 `64cbcd2d4b3`，2024-10-17），从 2.0.0 开始生效。我是用这条命令查到的：

```bash
git log --format='%h %ad %s' --date=short -S"submitJob(streamGraph)" -- flink-clients/src/main/java/org/apache/flink/client/deployment/executors/AbstractSessionClusterExecutor.java
```

为什么要这么改？把图的最终生成推迟到 JobManager，JobManager 就有机会根据运行时的信息去**调整**这张图，比如批作业根据上游实际产生的数据量决定下游的并行度。这部分在第六讲（调度与容错）里会详细讲。

---

## 五、JobManager 生成 JobGraph：算子链在这里决定

<figure class="ai-figure"><img src="/bigdata-img/flink/post-01a-2.webp" alt="JobManager 将算子合并成执行链" width="960" height="640" loading="lazy" /><figcaption>JobManager 将算子合并成执行链<span>AI 生成配图</span></figcaption></figure>

### 5.1 在哪里生成

StreamGraph 到了 JobManager 之后，依次经过 Dispatcher、JobMaster，在创建调度器的时候被转换成 JobGraph（`RT/runtime/scheduler/DefaultSchedulerFactory.java:88-96`）：

```java
if (executionPlan instanceof JobGraph) {
    jobGraph = (JobGraph) executionPlan;
} else if (executionPlan instanceof StreamGraph) {
    jobGraph = ((StreamGraph) executionPlan).getJobGraph(userCodeLoader);   // ← 在这里打断点
}
```

`StreamGraph.getJobGraph()`（`RT/streaming/api/graph/StreamGraph.java:1195`）最终调用 `StreamingJobGraphGenerator.createJobGraph()`（`RT/streaming/api/graph/StreamingJobGraphGenerator.java:222`）。它的核心是 `setChaining()`：从每个 Source 出发做深度优先遍历，决定哪些相邻的算子可以合并成一条**算子链**。

**一条算子链 = 一个 JobVertex = 运行时的一个 Task = Web UI 上的一个节点。**

### 5.2 为什么要做算子链

链在一起的算子运行在**同一个线程**里，上游算子输出一条数据，是**直接调用**下游算子的方法（下篇会看到 `ChainingOutput`），不需要序列化，不经过网络缓冲区，也不需要线程切换。

所以算子链是 Flink 最基础的性能优化之一。但它是有条件的。

### 5.3 两个算子能链在一起的 7 条规则

判定逻辑在 `isChainable()`（`StreamingJobGraphGenerator.java:1734`）、`isChainableInput()`（`:1756`）、`areOperatorsChainable()`（`:1802`）、`arePartitionerAndExchangeModeChainable()`（`:1786`）这几个方法里。整理下来，**以下条件全部满足**，上下游两个算子才能链在一起：

| # | 规则 | 源码 |
|---|---|---|
| 1 | 下游算子**只有一条输入边** | `:1738` |
| 2 | 没有全局关闭算子链（没有调用 `disableOperatorChaining()`） | `:1761` |
| 3 | 两个算子在**同一个 slot 共享组** | `:1762` |
| 4 | 两个算子的**链接策略**允许：上游不是 `NEVER`；下游不是 `NEVER` 或 `HEAD`；下游是 `HEAD_WITH_SOURCES` 时，上游必须是 Source | `:1825-1854` |
| 5 | 两个算子的**并行度相同** | `:1866` |
| 6 | 边上的分区器是 **FORWARD**，并且不是批处理的数据交换模式 | `:1793-1795` |
| 7 | 不是把同一个输入的多条边 union 到一起 | `:1777-1781` |

还有一个容易被问到的细节：**最大并行度不同的两个算子，默认也可以链在一起**。这由配置项 `pipeline.operator-chaining.chain-operators-with-different-max-parallelism` 控制，默认是 `true`（`flink-core/.../configuration/PipelineOptions.java:264-268`），只有把它关掉时，才会额外要求最大并行度相同（`StreamingJobGraphGenerator.java:1869-1872`）。

### 5.4 回到开头的谜题

现在可以解释了。把 WordCount 的三条边逐一对照上面的规则：

**IDE 里（默认并行度 = CPU 核数，本机是 15）：**

| 边 | 分区器 | 能否链接 | 原因 |
|---|---|---|---|
| Source → tokenizer | REBALANCE | ❌ | 违反规则 5、6：Source 固定为 1，tokenizer 是 15，并行度不同，分区器也因此变成了 REBALANCE |
| tokenizer → counter | HASH | ❌ | 违反规则 6：keyBy 带来的 HASH 不是 FORWARD |
| counter → print-sink | FORWARD | ✅ | 全部满足 |

结果是 3 个 JobVertex：`[Source]`、`[tokenizer]`、`[counter -> Sink]`。

**集群上（默认并行度 = 1）：**

集群的默认并行度来自配置项 `parallelism.default`，默认值是 **1**（`flink-core/.../configuration/CoreOptions.java:469-472`）。这样一来，Source 和 tokenizer 的并行度都是 1，第一条边就变成了 FORWARD，规则全部满足，两者被链在了一起：

| 边 | 分区器 | 能否链接 |
|---|---|---|
| Source → tokenizer | **FORWARD** | ✅ |
| tokenizer → counter | HASH | ❌ |
| counter → print-sink | FORWARD | ✅ |

结果是 2 个 JobVertex：`[Source -> tokenizer]`、`[counter -> Sink]`。这和 Web UI 上看到的完全一致。

而本地默认并行度为什么是 CPU 核数？因为在 IDE 里运行时没有命令行的上下文，`getExecutionEnvironment()` 会创建一个本地环境（`StreamExecutionEnvironment.java:2197`），本地环境的默认并行度取的是 `Runtime.getRuntime().availableProcessors()`（`:161`）。

**一句话总结：Task 的个数，不是由你写了几个算子决定的，而是由"边"决定的。** 并行度一变，分区器就可能跟着变，算子链也就跟着变。

### 5.5 再做一个对照：关掉算子链

把 `env.disableOperatorChaining()` 加上，并行度设为 1，本机实测：

```
========== parallelism = 1，disableOperatorChaining() ==========
③ JobGraph：4 个 JobVertex（= Web UI 上的节点数）
   [Source: in-memory-input]  parallelism=1
   [tokenizer]  parallelism=1
   [counter]  parallelism=1
   [Sink: print-sink]  parallelism=1
```

4 个算子，4 个 Task，违反的是规则 2。生产环境中很少需要这样做，但排查问题时它很有用：每个算子单独一个 Task，Web UI 上就能分别看到每个算子的吞吐和背压。

---

## 六、自己动手

### 1. 运行本篇的实验

实验代码在系列仓库的 `flink-notes/demos/src/main/java/study/exec/JobGraphShapeDemo.java`。它不提交作业，只在本地生成三张图并打印出来：

```bash
./run.sh study.exec.JobGraphShapeDemo           # 默认并行度
./run.sh study.exec.JobGraphShapeDemo 1 4       # 并行度 1 和 4
./run.sh study.exec.JobGraphShapeDemo 1 nochain # 关闭算子链
```

> 说明：实验中直接调用了 `StreamGraph.getJobGraph()`。真实提交时，这一步在 JobManager 上执行，调用的是同一个方法。

### 2. 打断点

在 IDEA 里运行 WordCount 的 `main` 方法（第零讲讲过，本地模式下所有组件都在一个 JVM 里），依次在下面几个位置打断点：

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `StreamExecutionEnvironment.java:1838` | `transformations` 列表里有几项 |
| 2 | `StreamGraphGenerator.java:463` | 每种 Transformation 是怎么变成节点的 |
| 3 | `StreamGraph.java:912` | Source → tokenizer 这条边为什么是 REBALANCE |
| 4 | `MiniCluster.java:1099` | 提交的是 StreamGraph，而不是 JobGraph |
| 5 | `DefaultSchedulerFactory.java:92` | StreamGraph 在 JobManager 上变成 JobGraph |
| 6 | `StreamingJobGraphGenerator.java:1756` | 每一条边是否能链接 |

断点上记得把 Suspend 改成 **Thread**（第零讲讲过原因）。

---

## 七、课后练习

1. **推导**：只把 `flatMap` 的并行度设成 1（`.flatMap(...).setParallelism(1)`），其余不变，在 IDE 里运行，会生成几个 JobVertex？先推导，再用实验验证。
2. **slot 共享组**：给 `sum` 加上 `.slotSharingGroup("g2")`，会有什么变化？对应的是哪条规则？
3. **读测试**：打开 `flink-streaming-java/src/test/java/org/apache/flink/streaming/api/graph/StreamingJobGraphGeneratorTest.java`，挑三个测试用例，说出它们各自验证的是 5.3 节的哪条规则。
4. **思考**：既然 FORWARD 能链接、REBALANCE 不能，那在 Source 和 tokenizer 之间手动加一个 `.forward()`，同时保持并行度不同，会发生什么？（提示：去 `StreamGraph.java` 中添加边的逻辑里找找，就在第三节那段分区器代码的后面）

---

## 写在最后

这一篇我们只做了一件事：**看懂作业图是怎么一步步生成的**。下一篇，JobGraph 会被展开成 ExecutionGraph，经过调度、申请资源、部署到 TaskManager，最后在 Task 线程的 **Mailbox 循环**里，一条数据终于来到 `processElement()`。我们也会回答一个经典问题：**为什么 Flink 的算子不需要加锁？**


**留一个问题**：你遇到过"Task 个数和预期不一样"的情况吗？当时是什么原因？

下一篇：**Flink 2.x 源码精读（一·下）：一条数据是怎么走到 `processElement()` 的？**
:::
