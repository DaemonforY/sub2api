<template>
  <div
    class="rounded-[22px] border border-gray-200/80 bg-white/90 p-2.5 shadow-2xl shadow-primary-900/10 backdrop-blur sm:p-3.5 dark:border-white/10 dark:bg-dark-900/80 dark:shadow-black/50"
    data-testid="home-hero-demo"
  >
    <div class="flex w-max gap-1 rounded-xl bg-gray-100 p-1 dark:bg-white/5" role="tablist">
      <button
        v-for="k in TABS"
        :key="k"
        type="button"
        role="tab"
        :aria-selected="tab === k"
        :class="['rounded-lg px-4 py-1.5 text-sm transition', tab === k ? 'bg-white text-gray-900 shadow-sm dark:bg-white/10 dark:text-white' : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200']"
        @click="pick(k)"
      >
        {{ t(`home.v3.demo.tabs.${k}`) }}
      </button>
    </div>

    <div class="mt-2.5 flex items-start gap-3 rounded-2xl border border-gray-200 bg-gray-50 p-3 dark:border-white/10 dark:bg-black/25">
      <span class="mt-1 shrink-0 rounded-md border border-primary-300/60 px-1.5 text-xs text-primary-600 dark:border-primary-400/30 dark:text-primary-300">{{ t('home.v3.demo.you') }}</span>
      <textarea
        v-model="prompt"
        maxlength="1000"
        rows="3"
        class="h-[3.4rem] min-w-0 flex-1 resize-none border-0 bg-transparent p-0 text-sm leading-relaxed text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-0 sm:h-[4.6rem] sm:text-[15px] dark:text-white dark:placeholder:text-dark-500"
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
        class="btn btn-primary shrink-0 self-end px-4 py-2 text-sm"
        data-testid="home-hero-generate"
        @click="generate"
      >
        {{ t('home.v3.demo.generate') }}
      </button>
      <a v-else :href="tab === 'write' ? WRITE_HREF : CODE_HREF" class="btn btn-secondary shrink-0 self-end px-4 py-2 text-sm">
        {{ t(`home.v3.demo.${tab}.go`) }}
      </a>
    </div>

    <div :class="['demo-result relative mt-2.5 h-[300px] overflow-hidden rounded-2xl bg-gray-900 sm:h-[400px]', { pending }]">
      <img
        src="/home/cat.jpg"
        :alt="t('home.v3.demo.draw.alt')"
        :class="['demo-layer object-cover object-[30%_45%]', { shown: shown === 'draw' }]"
        fetchpriority="high"
      />
      <img
        src="/home/editor.jpg"
        :alt="t('home.v3.demo.write.alt')"
        :class="['demo-layer object-cover object-top', { shown: shown === 'write' }]"
        loading="lazy"
      />
      <div :class="['demo-layer bg-[#0b0913] p-5 font-mono text-[13px] leading-8 text-gray-300 sm:p-7 sm:text-sm', { shown: shown === 'code' }]">
        <div><span class="text-teal-400">codex ›</span> {{ t('home.v3.demo.code.cmd') }}</div>
        <div v-for="n in 4" :key="n" class="text-gray-500">• {{ t(`home.v3.demo.code.line${n}`) }}</div>
        <div class="text-emerald-400">✓ {{ t('home.v3.demo.code.done') }}</div>
        <span class="absolute right-3 top-3 rounded border border-dashed border-white/20 px-1.5 text-[11px] text-gray-500">{{ t('home.v3.demo.code.mock') }}</span>
      </div>
      <div v-show="pending" class="demo-shimmer absolute inset-0 grid place-items-center text-sm text-primary-200">{{ t('home.v3.demo.generating') }}</div>
    </div>

    <div class="mt-2 flex items-center justify-between px-1 text-xs text-gray-500 dark:text-dark-400">
      <span>{{ t(`home.v3.demo.${shown}.caption`) }}</span>
      <a :href="worksHref" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline dark:text-primary-300">{{ t('home.v3.demo.more') }} →</a>
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
.demo-shimmer {
  background: linear-gradient(100deg, transparent 30%, rgba(139, 92, 246, 0.25) 50%, transparent 70%) 0 0 / 200% 100%;
  animation: demo-sh 1.1s linear infinite;
}
@keyframes demo-sh {
  to {
    background-position: -200% 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .demo-layer,
  .demo-shimmer {
    transition: none;
    animation: none;
  }
}
</style>
