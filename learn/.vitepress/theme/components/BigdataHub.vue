<script setup lang="ts">
// /bigdata/: the three source-reading tracks (Spark / Flink / Paimon) with progress.
import { onMounted } from 'vue'
import { withBase } from 'vitepress'
import { loadProgress, progress } from '../api'
import { findTrack, trackHref, type Track } from '../tracks'

const WHAT: Record<string, string> = {
  e: '批处理引擎：调度、Shuffle、内存、Catalyst、Codegen、AQE',
  f: '流处理引擎：执行链路、Checkpoint、状态、背压、Watermark',
  g: '流式湖仓：LSM 合并、提交、读路径、Changelog、Catalog',
}
const topics = ['e', 'f', 'g'].map((id) => findTrack(id)).filter((t): t is Track => !!t)
const done = (t: Track) => t.lessons.filter((l) => progress.completed[l.id]).length
const hours = (t: Track) => Math.round(t.lessons.reduce((n, l) => n + l.minutes, 0) / 6) / 10

onMounted(() => void loadProgress())
</script>

<template>
  <div class="home-tracks bigdata-hub">
    <a v-for="t in topics" :key="t.id" class="home-track" :href="withBase(trackHref(t))" data-testid="bigdata-track">
      <div class="home-track-icon" :style="{ background: t.color }">{{ t.letter }}</div>
      <h3>{{ t.title }}</h3>
      <p>{{ WHAT[t.id] }}</p>
      <div class="home-track-meta"><span>{{ t.lessons.length }} 课</span><span>约 {{ hours(t) }} 小时</span><span>结业证书</span></div>
      <div class="home-track-foot">
        <div class="home-bar"><b :style="{ width: (done(t) / t.lessons.length) * 100 + '%' }"></b></div>
        <span class="runbox-note">{{ done(t) }} / {{ t.lessons.length }}</span>
      </div>
    </a>
  </div>
</template>
