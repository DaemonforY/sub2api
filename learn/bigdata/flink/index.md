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

## 图文版：Flink 2.x 源码精读

和课程同一套内容的图文版：每篇从一个可以复现的现象讲起，适合先读它再读对应课时。

- [零：从哪里开始读](/bigdata/flink/post-00)：Flink 2.3 有 147 万行生产代码，老资料里的很多路径已经失效。本篇给出一条从入门到 Committer 的学习路线，并带你把 2.3.0 源码克隆、编译、跑起来，所有命令都在本机实测过。
- [一·上：作业图怎么生成](/bigdata/flink/post-01a)：同一份 WordCount，IDE 里跑出 3 个 Task，集群上只有 2 个。顺着这个现象，读懂 Flink 2.3 从 API 调用到 StreamGraph、再到 JobGraph 和算子链的全过程，每一步都给出源码位置和本机实测输出。
- [一·下：算子为什么不用加锁](/bigdata/flink/post-01b)：open、processElement、定时器、Checkpoint 快照、Checkpoint 完成通知，我在一个算子的 6 个回调里打印了线程名，结果全是同一个线程。顺着这个现象，读懂 Flink 2.3 从调度、部署到 Task 线程和 Mailbox 循环的全过程。
- [二·上：背压下的 Checkpoint](/bigdata/flink/post-02a)：同一个作业，有背压时 Checkpoint 从 15 毫秒涨到 3 秒，但快照本身只花了不到 1 毫秒。顺着这个现象，读懂 Flink 2.3 Checkpoint 从触发、Barrier 对齐、快照到完成通知的全过程，并实测"一次超时就全局重启"。
- [二·下：非对齐 Checkpoint](/bigdata/flink/post-02b)：同样的背压，开启非对齐 Checkpoint 后耗时从 3 秒降到 16 毫秒，Checkpoint 却从 5.7KB 涨到 600KB。拆开来看，这 600KB 几乎全部是被 Barrier 越过的、堆在 Source 输出缓冲区里的数据。读懂 Flink 2.3 非对齐 Checkpoint 的实现、代价和推荐配置。
- [三·上：读懂 KeyGroup](/bigdata/flink/post-03a)：同一个 Checkpoint，并行度从 2 改成 3，1000 个 key 一个不少地恢复了；maxParallelism 从 128 改成 256，作业直接起不来；而如果你从没设置过它，Flink 会悄悄沿用旧值。读懂 Flink 2.3 的 KeyGroup、HashMap 状态后端和扩缩容。
- [三·下：换成 RocksDB 之后](/bigdata/flink/post-03b)：一段在 HashMap 状态后端上运行正确的计数代码，换成 RocksDB 后，4 个 key 的计数全部是 0。顺着这个陷阱，读懂 Flink 2.3 中 RocksDB 的存储格式、增量 Checkpoint 每次到底上传了什么、扩缩容时状态怎么裁剪，以及 ForSt 和异步状态。
- [四：一个下游慢，另一个也停](/bigdata/flink/post-04)：两个下游子任务，只让一个变慢，另一个本来每 5 秒能处理 3600 万条，结果只处理了几千条，全程空等。顺着这个现象，读懂 Flink 2.3 的网络缓冲区、Credit 流控、背压的传导链，以及 Buffer Debloating 为什么能把 Checkpoint 从 2.8 秒降到 1 秒。
- [五·上：SQL 到 Transformation](/bigdata/flink/post-05a)：把 agg-phase-strategy 设成 TWO_PHASE，执行计划和默认配置一模一样，也没有任何警告。顺着这个现象，走一遍 Flink 2.3 中一条 SQL 从解析、校验、优化、ExecNode 到代码生成的全过程，看两阶段聚合到底要满足哪几个条件。
- [五·下：Changelog 为什么输出两条](/bigdata/flink/post-05b)：同一条 GROUP BY，写到 print 表会输出 -U/+U，写到 blackhole 表只输出 +U。顺着这个现象，读懂 Flink 2.3 的 Changelog 推导：上游输出什么由下游决定；聚合算子什么时候发 -U、+U、-D；MiniBatch 为什么能少发消息；以及 2.3 新增的 ON CONFLICT 检查。
- [六：Task 失败之后](/bigdata/flink/post-06)：两个作业，同样让一个 Task 抛异常，一个重启了 4 个 Task，另一个只重启了 1 个。顺着这个现象，读懂 Flink 2.3 的 Failover 全流程、Pipelined Region、默认的指数退避重启策略，以及 AdaptiveScheduler 如何按资源自动扩缩容。
- [七·上：Watermark 从哪来](/bigdata/flink/post-07a)：同样的数据，同样的 forBoundedOutOfOrderness(1s)，写在 fromSource 里 0 条迟到，写成 assignTimestampsAndWatermarks 有 197 条被判迟到。顺着这个现象，读懂 Flink 2.3 中 Watermark 的生成、Split 级别的合并、在网络中的传播，以及它如何驱动 Timer。
- [七·下：窗口从分配到触发](/bigdata/flink/post-07b)：用一天的滚动窗口按天统计订单，10 月 1 日凌晨的两笔订单却被算进了"9 月 30 日 8 点到 10 月 1 日 8 点"的窗口。顺着这个现象，读懂 Flink 2.3 窗口的分配、两个 Timer、触发与清理、迟到数据的四种结局，以及会话窗口的合并。
- [八·上：Source 分片](/bigdata/flink/post-08a)：split-2 在 Checkpoint 5 之后才分给 subtask 1，subtask 1 读了第一条就挂了。恢复后，split-2 被还给了 Enumerator，160 条记录一条不丢、一条不重。顺着这个现象，读懂 Flink 2.3 新 Source 架构：Enumerator 与 Reader、split 追踪、Coordinator 的"关门"机制，以及 fetcher 线程模型。
- [八·下：Sink 与 exactly-once](/bigdata/flink/post-08b)：每秒 10 条，Checkpoint 每 2 秒一次，用默认配置的 FileSink 写文件，第一个文件直到第 120 秒才出现，最早的数据等了 120 秒。换成 OnCheckpointRollingPolicy，最多 2.1 秒。顺着这个现象，读懂 Flink 2.3 Sink V2 的两阶段提交与端到端 exactly-once。
- [九：从读者到贡献者](/bigdata/flink/post-09)：2026 年 9 月，Flink 主仓库 133 个 commit 中有 35 个声明使用了 AI 工具。社区的规则是什么？一个合格的 bug 修复长什么样？读完这个系列，有哪些可以动手的第一批任务？全系列收官。

