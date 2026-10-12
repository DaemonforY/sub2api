<template>
  <div class="demo-card relative rounded-[22px] border border-cyan-300/20 bg-[#081222]/60 p-2.5 text-white shadow-2xl shadow-black/60 backdrop-blur-xl sm:p-3.5" data-testid="home-hero-demo">
    <div class="flex w-max gap-1 rounded-xl bg-white/5 p-1" role="tablist">
      <button
        v-for="k in TABS"
        :key="k"
        type="button"
        role="tab"
        :aria-selected="tab === k"
        :class="['rounded-lg px-4 py-1.5 text-sm transition', tab === k ? 'bg-cyan-300/15 text-white' : 'text-slate-400 hover:text-slate-200']"
        @click="pick(k)"
      >
        {{ t(`home.v3.demo.tabs.${k}`) }}
      </button>
    </div>

    <div class="mt-2.5 flex items-start gap-3 rounded-2xl border border-white/10 bg-black/30 p-3">
      <span class="mt-1 shrink-0 rounded-md border border-cyan-300/35 px-1.5 text-xs text-cyan-300">{{ t('home.v3.demo.you') }}</span>
      <textarea
        v-model="prompt"
        maxlength="1000"
        rows="3"
        class="h-[3.4rem] min-w-0 flex-1 resize-none border-0 bg-transparent p-0 text-sm leading-relaxed text-white placeholder:text-slate-500 focus:outline-none focus:ring-0 sm:h-[4.6rem] sm:text-[15px]"
        :placeholder="t('home.v3.demo.placeholder')"
        data-testid="home-hero-prompt"
        @focus="takeOver"
        @input="takeOver"
        @keydown.enter.meta.prevent="generate"
        @keydown.enter.ctrl.prevent="generate"
      ></textarea>
      <button
        v-if="tab === 'draw'"
        type="button"
        class="shrink-0 self-end rounded-xl bg-gradient-to-r from-blue-600 to-sky-500 px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-blue-600/30 transition hover:brightness-110"
        data-testid="home-hero-generate"
        @click="generate"
      >
        {{ t('home.v3.demo.generate') }}
      </button>
      <a v-else :href="tab === 'write' ? WRITE_HREF : CODE_HREF" class="shrink-0 self-end rounded-xl border border-white/15 px-4 py-2 text-sm text-slate-200 transition hover:border-cyan-300/50">
        {{ t(`home.v3.demo.${tab}.go`) }}
      </a>
    </div>

    <div :class="['demo-result relative mt-2.5 h-[300px] overflow-hidden rounded-2xl bg-[#050a14] sm:h-[400px]', { pending }]">
      <img
        src="/home-media/cat.jpg"
        :alt="t('home.v3.demo.draw.alt')"
        :class="['demo-layer object-cover object-[30%_45%]', { shown: shown === 'draw' }]"
        fetchpriority="high"
      />
      <img
        src="/home-media/editor.jpg"
        :alt="t('home.v3.demo.write.alt')"
        :class="['demo-layer object-cover object-top', { shown: shown === 'write' }]"
        loading="lazy"
      />
      <div :class="['demo-layer bg-[#06101e] p-5 font-mono text-[13px] leading-8 text-slate-300 sm:p-7 sm:text-sm', { shown: shown === 'code' }]">
        <div><span class="text-cyan-300">codex ›</span> {{ t('home.v3.demo.code.cmd') }}</div>
        <div v-for="n in 4" :key="n" class="text-slate-500">• {{ t(`home.v3.demo.code.line${n}`) }}</div>
        <div class="text-emerald-400">✓ {{ t('home.v3.demo.code.done') }}</div>
        <span class="absolute right-3 top-3 rounded border border-dashed border-white/20 px-1.5 text-[11px] text-slate-500">{{ t('home.v3.demo.code.mock') }}</span>
      </div>
      <!-- viewfinder corners and the scan while a piece is being made -->
      <span class="corner tl" aria-hidden="true"></span><span class="corner tr" aria-hidden="true"></span><span class="corner bl" aria-hidden="true"></span><span class="corner br" aria-hidden="true"></span>
      <div v-show="pending" class="demo-shimmer absolute inset-0" aria-hidden="true"></div>
      <div v-show="pending" class="demo-scan absolute inset-x-0" aria-hidden="true"></div>
      <div class="absolute left-3 top-3 rounded-md bg-black/55 px-2 py-0.5 font-mono text-[11px] tracking-wider text-cyan-200 backdrop-blur" data-testid="home-hero-demo-status">
        <template v-if="pending">{{ t('home.v3.demo.generating') }} {{ progress }}%</template>
        <template v-else>✓ {{ t(`home.v3.demo.${shown}.status`) }}</template>
      </div>
    </div>

    <div class="mt-2 flex items-center justify-between px-1 text-xs text-slate-400">
      <span>{{ t(`home.v3.demo.${shown}.caption`) }}</span>
      <a :href="worksHref" target="_blank" rel="noopener noreferrer" class="text-cyan-300 hover:underline">{{ t('home.v3.demo.more') }} →</a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { canvasUrl } from '@/constants/crossSites'

