<template>
  <div class="space-y-4" data-testid="ui-content-log">
    <div class="space-y-2 rounded-lg bg-gray-100 px-4 py-3 text-sm text-gray-700 dark:bg-dark-800 dark:text-dark-200" data-testid="ui-cl-notice">
      <p>{{ redactSecrets === false ? t('admin.userInsights.contentLog.noticeRaw') : t('admin.userInsights.contentLog.notice') }}</p>
      <label v-if="redactSecrets !== null" class="inline-flex cursor-pointer items-center gap-2 text-gray-900 dark:text-white">
        <input
          type="checkbox"
          class="h-4 w-4"
          :checked="redactSecrets"
          :disabled="configSaving"
          data-testid="ui-cl-redact"
          @click.prevent="askRedact(!redactSecrets)"
        />
        <span>{{ t('admin.userInsights.contentLog.redactLabel') }}</span>
      </label>
    </div>

    <!-- Targets -->
    <div class="card p-4" data-testid="ui-cl-targets">
      <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.contentLog.targetsTitle') }}</h3>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="targetsLoading" @click="loadTargets">{{ t('admin.userInsights.refresh') }}</button>
      </div>
      <div v-if="targetsError" class="mb-2 text-sm text-red-600">{{ t('admin.userInsights.loadFailed') }}：{{ targetsError }}</div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
              <th class="py-1">{{ t('admin.userInsights.col.user') }}</th>
              <th>{{ t('admin.userInsights.contentLog.col.note') }}</th>
              <th>{{ t('admin.userInsights.contentLog.col.createdBy') }}</th>
              <th>{{ t('admin.userInsights.contentLog.col.since') }}</th>
              <th>{{ t('admin.userInsights.contentLog.col.logCount') }}</th>
              <th>{{ t('admin.userInsights.contentLog.col.lastLogged') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="tg in targets"
              :key="tg.user_id"
              class="border-t border-gray-100 dark:border-dark-700"
              :class="tg.user_id === userId ? 'bg-primary-50/60 dark:bg-primary-900/20' : ''"
              :data-testid="`ui-cl-target-${tg.user_id}`"
            >
              <td class="py-1.5">
                <div class="text-gray-900 dark:text-white">{{ tg.email || tg.username || `#${tg.user_id}` }}</div>
                <div class="text-xs text-gray-500 dark:text-dark-400">#{{ tg.user_id }}<template v-if="tg.username && tg.email"> · {{ tg.username }}</template></div>
              </td>
              <td class="max-w-xs break-words text-gray-700 dark:text-dark-200">{{ tg.note || '–' }}</td>
              <td class="text-xs">{{ tg.created_by_email || (tg.created_by ? `#${tg.created_by}` : '–') }}</td>
              <td class="whitespace-nowrap text-xs">{{ formatDateTimeToMinute(tg.created_at) }}</td>
              <td>{{ tg.log_count }}</td>
              <td class="whitespace-nowrap text-xs">{{ tg.last_logged_at ? formatDateTimeToMinute(tg.last_logged_at) : '–' }}</td>
              <td class="whitespace-nowrap text-right">
                <button type="button" class="btn btn-secondary btn-sm mr-1" :data-testid="`ui-cl-view-${tg.user_id}`" @click="emit('select', tg.user_id)">
                  {{ t('admin.userInsights.contentLog.view') }}
                </button>
                <button type="button" class="btn btn-danger btn-sm" :data-testid="`ui-cl-stop-${tg.user_id}`" @click="askDisable(tg.user_id)">
                  {{ t('admin.userInsights.contentLog.stop') }}
                </button>
              </td>
            </tr>
            <tr v-if="!targets.length && !targetsLoading"><td colspan="7" class="py-2 text-gray-500">{{ t('admin.userInsights.contentLog.targetsEmpty') }}</td></tr>
          </tbody>
        </table>
      </div>

      <form class="mt-3 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3 dark:border-dark-700" data-testid="ui-cl-enable-form" @submit.prevent="askEnableFromForm">
        <input
          v-model="formUserId"
          type="number"
          min="1"
          class="input w-32"
          :placeholder="t('admin.userInsights.contentLog.userIdPlaceholder')"
          data-testid="ui-cl-enable-user"
        />
        <input
          v-model="formNote"
          type="text"
          class="input max-w-sm flex-1"
          :placeholder="t('admin.userInsights.contentLog.notePlaceholder')"
          data-testid="ui-cl-enable-note"
        />
        <button type="submit" class="btn btn-primary btn-sm" :disabled="acting" data-testid="ui-cl-enable">{{ t('admin.userInsights.contentLog.enable') }}</button>
      </form>
    </div>

    <!-- Selected user's log -->
    <div v-if="userId" class="card space-y-3 p-4" data-testid="ui-cl-log">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.contentLog.logTitle', { user: selectedName }) }}</h3>
          <span v-if="selectedEnabled" class="badge badge-success">{{ t('admin.userInsights.contentLog.recording') }}</span>
          <span v-else class="badge badge-gray">{{ t('admin.userInsights.contentLog.notRecording') }}</span>
          <span v-if="retentionDays" class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.contentLog.retention', { days: retentionDays }) }}</span>
        </div>
        <form class="flex flex-wrap items-center gap-2" data-testid="ui-cl-session-form" @submit.prevent="applySession(sessionInput)">
          <input
            v-model="sessionInput"
            type="text"
            class="input w-56 font-mono text-xs"
            :placeholder="t('admin.userInsights.contentLog.sessionPlaceholder')"
            data-testid="ui-cl-session-input"
          />
          <button type="submit" class="btn btn-secondary btn-sm" :disabled="logsLoading">{{ t('admin.userInsights.contentLog.filter') }}</button>
          <button v-if="session" type="button" class="btn btn-secondary btn-sm" :disabled="logsLoading" data-testid="ui-cl-session-clear" @click="applySession('')">
            {{ t('admin.userInsights.contentLog.clearFilter') }}
          </button>
        </form>
      </div>

      <div v-if="logsError" class="text-sm text-red-600">{{ t('admin.userInsights.loadFailed') }}：{{ logsError }}</div>

      <div class="space-y-3" data-testid="ui-cl-rows">
        <div
          v-for="(row, i) in logs"
          :key="row.id"
          class="rounded-lg border border-gray-100 p-3 dark:border-dark-700"
          :data-testid="`ui-cl-row-${row.id}`"
        >
          <div class="mb-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
            <span class="whitespace-nowrap font-medium text-gray-700 dark:text-dark-200">{{ formatDateTimeToMinute(row.created_at) }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ row.model || '–' }}</span>
            <span class="font-mono">{{ row.inbound_endpoint || '–' }}</span>
            <span class="font-mono" :title="row.user_agent">{{ shortUA(row.user_agent) }}</span>
            <span :title="t('admin.userInsights.contentLog.tokensTitle')">{{ t('admin.userInsights.contentLog.tokens', { in: row.input_tokens, out: row.output_tokens, cached: row.cache_read_tokens }) }}</span>
            <span v-if="row.ip_address" class="font-mono">{{ row.ip_address }}</span>
            <button
              v-if="row.session_id"
              type="button"
              class="max-w-[14rem] truncate rounded bg-gray-100 px-1.5 py-0.5 font-mono text-gray-700 hover:bg-primary-100 dark:bg-dark-700 dark:text-dark-200 dark:hover:bg-primary-900/30"
              :title="t('admin.userInsights.contentLog.sessionChipTitle', { session: row.session_id })"
              data-testid="ui-cl-session-chip"
              @click="applySession(row.session_id)"
            >
              {{ row.session_id }}
            </button>
          </div>

          <div class="space-y-2 text-sm">
            <!-- 提问 -->
            <div>
              <div class="mb-0.5 text-xs font-semibold text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.contentLog.question') }}</div>
              <p v-if="sameAsAbove(i)" class="text-gray-400 dark:text-dark-500" data-testid="ui-cl-same">{{ t('admin.userInsights.contentLog.sameAsAbove') }}</p>
              <p v-else-if="!row.user_text" class="text-gray-400 dark:text-dark-500">{{ t('admin.userInsights.contentLog.noText') }}</p>
              <template v-else>
                <div
                  class="whitespace-pre-wrap break-words text-gray-900 dark:text-white"
                  :class="isLong(row.user_text, 6, 500) && !expanded[`${row.id}:q`] ? 'line-clamp-6' : ''"
                  data-testid="ui-cl-question"
                >{{ row.user_text }}</div>
                <button v-if="isLong(row.user_text, 6, 500)" type="button" class="text-xs text-primary-600 hover:underline dark:text-primary-400" @click="toggle(`${row.id}:q`)">
                  {{ expanded[`${row.id}:q`] ? t('admin.userInsights.contentLog.collapse') : t('admin.userInsights.contentLog.expand') }}
                </button>
              </template>
            </div>

            <!-- 工具调用 -->
            <div v-if="row.tool_calls?.length">
              <div class="mb-0.5 text-xs font-semibold text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.contentLog.tools', { n: row.tool_calls.length }) }}</div>
              <div class="space-y-1.5">
                <div v-for="(tc, j) in row.tool_calls" :key="j" data-testid="ui-cl-tool">
                  <span class="badge badge-primary font-mono">{{ tc.name || '?' }}</span>
                  <pre
                    v-if="argText(tc.arguments)"
                    class="mt-1 overflow-x-auto whitespace-pre-wrap break-all rounded bg-gray-50 p-2 font-mono text-xs text-gray-800 dark:bg-dark-900 dark:text-dark-200"
                    :class="isLong(argText(tc.arguments), 8, 600) && !expanded[`${row.id}:t${j}`] ? 'max-h-40 overflow-y-hidden' : ''"
                  >{{ argText(tc.arguments) }}</pre>
                  <button
                    v-if="isLong(argText(tc.arguments), 8, 600)"
                    type="button"
                    class="text-xs text-primary-600 hover:underline dark:text-primary-400"
                    @click="toggle(`${row.id}:t${j}`)"
                  >
                    {{ expanded[`${row.id}:t${j}`] ? t('admin.userInsights.contentLog.collapse') : t('admin.userInsights.contentLog.expand') }}
                  </button>
                </div>
              </div>
            </div>

            <!-- 回复开头 -->
            <div>
              <div class="mb-0.5 text-xs font-semibold text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.contentLog.reply') }}</div>
              <div v-if="row.reply_head" class="whitespace-pre-wrap break-words text-gray-700 dark:text-dark-200">{{ row.reply_head }}</div>
              <p v-else class="text-gray-400 dark:text-dark-500">–</p>
            </div>
          </div>
        </div>

        <p v-if="!logs.length && !logsLoading && !logsError" class="py-2 text-sm text-gray-500 dark:text-dark-400" data-testid="ui-cl-empty">
          {{ t('admin.userInsights.contentLog.empty') }}
        </p>
        <p v-if="logsLoading" class="py-2 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.contentLog.loading') }}</p>
      </div>

      <div v-if="nextBeforeId && !logsLoading" class="text-center">
        <button type="button" class="btn btn-secondary btn-sm" data-testid="ui-cl-more" @click="loadMore">{{ t('admin.userInsights.contentLog.loadMore') }}</button>
      </div>
    </div>
    <p v-else class="card p-4 text-sm text-gray-500 dark:text-dark-400" data-testid="ui-cl-pick">{{ t('admin.userInsights.contentLog.pickUser') }}</p>

    <!-- Access trail -->
    <div class="card p-4" data-testid="ui-cl-access">
      <button type="button" class="flex w-full items-center justify-between text-left" data-testid="ui-cl-access-toggle" @click="toggleAccess">
        <span class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('admin.userInsights.contentLog.access.title') }}
          <span class="text-xs font-normal text-gray-500 dark:text-dark-400">{{ userId ? `#${userId}` : t('admin.userInsights.contentLog.access.all') }}</span>
        </span>
        <span class="text-xs text-primary-600 dark:text-primary-400">{{ accessOpen ? t('admin.userInsights.contentLog.collapse') : t('admin.userInsights.contentLog.expand') }}</span>
      </button>
      <div v-if="accessOpen" class="mt-2">
        <div v-if="accessError" class="mb-2 text-sm text-red-600">{{ t('admin.userInsights.loadFailed') }}：{{ accessError }}</div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                <th class="py-1">{{ t('admin.userInsights.col.time') }}</th>
                <th>{{ t('admin.userInsights.contentLog.access.admin') }}</th>
                <th>{{ t('admin.userInsights.contentLog.access.action') }}</th>
                <th>{{ t('admin.userInsights.contentLog.access.target') }}</th>
                <th>{{ t('admin.userInsights.col.detail') }}</th>
                <th>{{ t('admin.userInsights.col.ip') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in access" :key="a.id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="whitespace-nowrap py-1 text-xs">{{ formatDateTimeToMinute(a.created_at) }}</td>
                <td>{{ a.admin_email || `#${a.admin_id}` }}</td>
                <td class="whitespace-nowrap">{{ accessActionLabel(a.action) }}</td>
                <td>{{ a.target_email || (a.target_user_id ? `#${a.target_user_id}` : '–') }}</td>
                <td class="break-all text-gray-700 dark:text-dark-200">{{ accessDetail(a) }}</td>
                <td class="font-mono text-xs">{{ a.ip_address || '–' }}</td>
              </tr>
              <tr v-if="!access.length && !accessLoading"><td colspan="6" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <ConfirmDialog
      :show="pending !== null"
      :title="pending?.kind === 'enable' ? t('admin.userInsights.contentLog.enableTitle') : t('admin.userInsights.contentLog.disableTitle')"
      :message="pendingMessage"
      :confirm-text="pending?.kind === 'enable' ? t('admin.userInsights.contentLog.enable') : t('admin.userInsights.contentLog.stop')"
      :danger="pending?.kind === 'disable'"
      @confirm="confirmPending"
      @cancel="cancelPending"
    />
    <ConfirmDialog
      :show="pendingRedact !== null"
      :title="pendingRedact ? t('admin.userInsights.contentLog.redactOnTitle') : t('admin.userInsights.contentLog.redactOffTitle')"
      :message="pendingRedact ? t('admin.userInsights.contentLog.redactOnMessage') : t('admin.userInsights.contentLog.redactOffMessage')"
      :confirm-text="t('common.confirm')"
      :danger="pendingRedact === false"
      @confirm="confirmRedact"
      @cancel="pendingRedact = null"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import {
  disableContentLog,
  enableContentLog,
  getContentLogConfig,
  updateContentLogConfig,
  listContentLog,
  listContentLogAccess,
  listContentLogTargets,
  type ContentLogAccess,
  type ContentLogAccessAction,
  type ContentLogEntry,
  type ContentLogTarget
} from '@/api/admin/userInsights'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTimeToMinute } from '@/utils/format'

const props = defineProps<{ userId: number | null }>()
const emit = defineEmits<{ (e: 'select', userId: number): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const PAGE_SIZE = 50

/** A short client name from a User-Agent, e.g. "axonhub/1.0" or "Codex Desktop/0.162.0". */
function shortUA(ua: string): string {
  const s = (ua || '').trim()
  if (!s) return '–'
  const m = s.match(/^[^/()]*\/[^\s;()]*/)
  const head = (m ? m[0] : s).trim()
  return head.length > 28 ? `${head.slice(0, 27)}…` : head
}

function argText(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') {
    const s = v.trim()
    if (!s) return ''
    try {
      const parsed: unknown = JSON.parse(s)
      if (parsed && typeof parsed === 'object') return JSON.stringify(parsed, null, 2)
    } catch {
      // not JSON: show as-is
    }
    return s
  }
  return JSON.stringify(v, null, 2)
}

function isLong(s: string, lines: number, chars: number): boolean {
  return s.length > chars || s.split('\n').length > lines
}

// 遮盖密钥 switch: null until loaded.
const redactSecrets = ref<boolean | null>(null)
const pendingRedact = ref<boolean | null>(null)
const configSaving = ref(false)

async function loadConfig(): Promise<void> {
  try {
    redactSecrets.value = (await getContentLogConfig()).redact_secrets !== false
  } catch {
    redactSecrets.value = null
  }
}

function askRedact(on: boolean): void {
  if (!configSaving.value) pendingRedact.value = on
}

async function confirmRedact(): Promise<void> {
  const on = pendingRedact.value
  if (on === null) return
  configSaving.value = true
  try {
    redactSecrets.value = (await updateContentLogConfig({ redact_secrets: on })).redact_secrets !== false
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('admin.userInsights.actions.failed')))
  } finally {
    configSaving.value = false
    pendingRedact.value = null
  }
}

