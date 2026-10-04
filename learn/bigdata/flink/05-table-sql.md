---
title: "Flink 源码导读 05：Table / SQL —— 一条 SQL 如何变成 Transformation"
description: "Table/SQL 从 SQL 文本经 Calcite 解析优化，变成 ExecNode 和 Transformation 的过程。"
bigdata: "flink"
---

# Flink 源码导读 05：Table / SQL —— 一条 SQL 如何变成 Transformation

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

::: v-pre
> 版本：Flink **2.3.0**（本地分支 `study-2.3.0`）。文中 `文件:行号` 均按该版本核对过。
> 路径缩写：`API` = `flink-table/flink-table-api-java/src/main/java/org/apache/flink/table`，`PLJ` = `flink-table/flink-table-planner/src/main/java/org/apache/flink/table/planner`，`PLS` = `flink-table/flink-table-planner/src/main/scala/org/apache/flink/table/planner`，`TRT` = `flink-table/flink-table-runtime/src/main/java/org/apache/flink/table/runtime`
> 前置阅读：[01](/bigdata/flink/01-execution)（Transformation → StreamGraph）、[03](/bigdata/flink/03-state-backend)（keyed state）
> 配套示例：`demos/src/main/java/study/sql/SqlDemo.java`，第 9 节的所有输出都是在本机实测的

## 0. 本篇要回答的问题

1. 一条 SQL 从字符串到 `Transformation`，经过了哪几个阶段、哪几种"树"？
2. 优化器分哪些阶段？MiniBatch、两阶段聚合是在哪一步决定的？
3. **Changelog** 是什么？为什么同一个 `GROUP BY`，有时输出 `-U/+U`，有时只输出 `+U`？
4. 聚合算子在运行时是怎么产生 `+I / -U / +U / -D` 的？
5. 代码生成生成了什么？为什么要做代码生成？

## 1. 全景图：四种树，四个阶段

```
SQL 字符串
  │ ① 解析      CalciteParser.parseSqlList()                     SqlNode（语法树）
  │ ② 校验      FlinkPlannerImpl.validate()                      SqlNode（带类型、已解析 catalog 对象）
  │ ③ 转换      FlinkPlannerImpl.rel() → SqlToRelConverter        RelNode（逻辑关系代数，Calcite 的 Logical*）
  │             → PlannerQueryOperation / ModifyOperation
  ▼
PlannerBase.translate()
  │ ④ 优化      optimize() → FlinkStreamProgram（13 个阶段）       RelNode（FlinkLogical* → StreamPhysical*）
  │ ⑤ 生成 ExecNode  translateToExecNodeGraph()                  ExecNodeGraph（StreamExec*）
  │ ⑥ 翻译      translateToPlan() → StreamExecNode.translateToPlan()   Transformation（+ 代码生成）
  ▼
StreamExecutionEnvironment → StreamGraph → ...（01 篇）
```

| 树 | 类 | 谁来操作 |
|---|---|---|
| **SqlNode** | Calcite `SqlNode` | 解析器、校验器 |
| **逻辑 RelNode** | `LogicalAggregate` → `FlinkLogicalAggregate` | 基于规则的优化（HEP）、基于代价的优化（Volcano） |
| **物理 RelNode** | `StreamPhysicalGroupAggregate` | 物理优化、changelog 推导 |
| **ExecNode** | `StreamExecGroupAggregate` | 翻译成 Transformation，可以序列化成 JSON（Compiled Plan） |

**为什么物理 RelNode 之后还要多一层 ExecNode？** RelNode 和 Calcite 深度绑定，不可序列化，而且会随 Calcite 版本变化。ExecNode 是 Flink 自己定义的、**可以序列化成 JSON** 的执行计划（带版本号，`@ExecNodeMetadata`），支持 `COMPILE PLAN` / `EXECUTE PLAN`。这样升级 Flink 版本时，已有作业的拓扑和状态布局保持不变，可以从 savepoint 恢复（FLIP-190）。

## 2. 阶段 ①②③：解析、校验、转换

### 2.1 入口

`TableEnvironmentImpl.executeSql()`（`API/api/internal/TableEnvironmentImpl.java:940`）：

```java
List<Operation> operations = getParser().parse(statement);   // :941
...
return executeInternal(operations.get(0));                    // :1309
```

`ParserImpl.parse()`（`PLJ/delegation/ParserImpl.java:91`）：

```java
Optional<Operation> command = EXTENDED_PARSER.parse(statement);   // SET/RESET/ADD JAR 等 Flink 扩展命令用正则解析
...
SqlNodeList sqlNodeList = parser.parseSqlList(statement);          // ① Calcite 解析：CalciteParser.java:73
return SqlNodeToOperationConversion.convert(planner, catalogManager, parsed.get(0));
```

Flink 的 SQL 方言（`CREATE TABLE ... WITH (...)`、`WATERMARK FOR` 等）是用 JavaCC 模板扩展 Calcite 语法生成的，在 `flink-table/flink-sql-parser` 模块中。

