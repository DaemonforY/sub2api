<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar login-redirect="/courses" />
    <main class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div class="mb-8 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">{{ t('courses.title') }}</h1>
          <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ t('courses.subtitle') }}</p>
        </div>
        <RouterLink v-if="isAuthenticated" to="/my-courses" class="btn btn-secondary btn-sm">{{ t('courses.mine.title') }}</RouterLink>
      </div>

      <div v-if="categories.length > 1" class="mb-6 flex flex-wrap gap-2">
        <button
          v-for="c in ['', ...categories]"
          :key="c || 'all'"
          type="button"
          :class="['rounded-full border px-3 py-1 text-sm transition', category === c ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 text-gray-600 hover:border-gray-400 dark:border-dark-600 dark:text-dark-300']"
          @click="category = c"
        >
          {{ c || t('courses.allCategories') }}
        </button>
      </div>

      <div v-if="loading" class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="i in 3" :key="i" class="h-80 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      </div>

      <div v-else-if="shown.length === 0" class="card flex flex-col items-center py-16 text-center">
        <span class="text-5xl">📚</span>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">{{ loadFailed ? t('courses.loadFailed') : t('courses.empty') }}</p>
      </div>

      <div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3" data-testid="course-list">
        <RouterLink
          v-for="c in shown"
          :key="c.id"
          :to="`/courses/${c.slug}`"
          class="group flex flex-col overflow-hidden rounded-2xl border border-gray-200/70 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-800"
          data-testid="course-card"
        >
          <div class="relative aspect-video overflow-hidden bg-gradient-to-br from-primary-500 via-violet-500 to-teal-400">
            <img v-if="c.cover_url" :src="c.cover_url" alt="" class="h-full w-full object-cover transition duration-300 group-hover:scale-[1.03]" />
            <span v-else class="absolute inset-0 flex items-center justify-center text-6xl opacity-90">📘</span>
            <span v-if="c.owned" class="absolute left-3 top-3 rounded-full bg-emerald-500 px-2.5 py-0.5 text-xs font-medium text-white">{{ t('courses.owned') }}</span>
            <span v-else-if="c.sale_active" class="absolute left-3 top-3 rounded-full bg-rose-500 px-2.5 py-0.5 text-xs font-medium text-white">{{ t('courses.saleBadge') }}</span>
            <span v-if="c.category" class="absolute right-3 top-3 rounded-full bg-black/50 px-2.5 py-0.5 text-xs text-white">{{ c.category }}</span>
          </div>
          <div class="flex flex-1 flex-col p-5">
            <h2 class="line-clamp-2 text-base font-semibold text-gray-900 dark:text-white">{{ c.title }}</h2>
            <p v-if="c.subtitle" class="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-dark-300">{{ c.subtitle }}</p>
            <div class="mt-auto flex items-end justify-between pt-4">
              <span class="text-xs text-gray-400 dark:text-dark-400">
                {{ t('courses.lessons', { count: c.lesson_count }) }}<template v-if="c.student_count"> · {{ t('courses.students', { count: c.student_count }) }}</template>
              </span>
              <span class="flex items-baseline gap-1.5">
                <span v-if="c.edu_applied" class="text-xs text-emerald-600 dark:text-emerald-400">{{ t('courses.eduPrice') }}</span>
                <span v-if="strikePrice(c)" class="text-xs text-gray-400 line-through">¥{{ strikePrice(c) }}</span>
                <span class="text-lg font-bold text-primary-600 dark:text-primary-400">¥{{ c.current_price }}</span>
              </span>
            </div>
          </div>
        </RouterLink>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { listCourses, strikePrice, type Course } from '@/api/courses'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const courses = ref<Course[]>([])
const loading = ref(true)
const loadFailed = ref(false)
const category = ref('')

const categories = computed(() => [...new Set(courses.value.map(c => c.category).filter(Boolean))])
const shown = computed(() => (category.value ? courses.value.filter(c => c.category === category.value) : courses.value))

onMounted(async () => {
  try {
    courses.value = await listCourses()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})
</script>
