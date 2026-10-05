---
title: "Apache Spark 源码学习"
description: "从编译源码、跟踪一个 Job 的一生，到调度、Shuffle、内存、Catalyst、Codegen 和 AQE，每个结论都给出源码位置和可复现的实验。"
---

# Apache Spark 源码学习

<TrackPage id="e" />

## 这条路线讲什么

从编译源码、跟踪一个 Job 的一生，到调度、Shuffle、内存、Catalyst、Codegen 和 AQE，每个结论都给出源码位置和可复现的实验。

> 基于 Spark 4.2.0。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验见 [GitHub](https://github.com/DaemonforY/spark-source-notes)。

每课读完做一下课后小测验；读的时候有疑问，点右下角「问助教」，助教会结合这篇文章回答。

## 配套阅读

- [学习路线：从入门到 Committer](/bigdata/spark/roadmap)：L0 预备到 L4 专家五个阶段：每阶段读哪些源码、做哪些练习、怎样参与社区。

## 结业

学完全部 13 课、各课测验总正确率不低于 80%，再在 [Spark 模拟面试](/bigdata/interview/spark) 里拿到 60 分以上，就能在这里领取结业证书。

<CertPanel track="e" />