### 2.2 校验与转换

`SqlNodeToOperationConversion.convert()`（`PLJ/operations/SqlNodeToOperationConversion.java:172`）：

```java
final SqlNode validated = flinkPlanner.validate(sqlNode);                          // ② :175
Optional<Operation> operation = SqlNodeConverters.convertSqlNode(validated, context); // :186
```

- **校验**：`FlinkPlannerImpl.validate()`（`PLS/calcite/FlinkPlannerImpl.scala:117`）→ `FlinkCalciteSqlValidator.validate()`（`:201`）。它会解析表名（查 catalog），推导每个表达式的类型，检查函数调用是否合法。"Column 'xxx' not found" 这类错误就是在这里报出来的
- **转换成 RelNode**：查询语句最终走到 `SqlNodeConvertUtils.toQueryOperation()`（`PLJ/operations/converters/SqlNodeConvertUtils.java:58`）→ `context.toRelRoot(validated)` → `FlinkPlannerImpl.rel()`（`FlinkPlannerImpl.scala:209`）→ `sqlToRelConverter.convertQuery(...)`（`:235`）。得到的 RelNode 被包装进 `PlannerQueryOperation`

**DDL 不会走到优化器**：`CREATE TABLE` 被转换成 `CreateTableOperation`，由 `executeInternal()` 直接写入 catalog，不产生任何 Transformation。

### 2.3 从 Operation 到 Transformation

`INSERT INTO` 或 `collect()` 这类需要执行的语句，会被包装成 `ModifyOperation`，走到 `TableEnvironmentImpl.executeInternal(List<ModifyOperation>)`（`:1052`）→ `translate()`（`:1087` → `:1508`）→ **`planner.translate(modifyOperations)`** → 最后 `execEnv.executeAsync(pipeline)`（`:1242`），和 01 篇 DataStream 作业的提交路径汇合。

## 3. `PlannerBase.translate()`

`PLS/delegation/PlannerBase.scala:175`，这 4 行就是整个 planner 的骨架：

```scala
val relNodes = modifyOperations.asScala.map(translateToRel)               // Operation → RelNode（:248），加上 Sink 节点
val optimizedRelNodes = optimize(relNodes)                               // ④ 优化
val execGraph = translateToExecNodeGraph(optimizedRelNodes, isCompiled = false)   // ⑤ :412
val transformations = translateToPlan(execGraph)                         // ⑥
```

- `translateToExecNodeGraph()`：`new ExecNodeGraphGenerator()`（`:425`）→ `generate()`（`PLJ/plan/nodes/exec/ExecNodeGraphGenerator.java:57`）自底向上，对每个物理 RelNode 调用 `rel.translateToExecNode()`（`:80`）。它用 `visitedRels`（`:49`，一个 `IdentityHashMap`）保证 **DAG 中被多个下游共享的节点只生成一次**
- `translateToPlan()`：`StreamPlanner.scala:78` 对每个根节点调用 `StreamExecNode.translateToPlan(planner)`（`:82`）→ `ExecNodeBase.translateToPlan()`（`PLJ/plan/nodes/exec/ExecNodeBase.java:180`）：已经翻译过就直接返回缓存的 Transformation（`:181`），否则调用子类的 `translateToPlanInternal()`（`:183`）。每个节点在这个方法里先翻译自己的输入，再创建自己的 Transformation

---

## 4. 阶段 ④：优化器

### 4.1 以 Sink 为单位切块

`StreamCommonSubGraphBasedOptimizer.doOptimize()`（`PLS/plan/optimize/StreamCommonSubGraphBasedOptimizer.scala:111`）先用 `RelNodeBlockPlanBuilder.buildRelNodeBlockPlan()`（`:114`）把 DAG 切成若干个 **RelNodeBlock**（在多个 sink 共享的节点处切开），再对每个块调用 `optimizeTree()`（`:191`），使用 `FlinkStreamProgram.buildProgram(tableConfig)`（`:201`）。

切块的意义：`STATEMENT SET` 中的多条 `INSERT` 共享同一个 source 时，共享的部分只会被优化一次、执行一次。

### 4.2 `FlinkStreamProgram`：13 个阶段

`PLS/plan/optimize/program/FlinkStreamProgram.scala:32-44`，按顺序执行：

| 阶段 | 做什么 | 优化方式 |
|---|---|---|
| `subquery_rewrite` | 子查询改写成 join | HEP |
| `temporal_join_rewrite` | 时态表 join 改写 | HEP |
| `decorrelate` | 消除关联子查询 | — |
| `default_rewrite` | 常量折叠、表达式化简等 | HEP |
| `predicate_pushdown` | **谓词下推**，包括下推到 source（`SupportsFilterPushDown`） | HEP |
| `join_reorder` | join 重排（默认关闭） | — |
| `multi_join` | 多路 join 合并 | — |
| `project_rewrite` | **投影下推**，列裁剪下推到 source | HEP |
| **`logical`** | Calcite 逻辑节点 → `FlinkLogical*` | **Volcano**（基于代价） |
| `logical_rewrite` | 逻辑层的改写 | HEP |
| `time_indicator` | 处理时间属性（rowtime/proctime），`FlinkRelTimeIndicatorProgram` | — |
| **`physical`** | `FlinkLogical*` → `StreamPhysical*`，选择物理实现 | **Volcano** |
| **`physical_rewrite`** | **changelog 推导、MiniBatch、两阶段聚合**等 | HEP |

