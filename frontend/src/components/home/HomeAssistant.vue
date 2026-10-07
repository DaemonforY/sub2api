<template>
  <div v-if="config?.enabled" class="fixed bottom-4 right-4 z-40 sm:bottom-6 sm:right-6" data-testid="home-assistant">
    <!-- Launcher -->
    <button
      v-if="!open"
      type="button"
      class="flex items-center gap-2 rounded-full bg-gradient-to-br from-primary-600 to-violet-600 px-4 py-3 text-sm font-semibold text-white shadow-lg shadow-primary-600/30 transition hover:scale-105 hover:shadow-xl"
      data-testid="home-assistant-open"
      @click="openPanel"
    >
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM2.25 12c0 4.556 4.03 8.25 9 8.25a9.764 9.764 0 002.555-.337A5.972 5.972 0 0018 21a5.969 5.969 0 01-1.508-3.984C19.212 15.668 20.25 13.94 20.25 12c0-4.556-4.03-8.25-9-8.25s-9 3.694-9 8.25z" />
      </svg>
      <span>{{ t('supportAssistant.open') }}</span>
    </button>

    <!-- Panel -->
    <section
      v-else
      class="fixed inset-x-2 bottom-2 flex h-[min(620px,calc(100vh-1rem))] flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-2xl sm:absolute sm:inset-x-auto sm:bottom-0 sm:right-0 sm:w-[400px] dark:border-dark-700 dark:bg-dark-900"
      role="dialog"
      :aria-label="t('supportAssistant.title', { site: siteName })"
      data-testid="home-assistant-panel"
    >
      <header class="flex items-start justify-between gap-3 bg-gradient-to-br from-primary-600 to-violet-600 px-4 py-3 text-white">
        <div class="min-w-0">
          <div class="truncate text-sm font-semibold">{{ t('supportAssistant.title', { site: siteName }) }}</div>
          <div class="mt-0.5 truncate text-xs text-white/80">{{ t('supportAssistant.subtitle') }}</div>
        </div>
        <div class="flex flex-shrink-0 items-center gap-1">
          <button v-if="messages.length" type="button" class="rounded-md px-2 py-1 text-xs text-white/80 hover:bg-white/15 hover:text-white" :disabled="busy" @click="clear">{{ t('supportAssistant.clear') }}</button>
          <button type="button" class="rounded-md p-1 text-white/80 hover:bg-white/15 hover:text-white" :aria-label="t('supportAssistant.close')" @click="open = false">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
      </header>

      <div ref="scroller" class="flex-1 space-y-3 overflow-y-auto px-4 py-4 text-sm">
        <div class="max-w-[90%] rounded-2xl rounded-tl-sm bg-gray-100 px-3 py-2 text-gray-800 dark:bg-dark-800 dark:text-dark-100">
          {{ t('supportAssistant.greeting', { site: siteName }) }}
        </div>
        <div v-if="!messages.length" class="flex flex-wrap gap-2">
          <button
            v-for="key in suggestionKeys"
            :key="key"
            type="button"
            class="rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs text-primary-700 transition hover:bg-primary-100 dark:border-primary-800 dark:bg-primary-900/30 dark:text-primary-300"
            :disabled="busy || left === 0"
            @click="ask(t(`supportAssistant.suggestions.${key}`))"
          >
            {{ t(`supportAssistant.suggestions.${key}`) }}
          </button>
        </div>

        <template v-for="(m, i) in messages" :key="i">
          <div v-if="m.role === 'user'" class="ml-auto max-w-[85%] whitespace-pre-wrap break-words rounded-2xl rounded-tr-sm bg-primary-600 px-3 py-2 text-white">{{ m.content }}</div>
          <div v-else class="max-w-[92%]">
            <div
              v-if="m.content"
              class="assistant-md break-words rounded-2xl rounded-tl-sm bg-gray-100 px-3 py-2 text-gray-800 dark:bg-dark-800 dark:text-dark-100"
              v-html="renderMarkdown(m.content)"
            ></div>
            <div v-else-if="busy && i === messages.length - 1" class="inline-flex items-center gap-2 rounded-2xl rounded-tl-sm bg-gray-100 px-3 py-2 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <span class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>{{ t('supportAssistant.thinking') }}
            </div>
            <div v-if="m.sources?.length" class="mt-1.5 flex flex-wrap items-center gap-1.5 text-xs">
              <span class="text-gray-400">{{ t('supportAssistant.sources') }}</span>
              <a
                v-for="s in m.sources"
                :key="s.url"
                :href="s.url"
                target="_blank"
                rel="noopener"
                class="max-w-full truncate rounded-md bg-primary-50 px-2 py-0.5 text-primary-700 hover:underline dark:bg-primary-900/30 dark:text-primary-300"
              >{{ s.title }}</a>
            </div>
          </div>
        </template>
        <div v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="home-assistant-error">
          {{ t('supportAssistant.error', { msg: error }) }}
        </div>
      </div>

      <footer class="border-t border-gray-100 px-3 pb-3 pt-2 dark:border-dark-700">
        <div class="mb-1.5 flex items-center justify-between gap-2 text-[11px] text-gray-400">
          <span>{{ left > 0 ? t('supportAssistant.left', { n: left }) : t('supportAssistant.none') }}</span>
          <span v-if="!config.logged_in && config.user_per_day > config.per_day">
            {{ t('supportAssistant.guestHint', { n: config.user_per_day }) }} ·
            <a href="/login?redirect=%2F" class="text-primary-600 hover:underline">{{ t('supportAssistant.login') }}</a>
          </span>
        </div>
        <div class="flex items-end gap-2">
          <textarea
            ref="input"
            v-model="draft"
            rows="2"
            :maxlength="maxChars"
            class="max-h-32 min-h-[2.5rem] flex-1 resize-none rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-800 outline-none focus:border-primary-400 focus:ring-2 focus:ring-primary-100 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-100"
            :placeholder="t('supportAssistant.placeholder')"
            :disabled="left === 0"
            data-testid="home-assistant-input"
            @keydown.enter.exact.prevent="send"
          ></textarea>
          <button
            v-if="busy"
            type="button"
            class="rounded-xl border border-gray-200 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-dark-300"
            @click="stop"
          >
            {{ t('supportAssistant.stop') }}
          </button>
          <button
            v-else
            type="button"
            class="rounded-xl bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-primary-700 disabled:opacity-40"
            :disabled="!draft.trim() || left === 0"
            data-testid="home-assistant-send"
            @click="send"
          >
            {{ t('supportAssistant.send') }}
          </button>
        </div>
        <p class="mt-1.5 text-[11px] leading-4 text-gray-400">{{ t('supportAssistant.disclaimer') }}</p>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAssistantConfig, streamAssistant, type AssistantConfig, type AssistantSource } from '@/api/assistant'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ siteName: string }>()
