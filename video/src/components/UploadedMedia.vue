<template>
  <!-- An uploaded work: the SVG is only ever shown through <img> (no script runs there). -->
  <div class="absolute inset-0 bg-ink-950">
    <img v-if="media.kind === 'svg'" :src="media.url" alt="" class="h-full w-full bg-white object-contain" draggable="false" />
    <template v-else-if="card">
      <img v-if="media.poster && !playing" :src="media.poster" alt="" class="absolute inset-0 h-full w-full object-contain" loading="lazy" />
      <video v-if="active || !media.poster" ref="video" :src="media.url" class="absolute inset-0 h-full w-full object-contain" muted loop playsinline :preload="active ? 'auto' : 'metadata'" @playing="playing = true"></video>
    </template>
    <video v-else ref="video" :src="media.url" :poster="media.poster || undefined" class="h-full w-full object-contain" controls playsinline preload="metadata"></video>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  media: { type: Object, required: true },
  /** Gallery card: muted, plays while `active` (hovered). */
  card: { type: Boolean, default: false },
  active: { type: Boolean, default: false }
})
const video = ref(null)
const playing = ref(false)

watch(
  () => props.active,
  async (on) => {
    if (!props.card || props.media.kind === 'svg') return
    await nextTick()
    const v = video.value
    if (!v) return
    if (on) {
      v.play().catch(() => {})
    } else {
      v.pause()
      v.currentTime = 0
      playing.value = false
    }
  }
)
</script>
