<template>
  <div
    class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800/60"
    data-testid="wechat-login-panel"
  >
    <!-- PC: scan the 公众号 QR code with WeChat -->
    <div v-if="view === 'qr'" class="flex flex-col items-center text-center" data-testid="wechat-qr-panel">
      <div
        class="relative h-48 w-48 overflow-hidden rounded-xl border border-gray-100 bg-white p-2 dark:border-dark-600"
      >
        <img
          v-if="qrUrl"
          :src="qrUrl"
          :alt="t('auth.wechatLogin.qrAlt')"
          class="h-full w-full object-contain"
          data-testid="wechat-qr-image"
        />
        <div
          v-if="loading"
          class="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-dark-900/80"
        >
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-green-500 border-t-transparent"></div>
        </div>
        <button
          v-else-if="ticketState !== 'active'"
          type="button"
          class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-white/95 px-4 text-sm font-medium text-gray-700 dark:bg-dark-900/95 dark:text-dark-200"
          data-testid="wechat-qr-refresh"
          :disabled="disabled"
          @click="createTicket"
        >
          <span class="text-2xl text-green-600 dark:text-green-400" aria-hidden="true">↻</span>
          <span>{{ overlayText }}</span>
        </button>
        <div
          v-else-if="scanned"
          class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-white/95 px-4 text-sm font-medium text-gray-700 dark:bg-dark-900/95 dark:text-dark-200"
          data-testid="wechat-qr-scanned"
        >
          <span class="text-2xl text-green-600 dark:text-green-400" aria-hidden="true">✓</span>
          <span>{{ t('auth.wechatLogin.qrScanned') }}</span>
        </div>
      </div>
      <p class="mt-3 text-sm font-medium text-gray-800 dark:text-dark-100">
        {{ t('auth.wechatLogin.qrCaption') }}
      </p>
      <p v-if="accountName" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
        {{ t('auth.wechatLogin.qrAccountHint', { name: accountName }) }}
      </p>
      <p
        v-if="terminal === 'mobile'"
        class="mt-1 text-xs text-gray-500 dark:text-dark-400"
        data-testid="wechat-qr-save-hint"
      >
        {{ t('auth.wechatLogin.qrSaveHint') }}
      </p>
      <button
        v-if="codeEnabled"
        type="button"
        class="mt-3 text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
        data-testid="wechat-switch-code"
        @click="switchTo('code')"
      >
        {{ t('auth.wechatLogin.useCode') }}
      </button>
    </div>

    <!-- Phone browsers (and the PC fallback): send the code shown here to the 公众号 -->
    <div v-else data-testid="wechat-code-panel">
      <p class="text-sm font-medium text-gray-800 dark:text-dark-100">
        {{ accountName ? t('auth.wechatLogin.codeTitle', { name: accountName }) : t('auth.wechatLogin.codeTitleNoName') }}
        <button
          v-if="accountName"
          type="button"
          class="ml-1 whitespace-nowrap text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
          data-testid="wechat-copy-name"
          @click="copyAccountName"
        >
          {{ t('auth.wechatLogin.copyName') }}
        </button>
      </p>

      <div
        class="relative mt-3 flex min-h-[5.5rem] flex-col items-center justify-center rounded-xl bg-green-50 px-4 py-3 dark:bg-green-900/20"
      >
        <template v-if="loginCode && ticketState === 'active'">
          <span
            class="font-mono text-4xl font-bold tracking-[0.3em] text-green-700 dark:text-green-300"
            data-testid="wechat-login-code"
          >{{ loginCode }}</span>
          <button
            type="button"
            class="mt-2 text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
            data-testid="wechat-copy-code"
            @click="copyLoginCode"
          >
            {{ t('auth.wechatLogin.copyCode') }}
          </button>
        </template>
        <div
          v-if="loading"
          class="absolute inset-0 flex items-center justify-center rounded-xl bg-green-50/90 dark:bg-dark-900/80"
        >
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-green-500 border-t-transparent"></div>
        </div>
        <button
          v-else-if="ticketState !== 'active'"
          type="button"
          class="absolute inset-0 flex flex-col items-center justify-center gap-1 rounded-xl bg-green-50/95 px-4 text-sm font-medium text-gray-700 dark:bg-dark-900/95 dark:text-dark-200"
          data-testid="wechat-code-refresh"
          :disabled="disabled"
          @click="createTicket"
        >
          <span class="text-xl text-green-600 dark:text-green-400" aria-hidden="true">↻</span>
          <span>{{ overlayText }}</span>
        </button>
      </div>
      <p
        v-if="ticketState === 'active'"
        class="mt-2 text-center text-xs text-gray-500 dark:text-dark-400"
        data-testid="wechat-code-waiting"
      >
        {{ scanned ? t('auth.wechatLogin.qrScanned') : t('auth.wechatLogin.codeWaiting') }}
      </p>

      <a
        v-if="terminal === 'mobile'"
        href="weixin://"
        class="btn btn-primary mt-3 w-full"
        data-testid="wechat-open-app"
      >
        {{ t('auth.wechatLogin.openWeChat') }}
      </a>

      <ol class="mt-4 space-y-2 text-sm text-gray-700 dark:text-dark-200">
        <li v-for="(step, index) in codeSteps" :key="index" class="flex gap-3">
          <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-green-100 text-xs font-semibold text-green-700 dark:bg-green-900/30 dark:text-green-300">{{ index + 1 }}</span>
          <p class="min-w-0 flex-1">{{ step }}</p>
        </li>
      </ol>

      <div v-if="followQrUrl" class="mt-3 flex items-center gap-3" data-testid="wechat-follow-qr">
        <img
          :src="followQrUrl"
          :alt="t('auth.wechatLogin.followQrAlt')"
          class="h-24 w-24 shrink-0 rounded-lg border border-gray-100 bg-white object-contain p-1 dark:border-dark-600"
        />
        <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">
          {{ t('auth.wechatLogin.followQrHint') }}
        </p>
      </div>

      <button
        v-if="qrEnabled && terminal === 'pc'"
        type="button"
        class="mt-3 text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
        data-testid="wechat-switch-qr"
        @click="switchTo('qr')"
      >
        {{ t('auth.wechatLogin.useQr') }}
      </button>
    </div>

    <!-- Phones: the page can also be opened inside WeChat, where 网页授权 signs in directly. -->
    <div
      v-if="terminal === 'mobile'"
      class="mt-4 rounded-xl bg-gray-50 px-3 py-2.5 text-xs leading-5 text-gray-500 dark:bg-dark-900/60 dark:text-dark-400"
      data-testid="wechat-open-in-wechat"
    >
      {{ t('auth.wechatLogin.openInWeChatHint') }}
      <button
        type="button"
        class="ml-1 font-medium text-primary-600 hover:underline dark:text-primary-400"
        data-testid="wechat-copy-link"
        @click="copyPageLink"
      >
        {{ t('auth.wechatLogin.copyLink') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  createWeChatCodeLogin,
  createWeChatQRLogin,
  getWeChatQRLoginStatus,
  type WeChatLoginPanel,
  type WeChatLoginRequest,
  type WeChatLoginTerminal,
} from '@/api/auth'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ActionCaptchaRequestProof } from '@/types'

