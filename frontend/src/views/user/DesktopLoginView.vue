<template>
  <div class="flex min-h-screen items-start justify-center bg-gray-50 px-4 py-8 dark:bg-dark-950" data-testid="desktop-login">
    <div class="card w-full max-w-md p-6">
      <div class="flex items-center gap-3">
        <img :src="siteLogo || '/logo.svg'" alt="" class="h-9 w-9 rounded-lg object-contain" />
        <div>
          <h1 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('desktopLogin.title') }}</h1>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('desktopLogin.subtitle') }}</p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-10">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="outcome === 'approved'" class="mt-6 py-6 text-center" data-testid="desktop-login-approved">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-2xl text-emerald-600 dark:bg-emerald-900/30">✓</div>
        <p class="mt-3 font-medium text-gray-900 dark:text-white">{{ t('desktopLogin.approved') }}</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('desktopLogin.approvedHint') }}</p>
        <p class="mt-4 text-xs leading-5 text-gray-500 dark:text-dark-400" data-testid="desktop-login-not-me">
          {{ t('desktopLogin.approvedNotMe', { device: info?.device_name ?? '' }) }}
          <router-link to="/keys" class="text-primary-600 underline dark:text-primary-400">{{ t('desktopLogin.goKeys') }}</router-link>
        </p>
      </div>

      <div v-else-if="outcome === 'rejected'" class="mt-6 py-6 text-center" data-testid="desktop-login-rejected">
        <p class="font-medium text-gray-900 dark:text-white">{{ t('desktopLogin.rejected') }}</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('desktopLogin.rejectedHint') }}</p>
      </div>

      <div
        v-else-if="message"
        class="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200"
        data-testid="desktop-login-message"
      >
        {{ message }}
      </div>

      <template v-else-if="info">
        <p class="mt-5 text-base font-medium text-gray-900 dark:text-white" data-testid="desktop-login-device">
          {{ t('desktopLogin.asking', { device: info.device_name }) }}
        </p>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('desktopLogin.explain') }}</p>
        <div
          class="mt-4 rounded-xl border p-3 text-sm leading-6"
          :class="
            info.same_network
              ? 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300'
              : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200'
          "
          data-testid="desktop-login-network"
        >
          <p>{{ t('desktopLogin.network', { network: info.computer_network || t('desktopLogin.unknownNetwork'), ago: agoText }) }}</p>
          <p class="mt-1">{{ info.same_network ? t('desktopLogin.sameNetwork') : t('desktopLogin.otherNetwork') }}</p>
          <label v-if="!info.same_network" class="mt-2 flex items-start gap-2 font-medium">
            <input v-model="nearby" type="checkbox" class="mt-1" data-testid="desktop-login-nearby" />
            <span>{{ t('desktopLogin.confirmNearby') }}</span>
          </label>
        </div>
        <p class="mt-3 text-xs leading-5 text-red-600 dark:text-red-400" data-testid="desktop-login-phishing">{{ t('desktopLogin.phishing') }}</p>
        <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ t('desktopLogin.notYou') }}</p>
        <div class="mt-6 flex gap-3">
          <button class="btn btn-secondary flex-1" :disabled="submitting" data-testid="desktop-login-reject" @click="decide(false)">
            {{ t('desktopLogin.reject') }}
          </button>
          <button
            class="btn btn-primary flex-1"
            :disabled="submitting || (!info.same_network && !nearby)"
            data-testid="desktop-login-approve"
            @click="decide(true)"
          >
            {{ t('desktopLogin.approve') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { decideDesktopLogin, getDesktopLogin, type DesktopLoginInfo } from '@/api/desktop'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

/**
 * Opened by scanning the QR in HiveGPT 桌面助手 (/desktop-login?t=<ticket>). The router has already
 * made the user sign in (in WeChat that is the 公众号 in-app sign-in). Nothing is granted until the
 * user taps 确认登录; the desktop app then picks up its device key by polling.
 */
const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || '')
const ticketId = typeof route.query.t === 'string' ? route.query.t : ''

const loading = ref(true)
const submitting = ref(false)
const info = ref<DesktopLoginInfo | null>(null)
const message = ref('')
const outcome = ref<'' | 'approved' | 'rejected'>('')
/** 「这台电脑就在我面前」, required when the computer is on another network. */
const nearby = ref(false)

const agoText = computed(() => {
  const s = info.value?.started_ago ?? 0
  return s < 60 ? t('desktopLogin.justNow') : t('desktopLogin.minutesAgo', { n: Math.floor(s / 60) })
})

onMounted(async () => {
  if (!/^[a-f0-9]{32}$/.test(ticketId)) {
    message.value = t('desktopLogin.invalid')
    loading.value = false
    return
  }
  try {
    const res = await getDesktopLogin(ticketId)
    if (res.status === 'pending') info.value = res
    else message.value = t('desktopLogin.handled')
  } catch (err) {
    message.value = extractApiErrorMessage(err, t('desktopLogin.invalid'))
  } finally {
    loading.value = false
  }
})

async function decide(approve: boolean) {
  submitting.value = true
  try {
    const res = await decideDesktopLogin(ticketId, approve, approve && nearby.value)
    outcome.value = res.status === 'approved' ? 'approved' : 'rejected'
  } catch (err) {
    message.value = extractApiErrorMessage(err, t('desktopLogin.loadFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
