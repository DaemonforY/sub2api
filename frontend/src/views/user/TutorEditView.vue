<template>
  <AppLayout>
    <div v-if="tutor" class="space-y-4">
      <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
        <div class="min-w-0">
          <router-link to="/tutors" class="text-xs text-gray-400 hover:text-primary-600">‹ {{ t('tutors.title') }}</router-link>
          <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ tutor.name }}</h2>
        </div>
        <span class="rounded-full px-2 py-0.5 text-xs" :class="tutor.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700'">
          {{ tutor.enabled ? t('tutors.on') : t('tutors.off') }}
        </span>
      </div>

      <div v-if="isNew" class="card border-primary-200 bg-primary-50/60 p-4 text-sm text-primary-800 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-200">
        {{ t('tutors.nextSteps') }}
      </div>
      <div v-if="tutor.key_problem" class="card border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
        {{ t('tutors.keyProblem', { msg: tutor.key_problem }) }}
      </div>

      <div class="flex flex-wrap gap-1 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tb in tabs"
          :key="tb.id"
          class="-mb-px border-b-2 px-4 py-2 text-sm"
          :class="tab === tb.id ? 'border-primary-600 font-medium text-primary-700 dark:text-primary-300' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-dark-400'"
          :data-testid="`tutor-tab-${tb.id}`"
          @click="tab = tb.id"
        >
          {{ tb.label }}
        </button>
      </div>

      <!-- 资料 -->
      <div v-if="tab === 'materials'" class="card space-y-4 p-5">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('tutors.materialsIntro', { list: template?.materials || '' }) }}</p>
        <ul v-if="tutor.materials?.length" class="divide-y divide-gray-100 rounded-lg border border-gray-100 dark:divide-dark-700 dark:border-dark-700">
          <li v-for="m in tutor.materials" :key="m.id" class="flex items-center justify-between gap-3 px-3 py-2 text-sm" data-testid="tutor-material">
            <span class="min-w-0 truncate text-gray-800 dark:text-dark-100">📄 {{ m.name }}</span>
            <span class="flex flex-shrink-0 items-center gap-3 text-xs text-gray-400">
              {{ t('tutors.chars', { n: m.chars.toLocaleString() }) }}
              <button class="text-red-500 hover:underline" @click="removeMaterial(m.id)">{{ t('common.delete') }}</button>
            </span>
          </li>
        </ul>
        <p v-else class="text-sm text-amber-600">{{ t('tutors.noMaterials') }}</p>
        <div class="text-xs text-gray-400">{{ t('tutors.materialsTotal', { n: tutor.material_chars.toLocaleString() }) }}</div>
        <div class="flex flex-wrap items-center gap-3">
          <label class="btn btn-primary cursor-pointer">
            {{ uploading ? t('tutors.uploading') : t('tutors.upload') }}
            <input type="file" class="hidden" accept=".docx,.pptx,.pdf,.txt,.md" :disabled="uploading" data-testid="tutor-upload" @change="upload" />
          </label>
          <span class="text-xs text-gray-400">{{ t('tutors.uploadHint') }}</span>
        </div>
        <details class="text-sm">
          <summary class="cursor-pointer text-primary-600">{{ t('tutors.pasteTitle') }}</summary>
          <div class="mt-2 space-y-2">
            <input v-model.trim="pasteName" class="input" maxlength="100" :placeholder="t('tutors.pasteName')" />
            <textarea v-model="pasteText" class="input min-h-[8rem]" :placeholder="t('tutors.pastePlaceholder')"></textarea>
            <button class="btn btn-secondary" :disabled="!pasteText.trim() || uploading" @click="paste">{{ t('tutors.pasteAdd') }}</button>
          </div>
        </details>
      </div>

      <!-- 试一试 -->
      <div v-if="tab === 'try'" class="card p-5">
        <p class="mb-3 text-sm text-gray-500 dark:text-dark-400">{{ t('tutors.tryIntro') }}</p>
        <TutorChat :greeting="greeting" :suggestions="template?.suggestions || []" :send="previewSend" data-testid="tutor-preview" />
      </div>

      <!-- 分享 -->
      <div v-if="tab === 'share'" class="card space-y-4 p-5">
        <div class="grid gap-5 md:grid-cols-[auto_1fr]">
          <canvas ref="qrCanvas" class="h-44 w-44 rounded-lg border border-gray-100 dark:border-dark-700"></canvas>
          <div class="space-y-3 text-sm">
            <div>
              <div class="input-label">{{ t('tutors.shareLink') }}</div>
              <div class="flex gap-2">
                <input :value="shareUrl" class="input font-mono text-xs" readonly data-testid="tutor-share-url" />
                <button class="btn btn-secondary" @click="copy(shareUrl)">{{ t('tutors.copy') }}</button>
              </div>
            </div>
            <div>
              <div class="input-label">{{ t('tutors.passCode') }}</div>
              <div class="text-2xl font-semibold tracking-widest text-gray-900 dark:text-white">{{ tutor.pass_code || t('tutors.noPass') }}</div>
            </div>
            <button class="btn btn-primary" @click="copy(shareText)">{{ t('tutors.copyNotice') }}</button>
            <pre class="whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ shareText }}</pre>
          </div>
        </div>
        <p v-if="!tutor.enabled" class="text-sm text-amber-600">{{ t('tutors.shareOff') }}</p>
      </div>

      <!-- 学情 -->
      <div v-if="tab === 'stats'" class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-3">
          <div class="card p-4">
            <div class="text-xs text-gray-500">{{ t('tutors.statToday') }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stats?.today ?? '—' }}</div>
          </div>
          <div class="card p-4">
            <div class="text-xs text-gray-500">{{ t('tutors.statWeek') }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stats?.week ?? '—' }}</div>
          </div>
          <div class="card p-4">
            <div class="text-xs text-gray-500">{{ t('tutors.statStudents') }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ stats?.students.length ?? '—' }}</div>
          </div>
        </div>

        <div class="card space-y-3 p-5">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('tutors.insights') }}</h3>
            <button class="btn btn-primary" :disabled="insightsBusy" data-testid="tutor-insights" @click="runInsights">
              {{ insightsBusy ? t('tutors.insightsBusy') : t('tutors.insightsRun') }}
            </button>
          </div>
          <p v-if="!insights && !insightsBusy" class="text-sm text-gray-500 dark:text-dark-400">{{ t('tutors.insightsHint') }}</p>
          <div v-if="insights" class="tutor-md text-sm text-gray-800 dark:text-dark-100" v-html="renderMarkdown(insights)"></div>
        </div>

        <div class="card p-5">
          <h3 class="mb-3 font-semibold text-gray-900 dark:text-white">{{ t('tutors.students') }}</h3>
          <p v-if="!stats?.students.length" class="text-sm text-gray-500">{{ t('tutors.noStudents') }}</p>
          <div v-else class="flex flex-wrap gap-2">
            <span v-for="s in stats.students" :key="s.id" class="rounded-full bg-gray-100 px-3 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ s.name }} · {{ t('tutors.questionsN', { n: s.questions }) }} · {{ shortTime(s.last_seen_at) }}
            </span>
          </div>
        </div>

        <div class="card p-5">
          <h3 class="mb-3 font-semibold text-gray-900 dark:text-white">{{ t('tutors.recent') }}</h3>
          <p v-if="!stats?.recent.length" class="text-sm text-gray-500">{{ t('tutors.noQuestions') }}</p>
          <ul class="divide-y divide-gray-100 dark:divide-dark-700">
            <li v-for="m in stats?.recent || []" :key="m.id" class="py-2 text-sm">
              <button class="w-full text-left" @click="open = open === m.id ? 0 : m.id">
                <span class="text-xs text-gray-400">{{ shortTime(m.created_at) }} · {{ m.student_name }}</span>
                <span class="block text-gray-900 dark:text-white">{{ m.question }}</span>
              </button>
              <div v-if="open === m.id" class="tutor-md mt-2 rounded-lg bg-gray-50 p-3 text-gray-700 dark:bg-dark-800 dark:text-dark-200" v-html="renderMarkdown(m.answer)"></div>
            </li>
          </ul>
        </div>
      </div>

      <!-- 设置 -->
      <div v-if="tab === 'settings'" class="card space-y-4 p-5">
        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.name') }}</span>
            <input v-model.trim="form.name" class="input" maxlength="30" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.style') }}</span>
            <input v-model.trim="form.style" class="input" maxlength="100" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.subject') }}</span>
            <input v-model.trim="form.subject" class="input" maxlength="40" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.grade') }}</span>
            <input v-model.trim="form.grade" class="input" maxlength="40" />
          </label>
        </div>
        <label v-if="form.template !== 'oral'" class="flex items-start gap-2 text-sm">
          <input v-model="guide" type="checkbox" class="mt-0.5 h-4 w-4" />
          <span>
            {{ t('tutors.guide') }}
            <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t('tutors.guideHint') }}</span>
          </span>
        </label>
        <label class="block text-sm">
          <span class="input-label">{{ t('tutors.rules') }}</span>
          <textarea v-model="form.rules" class="input min-h-[5rem]" maxlength="2000" :placeholder="t('tutors.rulesPlaceholder')"></textarea>
        </label>
        <label class="block text-sm">
          <span class="input-label">{{ t('tutors.greeting') }}</span>
          <textarea v-model="form.greeting" class="input min-h-[3.5rem]" maxlength="500" :placeholder="defaultGreeting"></textarea>
        </label>
        <div class="grid gap-4 md:grid-cols-3">
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.passCode') }}</span>
            <input v-model.trim="form.pass_code" class="input" maxlength="20" :placeholder="t('tutors.passPlaceholder')" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.perStudent') }}</span>
            <input v-model.number="form.per_student_day" type="number" min="1" max="200" class="input" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('tutors.dailyCap') }}</span>
            <input v-model.number="form.daily_cap" type="number" min="0" max="20000" class="input" />
            <span class="input-hint">{{ t('tutors.dailyCapHint') }}</span>
          </label>
        </div>
        <label class="block text-sm">
          <span class="input-label">{{ t('tutors.key') }}</span>
          <select v-model.number="form.key_id" class="input">
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
            <option v-if="!keys.some((k) => k.id === form.key_id)" :value="form.key_id">{{ tutor.key_name || `#${form.key_id}` }}</option>
          </select>
          <span class="input-hint">{{ t('tutors.keyCostHint') }}</span>
        </label>
        <div class="text-sm">
          <span class="input-label">{{ t('tutors.model') }}</span>
          <div class="grid gap-3 sm:grid-cols-2">
            <label
              v-for="tier in tiers"
              :key="tier"
              class="flex cursor-pointer items-start gap-2 rounded-xl border p-3"
              :class="form.model_tier === tier ? 'border-primary-500 bg-primary-50/50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-600'"
              :data-testid="`tutor-tier-${tier}`"
            >
              <input v-model="form.model_tier" type="radio" :value="tier" class="mt-1" />
              <span>
                <span class="font-medium text-gray-900 dark:text-white">{{ t(`tutors.tier.${tier}`) }}</span>
                <span v-if="tutor.costs?.[tier]" class="ml-1 text-xs text-gray-500">{{ tutor.costs[tier]!.model }}</span>
                <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t(`tutors.tier.${tier}Hint`) }}</span>
                <span v-if="tutor.costs?.[tier]" class="mt-1 block text-xs font-medium text-amber-700 dark:text-amber-300">
                  {{ t('tutors.tierPrice', { low: money(tutor.costs[tier]!.low), high: money(tutor.costs[tier]!.high) }) }}
                </span>
              </span>
            </label>
          </div>
        </div>
        <div v-if="cost" class="rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:bg-amber-900/20 dark:text-amber-200" data-testid="tutor-cost">
          <div class="font-medium">{{ t('tutors.costTitle', { low: money(cost.low), high: money(cost.high) }) }}</div>
          <div class="mt-1 text-xs leading-5 opacity-80">
            {{ t('tutors.costDetail', { model: cost.model, input: cost.input_tokens.toLocaleString(), output: cost.output_tokens }) }}
          </div>
          <div class="mt-1 text-xs leading-5 opacity-80">
            {{ t('tutors.costClass', { students: classSize, n: classQuestions, low: money(cost.low * classSize * classQuestions), high: money(cost.high * classSize * classQuestions) }) }}
          </div>
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="form.enabled" type="checkbox" class="h-4 w-4" />
          {{ t('tutors.enabled') }}
        </label>
        <div class="flex flex-wrap justify-between gap-2">
          <button class="btn btn-danger" @click="remove">{{ t('tutors.delete') }}</button>
          <button class="btn btn-primary" :disabled="saving" data-testid="tutor-save" @click="save">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import AppLayout from '@/components/layout/AppLayout.vue'