/**
 * WeChat sign-in outside the WeChat app, through the 公众号. Both views create a login ticket and
 * poll its status until the 公众号 ties a WeChat user to it:
 * - QR (PC): a temporary QR code; scanning (and following) it signs this browser in.
 * - Code (phones, PC fallback): the page shows a 6-digit code; the user sends it to the 公众号.
 * Both end on /auth/wechat/callback, the same pending-session flow as 网页授权.
 */
const props = withDefaults(defineProps<{
  initialPanel: WeChatLoginPanel
  terminal: WeChatLoginTerminal
  qrEnabled: boolean
  codeEnabled: boolean
  request: WeChatLoginRequest
  /** Captcha / agreement gate run before each request; null cancels it. */
  requestProof?: () => Promise<ActionCaptchaRequestProof | null | undefined>
  accountName?: string
  followQrUrl?: string
  disabled?: boolean
}>(), {
  requestProof: undefined,
  accountName: '',
  followQrUrl: '',
  disabled: false,
})

const CALLBACK_PATH = '/auth/wechat/callback'
const TICKET_STORAGE_PREFIX = 'hg_wechat_login_ticket'
const DEFAULT_POLL_SECONDS = 2
const DEFAULT_EXPIRES_SECONDS = 300
const MAX_POLL_FAILURES = 5

const { t } = useI18n()
const route = useRoute()
const { copyToClipboard } = useClipboard()

const view = ref<WeChatLoginPanel>(props.initialPanel)

type TicketState = 'idle' | 'active' | 'expired' | 'error'
const ticketState = ref<TicketState>('idle')
const loading = ref(false)
const scanned = ref(false)
const errorMessage = ref('')
const qrUrl = ref('')
const loginCode = ref('')
let ticketId = ''
let pollSeconds = DEFAULT_POLL_SECONDS
let pollTimer: ReturnType<typeof setTimeout> | null = null
let expiryTimer: ReturnType<typeof setTimeout> | null = null
let pollInFlight = false
let pollFailures = 0
// Bumped whenever the ticket is replaced or the panel goes away, so late answers are ignored.
let generation = 0

