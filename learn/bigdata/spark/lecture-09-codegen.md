---
title: "第 9 讲：代码生成与 Tungsten"
description: "全阶段代码生成与 Tungsten：看生成的代码、做性能对比、理解 UnsafeRow。"
bigdata: "spark"
lesson: "e12"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/lecture-09-codegen-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-09-codegen-cover.webp" alt="代码熔炉加速Spark执行" width="1200" height="800" loading="eager" /><figcaption>代码熔炉加速Spark执行<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Spark 4.2.0 源码。路径缩写：`execution/…` = `sql/core/src/main/scala/org/apache/spark/sql/execution/`，`codegen/…` = `sql/catalyst/src/main/scala/org/apache/spark/sql/catalyst/expressions/codegen/`
> 配置默认值、类名、行号均在 4.2.0 源码中核实；版本演进来自 Git 提交历史；生成的代码、性能对比、UnsafeRow 布局都经过实验验证（仓库 `experiments/10-codegen/`）。

## 0. 本讲要回答的问题

1. 上一讲物理计划里的 `*(1)`、`*(2)` 到底是什么？
2. 传统的"火山模型"慢在哪里？
3. 全阶段代码生成生成的代码长什么样？快多少？
4. 什么情况下会**放弃**代码生成？
5. `UnsafeRow` 在内存里长什么样？为什么这样设计？
6. "向量化读取"又是什么？

---

## 1. 背景：CPU 成了瓶颈

Spark 早期的性能瓶颈主要在磁盘和网络。随着 SSD、万兆网卡普及，数据又多以列式格式（Parquet、ORC）存储，**CPU 逐渐成了瓶颈**。2015 年前后（Spark 1.4～1.5 时期）社区启动了 **Tungsten 项目**，目标就是榨干 CPU 和内存的效率，主要包括三件事（第 3 讲中 tungsten-sort 在 1.4 加入，就是其中的一部分）：

| 方向 | 内容 | 本系列中的位置 |
|---|---|---|
| **内存管理** | 绕过 Java 对象，直接管理二进制数据（堆内 `long[]` 或堆外内存） | 第 4 讲（页表、堆外内存） |
| **缓存友好的数据结构** | 紧凑的二进制格式，排序只排指针 | 第 3 讲（`PackedRecordPointer`）、本讲 `UnsafeRow` |
| **代码生成** | 运行时为具体的查询生成专用的 Java 代码 | 本讲 |

---

## 2. 火山模型：每一行都要层层调用

传统数据库和 Spark 早期使用的是**火山模型**（Volcano / Iterator Model）：每个算子都实现一个 `next()` 方法，上层算子调用下层的 `next()` 一行一行地"拉"数据。

```
Project.next()  → 调用 Filter.next()
  Filter.next() → 循环调用 Range.next()，直到找到满足条件的一行
    Range.next() → 返回下一行
```

**问题**：

1. **虚函数调用开销**：每处理一行，每个算子都要发生一次 `next()` 虚调用。数据量一大，这些调用本身就占了大量 CPU 时间
2. **通用的表达式求值**：`id % 3 = 0` 被表示成一棵表达式树，求值时要递归调用每个节点的 `eval()`，每一步都有类型判断、装箱拆箱
3. **难以利用 CPU 优化**：数据在算子之间通过对象传递，编译器和 CPU 很难做寄存器分配、循环展开、SIMD 等优化

> 这和第 1 讲 RDD 的"流水线执行"思路一样，都是迭代器嵌套。区别在于 RDD 的迭代器里跑的是用户的函数，而 SQL 有机会做得更好，因为 Spark 知道每个算子具体要干什么。

---

## 3. 全阶段代码生成（Whole-Stage Codegen）

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-09-codegen-1.webp" alt="多算子融合成一个高速循环" width="960" height="640" loading="lazy" /><figcaption>多算子融合成一个高速循环<span>AI 生成配图</span></figcaption></figure>

### 3.1 思路

