<template>
  <div class="flex h-full min-h-[24rem] flex-col">
    <div ref="scroller" class="flex-1 space-y-3 overflow-y-auto pb-3 text-sm">
      <div class="tutor-bubble max-w-[90%] rounded-2xl rounded-tl-sm bg-gray-100 px-3 py-2 text-gray-800 dark:bg-dark-800 dark:text-dark-100">{{ greeting }}</div>
      <div v-if="!messages.length && suggestions.length" class="flex flex-wrap gap-2">
        <button
          v-for="s in suggestions"
          :key="s"
          type="button"
          class="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs text-primary-700 hover:bg-primary-100 dark:border-primary-800 dark:bg-primary-900/30 dark:text-primary-300"
          :disabled="busy || blocked"
          @click="ask(s)"
        >
          {{ s }}
        </button>
      </div>
      <template v-for="(m, i) in messages" :key="i">
        <div v-if="m.role === 'user'" class="ml-auto max-w-[85%] whitespace-pre-wrap break-words rounded-2xl rounded-tr-sm bg-primary-600 px-3 py-2 text-white">{{ m.content }}</div>
        <div
          v-else-if="m.content"
          class="tutor-md max-w-[92%] break-words rounded-2xl rounded-tl-sm bg-gray-100 px-3 py-2 text-gray-800 dark:bg-dark-800 dark:text-dark-100"
          v-html="renderMarkdown(m.content)"
        ></div>
        <div v-else-if="busy && i === messages.length - 1" class="inline-flex items-center gap-2 rounded-2xl bg-gray-100 px-3 py-2 text-gray-500 dark:bg-dark-800">
          <span class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>{{ t('tutors.thinking') }}
        </div>
      </template>
      <div v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="tutor-chat-error">{{ error }}</div>
    </div>
    <div class="border-t border-gray-100 pt-2 dark:border-dark-700">
      <div v-if="left >= 0" class="mb-1 text-[11px] text-gray-400">{{ left > 0 ? t('tutors.left', { n: left }) : t('tutors.noneLeft') }}</div>
      <div class="flex items-end gap-2">
        <textarea
          v-model="draft"
          rows="2"
          maxlength="2000"
          class="max-h-40 min-h-[2.75rem] flex-1 resize-none rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm outline-none focus:border-primary-400 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-100"
          :placeholder="t('tutors.placeholder')"
          :disabled="blocked"
          data-testid="tutor-chat-input"
          @keydown.enter.exact.prevent="ask(draft)"
        ></textarea>
        <button v-if="busy" class="btn btn-secondary" @click="stop">{{ t('tutors.stop') }}</button>
        <button v-else class="btn btn-primary" :disabled="!draft.trim() || blocked" data-testid="tutor-chat-send" @click="ask(draft)">{{ t('tutors.send') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { renderMarkdown } from '@/utils/markdown'
import type { ChatMessage } from '@/api/tutors'

const props = withDefaults(
  defineProps<{
    greeting: string
    suggestions?: string[]
    /** Sends the conversation; resolves to how many questions are left today (-1: not counted). */
    send: (messages: ChatMessage[], onDelta: (t: string) => void, signal: AbortSignal) => Promise<{ left: number }>
    initialLeft?: number
    /** Keeps the conversation in sessionStorage under this key. */
    storageKey?: string
  }>(),
  { suggestions: () => [], initialLeft: -1, storageKey: '' }
)
const emit = defineEmits<{ (e: 'error', err: Error & { reason?: string }): void }>()

const { t } = useI18n()
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const busy = ref(false)
const error = ref('')
const left = ref(props.initialLeft)
const scroller = ref<HTMLElement | null>(null)
let controller: AbortController | null = null

const blocked = computed(() => left.value === 0)

onMounted(() => {
  if (!props.storageKey) return
  try {
    messages.value = JSON.parse(sessionStorage.getItem(props.storageKey) || '[]')
  } catch {
    messages.value = []
  }
  scrollDown()
})

function persist() {
  if (!props.storageKey) return
  try {
    sessionStorage.setItem(props.storageKey, JSON.stringify(messages.value.slice(-30)))
  } catch {
    // storage full: not kept
  }
}

async function scrollDown() {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}

function stop() {
  controller?.abort()
}

async function ask(text: string) {
  const q = text.trim()
  if (!q || busy.value || blocked.value) return
  error.value = ''
  draft.value = ''
  const history = messages.value.filter((m) => m.content)
  messages.value.push({ role: 'user', content: q })
  messages.value.push({ role: 'assistant', content: '' })
  const reply = messages.value[messages.value.length - 1]
  busy.value = true
  controller = new AbortController()
  scrollDown()
  try {
    const res = await props.send([...history, { role: 'user', content: q }], (d) => {
      reply.content += d
      scrollDown()
    }, controller.signal)
    if (res.left >= 0) left.value = res.left
  } catch (e) {
    if ((e as Error).name !== 'AbortError') {
      error.value = (e as Error).message
      if (!reply.content) messages.value.splice(messages.value.length - 2, 2)
      emit('error', e as Error)
    }
  } finally {
    if (!reply.content && messages.value[messages.value.length - 1] === reply) messages.value.pop()
    busy.value = false
    controller = null
    persist()
    scrollDown()
  }
}
</script>

<style scoped>
.tutor-md :deep(p) {
  margin: 0.25rem 0;
}
.tutor-md :deep(ul),
.tutor-md :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.25rem;
  list-style: disc;
}
.tutor-md :deep(ol) {
  list-style: decimal;
}
.tutor-md :deep(code) {
  border-radius: 0.25rem;
  background: rgba(0, 0, 0, 0.06);
  padding: 0 0.25rem;
}
.tutor-md :deep(pre) {
  overflow-x: auto;
  border-radius: 0.5rem;
  background: #1f2937;
  padding: 0.5rem 0.75rem;
  color: #f9fafb;
}
.tutor-md :deep(strong) {
  font-weight: 600;
}
.tutor-bubble {
  white-space: pre-wrap;
}
</style>
