<script setup lang="ts">
// Opens the HiveGPT canvas' image page (or video page) with this prompt filled in.
import { ref } from 'vue'
import { CANVAS_SITE } from '../tracks'

const props = defineProps<{ prompt: string; label?: string; page?: 'image' | 'video' }>()
const copied = ref(false)
const href = `${CANVAS_SITE}/${props.page === 'video' ? 'video' : 'image'}?prompt=${encodeURIComponent(props.prompt)}&utm_source=learn`

async function copy() {
  try {
    await navigator.clipboard.writeText(props.prompt)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    // clipboard unavailable
  }
}
</script>

<template>
  <div class="canvas-try">
    <pre class="canvas-prompt">{{ prompt }}</pre>
    <div class="runbox-actions">
      <a class="runbox-btn" :href="href" target="_blank" rel="noopener">{{ label || '在画布里试试' }} ↗</a>
      <button class="runbox-btn ghost" @click="copy">{{ copied ? '已复制' : '复制提示词' }}</button>
    </div>
  </div>
</template>