**既然 Spark 知道这一串算子要做什么，就干脆为这个查询"手写"一段专用代码**：把多个算子融合成一个循环，变量直接放在局部变量（最终是 CPU 寄存器）里，不经过任何迭代器和虚调用。

- `[SPARK-12796] [SQL] Whole stage codegen`，2016-01，**Spark 2.0 引入**
- 默认开启：`spark.sql.codegen.wholeStage=true`

### 3.2 实验 A：生成的代码长什么样

```scala
spark.range(0, 1000).filter("id % 3 = 0").selectExpr("id * 2 AS x")
```

物理计划：

```
*(1) Project [(id#0L * 2) AS x#4L]
+- *(1) Filter ((id#0L % 3) = 0)
   +- *(1) Range (0, 1000, step=1, splits=1)
```

三个算子都标着 **`*(1)`**：它们属于**同一个代码生成阶段**，会被编译成**一个** Java 类。生成的代码共 **142 行**（完整代码见仓库 `experiments/10-codegen/generated-code.java`），核心的 `processNext()` 节选：

```java
protected void processNext() throws java.io.IOException {
  ...
  while (true) {
    ...
    for (int range_localIdx_0 = 0; range_localIdx_0 < range_localEnd_0; range_localIdx_0++) {
      long range_value_0 = ((long)range_localIdx_0 * 1L) + range_nextIndex_0;   // ← Range：生成 id

      do {
        boolean filter_isNull_0 = true;
        boolean filter_value_0 = false;
        ...
        if (3L == 0) {
          throw QueryExecutionErrors.remainderByZeroError(...);                    // ← ANSI 模式的除零检查
        } else {
          filter_value_1 = (long)(range_value_0 % 3L);                             // ← Filter：id % 3
        }
        ...
        if (filter_isNull_0 || !filter_value_0) continue;                          // ← 不满足条件，跳过

        ((SQLMetric) references[1] /* numOutputRows */).add(1);
        // ...后面是 Project：id * 2，写入输出行
      } while(false);
    }
    ...
  }
}
```

**对照火山模型**：

- **没有 `next()` 调用**：Range、Filter、Project 被**融合进同一个 `for` 循环**
- **没有表达式树**：`id % 3` 直接变成了 `range_value_0 % 3L` 这样的原生 Java 运算
- **中间值都是局部变量**（`range_value_0`、`filter_value_1`），JIT 编译后通常会被放进 CPU 寄存器

> 💡 注意 `if (3L == 0) throw remainderByZeroError`：这是因为 **4.0 起 ANSI 模式默认开启**（`[SPARK-44444] Use ANSI SQL mode by default`，`spark.sql.ansi.enabled` 默认 `true`）。ANSI 模式下除以 0 会**报错**，而不是像以前那样返回 null。常量 `3L == 0` 这样的判断会被 JIT 直接优化掉，没有运行时开销。

### 3.3 实验 B：快多少

同一条查询：`sum(id * 2) WHERE id % 3 = 0`，id 从 0 到 **3 亿**，单线程（`local[1]`），每种配置预热一次后跑 3 次取中位数：

| 配置 | 中位数耗时 | 相对 ① |
|---|---|---|
| ① **全阶段代码生成**（默认） | **347 ms** | 1× |
| ② 关闭全阶段代码生成（`spark.sql.codegen.wholeStage=false`），表达式仍单独代码生成 | **5943 ms** | 约 **17 倍** |
| ③ 全部关闭（再加上 `spark.sql.codegen.factoryMode=NO_CODEGEN`），纯解释执行 | **25999 ms** | 约 **75 倍** |

- ① → ②：算子之间回到了迭代器调用，**慢了一个数量级**。这部分差距来自**算子融合**
- ② → ③：连表达式也不再生成代码，改为递归调用 `eval()`，又慢了 4 倍多。这部分差距来自**表达式代码生成**

> 具体数字取决于机器和数据，**倍数关系才是重点**。② 的 3 次结果波动较大（4832～10855 ms），推测和 GC 有关，所以取中位数。

### 3.4 代码是怎么生成的：produce / consume

