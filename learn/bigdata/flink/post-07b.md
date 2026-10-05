---
title: "Flink 2.x 源码精读（七·下）：窗口，从分配到触发"
description: "用一天的滚动窗口按天统计订单，10 月 1 日凌晨的两笔订单却被算进了\"9 月 30 日 8 点到 10 月 1 日 8 点\"的窗口。顺着这个现象，读懂 Flink 2.3 窗口的分配、两个 Timer、触发与清理、迟到数据的四种结局，以及会话窗口的合并。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-07b-cover.webp"}]]
---

# Flink 2.x 源码精读（七·下）：窗口，从分配到触发

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-07b-cover.webp" alt="Yui和Kai在时间窗口中追踪订单、触发器与迟到数据" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在时间窗口中追踪订单、触发器与迟到数据<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`ST` = `flink-runtime/src/main/java/org/apache/flink/streaming`
> 前置阅读：第七讲上篇（Watermark 与 Timer）；踩坑实验室 #02（窗口输出两次）、#03（差 1 毫秒）

---

先看一个实验。

按天统计订单。6 笔订单，时间都是北京时间：

```text
10-01 00:30, 10-01 07:59, 10-01 08:00, 10-01 12:00, 10-01 23:30, 10-02 07:30
```

窗口是一天的滚动窗口：

```java
.window(TumblingEventTimeWindows.of(Duration.ofDays(1)))
```

本机实测，每个窗口触发时，用北京时间打印窗口的起止和里面的订单：

```text
>>> window=[09-30 08:00, 10-01 08:00) 北京时间, start(UTC 毫秒)=1790726400000, count=2, orders=[10-01 00:30, 10-01 07:59]
>>> window=[10-01 08:00, 10-02 08:00) 北京时间, start(UTC 毫秒)=1790812800000, count=4, orders=[10-01 08:00, 10-01 12:00, 10-01 23:30, 10-02 07:30]
```

**10 月 1 日凌晨的两笔订单，被算进了"9 月 30 日 8 点到 10 月 1 日 8 点"这个窗口；10 月 2 日早上 7 点半的订单，却算进了"10 月 1 日"。** 每个窗口都是从北京时间早上 8 点开始的。

改一行：

```java
.window(TumblingEventTimeWindows.of(Duration.ofDays(1), Duration.ofHours(-8)))
```

```text
>>> window=[10-01 00:00, 10-02 00:00) 北京时间, start(UTC 毫秒)=1790784000000, count=5, orders=[10-01 00:30, 10-01 07:59, 10-01 08:00, 10-01 12:00, 10-01 23:30]
>>> window=[10-02 00:00, 10-03 00:00) 北京时间, start(UTC 毫秒)=1790870400000, count=1, orders=[10-02 07:30]
```

这才是我们想要的"每天"。

这一篇，我们来看窗口的完整生命周期：

1. 一条数据是怎么被分到窗口里的？为什么默认从北京时间 8 点开始？
2. 窗口什么时候触发、什么时候清理？每个窗口有几个 Timer？
3. 迟到的数据有哪几种结局？
4. 会话窗口是怎么合并的？

---

## 一、窗口算子的组成

```text
input.keyBy(...)
     .window(WindowAssigner)          分配：一条数据属于哪些窗口
     [.trigger(Trigger)]              触发：什么时候计算（默认由 Assigner 给出）
     [.evictor(Evictor)]              计算前后剔除元素
     [.allowedLateness(Duration)]     窗口结束后还保留多久
     [.sideOutputLateData(tag)]       真正迟到的数据去哪
     .reduce / aggregate / process
```

这些参数最终交给 `WindowOperatorBuilder`（`ST/runtime/operators/windowing/WindowOperatorBuilder.java`），由它决定窗口状态的类型（状态名固定是 `"window-contents"`，`:81`）：

| API | 窗口状态 | 每个窗口存什么 | 位置 |
|---|---|---|---|
| `reduce(f)` | `ReducingState` | **一个**聚合值 | `:163` 起，`:179` |
| `aggregate(f)` | `AggregatingState` | **一个**累加器 | `:267` 起，`:286` |
| `process(f)` | `ListState` | **所有**元素 | `:388` 起，`:399` |
| 任意 API + `evictor` | `ListState<StreamRecord>` | 所有元素，外加时间戳 | `:448-457` |

