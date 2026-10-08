<template>
  <AppLayout>
    <div class="space-y-4">
      <div v-if="loading" class="card py-16 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>

      <template v-else-if="home">
        <!-- Not a creator yet -->
        <template v-if="!creator || creator.status === 'rejected'">
          <div class="card space-y-3 p-6" data-testid="creator-intro">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('creator.intro.title') }}</h2>
            <ol class="list-decimal space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-300">
              <li>{{ t('creator.intro.step1') }}</li>
              <li>{{ t('creator.intro.step2') }}</li>
              <li>{{ t('creator.intro.step3', { percent: home.commission_percent }) }}</li>
              <li>{{ t('creator.intro.step4', { days: home.settle_days, min: home.withdraw_min_cny }) }}</li>
            </ol>
          </div>
          <div v-if="!home.enabled" class="card p-6 text-sm text-gray-500" data-testid="creator-closed">{{ t('creator.closed') }}</div>
          <div v-else class="card space-y-3 p-6" data-testid="creator-apply">
            <p v-if="creator?.status === 'rejected'" class="rounded-lg bg-rose-50 p-3 text-sm text-rose-700 dark:bg-rose-900/20 dark:text-rose-300">
              {{ t('creator.rejected') }}<template v-if="creator.admin_note">{{ creator.admin_note }}</template>
            </p>
            <h3 class="text-sm font-medium">{{ t('creator.apply.title') }}</h3>
            <label class="block text-sm">
              <span class="input-label">{{ t('creator.apply.displayName') }}</span>
              <input v-model="application.display_name" class="input" maxlength="40" data-testid="creator-apply-name" />
              <span class="input-hint">{{ t('creator.apply.displayNameHint') }}</span>
            </label>
            <label class="block text-sm">
              <span class="input-label">{{ t('creator.apply.contact') }}</span>
              <input v-model="application.contact" class="input" maxlength="100" data-testid="creator-apply-contact" />
            </label>
            <label class="block text-sm">
              <span class="input-label">{{ t('creator.apply.bio') }}</span>
              <textarea v-model="application.bio" rows="3" class="input" maxlength="1000"></textarea>
            </label>
            <label class="block text-sm">
              <span class="input-label">{{ t('creator.apply.plan') }}</span>
              <textarea v-model="application.plan" rows="4" class="input" maxlength="2000" :placeholder="t('creator.apply.planPlaceholder')" data-testid="creator-apply-plan"></textarea>
            </label>
            <button
              class="btn btn-primary"
              :disabled="busy || !application.display_name || !application.contact || !application.plan"
              data-testid="creator-apply-submit"
              @click="apply"
            >
              {{ t('creator.apply.submit') }}
            </button>
          </div>
        </template>

        <div v-else-if="creator.status === 'pending'" class="card space-y-2 p-6" data-testid="creator-pending">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('creator.pending.title') }}</h2>
          <p class="text-sm text-gray-500">{{ t('creator.pending.body', { name: creator.display_name }) }}</p>
        </div>

        <!-- Creator -->
        <template v-else>
          <p v-if="creator.status === 'suspended'" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-200" data-testid="creator-suspended">
            {{ t('creator.suspended') }}<template v-if="creator.admin_note">{{ creator.admin_note }}</template>
          </p>
          <div class="grid gap-3 sm:grid-cols-4" data-testid="creator-balance">
            <div v-for="stat in stats" :key="stat.key" class="card p-4">
              <div class="text-xs text-gray-500">{{ t(`creator.balance.${stat.key}`) }}</div>
              <div :class="['mt-1 text-xl font-semibold', stat.value < 0 ? 'text-rose-600' : 'text-gray-900 dark:text-white']">¥{{ stat.value.toFixed(2) }}</div>
            </div>
          </div>
          <p class="text-xs text-gray-500">{{ t('creator.rules', { percent: home.commission_percent, days: home.settle_days }) }}</p>

          <div v-if="editing === undefined" class="flex gap-1">
            <button v-for="tab in ['courses', 'earnings'] as const" :key="tab" :class="['btn btn-sm', panel === tab ? 'btn-primary' : 'btn-secondary']" :data-testid="`creator-tab-${tab}`" @click="panel = tab">
              {{ t(`creator.tabs.${tab}`) }}
            </button>
          </div>

          <CreatorCourseEditor
            v-if="editing !== undefined"
            :key="editing ?? 'new'"
            :course-id="editing"
            :commission-percent="home.commission_percent"
            @close="closeEditor"
            @saved="loadCourses"
          />

          <template v-else-if="panel === 'courses'">
            <div class="flex items-center justify-between gap-2">
              <p class="text-sm text-gray-500">{{ t('creator.coursesHint', { max: home.max_courses }) }}</p>
              <button v-if="creator.status === 'approved'" class="btn btn-primary" :disabled="courses.length >= home.max_courses" data-testid="creator-new" @click="editing = null">{{ t('creator.newCourse') }}</button>
            </div>
            <div v-if="!courses.length" class="card py-12 text-center text-sm text-gray-500">{{ t('creator.noCourses') }}</div>
            <div v-else class="card divide-y divide-gray-100 dark:divide-dark-700" data-testid="creator-courses">
              <div v-for="c in courses" :key="c.id" class="flex flex-wrap items-center gap-3 p-4">
                <img v-if="c.cover_url" :src="c.cover_url" alt="" class="h-12 w-20 rounded object-cover" />
                <div class="min-w-0 flex-1">
                  <div class="font-medium text-gray-900 dark:text-white">{{ c.title }}</div>
                  <div class="text-xs text-gray-500">¥{{ c.price }} · {{ t('creator.studentCount', { count: c.student_count }) }}</div>
                </div>
                <span :class="['badge', c.status === 'published' ? 'badge-success' : 'badge-gray']">{{ t(`creator.status.${c.status}`) }}</span>
                <span v-if="c.review_status && c.review_status !== 'approved'" :class="['badge', reviewBadge(c.review_status)]">{{ t(`creator.review.${c.review_status}`) }}</span>
                <button class="btn btn-secondary btn-sm" @click="editing = c.id">{{ t('common.edit') }}</button>
              </div>
            </div>
          </template>

          <CreatorEarnings v-else :home="home" @changed="loadHome" />
        </template>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import creatorAPI, { type CreatorApplication, type CreatorHome } from '@/api/creator'
