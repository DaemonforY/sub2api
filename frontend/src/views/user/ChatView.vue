<template>
  <div class="flex h-[100dvh] overflow-hidden bg-white text-gray-900 dark:bg-dark-950 dark:text-dark-50" data-testid="chat-page">
    <!-- Conversation list -->
    <div v-if="sidebarOpen" class="fixed inset-0 z-30 bg-black/30 md:hidden" @click="sidebarOpen = false"></div>
    <aside
      class="fixed inset-y-0 left-0 z-40 flex w-72 flex-col border-r border-gray-100 bg-gray-50 transition-transform dark:border-dark-800 dark:bg-dark-900 md:static md:translate-x-0"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <div class="flex items-center justify-between px-4 pb-2 pt-4">
        <router-link to="/dashboard" class="flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-800 dark:text-dark-100">
          <img v-if="siteLogo" :src="siteLogo" alt="" class="h-6 w-6 rounded" />
          <span class="truncate">{{ siteName }}</span>
          <span class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">{{ t('chat.title') }}</span>
        </router-link>
        <button type="button" class="rounded-md p-1 text-gray-400 hover:bg-gray-200 md:hidden dark:hover:bg-dark-800" @click="sidebarOpen = false">
          <Icon name="x" size="sm" />
        </button>
      </div>
      <div class="px-3">
        <button
          type="button"
          class="flex w-full items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 shadow-sm transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100"
          data-testid="chat-new"
          @click="newChat"
        >
          <Icon name="plus" size="sm" />{{ t('chat.newChat') }}
        </button>
        <div class="relative mt-2">
          <Icon name="search" size="xs" class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            v-model="query"
            type="search"
            class="w-full rounded-lg border-0 bg-gray-100 py-1.5 pl-8 pr-2 text-sm placeholder:text-gray-400 focus:ring-1 focus:ring-primary-300 dark:bg-dark-800"
            :placeholder="t('chat.search')"
            @input="onSearch"
          />
        </div>
      </div>
      <nav class="mt-2 flex-1 overflow-y-auto px-2 pb-3" data-testid="chat-list">
        <template v-for="section in sections" :key="section.label">
          <div class="px-2 pb-1 pt-3 text-[11px] font-medium text-gray-400">{{ section.label }}</div>
          <div
            v-for="c in section.items"
            :key="c.id"
            class="group relative flex items-center rounded-lg text-sm"
            :class="c.id === conv?.id ? 'bg-white shadow-sm dark:bg-dark-800' : 'hover:bg-gray-100 dark:hover:bg-dark-800/60'"
          >
            <input
              v-if="renamingId === c.id"
              v-model="renameDraft"
              class="m-1 w-full rounded-md border-primary-300 px-2 py-1 text-sm dark:bg-dark-900"
              maxlength="40"
              @keydown.enter.prevent="saveRename(c)"
              @keydown.esc="renamingId = 0"
              @blur="saveRename(c)"
            />
            <template v-else>
              <button type="button" class="min-w-0 flex-1 truncate px-3 py-2 text-left text-gray-700 dark:text-dark-200" @click="openConversation(c.id)">
                <span v-if="c.pinned" class="mr-1 text-primary-500">•</span>{{ c.title || t('chat.untitled') }}
              </button>
              <button
                type="button"
                class="mr-1 rounded p-1 text-gray-400 opacity-0 hover:bg-gray-200 hover:text-gray-700 group-hover:opacity-100 max-md:opacity-100 dark:hover:bg-dark-700"
                :aria-label="t('chat.more')"
                @click.stop="menuId = menuId === c.id ? 0 : c.id"
              >
                <Icon name="more" size="sm" />
              </button>
              <div v-if="menuId === c.id" class="absolute right-1 top-9 z-10 w-32 overflow-hidden rounded-lg border border-gray-100 bg-white py-1 text-sm shadow-lg dark:border-dark-700 dark:bg-dark-800">
                <button type="button" class="menu-item" @click="startRename(c)"><Icon name="edit" size="xs" />{{ t('chat.rename') }}</button>
                <button type="button" class="menu-item" @click="togglePin(c)">
                  <Icon name="arrowUp" size="xs" />{{ c.pinned ? t('chat.unpin') : t('chat.pin') }}
                </button>
                <button type="button" class="menu-item text-red-600 dark:text-red-400" @click="deleteTarget = c; menuId = 0"><Icon name="trash" size="xs" />{{ t('chat.delete') }}</button>
              </div>
            </template>
          </div>
        </template>
        <div v-if="!conversations.length && !listLoading" class="px-3 py-8 text-center text-xs text-gray-400">
          {{ query ? t('chat.noResults') : t('chat.noConversations') }}
        </div>
      </nav>
      <div class="border-t border-gray-100 px-3 py-3 text-xs dark:border-dark-800">
        <router-link to="/dashboard" class="flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-dark-800 dark:hover:text-dark-100">
          <Icon name="arrowLeft" size="xs" />{{ t('chat.backToConsole') }}
        </router-link>
      </div>
    </aside>

    <!-- Conversation -->
    <main class="relative flex min-w-0 flex-1 flex-col">
      <header class="flex items-center gap-2 border-b border-gray-100 px-3 py-2 dark:border-dark-800">
        <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 md:hidden dark:hover:bg-dark-800" :aria-label="t('chat.conversations')" @click="sidebarOpen = true">
          <Icon name="menu" size="md" />
        </button>
        <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
          <select v-if="keys.length" v-model.number="keyId" class="pick max-w-[11rem]" :title="t('chat.key')" data-testid="chat-key">
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }} · {{ k.group }}{{ k.rate && k.rate !== 1 ? ` ×${k.rate}` : '' }}</option>
          </select>
          <select v-if="mode === 'text' && models.length" v-model="model" class="pick max-w-[13rem]" :title="t('chat.model')" data-testid="chat-model">
            <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
          </select>
          <span v-else-if="mode === 'image'" class="rounded-lg bg-gray-100 px-2.5 py-1.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">gpt-image-2</span>
          <span v-else-if="modelsLoading" class="text-xs text-gray-400">{{ t('chat.loadingModels') }}</span>
        </div>
        <button
          type="button"
          class="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800"
          data-testid="chat-role"
          @click="roleOpen = true"
        >
          <span>{{ currentRole.icon }}</span>
          <span class="hidden sm:inline">{{ currentRole.label }}</span>
        </button>
      </header>

      <div ref="scroller" class="flex-1 overflow-y-auto" @scroll="onScroll">
        <div v-if="keysError" class="mx-auto flex max-w-md flex-col items-center px-6 py-20 text-center text-sm text-gray-500">
          {{ t('chat.loadFailed') }}
          <button type="button" class="btn btn-secondary mt-4" @click="loadKeys">{{ t('chat.retry') }}</button>
        </div>

        <!-- No key yet -->
        <div v-else-if="!keysLoading && !keys.length" class="mx-auto flex max-w-md flex-col items-center px-6 py-20 text-center" data-testid="chat-no-key">
          <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300">
            <Icon name="key" size="lg" />
          </div>
          <h2 class="text-lg font-semibold">{{ t('chat.noKeyTitle') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('chat.noKeyHint') }}</p>
          <router-link to="/keys" class="btn btn-primary mt-5">{{ t('chat.createKey') }}</router-link>
        </div>

        <!-- Empty conversation -->
        <div v-else-if="!messages.length && !detailLoading" class="mx-auto flex max-w-2xl flex-col items-center px-6 pb-10 pt-[12vh] text-center">
          <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-violet-600 text-white shadow-lg shadow-primary-500/20">
            <Icon name="sparkles" size="lg" />
          </div>
          <h1 class="text-2xl font-semibold">{{ mode === 'image' ? t('chat.drawHello') : t('chat.hello') }}</h1>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ mode === 'image' ? t('chat.drawHelloHint') : t('chat.helloHint') }}</p>
          <div v-if="mode === 'text'" class="mt-6 flex flex-wrap justify-center gap-2">
            <button
              v-for="r in CHAT_ROLES.slice(1, 7)"
              :key="r.id"
              type="button"
              class="rounded-full border px-3 py-1.5 text-sm transition"
              :class="systemPrompt === r.prompt ? 'border-primary-400 bg-primary-50 text-primary-700 dark:border-primary-600 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 text-gray-600 hover:border-primary-300 dark:border-dark-700 dark:text-dark-300'"
              @click="applyRole(systemPrompt === r.prompt ? '' : r.prompt)"
            >
              {{ r.icon }} {{ t(`chat.roles.${r.id}`) }}
            </button>
          </div>
          <div class="mt-6 grid w-full gap-2 sm:grid-cols-2">
            <button
              v-for="(ex, i) in examples"
              :key="i"
              type="button"
              class="rounded-xl border border-gray-200 px-4 py-3 text-left text-sm text-gray-600 transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-700 dark:text-dark-300 dark:hover:bg-dark-800"
              @click="composer?.setText(ex)"
            >
              {{ ex }}
            </button>
          </div>
        </div>

        <div v-else class="mx-auto max-w-3xl space-y-6 px-4 py-6" data-testid="chat-messages">
          <div v-if="detailLoading" class="flex justify-center py-10">
            <span class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>
          </div>
          <ChatMessageItem
            v-for="(m, i) in messages"
            :key="m.id"
            :message="m"
            :last="i === messages.length - 1 && m.role === 'assistant'"
            :live="busy && i === messages.length - 1"
            :busy="busy || running"
            @regenerate="regenerate"
            @edit="(content) => editAndSend(m, content)"
            @edit-image="editImage"
            @open-image="openImage"
            @media="stick && scrollToBottom()"
          />
        </div>
      </div>

      <button
        v-if="!stick && messages.length"
        type="button"
        class="absolute bottom-36 left-1/2 z-10 -translate-x-1/2 rounded-full border border-gray-200 bg-white p-2 text-gray-500 shadow-md hover:text-gray-800 dark:border-dark-700 dark:bg-dark-800"
        :aria-label="t('chat.toBottom')"
        @click="scrollToBottom(true)"
      >
        <Icon name="arrowDown" size="sm" />
      </button>

      <div class="mx-auto w-full max-w-3xl px-3 pb-3 pt-1 sm:px-4">
        <ChatComposer
          ref="composer"
          v-model:mode="mode"
          v-model:image-size="imageSize"
          :busy="busy"
          :can-draw="canDraw"
          :disabled="!keyId || (mode === 'text' && !model) || running"
          @send="(p) => send(p)"
          @stop="stop"
        />
        <p class="mt-1.5 text-center text-[11px] text-gray-400">{{ t('chat.footer') }}</p>
      </div>
    </main>

    <!-- Picture viewer -->
    <div v-if="lightbox" class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4" @click="lightbox = ''">
      <img :src="lightbox" class="max-h-full max-w-full rounded-lg object-contain" alt="" />
    </div>

    <ChatRoleDialog :show="roleOpen" :prompt="systemPrompt" @close="roleOpen = false" @apply="applyRole" />
    <ConfirmDialog
      :show="!!deleteTarget"
      :title="t('chat.deleteTitle')"
      :message="t('chat.deleteMessage', { title: deleteTarget?.title || t('chat.untitled') })"
      :confirm-text="t('chat.delete')"
      danger
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatMessageItem from '@/components/chat/ChatMessageItem.vue'
import ChatRoleDialog from '@/components/chat/ChatRoleDialog.vue'
import { CHAT_ROLES } from '@/components/chat/chatRoles'
import { useAppStore } from '@/stores/app'
import * as chatAPI from '@/api/chat'
import type { ChatConversation, ChatKey, ChatMessage } from '@/api/chat'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'HiveGPT')
const siteLogo = computed(() => appStore.siteLogo || '')

