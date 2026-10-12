<template>
  <section ref="section" class="live-stats relative min-h-[100svh] overflow-hidden bg-[#050a14] text-white" data-testid="home-live-stats">
    <HomeMotion kind="hive" />
    <div class="fade pointer-events-none absolute inset-x-0 top-0 z-[1] h-3/5"></div>
    <div class="relative z-[2] mx-auto max-w-6xl px-4 pt-20 sm:px-6 sm:pt-28">
      <p class="font-mono text-[11px] font-semibold uppercase tracking-[0.22em] text-cyan-300">{{ t('home.v3.motion.stats.eyebrow') }}</p>
      <h2 class="mt-3 max-w-[13em] text-[32px] font-extrabold leading-tight tracking-tight sm:text-6xl">
        <template v-if="tokens">{{ t('home.v3.motion.stats.titlePre') }}<span class="num">{{ tokenText }}</span>{{ t('home.v3.motion.stats.titlePost') }}</template>
        <template v-else>{{ t('home.v3.motion.stats.titlePlain') }}</template>
      </h2>
      <div class="mt-6 flex flex-wrap gap-x-9 gap-y-3 text-base text-slate-400">
        <span v-if="requests"><b class="mr-1.5 text-[22px] tabular-nums text-slate-100">{{ requestText }}</b>{{ t('home.v3.motion.stats.requests') }}</span>
        <span><b class="mr-1.5 text-[22px] text-slate-100">GPT-6</b>· gpt-image-2 · Codex</span>
        <span><b class="mr-1.5 text-[22px] text-slate-100">{{ t('home.v3.motion.stats.minTopup') }}</b>{{ t('home.v3.motion.stats.payg') }}</span>
      </div>
      <div class="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-[13px] text-slate-500">
        <span class="whitespace-nowrap"><i class="dia bg-[#5ee7ff]"></i>{{ t('home.v3.motion.stats.chat') }}</span>
        <span class="whitespace-nowrap"><i class="dia bg-[#ffbe6b]"></i>{{ t('home.v3.motion.stats.image') }}</span>
        <span class="whitespace-nowrap"><i class="dia bg-[#4d8dff]"></i>{{ t('home.v3.motion.stats.code') }}</span>
        <span>{{ t('home.v3.motion.stats.legend') }}</span>
      </div>
    </div>
    <p class="absolute bottom-5 left-4 z-[2] text-xs text-slate-500 sm:left-auto sm:right-8">{{ t('home.v3.motion.stats.source') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeMotion from './HomeMotion.vue'
import { prefersReducedMotion } from './motion/tier'

/** 数据在流动: the honeycomb floor with the real totals, counted up when the screen comes into view. */
const props = defineProps<{ totalTokens?: number; totalRequests?: number }>()
const { t, locale } = useI18n()
const section = ref<HTMLElement | null>(null)
const k = ref(0) // 0 → 1 count-up
const tokens = computed(() => props.totalTokens || 0)
const requests = computed(() => props.totalRequests || 0)
const zh = computed(() => String(locale.value).startsWith('zh'))
const tokenText = computed(() => (zh.value ? `${((tokens.value * k.value) / 1e8).toFixed(1)} 亿` : `${((tokens.value * k.value) / 1e9).toFixed(2)}B`))
const requestText = computed(() => (zh.value ? `${((requests.value * k.value) / 1e4).toFixed(2)} 万` : Math.round(requests.value * k.value).toLocaleString('en-US')))

let io: IntersectionObserver | null = null
let raf = 0
onMounted(() => {
  if (prefersReducedMotion() || typeof IntersectionObserver === 'undefined') { k.value = 1; return }
  io = new IntersectionObserver((entries) => {
    if (!entries.some((e) => e.isIntersecting)) return
    io?.disconnect()
    const t0 = performance.now()
    const tick = (now: number) => {
      const p = Math.min(1, (now - t0) / 1600)
      k.value = 1 - Math.pow(1 - p, 3)
      if (p < 1) raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
  }, { threshold: 0.3 })
  if (section.value) io.observe(section.value)
})
onBeforeUnmount(() => { io?.disconnect(); cancelAnimationFrame(raf) })
</script>

<style scoped>
.fade {
  background: linear-gradient(to bottom, rgba(5, 10, 20, 0.85), rgba(5, 10, 20, 0.3) 55%, transparent);
}
.num {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  background: linear-gradient(100deg, #eaf7ff, #5ee7ff 45%, #3b82f6);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.dia {
  display: inline-block;
  width: 9px;
  height: 9px;
  margin-right: 7px;
  border-radius: 2px;
  transform: rotate(45deg);
  vertical-align: 1px;
}
</style>
