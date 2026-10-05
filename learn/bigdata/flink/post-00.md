---
title: "Flink 2.x 源码精读（零）：147 万行代码，从哪里开始读？"
description: "Flink 2.3 有 147 万行生产代码，老资料里的很多路径已经失效。本篇给出一条从入门到 Committer 的学习路线，并带你把 2.3.0 源码克隆、编译、跑起来，所有命令都在本机实测过。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/post-00-cover.webp"}]]
---

# Flink 2.x 源码精读（零）：147 万行代码，从哪里开始读？

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/post-00-cover.webp" alt="Yui和Kai探索Flink源码与流处理世界" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai探索Flink源码与流处理世界<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 本系列基于 **Apache Flink 2.3.0**。文中的命令、数字和运行结果，都是我在本机（Apple M5 Pro，macOS，JDK 17）实际跑出来的。换一台机器，耗时会不一样，但步骤是一样的。

---

先说一个数字。

我在 Flink 2.3.0 的源码仓库里数了一下：`src/main` 下的 Java 代码有 **9553 个文件、147 万行**，测试代码还有 **5140 个文件、110 万行**。

第一次打开这个仓库的人，几乎都会遇到同一个问题：**从哪里开始读？**

更麻烦的是第二个问题：网上能搜到的 Flink 源码文章，大多是基于 1.x 写的。而 Flink 2.0 做了一次大调整，很多东西已经不在原来的位置了。举个例子：

> 老文章会告诉你，`StreamExecutionEnvironment` 在 `flink-streaming-java` 模块里。
> 但在 2.x 里，它已经搬到了 `flink-runtime` 模块。这次迁移来自 FLINK-36063（2024 年 8 月），从 2.0.0 开始生效。现在 `flink-runtime` 里 `org.apache.flink.streaming` 包下有 535 个文件，而 `flink-streaming-java` 的 `src/main` 只剩 96 个文件。

照着老文章找类，找不到；找到了，行为可能也变了。

所以我决定从头写一个系列：**基于 Flink 2.3.0，每个结论都给出源码位置，每个关键行为都写一个能跑的小实验来验证。**

这是第零讲，做两件事：

1. 给出一条**从入门到 Committer** 的学习路线，让你知道自己现在在哪、下一步该读什么；
2. 带你把 Flink 2.3.0 的源码**克隆、编译、跑起来**，并在 IDE 里打上第一个断点。

---

## 一、先想清楚：你为什么要读源码？

读源码很花时间，开始之前值得想清楚目的。我见过的动机大概有三种：

**1. 解决生产问题。** 窗口为什么一直不出结果？Checkpoint 为什么总是超时？数据为什么重复了？文档往往只告诉你"怎么用"，不告诉你"为什么会这样"。能把现象定位到源码的某一行，排查问题的效率会完全不同。

**2. 学习设计。** Flink 里有很多值得学的工程设计：怎么在不停止数据流的前提下做全局一致的快照？怎么让一个线程同时处理数据、定时器和 Checkpoint，却不需要加锁？这些问题的答案，在任何一本书里都没有源码讲得清楚。

**3. 参与开源。** 想给 Apache 项目提 PR，最终成为 Committer，读源码是绕不过去的一步。

反过来说，**如果你只是想把 Flink 用起来，先别急着读源码**。把官方文档的 Learn Flink 和 Concepts 两部分读完、写几个作业跑一跑，收益更高。源码要带着问题去读，没有问题的时候，读起来会非常痛苦。

---

## 二、一条从入门到 Committer 的路线

<figure class="ai-figure"><img src="/bigdata-img/flink/post-00-1.webp" alt="Yui沿五级台阶掌握Flink源码" width="960" height="640" loading="lazy" /><figcaption>Yui沿五级台阶掌握Flink源码<span>AI 生成配图</span></figcaption></figure>

我把学习过程分成五个阶段。时长是**按业余时间、每周 10 小时左右估算的参考值**，每个人的基础不同，差异会很大，关键是看"验收标志"有没有达到。

