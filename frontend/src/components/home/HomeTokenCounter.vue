<template>
  <div class="token-counter" data-testid="home-token-counter" :class="{ ready: total > 0 }">
    <span class="lbl">TOKENS PROCESSED</span>
    <span class="val">
      <b :class="{ bump }" data-testid="home-token-counter-value">{{ total > 0 ? formatCount(shown) : '—' }}</b>
      <span class="live" aria-hidden="true"></span>
      <span class="pops" aria-hidden="true">
        <i v-for="p in pops" :key="p.id" class="pop" data-testid="home-token-counter-pop">+{{ p.text }}</i>
      </span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { getUsageTotals } from '@/api/pricing'
import { formatCount, splitIncrease } from '@/utils/liveCounter'
import { prefersReducedMotion } from './motion/tier'

/**
 * Every token members' requests have used, live: counted up on arrival, then polled; each real
 * increase plays out as a few steps with a 「+N」 popping off the top-right corner.
 */
const POLL_MS = 6000
const ROLL_MS = 2400

const shown = ref(0)
const total = ref(0)
const bump = ref(false)
const pops = ref<{ id: number; text: string }[]>([])
let popId = 0
let aim = 0 // where the number is heading; each step moves it on, so steps never undercut each other
let poll: ReturnType<typeof setInterval> | undefined
let raf = 0
const timers = new Set<ReturnType<typeof setTimeout>>()
const later = (fn: () => void, ms: number) => {
  const id = setTimeout(() => { timers.delete(id); fn() }, ms)
  timers.add(id)
}

function rollTo(to: number, ms: number) {
  cancelAnimationFrame(raf)
  const from = shown.value, t0 = performance.now()
  const tick = (now: number) => {
    const k = Math.min(1, (now - t0) / ms)
    shown.value = from + (to - from) * (1 - Math.pow(2, -10 * k))
    if (k < 1) raf = requestAnimationFrame(tick)
    else shown.value = to
  }
  raf = requestAnimationFrame(tick)
}

function play(delta: number) {
  const steps = splitIncrease(delta)
  const gap = (POLL_MS * 0.7) / Math.max(1, steps.length)
  steps.forEach((step, i) =>
    later(() => {
      aim += step
      rollTo(aim, 500)
      const id = ++popId
      pops.value = [...pops.value, { id, text: formatCount(step) }]
      bump.value = false
      requestAnimationFrame(() => (bump.value = true))
      later(() => (pops.value = pops.value.filter((p) => p.id !== id)), 1500)
    }, i * gap)
  )
}

async function refresh(first = false) {
  if (!first && document.hidden) return
  try {
    const next = Number((await getUsageTotals()).total_tokens) || 0
    if (next <= 0) return
    if (first || total.value === 0) {
      total.value = aim = next
      if (prefersReducedMotion()) shown.value = next
      else rollTo(next, ROLL_MS)
    } else if (next > total.value) {
      const delta = next - total.value
      total.value = next
      if (prefersReducedMotion()) shown.value = aim = next
      else play(delta)
    } else if (next < total.value) {
      total.value = aim = next // a recount came in lower: just show it
      cancelAnimationFrame(raf)
      shown.value = next
    }
  } catch {
    // keep what is on screen
  }
}

onMounted(() => {
  void refresh(true)
  poll = setInterval(() => void refresh(), POLL_MS)
})

onBeforeUnmount(() => {
  clearInterval(poll)
  cancelAnimationFrame(raf)
  timers.forEach(clearTimeout)
})

defineExpose({ refresh })
</script>

<style scoped>
.token-counter {
  display: inline-flex;
  align-items: baseline;
  gap: 12px;
  font: 600 11px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: 0.14em;
  color: #6c8099;
}
.val {
  position: relative;
  display: inline-flex;
  align-items: baseline;
}
.val b {
  font: 700 20px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: 0.02em;
  color: #eef6ff;
  font-variant-numeric: tabular-nums;
}
.val b.bump {
  animation: bump 0.5s ease-out;
}
@keyframes bump {
  0% { color: #fff; text-shadow: 0 0 14px #5ee7ff; }
  100% { color: #eef6ff; text-shadow: none; }
}
.live {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-left: 8px;
  border-radius: 50%;
  background: #5ee7ff;
  box-shadow: 0 0 10px #5ee7ff;
  align-self: center;
  animation: live 1.6s ease-in-out infinite;
}
@keyframes live {
  50% { opacity: 0.25; }
}
.pops {
  position: absolute;
  right: -6px;
  top: -6px;
  width: 0;
  height: 0;
}
.pop {
  position: absolute;
  left: 0;
  bottom: 0;
  font: 700 13px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  font-style: normal;
  letter-spacing: 0;
  color: #5ee7ff;
  text-shadow: 0 0 10px rgba(94, 231, 255, 0.8);
  white-space: nowrap;
  animation: pop 1.5s cubic-bezier(0.2, 0.7, 0.2, 1) forwards;
  pointer-events: none;
}
@keyframes pop {
  0% { opacity: 0; transform: translate(0, 6px) scale(0.7); }
  18% { opacity: 1; transform: translate(0, -4px) scale(1.08); }
  70% { opacity: 1; transform: translate(0, -16px) scale(1); }
  100% { opacity: 0; transform: translate(0, -26px) scale(1); }
}
@media (prefers-reduced-motion: reduce) {
  .val b.bump, .live, .pop { animation: none; }
  .pop { display: none; }
}
</style>