const keys = ref<ChatKey[]>([])
const keysLoading = ref(true)
const keysError = ref(false)
const keyId = ref<number | null>(null)
const models = ref<string[]>([])
const modelsLoading = ref(false)
const model = ref('')
const modelCache = new Map<number, string[]>()

const conversations = ref<ChatConversation[]>([])
const listLoading = ref(true)
const query = ref('')
const conv = ref<ChatConversation | null>(null)
const messages = ref<ChatMessage[]>([])
const detailLoading = ref(false)
const running = ref(false)
/** System prompt of a new chat (before it is saved). */
const draftSystemPrompt = ref('')

const mode = ref<'text' | 'image'>('text')
const imageSize = ref('1024x1024')
const busy = ref(false)
let controller: AbortController | null = null

const sidebarOpen = ref(false)
const menuId = ref(0)
const renamingId = ref(0)
const renameDraft = ref('')
const deleteTarget = ref<ChatConversation | null>(null)
const roleOpen = ref(false)
const lightbox = ref('')
const scroller = ref<HTMLElement>()
const stick = ref(true)
const composer = ref<InstanceType<typeof ChatComposer>>()

const selectedKey = computed(() => keys.value.find((k) => k.id === keyId.value) || null)
const canDraw = computed(() => !!selectedKey.value?.can_draw)
const systemPrompt = computed(() => (conv.value ? conv.value.system_prompt : draftSystemPrompt.value))
const currentRole = computed(() => {
  const p = systemPrompt.value
  const preset = CHAT_ROLES.find((r) => r.prompt === p)
  if (preset) return { icon: preset.icon, label: preset.id === 'general' ? t('chat.role') : t(`chat.roles.${preset.id}`) }
  return { icon: '🎭', label: t('chat.customRole') }
})
const examples = computed(() => (mode.value === 'image' ? [t('chat.drawEx1'), t('chat.drawEx2'), t('chat.drawEx3'), t('chat.drawEx4')] : [t('chat.ex1'), t('chat.ex2'), t('chat.ex3'), t('chat.ex4')]))

