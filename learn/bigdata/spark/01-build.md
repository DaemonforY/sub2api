---
title: "源码通关 01｜从零编译 Spark 4.2.0 源码，国内踩坑全记录"
description: "在国内网络下从零编译 Spark 4.2.0 源码、导入 IDEA，以及编译中遇到的坑和解决办法。"
bigdata: "spark"
lesson: "e1"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/spark/01-build-cover.webp"}]]
---

::: info Apache Spark 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Spark 4.2.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。配套实验和代码在 [GitHub](https://github.com/DaemonforY/spark-source-notes)。
:::

<figure class="ai-figure"><img src="/bigdata-img/spark/01-build-cover.webp" alt="Yui 和 Kai 闯关编译 Spark 源码" width="1200" height="800" loading="eager" /><figcaption>Yui 和 Kai 闯关编译 Spark 源码<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> X老师读源码 · 《Spark 源码通关》第 1 期
> 环境：macOS（Apple M5 Pro）、Spark **4.2.0**、JDK 17、Maven 3.9.15

---

读源码的第一步，是把源码**编译**出来。

为什么一定要自己编译？因为只有编译通过了，你才能：

- 在 IDE 里**断点调试**源码，而不是干看
- **修改源码**后立刻验证效果
- 将来给社区提 PR 时，在本地**跑通测试**

听起来就是 `git clone` 加一条 `mvn` 命令的事。实际上，我在国内网络环境下踩了好几个坑：**依赖下载只有 20 KB/s、编译进程被杀了三次**。

这篇文章把完整过程和所有坑都记录下来，**照着做，你可以一次成功**。

## 一、准备环境

### 1. 版本要求（以 4.2.0 为准）

Spark 源码里的 `docs/building-spark.md` 写得很清楚：

> Building Spark using Maven requires Maven 3.9.15 and Java 17/21/25.

| 工具 | 要求 | 说明 |
|---|---|---|
| **JDK** | 17 / 21 / 25 | 官方支持的版本，都是 LTS（长期支持版） |
| **Maven** | 3.9.15 | 不用自己装，下文会讲 |
| **Git** | 任意较新版本 | |
| **磁盘** | 建议预留 15 GB 以上 | 源码 + 编译产物约 2 GB，Maven 依赖还要占不少空间 |
| **内存** | 建议 16 GB 以上 | 编译时 Maven 进程就要 4 GB 以上 |

⚠️ **关于 JDK 版本，有个容易误解的地方**：

Spark 的 `pom.xml` 里用 Maven Enforcer 插件做了版本检查，但规则是 `<requireJavaVersion><version>17</version>`，意思是 **"17 及以上"**。所以如果你用的是 JDK 23 这类非 LTS 版本，检查**不会拦你**，但它不在官方支持列表里，出了问题很难排查。

**建议直接用 JDK 17**，它是 Spark 4.x 编译时的默认目标版本（`pom.xml` 里 `<java.version>17</java.version>`）。

### 2. 多个 JDK 怎么切换

macOS 上可以先看看装了哪些 JDK：

```bash
/usr/libexec/java_home -V
```

不需要改系统默认的 JDK，**只在编译时临时指定**就行：

```bash
export JAVA_HOME=$(/usr/libexec/java_home -v 17)
java -version   # 确认输出是 17
```

> Linux 用户直接把 `JAVA_HOME` 指向 JDK 17 的安装目录即可。

## 二、拉取源码

<figure class="ai-figure"><img src="/bigdata-img/spark/01-build-1.webp" alt="用部分克隆轻装拉取庞大源码" width="960" height="640" loading="lazy" /><figcaption>用部分克隆轻装拉取庞大源码<span>AI 生成配图</span></figcaption></figure>

### 1. 先看有哪些版本

```bash
git ls-remote --tags https://github.com/apache/spark.git 'v4*' \
  | grep -v '\^{}' | awk -F/ '{print $3}' \
  | grep -v -- '-rc\|preview' | sort -V | tail -5
```

写这篇文章时，最新的稳定版是 `v4.2.0`。

### 2. 克隆：用 partial clone 节省时间和空间

Spark 是一个有十多年历史的项目，完整克隆所有历史文件会很大。推荐用 **partial clone（部分克隆）**：

```bash
cd <你的目录>
git clone --filter=blob:none https://github.com/apache/spark.git
```

`--filter=blob:none` 的意思是：**先只下载提交历史和目录结构，文件内容等用到时再下载**。

- 克隆完成后，我这边 `.git` 目录只有约 **286 MB**
- `git log`、`git blame` 照样能用。看很早的历史版本时，Git 会自动按需下载，只是第一次稍慢

> 为什么不用 `--depth 1` 浅克隆？因为读源码时，**"这段代码为什么这么写"的答案往往在 Git 历史里**。浅克隆丢掉了历史，后面做 Git 考古就麻烦了。

### 3. 切换到稳定版本

```bash
cd spark
git checkout -b learn-v4.2.0 v4.2.0
```

基于 tag 新建一个本地分支。学习时做的笔记、注释、修改都在这个分支上，**不会污染 master**。想看最新开发代码时，切回 master 再 `git pull` 就行。

## 三、坑 1：依赖下载只有 20 KB/s

### 现象

第一次编译时，终端里一直在滚动下载进度，速度大约 **20 KB/s**。Spark 依赖的 jar 包非常多，按这个速度，光下载就要好几个小时。

仔细看日志，下载地址是：

```
Downloading from gcs-maven-central-mirror: https://maven-central.storage-download.googleapis.com/...
```

原因是 Spark 的 `pom.xml` 里配置了 Google 的 Maven 中央仓库镜像，在国内访问很慢。

### 解决：换成阿里云镜像（不改全局配置）

网上很多教程会让你直接改 `~/.m2/settings.xml`，但这会影响你电脑上**所有** Maven 项目。

更干净的做法是：**单独给 Spark 写一个配置文件**，编译时用 `-s` 参数指定。

在 Spark 源码根目录下新建 `aliyun-settings.xml`：

```xml
<settings>
  <mirrors>
    <mirror>
      <id>aliyun</id>
      <mirrorOf>*</mirrorOf>
      <name>Aliyun Maven</name>
      <url>https://maven.aliyun.com/repository/public</url>
    </mirror>
  </mirrors>
</settings>
```

`<mirrorOf>*</mirrorOf>` 表示**所有仓库**的请求都走阿里云，包括 `pom.xml` 里配置的那个 Google 镜像。

换完之后，我实测下载速度在 **4.9 MB/s** 左右，是原来的**两百多倍**。

### 小技巧：别让 Git 看到这个文件

这个配置文件是你本地用的，不应该被提交。但如果加到 `.gitignore`，`.gitignore` 本身又会出现在 `git status` 里。

可以加到 **`.git/info/exclude`**。它的作用和 `.gitignore` 一样，但只对你本地生效，不会被提交：

```bash
echo "aliyun-settings.xml" >> .git/info/exclude
```

以后给社区提 PR 时，这个习惯能避免把无关文件带进去。

## 四、开始编译

### 1. 用 `build/mvn`，不用自己装 Maven

Spark 源码里自带了一个脚本 `build/mvn`。它会读取 `pom.xml` 里要求的 Maven 版本（4.2.0 要求 **3.9.15**），检查你本机的 `mvn` 版本对不对，**不对就自动下载正确版本**到 `build/` 目录下。

我本机装的是 Maven 3.9.9，运行时就看到它自动用了下载好的版本：

```
Using `mvn` from path: .../spark/build/apache-maven-3.9.15/bin/mvn
```

所以，**不用为了编译 Spark 去升级你本机的 Maven**。

### 2. 编译命令

```bash
export JAVA_HOME=$(/usr/libexec/java_home -v 17)
export MAVEN_OPTS="-Xss64m -Xmx4g -XX:ReservedCodeCacheSize=1g"

./build/mvn -s aliyun-settings.xml -T 4 -ntp \
  -DskipTests -Phive -Phive-thriftserver \
  clean package
```

每个参数的含义：

| 参数 | 含义 |
|---|---|
| `MAVEN_OPTS` | 给 Maven 进程更大的栈（`-Xss64m`）和堆（`-Xmx4g`）。Scala 编译器很吃内存，不设置可能会报栈溢出或内存不足。官方文档推荐的就是这组参数 |
| `-s aliyun-settings.xml` | 使用刚才的阿里云镜像配置 |
| `-T 4` | 用 4 个线程并行编译没有依赖关系的模块 |
| `-ntp` | 不打印下载进度条（no transfer progress），日志会干净很多 |
| `-DskipTests` | 跳过**运行**测试。注意：测试代码仍然会被**编译**，这一点后面会用到 |
| `-Phive -Phive-thriftserver` | 开启 Hive 支持。后面学 Spark SQL 时会用到，建议一开始就带上 |

> 💡 **一个容易混淆的点**：`-DskipTests` 只跳过运行测试，`-Dmaven.test.skip=true` 才是连测试代码都不编译。但 Spark 的一些模块依赖其他模块的测试 jar（test-jar），用后者可能导致编译失败。**用 `-DskipTests` 就好**。

## 五、坑 2：编译进程被杀了三次

<figure class="ai-figure"><img src="/bigdata-img/spark/01-build-2.webp" alt="镜像加速与限内存解决编译卡点" width="960" height="640" loading="lazy" /><figcaption>镜像加速与限内存解决编译卡点<span>AI 生成配图</span></figcaption></figure>

### 现象

依赖问题解决后，我以为一切顺利了。结果编译到一半，进程直接退出，日志最后一行是：

```
./build/mvn: line 173: 98353 Killed: 9    "${MVN_BIN}" "$@"
```

退出码分别出现过 **137** 和 **143**。

### 先看懂退出码

在 Unix 系统里，进程被信号终止时，退出码 = **128 + 信号编号**：

| 退出码 | 信号 | 含义 | 常见原因 |
|---|---|---|---|
| **137** | 9（SIGKILL） | 被强制杀死，进程无法拦截 | Linux 的 OOM Killer、容器内存超限、被手动 `kill -9` |
| **143** | 15（SIGTERM） | 被要求终止 | 被其他程序或用户结束、终端关闭、超时机制 |
| 130 | 2（SIGINT） | 被中断 | 你按了 `Ctrl+C` |

**看到 137，第一反应应该是检查是不是内存不够。**

### 排查过程

1. **日志里有没有 `OutOfMemoryError`？** 没有。如果是 Maven 的 JVM 堆不够，会抛 Java 异常，而不是被 `Killed: 9`
2. **日志里有没有 `[ERROR]` 编译错误？** 没有
3. **系统内存够不够？** 被杀后马上查看，系统空闲内存还有 44%，不像是系统内存不足
4. **系统日志里有没有内存压力杀进程的记录？** 没有



结论：**不是内存问题，也不是编译问题，进程是被外部终止的。**我当时是在开发工具内嵌的终端环境里跑的编译，这类环境可能会回收长时间运行的进程。

**建议**：编译 Spark 这种耗时长的任务，在**系统自带的独立终端**里运行；远程服务器上则用 `tmux` 或 `nohup`，避免 SSH 断开导致进程被终止。

### 真正有用的技巧：断点续编，不用从头来

进程被杀时，39 个模块已经进行到第 28 个，大部分模块都编译好了。如果重新 `clean package`，这些全白干了。

Maven 有一个参数 **`-rf`（resume from）**，可以**从指定模块继续编译**：

```bash
./build/mvn -s aliyun-settings.xml -T 2 -ntp \
  -DskipTests -Phive -Phive-thriftserver \
  package -rf :spark-sql_2.13
```

注意两点：

1. **去掉 `clean`**，否则前面编译好的产物会被删掉
2. `-rf` 后面跟的是模块的 **artifactId**，格式是 `:模块名`。Spark 的模块名带 Scala 版本后缀，比如 `spark-sql_2.13`、`spark-core_2.13`

**怎么知道该从哪个模块继续？** 看日志里最后一个 `Building` 的模块：

```bash
grep -E "Building .*\[[0-9]+/[0-9]+\]" build.log | tail -1
# [INFO] Building Spark Project SQL 4.2.0     [28/39]
```

然后去这个模块的 `pom.xml` 里找 `<artifactId>`，比如 `sql/core/pom.xml` 里就是 `spark-sql_2.13`。

> ⚠️ 用了 `-T` 并行编译时，日志里 `[28/39]` 这个编号是模块**开始编译的顺序**，而 `-rf` 是按 Maven 的**依赖顺序（Reactor 顺序）**往后续编的。所以续编时，部分在并行中已经编译过的模块可能会再编一遍。这不影响结果，只是多花一点时间。

> 续编时我把 `-T 4` 降到了 `-T 2`，同时把 `-Xmx` 调到 6g。Spark SQL 是整个项目里最大的模块，测试代码也最多，降低并行度可以减少峰值资源占用，更稳。

续编结果：

```
[INFO] Spark Project SQL .................................. SUCCESS [04:23 min]
[INFO] Spark Project ML Library ........................... SUCCESS [01:42 min]
[INFO] Spark Project Hive ................................. SUCCESS [01:21 min]
...
[INFO] Spark Project Assembly ............................. SUCCESS [  5.689 s]
[INFO] BUILD SUCCESS
[INFO] Total time:  08:09 min (Wall Clock)
```

在依赖基本都下载好的情况下，我这台 M5 Pro 上的编译时间合计约 **20 分钟**（中断前约 12 分钟，续编约 8 分钟）。从这里也能看出 **Spark SQL 是最重的模块**，单它一个就要 4 分多钟。

## 六、验证：编译出来的 Spark 能用吗

编译成功不代表能运行。用自己编译出来的 `spark-shell` 跑一个小任务：

```bash
./bin/spark-shell --master "local[2]"
```

进入后执行：

```scala
spark.range(1, 101).selectExpr("sum(id)").first().getLong(0)
// 输出：5050

spark.version
// 输出：4.2.0
```

1 加到 100 等于 5050，版本号是 4.2.0，说明一切正常。🎉

也可以直接跑官方示例：

```bash
./bin/run-example SparkPi 10
```

编译产物在哪里？

- 各模块的 jar：`<模块>/target/scala-2.13/` 下，例如 `core/target/scala-2.13/spark-core_2.13-4.2.0.jar`
- 运行时用到的全部 jar：`assembly/target/scala-2.13/jars/`，我这里一共 **241 个**

## 七、导入 IDEA（为下一期调试做准备）

1. 安装 **Scala 插件**：Settings → Plugins → 搜索 `Scala`
2. File → Open → 选择 Spark 根目录下的 **`pom.xml`** → Open as Project
3. Project Structure → SDK 选 **JDK 17**
4. Settings → Build Tools → Maven：
   - User settings file 勾选 Override，选择刚才的 `aliyun-settings.xml`
   - Importing 里的 JDK 也选 17
5. 右侧 Maven 面板 → Profiles，勾选 `hive` 和 `hive-thriftserver`（和命令行编译保持一致）
6. 等待索引完成，第一次需要一些时间

下一期，我们就用这个环境，**设 27 个断点，跟踪一个 Spark Job 从提交到执行的全过程**。

## 八、常见问题 FAQ

**Q1：Windows 能编译吗？**
可以，但建议用 WSL2，Spark 的构建脚本和测试对类 Unix 环境支持最好。

**Q2：只想读源码，不编译行不行？**
能看，但不建议。没编译就没法断点调试，IDE 的代码跳转也可能不完整，读复杂逻辑时会很痛苦。

**Q3：修改了某个模块的代码，要全部重新编译吗？**
不用。比如只改了 core：

```bash
./build/mvn -s aliyun-settings.xml -DskipTests -pl :spark-core_2.13 package
```

日常开发跑单测时，Spark 官方更推荐用 **sbt**，增量编译快很多（后面讲到提 PR 时会细说）：

```bash
./build/sbt "core/testOnly org.apache.spark.scheduler.DAGSchedulerSuite"
```

**Q4：编译时报 `StackOverflowError`？**
检查 `MAVEN_OPTS` 里有没有 `-Xss64m`。Scala 编译器处理复杂类型推导时需要很深的调用栈。

**Q5：为什么选 4.2.0 而不是 master？**
master 在持续变化，今天的行号明天可能就变了。学习时**锁定一个稳定版本**，笔记和文章里的源码位置才能长期有效。等需要给社区提 PR 时，再基于 master 开发。

---

## 一图总结：Spark 源码编译避坑清单

```
✅ JDK 用 17（官方支持 17/21/25；23 等非 LTS 版本不在支持列表）
✅ 克隆用 --filter=blob:none（保留历史，体积小）
✅ 基于 tag 建本地分支：git checkout -b learn-v4.2.0 v4.2.0
✅ 单独写 aliyun-settings.xml，用 -s 指定（不改全局配置）
✅ 用 ./build/mvn（自动下载正确的 Maven 版本）
✅ 设置 MAVEN_OPTS="-Xss64m -Xmx4g ..."
✅ 用 -DskipTests，不要用 -Dmaven.test.skip=true
✅ 在独立终端 / tmux 里编译
✅ 被中断了？去掉 clean，用 -rf :模块名 断点续编
✅ 编译完用 spark-shell 跑一下，验证 sum = 5050
```

---

**留个思考题：**

文中提到，`-DskipTests` 会跳过运行测试，但**测试代码依然会被编译**。那么问题来了：Spark 为什么要依赖其他模块的"测试 jar"？什么样的代码会被放进 test-jar 里共享？


**下一期**：《源码通关 02：27 个断点，跟踪一个 Spark Job 的一生》

—— X老师读源码
:::
