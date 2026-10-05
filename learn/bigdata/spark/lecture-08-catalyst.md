---
title: "第 8 讲：Spark SQL 与 Catalyst"
description: "Spark SQL 与 Catalyst：解析、分析、优化、物理计划四个阶段，以及实际生效的优化规则。"
bigdata: "spark"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-08-catalyst-cover.webp"}]]
---

# 第 8 讲：Spark SQL 与 Catalyst

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-08-catalyst-cover.webp" alt="SQL 经四阶段变成可执行计划" width="1200" height="800" loading="eager" /><figcaption>SQL 经四阶段变成可执行计划<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`catalyst/…` = `sql/catalyst/src/main/scala/org/apache/spark/sql/catalyst/`，`execution/…` = `sql/core/src/main/scala/org/apache/spark/sql/execution/`
> 类名、配置、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；四个阶段的计划变化经过实验验证（仓库 `experiments/09-catalyst/`）。

## 0. 本讲要回答的问题

1. 有了 RDD，为什么还要 Spark SQL？
2. 一条 SQL（或一段 DataFrame 代码）是怎么一步步变成 RDD 的？
3. Catalyst 优化器的"规则"长什么样？它们是怎么被执行的？
4. 分析、优化、物理计划三个阶段，各自具体做了哪些事？
5. 怎么读懂 `explain()` 的输出？

> 从这一讲开始进入 **Spark SQL**。它是 Spark 中代码量最大、最活跃、Committer 最多的模块，也是第 1 讲学习路线里说的"冲击 Committer 的主战场"。

---

## 1. 为什么需要 Spark SQL

RDD 的问题：**Spark 看不懂你的函数**。

```scala
rdd.map(r => (r.city, r.amount)).filter(_._2 > 100)
```

对 Spark 来说，`map` 和 `filter` 里的函数都是黑盒，它不知道你只用了两列，也不知道过滤条件能不能提前，只能老老实实地按你写的顺序执行。

DataFrame / SQL 的不同：**你描述"要什么"，而不是"怎么做"**。

```sql
SELECT city, amount FROM orders WHERE amount > 100
```

Spark 能看懂整个查询的结构：只需要两列（可以只读这两列）、过滤条件可以下推到读文件那一层、可以根据表的大小选择 Join 方式……这些都是 **Catalyst 优化器**自动完成的。

> 参考论文：*Spark SQL: Relational Data Processing in Spark*（SIGMOD 2015）。

**同一个引擎**：SQL、DataFrame、Dataset 最终都会变成同一种**逻辑计划**，走同一条优化路径。区别只在入口：SQL 要先**解析**成逻辑计划，DataFrame API 直接构造逻辑计划，跳过解析这一步。

> 4.x 的代码结构：Dataset 等 API 的**接口**放在 `sql/api` 模块，经典实现在 `sql/core/.../sql/classic/` 下（例如 `classic/Dataset.scala`、`classic/SparkSession.scala`）。这样 Spark Connect 的客户端可以共用同一套接口。

---

## 2. 全景：一条 SQL 的旅程

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-08-catalyst-1.webp" alt="QueryExecution 串起 SQL 全流程" width="960" height="640" loading="lazy" /><figcaption>QueryExecution 串起 SQL 全流程<span>AI 生成配图</span></figcaption></figure>

```
SQL 文本
  │ ① 解析 Parsing        ANTLR 语法 → AstBuilder
  ▼
未解析的逻辑计划（Unresolved Logical Plan）
  │ ② 分析 Analysis       Analyzer：绑定表、列、函数，检查类型
  ▼
已分析的逻辑计划（Analyzed Logical Plan）
  │ ③ 优化 Optimization   Optimizer：一批批规则改写计划
  ▼
优化后的逻辑计划（Optimized Logical Plan）
  │ ④ 物理计划 Planning   SparkPlanner：策略把逻辑算子翻译成物理算子
  │                       + preparations：插入 Shuffle、合并代码生成阶段……
  ▼
可执行的物理计划（Executed Physical Plan）
  │ execute()
  ▼
RDD[InternalRow]   ← 后面就是第 1～7 讲的世界
```