import type { Course } from '@/api/courses'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import CreatorCourseEditor from './creator/CreatorCourseEditor.vue'
import CreatorEarnings from './creator/CreatorEarnings.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const busy = ref(false)
const home = ref<CreatorHome | null>(null)
const courses = ref<Course[]>([])
const panel = ref<'courses' | 'earnings'>('courses')
// undefined: the list; null: a new course; a number: that course.
const editing = ref<number | null | undefined>(undefined)
const application = reactive<CreatorApplication>({ display_name: '', bio: '', contact: '', plan: '' })

const creator = computed(() => home.value?.creator || null)
const stats = computed(() => {
  const b = home.value?.balance
  return [
    { key: 'available', value: b?.available || 0 },
    { key: 'frozen', value: b?.frozen || 0 },
    { key: 'paid', value: (b?.paid || 0) + (b?.pending || 0) },
    { key: 'net', value: b?.net || 0 },
  ]
})

function fail(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

function reviewBadge(status: string) {
  return status === 'pending' ? 'badge-primary' : status === 'rejected' ? 'badge-danger' : 'badge-warning'
}

async function loadHome() {
  try {
    home.value = await creatorAPI.home()
    const c = home.value.creator
    if (c && c.status === 'rejected') Object.assign(application, { display_name: c.display_name, bio: c.bio, contact: c.contact, plan: c.plan })
  } catch (err) {
    fail(err)
  }
}

async function loadCourses() {
  if (!creator.value || (creator.value.status !== 'approved' && creator.value.status !== 'suspended')) return
  try {
    courses.value = await creatorAPI.courses()
  } catch (err) {
    fail(err)
  }
}

function closeEditor() {
  editing.value = undefined
  void loadCourses()
}

async function apply() {
  busy.value = true
  try {
    await creatorAPI.apply({ ...application })
    appStore.showSuccess(t('creator.apply.sent'))
    await loadHome()
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

onMounted(async () => {
  await loadHome()
  await loadCourses()
  loading.value = false
})
</script>