【配图 1：五阶段路线图（横向阶梯），每一级写阶段名 + 验收标志】

### 阶段 1：入门 —— 先会用（参考 1~2 个月）

读官方文档，写作业。核心概念要真正理解，而不是背下来：并行度、算子链、Slot、Keyed State、事件时间与 Watermark、窗口、Checkpoint 与 Savepoint、exactly-once。

**验收**：能独立写出一个"带状态 + 事件时间 + 窗口"的作业，并部署运行。能说清楚 operator state 和 keyed state 的区别。

### 阶段 2：中级 —— 读通执行主链路（参考 3~4 个月）

这一阶段只回答一个问题：**我写的 `map().keyBy().sum()`，是怎么变成集群上运行的 Task 的？**

这条链路上有四张"图"，是整个 Flink 运行时的骨架：

| 图 | 在哪里生成 | 一个节点代表 |
|---|---|---|
| Transformation 列表 | 调用 API 时 | 一次 API 调用 |
| StreamGraph | 客户端 | 一个算子 |
| JobGraph | **JobManager 侧**（2.x 的变化） | 一条算子链 |
| ExecutionGraph | JobManager 侧 | 一个并行实例 |

注意第三行：在 1.x 里，JobGraph 是在客户端生成好再提交的；而在 2.x 里，本地模式和 Session 模式会**直接提交 StreamGraph**，由 JobManager 生成 JobGraph。这又是一个老资料已经过时的地方。

**验收**：在 Web UI 上看到任何一个现象（比如两个算子没有 chain 在一起），都能说出是源码里哪一段逻辑决定的。

### 阶段 3：高级 —— 吃透核心子系统（参考 4~6 个月）

Flink 有几个核心子系统：Checkpoint、State Backend、网络栈与背压、调度与容错、Table/SQL、Source/Sink。**不需要全部吃透，至少选两个，其中一个作为你的主攻方向。**

如果不知道选哪个，我的建议是从 **Checkpoint** 开始：它是 Flink 最核心的机制，面试问得最多，也和其他所有子系统都有关联。

**验收**：能把一个生产问题定位到源码的具体位置；开始向社区提交第一批 PR。

### 阶段 4：专家 —— 参与设计（参考 6~12 个月）

读你主攻方向近几年的 FLIP（Flink Improvement Proposal，Flink 的设计提案），理解社区为什么这样设计、放弃了哪些方案；跟踪 dev 邮件列表的讨论；开始 review 别人的 PR。

**验收**：在主攻方向有多个已合入的重要 PR，经常 review 别人的代码。

### 阶段 5：Committer

**Committer 不是考出来的。** 它是社区对你长期、持续、可见的贡献的认可，由 PMC 提名并投票产生。代码只是其中一部分，review、回答用户问题、写文档、参与讨论和发版验证，都算贡献。

所以有一个反直觉的建议：**到了阶段 3 就开始参与社区，不要等到"学完了"再去。** 你永远不会觉得自己学完了。

---

## 三、动手：把 Flink 2.3.0 跑起来

<figure class="ai-figure"><img src="/bigdata-img/flink/post-00-2.webp" alt="数据流穿过算子并由屏障完成快照" width="960" height="640" loading="lazy" /><figcaption>数据流穿过算子并由屏障完成快照<span>AI 生成配图</span></figcaption></figure>

下面每一步我都在本机跑过。

### 1. 准备环境

| 工具 | 要求 | 说明 |
|---|---|---|
| JDK | 11、17 或 21 | **推荐 17**，这是 2.3.0 的默认构建版本。注意：不管用哪个 JDK 编译，Flink 的代码本身只允许使用 Java 11 的语法 |
| Maven | 3.8.6 | **不用自己装**，仓库自带的 `./mvnw` 会自动下载这个版本 |
| Git | 任意较新版本 | |
| 磁盘 | 预留 10GB 以上 | 我本机克隆并编译之后，整个目录约 4.7GB，还不算 Maven 本地仓库中下载的依赖 |

