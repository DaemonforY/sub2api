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
              :data-testid="`materials-tab-${tab}`"
              @click="active = tab"
            >
              {{ t(`materials.tabs.${tab}`) }}
            </button>
          </div>
          <div v-if="queue" class="text-xs text-gray-500 dark:text-dark-400" data-testid="materials-queue">
            {{ t('materials.queue', { queued: queue.queued, running: queue.running, today: queue.started_today, limit: queue.daily_limit }) }}
          </div>
        </div>
        <p
          v-if="queue?.paused && queue.queued > 0"
          class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
          data-testid="materials-paused"
        >
          {{ t('materials.paused', { reason: queue.paused }) }}
        </p>
      </div>

      <MaterialTextsPanel v-if="active === 'texts'" :tick="tick" @queued="refreshQueue" />
      <MaterialImageryPanel v-else-if="active === 'imagery'" />
      <MaterialArtworksPanel v-else-if="active === 'artworks'" :tick="tick" @queued="refreshQueue" />
      <MaterialLayoutPanel v-else-if="active === 'layout'" />
      <MaterialRulesPanel v-else />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import * as api from '@/api/admin/materials'
import MaterialTextsPanel from './materials/MaterialTextsPanel.vue'
import MaterialImageryPanel from './materials/MaterialImageryPanel.vue'
import MaterialArtworksPanel from './materials/MaterialArtworksPanel.vue'
import MaterialLayoutPanel from './materials/MaterialLayoutPanel.vue'
import MaterialRulesPanel from './materials/MaterialRulesPanel.vue'

const { t } = useI18n()
const tabs = ['texts', 'imagery', 'artworks', 'layout', 'rules'] as const
type Tab = (typeof tabs)[number]
const initial = new URLSearchParams(location.search).get('tab') as Tab
const active = ref<Tab>(tabs.includes(initial) ? initial : 'texts')
const queue = ref<api.MaterialQueue | null>(null)
// Bumped when a drawing finishes, so panels showing artworks reload them.
const tick = ref(0)
let timer: ReturnType<typeof setInterval> | undefined

watch(active, (tab) => {
  const url = new URL(location.href)
  url.searchParams.set('tab', tab)
  history.replaceState(history.state, '', url)
})

async function refreshQueue() {
  try {
    const before = queue.value
    queue.value = await api.getQueue()
    if (before && (before.running !== queue.value.running || before.queued !== queue.value.queued)) tick.value++
  } catch {
    /* the queue line just stays as it was */
  }
}

onMounted(() => {
  void refreshQueue()
  timer = setInterval(() => {
    if (!document.hidden) void refreshQueue()
  }, 10000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>
