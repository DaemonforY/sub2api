---
title: "Flink 源码导读 08：时间、Watermark 与窗口"
description: "事件时间与 Watermark 的生成和传播、窗口的分配、触发与清理。"
bigdata: "flink"
---

# Flink 源码导读 08：时间、Watermark 与窗口

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`CORE` = `flink-core/src/main/java/org/apache/flink`，`RT` = `flink-runtime/src/main/java/org/apache/flink`（注意：本篇的 `RT` 比前几篇少一层 `/runtime`，因为 `streaming` 包也在这个模块下）
> 前置阅读：[01](/bigdata/flink/01-execution) 第 5 节（Mailbox 模型）、[03](/bigdata/flink/03-state-backend)（KeyGroup）、[04](/bigdata/flink/04-network)（RecordWriter 与 InputGate）
> 配套示例：`demos/src/main/java/study/time/WindowDemo.java`，第 7 节的所有输出都是在本机实测的

## 0. 本篇要回答的问题

1. Watermark 是**谁**、在**什么时候**生成的？`forBoundedOutOfOrderness(2s)` 算出来的值为什么总是差 1 毫秒？
2. 一个算子有多个上游时，Watermark 怎么合并？为什么**一个空闲分区就能让整个作业的窗口停止触发**？
3. 窗口是怎么"触发"的？Watermark 和 Timer 之间是什么关系？
4. 迟到数据有几种结局？`allowedLateness` 会带来什么副作用？
5. 会话窗口是怎么合并的？合并之后状态放在哪里？
6. Watermark 对齐（Watermark Alignment）解决了什么问题？

先给出全篇的主线，后面每一节展开其中一段：

```
 Source 线程（Mailbox）                         网络                   下游 Task（Mailbox）
 ┌──────────────────────────────┐                                 ┌──────────────────────────────────────┐
 │ SourceReader 产生记录          │                                 │ AbstractStreamTaskNetworkInput        │
 │  └ SourceOutputWithWatermarks │                                 │  └ StatusWatermarkValve  多通道取最小值 │
 │     ├ TimestampAssigner       │  记录 + Watermark 同一条通道      │     └ operator.processWatermark()     │
 │     └ WatermarkGenerator      │ ───────────────────────────────▶ │        └ InternalTimerService         │
 │        onEvent / onPeriodic   │  （Watermark 也是 StreamElement，  │           .advanceWatermark()         │
 │  每 200ms 定时器 → 发 Watermark │    会被背压挡住，不会"插队"）        │           └ WindowOperator.onEventTime │
 └──────────────────────────────┘                                 │              └ Trigger → FIRE / PURGE │
                                                                  └──────────────────────────────────────┘
```

---

## 1. Watermark 的生成

### 1.1 两个接口，一个策略

用户只接触 `WatermarkStrategy`（`CORE/api/common/eventtime/WatermarkStrategy.java`），它是一个工厂，负责创建两样东西：

| 组件 | 职责 | 何时被调用 |
|---|---|---|
| `TimestampAssigner` | 从记录中提取事件时间 | 每条记录 |
| `WatermarkGenerator` | 决定 Watermark 的值 | `onEvent()`：每条记录（`WatermarkGenerator.java:38`）；`onPeriodicEmit()`：周期性调用（`:46`） |

内置的策略：

| 工厂方法 | 行号 | 生成器 |
|---|---|---|
| `forBoundedOutOfOrderness(d)` | `WatermarkStrategy.java:234` | `BoundedOutOfOrdernessWatermarks` |
| `forMonotonousTimestamps()` | `:219` | 乱序容忍为 0 的同一个类 |
| `noWatermarks()` | `:247` | 什么都不发 |
| `.withIdleness(d)` | `:147` | 用 `WatermarksWithIdleness` 包一层 |
| `.withWatermarkAlignment(group, drift)` | `:168` | 只是附加参数，逻辑在 SourceOperator 和 SourceCoordinator 中（第 6 节） |

### 1.2 `BoundedOutOfOrdernessWatermarks`：为什么要减 1

`CORE/api/common/eventtime/BoundedOutOfOrdernessWatermarks.java`

```java
this.maxTimestamp = Long.MIN_VALUE + outOfOrdernessMillis + 1;          // :57 保证初始 Watermark 恰好是 Long.MIN_VALUE

public void onEvent(T event, long eventTimestamp, WatermarkOutput output) {
    maxTimestamp = Math.max(maxTimestamp, eventTimestamp);              // :64 只记录最大值，不发
}

public void onPeriodicEmit(WatermarkOutput output) {
    output.emitWatermark(new Watermark(maxTimestamp - outOfOrdernessMillis - 1));   // :69
}
```

Watermark `W` 的语义是"**时间戳 ≤ W 的数据不会再来了**"。如果最大时间戳是 6000、容忍 2000，那么 4000 这个时间戳还**可能**到来，所以 Watermark 只能是 3999。这就是示例输出中 `watermark=5.999s` 这种数字的来源。

另外两点值得注意：
- **`onEvent` 不发 Watermark**，只有周期调用才发。而且只有**增长**的 Watermark 才会真正发出：Source 路径由 `WatermarkToDataOutput.emitWatermark()` 过滤（`RT/streaming/api/operators/source/WatermarkToDataOutput.java:77-79`），`assignTimestampsAndWatermarks` 路径由 `TimestampsAndWatermarksOperator.WatermarkEmitter` 过滤。周期由 `pipeline.auto-watermark-interval` 控制，默认 **200ms**（`CORE/configuration/PipelineOptions.java:91-94`）。
- 生成器的状态（`maxTimestamp`）**不进 Checkpoint**。作业恢复后，Watermark 会从 `Long.MIN_VALUE` 重新开始，直到新数据把它推起来。