### 2.1 总控：QueryExecution

把这些阶段串起来的是 `execution/QueryExecution.scala`。4.2.0 中的实际链路比上图更细：

```
logical → analyzed → commandExecuted → normalized → withCachedData → optimizedPlan → sparkPlan → executedPlan → toRdd
          (第 211 行)  (第 223 行)                   (第 305 行)       (第 329 行)       (第 347 行)  (第 370 行)    (第 390 行)
```

| 阶段 | 做了什么 |
|---|---|
| `analyzed` | 调用 Analyzer |
| `commandExecuted` | 对 `INSERT`、`CREATE TABLE` 这类命令，**在这里就立即执行**（所以 `spark.sql("CREATE TABLE ...")` 不需要 Action 就会生效） |
| `normalized` | 用自定义规则规范化计划，让它**更容易命中缓存** |
| `withCachedData` | 把计划里已经被 `cache()` 过的部分替换成缓存数据（第 5 讲 Dataset 缓存在这里生效） |
| `optimizedPlan` | 调用 Optimizer |
| `sparkPlan` | 调用 SparkPlanner 生成物理计划 |
| `executedPlan` | 应用 preparations 规则（插入 Exchange 等） |
| `toRdd` | `executedPlan.execute()`，得到 RDD |

**每个阶段都是惰性的**，第一次访问才计算。4.x 里它们不再是 Scala 的 `lazy val`，而是用 `LazyTry` 包装（`core/…/util/LazyTry.scala`，**4.0.0 引入**，`[SPARK-48195]`）。源码注释说明了原因：

1. Scala 的 `lazy val` 初始化抛异常后，**下次访问会再算一遍**，可能重复执行有副作用的操作；`LazyTry` 会**缓存异常**，不重试
2. Scala 的 `lazy val` 初始化时会**锁住整个外层对象**，可能导致性能问题甚至死锁

> 4.2.0 的 `QueryExecution` 里还能看到 SQL 脚本（`sqlScriptExecuted`）、事务（`withAbortTransactionOnFailure`）相关的逻辑，都是比较新的功能方向。

---

## 3. Catalyst 的地基：树和规则

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-08-catalyst-2.webp" alt="Catalyst 用树和规则改写计划" width="960" height="640" loading="lazy" /><figcaption>Catalyst 用树和规则改写计划<span>AI 生成配图</span></figcaption></figure>

### 3.1 一切皆树：TreeNode

逻辑计划（`LogicalPlan`）、表达式（`Expression`）、物理计划（`SparkPlan`）**都是 `TreeNode`**（`catalyst/trees/TreeNode.scala`）。比如 `age > 10 + 20` 是一棵表达式树：

```
GreaterThan
 ├── AttributeReference(age)
 └── Add
      ├── Literal(10)
      └── Literal(20)
```

`TreeNode` 最核心的能力是 **`transform`**：给它一个偏函数，它会遍历整棵树，**匹配上的节点就替换掉**。

```scala
def transform(rule: PartialFunction[BaseType, BaseType]): BaseType = transformDown(rule)   // TreeNode.scala:438
```

注意：**`transform` 就是 `transformDown`**，先处理父节点再处理子节点；需要先处理子节点时用 `transformUp`。

一个"常量折叠"的规则，本质上就是这么一段模式匹配：

```scala
plan.transformAllExpressions {
  case e if e.foldable => Literal.create(e.eval(), e.dataType)   // 能在编译期算出来的表达式，直接算成常量
}
```

> 这就是为什么第 1 讲的学习路线里说：**Scala 的模式匹配必须熟练**，Catalyst 里到处都是 `transform { case ... => ... }`。

**剪枝优化：TreePattern**（`[SPARK-34916]`，**3.2.0 引入**）。一条规则如果要遍历整棵树，计划很大时开销可观。现在每个节点都记录了自己子树里**包含哪些模式**（`treePatternBits`），规则可以用 `transformWithPruning(cond, ...)` 先判断"这棵子树里根本没有我关心的节点"，直接跳过。

### 3.2 规则与规则执行器