### 2. 克隆源码，切到 2.3.0

```bash
git clone --filter=blob:none https://github.com/apache/flink.git
cd flink
git checkout -b study-2.3.0 release-2.3.0
```

这里有两个细节：

**第一，`--filter=blob:none`** 是"部分克隆"：先只下载提交记录和目录结构，文件内容等到需要时再下载。Flink 的历史很长，国内访问 GitHub 又不稳定，这样做能大大减少第一次克隆的数据量。我本机克隆完的 `.git` 目录是 216MB。

但它有一个**副作用**，我自己踩过：之后用 `git log -p`、`git blame` 查看老文件的历史时，Git 会临时去 GitHub 下载对应的文件内容。网络不好时会直接超时报错。如果你经常需要翻历史，网络条件又允许，就用普通的完整克隆。

**第二，基于 release 标签新建一个自己的分支**，而不是直接在 master 上读。master 每天都在变，今天找到的行号明天可能就对不上了。固定在一个发布版本上，你的笔记才是稳定的。本系列所有的行号都基于 `release-2.3.0`。

### 3. 配置 Maven 镜像（国内网络建议）

Flink 的依赖很多，直接从 Maven 中央仓库下载会很慢。我在仓库外面放了一个单独的配置文件 `maven-settings-aliyun.xml`：

```xml
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0">
  <mirrors>
    <mirror>
      <id>aliyun</id>
      <mirrorOf>central</mirrorOf>
      <name>Aliyun Maven</name>
      <url>https://maven.aliyun.com/repository/public</url>
    </mirror>
  </mirrors>
</settings>
```

编译时用 `-s` 指定这个文件。**单独放一个文件的好处是不会影响你本机其他项目的 Maven 配置。**

### 4. 编译

在 `flink` 目录下执行（假设配置文件放在上一级目录）：

```bash
./mvnw -s ../maven-settings-aliyun.xml clean install -DskipTests -Dfast -Pskip-webui-build -T1C
```

每个参数的含义：

| 参数 | 作用 |
|---|---|
| `clean install` | 清理后编译，并把产物安装到本地 Maven 仓库。用 `install` 而不是 `package`，是为了之后自己写的小实验可以直接依赖本地编译出来的 2.3.0 |
| `-DskipTests` | 跳过测试。Flink 的测试量非常大，学习阶段不需要全部跑，用到哪个模块再单独跑哪个模块的测试 |
| `-Dfast` | 激活 `fast` profile，跳过 7 个检查类插件：许可证检查（rat）、checkstyle、spotless、enforcer、javadoc、API 兼容性检查（japicmp）、SBOM 生成（cyclonedx） |
| `-Pskip-webui-build` | 跳过 Web UI 前端的构建。前端构建要从 nodejs.org 下载 Node.js、再安装上千个 npm 包，网络不好时最容易卡在这一步。**代价是编译出来的 Flink 没有 Web UI 页面**，后面第 5 步会讲怎么单独补上 |
| `-T1C` | 按每个 CPU 核心一个线程并行构建 |

本机实测（依赖已经下载到本地 Maven 仓库的情况下）：

```
[INFO] BUILD SUCCESS
[INFO] Total time:  05:20 min (Wall Clock)
```

一共编译了 172 个模块，用时 5 分 20 秒。

**第一次编译会慢很多**，因为要下载全部依赖，具体耗时取决于网络。如果中途失败，大多是依赖下载超时，直接重新执行同一条命令即可（可以去掉 `clean`，从断点继续）。

编译完成后，仓库根目录会出现一个 `build-target`，它是一个指向 `flink-dist/target/flink-2.3.0-bin/flink-2.3.0` 的软链接，里面就是一个完整的 Flink 发行包。

### 5. 验证：跑一个 WordCount

```bash
cd build-target
./bin/start-cluster.sh
./bin/flink run examples/streaming/WordCount.jar
```

本机实测输出：

