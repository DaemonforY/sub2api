<template>
  <section id="teardown" class="teardown relative min-h-[100svh] overflow-hidden bg-[#050a14] text-white" data-testid="home-teardown">
    <HomeMotion kind="teardown" :context="context" />
    <svg ref="leaders" class="pointer-events-none absolute inset-0 z-[2] h-full w-full overflow-visible" aria-hidden="true"></svg>
    <div ref="labels" class="td-labels pointer-events-none absolute inset-0 z-[3]" aria-hidden="true"></div>
    <div class="relative z-[2] mx-auto max-w-6xl px-4 pt-24 sm:px-6 sm:pt-28">
      <p class="font-mono text-[11px] font-semibold uppercase tracking-[0.22em] text-cyan-300">{{ t('home.v3.motion.teardown.eyebrow') }}</p>
      <h2 class="mt-3 max-w-[9em] text-3xl font-extrabold leading-tight tracking-tight sm:text-5xl">{{ t('home.v3.motion.teardown.title') }}</h2>
      <p class="mt-3 max-w-sm text-sm text-slate-400 sm:text-base">{{ t('home.v3.motion.teardown.subtitle', { site: siteName }) }}</p>
    </div>
    <div class="absolute right-8 top-28 z-[2] hidden text-right font-mono text-[10.5px] font-semibold leading-[1.8] tracking-[0.14em] text-slate-500 md:block">
      <b class="block font-semibold text-slate-100">EXPLODED VIEW</b>GPT-6 ASTRA · GPT-IMAGE-2
    </div>
    <div class="absolute bottom-6 left-4 right-4 z-[2] sm:left-8 sm:right-auto sm:w-[460px]">
      <div class="flex justify-between font-mono text-[10px] font-semibold tracking-[0.14em] text-slate-500">
        <span>{{ t('home.v3.motion.teardown.timeline') }}</span><span ref="time">00:00 / 00:11</span>
      </div>
      <div class="mt-2.5 h-0.5 bg-white/10"><div ref="fill" class="h-full w-0 bg-gradient-to-r from-cyan-300/40 to-cyan-300 shadow-[0_0_10px_rgba(94,231,255,.6)]"></div></div>
      <div ref="phase" class="mt-2 text-xs tracking-wider text-slate-200">{{ phaseNames[0] }}</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeMotion from './HomeMotion.vue'
import { TEARDOWN_PROMPT } from './motion/content'

/**
 * 拆开看看: how one sentence becomes a picture, as an exploded view of five layers (after the phone
 * teardown film). The scene drives the labels, leader lines and timeline in this section.
 */
const props = defineProps<{ siteName: string; imageCostUsd?: number }>()
const { t } = useI18n()
const leaders = ref<SVGSVGElement | null>(null)
const labels = ref<HTMLElement | null>(null)
const time = ref<HTMLElement | null>(null)
const fill = ref<HTMLElement | null>(null)
const phase = ref<HTMLElement | null>(null)

const phaseNames = computed(() => [0, 1, 2, 3, 4, 5].map((i) => t(`home.v3.motion.teardown.phases.p${i}`)))
const layerText = computed(() => [
  { name: t('home.v3.motion.teardown.layers.prompt'), spec: t('home.v3.motion.teardown.layers.promptSpec', { n: [...TEARDOWN_PROMPT].length }) },
  { name: t('home.v3.motion.teardown.layers.tokens'), spec: t('home.v3.motion.teardown.layers.tokensSpec') },
  { name: 'GPT-6 Astra', spec: t('home.v3.motion.teardown.layers.modelSpec') },
  { name: 'gpt-image-2', spec: t('home.v3.motion.teardown.layers.imageSpec') },
  {
    name: t('home.v3.motion.teardown.layers.result'),
    spec: props.imageCostUsd && props.imageCostUsd > 0
      ? t('home.v3.motion.teardown.layers.resultSpec', { price: `$${Math.round(props.imageCostUsd * 100) / 100}` })
      : t('home.v3.motion.teardown.layers.resultSpecPlain'),
  },
])

const context = () => ({
  labels: labels.value!, leaders: leaders.value!, time: time.value!, fill: fill.value!, phase: phase.value!,
  layerText: layerText.value, phaseNames: phaseNames.value, imageUrl: '/home-media/cat.jpg',
})
</script>

<style scoped>
.td-labels :deep(.td-label) {
  position: absolute;
  left: 0;
  top: 0;
  min-width: 136px;
  padding: 7px 10px;
  border-left: 2px solid #5ee7ff;
  background: rgba(5, 12, 24, 0.7);
  backdrop-filter: blur(9px);
  opacity: 0;
  transition: border-color 0.2s;
}
.td-labels :deep(.idx) {
  font: 600 9px/1 ui-monospace, 'SF Mono', Menlo, Consolas, monospace;
  letter-spacing: 0.12em;
  color: #5ee7ff;
}
.td-labels :deep(.nm) {
  margin-top: 4px;
  font-size: 13px;
  white-space: nowrap;
  color: #eef6ff;
}
.td-labels :deep(.sp) {
  margin-top: 2px;
  font-size: 11px;
  color: #9fb3cc;
  white-space: nowrap;
}
.td-labels :deep(.td-label.focus) {
  border-color: #ffbe6b;
}
.td-labels :deep(.td-label.focus .idx) {
  color: #ffbe6b;
}
.teardown {
  background-image: radial-gradient(60% 55% at 62% 52%, rgba(37, 99, 235, 0.22), transparent 70%);
}
@media (max-width: 760px) {
  .td-labels :deep(.td-label) { min-width: 0; padding: 5px 7px; }
  .td-labels :deep(.sp) { display: none; }
}
</style>