### 1.3 生成器在哪里运行：两条路径

**路径 A：FLIP-27 Source 里（推荐）。** `fromSource(source, watermarkStrategy, name)` 把策略交给 `SourceOperator`：

```
SourceOperator.open()                                       RT/streaming/api/operators/SourceOperator.java:416
 ├ mainInputActivityClock = new PausableRelativeClock(...)  :418  背压时会暂停，见 1.4 节
 ├ emitProgressiveWatermarks ?                              :426
 │    createProgressiveEventTimeLogic(..., getAutoWatermarkInterval(), ...)   :429-433
 │  : createNoOpEventTimeLogic(...)                         :439  批模式走这里
 └ eventTimeLogic.startPeriodicWatermarkEmits()             :461
     └ ProgressiveTimestampsAndWatermarks                  RT/streaming/api/operators/source/ProgressiveTimestampsAndWatermarks.java
        timeService.scheduleWithFixedDelay(this::emitImmediateWatermark, interval, interval)   :162-166
```

每条记录经过 `SourceOutputWithWatermarks.collect()`（`RT/streaming/api/operators/source/SourceOutputWithWatermarks.java:105`）：先 `extractTimestamp`（`:107`），再 `watermarkGenerator.onEvent`（`:111`）。

Source 路径最重要的特性是 **Split 级别的 Watermark**：`SplitLocalOutputs.createOutputForSplit()`（`ProgressiveTimestampsAndWatermarks.java:265` 起）给每个 split 创建**独立的** `WatermarkGenerator`，注册到 `WatermarkOutputMultiplexer`。周期触发时（`:323-333`），先让每个 split 算出自己的候选值，再由 multiplexer 取**最小值**统一发出。这样即使一个 Kafka 分区的数据远远落后，Watermark 也不会被快的分区"带跑"。

**路径 B：`assignTimestampsAndWatermarks()`。** 在流中间插入一个 `TimestampsAndWatermarksOperator`（`RT/streaming/runtime/operators/TimestampsAndWatermarksOperator.java`）：

```java
public void open() {                                               // :86
    ...
    if (watermarkInterval > 0 && emitProgressiveWatermarks) {       // :116
        getProcessingTimeService().registerTimer(now + watermarkInterval, this);
    }
}
public void processElement(StreamRecord<T> element) {              // :133
    element.setTimestamp(timestampAssigner.extractTimestamp(...));
    output.collect(element);
    watermarkGenerator.onEvent(event, newTimestamp, wmOutput);
}
public void onProcessingTime(long timestamp) {                     // :145
    watermarkGenerator.onPeriodicEmit(wmOutput);
    registerTimer(now + watermarkInterval, this);                  // 自己给自己续约
}
public void processWatermark(Watermark mark) {                     // :157
    if (mark.getTimestamp() == Long.MAX_VALUE) { wmOutput.emitWatermark(MAX_WATERMARK); }
    // 其余上游 Watermark 全部丢弃！
}
public void processWatermarkStatus(WatermarkStatus s) {}           // :168 上游的空闲状态也丢弃
```

注意 `:157` 和 `:168`：**这个算子会吞掉上游的 Watermark 和空闲状态**，只转发表示输入结束的 `MAX_WATERMARK`。所以在已经有 Watermark 的流上再调一次 `assignTimestampsAndWatermarks`，就是把 Watermark 完全换掉，而不是叠加。另外它只能看到**一个 subtask 收到的所有记录**，没有 split 的概念，因此路径 A 更精确。

> **补充（2026-10-02）**：`demos/.../time/SplitWatermarkDemo.java` 实测了两条路径的差别：同一个 reader 交替读两个 split，其中一个落后 5 秒，`forBoundedOutOfOrderness(1s)`。路径 A 0 条迟到，路径 B 400 条中有 197 条被判为迟到。详见 `content-plan/19-第7讲上-公众号定稿.md` 第二节。

**批模式**：`SourceTransformationTranslator.translateForBatchInternal()` 传入 `false /* don't emit progressive watermarks */`（`RT/streaming/runtime/translators/SourceTransformationTranslator.java:45-51`）。批作业中不会产生中间 Watermark，只在输入结束时发一个 `MAX_WATERMARK`，一次性触发所有窗口。

### 1.4 空闲检测，以及"背压不算空闲"

`CORE/api/common/eventtime/WatermarksWithIdleness.java`

```java
public void onPeriodicEmit(WatermarkOutput output) {                  // :74
    if (idlenessTimer.checkIfIdle()) {
        if (!isIdleNow) { output.markIdle(); isIdleNow = true; }       // 只发一次 IDLE
    } else {
        watermarks.onPeriodicEmit(output);
    }
}

public boolean checkIfIdle() {                                        // :127
    if (counter != lastCounter) { ...; startOfInactivityNanos = 0L; return false; }   // 有新数据：重置
    else if (startOfInactivityNanos == 0L) {
        startOfInactivityNanos = clock.relativeTimeNanos(); return false;               // 第一次发现没数据：开始计时
    } else {
        return clock.relativeTimeNanos() - startOfInactivityNanos > maxIdleTimeNanos;
    }
}
```

两个细节：
1. 计时是从"**某次周期检查发现没有新数据**"才开始的，所以实际判定为空闲的时间是 `idleTimeout + 最多约 2 个 Watermark 周期`。
2. `clock` 是 `PausableRelativeClock`。`SourceOperator` 通过 `taskIOMetricGroup.registerBackPressureListener(...)`（`SourceOperator.java:421`）把它注册成背压监听器；每个 split 也有自己的一份（`ProgressiveTimestampsAndWatermarks.java:305-311`，注释写的是"will be paused both in case of back pressure and when split is paused due to watermark alignment"）。**Source 因为背压读不动数据时，空闲计时会暂停**，避免把"被下游堵住"误判成"没有数据"。