大窗口、高吞吐的场景下，这张表决定了状态有多大。只需要聚合结果时，用 `aggregate(aggFn, processWindowFn)`：增量聚合，同时还能在 `processWindowFn` 里拿到窗口的起止时间。开头的实验用的是 `process`，因为要把每笔订单都打印出来。

---

## 二、分配：窗口对齐到 1970 年

<figure class="ai-figure"><img src="/bigdata-img/flink/post-07b-1.webp" alt="订单光点按时间对齐进入不同窗口，展示窗口分配机制" width="960" height="640" loading="lazy" /><figcaption>订单光点按时间对齐进入不同窗口，展示窗口分配机制<span>AI 生成配图</span></figcaption></figure>

### 2.1 滚动窗口

`TumblingEventTimeWindows.assignWindows()`（`ST/api/windowing/assigners/TumblingEventTimeWindows.java:69`）：

```java
if (timestamp > Long.MIN_VALUE) {
    ...
    long start = TimeWindow.getWindowStartWithOffset(
            timestamp, (globalOffset + staggerOffset) % size, size);
    return Collections.singletonList(new TimeWindow(start, start + size));
} else {
    throw new RuntimeException(
            "Record has Long.MIN_VALUE timestamp (= no timestamp marker). "
                    + "Did you forget to call 'DataStream.assignTimestampsAndWatermarks(...)'?");   // :83
}
```

`getWindowStartWithOffset`（`ST/api/windowing/windows/TimeWindow.java:264-272`）：

```java
final long remainder = (timestamp - offset) % windowSize;
// handle both positive and negative cases
if (remainder < 0) {
    return timestamp - (remainder + windowSize);
} else {
    return timestamp - remainder;
}
```

**窗口的起点 = 时间戳减去"时间戳除以窗口大小的余数"。** 时间戳是从 1970-01-01 00:00:00 **UTC** 开始的毫秒数，所以窗口是对齐到 UTC 的整天、整小时上的，和作业什么时候启动、第一条数据是什么时间都没有关系。

开头实验里的 `start(UTC 毫秒)=1790726400000`，除以一天的毫秒数 86400000，正好是整数 20726。UTC 的零点，就是北京时间的早上 8 点。

### 2.2 offset

加上 `offset` 之后，余数按 `timestamp - offset` 算，窗口整体平移。北京时间比 UTC 早 8 小时，所以 offset 是 **-8 小时**。

这一点 Flink 的 Javadoc 里写得很清楚（`TumblingEventTimeWindows.java:117-121`）：

> （大意）以使用 UTC+08:00 的中国为例：想要一个一天大小、从本地时间 00:00:00 开始的窗口，可以用 `of(Duration.ofDays(1), Duration.ofHours(-8))`。

这个坑之所以常见，是因为按分钟、按小时的窗口不受影响：北京时间和 UTC 相差整 8 小时，整点对整点。**只有 8 小时不是窗口大小的整数倍时才会错位**：1、2、4、8 小时的窗口没问题；3、6、12 小时和一天的窗口，边界都会和北京时间的整点、零点对不上。

> offset 是一个固定值。北京时间没有夏令时，所以 -8 小时全年都对；如果你的时区有夏令时，一个固定的 offset 在一年里会有一段时间差 1 小时。这一点是按代码推断的，本篇没有做实验。

另外，构造函数要求 `abs(offset) < size`（`TumblingEventTimeWindows.java:58-61`）。

### 2.3 滑动窗口：一条数据进多个窗口

`SlidingEventTimeWindows.assignWindows()`（`ST/api/windowing/assigners/SlidingEventTimeWindows.java:77-86`）：

```java
List<TimeWindow> windows = new ArrayList<>((int) (size / slide));
long lastStart = TimeWindow.getWindowStartWithOffset(timestamp, offset, slide);
for (long start = lastStart; start > timestamp - size; start -= slide) {
    windows.add(new TimeWindow(start, start + size));
}
```

**一条数据会被分进 `size / slide` 个窗口。** 比如"最近 1 小时，每 1 分钟更新一次"，一条数据会进 60 个窗口。