// Sidebar sections: pinned, today, yesterday, this week, earlier.
const sections = computed(() => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const groups: { label: string; items: ChatConversation[] }[] = [
    { label: t('chat.pinned'), items: [] },
    { label: t('chat.today'), items: [] },
    { label: t('chat.yesterday'), items: [] },
    { label: t('chat.week'), items: [] },
    { label: t('chat.earlier'), items: [] }
  ]
  for (const c of conversations.value) {
    const at = new Date(c.updated_at).getTime()
    const g = c.pinned ? 0 : at >= today ? 1 : at >= today - 86400e3 ? 2 : at >= today - 7 * 86400e3 ? 3 : 4
    groups[g].items.push(c)
  }
  return groups.filter((g) => g.items.length)
})

function errMsg(err: unknown, fallback: string): string {
  return (err as { message?: string })?.message || fallback
}

// Keys and models -------------------------------------------------------------------------------

async function loadKeys() {
  keysLoading.value = true
  keysError.value = false
  try {
    keys.value = await chatAPI.listKeys()
    const saved = Number(localStorage.getItem('chat:key'))
    if (!keyId.value || !keys.value.some((k) => k.id === keyId.value)) {
      keyId.value = keys.value.find((k) => k.id === saved)?.id ?? keys.value[0]?.id ?? null
    }
  } catch (err) {
    keysError.value = true
    appStore.showError(errMsg(err, t('chat.loadFailed')))
  } finally {
    keysLoading.value = false
  }
}