---

## 2. Watermark 的传播

### 2.1 在网络中：Watermark 也是一条"记录"

`RecordWriterOutput.emitWatermark()`（`RT/streaming/runtime/io/RecordWriterOutput.java:148`）把 Watermark 包进 `serializationDelegate`，调用 `recordWriter.broadcastEmit(...)`（`:159`），**广播**给所有下游通道。

它走的是和数据完全相同的序列化和 Buffer 路径（04 篇），不是优先级事件。由此可以推出：
- 同一通道内，Watermark **严格排在它之前的数据后面**，下游看到 Watermark 时，前面的数据一定已经处理完了。这是窗口结果正确的基础。
- **背压会延迟 Watermark**。非对齐 Checkpoint 的 Barrier 可以越过缓冲区中的数据，Watermark 不行。背压严重时窗口"迟迟不出结果"，原因往往在这里。

### 2.2 在输入端：`StatusWatermarkValve` 取最小值

```
AbstractStreamTaskNetworkInput.processElement()          RT/streaming/runtime/io/AbstractStreamTaskNetworkInput.java:215-223
 ├ isWatermark()        → statusWatermarkValve.inputWatermark(watermark, channelIndex, output)
 └ isWatermarkStatus()  → statusWatermarkValve.inputWatermarkStatus(...)
```

`RT/streaming/runtime/watermarkstatus/StatusWatermarkValve.java` 为每个输入通道（准确地说是每个 subpartition）维护一个 `SubpartitionStatus`：`{watermark, watermarkStatus, isWatermarkAligned}`。所有 **ACTIVE 且已对齐**的通道放在一个小顶堆 `alignedSubpartitionStatuses` 中（`:73`）。

```java
public void inputWatermark(Watermark watermark, int channelIndex, DataOutput<?> output) {    // :153
    if (lastOutputWatermarkStatus.isActive() && subpartitionStatus.watermarkStatus.isActive()) {
        if (watermarkMillis > subpartitionStatus.watermark) {          // 单个通道内倒退的 Watermark 直接忽略
            subpartitionStatus.watermark = watermarkMillis;
            if (isWatermarkAligned) adjustAlignedSubpartitionStatuses(...);   // 调整堆
            else if (watermarkMillis >= lastOutputWatermark) markWatermarkAligned(...);
            findAndOutputNewMinWatermarkAcrossAlignedSubpartitions(output);
        }
    }
}

private void findAndOutputNewMinWatermarkAcrossAlignedSubpartitions(DataOutput<?> output) {   // :273
    if (hasAlignedSubpartitions
            && alignedSubpartitionStatuses.peek().watermark > lastOutputWatermark) {         // :281 堆顶就是最小值
        lastOutputWatermark = alignedSubpartitionStatuses.peek().watermark;
        output.emitWatermark(new Watermark(lastOutputWatermark));
    }
}
```

"已对齐"（aligned）的意思是：这个通道的 Watermark 已经追上了当前输出值。一个空闲通道恢复 ACTIVE 后，如果它的 Watermark 比当前输出值小，**不会**被放回堆里（`:259`），否则它会把输出 Watermark 往回拉，而 Watermark 必须单调不减。

**空闲状态的处理**（`inputWatermarkStatus`，`:199`）：
- 一个通道变为 IDLE：从堆中移除（`markWatermarkUnaligned`）。如果它恰好就是当前的最小值，就重新计算最小值，Watermark 可能因此**前进**。
- **所有**通道都变为 IDLE：发出所有通道中的**最大** Watermark（`findAndOutputMaxWatermarkAcrossAllSubpartitions`，`:328-338`），再向下游发 IDLE。

这就回答了开头的第 2 个问题：**一个从来没有数据的通道，Watermark 永远是 `Long.MIN_VALUE`，一直待在堆顶，整个算子的 Watermark 就永远无法前进。** 第 7.3 节的示例把这种情况复现了出来。

### 2.3 在算子中：推进 Timer，然后向下游转发

`OneInputStreamTask` 中的 `StreamTaskNetworkOutput.emitWatermark()` 先更新 `currentInputWatermark` 指标，再调用 `operator.processWatermark(watermark)`（`RT/streaming/runtime/tasks/OneInputStreamTask.java:255-257`）。

```java
// RT/streaming/api/operators/AbstractStreamOperator.java
public void processWatermark(Watermark mark) {                       // :690
    if (watermarkProcessor != null) watermarkProcessor.emitWatermarkInsideMailbox(mark);
    else emitWatermarkDirectly(mark);
}
private void emitWatermarkDirectly(Watermark mark) {
    if (timeServiceManager != null) timeServiceManager.advanceWatermark(mark);   // :700 先触发 Timer
    output.emitWatermark(mark);                                                  // :702 再向下游转发
}
```

**先触发 Timer，再转发 Watermark**，顺序不能反：窗口在 `onEventTime` 中输出的结果，必须排在触发它的 Watermark **之前**到达下游，否则下游会把这些结果当成迟到数据。

双输入算子（如 join）用 `IndexedCombinedWatermarkStatus`（`:122`）对两路输入再取一次最小值（`:706-707`）。

`watermarkProcessor` 是 `MailboxWatermarkProcessor`，只有开启 `execution.checkpointing.unaligned.interruptible-timers.enabled`（`CORE/configuration/CheckpointingOptions.java:605-606`）时才会启用（`AbstractStreamOperator.java:394-400`）。一个 Watermark 可能一次触发成千上万个 Timer，这些 Timer 执行期间 Mailbox 被占用，Checkpoint 无法进行。开启这个选项后，Timer 的执行可以被 Mailbox 中的其他动作打断，比如 Checkpoint。

