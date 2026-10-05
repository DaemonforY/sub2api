---
title: "踩坑实验室 #01：数据一直在进，窗口却一个结果都不出"
description: "一个上游没有数据，Watermark 就停在最小值，整个窗口算子被卡住；withIdleness 能解，但有副作用。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/pitfall-01-cover.webp"}]]
---

# 踩坑实验室 #01：数据一直在进，窗口却一个结果都不出

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-01-cover.webp" alt="Yui和Kai面对被空闲上游卡住的窗口数据流" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai面对被空闲上游卡住的窗口数据流<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-02）。对应课程：第七讲《时间、Watermark 与窗口》。配套示例：`flink-notes/demos/src/main/java/study/time/NoSplitDemo.java`、`WindowDemo.java` 的 `idle` 模式

## 现象

这个问题的特点是“没有任何报错”：

- 作业状态是 RUNNING；
- Source 一直在读数据；
- 日志里没有异常；
- 窗口却一个结果都没有。

很多时候，原因是 Watermark 被某一个“没有数据”的上游卡住了。

## 复现

`NoSplitDemo` 用一个自定义的 FLIP-27 Source：并行度 2，但只有 1 个分片；下游是 2 秒的滚动窗口，运行 12 秒。另一个示例 `WindowDemo idle` 并行度同样是 2，其中一个子任务的数据被全部过滤掉。

`NoSplitDemo` 不加 idleness 时的输出：

```
>>> assign split-0 -> subtask 1
>>> no split left for subtask 0
>>> splits=1 parallelism=2 idleness=false: 0 window firings in 12s
```

子任务 0 分不到分片，12 秒内窗口一次都没有触发。两个示例在本机（Apple M5 Pro）各跑 12 秒的结果：

| 实验 | 不加 idleness | 加 `withIdleness(2s)` |
|---|---|---|
| `NoSplitDemo`：1 个分片，并行度 2 | **0 次**触发 | 20 次 |
| `WindowDemo idle`：并行度 2，一个子任务的数据被全部过滤掉 | **0 次**触发 | 20～24 次（两次运行分别为 24、20） |

加上 idleness 之后，第一次触发的时间每次运行都不一样，测到过 3.9 秒、4.0 秒、5.5 秒，和启动耗时、空闲检测的周期有关。

![NoSplitDemo：子任务 0 分不到分片，12 秒内窗口触发 0 次](/bigdata-img/src/content-plan/assets/png/ep01/xhs-P3.webp)

## 原因

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-01-1.webp" alt="多个上游取最小水位，空闲通道卡住窗口" width="960" height="640" loading="lazy" /><figcaption>多个上游取最小水位，空闲通道卡住窗口<span>AI 生成配图</span></figcaption></figure>

一个算子有多个上游时，它的 Watermark 取所有上游 Watermark 的**最小值**。官方文档“Dealing With Idle Sources”一节说明了这条规则；源码在 `StatusWatermarkValve.java:273-285`，用一个小顶堆维护各个通道的 Watermark，输出的是堆顶的最小值。

每个通道的 Watermark 初始值是 `Long.MIN_VALUE`（`StatusWatermarkValve.java:125`）。如果某个上游一直没有数据，它的 Watermark 就一直停在这个初始值，堆顶永远是它，整个算子的 Watermark 都不会前进，窗口永远等不到“结束”的信号。

Source 并行度大于分片数时，多出来的子任务分不到分片。Flink 框架本身**不会**自动把这样的子任务标记为空闲：`CombinedWatermarkStatus.java:67` 在没有任何分片输出时直接返回，既不发 Watermark，也不标记空闲。

需要说明的是，这里验证的是 Flink 框架本身的行为（自定义 Source）。Kafka 等 connector 在不同版本里可能有自己的处理，具体以所用 connector 版本的文档为准。

![下游 Watermark 取所有上游的最小值，没有数据的子任务停在 MIN，把整个算子卡住](/bigdata-img/src/content-plan/assets/png/ep01/xhs-P4.webp)

## 怎么解决

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-01-2.webp" alt="空闲分区被标记后窗口恢复，但旧数据可能迟到" width="960" height="640" loading="lazy" /><figcaption>空闲分区被标记后窗口恢复，但旧数据可能迟到<span>AI 生成配图</span></figcaption></figure>

**办法一：在 `WatermarkStrategy` 上加 idleness。**

```java
WatermarkStrategy
    .forBoundedOutOfOrderness(Duration.ofSeconds(1))
    .withIdleness(Duration.ofSeconds(2));
```

超过设定时间没有数据的分区会被标记为空闲，不再参与取最小值，窗口恢复触发（`WatermarksWithIdleness.java`）。实测 12 秒内从 0 次变成了 20 次。

但它有副作用：空闲分区恢复之后，在追上之前不参与计算（`StatusWatermarkValve.java:259`）。如果它带来的数据时间比下游当前的 Watermark 还早，这些数据就成了迟到数据。没有配置侧输出时，迟到数据会被静默丢弃，只让指标 `numLateRecordsDropped` 加 1（`WindowOperator.java:440-445`，这一条是源码结论，没有单独实测）。所以：

- 配合 `sideOutputLateData` 把迟到数据收集起来；
- idleness 的超时时间不要设得太短。

**办法二：让 Source 并行度不超过分区数。** 不让任何一个子任务分不到分区，也就不会有“永远停在初始值”的上游。

## 记住这几点

- 下游算子的 Watermark = 所有上游 Watermark 的最小值，一个没有数据的上游就能卡住整个窗口。
- Source 并行度大于分片数时，Flink 框架本身不会自动把空闲子任务标记为空闲；connector 的行为以所用版本为准。
- `withIdleness` 能让窗口恢复触发，但空闲分区恢复后的旧数据可能变成迟到数据，记得配侧输出；也可以把 Source 并行度调到不超过分区数。
:::