- **`Rule[TreeType]`**：一条规则，就是一个 `apply(plan): plan` 函数
- **`RuleExecutor`**（`catalyst/rules/RuleExecutor.scala`）：按**批次（Batch）**执行规则。每个批次有一个执行策略：
  - **`Once`**：只执行一轮
  - **`FixedPoint(n)`**：反复执行，**直到计划不再变化（不动点）或达到 n 轮**

```
Batch("Operator Optimizations", FixedPoint(100), 规则A, 规则B, 规则C, ...)
   第 1 轮：A → B → C   计划变了
   第 2 轮：A → B → C   计划变了
   ...
   第 k 轮：A → B → C   计划没变 → 停止
```

为什么要反复执行：一条规则的改写，可能为另一条规则创造新的机会。比如谓词下推之后，两个 Filter 挨在一起了，"合并 Filter"规则就能生效了。

最大轮数默认都是 **100**：`spark.sql.analyzer.maxIterations`、`spark.sql.optimizer.maxIterations`。

---

## 4. 实验：一条 SQL 的四次变身

```sql
SELECT p.name, sum(o.amount) AS total
FROM people p JOIN orders o ON p.id = o.user_id
WHERE p.age > 10 + 20 AND p.city = 'BJ'
GROUP BY p.name
```

`people` 表有 `id, name, age, city, bio` 五列（`bio` 不会被用到），`orders` 表有 `user_id, amount` 两列，都是 Parquet 文件。

### ① 解析后：只有结构，没有含义

```
'Aggregate ['p.name], ['p.name, 'sum('o.amount) AS total#38]
+- 'Filter (('p.age > (10 + 20)) AND ('p.city = BJ))
   +- 'Join Inner, ('p.id = 'o.user_id)
      :- 'SubqueryAlias p
      :  +- 'UnresolvedRelation [people], [], false
      +- 'SubqueryAlias o
         +- 'UnresolvedRelation [orders], [], false
```

- **单引号 `'`** 表示"未解析"：此时 Spark 只知道语法结构，**不知道 `people` 表是否存在、`age` 是什么类型、`sum` 是哪个函数**
- `10 + 20` 原样保留

### ② 分析后：绑定了含义

```
Aggregate [name#33], [name#33, sum(amount#37) AS total#38]
+- Filter ((age#34 > (10 + 20)) AND (city#35 = BJ))
   +- Join Inner, (id#32 = user_id#36)
      :- SubqueryAlias p
      :  +- SubqueryAlias people
      :     +- View (`people`, [id#32, name#33, age#34, city#35, bio#..])
      :        +- Relation [...] parquet
      ...
```

- 表找到了（`Relation ... parquet`），每一列都有了**唯一 ID**（`age#34`）。Catalyst 用 ID 而不是名字来区分列，所以两张表都有同名列也不会混淆
- 真正生效的分析规则：**`ResolveRelations`**（找表）、**`ResolveReferences`**（找列）、**`ResolveFunctions`**（找函数）
- 如果列名写错了，就是在这个阶段报错（`CheckAnalysis`）

### ③ 优化后：被改写了

```
Aggregate [name#33], [name#33, sum(amount#37) AS total#38]
+- Project [name#33, amount#37]
   +- Join Inner, (id#32 = user_id#36)
      :- Project [id#32, name#33]
      :  +- Filter (((isnotnull(age#34) AND isnotnull(city#35)) AND ((age#34 > 30) AND (city#35 = BJ))) AND isnotnull(id#32))
      :     +- Relation [...] parquet
      +- Filter isnotnull(user_id#36)
         +- Relation [...] parquet
```

逐条对照，**真正生效的优化规则一共 5 条**：

| 变化 | 规则 |
|---|---|
| `10 + 20` → `30` | **ConstantFolding**（常量折叠） |
| Filter 从 Join 上面**移到了 people 表下面** | **PushDownPredicates**（谓词下推）：先过滤再 Join，参与 Join 的数据少得多 |
| 凭空多出 `isnotnull(id)`、`isnotnull(user_id)` | **InferFiltersFromConstraints**（约束推导）：内连接 `id = user_id` 意味着两边都不能为 null，于是推导出过滤条件，**两张表都能提前过滤** |
| 出现 `Project [id, name]`、`Project [name, amount]` | **ColumnPruning**（列裁剪）：只保留后面用得到的列 |
| `SubqueryAlias`、`View` 节点消失了 | **FinishAnalysis** 批次里的 `EliminateSubqueryAliases`、`EliminateView`：它们只在分析阶段有用 |

