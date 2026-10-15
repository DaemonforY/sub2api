<template>
  <div
    class="flex min-h-screen items-start justify-center bg-gray-50 px-4 py-10 dark:bg-dark-950"
    data-testid="wechat-synced"
  >
    <div class="card w-full max-w-md p-6 text-center">
      <div
        :class="[
          'mx-auto flex h-12 w-12 items-center justify-center rounded-full text-2xl',
          status === 'ok'
            ? 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300'
            : 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300',
        ]"
        aria-hidden="true"
      >
        {{ status === 'ok' ? '✓' : '!' }}
      </div>

      <template v-if="status === 'ok'">
        <h1 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white" data-testid="wechat-synced-title">
          {{ name ? t('auth.wechatSynced.okTitle', { name }) : t('auth.wechatSynced.okTitleNoName') }}
        </h1>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ t('auth.wechatSynced.okHint') }}
        </p>
      </template>
      <template v-else-if="status === 'not_bound'">
        <h1 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white" data-testid="wechat-synced-title">
          {{ t('auth.wechatSynced.notBoundTitle') }}
        </h1>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ t('auth.wechatSynced.notBoundHint') }}
        </p>
      </template>
      <template v-else>
        <h1 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white" data-testid="wechat-synced-title">
          {{ t('auth.wechatSynced.errorTitle') }}
        </h1>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ t('auth.wechatSynced.errorHint') }}
        </p>
      </template>

      <a
        v-if="continuePath"
        :href="continuePath"
        class="btn btn-primary mt-6 w-full"
        data-testid="wechat-synced-continue"
      >
        {{ t('auth.wechatSynced.continue') }}
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/**
 * End of 同步微信昵称头像 (GET /auth/oauth/wechat/sync-profile/start → 网页授权 → here). Usually opens
 * inside WeChat right after a PC sign-in, so it needs no login; when this browser is signed in, the
 * user in the store is refreshed so the new name shows.
 */
const SELF_PATH = '/auth/wechat/synced'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()

function queryString(value: unknown): string {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

const status = computed(() => {
  const value = queryString(route.query.status)
  return value === 'ok' || value === 'not_bound' ? value : 'error'
})
const name = computed(() => queryString(route.query.name).slice(0, 64))

/** Same-site path to continue to; never an external URL or this page itself. */
const continuePath = computed(() => {
  const path = queryString(route.query.redirect)
  if (!path.startsWith('/') || path.startsWith('//') || path.includes('://') || /[\r\n\\]/.test(path) || path.length > 512) {
    return ''
  }
  const pathname = path.split(/[?#]/)[0].replace(/\/+$/, '')
  if (pathname === SELF_PATH) return ''
  return path
})

onMounted(() => {
  if (status.value === 'ok' && authStore.isAuthenticated) {
    authStore.refreshUser().catch(() => {
      // the profile page reloads the user anyway
    })
  }
})
</script>