const overlayText = computed(() => {
  const isCode = view.value === 'code'
  if (ticketState.value === 'expired') {
    return isCode ? t('auth.wechatLogin.codeExpired') : t('auth.wechatLogin.qrExpired')
  }
  if (ticketState.value === 'error') {
    const retry = t('auth.wechatLogin.retry')
    return errorMessage.value
      ? `${errorMessage.value} ${retry}`
      : isCode ? t('auth.wechatLogin.codeFailed') : t('auth.wechatLogin.qrFailed')
  }
  return isCode ? t('auth.wechatLogin.codeLoad') : t('auth.wechatLogin.qrLoad')
})

const codeSteps = computed(() => [
  t('auth.wechatLogin.codeStep1'),
  props.accountName
    ? t('auth.wechatLogin.codeStep2', { name: props.accountName })
    : t('auth.wechatLogin.codeStep2NoName'),
  t('auth.wechatLogin.codeStep3'),
])

// ---------- Ticket persistence ----------
// The open ticket is kept per page (and intent) in sessionStorage, so a tab that the phone reloads
// while the user is in WeChat picks the same ticket up again instead of showing a new code.
interface StoredTicket {
  ticket_id: string
  kind: WeChatLoginPanel
  code?: string
  qr_url?: string
  poll_interval?: number
  expires_at: number
}

function ticketStorageKey(): string {
  const path = typeof window !== 'undefined' ? window.location.pathname : ''
  return `${TICKET_STORAGE_PREFIX}:${props.request.intent || 'login'}:${path}`
}

function saveTicket(ticket: StoredTicket): void {
  try {
    sessionStorage.setItem(ticketStorageKey(), JSON.stringify(ticket))
  } catch {
    // storage unavailable: the panel simply creates a new ticket after a reload
  }
}

function clearStoredTicket(): void {
  try {
    sessionStorage.removeItem(ticketStorageKey())
  } catch {
    // ignore
  }
}

function loadTicket(kind: WeChatLoginPanel): StoredTicket | null {
  try {
    const raw = sessionStorage.getItem(ticketStorageKey())
    if (!raw) return null
    const ticket = JSON.parse(raw) as StoredTicket
    const usable = ticket
      && ticket.kind === kind
      && typeof ticket.ticket_id === 'string' && ticket.ticket_id
      && typeof ticket.expires_at === 'number' && ticket.expires_at > Date.now()
      && (kind === 'code' ? Boolean(ticket.code) : Boolean(ticket.qr_url))
    if (!usable) {
      clearStoredTicket()
      return null
    }
    return ticket
  } catch {
    clearStoredTicket()
    return null
  }
}

/** Picks up a still-valid ticket of this kind; returns false when there is none. */
function resumeTicket(): boolean {
  const ticket = loadTicket(view.value)
  if (!ticket) return false
  stopPolling()
  const current = generation
  ticketId = ticket.ticket_id
  if (ticket.kind === 'code') loginCode.value = ticket.code || ''
  else qrUrl.value = ticket.qr_url || ''
  ticketState.value = 'active'
  scanned.value = false
  errorMessage.value = ''
  pollFailures = 0
  pollSeconds = ticket.poll_interval && ticket.poll_interval > 0 ? ticket.poll_interval : DEFAULT_POLL_SECONDS
  expiryTimer = setTimeout(() => markExpired(current), Math.max(0, ticket.expires_at - Date.now()))
  // The user is most likely back from WeChat: check at once.
  void poll(current)
  return true
}

function clearTimers(): void {
  if (pollTimer) clearTimeout(pollTimer)
  if (expiryTimer) clearTimeout(expiryTimer)
  pollTimer = null
  expiryTimer = null
}

function stopPolling(): void {
  generation += 1
  clearTimers()
  pollInFlight = false
}

function safeCallbackPath(redirect: unknown): string {
  if (typeof redirect !== 'string') return CALLBACK_PATH
  const path = redirect.trim()
  if (!path.startsWith('/') || path.startsWith('//') || path.includes('://') || /[\r\n\\]/.test(path)) {
    return CALLBACK_PATH
  }
  return path
}

function goToCallback(redirect: unknown): void {
  stopPolling()
  clearStoredTicket()
  window.location.href = safeCallbackPath(redirect)
}

async function acquireProof(): Promise<ActionCaptchaRequestProof | null> {
  if (!props.requestProof) return {}
  const proof = await props.requestProof()
  return proof ?? null
}