两种优化器：
- **HEP**（启发式）：按顺序把规则应用到树上，直到不再有变化。适合"改写后一定更好"的规则
- **Volcano**（基于代价）：枚举多种等价方案，用代价模型挑出最好的。适合"有多种实现，需要比较"的场景（比如逻辑节点 → 物理节点）

规则集定义在 `PLS/plan/rules/FlinkStreamRuleSets.scala`。例如 `PHYSICAL_OPT_RULES`（`:452`）中的 `StreamPhysicalGroupAggregateRule`（`:480`）负责把 `FlinkLogicalAggregate` 转换成 `StreamPhysicalGroupAggregate`。

### 4.3 `physical_rewrite` 阶段内部的顺序

`FlinkStreamProgram.scala:303` 起：

```
1. FlinkMarkChangelogNormalizeProgram          标记可以复用的 ChangelogNormalize
2. watermark transpose
3. ★ FlinkChangelogModeInferenceProgram       changelog 推导（第 5 节）
4. FlinkMiniBatchIntervalTraitInitProgram      MiniBatch 间隔初始化
5. MINI_BATCH_RULES                            插入 MiniBatchAssigner（MiniBatchIntervalInferRule，FlinkStreamRuleSets.scala:538）
6. 重复变更推导（FlinkDuplicateChangesTraitInitProgram）
7. PHYSICAL_REWRITE                            ★ TwoStageOptimizedAggregateRule（:549）、IncrementalAggregateRule（:551）等
```

**两阶段聚合必须在 changelog 推导之后**，因为它要根据输入是否 insert-only 来决定能不能拆分（6.4 节）。

---

## 5. ★ Changelog：流上的动态表

### 5.1 概念

流上的 SQL 查询是在一张**不断变化的表**（动态表）上持续计算的。结果表也在变化，这些变化被编码成带 `RowKind` 的记录（`flink-core/src/main/java/org/apache/flink/types/RowKind.java`）：

| RowKind | 缩写 | 含义 |
|---|---|---|
| `INSERT` | `+I` | 插入一行（`:31`） |
| `UPDATE_BEFORE` | `-U` | 撤回一行的旧值（`:40`） |
| `UPDATE_AFTER` | `+U` | 这一行的新值 |
| `DELETE` | `-D` | 删除一行 |

一个节点会输出哪几种 RowKind，就是它的 **ChangelogMode**（`flink-table/flink-table-common/.../connector/ChangelogMode.java`）。explain 中的 `changelogMode=[I,UB,UA,D]` 就是这个意思。

**两种更新模式**：
- **Retract**：更新要发 `-U` + `+U` 两条。下游不需要知道主键，拿 `-U` 去撤回旧值即可
- **Upsert**：更新只发 `+U`。下游必须**按主键覆盖**，所以要求下游知道主键。数据量减半

### 5.2 `FlinkChangelogModeInferenceProgram`

`PLS/plan/optimize/program/FlinkChangelogModeInferenceProgram.scala:54`，`optimize()`（`:56`）分三步：

**第 1 步：`SatisfyModifyKindSetTraitVisitor`（`:131`）自底向上**，推导每个节点**会产生**哪些类型的变更：
- Source：取决于 connector 声明的 `getChangelogMode()`。Kafka 是 insert-only，CDC 源是全部类型
- Group Aggregate（`:194`）：即使输入是 insert-only，也会产生 update（同一个 key 的聚合值会变）

**第 2 步：`SatisfyUpdateKindTraitVisitor`（`:558`）自顶向下**，根据**下游的需求**决定更新用哪种形式（`:70-80`）：

```scala
if (context.isUpdateBeforeRequired) {
  Seq(UpdateKindTrait.BEFORE_AND_AFTER)                              // 下游要求 -U
} else {
  // update_before is not required, and input contains updates
  // try ONLY_UPDATE_AFTER first, and then BEFORE_AND_AFTER
  Seq(UpdateKindTrait.ONLY_UPDATE_AFTER, UpdateKindTrait.BEFORE_AND_AFTER)   // 优先尝试 upsert
}
```