---

## 3. Timer：Watermark 的"消费者"

### 3.1 两个优先队列

`RT/streaming/api/operators/InternalTimerServiceImpl.java` 为每个 Timer 服务维护两个按时间排序的队列：`processingTimeTimersQueue`（`:54`）和 `eventTimeTimersQueue`（`:58`）。一个 Timer 由 **(timestamp, key, namespace)** 三元组唯一确定。对窗口算子来说，namespace 就是窗口本身。

```java
public void registerEventTimeTimer(N namespace, long time) {        // :249
    eventTimeTimersQueue.add(new TimerHeapInternalTimer<>(time, (K) keyContext.getCurrentKey(), namespace));
}

public boolean tryAdvanceWatermark(long time, ShouldStopAdvancingFn shouldStopAdvancingFn) {   // :328
    currentWatermark = time;
    while ((timer = eventTimeTimersQueue.peek()) != null
            && timer.getTimestamp() <= time && !cancelled && !interrupted) {
        keyContext.setCurrentKey(timer.getKey());       // 切换当前 key，保证 onEventTime 里读到的是这个 key 的状态
        eventTimeTimersQueue.poll();
        triggerTarget.onEventTime(timer);               // → WindowOperator.onEventTime / KeyedProcessFunction.onTimer
        interrupted = shouldStopAdvancingFn.test();     // 可中断 Timer 的检查点
    }
    return !interrupted;
}
```

几个由源码直接推出的结论：
- **事件时间 Timer 只在 Watermark 到来时检查**。在 `onElement` 里注册一个早于当前 Watermark 的 Timer，它不会立即触发，要等**下一个** Watermark。如果之后再也没有新的 Watermark，它就永远不会触发。
- **同一个 (时间, key, 窗口) 只存一份**。堆内存实现 `HeapPriorityQueueSet.add()` 先用每个 KeyGroup 一个的 HashMap 去重（`RT/runtime/state/heap/HeapPriorityQueueSet.java:121-122`），`TimerHeapInternalTimer.equals` 比较的正是这三个字段。所以窗口算子每来一条记录都调用一次 `registerEventTimeTimer(window.maxTimestamp())`，也不会产生重复的 Timer。
- **处理时间 Timer 只挂一个物理定时器**：`registerProcessingTimeTimer`（`:233`）只在新 Timer 比队头更早时，才取消旧的物理定时器、重新注册（`:239-244`）。

### 3.2 Timer 存在哪里

Timer 是 keyed state 的一部分，按 KeyGroup 切分（快照时走 `getSubsetForKeyGroup`，`:360-361`），扩缩容时随 KeyGroup 一起迁移（03 篇）。
- HashMap 状态后端：放在 JVM 堆中。
- RocksDB：由 `state.backend.rocksdb.timer-service.factory` 决定，**默认 `ROCKSDB`**（`flink-state-backends/flink-statebackend-rocksdb/.../RocksDBOptions.java:60-63`），Timer 存在 RocksDB 里，只在堆上为每个 KeyGroup 缓存 128 个（`:69-72`）。Timer 数量很大时，这是 RocksDB 作业 CPU 开销的一个常见来源。

---

## 4. 窗口算子

### 4.1 组成部分

```
input.keyBy(...)
     .window(WindowAssigner)          分配：一条记录属于哪些窗口
     [.trigger(Trigger)]              触发：什么时候计算（默认由 Assigner 给出）
     [.evictor(Evictor)]              计算前后剔除元素（会导致无法预聚合）
     [.allowedLateness(Duration)]     窗口结束后还保留多久
     [.sideOutputLateData(tag)]       真正迟到的数据去哪
     .reduce / aggregate / process
```

`WindowedStream`（`RT/streaming/api/datastream/WindowedStream.java`）把这些参数交给 `WindowOperatorBuilder`，由后者决定窗口状态的类型（`RT/streaming/runtime/operators/windowing/WindowOperatorBuilder.java`，状态名固定为 `"window-contents"`，`:81`）：

| API | 窗口状态 | 每个窗口存什么 |
|---|---|---|
| `reduce(f)` | `ReducingState`（`:178-185`） | **一个**聚合值 |
| `aggregate(f)` | `AggregatingState`（`:286`） | **一个**累加器 |
| `process(f)` | `ListState`（`:399`） | **所有**元素 |
| 任意 API + `evictor` | `ListState<StreamRecord>`（`:448-457`） | 所有元素，外加时间戳 |

大窗口、高吞吐的场景下，这张表决定了状态会有多大。只需要聚合结果时，应该用 `aggregate(aggFn, processWindowFn)`：增量聚合，同时保留拿到窗口元信息的能力。

如果流开启了异步状态（`isEnableAsyncState`，`WindowedStream.java:81`），会改用 `asyncReduce` 等方法（`:237-239`），生成基于 State V2 的异步窗口算子（03 篇第 7 节）。

### 4.2 窗口分配：对齐到纪元

`TumblingEventTimeWindows.assignWindows()`（`RT/streaming/api/windowing/assigners/TumblingEventTimeWindows.java:69`）：

```java
if (timestamp > Long.MIN_VALUE) {
    long start = TimeWindow.getWindowStartWithOffset(timestamp, (globalOffset + staggerOffset) % size, size);
    return Collections.singletonList(new TimeWindow(start, start + size));
} else {
    throw new RuntimeException("Record has Long.MIN_VALUE timestamp (= no timestamp marker). "
            + "Did you forget to call 'DataStream.assignTimestampsAndWatermarks(...)'?");     // :82-84
}
```

