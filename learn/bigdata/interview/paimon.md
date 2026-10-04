---
title: Paimon 面试题
description: 湖仓 / 数据集成岗常问的 Apache Paimon：元数据树、主键表 LSM、合并选文件与执行、乐观并发提交、删除向量、changelog、Flink 读写和综合调优，附参考答案和 AI 模拟面试。
---

# Paimon 面试题

湖仓、数据集成岗位问 Paimon，很少停在「会不会建表」。面试官更想知道你是否理解它的存储内核：一张主键表怎样按 bucket 组织成 LSM 树，一次写入怎样变成快照，合并怎样选文件，提交怎样处理并发冲突，流读怎样拿到 -U/+U，以及和 Flink 的 checkpoint 怎样配合做到 exactly-once。

被问到和 Hudi、Iceberg 的区别时，别背功能清单，要从存储结构讲：Paimon 的主键表是 LSM 树，适合高频流式 upsert，还能在合并时产出 changelog 给下游流读；元数据同样是 snapshot → manifest 的版本树，也能生成 Iceberg 兼容的元数据。讲清「它为什么适合流式更新、代价在哪」，比给出谁好谁坏的结论更有说服力。

## 题库

### Paimon 表在存储上是怎么组织的？请从 snapshot 一直讲到数据文件。

- **一句话**：按 bucket 组织的 LSM 树（数据）+ snapshot / manifest 元数据树（版本）+ 乐观并发提交（ACID）。主键表走 `KeyValueFileStore`，append 表走 `AppendOnlyFileStore`，只做小文件合并。
- **元数据树**：`snapshot-N` 指向三个 manifest list：base（上一版本的全量文件）、delta（本次新增 / 删除的文件）、changelog（本次产生的变更，供流读），另有 index manifest；manifest 里是 `ManifestEntry`（ADD / DELETE + `DataFileMeta`）；最后是 `bucket-N` 下的数据文件。
- `DataFileMeta` 记录 level、min/max key、sequence 范围和统计信息，扫描时据此裁剪。

延伸阅读：[paimon-core 整体架构](/bigdata/paimon/02-architecture)

### 主键表的一条数据从写入到对读可见，要经过哪些步骤？bucket 在其中起什么作用？

- **写入**：`TableWriteImpl` → `KeyValueFileStoreWrite` 按 (partition, bucket) 找到 writer → `MergeTreeWriter` 写进内存排序缓冲。记录存成 `KeyValue`：key + sequenceNumber + RowKind + value，sequenceNumber 决定同 key 多版本的新旧。
- **落文件**：缓冲满时默认先溢写本地磁盘（`write-buffer-spillable`），`prepareCommit` 时排序、合并同 key，写出 L0 文件并触发异步合并。
- **可见**：提交创建新快照后才对读可见，流作业延迟约为 checkpoint 间隔加提交耗时。
- **bucket**：每个 bucket 一棵独立 LSM，是读写的最小并行单元；改 bucket 数要 `INSERT OVERWRITE` 重分布。

延伸阅读：[paimon-core 整体架构](/bigdata/paimon/02-architecture)

### UniversalCompaction 是怎么挑选要合并的文件的？合并结果放到哪一层？

- **输入**：每个 L0 文件单独算一个 run（L0 文件可能重叠），L1 及以上每层一个 run；排在最前的最新，最后一个最老。
- **判断顺序**：可选的 `EarlyFullCompaction` → 空间放大：除最老 run 外的总大小超过最老 run 的 200%（`compaction.max-size-amplification-percent`）就全量合并 → 大小比例：从最新 run 滚雪球吸收（`compaction.size-ratio`）→ 文件数兜底：run 数超过 `num-sorted-run.compaction-trigger`（默认 5）强制合并。
- **输出层**：只选最新的连续前缀；全选到 maxLevel，否则放到下一个更老 run 的 level - 1，不输出到 L0，保持「level 越低数据越新」。

延伸阅读：[UniversalCompaction 选文件策略](/bigdata/paimon/03-universal-compaction)

### 合并真正执行时，哪些文件只「升级」不重写，哪些必须重写？为什么这样设计？