async function loadModels(id: number, prefer = '') {
  let list = modelCache.get(id)
  if (!list) {
    modelsLoading.value = true
    try {
      list = (await chatAPI.listModels(id)).models
      modelCache.set(id, list)
    } catch (err) {
      list = []
      appStore.showError(errMsg(err, t('chat.loadFailed')))
    } finally {
      modelsLoading.value = false
    }
  }
  if (keyId.value !== id) return
  models.value = list
  const saved = localStorage.getItem(`chat:model:${id}`) || ''
  model.value = [prefer, model.value, saved].find((m) => m && list!.includes(m)) || list[0] || ''
}

watch(keyId, (id) => {
  if (!id) return
  localStorage.setItem('chat:key', String(id))
  if (!canDraw.value && mode.value === 'image') mode.value = 'text'
  void loadModels(id)
})

watch(model, (m) => {
  if (m && keyId.value) localStorage.setItem(`chat:model:${keyId.value}`, m)
})

// Conversations ---------------------------------------------------------------------------------

async function loadList() {
  listLoading.value = true
  try {
    conversations.value = await chatAPI.listConversations(query.value.trim())
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.loadFailed')))
  } finally {
    listLoading.value = false
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadList, 300)
}

function upsertConversation(c: ChatConversation, toTop = false) {
  const i = conversations.value.findIndex((x) => x.id === c.id)
  if (i >= 0) conversations.value.splice(i, 1)
  if (i >= 0 && !toTop) conversations.value.splice(i, 0, c)
  else conversations.value.unshift(c)
  if (conv.value?.id === c.id) conv.value = c
}

