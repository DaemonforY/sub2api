<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card space-y-4 p-5" data-testid="learn-settings">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.learn.runTitle') }}</h3>
          <a href="/learn/" target="_blank" class="text-sm text-primary-600 hover:underline">{{ t('admin.learn.openSite') }} ↗</a>
        </div>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.learn.runHint') }}</p>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.run_enabled" type="checkbox" class="h-4 w-4" data-testid="learn-enabled" />
          {{ t('admin.learn.enabled') }}
        </label>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.apiKey') }}</span>
            <input v-model="apiKey" type="password" autocomplete="new-password" class="input" :placeholder="form.api_key_set ? t('admin.learn.apiKeySet') : t('admin.learn.apiKeyPlaceholder')" data-testid="learn-key" />
            <span class="input-hint">{{ t('admin.learn.apiKeyHint') }}</span>
            <button v-if="form.api_key_set" type="button" class="btn btn-secondary btn-sm mt-2" :disabled="saving" data-testid="learn-key-clear" @click.prevent="clearKey">
              {{ t('admin.learn.clearKey') }}
            </button>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.model') }}</span>
            <input v-model="form.model" class="input" placeholder="gpt-5.5" data-testid="learn-model" />
            <span class="input-hint">{{ t('admin.learn.modelHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.freeRuns') }}</span>
            <input v-model.number="form.free_runs_per_day" type="number" min="0" max="500" class="input" data-testid="learn-free" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.dailyCap') }}</span>
            <input v-model.number="form.daily_cap" type="number" min="0" class="input" />
            <span class="input-hint">{{ t('admin.learn.dailyCapHint') }}</span>
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.tutorFree') }}</span>
            <input v-model.number="form.tutor_free_per_day" type="number" min="0" max="500" class="input" data-testid="learn-tutor" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.learn.interviewsFree') }}</span>
            <input v-model.number="form.interviews_per_day" type="number" min="0" max="50" class="input" data-testid="learn-interviews" />
            <span class="input-hint">{{ t('admin.learn.interviewsHint') }}</span>
          </label>
        </div>
        <button class="btn btn-primary" :disabled="saving" data-testid="learn-save" @click="save">{{ t('common.save') }}</button>
      </div>

      <div class="grid gap-3 sm:grid-cols-3 lg:grid-cols-6" data-testid="learn-stats">
        <div v-for="item in statItems" :key="item.key" class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.learn.stats.${item.key}`) }}</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
        </div>
      </div>

      <div v-if="insights" class="grid gap-3 md:grid-cols-2 xl:grid-cols-4" data-testid="learn-funnels">
        <div v-for="f in insights.funnels" :key="f.track" class="card p-4">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ f.track.toUpperCase() }} · {{ f.title }}</div>
          <div class="mt-3 space-y-2">
            <div v-for="step in funnelSteps(f)" :key="step.key" class="text-xs">
              <div class="flex justify-between text-gray-500 dark:text-dark-400">
                <span>{{ t(`admin.learn.funnel.${step.key}`) }}</span><span class="font-medium text-gray-900 dark:text-white">{{ step.value }}</span>
              </div>
              <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-primary-500" :style="{ width: `${f.started ? (step.value * 100) / f.started : 0}%` }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="insights" class="card p-4" data-testid="learn-trend">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.learn.trend.title') }}</div>
          <div class="flex gap-3 text-xs text-gray-500">
            <span v-for="m in trendMetrics" :key="m.key" class="flex items-center gap-1"><i :class="['inline-block h-2 w-2 rounded-sm', m.color]"></i>{{ t(`admin.learn.trend.${m.key}`) }}</span>
          </div>
        </div>
        <div class="mt-4 flex h-36 items-end gap-1.5">
          <div v-for="d in insights.days" :key="d.date" class="flex h-full flex-1 flex-col items-center justify-end" :title="trendTitle(d)">
            <div class="flex w-full flex-1 items-end justify-center gap-px">
              <div v-for="m in trendMetrics" :key="m.key" :class="['w-1/4 rounded-t', m.color]" :style="{ height: `${(d[m.key] / trendMax) * 100}%` }"></div>
            </div>
            <div class="mt-1 text-[10px] text-gray-400">{{ d.date.slice(5) }}</div>
          </div>
        </div>
      </div>

      <div class="card overflow-x-auto">
        <table class="w-full whitespace-nowrap text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2">{{ t('admin.learn.lesson') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.completed') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.runs') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.quizTakers') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.quizPassRate') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.quizAvg') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="l in stats?.lessons || []" :key="l.lesson_id">
              <td class="px-4 py-2"><a :href="`/learn/${l.lesson_id[0]}/${l.lesson_id}.html`" target="_blank" class="font-mono text-primary-600 hover:underline">{{ l.lesson_id.toUpperCase() }}</a></td>
              <td class="px-4 py-2">{{ l.completed }}</td>
              <td class="px-4 py-2">{{ l.runs }}</td>
              <td class="px-4 py-2">{{ insights?.quizzes[l.lesson_id]?.takers ?? '—' }}</td>
              <td class="px-4 py-2">{{ passRate(l.lesson_id) }}</td>
              <td class="px-4 py-2">{{ insights?.quizzes[l.lesson_id] ? `${insights.quizzes[l.lesson_id].avg_score}%` : '—' }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="!stats?.lessons?.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.learn.noData') }}</p>
      </div>

      <div class="card overflow-x-auto" data-testid="learn-certs">
        <div class="px-4 pt-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.learn.certs.title') }}</div>
        <table class="mt-2 w-full whitespace-nowrap text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2">{{ t('admin.learn.certs.code') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.user') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.track') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.score') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.project') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.issued') }}</th>
              <th class="px-4 py-2">{{ t('admin.learn.certs.wall') }}</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="c in certs" :key="c.code" :class="{ 'opacity-50': c.revoked_at }">
              <td class="px-4 py-2"><a :href="`/learn/cert.html?c=${c.code}`" target="_blank" class="font-mono text-primary-600 hover:underline">{{ c.code }}</a></td>
              <td class="px-4 py-2">{{ c.display_name }}<div class="text-xs text-gray-400">{{ c.user_email }}</div></td>
              <td class="px-4 py-2">{{ c.track_title || c.track }}</td>
              <td class="px-4 py-2">{{ c.quiz_score ? `${c.quiz_score}%` : '—' }}</td>
              <td class="max-w-[16rem] truncate px-4 py-2"><a v-if="c.project_url" :href="c.project_url" target="_blank" rel="noopener noreferrer nofollow" class="text-primary-600 hover:underline">{{ c.project_url }}</a></td>
              <td class="px-4 py-2">{{ new Date(c.issued_at).toLocaleDateString() }}</td>
              <td class="px-4 py-2">
                <template v-if="c.showcase && !c.revoked_at">
                  <span :class="c.showcase_hidden ? 'text-gray-400' : 'text-emerald-600'">{{ c.showcase_hidden ? t('admin.learn.certs.wallHidden') : t('admin.learn.certs.wallShown') }}</span>
                  <button class="ml-2 text-xs text-primary-600 hover:underline" data-testid="learn-cert-wall" @click="toggleWall(c)">
                    {{ c.showcase_hidden ? t('admin.learn.certs.wallRestore') : t('admin.learn.certs.wallHide') }}
                  </button>
                </template>
                <span v-else class="text-gray-400">—</span>
              </td>
              <td class="px-4 py-2 text-right">
                <button class="btn btn-secondary btn-sm" data-testid="learn-cert-revoke" @click="toggleRevoke(c)">
                  {{ c.revoked_at ? t('admin.learn.certs.restore') : t('admin.learn.certs.revoke') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!certs.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.learn.certs.empty') }}</p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { LearnCertificate, LearnInsights, LearnSettings, LearnStats } from '@/api/admin/learn'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const form = reactive<LearnSettings>({ run_enabled: false, model: '', free_runs_per_day: 20, daily_cap: 1000, tutor_free_per_day: 10, interviews_per_day: 3, api_key_set: false })
const certs = ref<LearnCertificate[]>([])
const insights = ref<LearnInsights | null>(null)

type Day = LearnInsights['days'][number]
type TrendKey = 'learners' | 'runs' | 'tutor' | 'interviews'
const trendMetrics: { key: TrendKey; color: string }[] = [
  { key: 'learners', color: 'bg-primary-500' },
  { key: 'runs', color: 'bg-teal-400' },
  { key: 'tutor', color: 'bg-amber-400' },
  { key: 'interviews', color: 'bg-pink-400' }
]
const trendMax = computed(() => Math.max(1, ...(insights.value?.days || []).flatMap((d) => trendMetrics.map((m) => d[m.key]))))

function trendTitle(d: Day) {
  return `${d.date}：${trendMetrics.map((m) => `${t(`admin.learn.trend.${m.key}`)} ${d[m.key]}`).join(' · ')} · ${t('admin.learn.trend.completions')} ${d.completions} · ${t('admin.learn.trend.certificates')} ${d.certificates}`
}

function funnelSteps(f: LearnInsights['funnels'][number]) {
  return [
    { key: 'started', value: f.started },
    { key: 'half', value: f.half },
    { key: 'finished', value: f.finished },
    { key: 'certificates', value: f.certificates }
  ]
}

function passRate(lesson: string) {
  const q = insights.value?.quizzes[lesson]
  return q && q.takers ? `${Math.round((q.passed * 100) / q.takers)}%` : '—'
}

async function toggleWall(c: LearnCertificate) {
  try {
    await adminAPI.learn.setShowcaseHidden(c.code, !c.showcase_hidden)
    await loadCerts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}
const apiKey = ref('')
const saving = ref(false)
const stats = ref<LearnStats | null>(null)

const statItems = computed(() => {
  const s = stats.value
  return [
    { key: 'learners', value: s?.learners ?? 0 },
    { key: 'learnersToday', value: s?.learners_today ?? 0 },
    { key: 'runsToday', value: s?.runs_today ?? 0 },
    { key: 'runs7d', value: s?.runs_7d ?? 0 },
    { key: 'failed7d', value: s?.failed_runs_7d ?? 0 },
    { key: 'tokens7d', value: (s?.tokens_7d ?? 0).toLocaleString() },
    { key: 'tutor7d', value: s?.tutor_7d ?? 0 },
    { key: 'interviews7d', value: s?.interviews_7d ?? 0 },
    { key: 'ownKey7d', value: s?.own_key_runs_7d ?? 0 },
    { key: 'quizPassed', value: s?.quiz_passed ?? 0 },
    { key: 'certificates', value: s?.certificates ?? 0 },
  ]
})

async function clearKey() {
  saving.value = true
  try {
    Object.assign(form, await adminAPI.learn.saveSettings({ ...form, clear_api_key: true }))
    apiKey.value = ''
    appStore.showSuccess(t('admin.learn.keyCleared'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload: LearnSettings = { ...form }
    if (apiKey.value.trim()) payload.api_key = apiKey.value.trim()
    Object.assign(form, await adminAPI.learn.saveSettings(payload))
    apiKey.value = ''
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function loadCerts() {
  certs.value = (await adminAPI.learn.certificates()).items
}

async function toggleRevoke(c: LearnCertificate) {
  const revoke = !c.revoked_at
  if (revoke && !window.confirm(t('admin.learn.certs.confirm', { code: c.code }))) return
  try {
    await adminAPI.learn.setCertificateRevoked(c.code, revoke)
    await loadCerts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(async () => {
  try {
    Object.assign(form, await adminAPI.learn.getSettings())
    stats.value = await adminAPI.learn.stats()
    insights.value = await adminAPI.learn.insights()
    await loadCerts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
})
</script>
