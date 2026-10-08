<template>
  <div data-testid="login-image-captcha">
    <label for="login-captcha" class="input-label">{{ t('auth.imageCaptcha.label') }}</label>
    <div class="flex items-center gap-3">
      <input
        id="login-captcha"
        v-model="code"
        type="text"
        inputmode="text"
        maxlength="4"
        autocomplete="off"
        autocapitalize="characters"
        spellcheck="false"
        :disabled="disabled"
        class="input min-w-0 flex-1 uppercase tracking-[0.3em]"
        :placeholder="t('auth.imageCaptcha.placeholder')"
      />
      <button
        type="button"
        class="h-11 w-[132px] shrink-0 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800"
        :title="t('auth.imageCaptcha.refresh')"
        :disabled="loading || disabled"
        data-testid="login-image-captcha-refresh"
        @click="refresh"
      >
        <img
          v-if="image"
          :src="image"
          :alt="t('auth.imageCaptcha.alt')"
          width="132"
          height="44"
          class="block h-full w-full"
        />
        <span v-else class="flex h-full items-center justify-center text-xs text-gray-400">
          {{ loading ? t('auth.imageCaptcha.loading') : t('auth.imageCaptcha.reload') }}
        </span>
      </button>
    </div>
    <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
      {{ t('auth.imageCaptcha.hint') }} · {{ t('auth.imageCaptcha.refresh') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getLoginCaptcha } from '@/api/auth'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

defineProps<{ disabled?: boolean }>()

const code = defineModel<string>('code', { default: '' })
const captchaId = defineModel<string>('captchaId', { default: '' })

const { t } = useI18n()
const appStore = useAppStore()
const image = ref('')
const loading = ref(false)

// Each captcha works for one login attempt, so the form calls this after every try.
async function refresh(): Promise<void> {
  loading.value = true
  code.value = ''
  try {
    const res = await getLoginCaptcha()
    captchaId.value = res.captcha_id
    image.value = res.image
  } catch (error: unknown) {
    captchaId.value = ''
    image.value = ''
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(refresh)

defineExpose({ refresh })
</script>
