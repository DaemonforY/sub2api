---
title: "踩坑实验室 #20：作业挂了，分区会被读两次吗？"
description: "故障恢复后分片不会被两个 subtask 同时读，但数据会重读；不重复靠的是两阶段提交。"
bigdata: "flink"
---

# 踩坑实验室 #20：作业挂了，分区会被读两次吗？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-04）。对应课程：第八讲（上）《Source，分片是怎么分下去、又怎么收回来的》、第八讲（下）第三节。配套示例：`flink-notes/demos/src/main/java/study/connector/SourceSinkDemo.java`（`fail` 模式）

## 现象

"作业挂了重启后，分区会不会被读两次？"这个问题有两种理解，答案不一样：

- 同一个分片（split）同时被两个 subtask 读：**不会**；
- 同一条数据被读两次：**会**。

而最终结果一条都没有重复。很多人把 Flink 的"恰好一次"理解成"只读一次"，这正是容易出错的地方。

## 复现

示例 `SourceSinkDemo fail`：4 个 split，每个 40 条（split-0 是 `[0,40)`，split-1 是 `[40,80)`，依此类推），并行度 2，每秒一次 Checkpoint，Reader 读完一个 split 再要下一个。在记录 80（split-2 的第一条）上注入一次故障。Sink 是示例自带的两阶段提交 Sink，事务 ID 是 `s{subtask}-chk{n}`。

日志节选（省略了线程名）：

```
[+ 4.90s] ENUM     ...  snapshotState(chk 5), pending=[split-2[80,120), split-3[120,160)]
[+ 4.94s] READER   ...  subtask 1 snapshotState(chk 5) = [split-0[36,40)]
[+ 4.94s] WRITER   ...  subtask 1 prepareCommit -> s1-chk5 n=12 [24..35]
[+ 5.27s] READER   ...  subtask 1 finished split-0 -> sendSplitRequest
[+ 5.27s] ENUM     ...  subtask 1 requests a split -> assign split-2[80,120)
[+ 5.36s] MAP      ...  !!! injected failure at record 80
[+ 6.39s] ENUM     ...  addSplitsBack([split-2[80,120)]) from failed subtask 1
[+ 6.41s] READER   ...  subtask 1 addSplits[split-0[36,40)]
[+ 6.49s] WRITER   ...  subtask 1 prepareCommit -> s1-chk6 n=1 [36..36]
[+ 6.74s] ENUM     ...  subtask 1 requests a split -> assign split-2[80,120)
[+ 7.42s] WRITER   ...  subtask 1 prepareCommit -> s1-chk7 n=11 [37..87]
[+10.10s] MAIN     ...  committed records: total=160 distinct=160 expected=160 duplicates=0
```

| 事务 | 提交的记录 |
|---|---|
| `s1-chk5`（故障前） | 24～35，共 12 条 |
| `s1-chk6`（恢复后） | 36，共 1 条 |
| `s1-chk7`（恢复后） | 37～39 和 80～87，共 11 条 |
| 最终 | total=160、distinct=160、duplicates=0 |

连续跑了两次，过程和结果完全相同（时间戳相差零点几秒）。第八讲上篇写作时也跑过一次：那次同样是 subtask 1 在记录 80 上故障，但它当时手里是 split-1，恢复后拿回的是 `split-176,80)`，被重读的是 76～79，结论相同。

![split-2 分出去后故障，被还给 Enumerator 再重新分配（本机实测）

## 原因

**分片为什么不会被两个人同时读。** 分片由 JobManager 上的 Enumerator 分配，每次分配都会记账（`SourceCoordinatorContext.java:305` 的 `recordSplitAssignment`）。某个 subtask 失败、回滚到 Checkpoint N 时，它在 N 之后才拿到的分片会被还给 Enumerator，重新分配（`SplitAssignmentTracker.java:120`、`SourceCoordinator.java:372-389` 的 `addSplitsBack`）。本例中 split-2 在 Checkpoint 5 之后才分给 subtask 1，subtask 1 挂了，split-2 就被还了回去，之后再分出去。背后的不变式是：任意一个 Checkpoint 里，每个分片要么在 Enumerator 的状态里，要么在某个 Reader 的状态里，而且只在一处。

**数据为什么会被读两次。** N 之前拿到的分片随 Reader 自己的状态恢复，从 N 记录的位置接着读（`SourceOperator.java:443-451`）。Checkpoint 5 时 subtask 1 记录的位置是 `split-036,40)`，也就是"下一条读 36"。故障前它已经把 36～39 读完了（5.27s 的 `finished split-0`），恢复后又从 36 开始读了一遍。

**结果为什么不重复。** 故障前，36～39 还没赶上任何一次 `prepareCommit`，只存在于 Writer 尚未提交的事务里，故障时随事务一起丢弃。恢复后，36 在 `s1-chk6` 里提交，37～39 在 `s1-chk7` 里提交（`n=11` 是 37、38、39 加上 80～87）。读了两次，只提交了一次。

![36～39 第一次读的结果随未提交事务丢弃，只有重读那次被提交（本机实测）

## 怎么解决

"不重复"靠的不是"只读一次"，而是 **Source 能回退 + Sink 两阶段提交（且 Committer 幂等）**。设计或排查端到端一致性时，要两头都检查：

- Source 要能把读取位置存进 Checkpoint，并在恢复时回退到那里；
- Sink 要支持两阶段提交，并且 Committer 幂等。踩坑实验室 #06 里去掉幂等后出现了 `duplicates=13`，Source 能回退并不足以保证不重复。

几点注意：

- 被还回去的分片由 Enumerator 重新分配，谁先请求给谁，不一定回到原来的 subtask；
- 具体是哪个 subtask 故障、哪段数据被重读，取决于分配时序，换一次运行数字可能不同，但结论不变；
- 本文用的是示例自带的 Source 和 Sink，Kafka 连接器在独立的仓库里，本文没有核对它的代码，细节以它的实现为准。

## 记住这几点

- 故障恢复后，同一个分片不会被两个 subtask 同时读，Checkpoint 之后才分出去的分片会被还给 Enumerator。
- 上一个 Checkpoint 之后读过的数据，恢复后会再读一遍。
- "恰好一次"指的是结果：Source 能回退，加上 Sink 两阶段提交且 Committer 幂等。
:::