import TutorChat from '@/components/tutor/TutorChat.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import {
  deleteMaterial,
  deleteTutor,
  getStats,
  getTutor,
  listKeys,
  listTemplates,
  pasteMaterial,
  streamInsights,
  streamPreview,
  updateTutor,
  uploadMaterial,
  type ChatMessage,
  type KeyOption,
  type Tutor,
  type TutorInput,
  type TutorStats,
  type TutorTemplate
} from '@/api/tutors'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const id = Number(route.params.id)
const isNew = route.query.new === '1'
const tutor = ref<Tutor | null>(null)
const templates = ref<TutorTemplate[]>([])
const keys = ref<KeyOption[]>([])
const tab = ref(isNew ? 'materials' : 'stats')
const tabs = computed(() => [
  { id: 'stats', label: t('tutors.tabStats') },
  { id: 'materials', label: t('tutors.tabMaterials') },
  { id: 'try', label: t('tutors.tabTry') },
  { id: 'share', label: t('tutors.tabShare') },
  { id: 'settings', label: t('tutors.tabSettings') }
])
const form = reactive<TutorInput>({
  key_id: 0, name: '', template: 'qa', subject: '', grade: '', style: '', answer_mode: 'guide', model_tier: 'standard', rules: '', greeting: '',
  pass_code: '', per_student_day: 20, daily_cap: 300, enabled: true
})
const guide = ref(true)
const saving = ref(false)
const uploading = ref(false)
const pasteName = ref('')
const pasteText = ref('')
const stats = ref<TutorStats | null>(null)
const insights = ref('')
const insightsBusy = ref(false)
const open = ref(0)
const qrCanvas = ref<HTMLCanvasElement | null>(null)