`getWindowStartWithOffset`（`RT/streaming/api/windowing/windows/TimeWindow.java:264`）就是 `timestamp - (timestamp - offset) % size`，负数单独处理。窗口是**对齐到 1970-01-01 00:00:00 UTC 的**，与作业何时启动、第一条数据是什么时间无关。所以用 1 天的窗口统计北京时间的"每天"时，需要设置 `offset = -8h`。

`TimeWindow.maxTimestamp()` 是 `end - 1`（`:84-86`）：窗口是左闭右开的 `start, end)`，Timer 注册在 `end - 1` 上。

### 4.3 `processElement`：非合并窗口

`RT/streaming/runtime/operators/windowing/WindowOperator.java`

```java
public void processElement(StreamRecord<IN> element) {                   // :293
    elementWindows = windowAssigner.assignWindows(value, timestamp, ctx);
    boolean isSkippedElement = true;
    ...
    for (W window : elementWindows) {                                     // :405
        if (isWindowLate(window)) continue;                               // 窗口已经被清理：跳过
        isSkippedElement = false;
        windowState.setCurrentNamespace(window);                          // namespace = 窗口
        windowState.add(element.getValue());                              // 进状态（或增量聚合）
        TriggerResult r = triggerContext.onElement(element);
        if (r.isFire())  emitWindowContents(window, windowState.get());
        if (r.isPurge()) windowState.clear();
        registerCleanupTimer(window);                                     // 注册清理 Timer
    }
    if (isSkippedElement && isElementLate(element)) {                     // :440
        if (lateDataOutputTag != null) sideOutput(element);               // 侧输出
        else numLateRecordsDropped.inc();                                 // 否则静默丢弃，只增加一个指标
    }
}
```

默认的 `EventTimeTrigger`（`RT/streaming/api/windowing/triggers/EventTimeTrigger.java`）只有几行：

```java
public TriggerResult onElement(Object element, long timestamp, TimeWindow window, TriggerContext ctx) {   // :37
    if (window.maxTimestamp() <= ctx.getCurrentWatermark()) {
        return TriggerResult.FIRE;                         // Watermark 已经越过窗口：立即触发（迟到数据的重算就走这里）
    } else {
        ctx.registerEventTimeTimer(window.maxTimestamp()); // 否则注册 Timer，等 Watermark
        return TriggerResult.CONTINUE;
    }
}
public TriggerResult onEventTime(long time, TimeWindow window, TriggerContext ctx) {                     // :50
    return time == window.maxTimestamp() ? TriggerResult.FIRE : TriggerResult.CONTINUE;
}
```

注意它返回的是 `FIRE` 而不是 `FIRE_AND_PURGE`：**窗口触发之后状态还在**，一直保留到清理 Timer 触发。这正是 `allowedLateness` 能够工作的前提。

### 4.4 `onEventTime`：同一个窗口的两个 Timer

每个事件时间窗口最多有**两个** Timer：

| Timer | 时间 | 注册者 | 作用 |
|---|---|---|---|
| 触发 Timer | `window.maxTimestamp()` | `EventTimeTrigger.onElement` | 输出结果 |
| 清理 Timer | `window.maxTimestamp() + allowedLateness` | `registerCleanupTimer`（`:631`） | 清空窗口状态和 Trigger 状态 |

`allowedLateness = 0` 时，两个 Timer 的时间相同，去重后只剩一个。

```java
public void onEventTime(InternalTimer<K, W> timer) {                     // :450
    ...
    TriggerResult r = triggerContext.onEventTime(timer.getTimestamp());
    if (r.isFire())  emitWindowContents(...);
    if (r.isPurge()) windowState.clear();
    if (windowAssigner.isEventTime() && isCleanupTime(triggerContext.window, timer.getTimestamp())) {
        clearAllState(triggerContext.window, windowState, mergingWindows);   // :487
    }
}
```

### 4.5 迟到数据的四种结局

两个判断条件（`:609-624`）：

```java
isWindowLate(window)   = cleanupTime(window) <= currentWatermark        // 窗口已被清理
isElementLate(element) = element.timestamp + allowedLateness <= currentWatermark
```

把一条事件时间为 `t`、所属窗口为 `[s, e)` 的记录，按到达时的 Watermark `W` 分成四种情况（L 表示 allowedLateness）：

| W 的范围 | 结局 | 源码路径 |
|---|---|---|
| `W < e-1` | **正常**：进状态，等待触发 | `onElement` → 注册 Timer |
| `e-1 ≤ W < e-1+L` | **迟到但被接受**：进状态，**立即再触发一次**，输出更新后的结果 | `onElement` 返回 `FIRE` |
| `W ≥ e-1+L`，且配置了侧输出 | **进侧输出** | `:440-442` |
| `W ≥ e-1+L`，未配置侧输出 | **静默丢弃**，`numLateRecordsDropped` 加 1 | `:444` |

第二种情况带来的副作用：**同一个窗口会输出多次**，每次都是完整的新结果（不是增量）。下游如果是追加写入，就会出现重复。第 7.1 节的实测中，`[0s,5s)` 就输出了两次。

---

## 5. 会话窗口：合并与"状态窗口"

会话窗口（`EventTimeSessionWindows`）是 `MergingWindowAssigner`：每条记录先得到一个 `[t, t+gap)` 的窗口，再与已有的相交窗口合并。相交的判断是 `start <= other.end && end >= other.start`（`TimeWindow.java:116-117`），**端点相接也算相交**。

合并的难点在于状态：每次合并都把多个窗口的状态搬到一个新 namespace 下，代价很高。`MergingWindowSet`（`RT/streaming/runtime/operators/windowing/MergingWindowSet.java`）的做法是维护一个映射 `mapping: 窗口 → 状态窗口`（`:63`）：

