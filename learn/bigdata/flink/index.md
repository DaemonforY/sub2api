---
title: "Apache Flink 源码学习"
description: "从 env.execute() 到 processElement()，Checkpoint、状态后端、网络栈与背压、Table/SQL、调度容错、Watermark 与窗口、新版 Source/Sink。"
---

# Apache Flink 源码学习

<TrackPage id="f" />

## 这条路线讲什么

从 env.execute() 到 processElement()，Checkpoint、状态后端、网络栈与背压、Table/SQL、调度容错、Watermark 与窗口、新版 Source/Sink。

> 基于 Flink 2.3.0。作者 X老师（[DaemonforY](https://github.com/DaemonforY)），按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。

每课读完做一下课后小测验；读的时候有疑问，点右下角「问助教」，助教会结合这篇文章回答。

## 结业

学完全部 9 课、各课测验总正确率不低于 80%，再在 [Flink 模拟面试](/bigdata/interview/flink) 里拿到 60 分以上，就能在这里领取结业证书。

<CertPanel track="f" />