function accessDetail(a: ContentLogAccess): string {
  if (a.action === 'config') {
    if (a.detail === 'redact_secrets=false') return t('admin.userInsights.contentLog.access.redactOff')
    if (a.detail === 'redact_secrets=true') return t('admin.userInsights.contentLog.access.redactOn')
  }
  return a.detail || '–'
}

function accessActionLabel(a: ContentLogAccessAction | string): string {
  return ['view', 'enable', 'disable', 'config'].includes(a) ? t(`admin.userInsights.contentLog.access.actions.${a}`) : a
}

// ---------- targets ----------

const targets = ref<ContentLogTarget[]>([])
const targetsLoading = ref(false)
const targetsError = ref('')
const targetsLoaded = ref(false)

async function loadTargets(): Promise<void> {
  targetsLoading.value = true
  targetsError.value = ''
  try {
    const res = await listContentLogTargets()
    targets.value = res.items || []
    targetsLoaded.value = true
  } catch (e: unknown) {
    targetsError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    targetsLoading.value = false
  }
}

const selectedTarget = computed(() => targets.value.find((x) => x.user_id === props.userId) || null)
const selectedName = computed(() => {
  const tg = selectedTarget.value
  if (tg) return `${tg.email || tg.username || ''} #${tg.user_id}`.trim()
  return `#${props.userId}`
})
const logEnabled = ref<boolean | null>(null)
// The targets list is authoritative once loaded; until then fall back to the log response.
const selectedEnabled = computed(() => (targetsLoaded.value ? !!selectedTarget.value : !!logEnabled.value))

