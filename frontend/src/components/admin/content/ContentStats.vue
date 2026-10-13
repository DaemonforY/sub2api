<template>
  <div class="space-y-4" data-testid="content-stats">
    <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.content.stats.hint') }}</p>
      <div class="flex gap-2">
        <button
          v-for="d in DAYS"
          :key="d"
          type="button"
          :class="['btn btn-sm', days === d ? 'btn-primary' : 'btn-secondary']"
          @click="days = d"
        >{{ d === 0 ? t('admin.content.stats.allTime') : t('admin.content.stats.days', { n: d }) }}</button>
      </div>
    </div>

    <div v-if="stats?.platforms.length" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <div v-for="p in stats.platforms" :key="p.platform" class="card p-4">
        <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.content.platform.${p.platform}`) }}
          <span class="text-xs font-normal text-gray-400">{{ p.posts }} {{ t('admin.content.stats.posts') }}</span>
        </div>
        <dl class="mt-2 grid grid-cols-2 gap-1 text-xs text-gray-500">
          <dt>{{ t('admin.content.stats.clicks') }}</dt><dd class="text-right text-gray-900 dark:text-white">{{ p.clicks }}</dd>
          <dt>{{ t('admin.content.stats.signups') }}</dt><dd class="text-right text-gray-900 dark:text-white">{{ p.signups }}</dd>
          <dt>{{ t('admin.content.stats.promoUses') }}</dt><dd class="text-right text-gray-900 dark:text-white">{{ p.promo_uses }}</dd>
          <dt>{{ t('admin.content.stats.paid') }}</dt><dd class="text-right text-gray-900 dark:text-white">{{ p.paid_users }}</dd>
          <dt>{{ t('admin.content.stats.revenue') }}</dt><dd class="text-right text-gray-900 dark:text-white">{{ p.revenue }}</dd>
        </dl>
      </div>
    </div>

    <p v-if="stats && !stats.rows.length" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.content.stats.empty') }}</p>
    <div v-if="stats?.rows.length" class="card overflow-x-auto p-0">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800">
          <tr>
            <th class="px-3 py-2 text-left">{{ t('admin.content.stats.post') }}</th>
            <th class="px-3 py-2 text-left">{{ t('admin.content.stats.platform') }}</th>
            <th class="px-3 py-2 text-left">{{ t('admin.content.stats.published') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.clicks') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.visitors') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.signups') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.promoUses') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.paid') }}</th>
            <th class="px-3 py-2 text-right">{{ t('admin.content.stats.revenue') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-for="r in stats.rows" :key="r.id">
            <td class="max-w-xs px-3 py-2">
              <a :href="r.post_url" target="_blank" rel="noopener" class="block truncate text-primary-600 hover:underline">{{ r.title }}</a>
              <button type="button" class="block truncate text-xs text-gray-400 hover:underline" @click="emit('open-item', r.item_id)">{{ r.item_title }}</button>
            </td>
            <td class="px-3 py-2">{{ t(`admin.content.platform.${r.platform}`) }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-xs text-gray-500">{{ r.published_at ? r.published_at.slice(0, 10) : '' }}</td>
            <td class="px-3 py-2 text-right">{{ r.link_code ? r.clicks : '—' }}</td>
            <td class="px-3 py-2 text-right">{{ r.link_code ? r.visitors : '—' }}</td>
            <td class="px-3 py-2 text-right">{{ r.link_code ? r.signups : '—' }}</td>
            <td class="px-3 py-2 text-right">{{ r.promo_code ? r.promo_uses : '—' }}</td>
            <td class="px-3 py-2 text-right">{{ r.paid_users }}</td>
            <td class="px-3 py-2 text-right">{{ r.revenue }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentStats } from '@/api/admin/content'

const emit = defineEmits<{ (e: 'open-item', id: number): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const DAYS = [7, 30, 90, 0]
const days = ref(30)
const stats = ref<ContentStats | null>(null)

async function load() {
  try {
    stats.value = await contentAPI.getStats(days.value)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
watch(days, load)
onMounted(load)
</script>
