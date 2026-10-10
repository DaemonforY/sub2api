<template>
  <div class="card border-red-200 dark:border-red-900/50" data-testid="profile-delete-account">
    <div class="flex flex-wrap items-center justify-between gap-3 px-6 py-4">
      <div>
        <h2 class="text-lg font-medium text-gray-900 dark:text-white">{{ t('accountDeletion.title') }}</h2>
        <p v-if="isAdmin" class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('accountDeletion.adminHint') }}</p>
      </div>
      <button v-if="!isAdmin && !open" class="btn btn-secondary text-red-600" data-testid="delete-account-open" @click="open = true">
        {{ t('accountDeletion.open') }}
      </button>
    </div>
    <div v-if="open" class="space-y-4 border-t border-gray-100 px-6 py-5 dark:border-dark-700">
      <p class="text-sm text-gray-700 dark:text-dark-200">{{ t('accountDeletion.intro') }}</p>
      <ul class="list-disc space-y-1.5 pl-5 text-sm text-gray-600 dark:text-dark-300">
        <li>{{ t('accountDeletion.keys') }}</li>
        <li v-if="balance > 0" class="font-medium text-red-600 dark:text-red-400" data-testid="delete-account-balance">
          {{ t('accountDeletion.balance', { amount: formatCurrency(balance) }) }}
        </li>
        <li>{{ t('accountDeletion.content') }}</li>
        <li>{{ t('accountDeletion.personal') }}</li>
        <li>{{ t('accountDeletion.reuse') }}</li>
      </ul>
      <label class="block text-sm">
        <span class="input-label">{{ t('accountDeletion.typeToConfirm', { phrase }) }}</span>
        <input v-model="typed" class="input max-w-xs" autocomplete="off" data-testid="delete-account-confirm" />
      </label>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-secondary" :disabled="busy" @click="cancel">{{ t('common.cancel') }}</button>
        <button
          class="btn bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
          :disabled="busy || typed.trim() !== phrase"
          data-testid="delete-account-submit"
          @click="submit"
        >
          {{ busy ? t('accountDeletion.submitting') : t('accountDeletion.submit') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { deleteAccount } from '@/api/user'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency } from '@/utils/format'

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const open = ref(false)
const typed = ref('')
const busy = ref(false)
const isAdmin = computed(() => authStore.user?.role === 'admin')
const balance = computed(() => authStore.user?.balance || 0)
// The server accepts either phrase.
const phrase = computed(() => (String(locale.value).startsWith('zh') ? '注销账号' : 'DELETE'))

function cancel() {
  open.value = false
  typed.value = ''
}

async function submit() {
  busy.value = true
  try {
    await deleteAccount(typed.value.trim())
    appStore.showSuccess(t('accountDeletion.done'))
    await authStore.logout()
    router.replace('/')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}
</script>