### ④ 物理计划：变成可执行的算子

```
*(3) HashAggregate(keys=[name#33], functions=[sum(amount#37)])                      ← 最终聚合
+- Exchange hashpartitioning(name#33, 200), ENSURE_REQUIREMENTS                    ← Shuffle！Stage 边界
   +- *(2) HashAggregate(keys=[name#33], functions=[partial_sum(amount#37)])        ← 局部预聚合
      +- *(2) Project [name#33, amount#37]
         +- *(2) BroadcastHashJoin [id#32], [user_id#36], Inner, BuildRight         ← 广播 Join
            :- *(2) Project [id#32, name#33]
            :  +- *(2) Filter (...(age#34 > 30) AND (city#35 = BJ)...)
            :     +- *(2) ColumnarToRow
            :        +- FileScan parquet [id,name,age,city]
            :             PushedFilters: [IsNotNull(age), IsNotNull(city), GreaterThan(age,30), EqualTo(city,BJ), IsNotNull(id)]
            :             ReadSchema: struct<id:int,name:string,age:int,city:string>          ← 没有 bio！
            +- BroadcastExchange HashedRelationBroadcastMode(...)                          ← 小表广播出去
               +- *(1) Filter isnotnull(user_id#36)
                  +- *(1) ColumnarToRow
                     +- FileScan parquet [user_id,amount]
                          PushedFilters: [IsNotNull(user_id)]
```

| 现象 | 含义 |
|---|---|
| `PushedFilters` | 过滤条件被**继续下推到 Parquet 读取器**，Parquet 可以利用文件里的统计信息直接跳过不满足条件的数据块 |
| `ReadSchema` 里**没有 `bio`** | 列裁剪也作用到了**读文件这一层**，Parquet 是列式存储，不需要的列根本不会从磁盘读出来 |
| `BroadcastHashJoin ... BuildRight` | orders 表很小，被**广播**到每个 Executor（第 5 讲的广播），避免了一次 Shuffle |
| `HashAggregate` 出现两次 | **两阶段聚合**：Shuffle 前先 `partial_sum` 局部预聚合，Shuffle 后再求最终的 `sum`，大幅减少 Shuffle 数据量 |
| `Exchange hashpartitioning(name, 200)` | 一次 **Shuffle**（第 3 讲），也就是 **Stage 边界**（第 1 讲）；200 就是 `spark.sql.shuffle.partitions` |
| `*(1)`、`*(2)`、`*(3)` | **全阶段代码生成**（Whole-Stage Codegen）的编号，同一个编号的算子会被编译成一个函数。下一讲细讲 |
| `ColumnarToRow` | Parquet 读取器是**向量化**的（按列批量读），这里转换成按行处理 |

### ⑤ 规则统计：242 条里只有 8 条生效

通过 `queryExecution.tracker.rules` 统计：这条查询过程中**被调用过的规则有 242 条，真正改变了计划的只有 8 条**（3 条分析规则 + 5 条优化规则）。

绝大多数规则检查一下"这里没有我能做的"就返回了。这也是 TreePattern 剪枝（3.1 节）有价值的原因。

> 注意：物理计划的准备规则（`EnsureRequirements`、`CollapseCodegenStages` 等）是在 `prepareForExecution` 里用 `foldLeft` 逐个调用的（`QueryExecution.scala` 第 790 行附近），**不经过 RuleExecutor**，所以不在这个统计里。

**各阶段耗时**（`expected-output.txt` 那次运行）：解析 47 ms、分析 29 ms、优化 44 ms、物理计划 45 ms。每次运行会有波动，但量级相同：对这种规模的查询，Catalyst 的开销在百毫秒级，远小于实际执行时间。

---

## 5. 四个阶段详解