需求从 sink 开始往上传（`case sink: StreamPhysicalSink`，`:586`）。**sink 需不需要 `-U`，由它的 connector 实现 `DynamicTableSink.getChangelogMode(requestedMode)` 决定**，而不是由主键决定：
- `blackhole` 会主动去掉 `UPDATE_BEFORE`（`flink-table/flink-table-api-java-bridge/.../connector/blackhole/table/BlackHoleTableSinkFactory.java:71-78`）→ 上游只发 `+U`。upsert-kafka、带主键的 JDBC 这类 upsert sink 也是这样声明的
- `print` 原样返回请求的模式（`.../connector/print/table/PrintTableSinkFactory.java:121-122`），`toChangelogStream()` 也要求完整的 changelog → 需要 `-U`
- **下游是另一个聚合** → 需要 `-U`，因为下游要用旧值去撤回自己的累加器（第 9.1 节的 Q2）

第 9.4 节用 `print` 和 `blackhole` 两个 sink 做了对比实验。

**第 3 步：`SatisfyDeleteKindTraitVisitor`（`:1198`）**：决定 delete 用"只带主键"（`DELETE_BY_KEY`）还是"带完整行"（`FULL_DELETE`）的形式。只带主键可以减少传输量，也让上游不必为了发出完整的删除行而额外保存状态。

> 💡 这就是"**同一个 GROUP BY，有时输出 -U，有时不输出**"的原因：**是否输出 UPDATE_BEFORE 是由下游需求决定的，不是由聚合本身决定的。** 第 9.1 和 9.2 节会看到同一条 Q1，在 explain（没有 sink）时是 `[I,UA]`，接上 `toChangelogStream()` 后实际输出了 `-U`。

---

## 6. Group Aggregate：从物理节点到运行时

### 6.1 `StreamExecGroupAggregate.translateToPlanInternal()`

`PLJ/plan/nodes/exec/stream/StreamExecGroupAggregate.java:171`

1. **状态 TTL**：`stateRetentionTime`（`:174`）。有 group key 但没有配置 TTL 时会打印警告（`:176`），提醒状态可能无限增长（`table.exec.state.ttl`，`API/api/config/ExecutionConfigOptions.java:50`）
2. **代码生成**：`new AggsHandlerCodeGenerator(...)`（`:189`）→ `generator.generateAggsHandler("GroupAggsHandler", aggInfoList)`（`:216`）。此时生成的是 Java **源码字符串**，包装在 `GeneratedAggsHandleFunction` 里（第 7 节）
3. **选择运行时实现**（`:231-270`）：

   | 条件 | 函数 | 算子 |
   |---|---|---|
   | MiniBatch 开启（`:235`） | `MiniBatchGroupAggFunction` | `KeyedMapBundleOperator`（`:246`） |
   | 异步状态开启（`:249`，03 篇第 7 节） | `AsyncStateGroupAggFunction` | `AsyncKeyedProcessOperator` |
   | 默认 | `GroupAggFunction` | **`KeyedProcessOperator`** |

   也就是说，SQL 的 Group Aggregate 在运行时就是一个 **`KeyedProcessFunction`**，和你用 DataStream API 手写的没有本质区别
4. `ExecNodeUtil.createOneInputTransformation(...)` 创建 Transformation，并设置 keyBy 的 KeySelector（按 group key）

### 6.2 ★ 运行时：changelog 是怎么产生的

`GroupAggFunction.processElement()`（`TRT/operators/aggregate/GroupAggFunction.java:85`）→ `GroupAggHelper.processElement()`（`TRT/operators/aggregate/utils/GroupAggHelper.java:68`）：

```java
if (null == accumulators) {
    if (isRetractMsg(input)) { return; }          // 第一条就是撤回消息（比如状态已过期），直接忽略
    firstRow = true;
    accumulators = function.createAccumulators();
}
function.setAccumulators(accumulators);
RowData prevAggValue = function.getValue();        // 旧的聚合结果

if (isAccumulateMsg(input)) {                      // +I / +U：累加
    function.accumulate(input);
} else {                                           // -U / -D：撤回
    function.retract(input);
}
RowData newAggValue = function.getValue();         // 新的聚合结果

if (!recordCounter.recordCountIsZero(accumulators)) {
    updateAccumulatorsState(accumulators);         // 写回 ValueState
    if (!firstRow) {
        if (!ttlConfig.isEnabled() && equaliser.equals(prevAggValue, newAggValue)) {
            return;                                // ★ 结果没变：什么都不发
        }
        if (generateUpdateBefore) {                // ★ 由 5.2 节的推导结果决定
            out.collect(prevAggValue as UPDATE_BEFORE);    // -U
        }
        out.collect(newAggValue as UPDATE_AFTER);          // +U
    } else {
        out.collect(newAggValue as INSERT);                // +I
    }
} else {                                           // 这个 key 下的记录全部被撤回了
    if (!firstRow) {
        out.collect(prevAggValue as DELETE);               // -D
    }
    clearAccumulatorsState();
}
```

几个值得注意的细节：
- **`generateUpdateBefore`** 在 ExecNode 构造时传进来（`StreamExecGroupAggregate.java:107`），值就是 changelog 推导的结果
- **结果没变就不输出**：比如 `MAX(x)` 来了一个更小的值。这是一个重要的优化，但开启 TTL 时不能这么做（注释解释得很清楚：必须持续发消息，避免下游状态过早过期）
- **撤回消息先于任何累加到达时直接丢弃**：这就是配置了状态 TTL 后，结果可能"不准"的原因之一：状态过期后，迟到的撤回消息找不到对应的累加器

