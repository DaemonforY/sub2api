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

## 结业

学完全部 11 课、各课测验总正确率不低于 80%，再在 [Paimon 模拟面试](/bigdata/interview/paimon) 里拿到 60 分以上，就能在这里领取结业证书。

<CertPanel track="g" />
