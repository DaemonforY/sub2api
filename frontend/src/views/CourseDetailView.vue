<template>
  <div class="min-h-screen bg-gray-50 pb-24 dark:bg-dark-950 lg:pb-0">
    <PlazaNavBar :login-redirect="`/courses/${slug}`" />
    <main class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <RouterLink to="/courses" class="text-sm text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white">← {{ t('courses.back') }}</RouterLink>

      <div v-if="loading" class="mt-6 h-96 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>

      <div v-else-if="!course" class="card mt-6 flex flex-col items-center py-16 text-center">
        <span class="text-5xl">📚</span>
        <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">{{ errorText || t('courses.notFound') }}</p>
      </div>

      <div v-else class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]" data-testid="course-detail">
        <div class="min-w-0 space-y-6">
          <div class="overflow-hidden rounded-2xl border border-gray-200/70 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="relative aspect-video bg-gradient-to-br from-primary-500 via-violet-500 to-teal-400">
              <img v-if="course.cover_url" :src="course.cover_url" alt="" class="h-full w-full object-cover" />
              <span v-else class="absolute inset-0 flex items-center justify-center text-7xl">📘</span>
            </div>
            <div class="p-6">
              <span v-if="course.category" class="badge badge-gray">{{ course.category }}</span>
              <h1 class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ course.title }}</h1>
              <p v-if="course.subtitle" class="mt-2 text-gray-600 dark:text-dark-300">{{ course.subtitle }}</p>
            </div>
          </div>

          <section v-if="introHtml" class="card p-6">
            <h2 class="mb-3 text-lg font-semibold text-gray-900 dark:text-white">{{ t('courses.intro') }}</h2>
            <div class="course-md" v-html="introHtml"></div>
          </section>

          <section v-if="course.outline.length" class="card p-6" data-testid="course-outline">
            <h2 class="mb-1 text-lg font-semibold text-gray-900 dark:text-white">{{ t('courses.outline') }}</h2>
            <p class="mb-4 text-xs text-gray-500 dark:text-dark-400">{{ t('courses.lessons', { count: course.lesson_count }) }}</p>
            <div v-for="(section, si) in course.outline" :key="si" class="mb-4 last:mb-0">
              <h3 v-if="section.title" class="mb-2 text-sm font-semibold text-gray-800 dark:text-gray-200">{{ section.title }}</h3>
              <ul class="divide-y divide-gray-100 rounded-xl border border-gray-100 dark:divide-dark-700 dark:border-dark-700">
                <li v-for="(lesson, li) in section.lessons" :key="li" class="flex items-center gap-3 px-4 py-2.5 text-sm">
                  <span class="w-6 shrink-0 text-xs text-gray-400">{{ li + 1 }}</span>
                  <span class="min-w-0 flex-1 truncate text-gray-800 dark:text-gray-200">{{ lesson.title }}</span>
                  <span v-if="lesson.trial" class="rounded bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">{{ t('courses.trial') }}</span>
                  <span v-if="lesson.duration" class="shrink-0 text-xs text-gray-400">{{ lesson.duration }}</span>
                </li>
              </ul>
            </div>
          </section>

          <section v-if="trialHtml" class="card p-6" data-testid="course-trial">
            <h2 class="mb-3 text-lg font-semibold text-gray-900 dark:text-white">{{ t('courses.trialTitle') }}</h2>
            <div class="course-md" v-html="trialHtml"></div>
          </section>

          <section class="card p-6">
            <h2 class="mb-3 text-lg font-semibold text-gray-900 dark:text-white">{{ t('courses.faq') }}</h2>
            <div v-if="faqHtml" class="course-md mb-4" v-html="faqHtml"></div>
            <dl class="space-y-3 text-sm">
              <div v-for="key in ['delivery', 'netdisk', 'refund']" :key="key">
                <dt class="font-medium text-gray-800 dark:text-gray-200">{{ t(`courses.builtinFaq.${key}.q`) }}</dt>
                <dd class="mt-1 text-gray-600 dark:text-dark-300">{{ t(`courses.builtinFaq.${key}.a`) }}</dd>
              </div>
            </dl>
          </section>
        </div>

        <aside class="hidden lg:block">
          <div class="card sticky top-20 space-y-4 p-6">
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-bold text-primary-600 dark:text-primary-400">¥{{ course.price }}</span>
              <span v-if="course.original_price" class="text-sm text-gray-400 line-through">¥{{ course.original_price }}</span>
            </div>
            <button class="btn btn-primary w-full py-3" data-testid="course-buy" @click="buy">{{ buyLabel }}</button>
            <ul class="space-y-1.5 text-xs text-gray-500 dark:text-dark-400">
              <li>📚 {{ t('courses.lessons', { count: course.lesson_count }) }}</li>
              <li v-if="course.student_count">👥 {{ t('courses.students', { count: course.student_count }) }}</li>
              <li>🔗 {{ t('courses.deliveryHint') }}</li>
            </ul>
          </div>
        </aside>
      </div>
    </main>

    <!-- Mobile buy bar -->
    <div v-if="course" class="fixed inset-x-0 bottom-0 z-20 flex items-center justify-between gap-3 border-t border-gray-200 bg-white/95 px-4 py-3 backdrop-blur dark:border-dark-700 dark:bg-dark-900/95 lg:hidden">
      <span class="flex items-baseline gap-1.5">
        <span class="text-xl font-bold text-primary-600 dark:text-primary-400">¥{{ course.price }}</span>
        <span v-if="course.original_price" class="text-xs text-gray-400 line-through">¥{{ course.original_price }}</span>
      </span>
      <button class="btn btn-primary px-6" @click="buy">{{ buyLabel }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import { getCourse, type Course } from '@/api/courses'
import { useAuthStore } from '@/stores/auth'
import { renderMarkdown } from '@/utils/markdown'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const slug = computed(() => String(route.params.slug || ''))
const course = ref<Course | null>(null)
const loading = ref(true)
const errorText = ref('')

const introHtml = computed(() => renderMarkdown(course.value?.intro_md))
const trialHtml = computed(() => renderMarkdown(course.value?.trial_md))
const faqHtml = computed(() => renderMarkdown(course.value?.faq_md))
const buyLabel = computed(() => (course.value?.owned ? t('courses.mine.go') : t('courses.buy')))

function buy() {
  if (!course.value) return
  if (course.value.owned) {
    router.push('/my-courses')
    return
  }
  const target = `/purchase?course=${encodeURIComponent(course.value.slug)}`
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
    course.value = await getCourse(slug.value)
    document.title = course.value.title
  } catch (err: unknown) {
    course.value = null
    errorText.value = extractApiErrorMessage(err, t('courses.notFound'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(slug, load)
</script>

<style scoped>
.course-md :deep(h1),
.course-md :deep(h2),
.course-md :deep(h3) {
  @apply mb-2 mt-4 font-semibold text-gray-900 dark:text-white;
}
.course-md :deep(h2) {
  @apply text-base;
}
.course-md :deep(p),
.course-md :deep(li) {
  @apply text-sm leading-7 text-gray-700 dark:text-gray-300;
}
.course-md :deep(ul) {
  @apply my-2 list-disc pl-5;
}
.course-md :deep(ol) {
  @apply my-2 list-decimal pl-5;
}
.course-md :deep(img) {
  @apply my-3 max-w-full rounded-lg;
}
.course-md :deep(a) {
  @apply text-primary-600 underline dark:text-primary-400;
}
.course-md :deep(code) {
  @apply rounded bg-gray-100 px-1 text-xs dark:bg-dark-700;
}
.course-md :deep(blockquote) {
  @apply my-3 border-l-4 border-gray-200 pl-3 text-gray-500 dark:border-dark-600;
}
</style>