### 6.3 MiniBatch

每条记录都要读一次状态、写一次状态，在 RocksDB 上开销很大（03 篇 5.1 节：每次都要序列化）。MiniBatch 的思路是：**攒一批再处理，每个 key 在一批里只读写一次状态**。

- **`MiniBatchAssigner`**：`ProcTimeMiniBatchAssignerOperator`（`TRT/operators/wmassigners/ProcTimeMiniBatchAssignerOperator.java:41`）按处理时间，每隔 `table.exec.mini-batch.allow-latency` 往下游发一个 **watermark** 作为"批次结束"的标记（`:71-75`）
- **`KeyedMapBundleOperator`**：`AbstractMapBundleOperator`（`TRT/operators/bundle/AbstractMapBundleOperator.java`）
  - `processElement()`（`:111`）：放进内存中的 `bundle` Map（`:121`，key → 这个 key 的记录列表），**不访问状态**
  - 两种情况触发处理：
    - 攒够了 `table.exec.mini-batch.size` 条：`CountBundleTrigger.onElement()`（`TRT/operators/bundle/trigger/CountBundleTrigger.java:46`，由 `MinibatchUtil.createMiniBatchTrigger()` 创建，`PLJ/plan/utils/MinibatchUtil.java:47`）
    - 收到 MiniBatchAssigner 发来的 watermark：`processWatermark()`（`:141`）
  - `finishBundle()`（`:131`）→ `MiniBatchGroupAggFunction.finishBundle()`（`TRT/operators/aggregate/MiniBatchGroupAggFunction.java:163`）：对每个 key 读一次状态，把这个 key 的所有记录都累加进去，写回一次状态，**只输出一次结果**

配置（`ExecutionConfigOptions.java`）：`table.exec.mini-batch.enabled`（`:666`）、`allow-latency`（`:678`）、`size`（`:691`）。

代价：延迟最多增加 `allow-latency`；buffer 放在堆内存中，要注意 `size` 不能太大。

### 6.4 两阶段聚合（Local-Global）

**问题**：`GROUP BY` 的 key 有热点时（比如 90% 的数据都是同一个城市），这个 key 所在的 subtask 会成为瓶颈（04 篇 3.3 节：还会拖住所有上游）。

**思路**：先在 keyBy **之前**做一次局部聚合（Local），把同一个 key 的多条记录合并成一条部分结果，再 shuffle 给全局聚合（Global）做合并。热点 key 在 shuffle 之前就已经被合并掉了。

```
Source → MiniBatchAssigner → LocalGroupAggregate ─hash─► GlobalGroupAggregate
                             （无状态，只在批内合并）        （有状态，merge 部分结果）
```

**生效条件**：`TwoStageOptimizedAggregateRule.matches()`（`PLJ/plan/rules/physical/stream/TwoStageOptimizedAggregateRule.java:86`）

```java
return isMiniBatchEnabled                                     // ① 必须开启 MiniBatch！
        && isTwoPhaseEnabled                                  // ② table.optimizer.agg-phase-strategy 不是 ONE_PHASE（默认 AUTO）
        && matchesTwoStage(call.rel(0), call.rel(2));
// matchesTwoStage()（:96）：
//   ③ 所有聚合函数都支持 merge（doAllSupportPartialMerge，:115）
//   ④ 输入还没有按 group key 分区（:116）：已经分好区了，再做一次 local 也没有意义
```

**① 是最容易被忽视的**：Local 聚合是**没有状态**的，它只能在一个 MiniBatch 的**内存 bundle** 里做合并（`MiniBatchLocalGroupAggFunction.finishBundle()`，`TRT/operators/aggregate/MiniBatchLocalGroupAggFunction.java:84`）。没有 MiniBatch，就没有"批"可以合并，所以两阶段聚合必须依赖 MiniBatch。Global 端是 `MiniBatchGlobalGroupAggFunction.finishBundle()`（`MiniBatchGlobalGroupAggFunction.java:161`），它调用的是 `merge()` 而不是 `accumulate()`。

**COUNT DISTINCT 的热点**用两阶段是解决不了的（distinct 需要看到所有值才能去重），需要另一个优化：`table.optimizer.distinct-agg.split.enabled`（`API/api/config/OptimizerConfigOptions.java:68`），把它改写成先按 `(key, hash(distinct_key) % N)` 聚合、再按 key 聚合的两层结构。

---

## 7. 代码生成

### 7.1 为什么要代码生成？

SQL 的算子是**通用**的，但每条 SQL 的表达式、类型、聚合函数都不同。如果用解释执行（遍历表达式树、按类型分支判断、调用虚函数），每条记录都要付出大量的额外开销。代码生成为**每条 SQL 定制**一份专用代码，由 JIT 编译成机器码，执行效率接近手写代码。

