<template>
  <div class="card p-4" data-testid="ca-settings">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.conversationArchive.settings.title') }}</h3>
        <span class="badge" :class="STATUS_CLASS[statusKey]" data-testid="ca-status-chip">{{ t(`admin.conversationArchive.status.${statusKey}`) }}</span>
      </div>
      <button v-if="config" type="button" class="btn btn-secondary btn-sm" data-testid="ca-settings-toggle" @click="collapsed = !collapsed">
        {{ collapsed ? t('admin.conversationArchive.settings.expand') : t('admin.conversationArchive.settings.collapse') }}
      </button>
    </div>

    <div v-if="loadError" class="mt-2 text-sm text-red-600">{{ t('admin.conversationArchive.loadFailed') }}：{{ loadError }}</div>

    <!-- Status block (always visible once loaded) -->
    <dl v-if="config" class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm md:grid-cols-3 lg:grid-cols-6" data-testid="ca-status-block">
      <div>
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.queue') }}</dt>
        <dd class="text-gray-900 dark:text-white">{{ t('admin.conversationArchive.statusBlock.queueValue', { n: st.queue_len, size: formatBytes(st.queue_bytes || 0, 1) }) }}</dd>
      </div>
      <div>
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.dropped') }}</dt>
        <dd :class="st.dropped_total > 0 ? 'text-amber-600' : 'text-gray-900 dark:text-white'">{{ st.dropped_total }}</dd>
      </div>
      <div v-if="st.failed_total !== undefined">
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.failed') }}</dt>
        <dd :class="(st.failed_total ?? 0) > 0 ? 'text-red-600' : 'text-gray-900 dark:text-white'" data-testid="ca-failed-total">{{ st.failed_total }}</dd>
      </div>
      <div v-if="st.active !== undefined">
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.active') }}</dt>
        <dd :class="st.active ? 'text-emerald-600' : 'text-gray-500'" data-testid="ca-active">
          {{ st.active ? t('admin.conversationArchive.statusBlock.activeYes') : t('admin.conversationArchive.statusBlock.activeNo') }}
        </dd>
      </div>
      <div v-if="st.lifecycle_applied !== undefined || st.lifecycle_error" class="col-span-2">
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.lifecycle') }}</dt>
        <dd v-if="st.lifecycle_error" class="break-words text-red-600" data-testid="ca-lifecycle-status">{{ st.lifecycle_error }}</dd>
        <dd v-else :class="st.lifecycle_applied ? 'text-emerald-600' : 'text-gray-500'" data-testid="ca-lifecycle-status">
          {{ st.lifecycle_applied ? t('admin.conversationArchive.statusBlock.lifecycleApplied') : t('admin.conversationArchive.statusBlock.lifecycleNotApplied') }}
        </dd>
      </div>
      <div>
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.uploadedToday') }}</dt>
        <dd class="text-gray-900 dark:text-white">{{ st.uploaded_today }}</dd>
      </div>
      <div>
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.lastUpload') }}</dt>
        <dd class="text-gray-900 dark:text-white">{{ st.last_upload_at ? formatDateTime(st.last_upload_at) : t('admin.conversationArchive.statusBlock.never') }}</dd>
      </div>
      <div class="col-span-2">
        <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.statusBlock.lastError') }}</dt>
        <dd v-if="st.last_error" class="break-words text-red-600" data-testid="ca-last-error">
          {{ st.last_error }}<span v-if="st.last_error_at" class="ml-1 text-xs text-gray-500">（{{ formatDateTime(st.last_error_at) }}）</span>
        </dd>
        <dd v-else class="text-gray-900 dark:text-white">{{ t('admin.conversationArchive.statusBlock.none') }}</dd>
      </div>
    </dl>

    <form v-if="config && !collapsed" class="mt-4 space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" data-testid="ca-settings-form" @submit.prevent="askSave">
      <p class="rounded-lg bg-red-50 px-3 py-2 text-sm font-medium text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="ca-red-notice">
        {{ notice }}
      </p>

      <label class="inline-flex items-center gap-2 text-sm text-gray-800 dark:text-dark-100">
        <input v-model="form.enabled" type="checkbox" class="h-4 w-4" data-testid="ca-enabled" />
        <span>{{ t('admin.conversationArchive.settings.enabled') }}</span>
      </label>

      <p class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('admin.conversationArchive.settings.cosHint', { endpoint: COS_ENDPOINT, region: COS_REGION }) }}
      </p>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.endpoint') }}</label>
          <input v-model.trim="form.endpoint" class="input w-full" :placeholder="COS_ENDPOINT" data-testid="ca-endpoint" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.region') }}</label>
          <input v-model.trim="form.region" class="input w-full" :placeholder="COS_REGION" data-testid="ca-region" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.bucket') }}</label>
          <input v-model.trim="form.bucket" class="input w-full" placeholder="archive-1250000000" data-testid="ca-bucket" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.prefix') }}</label>
          <input v-model.trim="form.prefix" class="input w-full" placeholder="archive/" data-testid="ca-prefix" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.accessKeyId') }}</label>
          <input v-model.trim="form.access_key_id" class="input w-full" autocomplete="off" data-testid="ca-access-key-id" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.conversationArchive.settings.secretAccessKey') }}</label>
          <input
            v-model="form.secret_access_key"
            type="password"
            class="input w-full"
            autocomplete="new-password"
            :placeholder="config.secret_configured ? t('admin.conversationArchive.settings.secretConfigured') : ''"
            data-testid="ca-secret"
          />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ t('admin.conversationArchive.settings.retentionDays') }}
            <span class="font-normal text-gray-400">（{{ t('admin.conversationArchive.settings.retentionHint') }}）</span>
          </label>
          <input v-model.number="form.retention_days" type="number" min="7" max="730" class="input w-full" data-testid="ca-retention" />
        </div>
        <div class="flex flex-col justify-end gap-2 pb-1">
          <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.force_path_style" type="checkbox" class="h-4 w-4" />
            <span>{{ t('admin.conversationArchive.settings.forcePathStyle') }}</span>
          </label>
          <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.sse" type="checkbox" class="h-4 w-4" />
            <span>{{ t('admin.conversationArchive.settings.sse') }}</span>
          </label>
        </div>
      </div>

      <p v-if="dirty" class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.settings.unsavedHint') }}</p>

      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" data-testid="ca-test" @click="runTest">
          {{ testing ? t('common.loading') : t('admin.conversationArchive.settings.test') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy || !config.bucket" data-testid="ca-lifecycle" @click="lifecycleConfirmOpen = true">
          {{ t('admin.conversationArchive.settings.lifecycle') }}
        </button>
        <button type="submit" class="btn btn-primary btn-sm" :disabled="busy" data-testid="ca-save">
          {{ saving ? t('common.loading') : t('admin.conversationArchive.settings.save') }}
        </button>
      </div>
    </form>

    <ConfirmDialog
      :show="enableConfirmOpen"
      :title="t('admin.conversationArchive.settings.confirmEnableTitle')"
      :message="enableNotice"
      :confirm-text="t('admin.conversationArchive.settings.confirmEnable')"
      danger
      data-testid="ca-enable-confirm"
      @confirm="confirmEnable"
      @cancel="enableConfirmOpen = false"
    />
    <ConfirmDialog
      :show="lifecycleConfirmOpen"
      :title="t('admin.conversationArchive.settings.lifecycleTitle')"
      :message="t('admin.conversationArchive.settings.lifecycleConfirm', { prefix: config?.prefix || 'archive/', days: (config?.retention_days || 180) + 1 })"
      data-testid="ca-lifecycle-confirm"
      @confirm="runLifecycle"
      @cancel="lifecycleConfirmOpen = false"
    />
    <TotpStepUpDialog :controller="stepUp" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import {
  applyLifecycle,
  getConfig,
  testConnection,
  updateConfig,
  type ArchiveConfig,
  type ArchiveConfigUpdate,
  type ArchiveStatus
} from '@/api/admin/conversationArchive'
import { isStepUpBlocked, isStepUpCancelled, stepUpBlockReason, useStepUp } from '@/composables/useStepUp'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'

const COS_ENDPOINT = 'https://cos.ap-seoul.myqcloud.com'
const COS_REGION = 'ap-seoul'

type StatusKey = 'unconfigured' | 'enabled' | 'paused' | 'error'
const STATUS_CLASS: Record<StatusKey, string> = {
  unconfigured: 'badge-gray',
  enabled: 'badge-success',
  paused: 'badge-warning',
  error: 'badge-danger'
}

const { t } = useI18n()
const appStore = useAppStore()
const stepUp = useStepUp()

const config = ref<ArchiveConfig | null>(null)
const loadError = ref('')
const collapsed = ref(false)
const saving = ref(false)
const testing = ref(false)
const lifecycleRunning = ref(false)
const enableConfirmOpen = ref(false)
const lifecycleConfirmOpen = ref(false)
const busy = computed(() => saving.value || testing.value || lifecycleRunning.value)

const emptyForm = (): ArchiveConfigUpdate => ({
  enabled: false,
  endpoint: '',
  region: '',
  bucket: '',
  prefix: 'archive/',
  access_key_id: '',
  secret_access_key: '',
  force_path_style: false,
  retention_days: 180,
  sse: true
})
const form = ref<ArchiveConfigUpdate>(emptyForm())

// failed_total / active / lifecycle_* stay undefined unless the backend sends them
const EMPTY_STATUS: ArchiveStatus = {
  queue_len: 0,
  queue_bytes: 0,
  dropped_total: 0,
  uploaded_today: 0,
  last_upload_at: null,
  last_error: '',
  last_error_at: null
}
const st = computed<ArchiveStatus>(() => ({ ...EMPTY_STATUS, ...(config.value?.status || {}) }))

const statusKey = computed<StatusKey>(() => {
  const c = config.value
  if (!c || !c.bucket || !c.endpoint || !c.access_key_id || !c.secret_configured) return 'unconfigured'
  if (!c.enabled) return 'paused'
  const s = st.value
  if (s.last_error) {
    const errAt = s.last_error_at ? Date.parse(s.last_error_at) : 0
    const okAt = s.last_upload_at ? Date.parse(s.last_upload_at) : 0
    if (!okAt || errAt >= okAt) return 'error'
  }
  return 'enabled'
})

const retentionForNotice = computed(() => {
  const n = Number(form.value.retention_days)
  return Number.isFinite(n) && n > 0 ? n : config.value?.retention_days || 180
})
const notice = computed(() => t('admin.conversationArchive.settings.notice', { days: retentionForNotice.value }))
const enableNotice = computed(() => notice.value)

const FIELDS = ['enabled', 'endpoint', 'region', 'bucket', 'prefix', 'access_key_id', 'force_path_style', 'retention_days', 'sse'] as const
const dirty = computed(() => {
  const c = config.value
  if (!c) return false
  if (form.value.secret_access_key) return true
  return FIELDS.some((k) => form.value[k] !== c[k])
})

function fillForm(c: ArchiveConfig) {
  form.value = {
    enabled: !!c.enabled,
    endpoint: c.endpoint || '',
    region: c.region || '',
    bucket: c.bucket || '',
    prefix: c.prefix || 'archive/',
    access_key_id: c.access_key_id || '',
    secret_access_key: '',
    force_path_style: !!c.force_path_style,
    retention_days: c.retention_days || 180,
    sse: c.sse !== false
  }
}

async function load(initial = false) {
  loadError.value = ''
  try {
    const c = await getConfig()
    config.value = c
    fillForm(c)
    if (initial) collapsed.value = !!(c.enabled && c.secret_configured)
  } catch (e) {
    loadError.value = extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed'))
  }
}

function reportError(e: unknown) {
  if (isStepUpBlocked(e)) {
    appStore.showError(stepUpBlockReason(e) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN' ? t('stepUp.adminApiKeyForbidden') : t('stepUp.notEnabled'))
    return
  }
  appStore.showError(extractApiErrorMessage(e, t('errors.networkError')))
}

function validate(): boolean {
  const days = Number(form.value.retention_days)
  if (!Number.isInteger(days) || days < 7 || days > 730) {
    appStore.showError(t('admin.conversationArchive.settings.retentionInvalid'))
    return false
  }
  if (form.value.enabled) {
    const hasSecret = !!form.value.secret_access_key || !!config.value?.secret_configured
    if (!form.value.endpoint || !form.value.bucket || !form.value.access_key_id || !hasSecret) {
      appStore.showError(t('admin.conversationArchive.settings.required'))
      return false
    }
  }
  return true
}

/** Turning archiving on repeats the red notice in a confirm first. */
function askSave() {
  if (!validate()) return
  if (form.value.enabled && !config.value?.enabled) {
    enableConfirmOpen.value = true
    return
  }
  void save()
}

function confirmEnable() {
  enableConfirmOpen.value = false
  void save()
}

async function save() {
  saving.value = true
  try {
    const body: ArchiveConfigUpdate = { ...form.value, retention_days: Number(form.value.retention_days) }
    // An empty secret keeps the stored one (the API module drops the field).
    const c = await stepUp.run(() => updateConfig(body))
    config.value = c
    fillForm(c)
    appStore.showSuccess(t('admin.conversationArchive.settings.saved'))
  } catch (e) {
    if (!isStepUpCancelled(e)) reportError(e)
  } finally {
    saving.value = false
  }
}

async function runTest() {
  testing.value = true
  try {
    const body: ArchiveConfigUpdate = { ...form.value, retention_days: Number(form.value.retention_days) }
    const r = await stepUp.run(() => testConnection(body))
    if (r && r.ok === false) appStore.showError(r.message || t('errors.networkError'))
    else appStore.showSuccess(r?.message || t('admin.conversationArchive.settings.testOk'))
  } catch (e) {
    if (!isStepUpCancelled(e)) reportError(e)
  } finally {
    testing.value = false
  }
}

async function runLifecycle() {
  lifecycleConfirmOpen.value = false
  lifecycleRunning.value = true
  try {
    const r = await stepUp.run(() => applyLifecycle())
    if (r && r.ok === false) appStore.showError(r.message || t('errors.networkError'))
    else appStore.showSuccess(t('admin.conversationArchive.settings.lifecycleOk'))
  } catch (e) {
    if (!isStepUpCancelled(e)) reportError(e)
  } finally {
    lifecycleRunning.value = false
  }
}

onMounted(() => load(true))

defineExpose({ reload: load })
</script>