// ---------- enable / disable ----------

const formUserId = ref<string | number>('')
const formNote = ref('')
const pending = ref<{ kind: 'enable' | 'disable'; userId: number; note: string } | null>(null)
const acting = ref(false)

function targetLabel(id: number): string {
  const tg = targets.value.find((x) => x.user_id === id)
  return tg ? `${tg.email || tg.username || ''} #${id}`.trim() : `#${id}`
}

const pendingMessage = computed(() => {
  const p = pending.value
  if (!p) return ''
  return p.kind === 'enable'
    ? t('admin.userInsights.contentLog.enableMessage', { user: targetLabel(p.userId) })
    : t('admin.userInsights.contentLog.disableMessage', { user: targetLabel(p.userId) })
})

function askEnableFromForm(): void {
  const id = Number(formUserId.value)
  if (!Number.isInteger(id) || id <= 0) {
    appStore.showError(t('admin.userInsights.contentLog.invalidUserId'))
    return
  }
  pending.value = { kind: 'enable', userId: id, note: formNote.value.trim() }
}

function askDisable(id: number): void {
  pending.value = { kind: 'disable', userId: id, note: '' }
}

function cancelPending(): void {
  if (!acting.value) pending.value = null
}

async function confirmPending(): Promise<void> {
  const p = pending.value
  if (!p || acting.value) return
  acting.value = true
  try {
    if (p.kind === 'enable') {
      await enableContentLog(p.userId, p.note)
      appStore.showSuccess(t('admin.userInsights.contentLog.enabledToast'))
      formUserId.value = ''
      formNote.value = ''
    } else {
      await disableContentLog(p.userId)
      appStore.showSuccess(t('admin.userInsights.contentLog.disabledToast'))
    }
    if (p.userId === props.userId) logEnabled.value = p.kind === 'enable'
    acting.value = false
    pending.value = null
    await loadTargets()
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.userInsights.actions.failed')))
  } finally {
    acting.value = false
  }
}