let pollTimer: ReturnType<typeof setTimeout> | undefined

async function openConversation(id: number, quiet = false) {
  sidebarOpen.value = false
  menuId.value = 0
  if (busy.value && conv.value?.id !== id) stop()
  if (!quiet) {
    detailLoading.value = true
    messages.value = []
  }
  clearTimeout(pollTimer)
  try {
    const d = await chatAPI.getConversation(id)
    conv.value = d.conversation
    messages.value = d.messages
    running.value = d.running
    if (route.query.c !== String(id)) void router.replace({ name: 'Chat', query: { c: String(id) } })
    const k = d.conversation.api_key_id
    if (!quiet && k && keys.value.some((x) => x.id === k)) {
      if (keyId.value !== k) keyId.value = k
      void loadModels(k, d.conversation.model)
    } else if (!quiet && d.conversation.model && models.value.includes(d.conversation.model)) {
      model.value = d.conversation.model
    }
    if (running.value) pollTimer = setTimeout(() => openConversation(id, true), 4000)
    if (!quiet) {
      stick.value = true
      void nextTick(() => scrollToBottom())
    }
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.loadFailed')))
    if (!quiet) newChat()
  } finally {
    detailLoading.value = false
  }
}

function newChat() {
  if (busy.value) stop()
  clearTimeout(pollTimer)
  conv.value = null
  messages.value = []
  running.value = false
  draftSystemPrompt.value = ''
  sidebarOpen.value = false
  if (route.query.c) void router.replace({ name: 'Chat' })
  void nextTick(() => composer.value?.focus())
}

function startRename(c: ChatConversation) {
  menuId.value = 0
  renamingId.value = c.id
  renameDraft.value = c.title
}

async function saveRename(c: ChatConversation) {
  if (renamingId.value !== c.id) return
  renamingId.value = 0
  const title = renameDraft.value.trim()
  if (!title || title === c.title) return
  try {
    upsertConversation(await chatAPI.updateConversation(c.id, { title }))
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.saveFailed')))
  }
}

async function togglePin(c: ChatConversation) {
  menuId.value = 0
  try {
    upsertConversation(await chatAPI.updateConversation(c.id, { pinned: !c.pinned }))
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.saveFailed')))
  }
}

async function confirmDelete() {
  const c = deleteTarget.value
  deleteTarget.value = null
  if (!c) return
  try {
    await chatAPI.deleteConversation(c.id)
    conversations.value = conversations.value.filter((x) => x.id !== c.id)
    if (conv.value?.id === c.id) newChat()
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.saveFailed')))
  }
}

