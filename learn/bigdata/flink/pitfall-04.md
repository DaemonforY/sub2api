---
title: "踩坑实验室 #04：按天统计，为什么凌晨的订单算到了前一天？"
description: "Flink 窗口按 UTC 纪元对齐，一天的窗口从北京时间 8 点开始，加 −8 小时 offset 才对。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/pitfall-04-cover.webp"}]]
---

# 踩坑实验室 #04：按天统计，为什么凌晨的订单算到了前一天？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-04-cover.webp" alt="Yui和Kai将订单窗口从世界时间对齐到北京时间" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai将订单窗口从世界时间对齐到北京时间<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-02）。对应课程：第七讲（下）《窗口，从分配到触发》。配套示例：`flink-notes/demos/src/main/java/study/time/DailyWindowDemo.java`（`nooffset`、`offset`、`sizes` 模式）

## 现象

用一天的滚动窗口 `TumblingEventTimeWindows.of(Duration.ofDays(1))` 统计订单，结果每天的订单数都和数据库对不上：10 月 1 日凌晨的订单被算进了 9 月 30 日，10 月 2 日早上 7 点半的订单反而算进了 10 月 1 日。仔细看窗口边界，每个“一天”都是从北京时间早上 8 点开始的。

## 复现

`DailyWindowDemo`：并行度 1，6 笔订单，时间都是北京时间。跑了两次，输出完全一致：

```
# of(Duration.ofDays(1))
>>> window=[09-30 08:00, 10-01 08:00) 北京时间, start(UTC 毫秒)=1790726400000, count=2, orders=[10-01 00:30, 10-01 07:59]
>>> window=[10-01 08:00, 10-02 08:00) 北京时间, start(UTC 毫秒)=1790812800000, count=4, orders=[10-01 08:00, 10-01 12:00, 10-01 23:30, 10-02 07:30]

# of(Duration.ofDays(1), Duration.ofHours(-8))
>>> window=[10-01 00:00, 10-02 00:00) 北京时间, start(UTC 毫秒)=1790784000000, count=5, orders=[10-01 00:30, 10-01 07:59, 10-01 08:00, 10-01 12:00, 10-01 23:30]
>>> window=[10-02 00:00, 10-03 00:00) 北京时间, start(UTC 毫秒)=1790870400000, count=1, orders=[10-02 07:30]
```

不是所有窗口大小都会错位。`sizes` 模式直接调用 Flink 的 `TimeWindow.getWindowStartWithOffset`，看北京时间 10-01 09:30 这一时刻在不加 offset 时落在哪个窗口：

| 窗口大小 | 09:30 所在窗口 | 起点对齐北京时间的整倍数？ |
|---|---|---|
| 1 小时 | `09:00, 10:00)` | 是 |
| 2 小时 | `[08:00, 10:00)` | 是 |
| 3 小时 | `[08:00, 11:00)` | **否** |
| 4 小时 | `[08:00, 12:00)` | 是 |
| 6 小时 | `[08:00, 14:00)` | **否** |
| 8 小时 | `[08:00, 16:00)` | 是 |
| 12 小时 | `[08:00, 20:00)` | **否** |
| 24 小时 | `[10-01 08:00, 10-02 08:00)` | **否** |

比如 3 小时的窗口，期望从 0 点、3 点、6 点、9 点开始，实际是从 8 点开始；12 小时的窗口，期望 0 点、12 点，实际是 8 点、20 点。

![哪些窗口会错位：1、2、4、8 小时不受影响，3、6、12 小时和一天会错位

## 原因

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-04-1.webp" alt="订单被按世界时间分进从北京时间早上开始的窗口" width="960" height="640" loading="lazy" /><figcaption>订单被按世界时间分进从北京时间早上开始的窗口<span>AI 生成配图</span></figcaption></figure>

滚动窗口的起点 = 时间戳 − (时间戳 − offset) % 窗口大小（`TimeWindow.java:264-272` 的 `getWindowStartWithOffset`，由 `TumblingEventTimeWindows.java:78` 的 `assignWindows` 调用）。

时间戳是从 1970-01-01 00:00:00 **UTC** 开始的毫秒数，所以不加 offset 时，窗口对齐到 UTC 的整天、整小时上。实测的窗口起点 1790726400000 正好是一天毫秒数（86400000）的 20726 倍。UTC 的零点就是北京时间早上 8 点，于是一天的窗口变成了北京时间“今天 8 点到明天 8 点”。

公式里只有时间戳、offset 和窗口大小，所以窗口的划分和作业在哪台机器上跑、机器设的是什么时区**无关**，改服务器时区解决不了问题。这也不是 bug：这是按 UTC 纪元对齐的设计，Javadoc 里写明了 offset 的用法。

至于哪些大小会错位，实测的规律是：8 小时不是窗口大小的整数倍时才会错位。1、2、4、8 小时的窗口没问题；3、6、12 小时和一天的窗口会错位。

![窗口按 UTC 零点对齐：UTC 00:00 对应北京时间 08:00，凌晨的那一段被算到了前一天](/bigdata-img/src/content-plan/assets/png/ep04/xhs-P3.webp)

## 怎么解决

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-04-2.webp" alt="向后移动窗口锚点后订单按北京时间自然归档" width="960" height="640" loading="lazy" /><figcaption>向后移动窗口锚点后订单按北京时间自然归档<span>AI 生成配图</span></figcaption></figure>

给窗口加一个 **−8 小时**的 offset：

```java
TumblingEventTimeWindows.of(Duration.ofDays(1), Duration.ofHours(-8))
```

注意是负 8，不是正 8。Flink 自己的 Javadoc 就用中国举了这个例子（`TumblingEventTimeWindows.java:117-121`），原文的理由是 “since UTC+08:00 is 8 hours earlier than UTC time”。加上之后再跑，10 月 1 日的 5 笔订单都在同一个窗口里了。offset 还必须满足 `abs(offset) < size`（`TumblingEventTimeWindows.java:58-61`）。

几个容易走偏的方向：

- **改服务器时区**：没用，窗口划分和机器时区无关。
- **用 `WindowStagger.RANDOM`**：它会让窗口边界随机偏移，结果更不对。
- **有夏令时的时区**：本文没有验证。按公式推断，一个固定的 offset 在夏令时期间会差 1 小时；北京时间没有夏令时，−8 小时全年都对。
- **SQL 的 `TUMBLE` 窗口函数**：在不同时间类型下的行为本文没有验证，不要直接套用这里的结论。

## 记住这几点

- DataStream 的滚动窗口按 UTC 纪元对齐，和服务器时区无关；一天的窗口默认是北京时间 8 点到次日 8 点。
- 一天、12 小时、6 小时、3 小时这类窗口要加 offset，北京时间用 `Duration.ofHours(-8)`，是负 8。
- 1、2、4、8 小时的窗口不受影响，不需要加 offset。
:::
