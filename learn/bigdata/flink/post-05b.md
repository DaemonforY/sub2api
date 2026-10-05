---
title: "Flink 2.x 源码精读（五·下）：Changelog，同一个 GROUP BY 为什么有时输出两条"
description: "同一条 GROUP BY，写到 print 表会输出 -U/+U，写到 blackhole 表只输出 +U。顺着这个现象，读懂 Flink 2.3 的 Changelog 推导：上游输出什么由下游决定；聚合算子什么时候发 -U、+U、-D；MiniBatch 为什么能少发消息；以及 2.3 新增的 ON CONFLICT 检查。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-05b-cover.webp"}]]
---

# Flink 2.x 源码精读（五·下）：Changelog，同一个 GROUP BY 为什么有时输出两条

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-05b-cover.webp" alt="Yui和Kai在分流管道旁观察聚合消息流向" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai在分流管道旁观察聚合消息流向<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**，文中 `文件:行号` 都在 `release-2.3.0` 上核对过，运行结果来自本机实测（Apple M5 Pro，JDK 17）。
> 路径缩写：`PLN` = `flink-table/flink-table-planner/src/main`，`TRT` = `flink-table/flink-table-runtime/src/main/java/org/apache/flink/table/runtime`，`PRG` = `PLN/scala/org/apache/flink/table/planner/plan/optimize/program`
> 前置阅读：第五讲上篇（优化器的 13 个阶段、`physical_rewrite`）

---

先看一个实验。

同一条 SQL：

```sql
SELECT word, COUNT(*) AS cnt FROM words GROUP BY word
```

分别写到两张表里，一张用 `print` 连接器，一张用 `blackhole` 连接器，两张表的字段完全一样，都没有主键。用 `EXPLAIN` 看优化后的物理计划（本机实际输出，节选聚合那一行）：

```text
-- INSERT INTO print_sink ...
GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UB,UA])

-- INSERT INTO bh_sink ...（blackhole）
GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UA])
```

**同一个聚合，SQL 一个字没改，输出的消息类型不一样**：前者多了 `UB`（UPDATE_BEFORE，撤回旧值）。

上篇结尾还留了另一个现象：把这个查询再套一层聚合，内层也变成了 `[I,UB,UA]`。

这说明一件事：**一个算子输出什么类型的消息，不是它自己决定的，是它的下游决定的。**

这一篇，我们来看：

1. Changelog 里的 4 种消息是什么？`-U` 为什么是必要的？
2. 优化器是怎么推导每个节点输出哪些消息的？为什么"下游说了算"？
3. 聚合算子在运行时什么时候发 `-U`、`+U`、`-D`？
4. MiniBatch 为什么能让输出变少？
5. Flink 2.3 新增的 `ON CONFLICT` 检查：为什么升级后有的 INSERT 会报错？

---

## 一、四种消息

一张动态表的每一次变化，都用一条带类型的消息表示。类型定义在 `flink-core/src/main/java/org/apache/flink/types/RowKind.java:31-52`：

| 类型 | 简写 | 含义 |
|---|---|---|
| `INSERT` | `+I` | 新增一行 |
| `UPDATE_BEFORE` | `-U` | 更新前的旧值（撤回） |
| `UPDATE_AFTER` | `+U` | 更新后的新值 |
| `DELETE` | `-D` | 删除一行 |

`explain` 里的 `changelogMode=[I,UB,UA,D]` 用的是另一套简写，说的是同一件事：这个节点**可能**输出哪几种消息。

### 1.1 实测：6 条数据，看每一条输出

`SqlDemo changelog` 用固定输入 `a a b a c b`，通过 `toChangelogStream` 把结果打印出来。本机实际输出（连续跑了 3 次，输出完全相同，完整日志在 `assets/logs/sql-changelog.log`）：

**Q1：`SELECT word, COUNT(*) AS cnt FROM words GROUP BY word`**

```text
+I[a, 1]
-U[a, 1]
+U[a, 2]
+I[b, 1]
-U[a, 2]
+U[a, 3]
+I[c, 1]
-U[b, 1]
+U[b, 2]
```

