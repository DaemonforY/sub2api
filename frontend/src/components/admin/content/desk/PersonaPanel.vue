<template>
  <div class="space-y-4" data-testid="content-persona">
    <div class="card space-y-2 p-4">
      <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.desk.persona.intro') }}</p>
      <p v-if="persona.updated_at" class="text-xs text-gray-400">{{ t('admin.desk.persona.updated', { time: fmtTime(persona.updated_at) }) }}</p>
    </div>

    <!-- Samples -->
    <div class="card space-y-3 p-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.persona.samples') }}</h3>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.persona.samplesHint') }}</p>
        </div>
        <button class="btn btn-primary btn-sm" :disabled="distilling || persona.samples.length < 2" data-testid="persona-distill" @click="distill">
          {{ distilling ? t('admin.desk.persona.distilling') : t('admin.desk.persona.distill') }}
        </button>
      </div>
      <ul class="space-y-2">
        <li v-for="s in persona.samples" :key="s.id" class="flex items-start justify-between gap-3 rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
          <div class="min-w-0">
            <div class="font-medium text-gray-900 dark:text-white">{{ s.title }}</div>
            <div class="mt-0.5 line-clamp-2 text-xs text-gray-500">{{ s.text.slice(0, 160) }}</div>
          </div>
          <button class="flex-shrink-0 text-xs text-red-600 hover:underline" @click="removeSample(s.id)">{{ t('admin.desk.common.delete') }}</button>
        </li>
      </ul>
      <div v-if="persona.samples.length < 5" class="space-y-2 rounded-lg border border-dashed border-gray-200 p-3 dark:border-dark-600">
        <div class="flex flex-wrap gap-2">
          <input v-model.trim="sampleUrl" class="input min-w-0 flex-1" :placeholder="t('admin.desk.persona.urlPlaceholder')" data-testid="persona-sample-url" />
          <button class="btn btn-secondary" :disabled="adding || !sampleUrl" @click="addByUrl">{{ t('admin.desk.persona.addUrl') }}</button>
        </div>
        <details>
          <summary class="cursor-pointer text-xs text-gray-500">{{ t('admin.desk.persona.paste') }}</summary>
          <div class="mt-2 space-y-2">
            <input v-model.trim="sampleTitle" class="input" :placeholder="t('admin.desk.persona.titlePlaceholder')" />
            <textarea v-model="sampleText" class="input min-h-[140px] text-sm" :placeholder="t('admin.desk.persona.textPlaceholder')" />
            <button class="btn btn-secondary btn-sm" :disabled="adding || sampleText.trim().length < 200" @click="addByText">{{ t('admin.desk.persona.addText') }}</button>
          </div>
        </details>
      </div>
    </div>

    <!-- Fields -->
    <div class="card space-y-4 p-4">
      <p v-if="proposed" class="rounded-lg bg-primary-50 p-2 text-sm text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">{{ t('admin.desk.persona.proposed') }}</p>
      <div class="grid gap-4 md:grid-cols-2">
        <label class="text-sm">
          <span class="input-label">{{ t('admin.desk.persona.name') }}</span>
          <input v-model.trim="form.name" class="input" maxlength="40" />
        </label>
      </div>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.persona.identity') }}</span>
        <textarea v-model="form.identity" class="input min-h-[80px] text-sm" maxlength="2000" :placeholder="t('admin.desk.persona.identityPlaceholder')" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.persona.experiences') }}</span>
        <textarea v-model="lists.experiences" class="input min-h-[110px] text-sm" :placeholder="t('admin.desk.persona.experiencesPlaceholder')" />
        <span class="input-hint">{{ t('admin.desk.persona.experiencesHint') }}</span>
      </label>
      <div class="grid gap-4 md:grid-cols-2">
        <label class="text-sm">
          <span class="input-label">{{ t('admin.desk.persona.stances') }}</span>
          <textarea v-model="lists.stances" class="input min-h-[90px] text-sm" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.desk.persona.phrases') }}</span>
          <textarea v-model="lists.phrases" class="input min-h-[90px] text-sm" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.desk.persona.likes') }}</span>
          <textarea v-model="lists.likes" class="input min-h-[70px] text-sm" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.desk.persona.dislikes') }}</span>
          <textarea v-model="lists.dislikes" class="input min-h-[70px] text-sm" />
        </label>
      </div>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.persona.voice') }}</span>
        <textarea v-model="form.voice" class="input min-h-[120px] text-sm" maxlength="2000" :placeholder="t('admin.desk.persona.voicePlaceholder')" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.persona.taboos') }}</span>
        <textarea v-model="lists.taboos" class="input min-h-[70px] text-sm" :placeholder="t('admin.desk.persona.taboosPlaceholder')" />
      </label>
      <p class="text-xs text-gray-400">{{ t('admin.desk.persona.oneLine') }}</p>
      <button class="btn btn-primary" :disabled="saving" data-testid="persona-save" @click="save">{{ t('common.save') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { Persona } from '@/api/admin/content'

const { t } = useI18n()
const appStore = useAppStore()

const LIST_KEYS = ['experiences', 'stances', 'likes', 'dislikes', 'phrases', 'taboos'] as const
type ListKey = (typeof LIST_KEYS)[number]

const persona = ref<Persona>({ name: '', identity: '', experiences: [], stances: [], likes: [], dislikes: [], voice: '', phrases: [], taboos: [], samples: [] })
const form = reactive({ name: '', identity: '', voice: '' })
const lists = reactive<Record<ListKey, string>>({ experiences: '', stances: '', likes: '', dislikes: '', phrases: '', taboos: '' })
const proposed = ref(false)
const saving = ref(false)
const distilling = ref(false)
const adding = ref(false)
const sampleUrl = ref('')
const sampleTitle = ref('')
const sampleText = ref('')

function fill(p: Persona) {
  form.name = p.name || ''
  form.identity = p.identity || ''
  form.voice = p.voice || ''
  for (const k of LIST_KEYS) lists[k] = (p[k] || []).join('\n')
}

async function load() {
  try {
    persona.value = await contentAPI.getPersona()
    fill(persona.value)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
onMounted(load)

function lines(s: string) {
  return s.split('\n').map((x) => x.trim()).filter(Boolean)
}

async function save() {
  saving.value = true
  try {
    const out = await contentAPI.savePersona({
      ...persona.value,
      name: form.name,
      identity: form.identity,
      voice: form.voice,
      experiences: lines(lists.experiences),
      stances: lines(lists.stances),
      likes: lines(lists.likes),
      dislikes: lines(lists.dislikes),
      phrases: lines(lists.phrases),
      taboos: lines(lists.taboos)
    })
    persona.value = out
    proposed.value = false
    appStore.showSuccess(t('admin.desk.persona.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function distill() {
  distilling.value = true
  try {
    const p = await contentAPI.distillPersona()
    fill(p)
    proposed.value = true
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    distilling.value = false
  }
}

async function addSample(input: { url?: string; title?: string; text?: string }) {
  adding.value = true
  try {
    const p = await contentAPI.addSample(input)
    persona.value = { ...persona.value, samples: p.samples }
    sampleUrl.value = sampleTitle.value = sampleText.value = ''
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    adding.value = false
  }
}
const addByUrl = () => addSample({ url: sampleUrl.value })
const addByText = () => addSample({ title: sampleTitle.value, text: sampleText.value })

async function removeSample(id: string) {
  try {
    const p = await contentAPI.deleteSample(id)
    persona.value = { ...persona.value, samples: p.samples }
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>