```
Executing example with default input data.
Use --input to specify file input.
Printing result to stdout. Use --output to specify output path.
Job has been submitted with JobID baeadfce14c5ec4b092fbdc4631461b7
Program execution finished
Job with JobID baeadfce14c5ec4b092fbdc4631461b7 has finished.
Job Runtime: 457 ms
```

作业 457 毫秒就跑完了，但命令行里看不到任何单词计数。**结果在哪？** 在集群模式下，`print()` 的输出写到了 TaskManager 进程的标准输出里，也就是 `log/` 目录下以 `taskexecutor` 开头、`.out` 结尾的文件：

```bash
tail -5 log/*taskexecutor*.out
```

```
(be,4)
(all,2)
(my,1)
(sins,1)
(remember,1)
```

我这次一共输出了 287 行。默认的输入是《哈姆雷特》里"To be, or not to be"那段独白。流式 WordCount 每来一个单词就输出一次当前的计数，所以同一个单词会出现多次，比如 `(the,20)`、`(the,21)`、`(the,22)`。

**关于 Web UI**：如果你按第 4 步加了 `-Pskip-webui-build`，这时打开 `http://localhost:8081` 会得到一个 404：

```
{"errors":["Unable to load requested file /index.html."]}
```

REST 接口是正常的（比如 `http://localhost:8081/jobs/overview` 能看到刚才的作业），只是没有页面。需要 Web UI 时，单独编译前端模块和发行包即可：

```bash
./mvnw -s ../maven-settings-aliyun.xml install -DskipTests -Dfast -pl flink-runtime-web,flink-dist
```

本机实测用时 2 分 50 秒，其中会从 nodejs.org 下载 Node.js v22.16.0，再安装 1459 个 npm 包（这一步用了约 1 分钟）。完成后重启集群，就能看到 "Apache Flink Web Dashboard" 了。**我的建议是：第一次编译先跳过前端，确保主体编译成功；需要 Web UI 时再单独补。** 这样即使前端下载失败，也不会让整个编译前功尽弃。

验证完记得关掉集群：

```bash
./bin/stop-cluster.sh
```

【配图 2：Web UI 中 WordCount 作业的执行图截图】

### 6. 导入 IntelliJ IDEA

官方的 IDE 配置说明在仓库根目录的 `DEVELOPMENT.md` 里（文档是按 IntelliJ IDEA 2021.2 写的，新版本菜单位置可能略有不同），下面是关键步骤：

1. File → New → Project from Existing Sources，选择 `flink` 目录，导入方式选 **Maven**；
2. SDK 选择 **JDK 17**；
3. 导入完成后，在 Maven 面板中执行 **Generate Sources and Update Folders**。Flink 有一部分代码是构建时生成的，这一步会把它们生成出来并加入 IDE 的源码目录；
4. 安装 **Scala 插件**。Table/SQL 的优化器有大量 Scala 代码，不装插件这部分代码无法跳转。

`DEVELOPMENT.md` 里还有代码格式化（google-java-format）、Checkstyle、版权头等配置。**这些是给社区提代码时才需要的**，现在可以先跳过，等到第九讲讲社区贡献时再配。

另外建议执行一次：

```bash
git config blame.ignoreRevsFile .git-blame-ignore-revs
```

它会让 `git blame` 跳过历史上几次大规模的代码格式化提交，否则你 blame 出来的往往是"某次格式化"，而不是真正写这行代码的人。

### 7. 打下第一个断点

在 IDEA 中打开这个文件，直接运行它的 `main` 方法：

```
flink-examples/flink-examples-streaming/src/main/java/org/apache/flink/streaming/examples/wordcount/WordCount.java
```

在 IDE 里直接运行时，Flink 会在**同一个 JVM 里启动一个 MiniCluster**：JobManager 和 TaskManager 都在里面。这意味着，**从客户端到 TaskManager，任何一行代码都可以打断点**，不需要远程调试。

