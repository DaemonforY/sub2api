<template>
  <div class="min-h-screen bg-gray-50 pb-24 dark:bg-dark-950 lg:pb-0">
    <PlazaNavBar :login-redirect="`/goods/${slug}`" />
    <main class="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <div v-if="loading" class="h-80 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>

      <div v-else-if="!goods" class="card flex flex-col items-center py-16 text-center">
        <span class="text-5xl">🛒</span>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">{{ errorText || t('goods.notFound') }}</p>
      </div>

      <div v-else class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]" data-testid="goods-detail">
        <div class="min-w-0 space-y-6">
          <div class="card p-6">
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ goods.name }}</h1>
            <p v-if="goods.subtitle" class="mt-2 text-gray-600 dark:text-dark-300">{{ goods.subtitle }}</p>
          </div>

          <section v-if="goods.deliveries.length" class="card space-y-3 p-6" data-testid="goods-mine">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('goods.mine') }}</h2>
            <GoodsKeyCard v-for="d in goods.deliveries" :key="d.order_id" :delivery="d" />
          </section>

          <section v-if="descriptionHtml" class="card p-6">
            <div class="goods-md" v-html="descriptionHtml"></div>
          </section>

          <section class="card p-6">
            <h2 class="mb-3 text-lg font-semibold text-gray-900 dark:text-white">{{ t('goods.faqTitle') }}</h2>
            <dl class="space-y-3 text-sm">
              <div v-for="key in ['delivery', 'where', 'refund']" :key="key">
                <dt class="font-medium text-gray-800 dark:text-gray-200">{{ t(`goods.faq.${key}.q`) }}</dt>
                <dd class="mt-1 text-gray-600 dark:text-dark-300">{{ t(`goods.faq.${key}.a`) }}</dd>
              </div>
            </dl>
          </section>
        </div>

        <aside class="hidden lg:block">
          <div class="card sticky top-20 space-y-4 p-6">
            <div class="flex items-baseline gap-2" data-testid="goods-price">
              <span class="text-3xl font-bold text-primary-600 dark:text-primary-400">¥{{ goods.price }}</span>
              <span v-if="strike" class="text-sm text-gray-400 line-through">¥{{ strike }}</span>
            </div>
            <p class="text-sm" :class="soldOut ? 'text-rose-600 dark:text-rose-400' : 'text-emerald-600 dark:text-emerald-400'" data-testid="goods-stock">{{ stockText }}</p>
            <button class="btn btn-primary w-full py-3" :disabled="!canBuy" data-testid="goods-buy" @click="buy">{{ buyLabel }}</button>
            <ul class="space-y-1.5 text-xs text-gray-500 dark:text-dark-400">
              <li>⚡ {{ t('goods.autoDelivery') }}</li>
              <li v-if="goods.per_user_limit">🔢 {{ t('goods.limit', { n: goods.per_user_limit }) }}</li>
            </ul>
          </div>
        </aside>
      </div>
    </main>

    <!-- Mobile buy bar -->
    <div v-if="goods" class="fixed inset-x-0 bottom-0 z-20 flex items-center justify-between gap-3 border-t border-gray-200 bg-white/95 px-4 py-3 backdrop-blur dark:border-dark-700 dark:bg-dark-900/95 lg:hidden">
      <span class="flex flex-col">
        <span class="flex items-baseline gap-1.5">
          <span class="text-xl font-bold text-primary-600 dark:text-primary-400">¥{{ goods.price }}</span>
          <span v-if="strike" class="text-xs text-gray-400 line-through">¥{{ strike }}</span>
        </span>
        <span class="text-xs" :class="soldOut ? 'text-rose-500' : 'text-gray-500 dark:text-dark-400'">{{ stockText }}</span>
      </span>
      <button class="btn btn-primary px-6" :disabled="!canBuy" @click="buy">{{ buyLabel }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import GoodsKeyCard from '@/components/goods/GoodsKeyCard.vue'
import { getGoods, type GoodsProduct } from '@/api/goods'
import { useAuthStore } from '@/stores/auth'
import { renderMarkdown } from '@/utils/markdown'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const slug = computed(() => String(route.params.slug || ''))
const goods = ref<GoodsProduct | null>(null)
const loading = ref(true)
const errorText = ref('')

const descriptionHtml = computed(() => renderMarkdown(goods.value?.description))
const strike = computed(() => {
  const g = goods.value
  return g && g.original_price > g.price ? g.original_price : 0
})
const soldOut = computed(() => (goods.value?.stock ?? 0) <= 0)
const limitReached = computed(() => {
  const g = goods.value
  return !!g && g.per_user_limit > 0 && g.bought >= g.per_user_limit
})
const canBuy = computed(() => !!goods.value && !soldOut.value && !limitReached.value)
const stockText = computed(() => {
  const n = goods.value?.stock ?? 0
  if (n <= 0) return t('goods.soldOut')
  return n <= 10 ? t('goods.stockLow', { n }) : t('goods.inStock')
})
const buyLabel = computed(() => {
  if (soldOut.value) return t('goods.soldOutButton')
  if (limitReached.value) return t('goods.limitReached')
  return t('goods.buy')
})

function buy() {
  if (!goods.value || !canBuy.value) return
  const target = `/purchase?goods=${encodeURIComponent(goods.value.slug)}`
  if (!authStore.isAuthenticated) {
    router.push({ path: '/login', query: { redirect: target } })
    return
  }
  router.push(target)
}

async function load() {
  loading.value = true
  errorText.value = ''
  try {
    goods.value = await getGoods(slug.value)
    document.title = goods.value.name
  } catch (err: unknown) {
    goods.value = null
    errorText.value = extractApiErrorMessage(err, t('goods.notFound'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(slug, load)
</script>

<style scoped>
.goods-md :deep(h1),
.goods-md :deep(h2),
.goods-md :deep(h3) {
  @apply mb-2 mt-4 font-semibold text-gray-900 first:mt-0 dark:text-white;
}
.goods-md :deep(h2) {
  @apply text-base;
}
.goods-md :deep(p),
.goods-md :deep(li) {
  @apply text-sm leading-7 text-gray-700 dark:text-gray-300;
}
.goods-md :deep(ul) {
  @apply my-2 list-disc pl-5;
}
.goods-md :deep(ol) {
  @apply my-2 list-decimal pl-5;
}
.goods-md :deep(img) {
  @apply my-3 max-w-full rounded-lg;
}
.goods-md :deep(a) {
  @apply text-primary-600 underline dark:text-primary-400;
}
.goods-md :deep(code) {
  @apply rounded bg-gray-100 px-1 text-xs dark:bg-dark-700;
}
</style>
