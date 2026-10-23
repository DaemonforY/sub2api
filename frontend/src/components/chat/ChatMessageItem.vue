<template>
  <!-- The user's question -->
  <div v-if="message.role === 'user'" class="group flex flex-col items-end" :data-testid="`chat-msg-${message.id}`">
    <div v-if="message.files.length" class="mb-2 flex flex-wrap justify-end gap-2">
      <ChatImage v-for="f in message.files" :key="f" :name="f" box-class="h-28 w-28 sm:h-32 sm:w-32" @open="$emit('open-image', $event)" @loaded="$emit('media')" />
    </div>
    <div v-if="editing" class="w-full max-w-[85%] rounded-2xl border border-primary-300 bg-white p-2 dark:border-primary-700 dark:bg-dark-900">
      <textarea
        ref="editBox"
        v-model="draft"
        rows="3"
        class="w-full resize-none border-0 bg-transparent p-1 text-sm text-gray-800 focus:outline-none focus:ring-0 dark:text-dark-100"
        @keydown.enter.exact.prevent="saveEdit"
        @keydown.esc="editing = false"
      ></textarea>
      <div class="flex justify-end gap-2">
        <button type="button" class="rounded-lg px-3 py-1 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-800" @click="editing = false">{{ t('common.cancel') }}</button>
        <button type="button" class="rounded-lg bg-primary-600 px-3 py-1 text-xs font-medium text-white hover:bg-primary-700 disabled:opacity-50" :disabled="!draft.trim()" @click="saveEdit">{{ t('chat.sendEdit') }}</button>
      </div>
    </div>
    <div v-else-if="message.content" class="max-w-[85%] whitespace-pre-wrap break-words rounded-2xl rounded-tr-md bg-primary-600 px-4 py-2.5 text-[15px] leading-relaxed text-white">
      <span v-if="message.kind === 'image'" class="mr-1 rounded bg-white/20 px-1.5 py-0.5 text-[11px]">{{ t('chat.drawTag') }}</span>{{ message.content }}
    </div>
    <div v-if="!editing && !busy" class="mt-1 flex gap-1 opacity-0 transition group-hover:opacity-100 max-sm:opacity-100">
      <button type="button" class="chat-act" :title="t('chat.copy')" @click="copy(message.content)"><Icon name="copy" size="sm" /></button>
      <button v-if="message.id > 0" type="button" class="chat-act" :title="t('chat.edit')" @click="startEdit"><Icon name="edit" size="sm" /></button>
    </div>
  </div>

  <!-- The model's reply -->
  <div v-else class="group flex gap-3" :data-testid="`chat-msg-${message.id}`">
    <div class="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary-500 to-violet-600 text-white shadow-sm">
      <Icon name="sparkles" size="sm" />
    </div>
    <div class="min-w-0 flex-1">
      <details v-if="message.reasoning" class="mb-2 rounded-xl bg-gray-50 px-3 py-2 text-sm text-gray-500 dark:bg-dark-800/60 dark:text-dark-400" :open="live && !message.content">
        <summary class="cursor-pointer select-none text-xs font-medium">
          {{ live && !message.content ? t('chat.thinking') : t('chat.reasoning') }}
        </summary>
        <div class="mt-2 whitespace-pre-wrap break-words text-xs leading-relaxed">{{ message.reasoning }}</div>
      </details>

      <!-- 画图 -->
      <template v-if="message.kind === 'image'">
        <div v-if="message.status === 'pending'" class="flex h-64 w-64 max-w-full flex-col items-center justify-center gap-3 rounded-2xl bg-gradient-to-br from-gray-100 to-gray-200 text-sm text-gray-500 dark:from-dark-800 dark:to-dark-700 dark:text-dark-400">
          <span class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>
          {{ t('chat.drawing') }}
          <span class="px-6 text-center text-xs text-gray-400">{{ t('chat.drawingHint') }}</span>
        </div>
        <div v-for="f in message.files" :key="f" class="inline-block">
          <ChatImage
            :name="f"
            box-class="w-full max-w-md aspect-auto min-h-40"
            placeholder-class="h-64 w-64 max-w-full"
            @open="$emit('open-image', $event)"
            @loaded="loaded[f] = true; $emit('media')"
          />
          <div class="mt-2 flex flex-wrap gap-2">
            <button type="button" class="chat-pill" @click="$emit('edit-image', f)"><Icon name="edit" size="xs" />{{ t('chat.editThisImage') }}</button>
            <button v-if="loaded[f]" type="button" class="chat-pill" @click="download(f)"><Icon name="download" size="xs" />{{ t('chat.download') }}</button>
          </div>
        </div>
      </template>

      <!-- 文字 -->
      <template v-else>
        <div v-if="message.content" ref="mdBox" class="chat-md break-words text-[15px] leading-relaxed text-gray-800 dark:text-dark-100" @click="onMdClick" v-html="html"></div>
        <div v-else-if="message.status === 'pending'" class="flex items-center gap-1.5 py-2" data-testid="chat-typing">
          <span class="chat-dot"></span><span class="chat-dot [animation-delay:0.15s]"></span><span class="chat-dot [animation-delay:0.3s]"></span>
        </div>
      </template>

      <div v-if="message.status === 'error'" class="mt-2 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300" data-testid="chat-error">
        {{ message.error || t('chat.failed') }}
      </div>
      <div v-else-if="message.status === 'stopped'" class="mt-1 text-xs text-gray-400">{{ t('chat.stopped') }}</div>

      <div v-if="message.status !== 'pending'" class="mt-1.5 flex flex-wrap items-center gap-1 text-xs text-gray-400" :class="last ? '' : 'opacity-0 transition group-hover:opacity-100 max-sm:opacity-100'">
        <button v-if="message.content" type="button" class="chat-act" :title="t('chat.copy')" @click="copy(message.content)"><Icon name="copy" size="sm" /></button>
        <button v-if="last && !busy" type="button" class="chat-act" :title="t('chat.regenerate')" data-testid="chat-regenerate" @click="$emit('regenerate')"><Icon name="refresh" size="sm" /></button>
        <span v-if="message.model" class="ml-1">{{ message.model }}</span>
        <span v-if="message.completion_tokens">· {{ t('chat.tokens', { n: (message.prompt_tokens || 0) + message.completion_tokens }) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { fileURL, type ChatMessage } from '@/api/chat'
import { useAppStore } from '@/stores/app'
import ChatImage from './ChatImage.vue'
import { renderChatMarkdown } from './chatMarkdown'
import 'highlight.js/styles/github-dark.css'

const props = defineProps<{
  message: ChatMessage
  /** The newest reply: its actions stay visible and it can be regenerated. */
  last?: boolean
  /** This reply is streaming now. */
  live?: boolean
  /** A turn is running in the conversation. */
  busy?: boolean
}>()
const emit = defineEmits<{
  (e: 'regenerate'): void
  (e: 'edit', content: string): void
  (e: 'edit-image', name: string): void
  (e: 'open-image', name: string): void
  /** A picture finished loading (the message grew). */
  (e: 'media'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const appStore = useAppStore()
const html = computed(() => renderChatMarkdown(props.message.content, t('chat.copy')))
const loaded = reactive<Record<string, boolean>>({})

const editing = ref(false)
const draft = ref('')
const editBox = ref<HTMLTextAreaElement>()

function copy(text: string) {
  void copyToClipboard(text, t('chat.copied'))
}

function startEdit() {
  draft.value = props.message.content
  editing.value = true
  void nextTick(() => editBox.value?.focus())
}

function saveEdit() {
  const text = draft.value.trim()
  if (!text) return
  editing.value = false
  emit('edit', text)
}

// 下载 saves the original (the page shows a smaller preview).
async function download(name: string) {
  try {
    const url = await fileURL(name, false)
    const a = document.createElement('a')
    a.href = url
    a.download = `hivegpt-${props.message.id}.${name.split('.').pop() || 'png'}`
    document.body.appendChild(a)
    a.click()
    a.remove()
  } catch {
    appStore.showError(t('chat.downloadFailed'))
  }
}

function onMdClick(e: MouseEvent) {
  const btn = (e.target as HTMLElement).closest('.chat-code-copy')
  if (!btn) return
  const code = btn.closest('.chat-code')?.querySelector('code')
  if (code) copy(code.textContent || '')
}
</script>

<style scoped>
.chat-act {
  @apply rounded-md p-1.5 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-800 dark:hover:text-dark-200;
}
.chat-pill {
  @apply inline-flex items-center gap-1 rounded-full border border-gray-200 px-3 py-1 text-xs text-gray-600 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-dark-300 dark:hover:text-primary-300;
}
.chat-dot {
  @apply h-2 w-2 animate-bounce rounded-full bg-primary-400;
}
.chat-md :deep(p) {
  margin: 0.5rem 0;
}
.chat-md :deep(> :first-child) {
  margin-top: 0;
}
.chat-md :deep(> :last-child) {
  margin-bottom: 0;
}
.chat-md :deep(h1),
.chat-md :deep(h2),
.chat-md :deep(h3),
.chat-md :deep(h4) {
  margin: 1rem 0 0.5rem;
  font-weight: 600;
  line-height: 1.4;
}
.chat-md :deep(h1) {
  font-size: 1.35em;
}
.chat-md :deep(h2) {
  font-size: 1.2em;
}
.chat-md :deep(h3) {
  font-size: 1.08em;
}
.chat-md :deep(ul),
.chat-md :deep(ol) {
  margin: 0.5rem 0;
  padding-left: 1.5rem;
}
.chat-md :deep(ul) {
  list-style: disc;
}
.chat-md :deep(ol) {
  list-style: decimal;
}
.chat-md :deep(li) {
  margin: 0.2rem 0;
}
.chat-md :deep(a) {
  color: #8b5cf6;
  text-decoration: underline;
}
.chat-md :deep(blockquote) {
  margin: 0.5rem 0;
  border-left: 3px solid rgba(139, 92, 246, 0.4);
  padding-left: 0.75rem;
  opacity: 0.85;
}
.chat-md :deep(hr) {
  margin: 1rem 0;
  border-color: rgba(148, 163, 184, 0.35);
}
.chat-md :deep(table) {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  margin: 0.75rem 0;
  border-collapse: collapse;
  font-size: 0.9em;
}
.chat-md :deep(th),
.chat-md :deep(td) {
  border: 1px solid rgba(148, 163, 184, 0.35);
  padding: 0.35rem 0.6rem;
  text-align: left;
}
.chat-md :deep(th) {
  background: rgba(139, 92, 246, 0.08);
  font-weight: 600;
}
.chat-md :deep(:not(pre) > code) {
  border-radius: 0.3rem;
  background: rgba(139, 92, 246, 0.12);
  padding: 0.1rem 0.35rem;
  font-size: 0.88em;
  color: #8b5cf6;
}
.chat-md :deep(.chat-code) {
  margin: 0.75rem 0;
  overflow: hidden;
  border-radius: 0.75rem;
  background: #0d1117;
}
.chat-md :deep(.chat-code-bar) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.35rem 0.75rem;
  background: #161b22;
  color: #8b949e;
  font-size: 0.75rem;
}
.chat-md :deep(.chat-code-copy) {
  border-radius: 0.375rem;
  padding: 0.1rem 0.5rem;
  color: #c9d1d9;
}
.chat-md :deep(.chat-code-copy:hover) {
  background: rgba(255, 255, 255, 0.1);
}
.chat-md :deep(pre) {
  margin: 0;
  overflow-x: auto;
  padding: 0.85rem 1rem;
  font-size: 0.85em;
  line-height: 1.55;
}
.chat-md :deep(pre code.hljs) {
  padding: 0;
  background: transparent;
}
</style>
