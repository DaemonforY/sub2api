<script setup lang="ts">
// 一次只改一处 (学画画): pictures drawn from the same prompt with one sentence changed, side by side.
// `base` is the shared prompt with {} where the changed sentence goes; each item shows its sentence.
// With `guess`, the labels stay hidden until the learner has guessed (click to reveal).
import { computed, ref } from 'vue'
import { withBase } from 'vitepress'

interface Item {
  src: string
  label: string
  change?: string
}

const props = withDefaults(defineProps<{ items: string; base?: string; cols?: number; guess?: boolean; title?: string }>(), { cols: 3 })
const list = computed<Item[]>(() => {
  try {
    return JSON.parse(props.items)
  } catch {
    return []
  }
})
const revealed = ref<Record<number, boolean>>({})
const zoom = ref<number | null>(null)
const baseParts = computed(() => (props.base ? props.base.split('{}') : []))

function open(i: number) {
  if (props.guess && !revealed.value[i]) {
    revealed.value = { ...revealed.value, [i]: true }
    return
  }
  zoom.value = i
}
</script>

<template>
  <div class="cmp" data-testid="compare-grid">
    <div v-if="title" class="cmp-title">{{ title }}</div>
    <p v-if="base" class="cmp-base">
      <span class="cmp-k">同一段提示词</span>{{ baseParts[0] }}<mark>［只改这一句］</mark>{{ baseParts.slice(1).join('') }}
    </p>
    <div class="cmp-grid" :style="{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }">
      <button v-for="(it, i) in list" :key="it.src" class="cmp-card" @click="open(i)">
        <img :src="withBase(it.src)" :alt="guess && !revealed[i] ? `第 ${i + 1} 张` : it.label" loading="lazy" />
        <span v-if="guess && !revealed[i]" class="cmp-label hidden">第 {{ i + 1 }} 张 · 先猜，再点开答案</span>
        <span v-else class="cmp-label"><b>{{ it.label }}</b><small v-if="it.change">{{ it.change }}</small></span>
      </button>
    </div>
    <p class="cmp-foot">图片由 AI 生成（gpt-image-2）。点图片放大{{ guess ? '；猜之前先别看标签' : '' }}。</p>
    <div v-if="zoom !== null" class="cmp-zoom" @click="zoom = null">
      <img :src="withBase(list[zoom].src)" :alt="list[zoom].label" />
      <p><b>{{ list[zoom].label }}</b> {{ list[zoom].change }}</p>
    </div>
  </div>
</template>
