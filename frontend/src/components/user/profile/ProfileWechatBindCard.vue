<template>
  <section
    v-if="visible"
    class="card flex items-start gap-3 border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900/40 dark:bg-emerald-900/10"
    data-testid="profile-wechat-bind-reminder"
  >
    <div
      class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300"
    >
      <Icon name="chatBubble" size="sm" />
    </div>
    <div class="min-w-0 flex-1">
      <p class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('profile.wechatBindReminder.title') }}
      </p>
      <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">
        {{ t('profile.wechatBindReminder.subtitle') }}
      </p>
      <button
        type="button"
        class="btn btn-primary btn-sm mt-3"
        data-testid="profile-wechat-bind-reminder-bind"
        @click="emit('bind')"
      >
        {{ t('profile.wechatBindReminder.action') }}
      </button>
    </div>
    <button
      type="button"
      class="rounded-lg p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
      :aria-label="t('profile.wechatBindReminder.dismiss')"
      data-testid="profile-wechat-bind-reminder-dismiss"
      @click="dismiss"
    >
      ✕
    </button>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { User, UserAuthBindingStatus } from '@/types'
import { isPlaceholderEmail } from '@/utils/userDisplay'

/**
 * Nudge for email accounts that haven't bound WeChat: sign-up is WeChat-first now, and a bound
 * WeChat lets them sign in by scanning and get reminders there. Dismissed per user in localStorage.
 */
const props = defineProps<{ user: User | null; wechatEnabled: boolean }>()
const emit = defineEmits<{ bind: [] }>()

const { t } = useI18n()
const STORAGE_PREFIX = 'hg_profile_wechat_bind_dismissed'

const dismissed = ref(false)

function storageKey(): string {
  return `${STORAGE_PREFIX}:${props.user?.id ?? ''}`
}

watch(
  () => props.user?.id,
  () => {
    try {
      dismissed.value = localStorage.getItem(storageKey()) === '1'
    } catch {
      dismissed.value = false
    }
  },
  { immediate: true }
)

function bindingBound(binding: boolean | UserAuthBindingStatus | undefined): boolean {
  if (typeof binding === 'boolean') return binding
  return Boolean(binding?.bound)
}

const wechatBound = computed(() => {
  const user = props.user
  if (typeof user?.wechat_bound === 'boolean') return user.wechat_bound
  return bindingBound(user?.auth_bindings?.wechat ?? user?.identity_bindings?.wechat)
})

const visible = computed(
  () =>
    props.wechatEnabled &&
    Boolean(props.user?.email) &&
    !isPlaceholderEmail(props.user?.email) &&
    !wechatBound.value &&
    !dismissed.value
)

function dismiss(): void {
  dismissed.value = true
  try {
    localStorage.setItem(storageKey(), '1')
  } catch {
    // not persisted; hidden for this visit
  }
}
</script>
