---
title: "踩坑实验室 #07：不开 Checkpoint，事务永远不提交"
description: "两阶段提交的两步都挂在 Checkpoint 上，无界流作业不开 Checkpoint，事务型 Sink 永不提交。"
bigdata: "flink"
---

# 踩坑实验室 #07：不开 Checkpoint，事务永远不提交

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-02）。对应课程：第八讲《Source / Sink 新架构》。配套示例：`flink-notes/demos/src/main/java/study/connector/SourceSinkDemo.java` 的 `unbounded`、`unbounded-nockpt` 和 `nockpt` 模式

## 现象

一个永不结束的流作业，Sink 是两阶段提交（事务型）的，但没开 Checkpoint。作业跑得好好的，数据也都进了 Sink，可外部系统里一条已提交的数据都没有。用 FileSink 时，表现就是文件一直停在 in-progress 或 pending 状态。

## 复现

`SourceSinkDemo`：并行度 2，160 条记录，自己实现的两阶段提交 Sink。

| 模式 | Source | Checkpoint | Sink 收到的记录 | 已提交的记录 | `prepareCommit` 调用次数 |
|---|---|---|---|---|---|
| `unbounded-nockpt` | 无界，运行 12 秒后取消 | **关** | 160 | **0** | **0** |
| `unbounded` | 无界，运行 12 秒后取消 | 开（1 秒） | 160 | 160 | 18 |
| `nockpt` | 有界，读完自然结束 | 关 | 160 | 160（结束时一次性提交） | 2（每个并行度在结束时各 1 次） |

两个无界模式各跑了两次，结果一致。

`unbounded-nockpt` 的关键输出：

```
[+ 8.57s] ENUM     ... subtask 0 requests a split -> none left, keep waiting
[+ 8.61s] ENUM     ... subtask 1 requests a split -> none left, keep waiting
[+13.99s] MAIN     main   after 12s: written to sink=160, committed=0 (checkpointing=false)
```

`unbounded` 的关键输出：

```
[+ 1.84s] COMMIT   ... s0-chk1 n=1 [0..0] (retries=0)
[+ 1.93s] COMMIT   ... s1-chk2 n=1 [41..41] (retries=0)
...
[+13.57s] MAIN     main   after 12s: written to sink=160, committed=160 (checkpointing=true)
```

## 原因

两阶段提交的两步都挂在 Checkpoint 上：

- 第一步 `prepareCommit` 只在 Checkpoint 时调用，位置在 Barrier 发出之前（`SinkWriterOperator.java:188-193`，`prepareSnapshotPreBarrier`）；
- 第二步 `commit` 只在 Checkpoint **完成**之后调用（`CommitterOperator.java:159-162`，`notifyCheckpointComplete`）。

唯一的例外是输入结束：没开 Checkpoint（或者是批作业）时，在 `endInput` 时一次性提交全部数据（`SinkWriterOperator.java:206-212`、`CommitterOperator.java:151-156`）。有界作业读完会结束，所以 `nockpt` 在结束时提交了 160 条；无界的流作业不会结束，永远等不到这个时刻。

所以，**无界的流作业不开 Checkpoint，事务型 Sink 一次都不会提交**，数据一直停留在“已写入、未提交”的状态。注意这不等于“数据丢了”：数据没有丢，只是没提交；作业被取消或失败后，这些未提交的数据怎么处理取决于具体的 Sink。

而流作业恰恰是默认模式：`execution.runtime-mode` 默认是 `STREAMING`（`ExecutionOptions.java:41-44`）。官方文档对 FileSink 也有明确说明：STREAMING 模式下使用 FileSink 必须开启 Checkpoint，否则文件将**永远**停留在 in-progress 或 pending 状态，下游无法安全读取（`docs/content.zh/docs/connectors/datastream/filesystem.md:283-284`）。

![两步都挂在 Checkpoint 上：不开 Checkpoint，预提交和提交都不会发生](/bigdata-img/src/content-plan/assets/png/ep07/xhs-P3.webp)

![CommitterOperator 只在两个时刻提交：Checkpoint 完成后，或没开 Checkpoint 时的输入结束](/bigdata-img/src/content-plan/assets/png/ep07/xhs-P4.webp)

## 怎么解决

用事务型 Sink 的流作业，一定要开 Checkpoint，并确认它真的在成功：

- **开启 Checkpoint**：实测开启 1 秒间隔后，160 条全部提交。
- **确认 Checkpoint 在成功**：开了 Checkpoint 但一直失败，同样不会提交（这是由 `commit` 只在 Checkpoint 完成后调用得出的源码结论，本文没有单独实测）。可以在 Web UI 的 Checkpoints 页查看。
- **留意 Checkpoint 间隔**：它会影响数据多久之后才能被下游看到，不要随手设成几分钟。但它不是唯一因素，FileSink 还要看滚动策略（见踩坑实验室 #11）。

适用范围：这只针对两阶段提交（事务型）的 Sink，比如 FileSink、开启 exactly-once 的事务型 Sink；`print()`、at-least-once 模式这类不需要提交的 Sink 不受影响。本文验证的是 Flink 框架和自己实现的 Sink，Kafka 事务中的数据在提交前是否可见，还取决于消费端的隔离级别，以 Kafka 文档为准。

## 记住这几点

- 两阶段提交的 `prepareCommit` 在 Checkpoint 时、`commit` 在 Checkpoint 完成后，不开 Checkpoint 两步都不会发生。
- 无界流作业不开 Checkpoint，事务型 Sink 永远不提交；有界作业则在输入结束时一次性提交。
- 事务型 Sink 的流作业必须开 Checkpoint，还要确认它真的在成功，并合理设置间隔。
:::
