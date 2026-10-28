<template>
  <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
    <div
      v-for="a in items"
      :key="a.id"
      class="overflow-hidden rounded-lg border"
      :class="a.review_status === 'approved' ? 'border-emerald-500 ring-1 ring-emerald-500' : a.review_status === 'rejected' ? 'border-gray-200 opacity-50 dark:border-dark-600' : 'border-gray-200 dark:border-dark-600'"
      :data-testid="`materials-artwork-${a.id}`"
    >
      <a v-if="a.status === 'succeeded' && a.url" :href="a.url" target="_blank" rel="noopener">
        <img :src="a.thumb_url || a.url" class="w-full bg-gray-100 object-cover" :class="aspectClass(a.size)" loading="lazy" />
      </a>
      <div v-else class="flex w-full items-center justify-center bg-gray-50 p-2 text-center text-xs dark:bg-dark-800" :class="aspectClass(a.size)">
        <span :class="a.status === 'failed' ? 'text-red-500' : 'text-gray-500'">
          {{ t(`materials.artworkStatus.${a.status}`) }}<template v-if="a.error"> · {{ a.error }}</template>
        </span>
      </div>

      <div class="space-y-1.5 p-2 text-xs">
        <div v-if="showSource" class="truncate font-medium text-gray-700 dark:text-gray-200" :title="`${a.text_title} · ${a.segment_label}`">
          {{ a.text_title || '—' }} · {{ a.segment_label }}
        </div>
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-gray-500">
          <span>{{ a.style_name }}</span>
          <span :class="reviewClass(a.review_status)">{{ t(`materials.reviewStatus.${a.review_status}`) }}</span>
          <span :class="a.text_overlay_allowed ? 'text-emerald-600' : 'text-amber-600'">
            {{ a.text_overlay_allowed ? t('materials.artworks.overlayAllowed') : t('materials.artworks.overlayNo') }}
          </span>
        </div>
        <p v-if="a.review_status === 'rejected' && a.reject_reason" class="text-red-500">{{ a.reject_reason }}</p>

        <div v-if="a.status === 'succeeded'" class="flex items-center gap-0.5" :title="t('materials.artworks.rating')">
          <button
            v-for="n in 5"
            :key="n"
            class="text-base leading-none"
            :class="n <= a.rating ? 'text-amber-500' : 'text-gray-300 dark:text-dark-500'"
            :data-testid="`materials-rate-${a.id}-${n}`"
            @click="review(a, a.review_status, n === a.rating ? 0 : n)"
          >
            ★
          </button>
        </div>

        <div class="flex flex-wrap gap-x-2 gap-y-1">
          <template v-if="a.status === 'succeeded'">
            <button v-if="a.review_status !== 'approved'" class="font-medium text-emerald-600 hover:underline" :data-testid="`materials-approve-${a.id}`" @click="review(a, 'approved', a.rating)">
              {{ t('materials.artworks.approve') }}
            </button>
            <button v-if="a.review_status !== 'rejected'" class="text-red-600 hover:underline" :data-testid="`materials-reject-${a.id}`" @click="reject(a)">
              {{ t('materials.artworks.reject') }}
            </button>
            <button v-if="a.review_status !== 'pending'" class="text-gray-500 hover:underline" @click="review(a, 'pending', a.rating)">
              {{ t('materials.artworks.reset') }}
            </button>
          </template>
          <button v-if="a.status === 'succeeded' || a.status === 'failed'" class="text-primary-600 hover:underline" :data-testid="`materials-redraw-${a.id}`" @click="redraw(a)">
            {{ t('materials.artworks.redraw') }}
          </button>
          <button v-if="a.status === 'failed' || a.status === 'canceled'" class="text-primary-600 hover:underline" @click="retry(a)">
            {{ t('materials.artworks.retry') }}
          </button>
          <button v-if="a.status === 'queued'" class="text-gray-500 hover:underline" @click="cancel(a)">{{ t('materials.artworks.cancel') }}</button>
        </div>

        <div class="flex items-center justify-between text-gray-400">
          <span v-if="a.status === 'succeeded'">{{ t('materials.artworks.meta', { seconds: (a.duration_ms / 1000).toFixed(0), cost: a.cost_usd.toFixed(3) }) }}</span>
          <span v-else>#{{ a.id }}</span>
          <details class="relative">
            <summary class="cursor-pointer select-none hover:text-gray-600">{{ t('materials.artworks.openPrompt') }}</summary>
            <pre class="absolute bottom-5 right-0 z-10 max-h-72 w-80 overflow-auto whitespace-pre-wrap rounded-lg border border-gray-200 bg-white p-2 text-[11px] text-gray-700 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200">{{ a.prompt_final }}</pre>
          </details>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'

defineProps<{ items: api.MaterialArtwork[]; showSource?: boolean }>()
const emit = defineEmits<{ changed: []; queued: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

function aspectClass(size: string) {
  return size === '1536x1024' ? 'aspect-[3/2]' : size === '1024x1024' ? 'aspect-square' : 'aspect-[2/3]'
}

function reviewClass(status: api.MaterialReviewStatus) {
  return status === 'approved' ? 'text-emerald-600' : status === 'rejected' ? 'text-red-600' : ''
}

async function run(fn: () => Promise<unknown>, ok?: string, queued = false) {
  try {
    await fn()
    if (ok) appStore.showSuccess(ok)
    emit('changed')
    if (queued) emit('queued')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function review(a: api.MaterialArtwork, status: api.MaterialReviewStatus, rating: number, reason = '') {
  return run(() => api.reviewArtwork(a.id, { review_status: status, rating, reject_reason: reason }), t('materials.artworks.reviewed'))
}

function reject(a: api.MaterialArtwork) {
  const reason = window.prompt(t('materials.artworks.rejectReason'), a.reject_reason || '')
  if (reason === null) return
  return review(a, 'rejected', a.rating, reason)
}

function redraw(a: api.MaterialArtwork) {
  return run(() => api.redrawArtwork(a.id), t('materials.artworks.queuedOne'), true)
}

function retry(a: api.MaterialArtwork) {
  return run(() => api.retryArtwork(a.id), t('materials.artworks.queuedOne'), true)
}

function cancel(a: api.MaterialArtwork) {
  return run(() => api.cancelArtworks([a.id]), undefined, true)
}
</script>