再结合第一节的表：窗口算子对每个窗口分别写一次状态（下一节 `processElement` 的循环）。如果用的是 `process`，**每条数据会在状态里存 60 份**。按代码推断，大小和步长差别很大的滑动窗口，状态和 CPU 开销都会成倍增加，这时用 `aggregate` 会好很多。

### 2.4 WindowStagger：错开触发时间

`TumblingEventTimeWindows.of(size, offset, windowStagger)` 还有第三个参数（`ST/api/windowing/assigners/WindowStagger.java`）：

| 值 | 含义 |
|---|---|
| `ALIGNED`（默认） | 所有分区的窗口同一时刻触发（`:29-34`） |
| `RANDOM` | 每个算子实例收到第一条数据时，在 `[0, 窗口大小)` 里随机取一个偏移（`:36-46`） |
| `NATURAL` | 以收到第一条数据时的处理时间，相对于窗口起点的差作为偏移（`:48-59`） |

默认的 `ALIGNED` 下，所有 key 的窗口都在同一个 Watermark 上触发，可能造成瞬时的输出高峰。`RANDOM`、`NATURAL` 可以把触发时间错开，但代价是**窗口的边界也跟着偏移了**，不同算子实例的窗口边界还不一样。所以按"自然日"、"整点"统计的场景，不能用它们。

---

## 三、触发与清理：每个窗口两个 Timer

<figure class="ai-figure"><img src="/bigdata-img/flink/post-07b-2.webp" alt="两个计时环分别触发窗口计算与清理，展示迟到数据再次触发" width="960" height="640" loading="lazy" /><figcaption>两个计时环分别触发窗口计算与清理，展示迟到数据再次触发<span>AI 生成配图</span></figcaption></figure>

### 3.1 processElement

非合并窗口的处理在 `ST/runtime/operators/windowing/WindowOperator.java:405-446`：

```java
for (W window : elementWindows) {                                   // :405
    if (isWindowLate(window)) {                                     // :408 窗口已经被清理：跳过
        continue;
    }
    isSkippedElement = false;
    windowState.setCurrentNamespace(window);                        // namespace = 窗口
    windowState.add(element.getValue());                            // 进状态（或增量聚合）
    ...
    TriggerResult triggerResult = triggerContext.onElement(element);
    if (triggerResult.isFire()) { ... emitWindowContents(window, contents); }
    if (triggerResult.isPurge()) { windowState.clear(); }
    registerCleanupTimer(window);                                   // 注册清理 Timer
}
if (isSkippedElement && isElementLate(element)) {                   // :440
    if (lateDataOutputTag != null) {
        sideOutput(element);                                        // :442 侧输出
    } else {
        this.numLateRecordsDropped.inc();                           // :444 否则丢弃，只加一个指标
    }
}
```

### 3.2 EventTimeTrigger

默认的触发器 `EventTimeTrigger`（`ST/api/windowing/triggers/EventTimeTrigger.java`）只有几行：

```java
public TriggerResult onElement(Object element, long timestamp, TimeWindow window, TriggerContext ctx) {   // :37
    if (window.maxTimestamp() <= ctx.getCurrentWatermark()) {
        // if the watermark is already past the window fire immediately
        return TriggerResult.FIRE;
    } else {
        ctx.registerEventTimeTimer(window.maxTimestamp());
        return TriggerResult.CONTINUE;
    }
}

public TriggerResult onEventTime(long time, TimeWindow window, TriggerContext ctx) {                     // :50
    return time == window.maxTimestamp() ? TriggerResult.FIRE : TriggerResult.CONTINUE;
}
```

两个要点：

1. **Timer 注册在 `window.maxTimestamp()` 上**，也就是 `end - 1`（`TimeWindow.java:84-86`）。窗口是左闭右开的 `[start, end)`，所以 Watermark 到 `end - 1` 时窗口就触发了。这和上篇说的 Watermark 要减 1 正好配合（踩坑实验室 #03）。
2. **返回的是 `FIRE`，不是 `FIRE_AND_PURGE`**：窗口触发之后，状态还在，一直保留到清理 Timer 触发。这是 `allowedLateness` 能工作的前提。

### 3.3 两个 Timer

