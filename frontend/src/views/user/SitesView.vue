<template>
  <AppLayout>
    <div class="space-y-4">
      <div v-if="data && !data.quota.available" class="card p-5 text-sm text-gray-600 dark:text-gray-300">{{ t('sites.unavailable') }}</div>

      <template v-else-if="data">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('sites.quota.sites') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ data.quota.used }} / {{ data.quota.max_sites }}</div>
          </div>
          <div class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('sites.quota.free') }}</div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ t('sites.quota.freeValue', { used: data.quota.free_used, total: data.quota.free_sites }) }}</div>
          </div>
          <div class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('sites.quota.extra') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ t('sites.quota.extraValue', { price: data.quota.extra_price }) }}</div>
          </div>
          <div class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('sites.quota.limits') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ t('sites.quota.limitsValue', { mb: data.quota.max_mb, files: data.quota.max_files }) }}</div>
          </div>
        </div>

        <div v-if="!data.quota.subscribed" class="card flex flex-wrap items-center justify-between gap-3 p-4 text-sm">
          <span class="text-amber-700 dark:text-amber-300">{{ data.sites.length ? t('sites.lapsedNotice', { days: data.quota.grace_days }) : t('sites.subscribeFirst') }}</span>
          <router-link to="/purchase" class="btn btn-primary">{{ t('sites.subscribe') }}</router-link>
        </div>

        <div v-else-if="data.quota.used < data.quota.max_sites" class="card space-y-3 p-5">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('sites.create.title') }}</h3>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('sites.create.hint', { domain: data.quota.domain }) }}</p>
          <div class="grid gap-3 md:grid-cols-[1fr_auto_auto] md:items-end">
            <div>
              <label class="input-label">{{ t('sites.create.name') }}</label>
              <input v-model="createTitle" class="input" maxlength="60" :placeholder="t('sites.create.namePlaceholder')" />
            </div>
            <div>
              <label class="input-label">{{ t('sites.create.file') }}</label>
              <input ref="createInput" type="file" accept=".html,.htm,.zip" class="block w-full text-sm text-gray-600 dark:text-gray-300" data-testid="site-file" @change="onCreateFile" />
            </div>
            <button class="btn btn-primary" :disabled="busy || !createFile" data-testid="site-publish" @click="create">
              {{ busy ? t('sites.publishing') : nextPaid ? t('sites.create.publishPaid', { price: data.quota.extra_price }) : t('sites.create.publish') }}
            </button>
          </div>
          <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('sites.rules') }}</p>
        </div>

        <div class="space-y-3">
          <div v-for="site in data.sites" :key="site.id" class="card p-4" data-testid="site-card">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-semibold text-gray-900 dark:text-white">{{ site.title || site.name }}</span>
                  <span :class="['badge', statusBadge(site.status)]">{{ t(`sites.status.${site.status}`) }}</span>
                  <span v-if="site.paid" class="badge badge-gray">{{ t('sites.paidUntil', { date: formatDateOnly(site.paid_until) }) }}</span>
                  <span v-else class="badge badge-gray">{{ t('sites.free') }}</span>
                  <span v-if="site.has_password" class="badge badge-purple">{{ t('sites.locked') }}</span>
                  <span v-if="site.pending_version" class="badge badge-warning">{{ t('sites.pendingReview', { version: site.pending_version }) }}</span>
                  <a v-if="site.preview_url" :href="site.preview_url" target="_blank" rel="noopener" class="text-xs text-primary-600 hover:underline">{{ t('sites.preview') }}</a>
                </div>
                <a :href="site.url" target="_blank" rel="noopener" class="mt-1 block break-all text-sm text-primary-600 hover:underline">{{ site.url }}</a>
                <p v-if="site.status_reason" class="mt-1 text-sm text-amber-700 dark:text-amber-300">{{ site.status_reason }}</p>
                <p v-if="site.version" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('sites.meta', { size: formatBytes(site.size_bytes, 1), files: site.file_count, version: site.version, time: formatDateTime(site.updated_at) }) }}
                  · {{ t('sites.views', { week: site.views_7d, total: site.views_total }) }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary btn-sm" @click="copy(site.url)">{{ t('sites.actions.copy') }}</button>
                <button v-if="site.status !== 'disabled'" class="btn btn-secondary btn-sm" :disabled="!data.quota.subscribed" @click="openEdit(site)">{{ t('sites.actions.update') }}</button>
                <button class="btn btn-secondary btn-sm" @click="openVersions(site)">{{ t('sites.actions.versions') }}</button>
                <button class="btn btn-secondary btn-sm" @click="openPassword(site)">{{ t('sites.actions.password') }}</button>
                <button class="btn btn-secondary btn-sm" @click="openStats(site)">{{ t('sites.actions.stats') }}</button>
                <button v-if="site.status === 'unpaid'" class="btn btn-primary btn-sm" :disabled="busy" @click="renew(site)">{{ t('sites.actions.renew', { price: data.quota.extra_price }) }}</button>
                <button class="btn btn-danger btn-sm" @click="confirmDelete = site">{{ t('sites.actions.delete') }}</button>
              </div>
            </div>
          </div>
          <div v-if="!data.sites.length" class="card p-8 text-center text-sm text-gray-500">{{ t('sites.empty') }}</div>
        </div>

        <div class="card space-y-2 p-4 text-sm">
          <div class="font-semibold text-gray-900 dark:text-white">{{ t('sites.api.title') }}</div>
          <p class="text-gray-500 dark:text-dark-400">{{ t('sites.api.hint') }}</p>
          <pre class="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-gray-100">{{ apiExample }}</pre>
        </div>

        <div v-if="data.charges.length" class="card overflow-x-auto">
          <div class="px-4 pt-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('sites.charges.title') }}</div>
          <table class="mt-2 min-w-full text-sm">
            <thead class="text-left text-xs text-gray-500 dark:text-dark-400">
              <tr>
                <th class="px-4 py-2 font-medium">{{ t('sites.charges.time') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('sites.charges.site') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('sites.charges.amount') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('sites.charges.until') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="charge in data.charges" :key="charge.id">
                <td class="px-4 py-2 text-gray-700 dark:text-gray-300">{{ formatDateTime(charge.created_at) }}</td>
                <td class="px-4 py-2 text-gray-700 dark:text-gray-300">{{ charge.site_name }}</td>
                <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">¥{{ charge.amount }}</td>
                <td class="px-4 py-2 text-gray-500">{{ formatDateOnly(charge.period_end) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>

    <BaseDialog :show="!!editing" :title="t('sites.edit.title')" @close="editing = null">
      <div v-if="editing" class="space-y-4">
        <div>
          <label class="input-label">{{ t('sites.create.name') }}</label>
          <input v-model="editTitle" class="input" maxlength="60" />
        </div>
        <div>
          <label class="input-label">{{ t('sites.edit.file') }}</label>
          <input type="file" accept=".html,.htm,.zip" class="block w-full text-sm text-gray-600 dark:text-gray-300" @change="onEditFile" />
          <p class="mt-1 text-xs text-gray-500">{{ t('sites.edit.fileHint') }}</p>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="editing = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="busy" @click="saveEdit">{{ busy ? t('sites.publishing') : t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!versionsOf" :title="t('sites.versions.title')" width="wide" @close="versionsOf = null">
      <table class="min-w-full text-sm">
        <thead class="text-left text-xs text-gray-500">
          <tr>
            <th class="py-2 pr-3 font-medium">{{ t('sites.versions.version') }}</th>
            <th class="py-2 pr-3 font-medium">{{ t('sites.versions.time') }}</th>
            <th class="py-2 pr-3 font-medium">{{ t('sites.versions.review') }}</th>
            <th class="py-2 font-medium"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-for="v in versions" :key="v.version" data-testid="site-version-row">
            <td class="py-2 pr-3 text-gray-900 dark:text-white">{{ t('sites.versions.label', { version: v.version }) }} <span v-if="v.current" class="badge badge-success">{{ t('sites.versions.current') }}</span></td>
            <td class="py-2 pr-3 text-gray-500">{{ formatDateTime(v.created_at) }} · {{ formatBytes(v.size_bytes, 1) }}</td>
            <td class="py-2 pr-3">
              <span :class="['badge', reviewBadge(v.review_status)]">{{ t(`sites.versions.statuses.${v.review_status}`) }}</span>
              <div v-if="v.review_reason" class="mt-1 text-xs text-gray-500">{{ v.review_reason }}</div>
            </td>
            <td class="py-2 text-right">
              <button v-if="!v.current && v.available && v.review_status === 'approved'" class="btn btn-secondary btn-sm" :disabled="busy" @click="rollback(v.version)">{{ t('sites.versions.rollback') }}</button>
              <span v-else-if="!v.available" class="text-xs text-gray-400">{{ t('sites.versions.cleaned') }}</span>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="mt-3 text-xs text-gray-500">{{ t('sites.versions.hint') }}</p>
    </BaseDialog>

    <BaseDialog :show="!!passwordOf" :title="t('sites.password.title')" width="narrow" @close="passwordOf = null">
      <div class="space-y-2">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ passwordOf?.has_password ? t('sites.password.hintOn') : t('sites.password.hintOff') }}</p>
        <input v-model="password" type="text" class="input" maxlength="64" autocomplete="off" :placeholder="t('sites.password.placeholder')" />
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button v-if="passwordOf?.has_password" class="btn btn-secondary" :disabled="busy" @click="savePassword('')">{{ t('sites.password.remove') }}</button>
          <button class="btn btn-primary" :disabled="busy || password.length < 4" @click="savePassword(password)">{{ t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="!!statsOf" :title="t('sites.stats.title', { name: statsOf?.title || statsOf?.name || '' })" width="wide" @close="statsOf = null">
      <div v-if="stats" class="space-y-4">
        <div class="grid grid-cols-3 gap-3 text-center">
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500">{{ t('sites.stats.views') }}</div>
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ stats.views }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500">{{ t('sites.stats.visitors') }}</div>
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ stats.visitors }}</div>
          </div>
          <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
            <div class="text-xs text-gray-500">{{ t('sites.stats.traffic') }}</div>
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatBytes(stats.bytes, 1) }}</div>
          </div>
        </div>
        <div class="flex h-32 items-end gap-1" data-testid="site-stats-chart">
          <div v-for="(height, i) in bars" :key="i" class="flex-1 rounded-t bg-primary-500/70" :style="{ height: `${Math.max(height, 2)}%` }" :title="`${stats.days[i].day.slice(0, 10)}：${stats.days[i].views} / ${stats.days[i].visitors}`" />
        </div>
        <p class="text-xs text-gray-500">{{ t('sites.stats.hint') }}</p>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="!!confirmDelete"
      :title="t('sites.actions.delete')"
      :message="t('sites.deleteConfirm', { url: confirmDelete?.url || '' })"
      danger
      @confirm="remove"
      @cancel="confirmDelete = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateOnly, formatDateTime } from '@/utils/format'
