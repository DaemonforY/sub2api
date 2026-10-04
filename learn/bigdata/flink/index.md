---
title: "Apache Flink 源码学习"
description: "从 env.execute() 到 processElement()，Checkpoint、状态后端、网络栈与背压、Table/SQL、调度容错、Watermark 与窗口、新版 Source/Sink。"
---

# Apache Flink 源码学习

从 env.execute() 到 processElement()，Checkpoint、状态后端、网络栈与背压、Table/SQL、调度容错、Watermark 与窗口、新版 Source/Sink。

> 基于 Flink 2.3.0。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。

## 目录

| # | 文章 | 讲什么 |
|---|---|---|
| 1 | [作业执行全链路](/bigdata/flink/01-execution) | 用户代码怎样变成 StreamGraph、JobGraph、ExecutionGraph，Task 怎样部署运行到 processElement()。 |
| 2 | [Checkpoint 全流程](/bigdata/flink/02-checkpoint) | Checkpoint 从触发、Barrier 对齐、快照到确认完成的全流程，以及非对齐 Checkpoint。 |
| 3 | [State Backend：状态存储与扩缩容](/bigdata/flink/03-state-backend) | HashMap 与 RocksDB 状态后端的存储方式、增量 Checkpoint、KeyGroup 与扩缩容时的状态重分布。 |
| 4 | [网络栈与背压](/bigdata/flink/04-network) | 网络栈：ResultPartition、InputGate、Credit-based 流控，以及背压是怎样产生和传递的。 |
| 5 | [Table/SQL：从 SQL 到 Transformation](/bigdata/flink/05-table-sql) | Table/SQL 从 SQL 文本经 Calcite 解析优化，变成 ExecNode 和 Transformation 的过程。 |
| 6 | [调度与容错](/bigdata/flink/06-scheduling) | 调度器、Slot 分配、故障恢复策略和自适应调度（Adaptive Scheduler）的扩缩容。 |
| 7 | [从读者到贡献者](/bigdata/flink/07-contributor) | 从读源码到给 Flink 提 PR：社区数据、JIRA 与 PR 流程、适合新人的切入点。 |
| 8 | [时间、Watermark 与窗口](/bigdata/flink/08-watermark-window) | 事件时间与 Watermark 的生成和传播、窗口的分配、触发与清理。 |
| 9 | [Source / Sink 新架构](/bigdata/flink/09-source-sink) | FLIP-27 新 Source 架构（SplitEnumerator / SourceReader）与新版 Sink 的两阶段提交。 |

读的时候有疑问，可以打开源码对照；每篇开头都标了源码版本和路径缩写。

读完想检验一下？去做 [Flink 面试题](/bigdata/interview/flink)，AI 面试官会逐题打分、点评。
