<script setup lang="ts">
// Diagrams in the imported articles; mermaid loads only on pages that have one.
import { onMounted, ref } from 'vue'

const props = defineProps<{ code: string }>()
const svg = ref('')
const failed = ref(false)
const source = new TextDecoder().decode(Uint8Array.from(atob(props.code), (c) => c.charCodeAt(0)))
let seq = 0

onMounted(async () => {
  try {
    const mermaid = (await import('mermaid')).default
    mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'neutral' })
    const { svg: out } = await mermaid.render(`mmd-${Date.now()}-${seq++}`, source)
    svg.value = out
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <div class="mermaid-box">
    <div v-if="svg" v-html="svg"></div>
    <pre v-else-if="failed"><code>{{ source }}</code></pre>
    <div v-else class="runbox-note">图表加载中…</div>
  </div>
</template>