import { createSite, deleteSite, isSiteUploadFile, mySites, nextSiteIsPaid, renewSite, rollbackSite, setSitePassword, siteStats, siteVersions, statBars, updateSite, type MySites, type Site, type SiteStats, type SiteStatus, type SiteVersion } from '@/api/sites'
import { buildApiUrl } from '@/api/client'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const data = ref<MySites | null>(null)
const busy = ref(false)
const createTitle = ref('')
const createFile = ref<File | null>(null)
const createInput = ref<HTMLInputElement | null>(null)
const editing = ref<Site | null>(null)
const editTitle = ref('')
const editFile = ref<File | null>(null)
const confirmDelete = ref<Site | null>(null)

const nextPaid = computed(() => (data.value ? nextSiteIsPaid(data.value.quota) : false))
const versionsOf = ref<Site | null>(null)
const versions = ref<SiteVersion[]>([])
const passwordOf = ref<Site | null>(null)
const password = ref('')
const statsOf = ref<Site | null>(null)
const stats = ref<SiteStats | null>(null)
const bars = computed(() => (stats.value ? statBars(stats.value.days) : []))
const apiExample = computed(() => {
  const url = new URL(buildApiUrl('/hosting/sites'), window.location.origin).toString()
  const key = t('sites.api.keyPlaceholder')
  return `curl -X POST ${url} \\\n  -H "Authorization: Bearer ${key}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"title":"Demo","html":"<h1>Hello</h1>"}'\n\n# ${t('sites.api.zip')}\ncurl -X POST ${url} -H "Authorization: Bearer ${key}" -F title=Demo -F file=@site.zip`
})