- 窗口合并后，结果窗口**继续使用其中一个原窗口的 namespace** 存放状态（`:190-201`），只把其余原窗口的状态并过来：`windowMergingState.mergeNamespaces(stateWindowResult, mergedStateWindows)`（`WindowOperator.java` 合并回调的最后一步）。
- 这个映射本身存在 operator 的 `ListState` 中，每次处理完调用 `persist()`，只在映射有变化时才写回（`MergingWindowSet.java:100`）。

合并回调中还有一个检查（`WindowOperator.java:322-333`）：如果合并后的窗口已经被 Watermark 越过，直接抛出 `UnsupportedOperationException`。自定义 `MergingWindowAssigner` 时，如果合并逻辑会让窗口变短，就会遇到这个异常。

Trigger 也要支持合并：`EventTimeTrigger.canMerge()` 返回 true（`:66`），`onMerge` 为合并后的窗口注册新 Timer，并且**只在 Watermark 还没越过时才注册**（`:71-79`），否则会和 `onElement` 的立即触发重复。

---

## 6. Watermark 对齐（FLIP-182）

问题：一个作业从两个 Kafka topic 读数据，其中一个在追历史数据，事件时间比另一个落后几小时。下游 join 或窗口必须等较慢的那一路，快的一路的数据就只能堆积在状态里，状态会越来越大。

解决办法：让**跑得太快的 Source 暂停读取**。

```
SourceOperator（每个 subtask）                              SourceCoordinator（JM）
  emitLatestWatermark()                                       RT/runtime/source/coordinator/SourceCoordinator.java
   └ 发送 ReportedWatermarkEvent  ────────────────────────▶   WatermarkAggregator 汇总同一 group 所有 subtask 的
     （SourceOperator.java:640-656）                           Watermark，取最小值（:835-880，又是一个小顶堆）
                                                              announceCombinedWatermark()（:184），每 updateInterval 一次（:300-306）
                                                               maxAllowedWatermark = 全局最小 Watermark + maxAllowedDrift   (:205-215)
  handleOperatorEvent(WatermarkAlignmentEvent) ◀──────────   发送 WatermarkAlignmentEvent
   └ currentMaxDesiredWatermark = event.getMaxWatermark()     (SourceOperator.java:719-722, :774-776)
   └ checkWatermarkAlignment()                                (:900)
       shouldWaitForAlignment() = currentMaxDesiredWatermark < 自己的 Watermark   (:918-919)
       → 超过的 split 调用 sourceReader.pauseOrResumeSplits(...)                   (:868-881)
       → mainInputActivityClock.pause()，暂停期间不会被判定为空闲                   (:906)
```

`updateInterval` 默认 1 秒（`CORE/api/common/eventtime/WatermarksWithWatermarkAlignment.java:29`）。

有两点要注意：
- 对齐的单位是 **watermark group**。同一个 group 中的所有 Source（可以是不同的 Source）一起对齐。
- split 级别的暂停需要 SourceReader 实现 `pauseOrResumeSplits`。没有实现时，只能暂停整个 subtask，一个 subtask 读多个 split 时效果就会打折扣。相关测试见 `flink-runtime/src/test/.../SourceOperatorAlignmentTest.java` 和 `flink-tests/src/test/.../WatermarkAlignmentITCase.java`。

---

## 7. 动手实验

