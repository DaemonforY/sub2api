<template>
  <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]" data-testid="desk-final">
    <div class="card p-5">
      <div class="prose max-w-none dark:prose-invert" v-html="html" />
    </div>
    <div class="space-y-3">
      <div class="card space-y-3 p-4">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.final.title') }}</h3>
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.desk.final.hint') }}</p>
        <button class="btn btn-primary w-full" :disabled="busy" data-testid="desk-platforms" @click="platforms">{{ t('admin.desk.final.platforms') }}</button>
        <button class="btn btn-secondary w-full" :disabled="busy" @click="emit('stage', 'edit')">{{ t('admin.desk.final.backEdit') }}</button>
      </div>
      <div class="card space-y-2 p-4" data-testid="desk-learn">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.final.learn') }}</h4>
        <p class="text-xs text-gray-500">{{ t('admin.desk.final.learnHint') }}</p>
        <p v-if="busy && desk.job?.kind === 'learn'" class="animate-pulse text-sm text-gray-500">{{ t('admin.desk.job.learn') }}…</p>
        <p v-else-if="!desk.learn.length" class="text-xs text-gray-400">{{ t('admin.desk.final.noLearn') }}</p>
        <div v-for="l in desk.learn" :key="l.id" class="space-y-1 rounded-lg border border-gray-100 p-2 text-sm dark:border-dark-700">
          <div class="text-gray-800 dark:text-dark-100">{{ l.rule }}</div>
          <div class="text-xs text-gray-500">{{ l.evidence }}</div>
          <div v-if="l.status === 'new'" class="flex gap-3 text-xs">
            <button class="text-primary-600 hover:underline" @click="learn(l.id, true)">{{ t('admin.desk.final.addRule') }}</button>
            <button class="text-gray-400 hover:underline" @click="learn(l.id, false)">{{ t('admin.desk.final.dismiss') }}</button>
          </div>
          <div v-else class="text-xs text-gray-400">{{ l.status === 'added' ? t('admin.desk.final.added') : t('admin.desk.final.dismissed') }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, DeskStage } from '@/api/admin/content'

const props = defineProps<{ item: ContentItem; busy: boolean }>()
const emit = defineEmits<{ (e: 'updated', it: ContentItem): void; (e: 'stage', s: DeskStage): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const desk = computed(() => props.item.desk!)
const html = computed(() => renderMarkdown(desk.value.body))

async function call(fn: () => Promise<ContentItem>) {
  try {
    emit('updated', await fn())
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
const platforms = () => call(() => contentAPI.makePlatforms(props.item.id))
const learn = (id: string, accept: boolean) => call(() => contentAPI.applyLearn(props.item.id, id, accept))
</script>