### 7.2 生成与编译

- **生成**：planner 在翻译阶段拼出 Java 源码字符串。`AggsHandlerCodeGenerator.generateAggsHandler()`（`PLS/codegen/agg/AggsHandlerCodeGenerator.scala:347`）的代码模板从 `:366` 开始（`public final class $functionName implements $AGGS_HANDLER_FUNCTION`），最后包装成 `new GeneratedAggsHandleFunction(...)`（`:445`）。源码随作业一起提交，**planner 本身不需要出现在 TM 上**
- **编译**：在 **TM 上**，算子 open 时调用 `GeneratedClass.newInstance()`（`TRT/generated/GeneratedClass.java:66`）→ `compile()`（`:92`）→ `CompileUtils.compile()`（`TRT/generated/CompileUtils.java:86`）
  - 用 **Janino**（一个轻量的内存 Java 编译器）编译：`new SimpleCompiler()`（`:101`）→ `compiler.cook(code)`（`:104`）
  - 结果会被缓存：`COMPILED_CLASS_CACHE`（`:52`），key 是 classloader + 类名 + 代码，同一个 TM 上的多个并行实例只编译一次
  - 编译失败时会把**带行号的源码**打印出来（`:106`、`addLineNumber()` `:122`），方便排查问题
  - DEBUG 日志会打印所有生成的代码（`CODE_LOG.debug("Compiling: {} \n\n Code:\n{}", ...)`，`:100`），第 9.3 节就是用这个办法看到的

生成的代码不只有聚合函数，还包括：投影和过滤（`CalcCodeGenerator`，`StreamExecCalc`）、key 的抽取（`KeyProjection`）、结果比较（`RecordEqualiser`）、类型转换、Source 的数据构造等。

---

## 8. 断点清单

用 IDEA 运行 `SqlDemo`，参数 `changelog`。

| # | 位置 | 看什么 |
|---|---|---|
| 1 | `ParserImpl.java:91` | SQL 字符串 |
| 2 | `SqlNodeToOperationConversion.java:175` | 校验前后的 SqlNode |
| 3 | `FlinkPlannerImpl.scala:235` | 转换出来的 RelNode（调用 `RelOptUtil.toString(rel)` 可以打印） |
| 4 | `PlannerBase.scala:183` | 优化前后的 RelNode |
| 5 | `FlinkChangelogModeInferenceProgram.scala:70` | `isUpdateBeforeRequired` 的值 |
| 6 | `TwoStageOptimizedAggregateRule.java:93`（`explain` 模式） | 三个条件分别是多少 |
| 7 | `ExecNodeGraphGenerator.java:80` | 物理节点 → ExecNode |
| 8 | `StreamExecGroupAggregate.java:231` | 选择了哪种运行时实现 |
| 9 | `CompileUtils.java:101`（TM 端） | 生成的源码 |
| 10 | `GroupAggHelper.java:88` | 每条记录前后的聚合值、RowKind |

---

## 9. 实验

示例：`SqlDemo.java`。两条查询：

```sql
-- Q1：单层聚合
SELECT word, COUNT(*) AS cnt FROM words GROUP BY word
-- Q2：嵌套聚合 —— 统计"出现了 cnt 次的单词有几个"
SELECT cnt, COUNT(*) AS num_words FROM (Q1) GROUP BY cnt
```

示例工程直接依赖 `flink-table-planner_2.12`，而不是生产环境中常用的 `flink-table-planner-loader`。loader 会把 planner 隐藏在独立的 classloader 里，IDE 调试时就无法在 planner 源码上打断点。

### 9.1 执行计划：`explain`

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.sql.SqlDemo explain
```

实测的 **Optimized Physical Plan**（节选）：

**Q1，默认配置**：

```
GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UA])
+- Exchange(distribution=[hash[word]], changelogMode=[I])
   +- TableSourceScan(table=[[... words]], fields=[word], changelogMode=[I])
```

**Q1，MiniBatch（ONE_PHASE）**：多了 `MiniBatchAssigner`

```
GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UA])
+- Exchange(distribution=[hash[word]], changelogMode=[I])
   +- MiniBatchAssigner(interval=[1000ms], mode=[ProcTime], changelogMode=[I])
      +- TableSourceScan(...)
```

**Q1，MiniBatch + 两阶段（AUTO，默认值）**：拆成了 Local 和 Global

```
GlobalGroupAggregate(groupBy=[word], select=[word, COUNT(count1$0) AS cnt], changelogMode=[I,UA])
+- Exchange(distribution=[hash[word]], changelogMode=[I])
   +- LocalGroupAggregate(groupBy=[word], select=[word, COUNT(*) AS count1$0], changelogMode=[I])
      +- MiniBatchAssigner(interval=[1000ms], mode=[ProcTime], changelogMode=[I])
         +- TableSourceScan(...)
