<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('tutors.title') }}</h2>
            <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-dark-400">{{ t('tutors.intro') }}</p>
          </div>
          <button v-if="!creating && tutors.length" class="btn btn-primary" data-testid="tutor-new" @click="startCreate">{{ t('tutors.new') }}</button>
        </div>
      </div>

      <!-- 新建：先选模板，再填 3 项 -->
      <div v-if="creating || (!loading && !tutors.length)" class="card space-y-4 p-5" data-testid="tutor-create">
        <h3 class="font-semibold text-gray-900 dark:text-white">{{ step === 1 ? t('tutors.pickTemplate') : t('tutors.fillIn') }}</h3>
        <div v-if="step === 1" class="grid gap-3 sm:grid-cols-2">
          <button
            v-for="tpl in templates"
            :key="tpl.id"
            type="button"
            class="rounded-xl border border-gray-200 p-4 text-left transition hover:border-primary-400 hover:bg-primary-50/40 dark:border-dark-600 dark:hover:bg-primary-900/20"
            :data-testid="`tutor-template-${tpl.id}`"
            @click="pickTemplate(tpl)"
          >
            <div class="font-medium text-gray-900 dark:text-white">{{ templateIcon[tpl.id] }} {{ tpl.name }}</div>
            <div class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ tpl.description }}</div>
            <div class="mt-2 text-xs text-gray-400">{{ t('tutors.materialsHint', { list: tpl.materials }) }}</div>
          </button>
        </div>
        <template v-else-if="picked">
          <div class="text-sm text-gray-600 dark:text-dark-300">
            {{ templateIcon[picked.id] }} {{ picked.name }}
            <button class="ml-2 text-primary-600 hover:underline" @click="step = 1">{{ t('tutors.changeTemplate') }}</button>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <label class="text-sm">
              <span class="input-label">{{ t('tutors.subject') }}</span>
              <input v-model.trim="form.subject" class="input" maxlength="40" :placeholder="t('tutors.subjectPlaceholder')" data-testid="tutor-subject" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('tutors.grade') }}</span>
              <input v-model.trim="form.grade" class="input" maxlength="40" :placeholder="t('tutors.gradePlaceholder')" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('tutors.style') }}</span>
              <select v-model="form.style" class="input">
                <option v-for="s in styles" :key="s" :value="s">{{ s }}</option>
              </select>
            </label>
          </div>
          <label v-if="picked.id !== 'oral'" class="flex items-start gap-2 text-sm">
            <input v-model="guide" type="checkbox" class="mt-0.5 h-4 w-4" />
            <span>
              {{ t('tutors.guide') }}
              <span class="block text-xs text-gray-500 dark:text-dark-400">{{ t('tutors.guideHint') }}</span>
            </span>
          </label>
          <label class="block text-sm">
            <span class="input-label">{{ t('tutors.key') }}</span>
            <select v-model.number="form.key_id" class="input" data-testid="tutor-key">
              <option :value="0">{{ t('tutors.keyPlaceholder') }}</option>
              <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
            </select>
            <span class="input-hint">
              {{ t('tutors.keyHint') }}
              <router-link to="/keys" class="text-primary-600 hover:underline">{{ t('tutors.keyCreate') }}</router-link>
            </span>
          </label>
          <div class="flex flex-wrap justify-end gap-2">
            <button v-if="tutors.length" class="btn btn-secondary" @click="creating = false">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary" :disabled="saving || !form.key_id" data-testid="tutor-create-submit" @click="create">
              {{ saving ? t('tutors.creating') : t('tutors.create') }}
            </button>
          </div>
        </template>
      </div>

      <!-- 列表 -->
      <div v-if="tutors.length" class="grid gap-3 md:grid-cols-2">
        <router-link
          v-for="tu in tutors"
          :key="tu.id"
          :to="`/tutors/${tu.id}`"
          class="card block p-4 transition hover:shadow-md"
          data-testid="tutor-card"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ templateIcon[tu.template] }} {{ tu.name }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ [tu.grade, tu.subject].filter(Boolean).join(' · ') || templateName(tu.template) }}
              </div>
            </div>
            <span class="rounded-full px-2 py-0.5 text-xs" :class="tu.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700'">
              {{ tu.enabled ? t('tutors.on') : t('tutors.off') }}
            </span>
          </div>
          <div class="mt-3 text-xs text-gray-400">{{ t('tutors.materialsCount', { n: tu.material_chars.toLocaleString() }) }}</div>
        </router-link>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { createTutor, listKeys, listTemplates, listTutors, type KeyOption, type Tutor, type TutorTemplate } from '@/api/tutors'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const templateIcon: Record<string, string> = { qa: '💬', essay: '✍️', quiz: '📝', oral: '🗣️' }
const styles = ['亲切耐心', '简洁直接', '幽默活泼', '严谨专业']

const tutors = ref<Tutor[]>([])
const templates = ref<TutorTemplate[]>([])
const keys = ref<KeyOption[]>([])
const loading = ref(true)
const creating = ref(false)
const saving = ref(false)
const step = ref(1)
const picked = ref<TutorTemplate | null>(null)
const guide = ref(true)
const form = reactive({ subject: '', grade: '', style: styles[0], key_id: 0 })

function templateName(id: string) {
  return templates.value.find((x) => x.id === id)?.name || ''
}

function startCreate() {
  creating.value = true
  step.value = 1
}

function pickTemplate(tpl: TutorTemplate) {
  picked.value = tpl
  step.value = 2
  if (tpl.id === 'oral' && !form.subject) form.subject = '英语'
}

async function create() {
  if (!picked.value) return
  saving.value = true
  try {
    const tpl = picked.value
    const name = form.subject ? `${form.subject}${tpl.id === 'oral' ? '口语陪练' : tpl.id === 'essay' ? '作文批改' : '助教'}` : tpl.name
    const created = await createTutor({
      key_id: form.key_id,
      name,
      template: tpl.id,
      subject: form.subject,
      grade: form.grade,
      style: form.style,
      answer_mode: guide.value ? 'guide' : 'answer',
      rules: '',
      greeting: '',
      pass_code: String(Math.floor(1000 + Math.random() * 9000)),
      per_student_day: 20,
      daily_cap: 300,
      enabled: true
    })
    router.push(`/tutors/${created.id}?new=1`)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const [list, tpls, ks] = await Promise.all([listTutors(), listTemplates(), listKeys()])
    tutors.value = list
    templates.value = tpls
    keys.value = ks
    if (ks.length === 1) form.key_id = ks[0].id
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
})
</script>
