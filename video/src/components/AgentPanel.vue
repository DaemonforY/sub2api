<template>
  <aside class="flex min-h-0 flex-col bg-white dark:bg-ink-900">
    <div class="border-b border-ink-200 px-4 py-3 dark:border-ink-800">
      <h2 class="truncate text-[15px] font-semibold" :title="project?.title">{{ project?.title || '新作品' }}</h2>
      <p v-if="project" class="mt-1 flex items-center gap-1 text-xs text-ink-500" :title="usageTip">
        <Coins class="h-3.5 w-3.5" />用量 · 输入 {{ fmtK(project.usage.prompt_tokens) }} / 输出 {{ fmtK(project.usage.completion_tokens) }} tokens
        <template v-if="project.usage.tts_chars"> · 配音 {{ project.usage.tts_chars }} 字</template>
      </p>
    </div>

    <div ref="list" class="thin-scroll min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-4">
      <template v-for="(g, gi) in groups" :key="gi">
        <!-- the user's messages -->
        <div v-if="g.kind === 'user' || g.kind === 'answer'" class="flex justify-end">
          <div class="max-w-[85%] whitespace-pre-wrap rounded-2xl rounded-tr-md bg-gradient-to-br from-brand-500 to-brand-600 px-3.5 py-2.5 text-sm leading-6 text-white">{{ g.text }}</div>
        </div>

        <!-- agent steps, collapsed into one block -->
        <div v-else-if="g.kind === 'steps'" class="rounded-2xl border border-ink-200 bg-ink-50 text-sm dark:border-ink-800 dark:bg-ink-950">
          <button class="flex w-full items-center gap-2 px-3 py-2.5 text-left" @click="toggle(gi)">
            <Loader2 v-if="g.running" class="h-4 w-4 animate-spin text-brand-500" />
            <CircleCheck v-else class="h-4 w-4 text-emerald-500" />
            <span class="flex-1 truncate">{{ g.running ? g.items[g.items.length - 1].text : `已完成 ${g.items.length} 个步骤` }}</span>
            <ChevronDown class="h-4 w-4 text-ink-400 transition" :class="{ 'rotate-180': isOpen(gi, g) }" />
          </button>
          <ol v-if="isOpen(gi, g)" class="space-y-2 border-t border-ink-200 px-3 py-2.5 dark:border-ink-800">
            <li v-for="s in g.items" :key="s.id" class="flex items-start gap-2 text-[13px] text-ink-600 dark:text-ink-300">
              <Loader2 v-if="s.running" class="mt-0.5 h-3.5 w-3.5 shrink-0 animate-spin text-brand-500" />
              <Check v-else-if="s.state === 'done'" class="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" />
              <Sparkle v-else class="mt-0.5 h-3.5 w-3.5 shrink-0 text-ink-400" />
              <span>{{ s.text }}</span>
            </li>
          </ol>
        </div>

        <!-- clarifying questions -->
        <div v-else-if="g.kind === 'question'" class="rounded-2xl border border-brand-200 bg-brand-50/60 p-3.5 text-sm dark:border-brand-900/50 dark:bg-brand-900/10">
          <p class="leading-6">{{ g.text }}</p>
          <div v-for="q in g.questions" :key="q.id" class="mt-3">
            <p class="text-xs font-semibold text-ink-700 dark:text-ink-200">{{ q.title }}</p>
            <div class="mt-1.5 flex flex-wrap gap-1.5">
              <button
                v-for="o in q.options"
                :key="o.label"
                class="rounded-lg border px-2.5 py-1 text-xs transition"
                :class="picked[q.title] === o.label ? 'border-brand-500 bg-brand-500 text-white' : 'border-ink-200 bg-white hover:border-brand-300 dark:border-ink-700 dark:bg-ink-900'"
                :disabled="!g.open"
                @click="picked[q.title] = o.label"
              >
                {{ o.label }}<span v-if="o.recommended" class="ml-1 opacity-70">推荐</span>
              </button>
            </div>
          </div>
          <div v-if="g.open" class="mt-3 flex gap-2">
            <button class="btn-primary btn-sm" :disabled="busy" @click="submitAnswers(g)">确认并开始</button>
            <button class="btn-ghost btn-sm" :disabled="busy" @click="submitRecommended(g)">按推荐继续</button>
          </div>
        </div>

        <div v-else-if="g.kind === 'assistant'" class="flex gap-2">
          <span class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-brand-100 text-brand-600 dark:bg-brand-900/40"><Sparkles class="h-4 w-4" /></span>
          <div class="rounded-2xl rounded-tl-md bg-ink-100 px-3.5 py-2.5 text-sm leading-6 dark:bg-ink-800">{{ g.text }}</div>
        </div>

        <div v-else-if="g.kind === 'error'" class="rounded-2xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/20 dark:text-red-300">
          <p class="leading-6">{{ g.text }}</p>
          <button v-if="g.last && canResume" class="btn-ghost btn-sm mt-2" @click="$emit('resume')"><RotateCw class="h-3.5 w-3.5" />继续生成</button>
        </div>
      </template>
      <div v-if="!events.length" class="flex justify-center py-10 text-ink-400"><Loader2 class="h-5 w-5 animate-spin" /></div>
    </div>

    <div class="border-t border-ink-200 p-3 dark:border-ink-800">
      <div v-if="project?.mode === 'film' && project?.status === 'ready'" class="mb-2 flex items-center gap-2 text-xs text-ink-500">
        <Mic class="h-3.5 w-3.5" />配音
        <select class="rounded-lg border border-ink-200 bg-white px-2 py-1 text-xs dark:border-ink-700 dark:bg-ink-900" :value="project.options.voice" @change="$emit('voice', $event.target.value)">
          <option v-for="v in voices" :key="v.id" :value="v.id">{{ v.name }}（{{ v.gender }}）· {{ v.style }}</option>
        </select>
      </div>
      <div class="rounded-2xl border border-ink-200 bg-white focus-within:border-brand-400 dark:border-ink-700 dark:bg-ink-950">
        <textarea
          v-model="text"
          rows="3"
          maxlength="2000"
          class="block w-full resize-none bg-transparent px-3 py-2.5 text-sm leading-6 outline-none placeholder:text-ink-400"
          :placeholder="placeholder"
          :disabled="busy || !canChat"
          data-testid="agent-input"
          @keydown.enter.exact.prevent="submit"
        ></textarea>
        <div class="flex items-center justify-between px-2 pb-2">
          <span class="pl-1 text-[11px] text-ink-400">{{ focusScene && project?.mode === 'film' ? `当前在看分镜 ${focusScene.replace('s', '')}` : 'Enter 发送' }}</span>
          <button v-if="busy" class="flex items-center gap-1 rounded-lg bg-ink-900 px-3 py-1.5 text-xs font-medium text-white dark:bg-white dark:text-ink-900" @click="$emit('stop')"><Square class="h-3 w-3" fill="currentColor" />停止</button>
          <button v-else class="btn-primary btn-sm" :disabled="!text.trim() || !canChat" @click="submit"><ArrowUp class="h-3.5 w-3.5" />修改</button>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ArrowUp, Check, ChevronDown, CircleCheck, Coins, Loader2, Mic, RotateCw, Sparkle, Sparkles, Square } from 'lucide-vue-next'

