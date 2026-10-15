<template>
  <section
    v-if="visible"
    class="card flex items-start gap-3 border border-sky-200 bg-sky-50 p-4 dark:border-sky-900/40 dark:bg-sky-900/10"
    data-testid="profile-email-reminder"
  >
    <div
      class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-300"
    >
      <Icon name="mail" size="sm" />
    </div>
    <div class="min-w-0 flex-1">
      <p class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('profile.emailReminder.title') }}
      </p>
      <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300" data-testid="profile-email-reminder-subtitle">
        {{ wechatDelivers ? t('profile.emailReminder.subtitleWeChat') : t('profile.emailReminder.subtitle') }}
      </p>
      <button
        type="button"
        class="btn btn-primary btn-sm mt-3"
        data-testid="profile-email-reminder-bind"
        @click="emit('bind')"
      >
        {{ t('profile.emailReminder.action') }}
      </button>
    </div>
    <button
      type="button"
      class="rounded-lg p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
      :aria-label="t('profile.emailReminder.dismiss')"
      data-testid="profile-email-reminder-dismiss"
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
 * Optional nudge for accounts whose email is a sign-up placeholder (…@wechat-connect.invalid):
 * bills and expiry reminders need a real address unless they already arrive in WeChat.
 * Dismissed per user in localStorage.
 */
const props = defineProps<{ user: User | null }>()
const emit = defineEmits<{ bind: [] }>()

const { t } = useI18n()
const STORAGE_PREFIX = 'hg_profile_email_reminder_dismissed'

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

const emailBound = computed(() => {
  const user = props.user
  if (typeof user?.email_bound === 'boolean') return user.email_bound
  return bindingBound(user?.auth_bindings?.email ?? user?.identity_bindings?.email)
})

const wechatBound = computed(() => {
  const user = props.user
  if (typeof user?.wechat_bound === 'boolean') return user.wechat_bound
  return bindingBound(user?.auth_bindings?.wechat ?? user?.identity_bindings?.wechat)
})

/**
 * Reminders already go to WeChat once it is bound: scanning or sending a code means following the
 * 公众号. Only an explicit wechat_mp_subscribed: false (unfollowed) says otherwise.
 */
const wechatDelivers = computed(() => wechatBound.value && props.user?.wechat_mp_subscribed !== false)

const visible = computed(
  () => Boolean(props.user) && isPlaceholderEmail(props.user?.email) && !emailBound.value && !dismissed.value
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
