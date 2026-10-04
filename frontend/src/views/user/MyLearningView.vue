<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card flex flex-wrap items-center justify-between gap-3 p-5">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('learn.mine.headline') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('learn.mine.hint') }}</p>
        </div>
        <a href="/learn/" class="btn btn-primary" data-testid="my-learning-open">{{ t('learn.mine.open') }} →</a>
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 2" :key="i" class="h-28 animate-pulse rounded-2xl bg-gray-200 dark:bg-dark-800"></div>
      </div>

      <template v-else-if="data">
        <div class="grid gap-3 sm:grid-cols-3" data-testid="my-learning-usage">
          <div v-for="u in usage" :key="u.key" class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`learn.mine.usage.${u.key}`) }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ u.used }}</div>
            <div class="text-xs text-gray-400 dark:text-dark-500">{{ t('learn.mine.left', { n: u.left }) }}</div>
          </div>
        </div>

        <div v-if="data.certificates.length" class="card p-5" data-testid="my-learning-certs">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('learn.mine.certificates') }}</h4>
          <div class="mt-3 grid gap-3 sm:grid-cols-2">
            <a
              v-for="c in data.certificates"
              :key="c.code"
              :href="`/learn/cert.html?c=${c.code}`"
              class="flex items-center gap-3 rounded-xl border border-primary-100 bg-primary-50/60 p-4 hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800"
            >
              <span class="text-3xl">🎓</span>
              <span class="min-w-0">
                <span class="block truncate font-semibold text-gray-900 dark:text-white">{{ c.track_title }}</span>
                <span class="block text-xs text-gray-500 dark:text-dark-400">{{ formatDate(c.issued_at) }} · {{ c.code }}</span>
              </span>
            </a>
          </div>
        </div>

        <div v-for="track in trackList" :key="track.id" class="card p-5" data-testid="my-learning-track">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <a :href="`/learn/${track.id}/`" class="text-base font-semibold text-gray-900 hover:underline dark:text-white">{{ track.letter }} · {{ track.title }}</a>
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('learn.mine.progress', { done: doneCount(track), total: track.lessons.length }) }}
              <template v-if="hasCert(track.id)"> · <span class="font-semibold text-emerald-600">{{ t('learn.mine.certified') }}</span></template>
            </span>
          </div>
          <div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
            <div class="h-full rounded-full bg-primary-500" :style="{ width: `${track.lessons.length ? (doneCount(track) * 100) / track.lessons.length : 0}%` }"></div>
          </div>
          <div class="mt-3 flex flex-wrap gap-2">
            <a
              v-for="l in track.lessons"
              :key="l.id"
              :href="`/learn/${track.id}/${l.id}.html`"
              :title="l.title"
              :class="[
                'rounded-lg px-2 py-1 text-xs',
                data.completed[l.id] ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'
              ]"
            >
              {{ data.completed[l.id] ? '✓ ' : '' }}{{ l.id.toUpperCase() }}
              <template v-if="data.quizzes[l.id]"> · {{ data.quizzes[l.id].correct }}/{{ data.quizzes[l.id].total }}</template>
            </a>
          </div>
          <a v-if="nextLesson(track)" :href="`/learn/${track.id}/${nextLesson(track)!.id}.html`" class="mt-3 inline-block text-sm text-primary-600 hover:underline">
            {{ t('learn.mine.continue', { title: nextLesson(track)!.title }) }} →
          </a>
        </div>
        <p v-if="!trackList.length" class="card p-6 text-center text-sm text-gray-500">{{ t('learn.mine.noTracks') }}</p>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import learnAPI, { type LearnMe, type LearnTrack } from '@/api/learn'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const data = ref<LearnMe | null>(null)
const trackList = ref<LearnTrack[]>([])

const usage = computed(() => {
  const d = data.value
  return [
    { key: 'runs', used: d?.runs_today ?? 0, left: d?.runs_left ?? 0 },
    { key: 'tutor', used: d?.tutor_today ?? 0, left: d?.tutor_left ?? 0 },
    { key: 'interviews', used: d?.interviews_today ?? 0, left: d?.interviews_left ?? 0 }
  ]
})

function doneCount(track: LearnTrack) {
  return track.lessons.filter((l) => data.value?.completed[l.id]).length
}
function hasCert(track: string) {
  return !!data.value?.certificates.some((c) => c.track === track)
}
function nextLesson(track: LearnTrack) {
  return doneCount(track) > 0 ? track.lessons.find((l) => !data.value?.completed[l.id]) : undefined
}
function formatDate(s: string) {
  return new Date(s).toLocaleDateString()
}

onMounted(async () => {
  try {
    const [me, list] = await Promise.all([learnAPI.me(), learnAPI.tracks().catch(() => [] as LearnTrack[])])
    data.value = me
    trackList.value = list
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
})
</script>