6 条输入，9 条输出。每个 word 第一次出现发 `+I`，之后每次变化发一对 `-U` / `+U`。

**Q2：嵌套聚合，统计"出现了 cnt 次的单词有几个"**

```sql
SELECT cnt, COUNT(*) AS num_words FROM (Q1) GROUP BY cnt
```

把内层（Q1）和外层的输出对齐：

| 输入 | 内层输出 | 外层输出 |
|---|---|---|
| a | `+I[a,1]` | `+I[1,1]` |
| a | `-U[a,1]` `+U[a,2]` | `-D[1,1]` `+I[2,1]` |
| b | `+I[b,1]` | `+I[1,1]` |
| a | `-U[a,2]` `+U[a,3]` | `-D[2,1]` `+I[3,1]` |
| c | `+I[c,1]` | `-U[1,1]` `+U[1,2]` |
| b | `-U[b,1]` `+U[b,2]` | `-U[1,2]` `+U[1,1]` `+I[2,1]` |

> 外层一列是本机 Q2 的实际输出，按顺序共 11 条；内层一列是同一份输入下 Q1 的实际输出，按因果关系对齐到每一行。

看第二行：a 从 1 次变成 2 次。内层先发 `-U[a,1]`，外层收到后，把"出现 1 次的单词"减 1，从 1 变成 0，于是发 `-D[1,1]`，删掉这一行；再收到 `+U[a,2]`，"出现 2 次的单词"从无到有，发 `+I[2,1]`。

最终结果：出现 3 次的 1 个（a），出现 2 次的 1 个（b），出现 1 次的 1 个（c）。正确。

### 1.2 如果没有 `-U` 会怎样

假设内层只发 `+I` 和 `+U`，不发 `-U`。外层只能看到"某个单词现在出现了 N 次"，不知道它之前是几次，也就没法把旧的那一组减掉：

- `+I[a,1]`、`+I[b,1]`、`+I[c,1]` → "出现 1 次"累加了 3 次
- `+U[a,2]`、`+U[b,2]` → "出现 2 次"累加了 2 次
- `+U[a,3]` → "出现 3 次"累加了 1 次

结果是 `1→3, 2→2, 3→1`，而正确答案是 `1→1, 2→1, 3→1`。

> 这一段是按照上面的输入推演的，不是实际运行的输出。Flink 也不会生成这样的计划，下一节会看到，外层聚合会强制要求内层发 `-U`。

**`-U` 的作用：当下游按照上游的某个值做分组时，上游的值一变，下游需要知道旧值是什么，才能把它从旧的组里撤掉。**

---

## 二、优化器怎么推导：下游说了算

<figure class="ai-figure"><img src="/bigdata-img/flink/post-05b-1.webp" alt="下游接收器向上游聚合器提出消息需求" width="960" height="640" loading="lazy" /><figcaption>下游接收器向上游聚合器提出消息需求<span>AI 生成配图</span></figcaption></figure>

### 2.1 在哪里推导

上篇讲过，`physical_rewrite` 阶段的第 3 步是 `Changelog mode inference`，执行的是 `FlinkChangelogModeInferenceProgram`（`PRG/FlinkChangelogModeInferenceProgram.scala`，1757 行）。它的 `optimize` 方法（`:56`）分三步：

| 步骤 | 位置 | 推导什么 | 方向 |
|---|---|---|---|
| step1 | `:57-65` | `ModifyKindSet`：每个节点会产生哪些**种类**的变化（INSERT / UPDATE / DELETE） | 从下往上（Source → Sink） |
| step2 | `:67-87` | `UpdateKind`：有 UPDATE 的节点，要不要发 UPDATE_BEFORE | 从上往下（Sink → Source） |
| step3 | `:89-112` | `DeleteKind`：DELETE 是按 key 删，还是带完整的行 | 从上往下 |

step1 是"上游能产生什么"，step2、step3 是"下游需要什么"。两者合起来，才是 `explain` 里看到的 `changelogMode`。

### 2.2 step1：聚合会产生 UPDATE，输入有撤回时还会产生 DELETE

