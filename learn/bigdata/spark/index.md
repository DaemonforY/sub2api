---
title: "Apache Spark 源码学习"
description: "从编译源码、跟踪一个 Job 的一生，到调度、Shuffle、内存、Catalyst、Codegen 和 AQE，每个结论都给出源码位置和可复现的实验。"
---

# Apache Spark 源码学习

从编译源码、跟踪一个 Job 的一生，到调度、Shuffle、内存、Catalyst、Codegen 和 AQE，每个结论都给出源码位置和可复现的实验。

> 基于 Spark 4.2.0。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验见 [GitHub](https://github.com/DaemonforY/spark-source-notes)。

## 目录

| # | 文章 | 讲什么 |
|---|---|---|
| 1 | [学习路线：从入门到 Committer](/bigdata/spark/roadmap) | L0 预备到 L4 专家五个阶段：每阶段读哪些源码、做哪些练习、怎样参与社区。 |
| 2 | [从零编译 Spark 4.2.0 源码](/bigdata/spark/01-build) | 在国内网络下从零编译 Spark 4.2.0 源码、导入 IDEA，以及编译中遇到的坑和解决办法。 |
| 3 | [Job、Stage、Task 是什么关系](/bigdata/spark/02-job-stage-task) | 用 7 个实验说清楚 Action、Job、Stage、Task 的对应关系，以及常见面试题怎么答。 |
| 4 | [宽窄依赖：Stage 在哪里切开](/bigdata/spark/03-dependency) | 窄依赖和宽依赖的区别、Stage 在哪里被切开，用 10 组实验纠正 3 个常见误区。 |
| 5 | [L1 练习：跟踪 SparkPi 一个 Job 的一生](/bigdata/spark/l1-sparkpi) | 在 IDEA 里断点跟踪 SparkPi，从 rdd.reduce() 一路跟到 Executor 执行 Task 再回到 Driver。 |
| 6 | [第 2 讲：调度体系](/bigdata/spark/lecture-02-scheduling) | DAGScheduler 与 TaskScheduler：Stage 何时提交、多 Job 调度、本地性、失败重试和推测执行。 |
| 7 | [第 3 讲：Shuffle](/bigdata/spark/lecture-03-shuffle) | Shuffle 的写和读：三种 ShuffleWriter 的选择条件、溢写与合并、Shuffle 读取流程。 |
| 8 | [第 4 讲：内存管理](/bigdata/spark/lecture-04-memory) | 统一内存管理：执行内存与存储内存的划分、互相借用的规则，以及堆外内存。 |
| 9 | [第 5 讲：存储体系](/bigdata/spark/lecture-05-storage) | BlockManager 存储体系：存储级别、cache 的实现、广播变量的分块传输。 |
| 10 | [第 6 讲：RPC 与部署](/bigdata/spark/lecture-06-rpc-deploy) | RPC 框架与部署模式：RpcEndpoint、Driver 与 Executor 的通信、动态资源分配。 |
| 11 | [第 7 讲：容错机制](/bigdata/spark/lecture-07-fault-tolerance) | 容错机制：Task 重试、Executor 丢失、FetchFailed 与 Stage 重算、Checkpoint。 |
| 12 | [第 8 讲：Spark SQL 与 Catalyst](/bigdata/spark/lecture-08-catalyst) | Spark SQL 与 Catalyst：解析、分析、优化、物理计划四个阶段，以及实际生效的优化规则。 |
| 13 | [第 9 讲：代码生成与 Tungsten](/bigdata/spark/lecture-09-codegen) | 全阶段代码生成与 Tungsten：看生成的代码、做性能对比、理解 UnsafeRow。 |
| 14 | [第 10 讲：AQE、DSv2 与 Spark Connect](/bigdata/spark/lecture-10-aqe-dsv2) | AQE 自适应执行、DataSource V2 接口和 Spark Connect 的源码实现。 |

读的时候有疑问，可以打开源码对照；每篇开头都标了源码版本和路径缩写。

读完想检验一下？去做 [Spark 面试题](/bigdata/interview/spark)，AI 面试官会逐题打分、点评。
