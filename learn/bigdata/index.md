---
title: 大数据
description: Apache Spark、Flink、Paimon 源码学习：基于最新版本源码，每个结论都给出源码位置和可复现的实验。
---

# 大数据

批处理、流处理、湖仓存储三块，各选一个代表项目读到源码层面：**Spark** 讲清一个批作业从提交到执行的每一步，**Flink** 讲清流作业的状态、Checkpoint 和时间语义，**Paimon** 讲清流式湖仓怎样把数据写进 LSM 树、怎样提交和读取。

<BigdataHub />

三条都是正式的学习路线：每课读完有 4 道小测验，读的时候可以随时问 AI 助教；学完全部课程、测验总正确率不低于 80%，再通过这条路线的模拟面试（60 分以上），就能领取结业证书。

## 怎么学

- **有 Spark 使用经验**：从 [Spark 学习路线](/bigdata/spark/roadmap) 开始，先编译源码、跟一遍 SparkPi，再按讲次往下读。
- **做实时计算**：从 [Flink 作业执行全链路](/bigdata/flink/01-execution) 开始，Checkpoint 和状态后端两篇是重点。
- **做湖仓 / 数据集成**：读 [Paimon 学习计划](/bigdata/paimon/00-plan)，按 14 周的安排边读边做实验；Flink 两篇（08、09）讲 Paimon 和 Flink 的配合。
- 准备面试：每个系列都有一套面试题和 AI 模拟面试——[Spark](/bigdata/interview/spark)、[Flink](/bigdata/interview/flink)、[Paimon](/bigdata/interview/paimon)。每场随机抽 5 题，AI 面试官按 0–10 分打分、点评，并告诉你好的回答应该覆盖哪些要点。AI 方向的面试题在 [D · AI 面试训练](/d/)。

每篇开头都标了源码版本和路径缩写，`文件:行号` 都在对应版本里核对过。建议把源码拉到本地，边读边在 IDE 里跳转。

::: info 作者和许可
这些内容由 X老师（[DaemonforY](https://github.com/DaemonforY)）编写，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。Spark 和 Paimon 的配套实验在 GitHub：[spark-source-notes](https://github.com/DaemonforY/spark-source-notes)、[paimon-learning](https://github.com/DaemonforY/paimon-learning)。本站与 Apache 软件基金会无隶属关系，Apache Spark、Flink、Paimon 是 Apache 软件基金会的商标。
:::
