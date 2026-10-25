<template>
  <div class="space-y-4">
    <div class="card overflow-x-auto">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
        <div>
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('shihua.draw.jobs') }}</h3>
          <p class="mt-1 text-xs text-gray-500">{{ t('shihua.draw.costHint') }}</p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary btn-sm" @click="reload"><Icon name="refresh" size="sm" /></button>
          <button class="btn btn-secondary btn-sm" @click="cancel(0)">{{ t('shihua.draw.cancelAll') }}</button>
        </div>
      </div>
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
          <tr>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.name') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.works') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.progress') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.cost') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.created') }}</th>
            <th class="px-4 py-3 font-medium">{{ t('shihua.draw.columns.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-if="!jobs.length">
            <td colspan="6" class="px-4 py-10 text-center text-gray-400">{{ t('shihua.draw.empty') }}</td>
          </tr>
          <tr v-for="j in jobs" :key="j.id">
            <td class="px-4 py-3 text-gray-900 dark:text-white">{{ j.name }}</td>
            <td class="px-4 py-3">{{ j.works }} × {{ j.candidates }}</td>
            <td class="min-w-[14rem] px-4 py-3">
              <div class="flex h-2 overflow-hidden rounded bg-gray-100 dark:bg-dark-700">
                <div class="bg-emerald-500" :style="{ width: pct(j.counts.succeeded, j) }" />
                <div class="bg-sky-400" :style="{ width: pct(j.counts.running, j) }" />
                <div class="bg-red-400" :style="{ width: pct(j.counts.failed, j) }" />
              </div>
              <div class="mt-1 text-xs text-gray-500">
                {{ t('shihua.works.imagesCount', { ok: j.counts.succeeded, queued: j.counts.queued + j.counts.running, failed: j.counts.failed }) }}
              </div>
            </td>
            <td class="px-4 py-3">${{ j.cost_usd.toFixed(3) }}</td>
            <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ formatDateTime(j.created_at) }}</td>
            <td class="whitespace-nowrap px-4 py-3 text-xs">
              <button v-if="j.counts.queued" class="mr-3 text-gray-600 hover:underline" @click="cancel(j.id)">{{ t('shihua.draw.cancel') }}</button>
              <button v-if="j.counts.failed || j.canceled" class="text-primary-600 hover:underline" @click="retry(j.id)">{{ t('shihua.draw.retry') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="failures.length" class="card p-4">
      <h3 class="mb-3 font-semibold text-gray-900 dark:text-white">{{ t('shihua.draw.failures') }}</h3>
      <div v-for="img in failures" :key="img.id" class="border-t border-gray-100 py-2 text-sm first:border-0 dark:border-dark-700">
        <span class="font-medium">{{ titleOf(img.work_id) }}</span>
        <span class="ml-2 text-xs text-gray-400">#{{ img.id }} · {{ img.attempts }}×</span>
        <p class="text-xs text-red-500">{{ img.error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import * as api from '@/api/admin/shihua'

const props = defineProps<{ works: api.ShihuaWork[] }>()
const emit = defineEmits<{ (e: 'refresh-queue'): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const jobs = ref<api.ShihuaJob[]>([])
const failed = ref<api.ShihuaImage[]>([])
let timer: ReturnType<typeof setInterval> | undefined

const failures = computed(() => [...failed.value].sort((a, b) => b.id - a.id).slice(0, 30))
const titles = computed(() => new Map(props.works.map((w) => [w.id, w.title])))
function titleOf(id: number) {
  return titles.value.get(id) ?? `#${id}`
}
function pct(n: number, j: api.ShihuaJob) {
  return `${j.counts.total ? (n / j.counts.total) * 100 : 0}%`
}

async function reload() {
  try {
    const [j, f] = await Promise.all([api.listJobs(), api.listImages({ status: 'failed' })])
    jobs.value = j
    failed.value = f
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function cancel(id: number) {
  try {
    appStore.showSuccess(t('shihua.draw.canceled', { n: await api.cancelJob(id) }))
    await reload()
    emit('refresh-queue')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function retry(id: number) {
  try {
    appStore.showSuccess(t('shihua.draw.retried', { n: await api.retryJob(id) }))
    await reload()
    emit('refresh-queue')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

onMounted(() => {
  void reload()
  timer = setInterval(() => {
    if (!document.hidden) void reload()
  }, 8000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>
