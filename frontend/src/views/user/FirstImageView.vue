<template>
  <div class="flex min-h-screen items-start justify-center bg-gray-50 px-4 py-10 dark:bg-dark-950" data-testid="first-image">
    <div class="card w-full max-w-md p-6">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" alt="" class="h-9 w-9 rounded-lg object-contain" />
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.v3.firstImage.title') }}</h1>
      </div>

      <ul class="mt-5 space-y-2.5 text-sm text-gray-700 dark:text-dark-200">
        <li class="flex items-start gap-2"><span class="text-emerald-500" aria-hidden="true">✓</span>{{ t('home.v3.firstImage.account') }}</li>
        <li v-if="balance > 0" class="flex items-start gap-2" data-testid="first-image-balance">
          <span class="text-emerald-500" aria-hidden="true">✓</span>{{ t('home.v3.firstImage.balance', { amount: balanceText }) }}
        </li>
        <li class="flex items-start gap-2">
          <span v-if="keyState === 'ready'" class="text-emerald-500" aria-hidden="true">✓</span>
          <span v-else-if="keyState === 'loading'" class="mt-1 h-3 w-3 shrink-0 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" aria-hidden="true"></span>
          <span v-else class="text-amber-500" aria-hidden="true">!</span>
          <span>{{ keyState === 'failed' ? t('home.v3.firstImage.keyFailed') : t('home.v3.firstImage.key') }}</span>
        </li>
      </ul>

      <div v-if="prompt" class="mt-5 rounded-xl bg-gray-100 px-4 py-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-dark-300" data-testid="first-image-prompt">
        <span class="text-gray-500 dark:text-dark-400">{{ t('home.v3.firstImage.prompt') }}</span>{{ prompt }}
      </div>

      <p class="mt-5 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('home.v3.firstImage.next', { site: siteName }) }}</p>

      <a :href="canvasHref" class="btn btn-primary mt-5 w-full py-3 text-base" data-testid="first-image-go" @click="cancelAuto">
        {{ t('home.v3.firstImage.go') }}
      </a>
      <p v-if="countdown > 0" class="mt-3 text-center text-xs text-gray-500 dark:text-dark-400">{{ t('home.v3.firstImage.auto', { n: countdown }) }}</p>
      <p class="mt-4 text-center text-xs">
        <router-link to="/dashboard" class="text-gray-500 hover:text-primary-600 dark:text-dark-400" @click="cancelAuto">{{ t('home.v3.firstImage.dashboard') }}</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { canvasUrl } from '@/constants/crossSites'
import { takePendingPrompt } from '@/utils/pendingPrompt'

/**
 * Where a visitor lands after signing up from the home page's 「生成」: the picture they described
 * waits in localStorage. Make sure they have a key (the starter key in the pay-as-you-go group, so
 * the canvas's 「用 HiveGPT 账号连接」 finds one that can draw), then open the canvas with the prompt.
 */
const AUTO_SECONDS = 4

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || '')
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'HiveGPT')
const prompt = takePendingPrompt()
const canvasHref = canvasUrl({ medium: 'first-image', baseUrl: window.location.origin, path: '/image', prompt })
const keyState = ref<'loading' | 'ready' | 'failed'>('loading')
const balance = computed(() => Number(authStore.user?.balance || 0))
const balanceText = computed(() => String(Math.round(balance.value * 100) / 100))
const countdown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined

function cancelAuto() {
  if (timer) clearInterval(timer)
  timer = undefined
  countdown.value = 0
}

onMounted(async () => {
  authStore.refreshUser().catch(() => {})
  try {
    await keysAPI.starter()
    keyState.value = 'ready'
  } catch {
    keyState.value = 'failed' // the canvas's connect dialog can still make one
    return
  }
  countdown.value = AUTO_SECONDS
  timer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      cancelAuto()
      window.location.assign(canvasHref)
    }
  }, 1000)
})

onBeforeUnmount(cancelAuto)
</script>
