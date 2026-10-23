<template>
  <BaseDialog
    :show="visible"
    :title="t('profile.requireNickname.title')"
    width="narrow"
    :close-on-escape="false"
    :close-on-click-outside="false"
    :show-close-button="false"
    :z-index="75"
    @close="noop"
  >
    <form class="space-y-4" @submit.prevent="save">
      <p class="text-sm leading-6 text-gray-600 dark:text-dark-300">
        {{ t('profile.requireNickname.description', { name: currentName }) }}
      </p>
      <div>
        <label for="require-nickname" class="input-label">{{ t('profile.requireNickname.label') }}</label>
        <input
          id="require-nickname"
          v-model="nickname"
          type="text"
          class="input"
          :maxlength="NICKNAME_MAX"
          autocomplete="nickname"
          :placeholder="t('profile.requireNickname.placeholder')"
          :disabled="saving"
        />
        <p v-if="error" class="mt-1.5 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      </div>
      <button type="submit" class="btn btn-primary w-full" :disabled="saving">
        {{ saving ? t('profile.requireNickname.saving') : t('profile.requireNickname.save') }}
      </button>
    </form>

    <template v-if="terminal !== 'mobile'">
      <div class="my-4 flex items-center gap-3 text-xs text-gray-400 dark:text-dark-500">
        <span class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></span>
        {{ t('profile.requireNickname.or') }}
        <span class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></span>
      </div>
      <button v-if="terminal === 'wechat'" type="button" class="btn btn-secondary w-full" @click="syncInWeChat">
        {{ t('profile.requireNickname.syncInWeChat') }}
      </button>
      <template v-else>
        <button type="button" class="btn btn-secondary w-full" @click="toggleQr">
          {{ qrDataUrl ? t('profile.requireNickname.syncQrHide') : t('profile.requireNickname.syncQrAction') }}
        </button>
        <div v-if="qrDataUrl" class="mt-4 flex flex-col items-center gap-3">
          <img :src="qrDataUrl" :alt="t('profile.requireNickname.syncQrAlt')" class="h-44 w-44 rounded-lg border border-gray-200 dark:border-dark-700" />
          <p class="text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ t('profile.requireNickname.syncQrHint') }}
          </p>
        </div>
      </template>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
// Accounts still carrying the server-made 「微信用户_xxxx」 name must pick a nickname before using
// the user center: type one, or copy the WeChat nickname through 网页授权 (in WeChat directly, or
// by scanning a QR on a computer while this dialog polls for the new name).
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import QRCode from 'qrcode'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { userAPI } from '@/api'
import { buildWeChatProfileSyncURL, detectWeChatLoginTerminal } from '@/api/auth'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isWeChatFallbackUsername } from '@/utils/userDisplay'

const NICKNAME_MAX = 30
const SYNC_POLL_MS = 4000
const SYNC_POLL_MAX_TICKS = 75 // 5 minutes

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const terminal = detectWeChatLoginTerminal()
const nickname = ref('')
const error = ref('')
const saving = ref(false)
const qrDataUrl = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const currentName = computed(() => authStore.user?.username?.trim() || '')
const visible = computed(() => {
  const user = authStore.user
  return Boolean(user) && user?.role !== 'admin' && isWeChatFallbackUsername(user?.username)
})

function noop(): void {}

async function save(): Promise<void> {
  const name = nickname.value.trim()
  error.value = ''
  if (!name) {
    error.value = t('profile.usernameRequired')
    return
  }
  if ([...name].length > NICKNAME_MAX) {
    error.value = t('profile.requireNickname.tooLong', { max: NICKNAME_MAX })
    return
  }
  if (isWeChatFallbackUsername(name)) {
    error.value = t('profile.requireNickname.stillDefault')
    return
  }
  saving.value = true
  try {
    authStore.user = await userAPI.updateProfile({ username: name })
  } catch (err: unknown) {
    error.value = extractApiErrorMessage(err, t('profile.updateFailed'))
  } finally {
    saving.value = false
  }
}

function syncInWeChat(): void {
  window.location.href = buildWeChatProfileSyncURL(route.fullPath || '/dashboard')
}

function stopPolling(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// The phone finishes 网页授权 on its own; this page only sees the result by re-reading the user.
function startPolling(): void {
  stopPolling()
  let ticks = 0
  let inFlight = false
  pollTimer = setInterval(async () => {
    ticks += 1
    if (ticks > SYNC_POLL_MAX_TICKS) {
      stopPolling()
      qrDataUrl.value = ''
      return
    }
    if (inFlight) return
    inFlight = true
    try {
      await authStore.refreshUser()
    } catch {
      // keep polling; a network blip shouldn't end the wait
    } finally {
      inFlight = false
    }
  }, SYNC_POLL_MS)
}

async function toggleQr(): Promise<void> {
  if (qrDataUrl.value) {
    qrDataUrl.value = ''
    stopPolling()
    return
  }
  try {
    const target = new URL(buildWeChatProfileSyncURL('/profile'), window.location.origin).href
    qrDataUrl.value = await QRCode.toDataURL(target, { width: 320, margin: 1, errorCorrectionLevel: 'M' })
  } catch {
    appStore.showError(t('errors.tryAgain'))
    return
  }
  startPolling()
}

// Closes by itself once the name changes, whether typed here or synced from the phone.
watch(visible, (show, wasShown) => {
  if (show || !wasShown) return
  stopPolling()
  qrDataUrl.value = ''
  nickname.value = ''
  if (authStore.user) appStore.showSuccess(t('profile.requireNickname.saved', { name: currentName.value }))
})

onBeforeUnmount(stopPolling)
</script>