async function createTicket(): Promise<void> {
  if (props.disabled || loading.value) return
  stopPolling()
  const current = generation
  const kind = view.value
  loading.value = true
  scanned.value = false
  errorMessage.value = ''
  try {
    const proof = await acquireProof()
    if (current !== generation) return
    if (!proof) {
      ticketState.value = ticketId ? 'expired' : 'idle'
      return
    }
    let expiresIn: number
    let pollInterval: number
    if (kind === 'qr') {
      const created = await createWeChatQRLogin(props.request, proof)
      if (current !== generation) return
      ticketId = created.ticket_id
      qrUrl.value = created.qr_url
      expiresIn = created.expires_in
      pollInterval = created.poll_interval
    } else {
      const created = await createWeChatCodeLogin(props.request, proof)
      if (current !== generation) return
      ticketId = created.ticket_id
      loginCode.value = created.code
      expiresIn = created.expires_in
      pollInterval = created.poll_interval
    }
    ticketState.value = 'active'
    pollFailures = 0
    pollSeconds = pollInterval > 0 ? pollInterval : DEFAULT_POLL_SECONDS
    const seconds = expiresIn > 0 ? expiresIn : DEFAULT_EXPIRES_SECONDS
    saveTicket({
      ticket_id: ticketId,
      kind,
      ...(kind === 'code' ? { code: loginCode.value } : { qr_url: qrUrl.value }),
      poll_interval: pollSeconds,
      expires_at: Date.now() + seconds * 1000,
    })
    expiryTimer = setTimeout(() => markExpired(current), seconds * 1000)
    schedulePoll(current)
  } catch (error: unknown) {
    if (current !== generation) return
    ticketState.value = 'error'
    errorMessage.value = extractApiErrorMessage(
      error,
      kind === 'code' ? t('auth.wechatLogin.codeFailed') : t('auth.wechatLogin.qrFailed'),
    )
  } finally {
    // A newer request (or switching panels) owns the spinner from here on.
    if (current === generation) loading.value = false
  }
}

function markExpired(current: number): void {
  if (current !== generation) return
  stopPolling()
  clearStoredTicket()
  ticketState.value = 'expired'
  scanned.value = false
}

function schedulePoll(current: number): void {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = setTimeout(() => void poll(current), pollSeconds * 1000)
}

async function poll(current: number): Promise<void> {
  if (current !== generation || !ticketId || pollInFlight) return
  pollTimer = null
  pollInFlight = true
  try {
    const result = await getWeChatQRLoginStatus(ticketId)
    if (current !== generation) return
    pollFailures = 0
    if (result.status === 'done') {
      goToCallback(result.redirect)
      return
    }
    if (result.status === 'expired') {
      markExpired(current)
      return
    }
    if (result.status === 'error') {
      stopPolling()
      clearStoredTicket()
      ticketState.value = 'error'
      scanned.value = false
      errorMessage.value = result.message?.trim() || t('auth.wechatLogin.loginFailed')
      return
    }
    // pending / scanned: keep polling (scanned is only a short step before done).
    scanned.value = result.status === 'scanned'
  } catch (error: unknown) {
    if (current !== generation) return
    // 404: the ticket is gone (expired, or created from another browser).
    if ((error as { status?: number })?.status === 404) {
      markExpired(current)
      return
    }
    pollFailures += 1
    if (pollFailures >= MAX_POLL_FAILURES) {
      stopPolling()
      clearStoredTicket()
      ticketState.value = 'error'
      errorMessage.value = extractApiErrorMessage(error, t('auth.wechatLogin.loginFailed'))
      return
    }
  } finally {
    if (current === generation) pollInFlight = false
  }
  if (current === generation) schedulePoll(current)
}

// Mobile browsers pause timers while the user is in WeChat sending the code: check right away
// when the page comes back.
function onVisibilityChange(): void {
  if (document.visibilityState !== 'visible') return
  if (ticketState.value !== 'active' || !ticketId || pollInFlight) return
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = null
  void poll(generation)
}

function copyAccountName(): void {
  void copyToClipboard(props.accountName, t('auth.wechatLogin.nameCopied'))
}

function copyLoginCode(): void {
  void copyToClipboard(loginCode.value, t('auth.wechatLogin.codeCopied'))
}

function copyPageLink(): void {
  void copyToClipboard(window.location.href, t('auth.wechatLogin.linkCopied'))
}

function switchTo(next: WeChatLoginPanel): void {
  if (view.value === next) return
  stopPolling()
  loading.value = false
  ticketId = ''
  ticketState.value = 'idle'
  view.value = next
  if (!resumeTicket()) void createTicket()
}

onMounted(() => {
  document.addEventListener('visibilitychange', onVisibilityChange)
  if (!resumeTicket()) void createTicket()
})

watch(
  () => route?.path,
  (path, previous) => {
    if (path !== previous) stopPolling()
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
  stopPolling()
})
</script>