### 5.1 解析（Parsing）

- 语法定义：`sql/api/src/main/antlr4/.../parser/SqlBaseLexer.g4` 和 `SqlBaseParser.g4`（后者 **2724 行**），由 **ANTLR 4** 生成解析器
- `catalyst/parser/AstBuilder.scala`：遍历 ANTLR 生成的语法树，构造出 Catalyst 的逻辑计划；`execution/SparkSqlParser.scala` 在此基础上处理 Spark 特有的语法
- **DataFrame API 跳过这一步**：`df.filter($"age" > 30)` 直接构造 `Filter` 节点
- 🆕 **4.0 引入了 SQL 管道语法**：`FROM people |> WHERE age > 30 |> SELECT name`（词法规则 `OPERATOR_PIPE: '|>'`，`SqlBaseLexer.g4:582`）。优化器里有专门的 `EliminatePipeOperators` 规则处理它

### 5.2 分析（Analysis）

- `catalyst/analysis/Analyzer.scala`，4.2.0 中共 **19 个批次**，最核心的是 `"Resolution"` 批次（`FixedPoint`）
- 通过 Catalog（`SessionCatalog`，以及 DataSource V2 的 `CatalogManager`）查表、查函数
- 分析结束后由 `CheckAnalysis` 做检查：列不存在、类型不匹配、聚合用法错误等，都在这里报错

🆕 **新的单遍分析器（Single-pass Resolver）**

- `[SPARK-50472] Introduce initial implementation of the single-pass Analyzer`（2024-12，**4.0.0 起**）
- 目录 `catalyst/analysis/resolver/`，4.2.0 中已有约 **100 个文件**
- 思路：现在的分析器靠 `FixedPoint` 反复跑规则直到不动点；新的分析器**一次后序遍历**就完成解析，速度更快、行为更可预测
- 配置 `spark.sql.analyzer.singlePassResolver.enabled`：**内部配置，默认 `false`**，源码注释写着 *"This feature is currently under development"*

> 这是一个正在活跃开发的方向，代码新、改动多，**很适合关注和参与**。

### 5.3 优化（Optimization）

- `catalyst/optimizer/Optimizer.scala` + `execution/SparkOptimizer.scala`（加入 SQL 执行层特有的规则），4.2.0 中共 **30 多个批次**
- 重要的批次（按执行顺序的大致分组）：
  - `"Finish Analysis"`：去掉只对分析有用的节点
  - `"Operator Optimization before / after Inferring Filters"`、`"Infer Filters"`：谓词下推、列裁剪、常量折叠、合并算子等核心优化，以及约束推导
  - `"Join Reorder"`：基于代价的 Join 重排（需要开启 CBO 并收集统计信息）
  - `"Subquery"`、`"RewriteSubquery"`：子查询改写
  - `"Aggregate"`、`"Eliminate Sorts"`、`"Decimal Optimizations"` 等
- **调试技巧**：
  - `spark.sql.planChangeLog.level=WARN`：把每条规则对计划的改动都打印到日志
  - `spark.sql.optimizer.excludedRules=规则全名`：临时禁用某条规则，排查是不是它导致的问题

### 5.4 物理计划（Planning）

**策略（Strategy）**：`SparkPlanner`（`execution/SparkPlanner.scala`）里有一组策略，每个策略负责把某类逻辑算子翻译成物理算子，例如：

| 策略 | 负责 |
|---|---|
| `FileSourceStrategy` / `DataSourceV2Strategy` | 读文件、读数据源 |
| `JoinSelection` | 选择 Join 的物理实现（广播 / Shuffle Hash / Sort Merge……） |
| `Aggregation` | 聚合（HashAggregate / SortAggregate / ObjectHashAggregate） |
| `BasicOperators` | Project、Filter 等基础算子 |
| `InMemoryScans` | 读缓存的数据 |

**只取第一个候选计划**（`QueryExecution.createSparkPlan`，第 806 行起）：

```scala
// TODO: We use next(), i.e. take the first plan returned by the planner, here for now,
//       but we will implement to choose the best plan.
planner.plan(ReturnAnswer(plan)).next()
```

