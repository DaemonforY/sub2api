<script setup lang="ts">
// Above a lesson: track, position, time, and what you will be able to do.
import { computed } from 'vue'
import { useData, withBase } from 'vitepress'
import { findLesson } from '../tracks'
import { progress } from '../api'

const { frontmatter } = useData()
const found = computed(() => (frontmatter.value.lesson ? findLesson(String(frontmatter.value.lesson)) : null))
const goals = computed<string[]>(() => (Array.isArray(frontmatter.value.goals) ? frontmatter.value.goals : []))
const done = computed(() => !!(found.value && progress.completed[found.value.lesson.id]))
</script>

<template>
  <div v-if="found" class="lesson-head">
    <div class="lesson-crumb">
      <a :href="withBase(`/${found.track.id}/`)">{{ found.track.letter }} · {{ found.track.title }}</a>
      <span>第 {{ found.index + 1 }} 课</span>
      <span>⏱ {{ found.lesson.minutes }} 分钟</span>
      <span v-if="done" class="lesson-done">✓ 已完成</span>
    </div>
    <h1 class="lesson-title">{{ frontmatter.title }}</h1>
    <div v-if="goals.length" class="lesson-goals">
      <b>学完你能</b>
      <ul>
        <li v-for="g in goals" :key="g">{{ g }}</li>
      </ul>
    </div>
  </div>
</template>
