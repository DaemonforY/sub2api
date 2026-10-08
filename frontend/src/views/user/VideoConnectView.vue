<template>
  <div class="flex min-h-screen items-start justify-center bg-gray-50 px-4 py-8 dark:bg-dark-950" data-testid="video-connect">
    <div class="card w-full max-w-md p-6">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" alt="" class="h-9 w-9 rounded-lg object-contain" />
        <div>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('videoConnect.title') }}</h1>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('videoConnect.subtitle', { site: siteName }) }}</p>
        </div>
      </div>

      <div v-if="!canHandOff" class="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200" data-testid="video-connect-invalid">
        {{ t('videoConnect.openFromVideo') }}
        <a :href="videoOrigin" class="ml-1 font-medium underline" target="_blank" rel="noopener noreferrer">{{ t('videoConnect.goVideo') }}</a>
      </div>

      <div v-else-if="done" class="mt-6 py-6 text-center" data-testid="video-connect-done">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-2xl text-emerald-600 dark:bg-emerald-900/30">✓</div>
        <p class="mt-3 font-medium text-gray-900 dark:text-white">{{ t('videoConnect.done') }}</p>
      </div>

      <template v-else>
        <p class="mt-4 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('videoConnect.explain') }}</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('videoConnect.balance', { balance: balanceText }) }}</p>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>

        <template v-else>
          <template v-if="activeKeys.length">
            <p class="mt-4 text-sm font-medium text-gray-800 dark:text-dark-200">{{ t('videoConnect.pick') }}</p>
            <div class="mt-2 max-h-72 space-y-2 overflow-y-auto" role="radiogroup">
              <label
                v-for="k in activeKeys"
                :key="k.id"
                :class="['flex cursor-pointer items-center gap-3 rounded-xl border px-3 py-2.5 text-sm', selectedId === k.id ? 'border-primary-500 ring-1 ring-primary-500' : 'border-gray-200 hover:border-primary-400 dark:border-dark-700']"
              >
                <input v-model="selectedId" type="radio" :value="k.id" class="h-4 w-4" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-medium text-gray-900 dark:text-white">{{ k.name }}</span>
                  <span class="block truncate text-xs text-gray-500 dark:text-dark-400">{{ k.group?.name || '—' }} · {{ maskKey(k.key) }}</span>
                </span>
              </label>
            </div>
          </template>
          <div v-else class="mt-4 rounded-xl border border-dashed border-gray-300 p-4 text-sm dark:border-dark-700">
            <p class="text-gray-600 dark:text-dark-300">{{ t('videoConnect.none') }}</p>
            <div class="mt-3 flex items-center gap-2">
              <select v-model="createGroupId" class="input flex-1 text-sm" :aria-label="t('videoConnect.createIn')">
                <option v-for="g in activeGroups" :key="g.id" :value="g.id">{{ g.name }}</option>
              </select>
              <button class="btn btn-secondary" :disabled="!createGroupId || creating" @click="createKey">{{ t('videoConnect.create') }}</button>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <button class="btn btn-secondary" @click="cancel">{{ t('videoConnect.cancel') }}</button>
            <button class="btn btn-primary" :disabled="!selectedKey || authorizing" data-testid="video-connect-authorize" @click="authorize">{{ t('videoConnect.authorize') }}</button>
          </div>
          <p class="mt-3 text-center text-[11px] leading-5 text-gray-400">{{ t('videoConnect.privacy', { origin: videoOrigin }) }}</p>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { keysAPI } from '@/api/keys'
import { getAvailable } from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { VIDEO_SITE_URL } from '@/constants/crossSites'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ApiKey, Group } from '@/types'

/**
 * Popup opened by HiveGPT 视频 (video.<domain>) "登录". After the user picks a key and clicks 授权,
 * the key goes to the video window with postMessage — targeted at the video origin only, echoing
 * the video-generated `state` so it can match the answer to its request.
 */
const VIDEO_CONNECT_MESSAGE_TYPE = 'hivegpt:video-key'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'HiveGPT')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || '')
const videoOrigin = new URL(VIDEO_SITE_URL).origin

const state = typeof route.query.state === 'string' ? route.query.state : ''
const hasOpener = typeof window !== 'undefined' && Boolean(window.opener) && !window.opener.closed
const canHandOff = /^[A-Za-z0-9_-]{16,128}$/.test(state) && hasOpener

const loading = ref(true)
const creating = ref(false)
const authorizing = ref(false)
const done = ref(false)
const keys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const selectedId = ref<number | null>(null)
const createGroupId = ref<number | null>(null)

const activeKeys = computed(() => keys.value.filter((k) => k.status === 'active'))
const activeGroups = computed(() => groups.value.filter((g) => g.status === 'active'))
const selectedKey = computed(() => activeKeys.value.find((k) => k.id === selectedId.value) || null)
const balanceText = computed(() => (authStore.user?.balance ?? 0).toFixed(2))

function maskKey(key: string): string {
  if (!key) return ''
  return key.length <= 12 ? `${key.slice(0, 3)}…` : `${key.slice(0, 6)}…${key.slice(-4)}`
}

async function load() {
  loading.value = true
  try {
    const [keyPage, available] = await Promise.all([keysAPI.list(1, 100, { sort_by: 'created_at', sort_order: 'desc' }), getAvailable()])
    keys.value = keyPage.items
    groups.value = available
    selectedId.value = activeKeys.value[0]?.id ?? null
    const preferred = activeGroups.value.find((g) => g.subscription_type === 'subscription') || activeGroups.value[0]
    createGroupId.value = preferred?.id ?? null
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function createKey() {
  if (!createGroupId.value || creating.value) return
  creating.value = true
  try {
    const created = await keysAPI.create(t('videoConnect.keyName'), createGroupId.value)
    await load()
    selectedId.value = created.id
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    creating.value = false
  }
}

function authorize() {
  const key = selectedKey.value
  if (!key || authorizing.value || !canHandOff || !window.opener || window.opener.closed) return
  authorizing.value = true
  window.opener.postMessage({ type: VIDEO_CONNECT_MESSAGE_TYPE, state, apiKey: key.key, keyName: key.name }, videoOrigin)
  done.value = true
  setTimeout(() => window.close(), 1200)
}

function cancel() {
  window.close()
}

onMounted(() => {
  if (canHandOff) void load()
  else loading.value = false
})
</script>