const template = computed(() => templates.value.find((x) => x.id === tutor.value?.template))
// The class in the cost example: 40 students asking up to 10 questions a day (or the daily limit).
const classSize = 40
const tiers = ['standard', 'economy'] as const
// The quote follows the tier picked on the page, before it is saved.
const cost = computed(() => tutor.value?.costs?.[form.model_tier] || tutor.value?.cost || null)
const classQuestions = computed(() => Math.min(10, form.per_student_day || 10))

function money(v: number) {
  return `$${v >= 1 ? v.toFixed(2) : v >= 0.1 ? v.toFixed(3) : v.toFixed(4)}`
}
const defaultGreeting = computed(() =>
  (template.value?.greeting || '').replace('{name}', tutor.value?.name || '').replace('{subject}', tutor.value?.subject || '课')
)
const greeting = computed(() => tutor.value?.greeting || defaultGreeting.value)
const shareUrl = computed(() => (tutor.value ? `${window.location.origin}/t/${tutor.value.share_code}` : ''))
const shareText = computed(() => {
  const tu = tutor.value
  if (!tu) return ''
  return t('tutors.noticeText', { name: tu.name, url: shareUrl.value, n: tu.per_student_day }) + (tu.pass_code ? t('tutors.noticePass', { pass: tu.pass_code }) : '')
})

function shortTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fill(tu: Tutor) {
  tutor.value = tu
  Object.assign(form, {
    key_id: tu.key_id, name: tu.name, template: tu.template, subject: tu.subject, grade: tu.grade, style: tu.style, answer_mode: tu.answer_mode,
    model_tier: tu.model_tier || 'standard',
    rules: tu.rules, greeting: tu.greeting, pass_code: tu.pass_code, per_student_day: tu.per_student_day, daily_cap: tu.daily_cap, enabled: tu.enabled
  })
  guide.value = tu.answer_mode === 'guide'
}

async function load() {
  try {
    const [tu, tpls, ks] = await Promise.all([getTutor(id), listTemplates(), listKeys()])
    templates.value = tpls
    keys.value = ks
    fill(tu)
    if (tab.value === 'stats') loadStats()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
    router.replace('/tutors')
  }
}

async function loadStats() {
  try {
    stats.value = await getStats(id)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function save() {
  saving.value = true
  try {
    fill(await updateTutor(id, { ...form, answer_mode: guide.value ? 'guide' : 'answer' }))
    appStore.showSuccess(t('tutors.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!window.confirm(t('tutors.deleteConfirm'))) return
  try {
    await deleteTutor(id)
    router.replace('/tutors')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function afterMaterial(fn: () => Promise<unknown>) {
  uploading.value = true
  try {
    await fn()
    fill(await getTutor(id))
    appStore.showSuccess(t('tutors.materialAdded'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    uploading.value = false
  }
}

function upload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) afterMaterial(() => uploadMaterial(id, file))
}

function paste() {
  afterMaterial(async () => {
    await pasteMaterial(id, pasteName.value, pasteText.value)
    pasteName.value = ''
    pasteText.value = ''
  })
}

async function removeMaterial(materialId: number) {
  if (!window.confirm(t('tutors.deleteMaterialConfirm'))) return
  try {
    await deleteMaterial(id, materialId)
    fill(await getTutor(id))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function previewSend(messages: ChatMessage[], onDelta: (t: string) => void, signal: AbortSignal) {
  return streamPreview(id, messages, onDelta, signal).then(() => ({ left: -1 }))
}

async function runInsights() {
  insightsBusy.value = true
  insights.value = ''
  try {
    await streamInsights(id, (d) => (insights.value += d))
  } catch (e) {
    appStore.showError((e as Error).message)
  } finally {
    insightsBusy.value = false
  }
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess(t('tutors.copied'))
  } catch {
    appStore.showError(t('tutors.copyFailed'))
  }
}

watch(tab, async (v) => {
  if (v === 'stats') loadStats()
  if (v === 'share') {
    await nextTick()
    if (qrCanvas.value && shareUrl.value) await QRCode.toCanvas(qrCanvas.value, shareUrl.value, { width: 176, margin: 1 })
  }
})

onMounted(load)
</script>

<style scoped>
.tutor-md :deep(h2) {
  margin: 0.75rem 0 0.25rem;
  font-weight: 600;
}
.tutor-md :deep(ul),
.tutor-md :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.25rem;
  list-style: disc;
}
.tutor-md :deep(p) {
  margin: 0.25rem 0;
}
</style>