const props = defineProps({
  project: { type: Object, default: null },
  events: { type: Array, default: () => [] },
  busy: Boolean,
  voices: { type: Array, default: () => [] },
  focusScene: { type: String, default: '' }
})
const emit = defineEmits(['send', 'answer', 'stop', 'resume', 'voice'])
const text = ref('')
const list = ref(null)
const picked = reactive({})
const opened = reactive({})

const canChat = computed(() => props.project && (props.project.status === 'ready' || (props.project.status === 'failed' && props.project.spec)))
const canResume = computed(() => props.project && (props.project.status === 'failed' || props.project.status === 'stopped'))
const placeholder = computed(() => {
  if (props.busy) return 'Agent 正在工作，完成后可继续输入'
  if (props.project?.status === 'questions') return '请先回答上面的问题'
  return '继续修改，例如：背景换成深蓝色、第 2 个分镜的图表慢一点、标题再大一些'
})
const usageTip = '这个作品累计调用模型的 token 数，按所选模型的价格从余额扣费，明细见 HiveGPT 主站「用量明细」'

// Consecutive step events become one collapsible block, like an agent's tool log.
const groups = computed(() => {
  const out = []
  const evs = props.events
  const lastQuestion = [...evs].reverse().find((e) => e.kind === 'question')
  evs.forEach((e, i) => {
    if (e.kind === 'step') {
      const state = e.data?.state || 'done'
      let g = out[out.length - 1]
      if (!g || g.kind !== 'steps') {
        g = { kind: 'steps', items: [], running: false }
        out.push(g)
      }
      g.items.push({ id: e.id, text: e.text, state, running: false })
      return
    }
    if (e.kind === 'question') {
      out.push({ kind: 'question', text: e.text, questions: e.data?.questions || [], open: e === lastQuestion && props.project?.status === 'questions' })
      return
    }
    out.push({ kind: e.kind, text: e.text, last: i === evs.length - 1 })
  })
  // The newest "running" step is live while the project runs.
  if (props.busy) {
    const g = out[out.length - 1]
    if (g?.kind === 'steps') {
      g.running = true
      const last = g.items[g.items.length - 1]
      if (last && last.state === 'running') last.running = true
    }
  }
  return out
})
const isOpen = (gi, g) => opened[gi] ?? g.running
const toggle = (gi) => (opened[gi] = !isOpen(gi, groups.value[gi]))

watch(
  () => props.events.length,
  async () => {
    await nextTick()
    list.value?.scrollTo({ top: list.value.scrollHeight, behavior: 'smooth' })
  }
)
// Pre-select the recommended options.
watch(
  groups,
  (gs) => {
    gs.filter((g) => g.kind === 'question' && g.open).forEach((g) =>
      g.questions.forEach((q) => {
        if (!picked[q.title]) picked[q.title] = (q.options.find((o) => o.recommended) || q.options[0])?.label
      })
    )
  },
  { immediate: true }
)

function submit() {
  const v = text.value.trim()
  if (!v || props.busy || !canChat.value) return
  emit('send', v)
  text.value = ''
}
function submitAnswers(g) {
  const answers = {}
  g.questions.forEach((q) => {
    if (picked[q.title]) answers[q.title] = picked[q.title]
  })
  emit('answer', answers)
}
function submitRecommended(g) {
  const answers = {}
  g.questions.forEach((q) => {
    const o = q.options.find((x) => x.recommended) || q.options[0]
    if (o) answers[q.title] = o.label
  })
  emit('answer', answers)
}
const fmtK = (n) => (n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n || 0))
</script>
