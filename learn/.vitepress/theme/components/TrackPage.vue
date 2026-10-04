<script setup lang="ts">
// A track's page: who it is for, the lessons with progress, the final project.
import { computed, onMounted } from 'vue'
import { withBase } from 'vitepress'
import { tracks } from '../tracks'
import { loadProgress, progress } from '../api'

const props = defineProps<{ id: string }>()
const track = computed(() => tracks.find((t) => t.id === props.id)!)
const done = computed(() => track.value.lessons.filter((l) => progress.completed[l.id]).length)
const start = computed(() => track.value.lessons.find((l) => l.ready && !progress.completed[l.id]) || track.value.lessons[0])
onMounted(() => void loadProgress())
</script>

<template>
  <div class="track-page">
    <div class="track-intro">
      <div class="home-track-icon" :style="{ background: track.color }">{{ track.letter }}</div>
      <div>
        <p>{{ track.tagline }}</p>
        <p class="runbox-note">适合：{{ track.audience }}</p>
      </div>
    </div>
    <div class="track-progress">
      <div class="home-bar"><b :style="{ width: (done / track.lessons.length) * 100 + '%' }"></b></div>
      <span class="runbox-note">已完成 {{ done }} / {{ track.lessons.length }} 课</span>
      <a v-if="start" class="runbox-btn" :href="withBase(`/${track.id}/${start.id}`)">{{ done ? '继续学习' : '开始学习' }}</a>
    </div>
    <ol class="track-lessons">
      <li v-for="(l, i) in track.lessons" :key="l.id" :class="{ soon: !l.ready, done: progress.completed[l.id] }">
        <span class="track-num">{{ progress.completed[l.id] ? '✓' : i + 1 }}</span>
        <a v-if="l.ready" :href="withBase(`/${track.id}/${l.id}`)">{{ l.title }}</a>
        <span v-else>{{ l.title }}</span>
        <span class="runbox-note">{{ l.ready ? `${l.minutes} 分钟` : '即将上线' }}</span>
      </li>
    </ol>
    <div class="lesson-goals">
      <b>结业项目</b>
      <p>{{ track.project }}。满足条件后在本页下方领取结业证书。</p>
    </div>
  </div>
</template>