```

注意：Local 阶段输出的是 `count1$0`（部分计数），Global 阶段对它做 `COUNT(count1$0)`，实际语义是把部分计数加起来（merge）。

**Q2，嵌套聚合**：

```
GroupAggregate(groupBy=[cnt], select=[cnt, COUNT_RETRACT(*) AS num_words], changelogMode=[I,UA,D])
+- Exchange(distribution=[hash[cnt]], changelogMode=[I,UB,UA])
   +- Calc(select=[cnt], changelogMode=[I,UB,UA])
      +- GroupAggregate(groupBy=[word], select=[word, COUNT(*) AS cnt], changelogMode=[I,UB,UA])
         +- Exchange(distribution=[hash[word]], changelogMode=[I])
            +- TableSourceScan(...)
```

解读：
1. **同样是内层聚合，Q1 中是 `[I,UA]`，Q2 中是 `[I,UB,UA]`**：因为 Q2 的外层聚合要撤回旧值，需求沿着树往上传，内层聚合就必须输出 UPDATE_BEFORE（5.2 节第 2 步）
2. 外层聚合的函数从 `COUNT` 变成了 **`COUNT_RETRACT`**：输入中有撤回消息，必须使用支持 `retract()` 的聚合函数实现
3. 外层聚合的输出多了 **`D`**：当某个 `cnt` 值下已经没有单词了（比如单词 a 从出现 1 次变成 2 次），这个分组就要被删除

### 9.2 真实 changelog：`changelog`

输入固定为 6 条：`a a b a c b`，并行度 1。

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.sql.SqlDemo changelog
```

实测输出：

| 输入 | Q1 输出 | Q2 输出 |
|---|---|---|
| a | `+I[a, 1]` | `+I[1, 1]` |
| a | `-U[a, 1]` `+U[a, 2]` | `-D[1, 1]` `+I[2, 1]` |
| b | `+I[b, 1]` | `+I[1, 1]` |
| a | `-U[a, 2]` `+U[a, 3]` | `-D[2, 1]` `+I[3, 1]` |
| c | `+I[c, 1]` | `-U[1, 1]` `+U[1, 2]` |
| b | `-U[b, 1]` `+U[b, 2]` | `-U[1, 2]` `+U[1, 1]` `+I[2, 1]` |

Q2 的最终结果是 `{1: 1, 2: 1, 3: 1}`（出现 1 次的是 c，2 次的是 b，3 次的是 a），完全正确。但中间过程产生了大量的撤回和更新：**流式 SQL 的结果是"最终一致"的**。

同时注意：**Q1 这里输出了 `-U`，而 9.1 节 explain 中 Q1 是 `[I,UA]`**。区别在于这次接的是 `toChangelogStream()`，它要求完整的 changelog，于是 `isUpdateBeforeRequired` 变为 true（5.2 节的结论）。

> **勘误（2026-10-02）**：上面"`isUpdateBeforeRequired` 变为 true"的说法不准确。`isUpdateBeforeRequired` 是优化块（RelNodeBlock）之间传递要求的标志（`StreamCommonSubGraphBasedOptimizer.scala:297`）。`toChangelogStream()` 的根节点是一个 Sink，`-U` 的要求来自这个 Sink：不指定 ChangelogMode 时，`ExternalDynamicSink.getChangelogMode` 原样返回 `requestedMode`（`ExternalDynamicSink.java:68-73`），而 `ModifyKindSet.toDefaultChangelogMode()` 只要有 UPDATE 就带上 UPDATE_BEFORE（`ModifyKindSet.java:109-115`）；之后由 `inferSinkRequiredTraits`（`FlinkChangelogModeInferenceProgram.scala:1005-1027`）推导出 `BEFORE_AND_AFTER`。详见 `content-plan/17-第5讲下-公众号定稿.md` 2.4、2.5 节。

**开启 MiniBatch** 后：

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && ./run.sh study.sql.SqlDemo changelog minibatch
```

```
Q1: +I[b, 2]  +I[a, 3]  +I[c, 1]
Q2: +I[3, 1]  +I[2, 1]  +I[1, 1]
```

| | 不开 MiniBatch | 开启 MiniBatch |
|---|---|---|
| Q1 输出条数 | 9 | **3** |
| Q2 输出条数 | 11 | **3** |

6 条输入都落在了同一个批次里：每个 key 只读写一次状态，只输出一次最终结果，**中间的撤回和更新全部被合并掉了**。输出减少，下游的负担（以及外部存储的写入压力）也跟着减少。

> 日志中会出现几条 `WARN CollectResultFetcher - Failed to get job status ... IllegalStateException: MiniCluster is not yet running`：有界作业跑完后，本地 MiniCluster 自动关闭，collect 客户端再去查询作业状态时就会打出这条日志。数据已经全部收到，可以忽略。

### 9.3 生成的代码：`codegen`

```bash
cd /Users/miaoyongbin/Documents/work/opensource-codes/flink-notes/demos && JAVA_TOOL_OPTIONS=-Dcodegen.level=DEBUG ./run.sh study.sql.SqlDemo codegen
```

（`log4j2.properties` 中 `CompileUtils` 的日志级别由系统属性 `codegen.level` 控制，默认 WARN。）

Q1 这个作业在 TM 上编译了 4 个类：`GroupAggsHandler$6`（聚合逻辑）、`GroupAggValueEqualiser$0`（6.2 节中判断"结果是否变化"的比较器）、`KeyProjection$0`（从行中抽取 group key）、`StreamExecValues$2`（VALUES 数据源）。

`GroupAggsHandler$6` 的关键部分（实测输出，节选）：

```java
public final class GroupAggsHandler$6 implements org.apache.flink.table.runtime.generated.AggsHandleFunction {
  long agg0_count1;                  // 累加器被展开成了基本类型的字段，而不是对象
  boolean agg0_count1IsNull;