聚合节点在 step1 的处理（`:194-208`）：

```scala
case agg: StreamPhysicalGroupAggregate =>
  // agg support all changes in input
  val children = visitChildren(agg, ModifyKindSetTrait.ALL_CHANGES)
  val inputModifyKindSet = getModifyKindSet(children.head)
  val builder = ModifyKindSet
    .newBuilder()
    .addContainedKind(ModifyKind.INSERT)
    .addContainedKind(ModifyKind.UPDATE)
  if (
    inputModifyKindSet.contains(ModifyKind.UPDATE) ||
    inputModifyKindSet.contains(ModifyKind.DELETE)
  ) {
    builder.addContainedKind(ModifyKind.DELETE)
  }
```

- 聚合一定会产生 INSERT 和 UPDATE；
- 只有当输入本身带 UPDATE 或 DELETE 时，才会产生 DELETE。

对应到实验：Q1 的输入只有 INSERT，所以 Q1 不会有 `D`；Q2 外层的输入带 UPDATE，所以外层是 `[I,UA,D]`。前面表格里的 `-D[1,1]`，就是这种情况：一个组的计数被撤回到 0。

### 2.3 step2：聚合会要求上游发 UPDATE_BEFORE

聚合节点在 step2 的处理（`:617-626`）：

```scala
case _: StreamPhysicalGroupAggregate | _: StreamPhysicalGroupTableAggregate | ... =>
  // Aggregate, TableAggregate, OverAggregate, Limit, GroupWindowAggregate, WindowAggregate,
  // and WindowTableAggregate requires update_before if there are updates
  val requiredChildUpdateTrait = beforeAfterOrNone(getModifyKindSet(rel.getInput(0)))
  val children = visitChildren(rel, requiredChildUpdateTrait)
  // use requiredTrait as providedTrait, because they should support all kinds of UpdateKind
  createNewNode(rel, children, requiredUpdateTrait)
```

两件事：

1. **对上游**：只要输入里有 UPDATE，就要求上游发 `BEFORE_AND_AFTER`（`beforeAfterOrNone`，`PLN/scala/org/apache/flink/table/planner/plan/trait/UpdateKindTrait.scala:91-98`）。这就是 Q2 内层被要求发 `-U` 的原因。
2. **对自己**：输出哪种 UpdateKind，用的是下游传下来的 `requiredUpdateTrait`。聚合两种都能输出，下游要什么就给什么。

### 2.4 从哪里开始：Sink 的要求

推导从根节点开始（`:70-81`）：

```scala
val requiredUpdateKindTraits = if (rootModifyKindSet.contains(ModifyKind.UPDATE)) {
  if (context.isUpdateBeforeRequired) {
    Seq(UpdateKindTrait.BEFORE_AND_AFTER)
  } else {
    // update_before is not required, and input contains updates
    // try ONLY_UPDATE_AFTER first, and then BEFORE_AND_AFTER
    Seq(UpdateKindTrait.ONLY_UPDATE_AFTER, UpdateKindTrait.BEFORE_AND_AFTER)
  }
}
```

能不要 UB 就不要（先试 `ONLY_UPDATE_AFTER`）。如果根节点是 Sink，就由 Sink 自己的要求决定（`:586-588`，`inferSinkRequiredTraits`，`:1005-1027`）。

Sink 的要求来自连接器实现的 `getChangelogMode`。对比两个内置连接器：

```java
// print：上游给什么就要什么（PrintTableSinkFactory.java:121-123）
public ChangelogMode getChangelogMode(ChangelogMode requestedMode) {
    return requestedMode;
}

// blackhole：把 UPDATE_BEFORE 去掉（BlackHoleTableSinkFactory.java:71-79）
for (RowKind kind : requestedMode.getContainedKinds()) {
    if (kind != RowKind.UPDATE_BEFORE) {
        builder.addContainedKind(kind);
    }
}
```

（两个文件都在 `flink-table/flink-table-api-java-bridge/src/main/java/org/apache/flink/connector/` 下。）

