<template>
  <div class="rounded-xl border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-800/60 dark:bg-emerald-900/10" data-testid="goods-key">
    <div class="flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-dark-400">
      <span>{{ delivery.goods_name }} · {{ t('goods.orderNo', { id: delivery.order_id }) }}</span>
      <span>{{ deliveredText }}</span>
    </div>
    <div class="mt-2 flex items-start gap-2">
      <code class="min-w-0 flex-1 select-all break-all rounded-lg bg-white px-3 py-2 font-mono text-sm text-gray-900 dark:bg-dark-800 dark:text-white" data-testid="goods-key-content">{{ delivery.content }}</code>
      <button type="button" class="btn btn-primary btn-sm shrink-0" data-testid="goods-key-copy" @click="copyToClipboard(delivery.content, t('goods.keyCopied'))">
        {{ copied ? t('common.copied') : t('goods.copyKey') }}
      </button>
    </div>
    <div v-if="usageHtml" class="goods-md mt-3 border-t border-emerald-200/70 pt-3 dark:border-emerald-800/50">
      <p class="mb-1 text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('goods.usageTitle') }}</p>
      <div v-html="usageHtml"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GoodsDelivery } from '@/api/goods'
import { useClipboard } from '@/composables/useClipboard'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ delivery: GoodsDelivery }>()
const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const usageHtml = computed(() => renderMarkdown(props.delivery.usage_note))
const deliveredText = computed(() => {
  const d = new Date(props.delivery.delivered_at)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString()
})
</script>

<style scoped>
.goods-md :deep(p),
.goods-md :deep(li) {
  @apply text-sm leading-6 text-gray-700 dark:text-gray-300;
}
.goods-md :deep(ul) {
  @apply my-1 list-disc pl-5;
}
.goods-md :deep(ol) {
  @apply my-1 list-decimal pl-5;
}
.goods-md :deep(a) {
  @apply text-primary-600 underline dark:text-primary-400;
}
.goods-md :deep(code) {
  @apply rounded bg-white px-1 text-xs dark:bg-dark-700;
}
</style>
