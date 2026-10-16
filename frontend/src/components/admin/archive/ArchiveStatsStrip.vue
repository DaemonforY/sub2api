<template>
  <div class="card p-4" data-testid="ca-stats">
    <div class="mb-2 flex items-center justify-between">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.conversationArchive.stats.title') }}</h3>
    </div>
    <div v-if="error" class="text-sm text-red-600">{{ t('admin.conversationArchive.loadFailed') }}：{{ error }}</div>
    <div v-else-if="!days.length" class="text-sm text-gray-500 dark:text-dark-400">{{ loading ? t('common.loading') : t('admin.conversationArchive.stats.empty') }}</div>
    <div v-else class="flex flex-wrap items-end gap-6">
      <dl class="grid grid-cols-2 gap-x-6 gap-y-1 text-sm sm:grid-cols-4">
        <div>
          <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.stats.turns') }}</dt>
          <dd class="text-lg font-semibold text-gray-900 dark:text-white" data-testid="ca-stats-turns">{{ totals.turns }}</dd>
        </div>
        <div>
          <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.stats.users') }}</dt>
          <dd class="text-lg font-semibold text-gray-900 dark:text-white">{{ totals.avgUsers }}</dd>
        </div>
        <div>
          <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.stats.bytes') }}</dt>
          <dd class="text-lg font-semibold text-gray-900 dark:text-white">{{ formatBytes(totals.bytes, 1) }}</dd>
        </div>
        <div>
          <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.conversationArchive.stats.images') }}</dt>
          <dd class="text-lg font-semibold text-gray-900 dark:text-white">{{ totals.images }}</dd>
        </div>
      </dl>
      <!-- turns per day, oldest → newest -->
      <div class="flex h-12 min-w-[12rem] flex-1 items-end gap-px" data-testid="ca-stats-spark">
        <div
          v-for="d in sorted"
          :key="d.day"
          class="flex-1 rounded-t bg-primary-400/80 dark:bg-primary-500/70"
          :style="{ height: `${Math.max(4, Math.round((d.turns / maxTurns) * 100))}%` }"
          :title="t('admin.conversationArchive.stats.day', { day: d.day, turns: d.turns, users: d.users, bytes: formatBytes(d.bytes, 1), images: d.images })"
        ></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getStats, type ArchiveStatsDay } from '@/api/admin/conversationArchive'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes } from '@/utils/format'

// /stats is read from the index and is not audited, so it loads with the page.
const { t } = useI18n()
const days = ref<ArchiveStatsDay[]>([])
const loading = ref(false)
const error = ref('')

const sorted = computed(() => [...days.value].sort((a, b) => a.day.localeCompare(b.day)))
const maxTurns = computed(() => Math.max(1, ...days.value.map((d) => d.turns || 0)))
const totals = computed(() => {
  const n = days.value.length || 1
  const sum = (k: 'turns' | 'users' | 'bytes' | 'images') => days.value.reduce((acc, d) => acc + (d[k] || 0), 0)
  return { turns: sum('turns'), avgUsers: Math.round(sum('users') / n), bytes: sum('bytes'), images: sum('images') }
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const r = await getStats()
    days.value = r?.days || []
  } catch (e) {
    error.value = extractApiErrorMessage(e, t('admin.conversationArchive.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
defineExpose({ reload: load })
</script>