也就是说，物理计划阶段**并不会生成多个候选、按代价比较选最好的**。Join 方式的选择靠 `JoinSelection` 里的启发式规则（比如表小于 `spark.sql.autoBroadcastJoinThreshold`，默认 **10MB**，就广播），以及运行时的 **AQE**（第 10 讲）。

**准备规则（preparations）**（`QueryExecution.preparations`，第 750 行）：生成物理计划后，还要再经过一组规则，其中最重要的：

| 规则 | 作用 |
|---|---|
| `EnsureRequirements` | 检查每个算子对输入**分区和排序**的要求，不满足就**插入 `Exchange`（Shuffle）或 `Sort`**。实验里那个 `ENSURE_REQUIREMENTS` 标记就是它加的 |
| `CollapseCodegenStages` | 把能合并的算子合并成**全阶段代码生成**块，就是 `*(n)` 标记 |
| `ReuseExchangeAndSubquery` | 同样的 Shuffle / 子查询只算一次，复用结果 |
| `PlanSubqueries` / `PlanDynamicPruningFilters` | 子查询、动态分区裁剪 |
| （开启 AQE 时）`InsertAdaptiveSparkPlan` | 把整个计划包进 `AdaptiveSparkPlanExec`，运行时再调整 |

---

## 6. 怎么读 explain

```scala
df.explain()                 // 只看物理计划
df.explain("extended")       // 四个阶段都看
df.explain("formatted")      // 物理计划 + 每个算子的详细信息，更易读
df.explain("codegen")        // 生成的 Java 代码（下一讲）
df.explain("cost")           // 带统计信息
```

（五种模式定义在 `execution/ExplainMode.scala`。）

**读物理计划的技巧**：

1. **从下往上读**：最底下是读数据，最上面是输出
2. **找 `Exchange`**：每个 `Exchange` 就是一次 Shuffle、一个 Stage 边界，也往往是性能瓶颈所在
3. **看 `FileScan` 的 `PushedFilters` 和 `ReadSchema`**：过滤条件下推了吗？读了不需要的列吗？
4. **看 Join 类型**：`BroadcastHashJoin` 通常最快；大表之间常见 `SortMergeJoin`
5. **看 `*(n)`**：没有星号的算子没能参与代码生成

---

## 7. 和贡献的关系

Spark SQL 是社区 PR 最多的地方，常见的贡献类型都和本讲有关：

- **新增或修复优化规则**：在 `Optimizer.scala` 的某个批次里加一条 `Rule[LogicalPlan]`
- **新增内置函数**：实现一个 `Expression`（`eval` + `doGenCode`），在函数注册表里登记
- **改进报错信息**：分析阶段的错误都走统一的错误类
- **单遍分析器**：一个正在建设中的大方向

测试方式：`sql/core/src/test/resources/sql-tests/` 下的 **golden file 测试**（`SQLQueryTestSuite`）：写一个 `.sql` 文件，自动生成并比对结果文件。

---

## 自测题

1. 为什么 Spark 能优化 DataFrame / SQL，却很难优化 RDD 里的 `map` 函数？
2. SQL 和 DataFrame API 走的是同一条路径吗？有什么区别？
3. 解析后的计划里，单引号 `'` 表示什么？列名写错会在哪个阶段报错？
4. `transform` 是自顶向下还是自底向上？`FixedPoint` 批次什么时候停止？
5. 实验里 `isnotnull(user_id)` 是从哪里来的？为什么能提升性能？
6. `PushedFilters` 和优化后逻辑计划里的 `Filter` 有什么区别？
7. 实验里为什么 `HashAggregate` 出现了两次？
8. 物理计划阶段会比较多个候选计划的代价吗？Join 方式是怎么选出来的？
9. `Exchange` 是谁、在什么时候插入的？
10. 4.x 里有哪些和 Catalyst 相关的新方向？（提示：分析器、语法）

## 下一讲预告

第 9 讲：**代码生成与 Tungsten 执行引擎**。物理计划里的 `*(1)`、`*(2)` 到底生成了什么样的代码？为什么"全阶段代码生成"能比传统的火山模型快好几倍？
:::