| Timer | 时间 | 谁注册的 | 作用 |
|---|---|---|---|
| 触发 Timer | `window.maxTimestamp()` | `EventTimeTrigger.onElement` | 输出结果 |
| 清理 Timer | `window.maxTimestamp() + allowedLateness` | `registerCleanupTimer`（`WindowOperator.java:631`，时间由 `cleanupTime()` 计算，`:670-677`） | 清空窗口状态和触发器状态 |

`allowedLateness = 0` 时，两个 Timer 的时间相同，上篇讲过 Timer 按 (时间, key, 窗口) 去重，所以只剩一个。

`onEventTime`（`WindowOperator.java:450`）里，先问触发器要不要输出，再检查是不是清理时间：

```java
if (windowAssigner.isEventTime() && isCleanupTime(triggerContext.window, timer.getTimestamp())) {
    clearAllState(triggerContext.window, windowState, mergingWindows);       // :487
}
```



---

## 四、迟到数据的四种结局

### 4.1 两个判断条件

```java
protected boolean isWindowLate(W window) {                                   // :609
    return (windowAssigner.isEventTime()
            && (cleanupTime(window) <= internalTimerService.currentWatermark()));
}

protected boolean isElementLate(StreamRecord<IN> element) {                  // :620
    return (windowAssigner.isEventTime())
            && (element.getTimestamp() + allowedLateness
                    <= internalTimerService.currentWatermark());
}
```

把一条事件时间为 `t`、所属窗口为 `[s, e)` 的数据，按它到达时的 Watermark `W` 分成四种情况（`L` 表示 `allowedLateness`）：

| W 的范围 | 结局 | 源码路径 |
|---|---|---|
| `W < e-1` | **正常**：进状态，等待触发 | `onElement` → 注册 Timer |
| `e-1 ≤ W < e-1+L` | **迟到但被接受**：进状态，**立即再触发一次**，输出更新后的完整结果 | `onElement` 返回 `FIRE` |
| `W ≥ e-1+L`，配置了侧输出 | **进侧输出** | `:440-442` |
| `W ≥ e-1+L`，没配置侧输出 | **丢弃**，`numLateRecordsDropped` 加 1 | `:444` |



### 4.2 实测

`WindowDemo late`：并行度 1，`forBoundedOutOfOrderness(2s)`，5 秒滚动窗口，迟到数据走侧输出。Source 每 500ms 发一条，事件时间（秒）依次是 `1, 2, 4, 6, 3, 8, 2, 11, 1, 13`。

> 每条之间间隔 500ms，大于 Watermark 的周期 200ms，保证每两条数据之间至少发出一次 Watermark，结果才是确定的。

**`allowedLateness = 3s`**（本机实测，完整日志在 `assets/logs/window-late.log`）：

```text
FIRE window=[0s,5s) count=4 elements=[1s, 2s, 4s, 3s] watermark=5.999s wall=+4.4s
FIRE window=[0s,5s) count=5 elements=[1s, 2s, 4s, 3s, 2s] watermark=5.999s wall=+4.7s
LATE (side output) event=1s
FIRE window=[5s,10s) count=2 elements=[6s, 8s] watermark=10.999s wall=+6.3s
FIRE window=[10s,15s) count=2 elements=[11s, 13s] watermark=MAX wall=+7.2s
```

**`allowedLateness = 0`**（`assets/logs/window-late-0.log`）：

```text
FIRE window=[0s,5s) count=4 elements=[1s, 2s, 4s, 3s] watermark=5.999s wall=+4.3s
LATE (side output) event=2s
LATE (side output) event=1s
FIRE window=[5s,10s) count=2 elements=[6s, 8s] watermark=10.999s wall=+6.1s
FIRE window=[10s,15s) count=2 elements=[11s, 13s] watermark=MAX wall=+7.1s
```

逐条对照：