async function applyRole(prompt: string) {
  roleOpen.value = false
  if (!conv.value) {
    draftSystemPrompt.value = prompt
    return
  }
  try {
    upsertConversation(await chatAPI.updateConversation(conv.value.id, { system_prompt: prompt }))
    appStore.showSuccess(t('chat.roleSaved'))
  } catch (err) {
    appStore.showError(errMsg(err, t('chat.saveFailed')))
  }
}

// Sending ---------------------------------------------------------------------------------------

function onScroll() {
  const el = scroller.value
  if (el) stick.value = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

function scrollToBottom(smooth = false) {
  const el = scroller.value
  if (el) el.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'auto' })
}

watch(
  () => {
    const last = messages.value[messages.value.length - 1]
    return [messages.value.length, last?.content.length, last?.reasoning?.length, last?.status]
  },
  () => {
    if (stick.value) void nextTick(() => scrollToBottom())
  }
)

function tempMessage(role: 'user' | 'assistant', kind: 'text' | 'image', content = '', files: string[] = []): ChatMessage {
  return {
    id: -Math.floor(Math.random() * 1e9) - 1,
    conversation_id: conv.value?.id || 0,
    role,
    kind,
    content,
    files,
    status: role === 'user' ? 'done' : 'pending',
    created_at: new Date().toISOString()
  }
}

interface TurnOptions {
  regenerate?: boolean
  editId?: number
  kind?: 'text' | 'image'
}

async function send(payload: { content: string; files: string[] }, opts: TurnOptions = {}) {
  if (busy.value || running.value) return
  const kind = opts.kind || mode.value
  if (!keyId.value) return appStore.showError(t('chat.pickKey'))
  if (kind === 'text' && !model.value) return appStore.showError(t('chat.pickModel'))
  if (kind === 'image' && !canDraw.value) return appStore.showError(t('chat.drawNeedsGpt'))

  busy.value = true
  const fromComposer = !opts.regenerate && !opts.editId
  if (fromComposer) composer.value?.clear()
  let c = conv.value
  if (!c) {
    try {
      c = await chatAPI.createConversation({ system_prompt: draftSystemPrompt.value })
    } catch (err) {
      busy.value = false
      if (fromComposer) composer.value?.restore(payload.content, payload.files)
      return appStore.showError(errMsg(err, t('chat.sendFailed')))
    }
    conv.value = c
    upsertConversation(c, true)
    void router.replace({ name: 'Chat', query: { c: String(c.id) } })
  }

  // Show the turn at once; the server's copies replace these.
  const before = messages.value.slice()
  if (opts.regenerate) {
    let i = messages.value.length - 1
    while (i >= 0 && messages.value[i].role !== 'user') i--
    messages.value.splice(i + 1)
  } else {
    if (opts.editId) {
      const i = messages.value.findIndex((m) => m.id === opts.editId)
      if (i >= 0) messages.value.splice(i)
    }
    messages.value.push(tempMessage('user', kind, payload.content, payload.files))
  }
  messages.value.push(tempMessage('assistant', kind))
  const reply = messages.value[messages.value.length - 1]
  stick.value = true
  void nextTick(() => scrollToBottom())

  // Pieces of text are applied once per frame, so long answers don't re-render on every token.
  let pendingText = ''
  let pendingReasoning = ''
  let frame = 0
  const flush = () => {
    frame = 0
    reply.content += pendingText
    reply.reasoning = (reply.reasoning || '') + pendingReasoning
    pendingText = ''
    pendingReasoning = ''
  }

  controller = new AbortController()
  let started = false
  try {
    await chatAPI.sendTurn(
      c.id,
      {
        key_id: keyId.value,
        model: kind === 'text' ? model.value : '',
        mode: kind,
        content: payload.content,
        files: payload.files,
        image_size: imageSize.value,
        regenerate: opts.regenerate,
        edit_id: opts.editId
      },
      (ev) => {
        if (ev.user) {
          started = true
          const i = messages.value.length - 2
          if (i >= 0 && messages.value[i].role === 'user') messages.value[i] = ev.user
        }
        if (ev.assistant) Object.assign(reply, ev.assistant, { content: '', reasoning: '' })
        if (ev.conversation) upsertConversation(ev.conversation, true)
        if (ev.delta || ev.reasoning) {
          pendingText += ev.delta || ''
          pendingReasoning += ev.reasoning || ''
          if (!frame) frame = requestAnimationFrame(flush)
        }
        if (ev.done) {
          if (frame) cancelAnimationFrame(frame)
          flush()
          Object.assign(reply, ev.done)
        }
      },
      controller.signal
    )
    if (reply.status === 'pending') {
      // The stream ended without the final event: read what the server saved.
      setTimeout(() => conv.value?.id === c!.id && openConversation(c!.id, true), 800)
    }
  } catch (err) {
    if (frame) cancelAnimationFrame(frame)
    flush()
    const aborted = controller?.signal.aborted
    if (!started) {
      messages.value = before
      if (!aborted) {
        if (fromComposer) composer.value?.restore(payload.content, payload.files)
        appStore.showError(errMsg(err, t('chat.sendFailed')))
      }
    } else if (aborted) {
      if (reply.kind === 'image') {
        // The picture is still being made on the server.
        running.value = true
        appStore.showInfo(t('chat.drawContinues'))
        pollTimer = setTimeout(() => openConversation(c!.id, true), 4000)
      } else {
        reply.status = 'stopped'
      }
    } else {
      reply.status = 'error'
      reply.error = t('chat.connectionLost')
      setTimeout(() => conv.value?.id === c!.id && openConversation(c!.id, true), 1500)
    }
  } finally {
    busy.value = false
    controller = null
  }
}

