<template>
  <div class="flex min-h-screen items-start justify-center bg-gray-50 px-4 py-8 dark:bg-dark-950" data-testid="canvas-connect">
    <div class="card w-full max-w-md p-6">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" alt="" class="h-9 w-9 rounded-lg object-contain" />
        <div>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('canvasConnect.title') }}</h1>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('canvasConnect.subtitle', { site: siteName }) }}</p>
        </div>
      </div>

      <!-- Opened directly (not from the canvas popup), or the request is malformed. -->
      <div v-if="!canHandOff" class="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200" data-testid="canvas-connect-invalid">
        {{ t('canvasConnect.openFromCanvas') }}
        <a :href="canvasHome" class="ml-1 font-medium underline" target="_blank" rel="noopener noreferrer">{{ t('canvasConnect.goCanvas') }}</a>
      </div>

      <div v-else-if="done" class="mt-6 py-6 text-center" data-testid="canvas-connect-done">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-2xl text-emerald-600 dark:bg-emerald-900/30">✓</div>
        <p class="mt-3 font-medium text-gray-900 dark:text-white">{{ t('canvasConnect.done') }}</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('canvasConnect.doneHint') }}</p>
      </div>

      <template v-else>
        <p class="mt-4 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('canvasConnect.explain', { site: siteName }) }}</p>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>

        <template v-else>
          <div v-if="keys.length" class="mt-4 space-y-2" role="radiogroup">
            <label
              v-for="k in keys"
              :key="k.id"
              :class="[
                'flex items-center gap-3 rounded-xl border px-3 py-2.5 text-sm',
                usable(k) ? 'cursor-pointer border-gray-200 hover:border-primary-400 dark:border-dark-700' : 'cursor-not-allowed border-gray-100 opacity-55 dark:border-dark-800',
                selectedId === k.id ? 'border-primary-500 ring-1 ring-primary-500' : ''
              ]"
            >
              <input v-model="selectedId" type="radio" :value="k.id" :disabled="!usable(k)" class="accent-primary-600" />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate font-medium text-gray-900 dark:text-white">{{ k.name }}</span>
                  <span v-if="imageCapable(k)" class="shrink-0 rounded bg-emerald-50 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">{{ t('canvasConnect.canDraw') }}</span>
                </div>
                <div class="mt-0.5 truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ k.group?.name || t('canvasConnect.noGroup') }} · <span class="font-mono">{{ maskKey(k.key) }}</span>
                  <template v-if="!usable(k)"> · {{ imageCapable(k) ? t('canvasConnect.inactive') : t('canvasConnect.cannotDraw') }}</template>
                </div>
              </div>
            </label>
          </div>

          <!-- Create a dedicated key when none can draw (or on demand). -->
          <div v-if="drawableGroups.length && (!hasUsableKey || showCreate)" class="mt-4 rounded-xl border border-dashed border-gray-300 p-3 dark:border-dark-700" data-testid="canvas-connect-create">
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ hasUsableKey ? t('canvasConnect.createAnother') : t('canvasConnect.createTitle') }}</p>
            <div class="mt-2 flex gap-2">
              <select v-model="createGroupId" class="input flex-1 text-sm">
                <option v-for="g in drawableGroups" :key="g.id" :value="g.id">{{ g.name }}{{ g.subscription_type === 'subscription' ? t('canvasConnect.subscriptionSuffix') : t('canvasConnect.balanceSuffix') }}</option>
              </select>
              <button type="button" class="btn btn-secondary shrink-0" :disabled="creating || !createGroupId" @click="createKey">{{ t('canvasConnect.create') }}</button>
            </div>
            <p v-if="selectedCreateGroup?.subscription_type !== 'subscription'" class="mt-2 text-xs text-gray-500 dark:text-dark-400">
              {{ t('canvasConnect.balanceHint', { balance: balanceText }) }}
              <a href="/purchase" target="_blank" rel="noopener" class="ml-1 text-primary-600 underline">{{ t('canvasConnect.topUp') }}</a>
            </p>
          </div>
          <p v-else-if="!drawableGroups.length && !hasUsableKey" class="mt-4 text-sm text-amber-700 dark:text-amber-300">
            {{ t('canvasConnect.noDrawableGroup') }}
            <a href="/purchase?tab=subscription" target="_blank" rel="noopener" class="ml-1 underline">{{ t('canvasConnect.buyPlan') }}</a>
          </p>
          <button v-if="hasUsableKey && drawableGroups.length && !showCreate" type="button" class="mt-3 text-xs text-primary-600 hover:underline" @click="showCreate = true">
            {{ t('canvasConnect.createAnother') }}
          </button>

          <div class="mt-6 flex gap-2">
            <button type="button" class="btn btn-secondary flex-1" @click="cancel">{{ t('common.cancel') }}</button>
            <button type="button" class="btn btn-primary flex-1" :disabled="authorizing" data-testid="canvas-connect-authorize" @click="authorize">
              {{ selectedKey ? t('canvasConnect.authorize') : t('canvasConnect.signInOnly') }}
            </button>
          </div>
          <p v-if="!selectedKey" class="mt-3 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('canvasConnect.signInOnlyHint') }}</p>
          <p class="mt-3 text-center text-[11px] leading-5 text-gray-400">{{ t('canvasConnect.privacy', { canvas: canvasOrigin }) }}</p>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { createCanvasSession } from '@/api/canvasSessions'