// ---------- logs (every fetch is audited: explicit actions only) ----------

const logs = ref<ContentLogEntry[]>([])
const logsLoading = ref(false)
const logsError = ref('')
const nextBeforeId = ref<number | null>(null)
const retentionDays = ref(0)
const session = ref('')
const sessionInput = ref('')
const expanded = ref<Record<string, boolean>>({})
let logsSeq = 0

async function fetchLogs(append: boolean): Promise<void> {
  const uid = props.userId
  if (!uid) return
  const seq = ++logsSeq
  logsLoading.value = true
  logsError.value = ''
  try {
    const res = await listContentLog({
      user_id: uid,
      session_id: session.value || undefined,
      before_id: append ? nextBeforeId.value || undefined : undefined,
      limit: PAGE_SIZE
    })
    if (seq !== logsSeq) return
    const items = res.items || []
    logs.value = append ? [...logs.value, ...items] : items
    nextBeforeId.value = res.next_before_id ?? null
    retentionDays.value = res.retention_days || 0
    logEnabled.value = !!res.enabled
  } catch (e: unknown) {
    if (seq !== logsSeq) return
    logsError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    if (seq === logsSeq) logsLoading.value = false
  }
}

function loadMore(): void {
  if (nextBeforeId.value && !logsLoading.value) void fetchLogs(true)
}

