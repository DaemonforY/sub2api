<template>
  <div
    v-if="open"
    class="fixed inset-0 z-[60] flex items-end justify-center bg-black/50 p-3 backdrop-blur-sm sm:items-center"
    data-testid="home-signup-sheet"
    @click.self="emit('close')"
  >
    <div role="dialog" aria-modal="true" :aria-label="t('home.v3.sheet.title')" class="relative w-full max-w-md rounded-3xl bg-white p-6 shadow-2xl dark:bg-dark-900">
      <button type="button" class="absolute right-4 top-4 rounded-lg p-1 text-gray-400 hover:text-gray-600 dark:hover:text-dark-200" :aria-label="t('home.v3.sheet.close')" @click="emit('close')">✕</button>
      <h3 class="pr-8 text-xl font-bold text-gray-900 dark:text-white">{{ t('home.v3.sheet.title') }}</h3>
      <div class="mt-4 rounded-xl bg-gray-100 px-4 py-3 text-sm text-gray-600 dark:bg-dark-800 dark:text-dark-300">
        <p class="line-clamp-3">{{ t('home.v3.sheet.prompt') }}<span class="text-gray-900 dark:text-white" data-testid="home-signup-sheet-prompt">{{ prompt }}</span></p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('home.v3.sheet.promptHint') }}</p>
      </div>
      <ul class="mt-4 space-y-2 text-sm text-gray-700 dark:text-dark-200">
        <li v-if="bonus > 0" class="flex gap-2">
          <span class="text-amber-500" aria-hidden="true">◆</span>
          <span>{{ t('home.v3.sheet.perkBonus', { amount: bonusText }) }}{{ images > 0 ? t('home.v3.sheet.perkImages', { images }) : '' }}</span>
        </li>
        <li class="flex gap-2"><span class="text-teal-500" aria-hidden="true">◆</span>{{ t('home.v3.sheet.perkModels') }}</li>
      </ul>
      <router-link
        v-if="registrationEnabled"
        :to="{ path: '/register', query: wechatPanel ? { redirect: REDIRECT, wechat: '1' } : { redirect: REDIRECT } }"
        :class="['btn mt-6 w-full py-3 text-base', wechat ? 'bg-[#07c160] text-white hover:bg-[#06ad56]' : 'btn-primary']"
        data-testid="home-signup-sheet-register"
        @click="track('cta_click', { where: 'sheet-register' })"
      >
        {{ wechat ? t('home.v3.sheet.wechat') : t('home.v3.sheet.register') }}
      </router-link>
      <router-link
        :to="{ path: '/login', query: { redirect: REDIRECT } }"
        :class="registrationEnabled ? 'mt-4 block text-center text-sm text-gray-500 hover:text-primary-600 dark:text-dark-400' : 'btn btn-primary mt-6 w-full py-3 text-base'"
        data-testid="home-signup-sheet-login"
        @click="track('cta_click', { where: 'sheet-login' })"
      >
        {{ t('home.v3.sheet.login') }}
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { isWeChatWebOAuthEnabled, resolveWeChatLoginPanels } from '@/api/auth'
import { useAppStore } from '@/stores'
import { track } from '@/utils/analytics'

/**
 * Opened by the hero's 生成 for visitors: shows the sentence they wrote (already saved by the page)
 * and sends them through sign-up — or login — to /first-image, which takes it to the canvas.
 */
const REDIRECT = '/first-image'

const props = defineProps<{ open: boolean; prompt: string; bonus: number; images: number; registrationEnabled: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const onKey = (e: KeyboardEvent) => e.key === 'Escape' && emit('close')
watch(
  () => props.open,
  (open) => (open ? window.addEventListener('keydown', onKey) : window.removeEventListener('keydown', onKey))
)
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

const bonusText = computed(() => String(Math.round(props.bonus * 100) / 100))
// The register page leads with WeChat sign-in inside WeChat (网页授权) and, when the 公众号 QR / code
// login is on, in other browsers too (?wechat=1 opens that panel), so say so on the button.
const inWeChat = typeof navigator !== 'undefined' && /MicroMessenger/i.test(navigator.userAgent || '')
const wechatPanel = computed(() => !inWeChat && resolveWeChatLoginPanels(appStore.cachedPublicSettings).panel !== null)
const wechat = computed(
  () => (inWeChat && isWeChatWebOAuthEnabled(appStore.cachedPublicSettings)) || wechatPanel.value
)
</script>