- `MergeTreeCompactTask` 先用 `IntervalPartition` 按 key 区间切 section（section 之间不相交），section 内打包成最少的 sorted run。
- **重写**：section 有多个 run（key 重叠）必须归并；单个 run 里小于 `target-file-size` × `compaction.small-file-ratio`（主键表约 89.6MB）的小文件跟着重写。
- **升级**：大文件只改 level 元数据，零数据 IO。遇到大文件要先把攒下的候选刷掉，否则输出的 key 范围会跨过它，破坏「同层不重叠」。
- **大文件也强制重写**：输出到最高层且含 DELETE、要求重写全部文件、含行级 TTL 过期数据。

延伸阅读：[MergeTreeCompactTask 合并执行](/bigdata/paimon/04-compact-task)

### Paimon 的提交怎么保证并发安全？「抢快照号失败」和「真正的冲突」有什么区别？

- **乐观并发**：不锁表，先写好 manifest，再原子创建 `snapshot-(N+1)`，谁先创建谁赢（`RenamingSnapshotCommit`，对象存储配外部锁；REST 模式由服务端 CAS）。
- **抢号失败**：重读最新快照，指数退避重试（`commit.max-retries` 默认 10），可复用上次的 base 和 manifest 合并结果。
- **真实冲突**：直接抛异常让作业 failover。base + delta 合并后还残留 DELETE，说明要删的文件已被别人删了（File deletion conflicts）；主键表同层 key 范围重叠（LSM conflicts）。APPEND 只新增文件，默认不检测；COMPACT 必须检测。
- **幂等**：`commitUser + commitIdentifier + commitKind` 唯一标识一次提交，原子提交抛异常后，下次先查是否其实已成功。

延伸阅读：[提交：冲突检测与重试](/bigdata/paimon/05-commit)

### 主键表批读什么时候需要归并？删除向量（deletion vectors）是怎么做到免合并读的？

- **在规划阶段决定**：`MergeTreeSplitGenerator` 计算 `rawConvertible`。没有 L0、没有 DELETE 行，且满足开启 DV、first-row 引擎、所有文件在同一层三者之一，整个 bucket 才能直接读；否则走 `MergeFileSplitRead` 多路归并。
- **归并的代价**：CPU 和内存开销；重叠 section 只能下推主键过滤，下推 value 过滤会读到旧版本；并行度受 section 限制。
- **DV**：L0 上推时 lookup 旧值，用位图标记旧记录的文件和行号，保证 L≥1 每个 key 只有一条存活记录；DV 索引与新文件进同一个快照。读时跳过 L0，直接读文件再按位图过滤，value 过滤可以下推。
- **代价**：写入要 lookup，L0 合并前不可见。

延伸阅读：[读路径与删除向量](/bigdata/paimon/06-read-path)

### Paimon 的 changelog 有哪几种产生方式？lookup 模式是怎么产出 -U/+U 的？

- **四种 `changelog-producer`**：`none` 不产，Flink 流读主键表会插入 `ChangelogNormalize`，状态很重；`input` 把完整 CDC 输入直接当 changelog；`lookup` 在每次合并时产出；`full-compaction` 在全量合并时对比产出，延迟取决于全量合并的频率。
- **lookup 的时机是 L0 上推**：`ForceUpLevel0Compaction` 强制上推，`lookup-wait` 让 `prepareCommit` 等合并完成，数据和 changelog 进同一个快照。
- **计算**：每个 key 从 outputLevel + 1 逐层 lookup 出旧值作为 before，与 L0 合并得到 after：无 → 有是 +I，有 → 删是 -D，值不同是 -U/+U；值相同只有开了 `changelog-producer.row-deduplicate` 才不输出。

延伸阅读：[Lookup Changelog](/bigdata/paimon/07-lookup-changelog)

### Paimon 的 Flink Sink 如何配合 checkpoint 做两阶段提交，保证 exactly-once？