const { t } = useI18n()

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  sources?: AssistantSource[]
}

const STORE_KEY = 'support_assistant_chat'
const maxChars = 500
const suggestionKeys = ['q1', 'q2', 'q3', 'q4']

const config = ref<AssistantConfig | null>(null)
const open = ref(false)
const messages = ref<ChatMessage[]>([])
const draft = ref('')
const busy = ref(false)
const error = ref('')
const left = ref(0)
const scroller = ref<HTMLElement | null>(null)
const input = ref<HTMLTextAreaElement | null>(null)
let controller: AbortController | null = null

const siteName = computed(() => props.siteName || 'HiveGPT')

onMounted(async () => {
  try {
    messages.value = JSON.parse(sessionStorage.getItem(STORE_KEY) || '[]')
  } catch {
    messages.value = []
  }
  try {
    config.value = await getAssistantConfig()
    left.value = config.value.left
  } catch {
    config.value = null
  }
})

function persist() {
  try {
    sessionStorage.setItem(STORE_KEY, JSON.stringify(messages.value.slice(-20)))
  } catch {
    // storage full or disabled: the chat just isn't kept
  }
}

async function scrollDown() {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}

function openPanel() {
  open.value = true
  scrollDown()
  nextTick(() => input.value?.focus())
}

function clear() {
  messages.value = []
  error.value = ''
  persist()
}

function stop() {
  controller?.abort()
}

function send() {
  ask(draft.value)
}

async function ask(text: string) {
  const question = text.trim()
  if (!question || busy.value || left.value === 0) return
  if (question.length > maxChars) {
    error.value = t('supportAssistant.tooLong', { n: maxChars })
    return
  }
  error.value = ''
  draft.value = ''
  const history = messages.value.filter((m) => m.content).map((m) => ({ role: m.role, content: m.content }))
  messages.value.push({ role: 'user', content: question })
  const answer: ChatMessage = { role: 'assistant', content: '' }
  messages.value.push(answer)
  const reply = messages.value[messages.value.length - 1]
  busy.value = true
  controller = new AbortController()
  scrollDown()
  try {
    const done = await streamAssistant(
      [...history, { role: 'user', content: question }],
      (delta) => {
        reply.content += delta
        scrollDown()
      },
      controller.signal
    )
    reply.sources = done.sources
    left.value = done.left
  } catch (e) {
    if ((e as Error).name !== 'AbortError') {
      error.value = (e as Error).message
      if (!reply.content) messages.value.splice(messages.value.length - 2, 2)
      // The server refunds failed questions; refresh what is left.
      getAssistantConfig().then((c) => (left.value = c.left)).catch(() => {})
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
.assistant-md :deep(p) {
  margin: 0.25rem 0;
}
.assistant-md :deep(ul),
.assistant-md :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.25rem;
}
.assistant-md :deep(ul) {
  list-style: disc;
}
.assistant-md :deep(ol) {
  list-style: decimal;
}
.assistant-md :deep(a) {
  color: #6d28d9;
  text-decoration: underline;
}
:global(.dark) .assistant-md :deep(a) {
  color: #c4b5fd;
}
:global(.dark) .assistant-md :deep(code) {
  background: rgba(255, 255, 255, 0.1);
}
.assistant-md :deep(code) {
  border-radius: 0.25rem;
  background: rgba(0, 0, 0, 0.06);
  padding: 0 0.25rem;
  font-size: 0.85em;
}
.assistant-md :deep(pre) {
  margin: 0.375rem 0;
  overflow-x: auto;
  border-radius: 0.5rem;
  background: #1f2937;
  padding: 0.5rem 0.75rem;
  color: #f9fafb;
}
.assistant-md :deep(pre code) {
  background: transparent;
  padding: 0;
}
.assistant-md :deep(strong) {
  font-weight: 600;
}
</style>