而传给 `getChangelogMode` 的 `requestedMode`，是 `ModifyKindSet.toDefaultChangelogMode()`（`PLN/java/org/apache/flink/table/planner/plan/trait/ModifyKindSet.java:109-115`）：**只要有 UPDATE，默认就带上 UPDATE_BEFORE**。所以 print 拿到的是带 UB 的，blackhole 把它去掉了。

开头实验的答案就在这里：**print 要 UB，blackhole 不要；聚合下游要什么就输出什么。**

### 2.5 回头看上篇的两个现象

**现象一**：上篇对单独一条 `SELECT` 做 `explainSql`，Q1 显示的是 `[I,UA]`；但这一篇用 `toChangelogStream` 实际运行，却收到了 `-U`。

原因是两者的根节点不一样：

- 对单独一条 `SELECT` 做 `explainSql` 时，计划里**没有 Sink**（`PLN/scala/org/apache/flink/table/planner/delegation/PlannerBase.scala:629-630`，`case _ => relNode`），根节点就是聚合本身，走的是"能不要 UB 就不要"；
- `toChangelogStream(table)` 不指定 ChangelogMode 时，内部 Sink 的 `getChangelogMode` 直接返回 `requestedMode`（`PLN/java/org/apache/flink/table/planner/connectors/ExternalDynamicSink.java:68-73`），也就是带 UB 的默认模式。

**所以，想知道一条 SQL 实际会输出什么，要 `EXPLAIN` 完整的 `INSERT INTO ...` 语句，而不是只 `EXPLAIN` 里面的 `SELECT`。**

**现象二**：Q2 内层是 `[I,UB,UA]`，因为外层聚合要求它发 UB（2.3 节）。

### 2.6 有主键的 upsert Sink

Sink 声明自己只要 `ONLY_UPDATE_AFTER` 时，还有一层检查（`:1012-1020`）：Sink 的主键能不能被输入的 **upsert key** 满足（`canUpsertKeysWithImmutableColsSatisfyPk`，`:1047-1072`）。

upsert key 可以理解为"能唯一确定一行的那几列"。`GROUP BY word` 的结果，upsert key 就是 `word`。

| Sink | 判断 | 聚合输出 |
|---|---|---|
| blackhole，无主键 | 没有主键，直接通过（`:1049-1051`） | `[I,UA]` |
| blackhole，主键 `word` | upsert key `{word}` ⊆ 主键 `{word}`，通过 | `[I,UA]` |
| blackhole，主键 `cnt` | upsert key `{word}` 不在主键 `{cnt}` 里，不通过，回退到 `BEFORE_AND_AFTER` | 见第五节 |

前两行是 `SqlDemo sinks` 的实测结果（`assets/logs/sql-sinks.log`）。第三行比较特殊，放到第五节讲。

---

## 三、运行时：聚合算子什么时候发什么

优化器决定了"能发什么"，具体每条数据发什么，在运行时决定。

### 3.1 generateUpdateBefore 从哪来

`StreamPhysicalGroupAggregate` 翻译成 ExecNode 时，带上了一个标志（`PLN/scala/org/apache/flink/table/planner/plan/nodes/physical/stream/StreamPhysicalGroupAggregate.scala:91`）：

```scala
val generateUpdateBefore = ChangelogPlanUtils.generateUpdateBefore(this)
```

它的实现就是看推导出来的 UpdateKind 是不是 `BEFORE_AND_AFTER`（`PLN/scala/org/apache/flink/table/planner/plan/utils/ChangelogPlanUtils.scala:61-64`）。**这是优化器的推导结果传到运行时的唯一通道。**

### 3.2 GroupAggHelper.processElement

非 MiniBatch 模式下，每条数据的处理在 `TRT/operators/aggregate/utils/GroupAggHelper.java:68-151`。简化后的逻辑：

