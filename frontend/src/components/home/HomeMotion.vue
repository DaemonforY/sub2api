<template>
  <div ref="host" class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true"></div>
</template>

<script setup lang="ts" generic="K extends MotionKind">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { MotionContexts, MotionKind } from './motion'
import { detectMotion } from './motion/tier'

/**
 * A WebGL motion layer behind one home-page screen. three.js and the scene load only on devices
 * that get motion, after the page's own content has painted; otherwise this stays an empty box and
 * the section's CSS background shows.
 */
const props = defineProps<{ kind: K; context?: () => MotionContexts[K] }>()
const emit = defineEmits<{ started: [] }>()
const host = ref<HTMLElement | null>(null)
let stop: (() => void) | undefined
let gone = false

onMounted(() => {
  const opts = detectMotion()
  if (!opts || !host.value) return
  const start = async () => {
    try {
      const { mountMotion } = await import('./motion')
      if (gone || !host.value) return
      stop = mountMotion(props.kind, host.value, opts, (props.context?.() ?? undefined) as MotionContexts[K])
      emit('started')
    } catch {
      // no motion is fine; the static background stays
    }
  }
  const idle = (window as Window & { requestIdleCallback?: (cb: () => void, o?: { timeout: number }) => number }).requestIdleCallback
  if (idle) idle(() => void start(), { timeout: 1200 })
  else setTimeout(() => void start(), 300)
})

onBeforeUnmount(() => {
  gone = true
  stop?.()
})
</script>