示例：[`WindowDemo.java`，一共三种模式。

### 7.1 迟到数据（`late`）

配置：并行度 1，`forBoundedOutOfOrderness(2s)`，5 秒滚动窗口，`allowedLateness(3s)`，迟到数据走侧输出。Source 按固定顺序每 500ms 发一条，事件时间（秒）依次为 `1, 2, 4, 6, 3, 8, 2, 11, 1, 13`。500ms 大于 200ms 的 Watermark 周期，保证每两条记录之间至少发出一次 Watermark，结果才是确定的。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.time.WindowDemo late
```

本机实测输出：

```
FIRE window=[0s,5s) count=4 elements=[1s, 2s, 4s, 3s] watermark=5.999s wall=+4.4s
FIRE window=[0s,5s) count=5 elements=[1s, 2s, 4s, 3s, 2s] watermark=5.999s wall=+4.8s
LATE (side output) event=1s
FIRE window=[5s,10s) count=2 elements=[6s, 8s] watermark=10.999s wall=+6.3s
FIRE window=[10s,15s) count=2 elements=[11s, 13s] watermark=MAX wall=+7.2s
```

逐条对照源码：

| 事件 | 到达时 W | 发生了什么 |
|---|---|---|
| 1, 2, 4 | MIN → 0.999 → 1.999 | 进入 `[0,5)`，注册 Timer 4.999 和清理 Timer 7.999 |
| 6 | 1.999 | 进入 `[5,10)`；之后 W 变为 **3.999** |
| **3** | 3.999 | 乱序但没有迟到（3.999 < 4.999），正常进入 `[0,5)` |
| 8 | 3.999 | 之后 W 变为 **5.999**，越过 4.999 → **第 1 行**：`[0,5)` 第一次触发，4 个元素 |
| **2** | 5.999 | 迟到了，但 5.999 < 7.999（清理时间），窗口还在。`EventTimeTrigger.onElement` 发现 4.999 ≤ W，立即 `FIRE` → **第 2 行**：`[0,5)` 再次触发，5 个元素。注意两行的 `watermark` 相同 |
| 11 | 5.999 | 之后 W 变为 8.999，越过 7.999 → 清理 Timer 触发，`onEventTime` 返回 `CONTINUE`（7.999 ≠ 4.999），但 `isCleanupTime` 为真，`[0,5)` 的状态被清空 |
| **1** | 8.999 | `isWindowLate`：7.999 ≤ 8.999，跳过；`isElementLate`：1+3=4 ≤ 8.999 → **第 3 行**：进侧输出 |
| 13 | 8.999 | 之后 W 变为 10.999，越过 9.999 → **第 4 行**：`[5,10)` 触发 |
| 输入结束 | — | 有界 Source 发出 `MAX_WATERMARK` → **第 5 行**：`[10,15)` 触发，`watermark=MAX` |

### 7.2 会话窗口合并（`session`）

配置：3 秒 gap，事件时间依次为 `1, 2, 10, 7, 20`。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.time.WindowDemo session
```

本机实测输出：

```
FIRE window=[1s,5s) count=2 elements=[1s, 2s] watermark=7.999s wall=+2.6s
FIRE window=[7s,13s) count=2 elements=[10s, 7s] watermark=17.999s wall=+3.6s
FIRE window=[20s,23s) count=1 elements=[20s] watermark=MAX wall=+4.5s
```

- `1` 和 `2` 的窗口 `[1,4)`、`[2,5)` 相交，合并为 `[1,5)`。
- `10` 先得到 `[10,13)`；随后到达的 `7` 得到 `[7,10)`，端点 10 相接，也算相交，于是**向前**合并为 `[7,13)`。`elements=[10s, 7s]` 的顺序说明，`7` 是被追加到 `[10,13)` 原有的状态里的。对照 `MergingWindowSet.addWindow()`：新窗口会先从待合并集合中移除（`:183`），因为它还没有任何状态；状态窗口只从已有窗口中选（`:190`）。所以合并后的 `[7,13)`，状态仍然存放在 namespace `[10,13)` 下（第 5 节）。

### 7.3 空闲分区让窗口停止触发（`idle`）

配置：并行度 2，Source 每秒共发 20 条，事件时间取当前系统时间。之后的 filter **丢掉 subtask 1 的所有数据**，再调用 `assignTimestampsAndWatermarks`，最后是 2 秒滚动窗口。运行 12 秒。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.time.WindowDemo idle
```

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.time.WindowDemo idle idleness
```

本机实测：

```
# 不加 idleness
>>> idleness=false: 0 window firings in 12s

# 加 withIdleness(2s)
1> FIRE key=k1 count=1 wall=+3.9s
2> FIRE key=k0 count=1 wall=+3.9s
1> FIRE key=k3 count=1 wall=+3.9s
2> FIRE key=k2 count=1 wall=+3.9s
1> FIRE key=k1 count=5 wall=+5.1s
2> FIRE key=k0 count=5 wall=+5.1s
>>> idleness=true: 24 window firings in 12s
```

不加 idleness 时，subtask 1 的 `TimestampsAndWatermarksOperator` 一条数据也收不到。`BoundedOutOfOrdernessWatermarks` 算出的值一直是 `Long.MIN_VALUE`，被 `WatermarkEmitter` 的 `ts <= currentWatermark` 检查拦下，**一个 Watermark 也不会发出**。下游每个窗口 subtask 的 `StatusWatermarkValve` 中，来自 subtask 1 的通道永远停在 `MIN`，占据堆顶，所以 12 秒内一个窗口都没有触发。

加上 idleness 之后，大约启动 4 秒时（JVM 和作业启动约 1~2 秒，加上 2 秒空闲超时和周期检查的延迟），subtask 1 发出 IDLE，被移出堆，窗口开始正常触发。之后每个窗口有 5 条：每秒 20 条，只保留 subtask 0 的一半，每个 2 秒窗口 20 条，分给 4 个 key。

**生产环境中最常见的情况**：Source 的并行度大于 Kafka 分区数，多出来的 subtask 没有分到分区；或者某个分区长时间没有新数据。表现就是"作业在跑，数据在进，窗口却一直不出结果"。

### 7.4 读社区的测试

Watermark 合并逻辑有专门的单元测试，本机实测 14 个测试全部通过，耗时约 16 秒：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink && ./mvnw -s ../maven-settings-aliyun.xml -B -q -pl flink-runtime -Dfast -Dtest=StatusWatermarkValveTest -Dsurefire.failIfNoSpecifiedTests=false test
```

推荐先读 `testMultipleInputWatermarkAdvancingAsChannelsIndividuallyBecomeIdle`（`flink-runtime/src/test/.../watermarkstatus/StatusWatermarkValveTest.java:276`），它就是 7.3 节现象的最小复现。

窗口算子的测试比较特别：`WindowOperator` 的代码在 `flink-runtime`，测试却在 `flink-streaming-java/src/test/java/org/apache/flink/streaming/runtime/operators/windowing/WindowOperatorTest.java`，其中 `testLateness`（`:2038`）和 `testSideOutputDueToLatenessTumbling`（`:2249`）对应本篇第 4.5 节。这些测试都用 `KeyedOneInputStreamOperatorTestHarness` 手动推送记录和 Watermark，**不需要启动集群**。以后给窗口相关代码提 PR，测试基本都是照这个模式写的。

---

## 8. 断点清单

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `BoundedOutOfOrdernessWatermarks.java:69` | 周期发出的 Watermark 值 |
| 2 | `ProgressiveTimestampsAndWatermarks.java:332` | 多个 split 的 Watermark 合并 |
| 3 | `TimestampsAndWatermarksOperator.java:157` | 上游 Watermark 被丢弃 |
| 4 | `StatusWatermarkValve.java:281` | 条件断点 `alignedSubpartitionStatuses.peek().watermark == Long.MIN_VALUE`：找出卡住的通道 |
| 5 | `StatusWatermarkValve.java:199` | 空闲状态切换 |
| 6 | `AbstractStreamOperator.java:700` | Watermark 推进 Timer |
| 7 | `InternalTimerServiceImpl.java:340` | 每一个被触发的事件时间 Timer |
| 8 | `WindowOperator.java:405` | 记录被分配到哪些窗口 |
| 9 | `EventTimeTrigger.java:42` | 迟到数据引起的立即触发 |
| 10 | `WindowOperator.java:440` | 迟到数据进侧输出或被丢弃 |
| 11 | `WindowOperator.java:487` | 窗口状态被清理 |
| 12 | `MergingWindowSet.java:153` | 会话窗口合并 |
| 13 | `SourceCoordinator.java:184` | Watermark 对齐的全局计算 |

调试 `late` 模式时，建议在 IDE 中把 `RateLimiterStrategy.perSecond(2)` 调低，或者直接在断点处暂停。暂停不会影响结果，因为 Watermark 的推进完全由数据驱动：暂停期间处理时间在流逝，但周期定时器只是把**同一个** Watermark 重复发出，`WatermarkToDataOutput.emitWatermark()` 会过滤掉不增长的值（`RT/streaming/api/operators/source/WatermarkToDataOutput.java:77-79`）。

---

## 9. 课后练习

1. **推导题**：把 7.1 节的 `allowedLateness` 改成 0，不运行程序，先写出预期输出，再运行验证。提示：清理时间变为 4.999，第二个 `2` 和最后的 `1` 分别是什么结局？
2. **滑动窗口**：改用 `SlidingEventTimeWindows.of(10s, 5s)`，一条记录会进入几个窗口？部分窗口已经迟到、部分还没有时，`isSkippedElement` 的值是什么？这条记录会进侧输出吗？（对照 `WindowOperator.java:405-446`）
3. **Timer 实验**：写一个 `KeyedProcessFunction`，在 `processElement` 中注册一个时间为 `ctx.timerService().currentWatermark() - 1` 的事件时间 Timer，验证它不会立即触发，而是在下一个 Watermark 到来时触发（第 3.1 节的结论）。
4. **读测试**：读 `StatusWatermarkValveTest.java:276` 的测试，画出每一步之后堆中的内容和输出的 Watermark。
5. **SQL 对照**：SQL 中 `WATERMARK FOR ts AS ts - INTERVAL '2' SECOND` 对应的运行时算子是 `flink-table/flink-table-runtime/.../wmassigners/WatermarkAssignerOperator.java`。对比它和 `TimestampsAndWatermarksOperator`：Watermark 的计算方式有什么不同？有没有"减 1"？
6. **进阶**：Flink 2.x 的 DataStream V2 引入了通用 Watermark（`flink-datastream-api/.../extension/eventtime/EventTimeExtension.java`、`RT/streaming/runtime/watermark/extension/eventtime/EventTimeWatermarkHandler.java`）。读一读它和本篇的 `Watermark` 有什么区别，为什么 `RecordWriterOutput` 中有两个 `emitWatermark` 重载（`:148`、`:254`）。

---

## 10. 踩坑记录

| 现象 | 原因 | 解决 |
|---|---|---|
| `Record has Long.MIN_VALUE timestamp (= no timestamp marker)` | 用了事件时间窗口，但没有设置时间戳（`TumblingEventTimeWindows.java:82-84`） | 在 `fromSource` 中传入带 `withTimestampAssigner` 的策略 |
| 作业在跑，窗口一直不出结果 | 有 subtask 或分区没有数据，Watermark 被 `MIN` 卡住（7.3 节实测） | `withIdleness(...)`；或者让 Source 的并行度不超过分区数 |
| 同一个窗口的结果输出了多次 | `allowedLateness > 0` 时，每条迟到数据都会引起一次完整的重新触发（7.1 节第 2 行） | 下游用 upsert 写入，或者 `allowedLateness = 0` 加侧输出 |
| 窗口作业状态很大 | `process()` 或 `evictor` 会让窗口保存所有元素（4.1 节） | 改用 `aggregate(aggFn, processWindowFn)` |
| 背压时窗口结果延迟 | Watermark 和数据走同一条通道，被背压挡住（2.1 节） | 先解决背压（04 篇） |
| 批模式下窗口只在最后一次性输出 | 批模式不产生中间 Watermark（1.3 节） | 这是预期行为 |
| 写 `late` 示例时，结果时对时错 | 最初让 Source 发得太快，两条记录之间可能没有 Watermark，结果依赖时序 | 让记录间隔（500ms）大于 Watermark 周期（200ms） |
| 一天的窗口从北京时间早上 8 点开始 | 窗口对齐到 UTC 纪元（4.2 节） | `TumblingEventTimeWindows.of(Duration.ofDays(1), Duration.ofHours(-8))` |

---

## 11. 小结

- **生成**：`WatermarkGenerator` 在 Source 中按 split 运行，每 200ms 周期性发出 `最大时间戳 - 乱序容忍 - 1`。
- **传播**：Watermark 和数据走同一条通道；多输入时由 `StatusWatermarkValve` 用小顶堆取所有 ACTIVE 通道的最小值；IDLE 通道不参与计算。
- **消费**：算子**先**触发 Timer，**再**向下游转发 Watermark；Timer 由 (时间, key, namespace) 唯一确定并去重。
- **窗口**：每个窗口一个触发 Timer、一个清理 Timer；`FIRE` 不清状态，因此迟到数据可以重新触发；超过 `allowedLateness` 的数据进侧输出，否则被静默丢弃。
- **对齐**：SourceCoordinator 汇总全局最小 Watermark，让跑得太快的 split 暂停读取。

下一篇（09）讲 **Source / Sink 新架构**：本篇第 1.3 节的 `SourceOperator` 和第 6 节的 `SourceCoordinator` 会在那里完整展开，另外还有 Sink V2 如何通过两阶段提交实现端到端的 exactly-once。
:::