每个支持代码生成的物理算子都实现 `CodegenSupport`（`execution/WholeStageCodegenExec.scala:47`），核心是两个方法：

| 方法 | 方向 | 作用 |
|---|---|---|
| `produce()` → `doProduce()` | **自顶向下** | "我需要数据"：向子节点要数据，最底层的算子（如 Range、Scan）负责生成**循环** |
| `consume()` → `doConsume()` | **自底向上** | "这一行给你"：子节点把当前行交给父节点，父节点把**自己的处理逻辑**拼进循环体里 |

```
WholeStageCodegenExec.produce()
  → Project.produce()
    → Filter.produce()
      → Range.produce()  生成 for 循环，在循环体里调用 consume(当前行)
                         ↓
      ← Filter.doConsume()   生成 "if (!(id % 3 == 0)) continue;"，再调用 consume
    ← Project.doConsume()    生成 "x = id * 2"，再调用 consume
  ← WholeStageCodegenExec    生成 "把结果写入输出缓冲区"
```

最终得到一个继承 `BufferedRowIterator` 的类（第 698 行），名字是 `GeneratedIteratorForCodegenStage{n}`（第 663 行），`{n}` 就是计划里 `*(n)` 的编号。

**谁来划分代码生成阶段**：上一讲提到的准备规则 **`CollapseCodegenStages`**，它把连续的、都支持代码生成的算子包进一个 `WholeStageCodegenExec`。**`Exchange`（Shuffle）是天然的边界**，所以第 8 讲实验里 Shuffle 前后分别是 `*(2)` 和 `*(3)`。

**表达式的代码生成**：每个 `Expression` 有两套实现（`Expression.scala`）：

- `eval(input)`（第 207 行）：解释执行
- `doGenCode(ctx, ev)`（第 280 行）：生成 Java 代码片段

没有实现 `doGenCode` 的表达式可以混入 `CodegenFallback`（`codegen/CodegenFallback.scala`），在生成的代码里直接回调它的 `eval()`，这样整个阶段仍然可以做代码生成。

> 第 1 讲学习路线里"自己实现一个内置函数"的练习，要写的就是 `eval` 和 `doGenCode` 这两个方法。

### 3.5 编译：Janino

生成的是 Java **源代码字符串**，运行时用 **Janino**（一个轻量级的 Java 编译器）编译成字节码（`codegen/CodeGenerator.scala`，`import org.codehaus.janino.ClassBodyEvaluator`）。

- Janino 编译速度很快（毫秒级），适合在运行时为每个查询现编现用
- 编译结果有缓存，最多 `spark.sql.codegen.cache.maxEntries`（默认 **100**）个，相同的代码不会重复编译

---

## 4. 什么时候放弃代码生成

### 4.1 字段太多：`spark.sql.codegen.maxFields`

```
 50 列：*(1) Project [(id + 1) AS c1, (id + 2) AS c2, ...    ← 有 *，参与了全阶段代码生成
120 列：Project [(id + 1) AS c1, (id + 2) AS c2, ...         ← 没有 *，回退为普通执行
```

`CollapseCodegenStages` 会检查算子的输出 schema，嵌套字段总数超过 `spark.sql.codegen.maxFields`（默认 **100**）就不做全阶段代码生成（`numOfNestedFields`，`WholeStageCodegenExec.scala:582-591`）。字段太多会让生成的代码过于庞大，编译慢、JIT 也优化不了。

### 4.2 方法太大：`spark.sql.codegen.hugeMethodLimit`

代码编译完成后，如果其中**最大的一个方法的字节码**超过 `spark.sql.codegen.hugeMethodLimit`（默认 **65535**），也会放弃全阶段代码生成，回退成普通执行（`WholeStageCodegenExec.scala:753`），日志里会出现：

```
Found too long generated codes and JIT optimization might not work: the bytecode size (...) is above the limit ...
```

这里有个容易忽视的细节（配置说明原文）：

