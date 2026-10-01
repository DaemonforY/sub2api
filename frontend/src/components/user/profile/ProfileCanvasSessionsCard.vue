<template>
  <div class="card" data-testid="canvas-sessions-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">{{ t('profile.canvasSessions.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('profile.canvasSessions.description') }}</p>
    </div>
    <div class="px-6 py-4">
      <p v-if="loading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
      <p v-else-if="!sessions.length" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.canvasSessions.empty') }}
        <a :href="canvasHome" target="_blank" rel="noopener" class="ml-1 text-primary-600 hover:underline">{{ t('profile.canvasSessions.open') }}</a>
      </p>
      <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <li v-for="s in sessions" :key="s.id" class="flex items-center justify-between gap-3 py-3 text-sm" data-testid="canvas-session">
          <div class="min-w-0">
            <div class="font-medium text-gray-900 dark:text-white">{{ describeUserAgent(s.user_agent) || t('profile.canvasSessions.unknownDevice') }}</div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('profile.canvasSessions.meta', { ip: s.ip || '-', used: formatDateTime(s.last_used_at), created: formatDateOnly(s.created_at) }) }}</div>
          </div>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" :disabled="busyId === s.id" @click="revoke(s.id)">{{ t('profile.canvasSessions.signOut') }}</button>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { describeUserAgent, listCanvasSessions, revokeCanvasSession, type CanvasSession } from '@/api/canvasSessions'
import { canvasUrl } from '@/constants/crossSites'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateOnly, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const sessions = ref<CanvasSession[]>([])
const loading = ref(true)
const busyId = ref(0)
const canvasHome = canvasUrl({ medium: 'profile' })

async function load() {
  try {
    sessions.value = await listCanvasSessions()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function revoke(id: number) {
  busyId.value = id
  try {
    await revokeCanvasSession(id)
    sessions.value = sessions.value.filter((s) => s.id !== id)
    appStore.showSuccess(t('profile.canvasSessions.signedOut'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    busyId.value = 0
  }
}

onMounted(load)
</script>
