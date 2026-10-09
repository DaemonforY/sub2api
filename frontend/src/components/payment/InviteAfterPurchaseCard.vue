<template>
  <div v-if="detail" class="rounded-xl bg-gradient-to-r from-primary-50 to-violet-50 p-5 dark:from-primary-900/20 dark:to-violet-900/20" data-testid="invite-after-purchase">
    <p class="text-sm font-semibold text-gray-900 dark:text-white">🎁 {{ t('payment.result.invite.title') }}</p>
    <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-dark-300">
      {{ rate > 0 ? t('payment.result.invite.descRate', { rate: rateText }) : t('payment.result.invite.desc') }}
    </p>
    <div class="mt-3 flex gap-2">
      <button type="button" class="btn btn-primary btn-sm flex-1" data-testid="invite-after-purchase-copy" @click="copyLink">
        {{ t('payment.result.invite.copy') }}
      </button>
      <router-link to="/affiliate" class="btn btn-secondary btn-sm flex-1">{{ t('payment.result.invite.more') }}</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
// Right after a successful payment, invite friends: the moment people are happiest with the product.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useClipboard } from '@/composables/useClipboard'
import { track } from '@/utils/analytics'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const detail = ref<UserAffiliateDetail | null>(null)

const rate = computed(() => detail.value?.effective_rebate_rate_percent ?? 0)
const rateText = computed(() => String(Math.round(rate.value * 100) / 100))
const link = computed(() =>
  detail.value ? `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}` : ''
)

async function copyLink(): Promise<void> {
  track('invite_link_copy', { where: 'payment_result' })
  await copyToClipboard(link.value, t('affiliate.linkCopied'))
}

onMounted(async () => {
  try {
    const d = await userAPI.getAffiliateDetail()
    if (d?.aff_code) detail.value = d
  } catch {
    // no card when the program is off or the request fails
  }
})
</script>