- 默认值 **65535** 是 **Java 方法字节码大小的上限**，超过它代码根本无法加载
- 但 **HotSpot JVM 默认不会 JIT 编译字节码超过 8000 字节的方法**（HotSpot 的 `HugeMethodLimit` 默认 8000，由默认开启的 `DontCompileHugeMethods` 选项控制），这种方法只能解释执行，非常慢
- 所以源码里定义了 `CodeGenerator.DEFAULT_JVM_HUGE_METHOD_LIMIT = 8000`（注释："This is the default value of HugeMethodLimit in the OpenJDK HotSpot JVM, beyond which methods will be rejected from JIT compilation"），配置说明也建议：**在 HotSpot 上可以把它设成 8000**，和 JVM 的行为保持一致

> 生成的方法太大时，Spark 也会尝试把表达式拆分成多个小方法（`spark.sql.codegen.methodSplitThreshold`，默认 1024），尽量避免触发这个限制。

### 4.3 编译失败：`spark.sql.codegen.fallback`

如果生成的代码编译失败（比如触发了 Janino 的 bug），默认会**回退到解释执行**而不是让查询失败（`spark.sql.codegen.fallback=true`）。`spark.sql.codegen.factoryMode`（默认 `FALLBACK`）也控制类似的行为：先尝试代码生成，失败了再用解释执行。

---

## 5. UnsafeRow：行的二进制格式

<figure class="ai-figure"><img src="/bigdata-img/spark/lecture-09-codegen-2.webp" alt="UnsafeRow把行压成三段内存" width="960" height="640" loading="lazy" /><figcaption>UnsafeRow把行压成三段内存<span>AI 生成配图</span></figcaption></figure>

### 5.1 布局

`UnsafeRow`（`sql/catalyst/src/main/java/.../expressions/UnsafeRow.java`）是 Spark SQL 内部最常用的行格式。源码注释：

> Each tuple has three parts: **[null-tracking bit set] [values] [variable length portion]**

| 区域 | 内容 |
|---|---|
| **null 位图** | 每个字段 1 位，按 8 字节对齐 |
| **定长区** | **每个字段固定占 8 字节**。int、long、double 等定长类型直接存值；字符串等变长类型存一个 long：**高 32 位是偏移量，低 32 位是长度** |
| **变长区** | 字符串、数组等的实际内容 |

### 5.2 实验 D：亲眼看看字节

`(7, 100L, "spark")`，schema 为 `(int, long, string)`：

```
共 40 字节
word 0  0x0000000000000000   null 位图：没有 null
word 1  0x0000000000000007   字段 0 (int)：7
word 2  0x0000000000000064   字段 1 (long)：100（0x64）
word 3  0x0000002000000005   字段 2 (string)：偏移 0x20 = 32 字节，长度 5
word 4  0x0000006b72617073   变长区：字符串内容
```

- **word 3**：高 32 位 `0x20` = 32，表示字符串从第 32 字节开始（正好是 word 4 的起点）；低 32 位 `5` 是长度
- **word 4**：内存中按小端存储，字节顺序是 `73 70 61 72 6b`，也就是 ASCII 的 `s p a r k`（上面为了按数值显示把字节倒过来了），剩下 3 个字节补 0，凑满 8 字节对齐

把第一个字段换成 `null`：

```
word 0  0x0000000000000001   null 位图：第 0 位为 1，表示字段 0 是 null
word 1  0x0000000000000000   字段 0：值被清零
```

### 5.3 为什么这样设计

- **按位置直接访问**：读第 i 个字段，就是读"基址 + 位图长度 + i × 8"处的 8 个字节，**不用解析、不用反序列化**
- **紧凑**：没有 Java 对象头、没有指针，40 字节就是一行的全部
- **整行就是一段连续字节**：排序时可以直接比较字节；Shuffle 时可以直接拷贝，不需要再序列化。这就是第 3 讲 SQL 的 Shuffle 用 `UnsafeRowSerializer`、并且支持"重定位"的原因
- **哈希、比较都可以直接作用在二进制上**，HashAggregate、Join 都受益

---

