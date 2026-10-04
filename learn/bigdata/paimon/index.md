---
title: "Apache Paimon 源码学习"
description: "流式湖仓的存储内核：合并策略、提交与冲突检测、读路径与删除向量、Changelog、Flink / Spark 集成和 REST Catalog，每个结论都能用实验复现。"
---

# Apache Paimon 源码学习

流式湖仓的存储内核：合并策略、提交与冲突检测、读路径与删除向量、Changelog、Flink / Spark 集成和 REST Catalog，每个结论都能用实验复现。

> 基于 Paimon 2.0 / master。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验见 [GitHub](https://github.com/DaemonforY/paimon-learning)。

## 目录

| # | 文章 | 讲什么 |
|---|---|---|
| 1 | [学习计划](/bigdata/paimon/00-plan) | 14 周从入门到专家的 Paimon 学习路线，每周的目标、阅读材料和实验。 |
| 2 | [环境搭建与源码编译](/bigdata/paimon/01-setup) | 编译 Paimon 源码、导入 IDEA、跑通第一个实验，以及编译中的常见问题。 |
| 3 | [paimon-core 整体架构](/bigdata/paimon/02-architecture) | paimon-core 的分层、snapshot 到 data file 的元数据树，写、提交、读三条主流程。 |
| 4 | [UniversalCompaction 选文件策略](/bigdata/paimon/03-universal-compaction) | UniversalCompaction 怎样根据 sorted run 的数量和大小选出要合并的文件。 |
| 5 | [MergeTreeCompactTask 合并执行](/bigdata/paimon/04-compact-task) | MergeTreeCompactTask 的执行：哪些文件直接升级层级，哪些需要重写合并。 |
| 6 | [提交：冲突检测与重试](/bigdata/paimon/05-commit) | FileStoreCommitImpl 的乐观并发提交：冲突检测、重试和幂等。 |
| 7 | [读路径与删除向量](/bigdata/paimon/06-read-path) | 主键表的读路径 MergeFileSplitRead，以及删除向量怎样避免读时合并。 |
| 8 | [Lookup Changelog](/bigdata/paimon/07-lookup-changelog) | lookup changelog producer 怎样产生 -U/+U 变更记录。 |
| 9 | [FlinkSink 与 Checkpoint 提交](/bigdata/paimon/08-flink-sink) | Paimon 的 Flink Sink：写入、PrepareCommit 与 Checkpoint 配合的两阶段提交，保证 exactly-once。 |
| 10 | [FlinkSource 流式读取](/bigdata/paimon/09-flink-source) | Paimon 的 Flink 流式读取：增量 split 的生成、consumer-id 与过期保护。 |
| 11 | [Spark MERGE INTO](/bigdata/paimon/10-spark-merge-into) | Spark MERGE INTO 在 Paimon 中的三条实现路径。 |
| 12 | [REST Catalog](/bigdata/paimon/11-rest-catalog) | REST Catalog：服务端 CAS 提交与临时凭证下发。 |
| 13 | [附录：配置速查与调试技巧](/bigdata/paimon/appendix) | 常用配置速查、源码索引、调试技巧和排障表。 |
| 14 | [勘误与版本说明](/bigdata/paimon/errata) | 教程的勘误记录和源码 / 实验的版本说明。 |

读的时候有疑问，可以打开源码对照；每篇开头都标了源码版本和路径缩写。

读完想检验一下？去做 [Paimon 面试题](/bigdata/interview/paimon)，AI 面试官会逐题打分、点评。