- **拓扑**：多个 writer（`TableWriteOperator`）+ 并行度为 1 的 committer（`CommitterOperator`）。
- **Prepare**：writer 在 `prepareSnapshotPreBarrier` 中刷出文件，把 `Committable` 先于 barrier 发出；committer 在 `snapshotState` 时合成 `ManifestCommittable`（identifier 就是 checkpointId）存进状态。所以要求 EXACTLY_ONCE，且不支持 unaligned checkpoint。
- **Commit**：`notifyCheckpointComplete(N)` 后提交所有 ≤ N 的 committable，因为前一次通知可能丢失。
- **故障恢复**：`commitUser` 存在状态里，重启不变；恢复时 `filterCommitted` 跳过已提交的，补提交后故意失败一次，让 writer 基于新快照重建。

延伸阅读：[FlinkSink 与 Checkpoint 提交](/bigdata/paimon/08-flink-sink)

### Flink 流式读取 Paimon 时，消费位点是什么？consumer-id 解决了什么问题？

- **结构**：FLIP-27 Source。JobManager 上的 `ContinuousFileSplitEnumerator` 每隔 `continuous.discovery-interval`（默认 10s）追新快照、生成 split，同一 bucket 固定分给同一个 reader，保证同 key 变更有序；所以并行度超过 bucket 数是浪费。
- **位点**：就是一个 `nextSnapshotId`，加上未分配的 split 和 split 内的 `recordsToSkip`。
- **问题**：Flink checkpoint 只记自己的位点，表把快照过期后，重启会报快照已过期。
- **consumer-id**：把进度写进 `consumer/` 目录，`ExpireSnapshotsImpl` 不会过期它还没读的快照；新作业没有状态也能接着读。`consumer.expiration-time` 负责清理死 consumer。

延伸阅读：[FlinkSource 流式读取](/bigdata/paimon/09-flink-source)

### 一张主键表同时出现小文件多、checkpoint 慢、批读慢，你怎么排查？在写放大和读放大之间怎么取舍？

- **先定位**：`t$files` 看文件数和 level 分布，`t$snapshots` 看提交频率，日志看 `Universal compaction due to ...` 的合并原因。
- **小文件**：checkpoint 间隔决定小文件数量，适当加大；调 `num-sorted-run.*`。
- **checkpoint 慢**：lookup / DV 表要等 L0 合并，可改 `lookup-compact=GENTLE`；写入作业设 `write-only=true`，另起独立的 compaction 作业，也避免多个作业合并同一 bucket 引发冲突。
- **批读慢**：merge-on-read 是读放大，开 DV 或定期全量合并，把成本挪到写入端。
- **取舍**：看业务更在乎写入实时性还是查询性能，决定代价放在写还是读。

延伸阅读：[附录：配置速查与调试技巧](/bigdata/paimon/appendix)

## 答题思路

Paimon 的题几乎都能用同一条主线串起来：**写入 → 合并 → 提交 → 读取**。回答时建议三步：

1. **先结论**：一两句话说清机制。「提交是乐观并发，抢快照号失败会重试，删除冲突直接抛异常。」
2. **再讲源码里的关键点**：说出负责这件事的类或配置，以及它的判断条件，比如 `rawConvertible` 什么时候为真、UniversalCompaction 的判断顺序。不用背行号，但要讲得出「为什么这样设计」，比如为什么只能选最新的连续前缀，为什么大文件只升级不重写。
3. **再落到线上**：这个机制在生产上会表现成什么现象、怎么调。比如多个作业合并同一 bucket 会持续 failover，解决办法是 `write-only` 加独立 compaction 作业。

对比类题（和 Hudi / Iceberg、merge-on-read 和删除向量、各种 changelog producer）一律按「原理 → 代价 → 适用场景」回答，说清代价落在写入还是读取。被追问到没读过的细节，直接说明自己了解到哪一层，再给出推断思路，比硬编要好。

## 模拟面试

准备好了就开始一场：5 道题从上面的题库随机抽，逐题作答，每题都会得到打分、点评和要点。用自己的话回答，不会的题可以直接写「不会」，看完要点和延伸阅读再来一场。

<MockInterview topic="paimon" />

## 小结

- 主线是「按 bucket 组织的 LSM 树 + snapshot / manifest 元数据树 + 乐观并发提交」，所有题都能挂到这条主线上。
- 合并、删除向量、lookup changelog 本质上都是取舍：把成本放在写入端还是读取端。
- 和 Flink 配合的题重点是两阶段提交、commitUser 幂等、consumer-id 防过期，要能讲清各种故障时刻会发生什么。
