<template>
  <div class="min-w-[9.5rem]">
    <div v-if="props.loading && !props.stats" class="h-3 w-36 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    <div v-else-if="props.error && !props.stats" class="text-xs text-red-500">{{ props.error }}</div>
    <div v-else-if="props.stats" class="space-y-1">
      <div class="flex gap-[2px]" data-testid="recent-requests-grid">
        <span
          v-for="(bucket, index) in props.stats.buckets"
          :key="index"
          class="h-3 w-[5px] rounded-[1px]"
          :class="bucketClass(bucket)"
          :title="bucketTitle(bucket, index)"
        ></span>
      </div>
      <div v-if="total > 0" class="flex items-center gap-1.5 text-[11px] leading-4">
        <span class="text-emerald-600 dark:text-emerald-400">{{ t('admin.accounts.recentRequests.success') }} {{ props.stats.success }}</span>
        <span :class="props.stats.failure > 0 ? 'text-red-500' : 'text-gray-400'">{{ t('admin.accounts.recentRequests.failure') }} {{ props.stats.failure }}</span>
        <span class="font-medium" :class="successRateClass">{{ successRate }}%</span>
      </div>
      <div v-else class="text-[11px] leading-4 text-gray-400">{{ t('admin.accounts.recentRequests.empty') }}</div>
    </div>
    <div v-else class="text-xs text-gray-400">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountRecentRequests, AccountRequestBucket } from '@/types'

const props = defineProps<{
  stats: AccountRecentRequests | null
  windowStart: string | null
  bucketSeconds: number
  loading?: boolean
  error?: string | null
}>()

const { t } = useI18n()

const total = computed(() => (props.stats ? props.stats.success + props.stats.failure : 0))

// 失败次数可能含同一请求之后重试成功的，所以成功率按「成功 / (成功 + 失败)」算，只作参考。
const successRate = computed(() => {
  if (!props.stats || total.value === 0) return 0
  return Math.round((props.stats.success / total.value) * 1000) / 10
})

const successRateClass = computed(() => {
  if (successRate.value >= 95) return 'text-emerald-600 dark:text-emerald-400'
  if (successRate.value >= 80) return 'text-amber-500'
  return 'text-red-500'
})

const bucketClass = (bucket: AccountRequestBucket) => {
  if (bucket.f > 0 && bucket.s === 0) return 'bg-red-500'
  if (bucket.f > 0) return 'bg-amber-400'
  if (bucket.s > 0) return 'bg-emerald-500'
  return 'bg-gray-200 dark:bg-dark-600'
}

const bucketTitle = (bucket: AccountRequestBucket, index: number) => {
  let time = ''
  if (props.windowStart) {
    const start = new Date(new Date(props.windowStart).getTime() + index * props.bucketSeconds * 1000)
    time = start.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return t('admin.accounts.recentRequests.cellTitle', { time, s: bucket.s, f: bucket.f })
}
</script>
