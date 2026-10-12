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
 *
 * `eager` loads right away instead of waiting for the browser to be idle (the first screen).
 * With `deadline` (ms), a scene that hasn't loaded by then is dropped and 'fallback' fires, so a
 * slow connection gets the plain page instead of something appearing late; 'fallback' also fires
 * at once when the device gets no motion.
 */
const props = defineProps<{ kind: K; context?: () => MotionContexts[K]; eager?: boolean; deadline?: number }>()
const emit = defineEmits<{ started: []; fallback: [] }>()
const host = ref<HTMLElement | null>(null)
let stop: (() => void) | undefined
let gone = false
let late = false
let timer: ReturnType<typeof setTimeout> | undefined

function giveUp() {
  if (late) return
  late = true
  clearTimeout(timer)
  emit('fallback')
}

onMounted(() => {
  const opts = detectMotion()
  if (!opts || !host.value) return giveUp()
  if (props.deadline) timer = setTimeout(giveUp, props.deadline)
  const start = async () => {
    try {
      const { mountMotion } = await import('./motion')
      if (gone || late || !host.value) return
      clearTimeout(timer)
      stop = mountMotion(props.kind, host.value, opts, (props.context?.() ?? undefined) as MotionContexts[K])
      emit('started')
    } catch {
      giveUp() // no motion is fine; the static background stays
    }
  }
  const idle = (window as Window & { requestIdleCallback?: (cb: () => void, o?: { timeout: number }) => number }).requestIdleCallback
  if (props.eager) void start()
  else if (idle) idle(() => void start(), { timeout: 1200 })
  else setTimeout(() => void start(), 300)
})

onBeforeUnmount(() => {
  gone = true
  clearTimeout(timer)
  stop?.()
})
</script>
