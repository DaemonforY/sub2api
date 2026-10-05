---
title: "踩坑实验室 #03：Watermark 为什么总是差 1 毫秒？"
description: "两个“减 1”：Watermark 是最大事件时间减乱序再减 1，窗口左闭右开，6999 不触发、7000 才触发。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/pitfall-03-cover.webp"}]]
---

# 踩坑实验室 #03：Watermark 为什么总是差 1 毫秒？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-03-cover.webp" alt="Yui和Kai在时间河流旁观察水位线与窗口闸门" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在时间河流旁观察水位线与窗口闸门<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-02）。对应课程：第七讲《时间、Watermark 与窗口》。配套示例：`flink-notes/demos/src/main/java/study/time/WatermarkBoundaryDemo.java`

## 现象

5 秒的滚动窗口，允许乱序 2 秒。按顺序发数据：事件时间到了 6.999 秒，Watermark 是 4.998 秒，`[0s,5s)` 窗口不动；到了 7 秒，Watermark 变成 4.999 秒，窗口才触发。日志里的 Watermark 也总是 x.999 这样的数字。

这不是 Flink 算错了，而是 Watermark 语义要求的结果。

## 复现

`WatermarkBoundaryDemo`：并行度 1，`forBoundedOutOfOrderness(Duration.ofSeconds(2))`，5 秒滚动窗口，`allowedLateness = 0`，迟到数据走侧输出，每 500 毫秒发一条。跑了两次，输出逐行一致：

```
EVENT ts=1s      watermark=MIN
EVENT ts=4.500s  watermark=-1.001s
EVENT ts=6s      watermark=2.499s
EVENT ts=6.999s  watermark=3.999s
EVENT ts=7s      watermark=4.998s
FIRE  window=[0s,5s) maxTimestamp=4.999s elements=[1s, 4.500s] watermark=4.999s
EVENT ts=4.999s  watermark=4.999s
LATE  ts=4.999s (side output)
EVENT ts=5s      watermark=4.999s
FIRE  window=[5s,10s) maxTimestamp=9.999s elements=[6s, 6.999s, 7s, 5s] watermark=MAX
```

每行 `EVENT` 打印的是这条记录**到达时**的 Watermark，也就是上一条记录之后发出的那个，所以要错一行看：

| 已经见过的最大事件时间 | 之后发出的 Watermark | `0s,5s)` 窗口 |
|---|---|---|
| 1000 | 1000 − 2000 − 1 = **−1001** | 等待 |
| 4500 | **2499** | 等待 |
| 6000 | **3999** | 等待 |
| 6999 | **4998** | 等待，**差 1 毫秒** |
| 7000 | **4999** | **触发** |

还没有任何数据时，Watermark 恰好是 `Long.MIN_VALUE`（第一条记录看到的 `MIN`，源码 `BoundedOutOfOrdernessWatermarks.java:57`）。

![实测输出：数据到 6.999 秒时 Watermark 是 4.998，到 7 秒才变成 4.999 并触发窗口

## 原因

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-03-1.webp" alt="Yui和Kai观察落后于最大事件时间的安全水位线" width="960" height="640" loading="lazy" /><figcaption>Yui和Kai观察落后于最大事件时间的安全水位线<span>AI 生成配图</span></figcaption></figure>

**第一个“减 1”在 Watermark 上。** Watermark `T` 的含义是：时间戳**小于或等于** `T` 的数据不会再来了（`Watermark.java:27-28` 的类注释）。`forBoundedOutOfOrderness(d)` 发出的 Watermark = **目前见过的最大事件时间 − d − 1 毫秒**（`BoundedOutOfOrdernessWatermarks.java:69`）。见过的最大时间是 7000、乱序容忍 2000 时，时间戳 5000 的数据还可能到来，4999 不会再来，所以 Watermark 只能是 4999。注意这里是“见过的最大事件时间”，和机器的当前时间无关。

`forMonotonousTimestamps()` 是乱序容忍为 0 的特例：`AscendingTimestampsWatermarks` 继承 `BoundedOutOfOrdernessWatermarks`，传入 `Duration.ofMillis(0)`，Watermark = 最大事件时间 − 1。

**第二个“减 1”在窗口上。** 窗口是左闭右开的 `start, end)`，`maxTimestamp = end − 1`（`TimeWindow.java:84-86`）。触发 Timer 注册在 `maxTimestamp` 上（`EventTimeTrigger.java:44`），Timer 在“时间 ≤ Watermark”时执行（`InternalTimerServiceImpl.java:334-335`），所以窗口在 **Watermark ≥ end − 1** 时触发。

两个“减 1”合起来：`最大时间 − d − 1 ≥ end − 1`，也就是**某条数据的事件时间 ≥ end + d**，窗口 `[start, end)` 才会触发。5 秒窗口、乱序 2 秒，就要等到事件时间 7 秒的数据。

还有一个边界：Watermark 已经等于 `end − 1` 时，再来一条时间戳正好是 4.999 秒的数据，它就在窗口的最后 1 毫秒，但窗口已经关闭（`allowedLateness = 0` 时，`WindowOperator.java:609-612` 以“清理时间 ≤ Watermark”判定窗口已关闭），只能当作迟到数据进侧输出。

![第一个“减 1”：见过 7000、乱序 2000，5000 还可能来，Watermark 只能是 4999

## 怎么解决

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-03-2.webp" alt="周期性水位推进让窗口闸门延迟开启并接收迟到数据" width="960" height="640" loading="lazy" /><figcaption>周期性水位推进让窗口闸门延迟开启并接收迟到数据<span>AI 生成配图</span></figcaption></figure>

这不是需要“修”的问题，而是要在设计作业时把它算进去：

- **估算结果延迟**：数据到达后不会立刻发 Watermark。`onEvent` 只记录最大值（`BoundedOutOfOrdernessWatermarks.java:64`），Watermark 周期性发出，默认每 200 毫秒一次（`PipelineOptions.java:91-94`）。所以窗口出结果的延迟，大约是乱序容忍再加上 Watermark 的发出周期。
- **乱序容忍不要随手设得很大**：它直接决定了结果要等多久。
- **给边界数据留后路**：恰好落在窗口最后 1 毫秒的数据也可能迟到，配置侧输出，或按需设置 `allowedLateness`（见踩坑实验室 #02）。

适用范围要说清楚：本文只验证了 `forBoundedOutOfOrderness` 和 `forMonotonousTimestamps`。自定义的 `WatermarkGenerator` 怎么算由实现决定；SQL 里 `WATERMARK FOR ... AS ...` 的计算方式本文没有验证，不要直接套用。

## 记住这几点

- `forBoundedOutOfOrderness(d)` 的 Watermark = 目前见过的最大事件时间 − d − 1 毫秒，因为 Watermark `T` 表示“≤ T 的数据不会再来”。
- 窗口 `[start, end)` 左闭右开，在 Watermark ≥ end − 1 时触发；合起来要等到某条数据的事件时间 ≥ 窗口结束 + 乱序容忍。
- Watermark 默认每 200 毫秒发一次，结果延迟约等于乱序容忍加发出周期，乱序容忍越大，结果出得越晚。
:::
