<template>
  <div class="space-y-4">
    <div class="card flex flex-wrap items-center gap-2 p-3">
      <select v-model="status" class="input w-36 text-sm" data-testid="materials-artworks-status" @change="reload">
        <option value="">{{ t('materials.artworks.allStatuses') }}</option>
        <option v-for="s in statuses" :key="s" :value="s">{{ t(`materials.artworkStatus.${s}`) }}</option>
      </select>
      <select v-model="review" class="input w-36 text-sm" data-testid="materials-artworks-review" @change="reload">
        <option value="">{{ t('materials.artworks.allReviews') }}</option>
        <option v-for="r in reviews" :key="r" :value="r">{{ t(`materials.reviewStatus.${r}`) }}</option>
      </select>
      <span class="text-xs text-gray-500">{{ t('materials.artworks.total', { n: total }) }}</span>
      <button class="btn btn-secondary btn-sm ml-auto" @click="reload">{{ t('materials.refresh') }}</button>
    </div>
    <div v-if="!items.length && !loading" class="card p-10 text-center text-sm text-gray-400">{{ t('materials.artworks.empty') }}</div>
    <div v-else class="card p-4">
      <MaterialArtworkGrid :items="items" show-source @changed="reload" @queued="emit('queued')" />
    </div>
    <button v-if="items.length < total" class="btn btn-secondary btn-sm w-full" :disabled="loading" @click="more">…</button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'
import MaterialArtworkGrid from './MaterialArtworkGrid.vue'

const props = defineProps<{ tick: number }>()
const emit = defineEmits<{ queued: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const statuses = ['queued', 'running', 'succeeded', 'failed', 'canceled'] as const
const reviews = ['pending', 'approved', 'rejected'] as const
const PAGE = 60

const status = ref('')
const review = ref('')
const items = ref<api.MaterialArtwork[]>([])
const total = ref(0)
const loading = ref(false)

async function fetchPage(offset: number) {
  loading.value = true
  try {
    const page = await api.listArtworks({ status: status.value || undefined, review_status: review.value || undefined, limit: PAGE, offset })
    items.value = offset ? [...items.value, ...page.items] : page.items
    total.value = page.total
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}
const reload = () => fetchPage(0)
const more = () => fetchPage(items.value.length)

watch(() => props.tick, reload)
onMounted(reload)
</script>