| 数据 | 到达时 W | `L = 3s` | `L = 0` |
|---|---|---|---|
| 1, 2, 4 | MIN → 0.999 → 1.999 | 正常进入 `[0,5)` | 同左 |
| 6 | 1.999 | 进入 `[5,10)`；之后 W 变为 3.999 | 同左 |
| **3** | 3.999 | 乱序但不迟到（3.999 < 4.999），正常进入 `[0,5)` | 同左 |
| 8 | 3.999 | 之后 W 变为 5.999，越过 4.999 → `[0,5)` **第一次触发**，4 条 | 同左；同时清理 Timer（4.999）也到了，`[0,5)` 的状态被清空 |
| **2** | 5.999 | 迟到，但没超过清理时间 7.999 → 进状态并**立即再触发一次**，5 条 | 窗口已被清理 → **进侧输出** |
| 11 | 5.999 | 之后 W 变为 8.999，越过 7.999 → 清理 `[0,5)` | — |
| **1** | 8.999 | 窗口已被清理，`1 + 3 = 4 ≤ 8.999` → **进侧输出** | 进侧输出 |
| 13 | 8.999 | 之后 W 变为 10.999 → `[5,10)` 触发 | 同左 |
| 输入结束 | — | 有界 Source 发出 `MAX_WATERMARK` → `[10,15)` 触发 | 同左 |

`L = 3s` 时，`[0s,5s)` 输出了**两次**，第二次是包含迟到数据的完整新结果，不是增量。下游如果是追加写入，就会出现重复。这就是踩坑实验室 #02 讲的问题。

### 4.3 一个细节：滑动窗口里"部分迟到"的数据

注意 `:440` 的条件是 `isSkippedElement && isElementLate(element)`。`isSkippedElement` 只有在**所有**窗口都被跳过时才为 true。

对滑动窗口来说，一条数据属于多个窗口。如果其中一部分窗口已经被清理、另一部分还在，`isSkippedElement` 就是 false：这条数据会进入那些还在的窗口，**但不会进侧输出**，被清理掉的那些窗口里就少了它，而且没有任何记录。

> 这一点是按代码推断的，本篇没有做滑动窗口的实验。可以用课后练习 2 自己验证。

---

## 五、会话窗口：合并

### 5.1 合并的规则

会话窗口（`EventTimeSessionWindows`）是一种"可合并"的窗口：每条数据先得到一个 `[t, t+gap)` 的窗口，再和已有的相交的窗口合并。相交的判断是（`TimeWindow.java:116-117`）：

```java
return this.start <= other.end && this.end >= other.start;
```

**端点相接也算相交。**

### 5.2 合并时状态怎么办

每次合并都把几个窗口的状态搬到一个新窗口下，代价很高。`MergingWindowSet`（`ST/runtime/operators/windowing/MergingWindowSet.java`）的做法是维护一个映射：**窗口 → 存放状态的窗口**（`mapping`，`:63`）。

```java
// pick any of the merged windows and choose that window's state window
// as the state window for the merge result
W mergedStateWindow = this.mapping.get(mergedWindows.iterator().next());     // :190
...
this.mapping.put(mergeResult, mergedStateWindow);
```

合并后的窗口**继续用其中一个原窗口的 namespace 存状态**，只把其他原窗口的状态并过来（`WindowOperator.java:363`，`windowMergingState.mergeNamespaces`）。这个映射本身存在算子的 `ListState` 里，只有发生变化时才写回（`MergingWindowSet.java:99-100`，`persist()`）。

### 5.3 实测

`WindowDemo session`：3 秒 gap，事件时间依次是 `1, 2, 10, 7, 20`（本机实测，`assets/logs/window-session.log`）：

```text
FIRE window=[1s,5s) count=2 elements=[1s, 2s] watermark=7.999s wall=+2.9s
FIRE window=[7s,13s) count=2 elements=[10s, 7s] watermark=17.999s wall=+3.9s
FIRE window=[20s,23s) count=1 elements=[20s] watermark=MAX wall=+4.8s
```

- `1` 和 `2` 的窗口 `[1,4)`、`[2,5)` 相交，合并成 `[1,5)`。
- `10` 先得到 `[10,13)`；后到的 `7` 得到 `[7,10)`，端点 10 相接，也算相交，于是**向前**合并成 `[7,13)`。

注意 `elements=[10s, 7s]` 的顺序：`7` 是被**追加**到 `[10,13)` 原有的状态里的。对照代码：新窗口在合并时会先从待合并的集合里去掉（`MergingWindowSet.java:183`），因为它还没有任何状态；存放状态的窗口只从已有的窗口里选（`:190`）。所以合并后的 `[7,13)`，状态仍然存在 namespace `[10,13)` 下。