## 踩坑实验室

生产中常见的 21 个坑，每个都在本机复现过：现象、复现数据、源码里的原因和解决办法。

- [#01 窗口不出结果](/bigdata/flink/pitfall-01)：一个上游没有数据，Watermark 就停在最小值，整个窗口算子被卡住；withIdleness 能解，但有副作用。
- [#02 窗口输出两次](/bigdata/flink/pitfall-02)：窗口触发后状态还在，allowedLateness 内的迟到数据会让窗口带着完整新结果再输出一次。
- [#03 Watermark 差 1 毫秒](/bigdata/flink/pitfall-03)：两个“减 1”：Watermark 是最大事件时间减乱序再减 1，窗口左闭右开，6999 不触发、7000 才触发。
- [#04 一天的窗口早上 8 点才关](/bigdata/flink/pitfall-04)：Flink 窗口按 UTC 纪元对齐，一天的窗口从北京时间 8 点开始，加 −8 小时 offset 才对。
- [#05 背压时窗口迟迟不出结果](/bigdata/flink/pitfall-05)：Watermark 和数据走同一条通道，背压时被堵在旧数据后面；开启 Buffer Debloating 可缓解。
- [#06 exactly-once 还重复](/bigdata/flink/pitfall-06)：作业恢复后 Flink 会把已提交过的事务再提交一次，Committer 必须幂等，事务 ID 必须唯一。
- [#07 不开 Checkpoint 不提交](/bigdata/flink/pitfall-07)：两阶段提交的两步都挂在 Checkpoint 上，无界流作业不开 Checkpoint，事务型 Sink 永不提交。
- [#08 retryLater 不会稍后](/bigdata/flink/pitfall-08)：自己实现 Committer 时，retryLater() 会在毫秒内立即重试，默认 10 次用完作业就失败。
- [#09 只重启了一部分](/bigdata/flink/pitfall-09)：Flink 默认按 pipelined region 重启，keyBy、rebalance 会把整个作业连成一个 region。
- [#10 不开 Checkpoint 不重启](/bigdata/flink/pitfall-10)：没配重启策略时，Flink 看是否开了 Checkpoint：没开就不重启，只配重启策略状态也会从 0 开始。
- [#11 FileSink 数据等了 120 秒](/bigdata/flink/pitfall-11)：Row 格式 FileSink 的默认滚动策略不在 Checkpoint 时关文件，数据要等一两分钟才可见。
- [#12 最大并行度改了恢复不了](/bigdata/flink/pitfall-12)：并行度可以改，maxParallelism 一改状态就恢复不了；从没设置过的作业会锁定在首次推算的值上。
- [#13 扩缩容时状态怎么切](/bigdata/flink/pitfall-13)：扩缩容时 keyed state 按 KeyGroup 区间求交集，非整数倍调整会让部分子任务读多份旧状态。
- [#14 非对齐 Checkpoint 快在哪](/bigdata/flink/pitfall-14)：背压下对齐 Checkpoint 慢在 Barrier 排队；非对齐快了，但要把在途数据一起存下来。
- [#15 背压一级一级传上去](/bigdata/flink/pitfall-15)：背压沿数据流一级一级往上游传；真正的瓶颈不被背压、但很忙。
- [#16 两阶段聚合没生效](/bigdata/flink/pitfall-16)：流模式下两阶段聚合要先开 MiniBatch；用 PLAN_ADVICE 能直接看到缺哪几项配置。
- [#17 print 出来的 -U 和 +U](/bigdata/flink/pitfall-17)：Flink SQL 的结果是会变化的表：读懂 +I、-U、+U、-D，以及谁决定要不要 -U。
- [#18 算子为什么没链在一起](/bigdata/flink/pitfall-18)：8 种写法实测哪些会让算子链断开，以及 disableChaining 和 startNewChain 的区别。
- [#19 单线程的两个坑](/bigdata/flink/pitfall-19)：Mailbox 单线程模型的两个坑：在回调里阻塞，以及在自己开的线程里改 keyed state。
- [#20 分区会不会读两次](/bigdata/flink/pitfall-20)：故障恢复后分片不会被两个 subtask 同时读，但数据会重读；不重复靠的是两阶段提交。
- [#21 配置项已删除不报错](/bigdata/flink/pitfall-21)：Flink 2.0 删除的四个网络缓冲区配置项，在 2.3 里设成 abc 也能正常运行。

## 结业

学完全部 9 课、各课测验总正确率不低于 80%，再在 [Flink 模拟面试](/bigdata/interview/flink) 里拿到 60 分以上，就能在这里领取结业证书。

<CertPanel track="f" />
