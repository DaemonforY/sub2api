<template>
  <AppLayout>
    <div class="space-y-4" data-testid="conversation-archive-view">
      <ArchiveSettingsCard />
      <ArchiveStatsStrip />

      <!-- Browser. Every call here is audited server-side, so nothing loads until the admin asks. -->
      <div class="card space-y-3 p-4" data-testid="ca-browser">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.conversationArchive.browser.title') }}</h3>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.browser.auditNotice') }}</p>
        </div>

        <form class="flex flex-wrap items-end gap-2" data-testid="ca-search-form" @submit.prevent="search">
          <input v-model.trim="q" type="text" class="input w-72" :placeholder="t('admin.conversationArchive.browser.q')" data-testid="ca-q" />
          <label class="flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.conversationArchive.browser.from') }}
            <input v-model="fromDate" type="date" class="input w-40" data-testid="ca-from" />
          </label>
          <label class="flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.conversationArchive.browser.to') }}
            <input v-model="toDate" type="date" class="input w-40" data-testid="ca-to" />
          </label>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="sessionsLoading" data-testid="ca-search">{{ t('admin.conversationArchive.browser.search') }}</button>
        </form>

        <form class="flex flex-wrap items-center gap-2" data-testid="ca-by-request-form" @submit.prevent="findByRequest">
          <input
            v-model.trim="requestId"
            type="text"
            class="input w-96 max-w-full font-mono text-xs"
            :placeholder="t('admin.conversationArchive.browser.requestIdPlaceholder')"
            data-testid="ca-request-id"
          />
          <button type="submit" class="btn btn-secondary btn-sm" :disabled="!requestId || lookupLoading" data-testid="ca-find">
            {{ t('admin.conversationArchive.browser.byRequest') }}
          </button>
        </form>

        <div v-if="sessionsError" class="text-sm text-red-600" data-testid="ca-sessions-error">{{ t('admin.conversationArchive.loadFailed') }}：{{ sessionsError }}</div>

        <p v-if="!searched" class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.browser.hint') }}</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-sm" data-testid="ca-sessions">
            <thead>
              <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                <th class="py-1">{{ t('admin.conversationArchive.browser.col.lastAt') }}</th>
                <th>{{ t('admin.conversationArchive.browser.col.user') }}</th>
                <th>{{ t('admin.conversationArchive.browser.col.client') }}</th>
                <th>{{ t('admin.conversationArchive.browser.col.models') }}</th>
                <th class="text-right">{{ t('admin.conversationArchive.browser.col.turns') }}</th>
                <th>{{ t('admin.conversationArchive.browser.col.preview') }}</th>
                <th class="text-right">{{ t('admin.conversationArchive.browser.col.images') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="s in sessions"
                :key="`${s.user_id}:${s.session_key}`"
                class="cursor-pointer border-t border-gray-100 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-800"
                :class="isCurrent(s) ? 'bg-primary-50/60 dark:bg-primary-900/20' : ''"
                data-testid="ca-session-row"
                @click="openSession(s)"
              >
                <td class="whitespace-nowrap py-1.5 text-xs">{{ formatDateTimeToMinute(s.last_at) }}</td>
                <td>
                  <div class="max-w-[14rem] truncate text-gray-900 dark:text-white">{{ s.email || `#${s.user_id}` }}</div>
                  <div class="text-xs text-gray-500 dark:text-dark-400">#{{ s.user_id }}</div>
                </td>
                <td class="font-mono text-xs" :title="s.user_agent">{{ shortUA(s.user_agent) }}</td>
                <td class="max-w-[12rem] truncate font-mono text-xs" :title="(s.models || []).join(', ')">{{ (s.models || []).join(', ') || '–' }}</td>
                <td class="text-right">{{ s.turns }}</td>
                <td class="max-w-md truncate text-gray-700 dark:text-dark-200" :title="s.preview">{{ s.preview || '–' }}</td>
                <td class="text-right">{{ s.image_count || '' }}</td>
              </tr>
              <tr v-if="!sessions.length && !sessionsLoading">
                <td colspan="7" class="py-2 text-gray-500">{{ t('admin.conversationArchive.browser.empty') }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="nextBefore" class="mt-2 text-center">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="sessionsLoading" data-testid="ca-sessions-more" @click="loadMoreSessions">
              {{ t('admin.conversationArchive.browser.loadMore') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Transcript -->
      <div v-if="current" class="card space-y-3 p-4" data-testid="ca-transcript">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="min-w-0">
            <div class="font-semibold text-gray-900 dark:text-white">{{ current.email || `#${current.user_id}` }} <span class="text-xs font-normal text-gray-500">#{{ current.user_id }}</span></div>
            <div class="truncate font-mono text-xs text-gray-500 dark:text-dark-400" :title="current.session_key">{{ current.session_key }}</div>
          </div>
          <div class="flex items-center gap-2">
            <template v-if="anchor">
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.transcript.anchored', { id: anchor.request_id }) }}</span>
              <button
                v-if="current.session_key"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="turnsLoading"
                data-testid="ca-show-latest"
                @click="showLatest"
              >
                {{ t('admin.conversationArchive.transcript.showLatest') }}
              </button>
            </template>
            <button type="button" class="btn btn-secondary btn-sm" data-testid="ca-close" @click="closeSession">{{ t('admin.conversationArchive.transcript.close') }}</button>
          </div>
        </div>

        <div ref="scrollBox" class="max-h-[75vh] space-y-5 overflow-y-auto pr-1">
          <div class="text-center">
            <button
              v-if="nextBeforeId"
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="turnsLoading"
              data-testid="ca-load-earlier"
              @click="loadEarlier"
            >
              {{ turnsLoading ? t('common.loading') : t('admin.conversationArchive.transcript.loadEarlier') }}
            </button>
            <span v-else-if="turns.length" class="text-xs text-gray-400">{{ t('admin.conversationArchive.transcript.noEarlier') }}</span>
          </div>

          <div v-if="turnsError" class="text-sm text-red-600" data-testid="ca-turns-error">{{ t('admin.conversationArchive.loadFailed') }}：{{ turnsError }}</div>

          <ArchiveTurn
            v-for="tn in turns"
            :key="tn.id"
            :turn="tn"
            :highlighted="!!anchor && anchor.id === tn.id"
            @open-image="previewSrc = $event"
          />

          <p v-if="!turns.length && !turnsLoading && !turnsError" class="text-sm text-gray-500">{{ t('admin.conversationArchive.transcript.empty') }}</p>
        </div>
      </div>

      <!-- Larger image -->
      <div
        v-if="previewSrc"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
        data-testid="ca-image-modal"
        @click="previewSrc = ''"
      >
        <img :src="previewSrc" alt="" class="max-h-full max-w-full rounded bg-white object-contain shadow-xl" @click.stop />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ArchiveSettingsCard from '@/components/admin/archive/ArchiveSettingsCard.vue'
import ArchiveStatsStrip from '@/components/admin/archive/ArchiveStatsStrip.vue'
import ArchiveTurn from '@/components/admin/archive/ArchiveTurn.vue'
import {
  getTurnByRequest,
  listSessions,
  listTurns,
  type ArchiveSession,
  type ArchiveTurn as ArchiveTurnItem
} from '@/api/admin/conversationArchive'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { shortUA } from '@/utils/archiveTranscript'
import { formatDateLocalInput, formatDateTimeToMinute } from '@/utils/format'

const SESSION_PAGE = 30
const TURN_PAGE = 20

const { t } = useI18n()
const appStore = useAppStore()

// ---------- Session search ----------

function daysAgo(n: number): Date {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d
}

const q = ref('')
const fromDate = ref(formatDateLocalInput(daysAgo(6)))
const toDate = ref(formatDateLocalInput(new Date()))
const sessions = ref<ArchiveSession[]>([])
const nextBefore = ref<string | null>(null)
const sessionsLoading = ref(false)
const sessionsError = ref('')
const searched = ref(false)
// the filter the current list was loaded with, so 加载更多 continues the same query
let activeQuery: { q?: string; from?: string; to?: string } = {}

/** 'YYYY-MM-DD' (local) → RFC3339 of that local midnight; `endOfDay` gives the next midnight. */
function dayToRFC3339(day: string, endOfDay = false): string | undefined {
  if (!day) return undefined
  const [y, m, d] = day.split('-').map(Number)
  if (!y || !m || !d) return undefined
  return new Date(y, m - 1, endOfDay ? d + 1 : d).toISOString()
}

async function fetchSessions(before?: string) {
  sessionsLoading.value = true
  sessionsError.value = ''
  try {
    const page = await listSessions({ ...activeQuery, before, limit: SESSION_PAGE })
    const items = page?.items || []
    sessions.value = before ? [...sessions.value, ...items] : items
    nextBefore.value = page?.next_before || null
    searched.value = true
  } catch (e) {
    sessionsError.value = extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed'))
  } finally {
    sessionsLoading.value = false
  }
}

function search() {
  activeQuery = {
    q: q.value || undefined,
    from: dayToRFC3339(fromDate.value),
    to: dayToRFC3339(toDate.value, true)
  }
  return fetchSessions()
}

function loadMoreSessions() {
  if (nextBefore.value) return fetchSessions(nextBefore.value)
}

// ---------- Transcript ----------

interface CurrentSession {
  user_id: number
  session_key: string
  email: string
}

const current = ref<CurrentSession | null>(null)
/** oldest first */
const turns = ref<ArchiveTurnItem[]>([])
const nextBeforeId = ref<number | null>(null)
const turnsLoading = ref(false)
const turnsError = ref('')
/** set when opened from 按请求 ID 查找: the transcript ends at this turn */
const anchor = ref<{ id: number; request_id: string } | null>(null)
const previewSrc = ref('')
const scrollBox = ref<HTMLElement | null>(null)
// bumps on every session switch so a late response for the old session is ignored
let generation = 0

function isCurrent(s: ArchiveSession): boolean {
  return !!current.value && current.value.user_id === s.user_id && current.value.session_key === s.session_key
}

async function loadTurns(beforeId?: number, prepend = false) {
  const cur = current.value
  if (!cur) return
  const gen = generation
  turnsLoading.value = true
  turnsError.value = ''
  try {
    const page = await listTurns({ user_id: cur.user_id, session_key: cur.session_key, before_id: beforeId, limit: TURN_PAGE })
    if (gen !== generation) return
    const older = [...(page?.items || [])].reverse() // API is newest first; read oldest first
    if (prepend) {
      const box = scrollBox.value
      const prevHeight = box?.scrollHeight ?? 0
      const prevTop = box?.scrollTop ?? 0
      turns.value = [...older, ...turns.value]
      await nextTick()
      if (box) box.scrollTop = prevTop + (box.scrollHeight - prevHeight)
    } else {
      turns.value = older
      await nextTick()
      if (scrollBox.value) scrollBox.value.scrollTop = scrollBox.value.scrollHeight
    }
    nextBeforeId.value = page?.next_before_id || null
  } catch (e) {
    if (gen === generation) turnsError.value = extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed'))
  } finally {
    if (gen === generation) turnsLoading.value = false
  }
}

function resetTranscript(next: CurrentSession | null) {
  generation++
  previewSrc.value = ''
  turns.value = [] // unmounts the thumbnails, which revokes their object URLs
  nextBeforeId.value = null
  turnsError.value = ''
  turnsLoading.value = false
  anchor.value = null
  current.value = next
}

function openSession(s: ArchiveSession) {
  resetTranscript({ user_id: s.user_id, session_key: s.session_key, email: s.email })
  return loadTurns()
}

function loadEarlier() {
  if (nextBeforeId.value && !turnsLoading.value) return loadTurns(nextBeforeId.value, true)
}

function showLatest() {
  const cur = current.value
  if (!cur) return
  resetTranscript(cur)
  return loadTurns()
}

function closeSession() {
  resetTranscript(null)
}

// ---------- 按请求 ID 查找 ----------

const requestId = ref('')
const lookupLoading = ref(false)

async function findByRequest() {
  const id = requestId.value
  if (!id) return
  lookupLoading.value = true
  try {
    const turn = await getTurnByRequest(id)
    if (turn.user_id && turn.session_key) {
      resetTranscript({ user_id: turn.user_id, session_key: turn.session_key, email: turn.email || '' })
      anchor.value = { id: turn.id, request_id: turn.request_id || id }
      // the page that ends at this turn; 加载更早 continues from there, 查看最新 jumps to the end
      await loadTurns(turn.id + 1)
    } else {
      // the backend didn't say which session it belongs to: show the turn on its own
      resetTranscript({ user_id: turn.user_id || 0, session_key: turn.session_key || '', email: turn.email || '' })
      anchor.value = { id: turn.id, request_id: turn.request_id || id }
      turns.value = [turn]
    }
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed')))
  } finally {
    lookupLoading.value = false
  }
}
</script>