function stop() {
  controller?.abort()
}

function regenerate() {
  const lastUser = [...messages.value].reverse().find((m) => m.role === 'user')
  void send({ content: '', files: [] }, { regenerate: true, kind: lastUser?.kind || mode.value })
}

function editAndSend(m: ChatMessage, content: string) {
  void send({ content, files: m.files }, { editId: m.id, kind: m.kind })
}

// The viewer shows the preview at once and swaps in the original when it arrives.
async function openImage(name: string) {
  try {
    lightbox.value = await chatAPI.fileURL(name)
    const full = await chatAPI.fileURL(name, false)
    if (lightbox.value) lightbox.value = full
  } catch {
    // the preview stays
  }
}

function editImage(name: string) {
  if (canDraw.value) mode.value = 'image'
  composer.value?.addExisting(name)
  composer.value?.focus()
}

// Lifecycle -------------------------------------------------------------------------------------

function onDocClick(e: MouseEvent) {
  if (menuId.value && !(e.target as HTMLElement).closest('nav')) menuId.value = 0
}

onMounted(async () => {
  document.addEventListener('click', onDocClick)
  const id = Number(route.query.c)
  await Promise.all([loadKeys(), loadList()])
  if (id) await openConversation(id)
})

watch(
  () => route.query.c,
  (id) => {
    const n = Number(id)
    if (n && n !== conv.value?.id) void openConversation(n)
    else if (!id && conv.value && !busy.value) newChat()
  }
)

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  clearTimeout(pollTimer)
  clearTimeout(searchTimer)
  controller?.abort()
})
</script>

<style scoped>
.pick {
  @apply rounded-lg border-0 bg-gray-100 py-1.5 pl-2.5 pr-7 text-xs font-medium text-gray-700 focus:ring-1 focus:ring-primary-400 dark:bg-dark-800 dark:text-dark-200;
}
.menu-item {
  @apply flex w-full items-center gap-2 px-3 py-1.5 text-left text-gray-700 hover:bg-gray-50 dark:text-dark-200 dark:hover:bg-dark-700;
}
</style>