```java
if (accumulators == null) {               // 这个 key 第一次出现
    if (isRetractMsg(input)) return;      // :76-78 第一条就是撤回，直接丢掉
    firstRow = true;
    accumulators = function.createAccumulators();
}
function.setAccumulators(accumulators);
RowData prevAggValue = function.getValue();          // :88 旧结果
if (isAccumulateMsg(input)) function.accumulate(input);   // :91-93 +I / +U
else function.retract(input);                             // :94-97 -U / -D
RowData newAggValue = function.getValue();           // :99 新结果

if (!recordCounter.recordCountIsZero(accumulators)) {     // :104 这个组里还有数据
    updateAccumulatorsState(accumulators);
    if (!firstRow) {
        if (!ttlConfig.isEnabled() && equaliser.equals(prevAggValue, newAggValue)) {
            return;                                       // :112-117 结果没变，什么都不发
        }
        if (generateUpdateBefore) { 发 -U(prevAggValue) } // :120-126
        发 +U(newAggValue)                                // :128
    } else {
        发 +I(newAggValue)                                // :133
    }
} else {                                                  // :138 组里没数据了
    if (!firstRow) 发 -D(prevAggValue)                    // :141-145
    clearAccumulatorsState();
}
```

（`isAccumulateMsg` / `isRetractMsg` 在 `flink-table/flink-table-runtime/src/main/java/org/apache/flink/table/data/util/RowDataUtil.java:31-43`：`+I`、`+U` 是累加，`-U`、`-D` 是撤回。）

对照 1.1 节的表格：

| 输出 | 对应的分支 |
|---|---|
| `+I[2,1]`：组 2 第一次出现 | `firstRow` → `+I` |
| `-U[1,1]` `+U[1,2]`：组 1 从 1 变成 2 | 不是第一次、结果变了、`generateUpdateBefore=true` → `-U` + `+U` |
| `-D[1,1]`：组 1 被撤回到 0 | `recordCountIsZero` → `-D` |

三个容易忽略的细节：

1. **结果没变就不发**（`:112-117`）：比如 `MAX` 收到一个比当前最大值小的数，结果不变，下游什么也收不到。但开启了状态 TTL 时**仍然会发**，注释里的原因是防止下游的状态被过早清理。
2. **第一条就是撤回，直接丢掉**（`:76-78`）：注释说这可能发生在状态被清理之后。也就是说，开了 TTL、状态过期后再收到旧数据的撤回，**会被静默丢弃**。
3. **`-U` 可以省，`+U` 不能省**：`generateUpdateBefore=false` 时只是不发 `-U`，`+U` 照发（`:120-128`）。

### 3.3 retract 方法：生成还是不生成

上篇看到，Q1 生成的 `retract` 方法直接抛异常，因为它的输入只有 INSERT。

这次用 `codegen q2` 看 Q2 外层的聚合（`GroupAggsHandler$8`，本机实际输出，删掉了空行，完整日志在 `assets/logs/sql-codegen-q2.log`）：

```java
public void retract(org.apache.flink.table.data.RowData retractInput) throws Exception {
  boolean isNull$5;
  long result$6;
  isNull$5 = agg0_count1IsNull || false;
  result$6 = -1L;
  if (!isNull$5) {
  result$6 = (long) (agg0_count1 - ((long) 1L));
  }
  agg0_count1 = result$6;;
  agg0_count1IsNull = isNull$5;
}
```

和 `accumulate` 唯一的区别是 `+ 1L` 变成了 `- 1L`。

这就是上篇第四节说的"Changelog 推导排在两阶段聚合和代码生成之前"的意义：**生成代码的时候，已经知道这个聚合会不会收到撤回消息。** 不会收到，就不生成 `retract` 的逻辑；`explain` 里也用 `COUNT` 和 `COUNT_RETRACT` 区分开。

> 对于 `COUNT`、`SUM` 这类函数，支持撤回很简单，减回去就行。但 `MAX`、`MIN` 撤回之后要知道"第二大的值"是多少，就必须把所有值都存下来，状态会大很多。这是嵌套聚合、或者在撤回流上做 `MAX` 时状态变大的原因之一。本篇没有对 `MAX` 做实验，这一点请以你自己的测试为准。

---

## 四、MiniBatch：同样的输入，输出少了很多