## 6. 向量化执行：一次处理一批

代码生成是"按行"处理的，但**读 Parquet、ORC 这种列式文件时**，按列批量处理更高效。

- **`ColumnarBatch`**（`sql/catalyst/src/main/java/.../vectorized/ColumnarBatch.java`）：一批行，按列组织，每列是一个 `ColumnVector`（实现有 `OnHeapColumnVector` / `OffHeapColumnVector`）
- **向量化 Parquet 读取器** `VectorizedParquetRecordReader`：一次解码一整批（默认 **4096** 行，`spark.sql.parquet.columnarReaderBatchSize`）的一列数据，而不是一个值一个值地解码。默认开启：`spark.sql.parquet.enableVectorizedReader=true`
- **`ColumnarToRow`**（`execution/Columnar.scala:67`，`ColumnarToRowExec`）：第 8 讲物理计划里出现过它。读取是列式的，但后面的算子（Filter、Join……）是按行处理的，需要在这里转换一下。它本身也实现了 `CodegenSupport`，参与代码生成：生成的代码直接从 `ColumnVector` 里按下标取值

> 一个完全列式的执行引擎（所有算子都按列批量处理）还能更快，这也是 Spark 生态里各种**原生加速引擎**（用 C++ 或 Rust 实现、通过插件替换 Spark 物理算子）的出发点。准备规则里的 `ApplyColumnarRulesAndInsertTransitions`（第 8 讲）就是给这类插件留的扩展点。

---

## 7. 配置速查（4.2.0 默认值）

| 配置 | 默认值 | 说明 |
|---|---|---|
| `spark.sql.codegen.wholeStage` | `true` | 全阶段代码生成 |
| `spark.sql.codegen.maxFields` | `100` | 输出字段超过这个数不做全阶段代码生成 |
| `spark.sql.codegen.hugeMethodLimit` | `65535` | 单个方法字节码上限；HotSpot 上可考虑设为 `8000` |
| `spark.sql.codegen.methodSplitThreshold` | `1024` | 表达式代码超过这个长度时拆成多个方法 |
| `spark.sql.codegen.fallback` | `true` | 编译失败时回退到解释执行 |
| `spark.sql.codegen.factoryMode` | `FALLBACK` | 先尝试代码生成，失败再解释执行 |
| `spark.sql.codegen.cache.maxEntries` | `100` | 编译结果缓存数 |
| `spark.sql.parquet.enableVectorizedReader` | `true` | 向量化 Parquet 读取 |
| `spark.sql.parquet.columnarReaderBatchSize` | `4096` | 向量化读取的批大小 |
| `spark.sql.ansi.enabled` | `true`（4.0 起） | ANSI 模式 |

---

## 自测题

1. 火山模型慢在哪里？全阶段代码生成分别是怎么解决这些问题的？
2. 物理计划里 `*(1)`、`*(2)` 的编号是怎么来的？为什么 Shuffle 前后的编号不同？
3. 实验 B 中，① → ② 和 ② → ③ 两次变慢，分别来自什么？
4. `produce` 和 `consume` 分别是什么方向？最底层的算子在 `doProduce` 里生成了什么？
5. 一个查询选了 150 列，它的 `Project` 会参与全阶段代码生成吗？
6. `hugeMethodLimit` 默认是 65535，为什么源码建议在 HotSpot 上设成 8000？
7. 一个没有实现 `doGenCode` 的表达式，会让整个阶段都放弃代码生成吗？
8. 实验 D 中，word 3 的值 `0x0000002000000005` 表示什么？
9. 为什么说 `UnsafeRow` 的设计让 Shuffle 和排序都更快？
10. 生成的代码里为什么会有 `if (3L == 0) throw ...`？在 Spark 3.x 的默认配置下会有吗？

## 下一讲预告

第 10 讲：**AQE、DataSource V2 与 Spark Connect**。运行时如何根据真实数据重新优化执行计划？外部数据源（Iceberg、Paimon 等）是怎么接入 Spark 的？Spark Connect 又改变了什么？
:::