function reviewBadge(status: SiteVersion['review_status']) {
  return status === 'approved' ? 'badge-success' : status === 'pending' ? 'badge-warning' : status === 'rejected' ? 'badge-danger' : 'badge-gray'
}

async function openVersions(site: Site) {
  versionsOf.value = site
  versions.value = []
  try {
    versions.value = await siteVersions(site.id)
  } catch (error) {
    showError(error)
  }
}

async function rollback(version: number) {
  if (!versionsOf.value) return
  busy.value = true
  try {
    await rollbackSite(versionsOf.value.id, version)
    appStore.showSuccess(t('sites.versions.rolledBack', { version }))
    versions.value = await siteVersions(versionsOf.value.id)
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

function openPassword(site: Site) {
  passwordOf.value = site
  password.value = ''
}

async function savePassword(value: string) {
  if (!passwordOf.value) return
  busy.value = true
  try {
    await setSitePassword(passwordOf.value.id, value)
    appStore.showSuccess(value ? t('sites.password.saved') : t('sites.password.removed'))
    passwordOf.value = null
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function openStats(site: Site) {
  statsOf.value = site
  stats.value = null
  try {
    stats.value = await siteStats(site.id, 30)
  } catch (error) {
    showError(error)
  }
}

function statusBadge(status: SiteStatus) {
  return status === 'active' ? 'badge-success' : status === 'disabled' ? 'badge-danger' : 'badge-warning'
}

function showError(error: unknown) {
  appStore.showError(extractApiErrorMessage(error, t('common.error')))
}

function pickFile(event: Event): File | null {
  const file = (event.target as HTMLInputElement).files?.[0] || null
  if (file && !isSiteUploadFile(file)) {
    appStore.showError(t('sites.wrongFile'))
    ;(event.target as HTMLInputElement).value = ''
    return null
  }
  if (file && data.value && file.size > data.value.quota.max_mb * 1024 * 1024) {
    appStore.showError(t('sites.tooLarge', { mb: data.value.quota.max_mb }))
    ;(event.target as HTMLInputElement).value = ''
    return null
  }
  return file
}

function onCreateFile(event: Event) {
  createFile.value = pickFile(event)
}
function onEditFile(event: Event) {
  editFile.value = pickFile(event)
}

async function load() {
  try {
    data.value = await mySites()
  } catch (error) {
    showError(error)
  }
}

async function create() {
  if (!createFile.value) return
  busy.value = true
  try {
    const site = await createSite(createTitle.value, createFile.value)
    appStore.showSuccess(t('sites.published', { url: site.url }))
    createTitle.value = ''
    createFile.value = null
    if (createInput.value) createInput.value.value = ''
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

function openEdit(site: Site) {
  editing.value = site
  editTitle.value = site.title
  editFile.value = null
}

async function saveEdit() {
  if (!editing.value) return
  busy.value = true
  try {
    await updateSite(editing.value.id, editTitle.value, editFile.value)
    appStore.showSuccess(t('sites.updated'))
    editing.value = null
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function renew(site: Site) {
  busy.value = true
  try {
    await renewSite(site.id)
    appStore.showSuccess(t('sites.renewed'))
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function remove() {
  const site = confirmDelete.value
  confirmDelete.value = null
  if (!site) return
  try {
    await deleteSite(site.id)
    appStore.showSuccess(t('sites.deleted'))
    await load()
  } catch (error) {
    showError(error)
  }
}

function copy(text: string) {
  void copyToClipboard(text, t('sites.copied'))
}

onMounted(load)
</script>