<figure class="ai-figure"><img src="/bigdata-img/flink/post-05b-2.webp" alt="MiniBatch把多次变化合并后一次输出" width="960" height="640" loading="lazy" /><figcaption>MiniBatch把多次变化合并后一次输出<span>AI 生成配图</span></figcaption></figure>

同样的输入 `a a b a c b`，开启 MiniBatch（`allow-latency=1s`，`size=1000`）后的实际输出（`assets/logs/sql-changelog-minibatch.log`）：

```text
-- Q1
+I[b, 2]
+I[a, 3]
+I[c, 1]

-- Q2
+I[3, 1]
+I[2, 1]
+I[1, 1]
```

Q1 从 9 条变成 3 条，Q2 从 11 条变成 3 条，最终结果完全一样。

原因在 `TRT/operators/aggregate/MiniBatchGroupAggFunction.java:163` 起的 `finishBundle`：一批数据里，同一个 key 的所有输入先全部累加完（`:205-209`），再只比较一次"批前的结果"和"批后的结果"（`:203`、`:214`），只发一次消息（`:227-253`）。中间状态 `a=1`、`a=2` 根本没有发出去。

另外两个观察：

- 输出顺序是 `b, a, c`，不是输入顺序。一批数据放在一个 `HashMap` 里（`TRT/operators/bundle/AbstractMapBundleOperator.java:84`），按 key 遍历。**不要依赖 MiniBatch 下不同 key 之间的输出顺序。**
- 这个实验的输入只有 6 条，一批就处理完了，所以每个 key 只发了一次 `+I`。真实场景下数据会跨多个批次，仍然会有 `-U` / `+U`，只是数量少很多。**"减少到三分之一"只是这个小例子的结果，不能推广。**

---

## 五、Flink 2.3 的新变化：ON CONFLICT

### 5.1 实测：升级后可能报错的 INSERT

把 Q1 写进一张主键是 `cnt` 的表（这个主键显然选错了，几个不同的 word 可能有相同的 cnt）：

```sql
CREATE TABLE bh_pkcnt_sink (word STRING, cnt BIGINT, PRIMARY KEY (cnt) NOT ENFORCED)
  WITH ('connector' = 'blackhole');
INSERT INTO bh_pkcnt_sink SELECT word, COUNT(*) AS cnt FROM words GROUP BY word;
```

Flink 2.3 在**生成计划时**直接报错（本机实际输出）：

```text
ValidationException: The query has an upsert key that differs from the primary key of the sink table 'default_catalog.default_database.bh_pkcnt_sink'. Primary key: [cnt], upsert key: [word]. This can lead to non-deterministic results when multiple records with different upsert keys map to the same primary key. Please specify an ON CONFLICT clause to define how conflicts should be handled: ON CONFLICT DO DEDUPLICATE (update to the latest record, state intensive, since we need to keep the entire history), or ON CONFLICT DO ERROR (fail on conflict), or ON CONFLICT DO NOTHING (keep first record).
```

报错的位置在 `PRG/FlinkChangelogModeInferenceProgram.scala:1142-1155`，由配置项 `table.exec.sink.require-on-conflict` 控制，**默认是 `true`**（`flink-table/flink-table-api-java/src/main/java/org/apache/flink/table/api/config/ExecutionConfigOptions.java:261-264`）。

这是 Flink 2.3 的改动，release notes 里记录为 FLINK-38926（FLIP-558）（`docs/content/release-notes/flink-2.3.md:87-103`）：以前在这种情况下，Flink 会自动加一个 `SinkUpsertMaterializer`，需要保存完整的历史记录，状态可能会膨胀；2.3 改为默认报错，要求用户明确选择冲突策略。

**如果你从 2.2 或更早的版本升级，原来能跑的 INSERT 可能会在这里报错。** 有两种处理方式：

1. 检查主键是否选对了。大多数情况下，upsert key 和主键不一致，说明表设计或 SQL 有问题（文档里举的例子是漏写了 `GROUP BY`，见 `docs/content/docs/sql/reference/dml/insert.md:321`）；
2. 确认确实需要，就加上 `ON CONFLICT` 子句，或者把 `table.exec.sink.require-on-conflict` 设为 `false`，恢复旧行为（配置说明里提醒，这可能导致结果不确定）。