/**
 * 「一句话 → 作品」 on the home hero: types an example sentence for drawing, writing and coding and
 * shows what came back (a member's picture, the editor, a Codex run — the last one is a mock and
 * says so). The box is real: typing in it switches to drawing and 生成 emits the prompt, which the
 * page turns into sign-up-then-draw for visitors and the canvas for members.
 */
type Tab = 'draw' | 'write' | 'code'
const TABS: Tab[] = ['draw', 'write', 'code']
const WRITE_HREF = '/editor/?utm_source=home&utm_medium=hero-demo'
const CODE_HREF = '/learn/codex/hivegpt?utm_source=home&utm_medium=hero-demo'
const TYPE_MS = 34
const HOLD_MS = 5200

const emit = defineEmits<{ generate: [prompt: string] }>()
const { t } = useI18n()

const worksHref = canvasUrl({ medium: 'home-hero', path: '/explore' })
const tab = ref<Tab>('draw')
const shown = ref<Tab>('draw') // the result on screen; lags `tab` while the next sentence types
const prompt = ref('')
const pending = ref(false)
const progress = ref(0) // the 「生成中 N%」 chip while a sentence is being typed
let auto = true
let timers: ReturnType<typeof setTimeout>[] = []
const reduceMotion = typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches

const later = (fn: () => void, ms: number) => timers.push(setTimeout(fn, ms))
function stop() {
  timers.forEach(clearTimeout)
  timers = []
  pending.value = false
}

function play(next: Tab) {
  stop()
  tab.value = next
  const text = t(`home.v3.demo.${next}.prompt`)
  if (reduceMotion) {
    prompt.value = text
    shown.value = next
    return
  }
  prompt.value = ''
  pending.value = true
  let i = 0
  const step = () => {
    prompt.value = text.slice(0, ++i)
    progress.value = Math.round((i / text.length) * 92)
    if (i < text.length) return later(step, TYPE_MS)
    later(() => {
      pending.value = false
      shown.value = next
    }, 900)
    if (auto) later(() => play(TABS[(TABS.indexOf(next) + 1) % TABS.length]), HOLD_MS)
  }
  later(step, 300)
}

function pick(k: Tab) {
  auto = false
  play(k)
}

// The visitor's own words: stop the show and make it a drawing prompt.
function takeOver() {
  if (!auto && tab.value === 'draw' && !pending.value) return
  auto = false
  const typing = pending.value
  stop()
  if (tab.value !== 'draw') {
    tab.value = 'draw'
    prompt.value = ''
  } else if (typing) {
    prompt.value = t('home.v3.demo.draw.prompt')
  }
  shown.value = 'draw'
}

function generate() {
  takeOver()
  emit('generate', prompt.value.trim() || t('home.v3.demo.draw.prompt'))
}

onMounted(() => play('draw'))
onBeforeUnmount(stop)
</script>

<style scoped>
.demo-layer {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  filter: blur(16px);
  transform: scale(1.03);
  transition: opacity 0.6s, filter 0.9s, transform 0.9s;
}
.demo-layer.shown {
  opacity: 1;
  filter: none;
  transform: none;
}
.pending .demo-layer.shown {
  opacity: 0.35;
  filter: blur(6px) saturate(0.6);
}
.demo-card {
  animation: card-breathe 4s ease-in-out infinite;
}
@keyframes card-breathe {
  0%, 100% { box-shadow: 0 40px 80px -30px rgba(0, 0, 0, 0.8), 0 0 50px -24px rgba(59, 130, 246, 0.55); }
  50% { box-shadow: 0 40px 80px -30px rgba(0, 0, 0, 0.8), 0 0 70px -16px rgba(94, 231, 255, 0.55); }
}
.corner {
  position: absolute;
  width: 18px;
  height: 18px;
  border-color: rgba(94, 231, 255, 0.85);
  border-style: solid;
  pointer-events: none;
  z-index: 2;
}
.corner.tl { left: 10px; top: 10px; border-width: 2px 0 0 2px; border-top-left-radius: 6px; }
.corner.tr { right: 10px; top: 10px; border-width: 2px 2px 0 0; border-top-right-radius: 6px; }
.corner.bl { left: 10px; bottom: 10px; border-width: 0 0 2px 2px; border-bottom-left-radius: 6px; }
.corner.br { right: 10px; bottom: 10px; border-width: 0 2px 2px 0; border-bottom-right-radius: 6px; }
.demo-scan {
  top: 0;
  height: 70px;
  background: linear-gradient(to bottom, transparent, rgba(94, 231, 255, 0.28) 70%, rgba(234, 247, 255, 0.85) 98%, transparent);
  mix-blend-mode: screen;
  animation: demo-scan 1.6s linear infinite;
  pointer-events: none;
}
@keyframes demo-scan {
  from { transform: translateY(-70px); }
  to { transform: translateY(400px); }
}
.demo-shimmer {
  background: linear-gradient(100deg, transparent 30%, rgba(94, 231, 255, 0.18) 50%, transparent 70%) 0 0 / 200% 100%;
  animation: demo-sh 1.1s linear infinite;
}
@keyframes demo-sh {
  to {
    background-position: -200% 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .demo-layer,
  .demo-card,
  .demo-scan,
  .demo-shimmer {
    transition: none;
    animation: none;
  }
}
</style>
