---
title: "踩坑实验室 #02：同一个窗口，为什么输出了两次？"
description: "窗口触发后状态还在，allowedLateness 内的迟到数据会让窗口带着完整新结果再输出一次。"
bigdata: "flink"
---

# 踩坑实验室 #02：同一个窗口，为什么输出了两次？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-02）。对应课程：第七讲《时间、Watermark 与窗口》。配套示例：`flink-notes/demos/src/main/java/study/time/WindowDemo.java` 的 `late` 模式（`late 0` 表示 `allowedLateness = 0`）

## 现象

5 秒的事件时间滚动窗口，设置了 `allowedLateness`。结果 `[0s,5s)` 这个窗口输出了两条结果：第一次 4 条，紧接着又来一次，变成 5 条。如果下游是追加写入，同一个窗口就会出现两行。

先说结论：这不是 bug，而是 `allowedLateness` 的设计行为。

## 复现

`WindowDemo late`：并行度 1，乱序容忍 2 秒，5 秒滚动窗口，事件时间依次为 1、2、4、6、3、8、2、11、1、13 秒，每 500 毫秒发一条，迟到数据走侧输出。

`allowedLateness = 3s`：

```
FIRE window=[0s,5s) count=4 elements=[1s, 2s, 4s, 3s] watermark=5.999s
FIRE window=[0s,5s) count=5 elements=[1s, 2s, 4s, 3s, 2s] watermark=5.999s
LATE (side output) event=1s
FIRE window=[5s,10s) count=2 elements=[6s, 8s] watermark=10.999s
FIRE window=[10s,15s) count=2 elements=[11s, 13s] watermark=MAX
```

`allowedLateness = 0`：

```
FIRE window=[0s,5s) count=4 elements=[1s, 2s, 4s, 3s] watermark=5.999s
LATE (side output) event=2s
LATE (side output) event=1s
FIRE window=[5s,10s) count=2 elements=[6s, 8s] watermark=10.999s
FIRE window=[10s,15s) count=2 elements=[11s, 13s] watermark=MAX
```

| 配置 | `0s,5s)` 输出次数 | 迟到数据 |
|---|---|---|
| `allowedLateness = 3s` | **2 次**（4 条 → 5 条） | 2s 的那条被接受；1s 的那条进侧输出 |
| `allowedLateness = 0` | 1 次 | 2s、1s 两条都进侧输出 |

两种配置各跑过两次，输出完全一致（行尾的墙上时间已省略）。输入顺序是固定的，所以结果是确定的。`watermark=5.999s` 这样的数字从何而来，见踩坑实验室 #03。

## 原因

事件时间滚动窗口的默认触发器是 `EventTimeTrigger`（`TumblingEventTimeWindows.java:89-90`）。Watermark 到达窗口的 `maxTimestamp`（结束时间 − 1 毫秒）时，它返回 `FIRE`（`EventTimeTrigger.java:50-51`）。

关键在于返回的是 `FIRE`，不是 `FIRE_AND_PURGE`：**窗口触发之后，状态还留着**。状态要等 Watermark 到达 `maxTimestamp + allowedLateness` 才会清理（清理时间的计算在 `WindowOperator.java:670-673`，到点清空在 `:484-487`）。

在“已触发、未清理”这段时间里来了一条迟到数据：它会被加进窗口，而且因为 Watermark 已经越过窗口，`onElement` 直接返回 `FIRE`（`EventTimeTrigger.java:40-42`），窗口**立即再触发一次**。再次输出的是**完整的新结果**，不是增量：实测第二次输出的 `elements` 包含全部 5 条。

超过 `allowedLateness` 的数据：配置了侧输出就进侧输出，没配置就被静默丢弃，只让指标 `numLateRecordsDropped` 加 1（`WindowOperator.java:440-445`）。`allowedLateness` 默认是 0（`WindowOperatorBuilder.java:99`）。

把这几条合起来，一条迟到数据有四种结局：

| 迟到数据到达时，Watermark 在哪 | 结局 |
|---|---|
| 还没到窗口结束 | 正常进入窗口，等待触发 |
| 过了窗口结束，没到清理时间 | 进入窗口，**立即重新输出完整结果** |
| 过了清理时间，配置了侧输出 | 进入侧输出 |
| 过了清理时间，没配侧输出 | **静默丢弃**，只计入 `numLateRecordsDropped` |

![触发不等于清理：两条竖线之间，窗口已经输出过，但还能改结果

## 怎么解决

用了 `allowedLateness`，下游就要能处理“同一个窗口的多次结果”。会不会造成重复，取决于下游怎么写：

1. **下游 upsert**：用“窗口开始时间 + key”做主键，新结果覆盖旧结果。适合写 MySQL、HBase、Redis 这类支持更新的存储。
2. **`allowedLateness = 0` + 侧输出**：窗口只输出一次，迟到数据用侧输出单独收集，另行修正或告警。适合下游只能追加的场景。

不管哪种做法，都建议配置侧输出，否则超过允许时间的迟到数据被丢掉了，日志里也不会有任何记录。

![allowedLateness 改成 0：窗口只输出一次，两条迟到数据都进了侧输出](/bigdata-img/src/content-plan/assets/png/ep02/xhs-P6.webp)

## 记住这几点

- 窗口触发不等于窗口关闭：状态要到 `maxTimestamp + allowedLateness` 才清理。
- 这段时间内的迟到数据会让窗口**再输出一次完整结果**，下游要按“窗口 + key”做 upsert，或者把 `allowedLateness` 设成 0。
- 超过允许时间、又没配侧输出的迟到数据会被静默丢弃，只留下 `numLateRecordsDropped` 指标。
:::
