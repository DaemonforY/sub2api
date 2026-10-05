---
title: "Apache Paimon 源码学习"
description: "流式湖仓的存储内核：合并策略、提交与冲突检测、读路径与删除向量、Changelog、Flink / Spark 集成和 REST Catalog，每个结论都能用实验复现。"
---

# Apache Paimon 源码学习

<TrackPage id="g" />

## 这条路线讲什么

流式湖仓的存储内核：合并策略、提交与冲突检测、读路径与删除向量、Changelog、Flink / Spark 集成和 REST Catalog，每个结论都能用实验复现。

> 基于 Paimon 2.0 / master。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验见 [GitHub](https://github.com/DaemonforY/paimon-learning)。

每课读完做一下课后小测验；读的时候有疑问，点右下角「问助教」，助教会结合这篇文章回答。

## 配套阅读

- [学习计划](/bigdata/paimon/00-plan)：14 周从入门到专家的 Paimon 学习路线，每周的目标、阅读材料和实验。
- [附录：配置速查与调试技巧](/bigdata/paimon/appendix)：常用配置速查、源码索引、调试技巧和排障表。
- [勘误与版本说明](/bigdata/paimon/errata)：教程的勘误记录和源码 / 实验的版本说明。

## S1 · 30 分钟玩转 Paimon

入门系列，从看得见的现象讲起：不用装集群，跟着做就能看懂 Paimon 的文件和快照。

- [第 1 期：一个 Java 程序跑通 Paimon](/bigdata/paimon/s1-01)：Flink 的 Table API 自带一个嵌入式的 MiniCluster，不用下载和启动 Flink 集群，一个普通的 Java 程序就能执行 Flink SQL、读写 Paimon。好处是启动快、可以直接在 IDE
- [第 2 期：表目录里都是什么](/bigdata/paimon/s1-02)：一张 Paimon 表就是一个目录。数据在 分区/bucket-N/ 下的 parquet 文件里；“哪些数据文件属于当前版本”由 snapshot → manifest list → manifest 三层元数据描述。
- [第 3 期：主键表 vs Append 表](/bigdata/paimon/s1-03)：Paimon 有两种表。主键表按主键去重，同一主键只保留（合并后的）一条，底层是 LSM 树，读时要按主键归并；Append 表没有主键，写什么存什么，读时直接读文件。订单、用户、维表、CDC 同步用主键表；日志、埋点、
- [第 4 期：删一行为什么多了文件](/bigdata/paimon/s1-04)：在 Paimon 主键表里执行 DELETE，不会修改或删除任何已有文件，而是新写一个只含“删除标记（-D）”的小文件。读取时按主键取最新版本，最新的是删除标记，这一行就不输出；合并时删除标记和旧数据一起被丢弃；旧文件要
- [第 5 期：3 行变 2 行](/bigdata/paimon/s1-05)：Paimon 主键表的写入先进内存里的写缓冲，不会立刻生成文件；提交前刷盘时，缓冲里的数据按“主键 + 写入顺序号”排好序，同一主键的多条记录交给合并引擎合成一条再写进 L0 文件。所以一次写 3 行、其中两行主键相同，
- [第 6 期：时间旅行](/bigdata/paimon/s1-06)：Paimon 每次提交生成一个快照，快照只是一个几百字节的 JSON，经 manifest list → manifest 最终指向一组数据文件。新快照复用旧快照的数据文件，只加入本次新写的，所以保存多个历史版本不需要复
- [第 7 期：快照删了文件还在](/bigdata/paimon/s1-07)：快照只是一份“文件清单”，同一个数据文件会被多个快照共享。过期一个快照时，Paimon 不会去扫描所有快照找引用，而是只看后面的快照把哪些文件标记成了 DELETE，再排除仍被 Tag 引用的，剩下的才物理删除。所以在实
- [第 8 期：Tag](/bigdata/paimon/s1-08)：Tag 就是把某个快照的 JSON 原样复制一份放到 tag/ 目录，不复制任何数据。它和快照内容相同，但不会被快照过期清理，所以能长期保留一个数据版本。Tag 可以设保留时间（到期后在下一次提交时删除），可以用 rol

## S2 · 源码精读图文版

和课程同一套内容的图文版，配有用 jdb 抓到的真实调用栈。

- [第 1 讲：全景图](/bigdata/paimon/s2-01)：paimon-core 可以看成四层——Catalog、Table、FileStore、存储结构。Paimon = 按 bucket 组织的 LSM 树（数据） + snapshot / manifest 元数据树（版本
- [第 2 讲：一条数据的写入之旅](/bigdata/paimon/s2-02)：一行数据写进 Paimon 主键表要经过六站：① 按 abs(主键哈希 % 桶数) 算出 bucket（Source 和 Writer 线程各算一次）；② 每个 (分区, bucket) 第一次出现时才创建 writer
- [第 3 讲：合并怎么选文件](/bigdata/paimon/s2-03)：Paimon 主键表默认用 UniversalCompaction 决定合并哪些文件。它挑的不是文件，而是 sorted run：L0 每个文件一个 run，L1~L5 每层一个 run，底座也算。pick() 依次判断
- [第 4 讲：合并怎么执行](/bigdata/paimon/s2-04)：第 3 讲的 UniversalCompaction 只决定“合并哪些 run、输出到哪层”，真正干活的是 MergeTreeCompactTask。它先把输入文件按 key 范围切成互不相交的 section，再逐个处
- [第 5 讲：并发提交](/bigdata/paimon/s2-05)：Paimon 提交不锁表。每次提交先把数据文件和 manifest 都写好，最后抢一个文件名 snapshot-(N+1)：rename 成功就赢，失败说明被别人抢了，重读最新快照、退避后重试——实验里两个作业并发提交 
- [第 6 讲：读路径](/bigdata/paimon/s2-06)：主键表有两条读路径——直接读（RawFileSplitRead，像 append 表一样顺序读文件）和合并读（MergeFileSplitRead，按 key 归并多个文件、同 key 取最新、最后去掉删除记录）。走哪条
- [第 7 讲：删除向量](/bigdata/paimon/s2-07)：默认的主键表把同一个 key 的多个版本留到读的时候归并去重（第 6 讲）。开启 deletion-vectors.enabled 后，写入时 L0 文件上推，会到高层查（lookup）同一个 key 的旧版本，把它“在
- [第 8 讲：Lookup Changelog](/bigdata/paimon/s2-08)：第一季第五讲（下）讲过，下游按某个值做聚合时，需要 -U 告诉它旧值是什么，才能把旧值撤回。Paimon 主键表默认（changelog-producer = none）不存这种变更，流读时 Flink 要加一个 Cha
- [第 9 讲：Flink 写入两阶段提交](/bigdata/paimon/s2-09)：第一季第八讲（下）讲过 Flink Sink V2 的两阶段提交：barrier 之前 prepareCommit，checkpoint 完成后 commit，恢复时“一律再提交一次”，所以提交必须幂等。Paimon 的
- [第 10 讲：Flink 流读与 consumer-id](/bigdata/paimon/s2-10)：第一季第八讲（上）拆过 FLIP-27：JobManager 上的 Enumerator 发现并分配 split，TaskManager 上的 Reader 读 split。Paimon 的默认流读就是一个标准的 FLI
- [第 11 讲：Spark MERGE INTO](/bigdata/paimon/s2-11)：Paimon 的 Spark MERGE INTO 有三条落地方式，但前半段是同一个引擎：source 和 target 做 full outer join，每一行按 MATCHED / NOT MATCHED / NO

## 结业

学完全部 11 课、各课测验总正确率不低于 80%，再在 [Paimon 模拟面试](/bigdata/interview/paimon) 里拿到 60 分以上，就能在这里领取结业证书。

<CertPanel track="g" />
