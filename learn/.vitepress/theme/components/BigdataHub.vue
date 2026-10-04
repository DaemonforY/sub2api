<script setup lang="ts">
// /bigdata/: the three source-reading series.
import { withBase } from 'vitepress'
import sidebar from '../../bigdata-sidebar.json'

const META: Record<string, { id: string; color: string; letter: string; what: string }> = {
  'Apache Spark': { id: 'spark', color: 'linear-gradient(135deg,#fb923c,#ea580c)', letter: 'S', what: '批处理引擎：调度、Shuffle、内存、Catalyst、Codegen、AQE' },
  'Apache Flink': { id: 'flink', color: 'linear-gradient(135deg,#a78bfa,#7c3aed)', letter: 'F', what: '流处理引擎：执行链路、Checkpoint、状态、背压、Watermark' },
  'Apache Paimon': { id: 'paimon', color: 'linear-gradient(135deg,#38bdf8,#0284c7)', letter: 'P', what: '流式湖仓：LSM 合并、提交、读路径、Changelog、Catalog' },
}

const topics = (sidebar as { text: string; items: { text: string; link: string }[] }[]).map((g) => ({
  ...META[g.text],
  title: g.text,
  count: g.items.length - 1,
}))
</script>

<template>
  <div class="home-tracks bigdata-hub">
    <a v-for="t in topics" :key="t.id" class="home-track" :href="withBase(`/bigdata/${t.id}/`)">
      <div class="home-track-icon" :style="{ background: t.color }">{{ t.letter }}</div>
      <h3>{{ t.title }}</h3>
      <p>{{ t.what }}</p>
      <div class="home-track-meta"><span>{{ t.count }} 篇</span><span>源码 + 实验</span></div>
      <div class="home-track-project">看目录，从第一篇开始 →</div>
    </a>
  </div>
</template>