### 5.2 加上 ON CONFLICT 之后

在同一条 INSERT 后面加上 `ON CONFLICT DO DEDUPLICATE`，计划生成成功（本机实际输出）：

```text
Sink(table=[default_catalog.default_database.bh_pkcnt_sink], fields=[word, cnt], upsertMaterialize=[true], conflictStrategy=[DEDUPLICATE], changelogMode=[NONE])
+- GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UB,UA])
```

两个变化：

- Sink 上多了 `upsertMaterialize=[true]`：在 Sink 前面做物化，处理主键冲突；
- **聚合的输出从 `[I,UA]` 变回了 `[I,UB,UA]`**：这就是 2.6 节表格第三行说的"回退到 `BEFORE_AND_AFTER`"。主键和 upsert key 对不上，Sink 光靠 `+U` 没法知道哪条旧记录要被替换，需要 `-U` 来撤回。

> `DO ERROR` 和 `DO NOTHING` 两种策略要求 Source 有 Watermark（`:1109-1115`，`validateSourcesHaveWatermarks`）。release notes 里还提到了基于 Watermark 的压缩（`table.exec.sink.upserts.compaction-mode`）。这两部分本篇没有做实验，就不展开了。

---

## 六、自己动手

示例代码在 `flink-notes/demos`：

```bash
./run.sh study.sql.SqlDemo changelog
```

```bash
./run.sh study.sql.SqlDemo changelog minibatch
```

```bash
./run.sh study.sql.SqlDemo sinks
```

```bash
JAVA_TOOL_OPTIONS=-Dcodegen.level=DEBUG ./run.sh study.sql.SqlDemo codegen q2
```

**推荐的断点**：

| 断点 | 看什么 |
|---|---|
| `FlinkChangelogModeInferenceProgram.scala:71` | 根节点是否要求 UB |
| `FlinkChangelogModeInferenceProgram.scala:623` | 聚合对上游提出的要求 |
| `FlinkChangelogModeInferenceProgram.scala:1012` | Sink 的要求，以及主键检查 |
| `GroupAggHelper.java:112` | 结果没变时直接返回 |
| `GroupAggHelper.java:141` | 发 `-D` 的时机 |

---

## 七、课后练习

1. **改一下 Sink**：在 `SqlDemo sinks` 里，把 Q2 写到一张**主键是 `num_words`** 的表，计划会怎样？报错信息里的 upsert key 是什么？
2. **TTL 的影响**：开启 `table.exec.state.ttl`，再跑 `changelog`，输出会不会变？结合 `GroupAggHelper.java:112` 解释。
3. **toChangelogStream 指定模式**：用 `toChangelogStream(table, ChangelogMode.upsert())` 代替默认的版本，Q1 还会输出 `-U` 吗？需要满足什么条件？
4. **思考**：为什么推导分成两步，先从下往上推"能产生什么"，再从上往下推"需要什么"？如果只从下往上推，会有什么问题？

---

## 写在最后

这一讲的核心可以用一句话概括：**一个算子会产生哪些变化，由上游决定；要不要输出撤回消息，由下游决定。**

回到开头：同一个 `GROUP BY`，写到 print 表输出 `-U` 和 `+U`，写到 blackhole 表只输出 `+U`，因为 print 要 UB，blackhole 不要。

**实际使用时的建议**：
- 想知道一条 SQL 实际输出什么，`EXPLAIN` 完整的 `INSERT INTO`，并加上 `CHANGELOG_MODE`；
- 看到 `COUNT_RETRACT`、`MAX_RETRACT` 这类函数，说明这个聚合在处理撤回流，要关注它的状态大小；
- 从旧版本升级到 2.3，先检查一遍所有写入主键表的 INSERT，看 upsert key 和主键是否一致。

到这里，第五讲（Table / SQL）就结束了。


**留一个问题**：你遇到过"结果表里的数据和预期不一样，最后发现是撤回没处理好"的情况吗？
:::
