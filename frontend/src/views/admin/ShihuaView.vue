<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap gap-1">
            <button
              v-for="tab in tabs"
              :key="tab"
              class="rounded-lg px-3 py-1.5 text-sm"
              :class="active === tab ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
              :data-testid="`shihua-tab-${tab}`"
              @click="active = tab"
            >
              {{ t(`shihua.tabs.${tab}`) }}
            </button>
          </div>
          <div v-if="settings" class="text-xs text-gray-500 dark:text-dark-400" data-testid="shihua-queue">
            {{
              t('shihua.queue', {
                queued: settings.queue.queued,
                running: settings.queue.running,
                today: settings.queue.started_today,
                limit: settings.daily_limit,
                upscale: settings.queue.upscale_queue
              })
            }}
          </div>
        </div>
        <p v-if="settings?.paused && settings.queue.queued > 0" class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
          {{ t('shihua.paused', { reason: settings.paused }) }}
        </p>
      </div>

      <ShihuaWorksPanel v-if="active === 'works'" :works="works" :loading="loadingWorks" @reload="reloadWorks" @refresh-queue="refreshSettings" />
      <ShihuaDrawPanel v-else-if="active === 'draw'" :works="works" @refresh-queue="refreshSettings" />
      <ShihuaPickPanel v-else-if="active === 'pick'" :works="works" @reload="reloadWorks" />
      <ShihuaLayoutPanel v-else-if="active === 'layout'" :works="works" @reload="reloadWorks" />
      <ShihuaSettingsPanel v-else :settings="settings" @saved="(v) => (settings = v)" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/shihua'
import ShihuaWorksPanel from './shihua/ShihuaWorksPanel.vue'
import ShihuaDrawPanel from './shihua/ShihuaDrawPanel.vue'
import ShihuaPickPanel from './shihua/ShihuaPickPanel.vue'
import ShihuaLayoutPanel from './shihua/ShihuaLayoutPanel.vue'
import ShihuaSettingsPanel from './shihua/ShihuaSettingsPanel.vue'

const { t } = useI18n()
const appStore = useAppStore()
const tabs = ['works', 'draw', 'pick', 'layout', 'settings'] as const
type Tab = (typeof tabs)[number]
const active = ref<Tab>((new URLSearchParams(location.search).get('tab') as Tab) || 'works')
const works = ref<api.ShihuaWork[]>([])
const loadingWorks = ref(false)
const settings = ref<api.ShihuaSettingsView | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

watch(active, (tab) => {
  const url = new URL(location.href)
  url.searchParams.set('tab', tab)
  history.replaceState(history.state, '', url)
  void reloadWorks()
})

async function reloadWorks() {
  loadingWorks.value = true
  try {
    works.value = await api.listWorks()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loadingWorks.value = false
  }
}

async function refreshSettings() {
  try {
    const before = settings.value?.queue
    settings.value = await api.getSettings()
    const after = settings.value.queue
    // A drawing finished: the counts in the lists changed.
    if (before && (before.running !== after.running || before.queued !== after.queued)) void reloadWorks()
  } catch {
    /* the queue line just stays as it was */
  }
}

onMounted(() => {
  void reloadWorks()
  void refreshSettings()
  timer = setInterval(() => {
    if (!document.hidden) void refreshSettings()
  }, 10000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>