为了确认这一点，我没有用 IDE，而是直接用 `java` 命令运行了同一个 `main` 方法，效果和在 IDE 里点运行是一样的。3 秒跑完，同样输出 287 行，这次结果**直接打印在控制台上**：

```
5> (quietus,1)
13> (by,1)
9> (sweat,1)
9> (dread,1)
12> (end,1)
```

每行前面的数字是输出这一行的**并行子任务编号**。本地模式下默认并行度等于 CPU 核数，所以你会看到很多不同的编号，而且每次运行的顺序都可能不同。

下一讲，我们就从这里出发，跟着断点走一遍"从 `env.execute()` 到 `processElement()`"的完整链路。

一个调试小技巧：断点上点右键，把 **Suspend** 从 "All" 改成 **"Thread"**。选 "All" 时，断点一停，JVM 里所有线程都会暂停，包括负责心跳和 RPC 的线程，停得太久，作业可能会因为超时而失败。

---

## 四、读源码的四个方法

最后分享四个我觉得最有用的方法。

**1. 带着问题读，跟着断点走。** 不要从某个包的第一个文件开始往下读，那样很快会迷失。先提一个具体的问题（比如"Checkpoint Barrier 是怎么从 Source 传到 Sink 的"），然后在关键类上打断点、跑一个小例子，看调用栈。调用栈就是最好的阅读顺序。

**2. 读测试。** 每个核心类基本都有单元测试，测试代码就是最好的使用说明：它告诉你这个类应该怎么用、边界情况是什么。2.3.0 有 110 万行测试代码，这是一笔巨大的财富。

**3. 读提交记录。** 某段代码看不懂为什么这样写，就去查它的历史：

```bash
git log --follow --format='%h %ad %s' --date=short -- <文件路径>
```

Flink 的提交信息都带有 JIRA 编号（形如 `[FLINK-36063]`），顺着编号就能找到当时的讨论和设计动机。前面提到的"`StreamExecutionEnvironment` 搬到了 `flink-runtime`"，就是这样查出来的。

**4. 自己写实验验证。** 读到一个结论，就写一个最小的程序验证它。比如读到"下游的 Watermark 取所有上游的最小值"，就构造一个其中一个分区没有数据的作业，看窗口还会不会触发（剧透：12 秒内一次都不触发，第七讲会详细讲）。**能跑出来的结论，才是你自己的。**

---

## 五、这个系列会讲什么

| 讲 | 主题 | 要回答的核心问题 |
|---|---|---|
| 零 | 学习路线与环境准备 | 从哪里开始读（本篇） |
| 一 | 作业执行全链路 | `map().keyBy().sum()` 是怎么变成 Task 跑起来的 |
| 二 | Checkpoint 全流程 | 不停下数据流，怎么做全局一致的快照 |
| 三 | State Backend 与扩缩容 | 状态存在哪？改了并行度，状态怎么切分 |
| 四 | 网络栈与背压 | 背压是怎么一级一级传到上游的 |
| 五 | Table/SQL | 一条 SQL 是怎么变成 Java 代码的 |
| 六 | 调度与容错 | 一个 Task 失败，为什么只重启了一部分 |
| 七 | 时间、Watermark 与窗口 | 窗口为什么不触发？迟到数据去哪了 |
| 八 | Source / Sink 新架构 | exactly-once 为什么还会重复 |
| 九 | 从读者到贡献者 | 怎么给 Apache Flink 提第一个 PR |

每一讲都会包括：主线源码分析、可以运行的实验和本机实测的输出、断点清单、课后练习，以及我自己踩过的坑。

**实验代码**：【待填写：示例仓库地址】

---

## 写在最后


**留一个问题**：你现在处在哪个阶段？读 Flink 源码时，最想搞明白的是哪个问题？

呼声最高的问题，我会优先安排。

下一讲：**从 `env.execute()` 到 `processElement()`，跟着断点走一遍作业执行的完整链路。**



---

## 配图

本文全部配图在 `assets/png/lesson0/`，逐张用途见 配图索引。
:::
