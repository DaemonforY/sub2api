<template>
  <AppLayout>
    <div class="space-y-4">
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 2" :key="i" class="h-32 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      </div>

      <div v-else-if="items.length === 0" class="card flex flex-col items-center py-16 text-center">
        <span class="text-5xl">📚</span>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">{{ t('courses.mine.empty') }}</p>
        <RouterLink to="/courses" class="btn btn-primary mt-4">{{ t('courses.browse') }}</RouterLink>
      </div>

      <div v-for="item in items" :key="item.course.id" class="card overflow-hidden" data-testid="my-course">
        <div class="flex flex-col gap-4 p-5 sm:flex-row">
          <RouterLink :to="`/courses/${item.course.slug}`" class="block shrink-0">
            <img v-if="item.course.cover_url" :src="item.course.cover_url" alt="" class="h-24 w-full rounded-lg object-cover sm:w-40" />
            <div v-else class="flex h-24 w-full items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-teal-400 text-4xl sm:w-40">📘</div>
          </RouterLink>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <RouterLink :to="`/courses/${item.course.slug}`" class="truncate text-base font-semibold text-gray-900 hover:underline dark:text-white">{{ item.course.title }}</RouterLink>
              <span v-if="item.enrollment.refunded" class="badge badge-gray">{{ t('courses.mine.refunded') }}</span>
              <span v-else-if="item.updated" class="badge badge-warning" data-testid="my-course-updated">{{ t('courses.mine.updated') }}</span>
            </div>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('courses.mine.since', { date: formatDate(item.enrollment.created_at) }) }} · {{ t('courses.lessons', { count: item.course.lesson_count }) }}
            </p>
            <p v-if="item.enrollment.refunded" class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('courses.mine.refundedHint') }}</p>
            <button
              v-else-if="!deliveries[item.course.id]"
              class="btn btn-primary btn-sm mt-3"
              :disabled="fetching === item.course.id"
              data-testid="my-course-get"
              @click="fetchDelivery(item)"
            >
              {{ fetching === item.course.id ? t('common.processing') : t('courses.mine.get') }}
            </button>
          </div>
        </div>

        <div v-if="deliveries[item.course.id]" class="space-y-3 border-t border-gray-100 bg-gray-50/60 p-5 text-sm dark:border-dark-700 dark:bg-dark-900/40" data-testid="my-course-delivery">
          <div class="flex flex-wrap items-center gap-2">
            <span class="w-20 shrink-0 text-gray-500 dark:text-dark-400">{{ t('courses.mine.link') }}</span>
            <a :href="deliveries[item.course.id].link" target="_blank" rel="noopener noreferrer" class="min-w-0 flex-1 truncate font-mono text-primary-600 dark:text-primary-400">{{ deliveries[item.course.id].link }}</a>
            <button class="btn btn-secondary btn-sm" @click="copy(deliveries[item.course.id].link)">{{ t('courses.mine.copy') }}</button>
            <a :href="deliveries[item.course.id].link" target="_blank" rel="noopener noreferrer" class="btn btn-primary btn-sm">{{ t('courses.mine.open') }}</a>
          </div>
          <div v-if="deliveries[item.course.id].code" class="flex items-center gap-2">
            <span class="w-20 shrink-0 text-gray-500 dark:text-dark-400">{{ t('courses.mine.code') }}</span>
            <span class="font-mono text-base font-semibold text-gray-900 dark:text-white">{{ deliveries[item.course.id].code }}</span>
            <button class="btn btn-secondary btn-sm" @click="copy(deliveries[item.course.id].code)">{{ t('courses.mine.copy') }}</button>
          </div>
          <div v-if="deliveries[item.course.id].password" class="flex items-center gap-2">
            <span class="w-20 shrink-0 text-gray-500 dark:text-dark-400">{{ t('courses.mine.password') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ deliveries[item.course.id].password }}</span>
            <button class="btn btn-secondary btn-sm" @click="copy(deliveries[item.course.id].password)">{{ t('courses.mine.copy') }}</button>
          </div>
          <p v-if="deliveries[item.course.id].note" class="whitespace-pre-wrap rounded-lg bg-white p-3 text-gray-700 dark:bg-dark-800 dark:text-gray-300">{{ deliveries[item.course.id].note }}</p>
          <p class="text-xs text-gray-400 dark:text-dark-500">
            {{ t('courses.mine.updatedAt', { date: formatDate(deliveries[item.course.id].updated_at) }) }} · {{ t('courses.mine.keepPrivate') }}
          </p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getDelivery, myCourses, type CourseDelivery, type MyCourse } from '@/api/courses'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const items = ref<MyCourse[]>([])
const loading = ref(true)
const fetching = ref(0)
const deliveries = reactive<Record<number, CourseDelivery>>({})

function formatDate(value: string): string {
  return value ? new Date(value).toLocaleDateString() : ''
}

async function copy(text: string) {
  await copyToClipboard(text, t('courses.mine.copied'))
}

async function fetchDelivery(item: MyCourse) {
  fetching.value = item.course.id
  try {
    deliveries[item.course.id] = await getDelivery(item.course.id)
    item.updated = false
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('courses.mine.getFailed')))
  } finally {
    fetching.value = 0
  }
}

onMounted(async () => {
  try {
    items.value = await myCourses()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('courses.loadFailed')))
  } finally {
    loading.value = false
  }
})
</script>