function applySession(s: string): void {
  const v = s.trim()
  sessionInput.value = v
  if (v === session.value && logs.value.length) return
  session.value = v
  void fetchLogs(false)
}

/** Agent loops resend the same question many times: show it once per run of rows. */
function sameAsAbove(i: number): boolean {
  if (i === 0) return false
  const cur = logs.value[i]
  const prev = logs.value[i - 1]
  return !!cur.user_text && cur.user_text === prev.user_text && (cur.session_id || '') === (prev.session_id || '')
}

function toggle(key: string): void {
  expanded.value = { ...expanded.value, [key]: !expanded.value[key] }
}

// ---------- access trail (fetched only when expanded) ----------

const accessOpen = ref(false)
const access = ref<ContentLogAccess[]>([])
const accessLoading = ref(false)
const accessError = ref('')

async function loadAccess(): Promise<void> {
  accessLoading.value = true
  accessError.value = ''
  try {
    const res = await listContentLogAccess(props.userId ? { user_id: props.userId, limit: 100 } : { limit: 100 })
    access.value = res.items || []
  } catch (e: unknown) {
    accessError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    accessLoading.value = false
  }
}

function toggleAccess(): void {
  accessOpen.value = !accessOpen.value
  if (accessOpen.value) void loadAccess()
}

watch(
  () => props.userId,
  (uid) => {
    logs.value = []
    nextBeforeId.value = null
    retentionDays.value = 0
    logEnabled.value = null
    session.value = ''
    sessionInput.value = ''
    expanded.value = {}
    accessOpen.value = false
    access.value = []
    if (uid) void fetchLogs(false)
  },
  { immediate: true }
)

onMounted(() => {
  void loadTargets()
  void loadConfig()
})
</script>