import { getAvailable } from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { CANVAS_SITE_URL, canvasUrl } from '@/constants/crossSites'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ApiKey, Group } from '@/types'

/**
 * Popup opened by the canvas (canvas.<domain>) "登录 / 连接" button. Authorizing signs the canvas in
 * (the API sets the canvas session cookie on this host) and, when the user picked a key, hands the
 * key to the canvas window with postMessage — targeted at the canvas origin only, echoing the
 * canvas-generated `state` so the canvas can match the answer to its request. Nothing happens until
 * the user clicks 授权; without an image-capable key the canvas is signed in only.
 */
const CANVAS_CONNECT_MESSAGE_TYPE = 'hivegpt:canvas-key'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'HiveGPT')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || '')
const canvasOrigin = new URL(CANVAS_SITE_URL).origin
const canvasHome = canvasUrl({ medium: 'canvas-connect' })

const state = typeof route.query.state === 'string' ? route.query.state : ''
const hasOpener = typeof window !== 'undefined' && Boolean(window.opener) && !window.opener.closed
const canHandOff = /^[A-Za-z0-9_-]{16,128}$/.test(state) && hasOpener

const loading = ref(true)
const creating = ref(false)
const authorizing = ref(false)
const done = ref(false)
const showCreate = ref(false)
const keys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const selectedId = ref<number | null>(null)
const createGroupId = ref<number | null>(null)

const imageCapable = (k: ApiKey) => Boolean(k.group?.allow_image_generation)
const usable = (k: ApiKey) => imageCapable(k) && k.status === 'active'
const hasUsableKey = computed(() => keys.value.some(usable))
const selectedKey = computed(() => keys.value.find((k) => k.id === selectedId.value && usable(k)) || null)
const drawableGroups = computed(() => groups.value.filter((g) => g.allow_image_generation && g.status === 'active'))
const selectedCreateGroup = computed(() => drawableGroups.value.find((g) => g.id === createGroupId.value))
const balanceText = computed(() => (authStore.user?.balance ?? 0).toFixed(2))

function maskKey(key: string): string {
  if (!key) return ''
  return key.length <= 12 ? `${key.slice(0, 3)}…` : `${key.slice(0, 6)}…${key.slice(-4)}`
}

async function load() {
  loading.value = true
  try {
    const [keyPage, available] = await Promise.all([keysAPI.list(1, 100, { sort_by: 'created_at', sort_order: 'desc' }), getAvailable()])
    // Image-capable keys first.
    keys.value = [...keyPage.items].sort((a, b) => Number(usable(b)) - Number(usable(a)))
    groups.value = available
    selectedId.value = keys.value.find(usable)?.id ?? null
    const preferred = drawableGroups.value.find((g) => g.subscription_type === 'subscription') || drawableGroups.value[0]
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
    const created = await keysAPI.create(t('canvasConnect.keyName'), createGroupId.value)
    await load()
    selectedId.value = created.id
    showCreate.value = false
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    creating.value = false
  }
}

async function authorize() {
  const key = selectedKey.value
  if (authorizing.value || !canHandOff || !window.opener || window.opener.closed) return
  authorizing.value = true
  try {
    await createCanvasSession()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
    authorizing.value = false
    return
  }
  const message = key ? { type: CANVAS_CONNECT_MESSAGE_TYPE, state, signedIn: true, apiKey: key.key, keyName: key.name } : { type: CANVAS_CONNECT_MESSAGE_TYPE, state, signedIn: true }
  window.opener.postMessage(message, canvasOrigin)
  done.value = true
  setTimeout(() => window.close(), 1500)
}

function cancel() {
  window.close()
}

onMounted(() => {
  if (canHandOff) void load()
  else loading.value = false
})
</script>