  public void accumulate(org.apache.flink.table.data.RowData accInput) throws Exception {
    boolean isNull$3;
    long result$4;
    isNull$3 = agg0_count1IsNull || false;
    result$4 = -1L;
    if (!isNull$3) {
      result$4 = (long) (agg0_count1 + ((long) 1L));    // COUNT(*) 被内联成一次加法
    }
    agg0_count1 = result$4;;
    agg0_count1IsNull = isNull$3;
  }

  public void retract(org.apache.flink.table.data.RowData retractInput) throws Exception {
    throw new java.lang.RuntimeException("This function not require retract method, but the retract method is called.");
  }
  ...
}
```

观察：
- **累加器变成了基本类型字段**（`long agg0_count1`）：没有装箱，没有 `Map`，没有虚函数调用
- `COUNT(*)` 直接内联成 `agg0_count1 + 1L`
- 因为 Q1 的输入是 insert-only，`retract()` 和 `merge()` **根本不需要**，所以直接生成了一个抛异常的方法体。换成 Q2 的外层聚合（`COUNT_RETRACT`），就会生成真正的 `retract` 逻辑。你可以把 `codegen()` 中的查询换成 Q2 试试

### 9.4 sink 决定 changelog：`print` vs `blackhole`

对同一条 Q1 分别 `INSERT INTO` 两个 sink，每个 sink 都分别测试带主键（`PRIMARY KEY (word) NOT ENFORCED`）和不带主键两种情况，用 `EXPLAIN CHANGELOG_MODE` 观察 GroupAggregate 的输出：

| sink connector | 带主键 | 不带主键 |
|---|---|---|
| `print` | `[I,UB,UA]` | `[I,UB,UA]` |
| `blackhole` | `[I,UA]` | `[I,UA]` |

**主键对结果没有影响，决定性因素是 connector 的 `getChangelogMode()`**（5.2 节）。

> 🐛 **踩坑记录**：我最初写练习题时，以为"给 print sink 加上主键就会变成 upsert 模式"，实测之后发现并非如此。print 的 `getChangelogMode()` 直接返回 `requestedMode`，从不去掉 UPDATE_BEFORE。凡是涉及 changelog 的结论，最好都用 `EXPLAIN CHANGELOG_MODE` 实际验证一下。

---

## 10. 课后练习

1. **Upsert sink**：读 `BlackHoleTableSinkFactory` 和 `PrintTableSinkFactory` 的 `getChangelogMode()`，再找一个真正的 upsert connector（比如 `flink-connector-kafka` 仓库中的 upsert-kafka，或 flink-connector-jdbc），看看它是怎么利用主键来决定 changelog 模式的
2. **两阶段的条件**：在 explain 模式中，把 `COUNT(*)` 换成一个不支持 merge 的自定义聚合函数（UDAF 不实现 `merge` 方法），观察两阶段是否还会生效。对照 6.4 节的条件 ③
3. **Distinct 热点**：分别在关闭和开启 `table.optimizer.distinct-agg.split.enabled` 时，explain `SELECT day, COUNT(DISTINCT user_id) FROM t GROUP BY day`，画出两种执行计划
4. **状态 TTL 的副作用**：设置 `table.exec.state.ttl = 1 s`，在 changelog 模式中让输入之间 sleep 2 秒（改用 datagen），观察 Q2 的结果是否还正确，并用 6.2 节的"撤回消息找不到累加器"解释原因
5. **Compiled Plan**：执行 `COMPILE PLAN '/tmp/q1.json' FOR INSERT INTO ...`，打开这个 JSON，找到 `StreamExecGroupAggregate` 节点，对照 `@ExecNodeMetadata` 的版本号和字段，思考：为什么升级 Flink 时需要它？

## 11. 下一篇预告

**06：调度与容错**。`DefaultScheduler` 的 failover 流程：Task 失败 → `FailoverStrategy` 计算需要重启的 region → 取消、重新分配 slot、从最近的 Checkpoint 恢复；`AdaptiveScheduler` 如何根据可用资源自动扩缩容；以及 `AdaptiveBatchScheduler` 如何根据上游的实际数据量，动态决定下游的并行度。
:::
