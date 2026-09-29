<template>
  <div v-if="status?.enabled" class="card" data-testid="edu-verify-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">🎓 {{ t('growth.edu.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        <template v-if="status.discount_percent > 0">{{ t('growth.edu.descriptionWithDiscount', { zhe: zheLabel, percent: status.discount_percent }) }}</template>
        <template v-else>{{ t('growth.edu.description') }}</template>
      </p>
    </div>
    <div class="px-6 py-5">
      <div v-if="status.verification" class="flex items-start gap-3" data-testid="edu-verified">
        <div class="rounded-full bg-emerald-100 p-2 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400">✓</div>
        <div class="text-sm">
          <p class="font-medium text-gray-900 dark:text-white">{{ t('growth.edu.verified', { email: status.verification.email }) }}</p>
          <p v-if="status.discount_percent > 0" class="mt-1 text-emerald-600 dark:text-emerald-400">
            {{ t('growth.edu.verifiedPerk', { zhe: zheLabel }) }}
          </p>
        </div>
      </div>
      <form v-else class="space-y-3" @submit.prevent="verify">
        <div>
          <label class="input-label" for="edu-email">{{ t('growth.edu.emailLabel') }}</label>
          <div class="flex gap-2">
            <input
              id="edu-email"
              v-model.trim="email"
              type="email"
              autocomplete="email"
              class="input flex-1"
              :placeholder="t('growth.edu.emailPlaceholder', { suffix: status.suffixes[0] || 'edu.cn' })"
            />
            <button type="button" class="btn btn-secondary shrink-0" :disabled="!email || sending || cooldown > 0" data-testid="edu-send-code" @click="sendCode">
              {{ cooldown > 0 ? t('growth.edu.resendIn', { seconds: cooldown }) : t('growth.edu.sendCode') }}
            </button>
          </div>
          <p class="mt-1 text-xs text-gray-400">{{ t('growth.edu.suffixHint', { suffixes: status.suffixes.join('、') }) }}</p>
        </div>
        <div v-if="codeSent">
          <label class="input-label" for="edu-code">{{ t('growth.edu.codeLabel') }}</label>
          <div class="flex gap-2">
            <input id="edu-code" v-model.trim="code" inputmode="numeric" maxlength="6" class="input flex-1" autocomplete="one-time-code" />
            <button type="submit" class="btn btn-primary shrink-0" :disabled="code.length < 6 || verifying" data-testid="edu-verify">
              {{ t('growth.edu.verify') }}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { growthAPI, type EduStatus } from '@/api/growth'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'

const emit = defineEmits<{ verified: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const status = ref<EduStatus | null>(null)
const email = ref('')
const code = ref('')
const codeSent = ref(false)
const sending = ref(false)
const verifying = ref(false)
const cooldown = ref(0)
let timer: ReturnType<typeof setInterval> | undefined

// 20% off is written "8 折" in Chinese; 15% off → "8.5 折".
const zheLabel = computed(() => {
  const d = status.value?.discount_percent ?? 0
  return String(Math.round((100 - d) / 10 * 10) / 10)
})

async function load() {
  try {
    status.value = await growthAPI.getEduStatus()
  } catch {
    status.value = null
  }
}

function startCooldown(seconds = 60) {
  cooldown.value = seconds
  clearInterval(timer)
  timer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) clearInterval(timer)
  }, 1000)
}

async function sendCode() {
  if (!email.value || sending.value) return
  sending.value = true
  try {
    await growthAPI.sendEduCode(email.value)
    codeSent.value = true
    startCooldown()
    appStore.showSuccess(t('growth.edu.codeSent'))
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'growth.errors', t('common.error')))
  } finally {
    sending.value = false
  }
}

async function verify() {
  if (code.value.length < 6 || verifying.value) return
  verifying.value = true
  try {
    await growthAPI.verifyEdu(email.value, code.value)
    appStore.showSuccess(t('growth.edu.success'))
    await load()
    emit('verified')
  } catch (err) {
    appStore.showError(extractI18nErrorMessage(err, t, 'growth.errors', t('common.error')))
  } finally {
    verifying.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => clearInterval(timer))
</script>
