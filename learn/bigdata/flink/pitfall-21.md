---
title: "踩坑实验室 #21：照着网上的教程调网络缓冲区，配了不报错，也不生效"
description: "Flink 2.0 删除的四个网络缓冲区配置项，在 2.3 里设成 abc 也能正常运行。"
bigdata: "flink"
head: [["meta", {"property": "og:image", "content": "https://hivegpt.cn/learn/bigdata-img/flink/pitfall-21-cover.webp"}]]
---

# 踩坑实验室 #21：照着网上的教程调网络缓冲区，配了不报错，也不生效

::: info Apache Flink 源码学习
作者 X老师（[DaemonforY](https://github.com/DaemonforY)），Flink 2.3.0 源码，按 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh-hans) 发布。
:::

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-21-cover.webp" alt="Yui和Kai面对失效配置与自动调节的网络缓冲区" width="1200" height="800" loading="eager" /><figcaption>Yui和Kai面对失效配置与自动调节的网络缓冲区<span>AI 生成配图</span></figcaption></figure>

::: v-pre
> 基于 Flink 2.3.0 本机实测（核对日期 2026-10-04）。对应课程：第四讲第三节、第九讲第五节（候选任务 1）。配套示例：`flink-notes/demos/src/main/java/study/network/RemovedOptionDemo.java`

## 现象

Flink 1.x 有四个常用的网络缓冲区调优参数，很多调背压、调网络的教程都在讲：

- `taskmanager.network.memory.buffers-per-channel`
- `taskmanager.network.memory.floating-buffers-per-gate`
- `taskmanager.network.memory.max-buffers-per-channel`
- `taskmanager.network.memory.max-overdraft-buffers-per-gate`

在 Flink 2.x 里照着教程配上它们，作业正常运行，看不出任何问题，但配置并没有生效。

## 复现

示例 `RemovedOptionDemo`：作业 `source（1000 条）→ rebalance → map → sink`，并行度 2，MiniCluster 启动 2 个 TaskManager（走网络）。

| 模式 | 配置 | 结果 |
|---|---|---|
| `removed` | 四个已删除的配置项都设成 `abc` | 作业正常结束 |
| `existing` | `taskmanager.network.memory.buffer-debloat.target`（仍然存在）设成 `abc` | 作业立刻失败，`NumberFormatException` |

```
# removed
>>> mode=removed, config={taskmanager.network.memory.max-buffers-per-channel=abc, taskmanager.network.memory.floating-buffers-per-gate=abc, taskmanager.network.memory.max-overdraft-buffers-per-gate=abc, minicluster.number-of-taskmanagers=2, taskmanager.network.memory.buffers-per-channel=abc}
>>> job finished normally

# existing
>>> mode=existing, config={taskmanager.network.memory.buffer-debloat.target=abc, minicluster.number-of-taskmanagers=2}
>>> job failed: IllegalArgumentException
>>> root cause: java.lang.NumberFormatException: text does not start with a number, and is not a valid ISO-8601 duration format: abc
```

`removed` 模式的日志里一共 5 条 WARN，都是 token、Web 日志文件之类和配置无关的内容，没有一条提到这四个配置项。两个模式各跑了一次。

![四个已删除的配置项设成 abc 照常运行，仍存在的配置项立刻失败（本机实测）](/bigdata-img/src/content-plan/assets/png/ep21/xhs-P3.webp)

## 原因

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-21-1.webp" alt="失效配置像断开的开关，网络数据仍沿管道正常流动" width="960" height="640" loading="lazy" /><figcaption>失效配置像断开的开关，网络数据仍沿管道正常流动<span>AI 生成配图</span></figcaption></figure>

这四个配置项在 **Flink 2.0 已经删除**，release notes 的 "List of removed configuration options" 里有记录（`docs/content/release-notes/flink-2.0.md:1546`，四个 key 分别在 `:1622`、`:1624`、`:1627`、`:1628`）。在 2.3.0 的 `flink-core`、`flink-runtime` 源码里 grep 这四个 key，结果为 0。

它们在 2.x 里变成了代码里的常量：独占 2、浮动 8、每个子分区最多 10、透支最多 20（`NettyShuffleEnvironmentConfiguration.java:325`、`:326`、`:332`、`:401`）。

两个模式的对比说明了问题：一个配置项如果真的有人读取，非法值会在解析时报错，就像 `existing` 模式那样；四个已删除的配置项设成 `abc` 也照样跑完，说明根本没有人读它们。所以它们是被**静默忽略**的：不报错，也不生效，按老教程调成什么值都一样。

文档也加深了这种误解。截至本地的 2.3.0 源码，官方文档里仍有多处在讲这四个配置项：`network_mem_tuning.md` 中英文各 7 行，`adaptive_batch.md` 各 1 行，`batch_shuffle.md` 各 2 行。`network_mem_tuning.md:194`（中文版在 `:182`）还链接到配置页的 `#taskmanager-network-memory-buffers-per-channel` 锚点，但 `docs/layouts/shortcodes/generated/` 下自动生成的配置表里已经搜不到这四个 key。这几处属于还没跟上 2.0 删除的更新，文档的其他内容依然有用。

![2.0 release notes 中的删除记录，以及 2.x 里写死的常量](/bigdata-img/src/content-plan/assets/png/ep21/xhs-P4.webp)

## 怎么解决

<figure class="ai-figure"><img src="/bigdata-img/flink/pitfall-21-2.webp" alt="自动缓冲区随数据流量伸缩，持续平稳承接网络数据" width="960" height="640" loading="lazy" /><figcaption>自动缓冲区随数据流量伸缩，持续平稳承接网络数据<span>AI 生成配图</span></figcaption></figure>

- 升级到 2.x 时，检查配置里有没有这四个配置项，有的话可以删掉，它们不会起作用。
- 2.x 的思路是不再手动调缓冲区的个数，而是开启 Buffer Debloating，自动调整每个缓冲区的大小。它默认关闭，需要设置 `taskmanager.network.memory.buffer-debloat.enabled: true`（`TaskManagerOptions.java:492`）。原理和实测见第四讲第八节和踩坑实验室 #05。

几点注意：

- 本文只验证了"在代码里设置配置、MiniCluster 运行"这一种方式；用 `config.yaml` 部署到集群时会不会有提示，没有验证；
- 缓冲区个数虽然写死了，但每个缓冲区装多少可以由 Buffer Debloating 自动调整，不等于网络缓冲区没法调；
- 本文只讨论这四个配置项，不代表 1.x 的调优教程整体都失效了。

## 记住这几点

- `buffers-per-channel` 等四个网络缓冲区参数在 Flink 2.0 已删除，2.x 里配了不报错，也不生效。
- 配置项是否真的被读取，可以用非法值试一下：被读取的会立刻报错。
- 2.x 的网络调优改用 Buffer Debloating，它默认关闭。
:::
