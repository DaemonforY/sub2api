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
                </div>
                <a :href="site.url" target="_blank" rel="noopener" class="mt-1 block break-all text-sm text-primary-600 hover:underline">{{ site.url }}</a>
                <p v-if="site.status_reason" class="mt-1 text-sm text-amber-700 dark:text-amber-300">{{ site.status_reason }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('sites.meta', { size: formatBytes(site.size_bytes, 1), files: site.file_count, version: site.version, time: formatDateTime(site.updated_at) }) }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary btn-sm" @click="copy(site.url)">{{ t('sites.actions.copy') }}</button>
                <button v-if="site.status !== 'disabled'" class="btn btn-secondary btn-sm" :disabled="!data.quota.subscribed" @click="openEdit(site)">{{ t('sites.actions.update') }}</button>
                <button v-if="site.status === 'unpaid'" class="btn btn-primary btn-sm" :disabled="busy" @click="renew(site)">{{ t('sites.actions.renew', { price: data.quota.extra_price }) }}</button>
                <button class="btn btn-danger btn-sm" @click="confirmDelete = site">{{ t('sites.actions.delete') }}</button>
              </div>
            </div>
          </div>
          <div v-if="!data.sites.length" class="card p-8 text-center text-sm text-gray-500">{{ t('sites.empty') }}</div>
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
import { createSite, deleteSite, isSiteUploadFile, mySites, nextSiteIsPaid, renewSite, updateSite, type MySites, type Site, type SiteStatus } from '@/api/sites'

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