### 5.4 两个限制

1. **合并后的窗口不能已经"过期"**：合并回调里有一个检查（`WindowOperator.java:322-333`），如果合并结果的 `maxTimestamp + allowedLateness` 已经不大于当前 Watermark，直接抛 `UnsupportedOperationException`。自定义 `MergingWindowAssigner` 时，如果合并逻辑会让窗口变短，就会遇到这个异常。
2. **触发器要支持合并**：`EventTimeTrigger.canMerge()` 返回 true（`:66`）；`onMerge` 为合并后的窗口注册新 Timer，并且**只在 Watermark 还没越过时才注册**（`:71-79`），注释里的原因是：否则 `onElement` 已经立即触发了一次，这里再注册就会触发两次。

---

## 六、自己动手

示例代码在 `flink-notes/demos`：

```bash
./run.sh study.time.DailyWindowDemo nooffset
```

```bash
./run.sh study.time.DailyWindowDemo offset
```

```bash
./run.sh study.time.WindowDemo late
```

```bash
./run.sh study.time.WindowDemo late 0
```

```bash
./run.sh study.time.WindowDemo session
```

**推荐的断点**：

| 断点 | 看什么 |
|---|---|
| `TimeWindow.java:265` | 余数的计算，对比有没有 offset |
| `WindowOperator.java:405` | 一条数据被分到哪些窗口 |
| `EventTimeTrigger.java:40` | 迟到数据引起的立即触发 |
| `WindowOperator.java:440` | 迟到数据进侧输出还是被丢弃 |
| `WindowOperator.java:487` | 窗口状态被清理 |
| `MergingWindowSet.java:190` | 会话窗口合并时选哪个窗口存状态 |

窗口算子的测试在 `flink-streaming-java/src/test/java/org/apache/flink/streaming/runtime/operators/windowing/WindowOperatorTest.java`，用 `KeyedOneInputStreamOperatorTestHarness` 手动推送数据和 Watermark，不需要启动集群。以后给窗口相关的代码提 PR，测试基本都是照这个模式写的。

---

## 七、课后练习

1. **推导题**：开头的实验里，如果窗口改成 `of(Duration.ofHours(12))`，不加 offset，6 笔订单会分到哪几个窗口？先算，再运行验证。
2. **滑动窗口**：改用 `SlidingEventTimeWindows.of(Duration.ofSeconds(10), Duration.ofSeconds(5))`，跑 `WindowDemo late`，一条数据进了几个窗口？4.3 节说的"部分迟到"能复现吗？
3. **状态大小**：同一个滑动窗口，分别用 `process` 和 `aggregate`，开启 Checkpoint 后比较 Checkpoint 的大小。
4. **思考**：为什么 Flink 不直接用作业所在机器的时区来对齐窗口？

---

## 写在最后

这一讲的核心可以用一句话概括：**窗口按 UTC 对齐分配，触发 Timer 在 `end - 1`，清理 Timer 在 `end - 1 + allowedLateness`，两个 Timer 之间到达的数据会让窗口再触发一次。**

回到开头：一天的窗口，起点是 UTC 的零点，也就是北京时间早上 8 点；加上 `Duration.ofHours(-8)` 的 offset，才是北京时间的"每天"。

**实际使用时的建议**：

- 窗口大小不能整除 8 小时的（3、6、12 小时、一天等），一定要检查 offset；
- 只需要聚合结果时用 `aggregate`，不要用 `process`；大小和步长差别很大的滑动窗口尤其如此；
- `allowedLateness > 0` 会让同一个窗口输出多次，下游要能处理更新（比如 upsert）；
- 迟到数据最好配一个侧输出，否则它们只是指标里的一个数字。

到这里，第七讲（Watermark 与窗口）就结束了。下一讲我们看 **Source 和 Sink**：上篇里那个"不到 150 行"的 Source，在真实的连接器里是怎么设计的？Sink 又是怎么用两阶段提交做到端到端的精确一次的？


**留一个问题**：你遇到过"按天统计，数字总是对不上"的情况吗？最后发现是什么原因？

下一讲：**Flink 2.x 源码精读（八）：Source / Sink 新架构**
:::
