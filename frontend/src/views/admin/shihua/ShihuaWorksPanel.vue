<template>
  <div class="space-y-4">
    <div class="card p-4">
      <div class="flex flex-wrap items-end gap-3">
        <input v-model="filters.q" class="input w-56" :placeholder="t('shihua.works.search')" />
        <select v-model="filters.stage" class="input w-32">
          <option value="">{{ t('shihua.stages.all') }}</option>
          <option v-for="s in stages" :key="s" :value="s">{{ t(`shihua.stages.${s}`) }}</option>
        </select>
        <input v-model="filters.grade" class="input w-36" :placeholder="t('shihua.works.grade')" />
        <select v-model="filters.status" class="input w-36">
          <option value="">{{ t('shihua.textStatus.all') }}</option>
          <option v-for="s in statuses" :key="s" :value="s">{{ t(`shihua.textStatus.${s}`) }}</option>
        </select>
        <div class="ml-auto flex flex-wrap gap-2">
          <button class="btn btn-primary" data-testid="shihua-ai-open" @click="aiOpen = true">{{ t('shihua.ai.button') }}</button>
          <button class="btn btn-secondary" data-testid="shihua-import" @click="openImport">{{ t('shihua.works.import') }}</button>
          <button class="btn btn-secondary" @click="openEdit(null)">{{ t('shihua.works.add') }}</button>
        </div>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-3 border-t border-gray-100 pt-3 text-sm dark:border-dark-700">
        <span class="text-gray-500">{{ t('shihua.works.selected', { n: selected.size }) }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="!selected.size || themeBusy" data-testid="shihua-theme-selected" @click="themeSelected">
          {{ themeBusy ? t('shihua.works.themeProgress', themeProgress) : t('shihua.works.themeSelected') }}
        </button>
        <label class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
          {{ t('shihua.works.candidates') }}
          <select v-model.number="candidates" class="input w-16 py-1">
            <option v-for="n in 4" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
        <button class="btn btn-primary btn-sm" :disabled="!selected.size || drawing" data-testid="shihua-draw-selected" @click="drawSelected">{{ t('shihua.works.drawSelected') }}</button>
        <span class="text-xs text-gray-400">{{ t('shihua.works.verifyHint') }}</span>
      </div>
    </div>

    <div class="card overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
          <tr>
            <th class="px-3 py-3"><input type="checkbox" :checked="allSelected" @change="toggleAll" /></th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.title') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.grade') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.chars') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.text') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.theme') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.images') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.picked') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.renders') }}</th>
            <th class="px-3 py-3 font-medium">{{ t('shihua.works.columns.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-if="!filtered.length">
            <td colspan="10" class="px-3 py-10 text-center text-gray-400">{{ loading ? t('common.loading') : t('shihua.works.empty') }}</td>
          </tr>
          <tr v-for="w in filtered" :key="w.id" class="align-top" :data-testid="`shihua-work-${w.slug}`">
            <td class="px-3 py-3"><input type="checkbox" :checked="selected.has(w.id)" @change="toggle(w.id)" /></td>
            <td class="px-3 py-3">
              <div class="font-medium text-gray-900 dark:text-white">{{ w.title }}</div>
              <div class="text-xs text-gray-500">{{ w.dynasty }} {{ w.author }} · {{ t(`shihua.genres.${w.genre}`) }}</div>
            </td>
            <td class="whitespace-nowrap px-3 py-3 text-gray-600 dark:text-gray-300">{{ w.grade }}</td>
            <td class="whitespace-nowrap px-3 py-3 text-gray-600 dark:text-gray-300">{{ w.display_char_count }} / {{ w.char_count }}</td>
            <td class="whitespace-nowrap px-3 py-3">
              <span class="rounded px-1.5 py-0.5 text-xs" :class="statusClass(w.text_status)">{{ t(`shihua.textStatus.${w.text_status}`) }}</span>
              <div class="mt-1 flex gap-2 text-xs">
                <button v-if="w.text_status === 'draft'" class="text-primary-600 hover:underline" @click="verify(w, 1)">{{ t('shihua.works.verify1') }}</button>
                <button v-if="w.text_status === 'verified_1'" class="text-primary-600 hover:underline" @click="verify(w, 2)">{{ t('shihua.works.verify2') }}</button>
                <button v-if="w.text_status !== 'draft'" class="text-gray-400 hover:underline" @click="verify(w, 0)">{{ t('shihua.works.unverify') }}</button>
              </div>
            </td>
            <td class="max-w-xs px-3 py-3">
              <p v-if="w.theme.prompt_theme" class="line-clamp-3 text-xs text-gray-600 dark:text-gray-300" :title="w.theme.prompt_theme">{{ w.theme.prompt_theme }}</p>
              <p v-else class="text-xs text-gray-400">{{ t('shihua.works.noTheme') }}</p>
              <button class="mt-1 text-xs text-primary-600 hover:underline disabled:opacity-50" :disabled="busy.has(w.id)" @click="generate(w)">
                {{ busy.has(w.id) ? t('shihua.works.generating') : w.theme.prompt_theme ? t('shihua.works.regenerate') : t('shihua.works.generate') }}
              </button>
            </td>
            <td class="whitespace-nowrap px-3 py-3 text-xs text-gray-600 dark:text-gray-300">
              {{ t('shihua.works.imagesCount', { ok: w.images.succeeded, queued: w.images.queued + w.images.running, failed: w.images.failed }) }}
            </td>
            <td class="px-3 py-3 text-center">{{ w.picked_image_id ? '✓' : '' }}</td>
            <td class="px-3 py-3 text-center">{{ w.renders || '' }}</td>
            <td class="whitespace-nowrap px-3 py-3 text-xs">
              <button class="mr-2 text-primary-600 hover:underline" @click="openEdit(w)">{{ t('shihua.works.edit') }}</button>
              <button class="text-red-500 hover:underline" @click="remove(w)">{{ t('shihua.works.delete') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <BaseDialog :show="!!editing" :title="editingId ? t('shihua.edit.titleEdit', { title: form.title }) : t('shihua.edit.titleNew')" width="extra-wide" @close="editing = false">
      <div class="grid gap-3 md:grid-cols-4">
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.slug') }}</span><input v-model="form.slug" class="input" /></label>
        <label><span class="input-label">{{ t('shihua.edit.title') }}</span><input v-model="form.title" class="input" /></label>
        <label><span class="input-label">{{ t('shihua.edit.author') }}</span><input v-model="form.author" class="input" /></label>
        <label><span class="input-label">{{ t('shihua.edit.dynasty') }}</span><input v-model="form.dynasty" class="input" /></label>
        <label>
          <span class="input-label">{{ t('shihua.edit.genre') }}</span>
          <select v-model="form.genre" class="input"><option v-for="g in genres" :key="g" :value="g">{{ t(`shihua.genres.${g}`) }}</option></select>
        </label>
        <label>
          <span class="input-label">{{ t('shihua.edit.stage') }}</span>
          <select v-model="form.stage" class="input"><option v-for="s in stages" :key="s" :value="s">{{ t(`shihua.stages.${s}`) }}</option></select>
        </label>
        <label><span class="input-label">{{ t('shihua.edit.grade') }}</span><input v-model="form.grade" class="input" /></label>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.fullText') }}</span><textarea v-model="form.full_text" rows="8" class="input font-serif" /></label>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.displayText') }}</span><textarea v-model="form.display_text" rows="8" class="input font-serif" /></label>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.famousLines') }}</span><textarea v-model="form.famous_lines" rows="2" class="input" /></label>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.excerptNote') }}</span><textarea v-model="form.excerpt_note" rows="2" class="input" /></label>
        <label class="md:col-span-4"><span class="input-label">{{ t('shihua.edit.promptTheme') }}</span><textarea v-model="form.prompt_theme" rows="4" class="input" /></label>
        <div v-if="themeNotes.length" class="rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300 md:col-span-4">
          <div class="mb-1 font-medium">{{ t('shihua.edit.themeNotes') }}</div>
          <div v-for="line in themeNotes" :key="line">{{ line }}</div>
        </div>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.textbooks') }}</span><textarea v-model="form.textbooks" rows="3" class="input font-mono text-xs" /></label>
        <label class="md:col-span-2"><span class="input-label">{{ t('shihua.edit.sources') }}</span><textarea v-model="form.sources" rows="3" class="input" /></label>
        <label class="md:col-span-3"><span class="input-label">{{ t('shihua.edit.variantNote') }}</span><input v-model="form.variant_note" class="input" /></label>
        <label><span class="input-label">{{ t('shihua.edit.sort') }}</span><input v-model.number="form.sort" type="number" class="input" /></label>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="editing = false">{{ t('shihua.edit.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving" data-testid="shihua-save-work" @click="save">{{ t('shihua.edit.save') }}</button>
      </template>
    </BaseDialog>

    <ShihuaAiDraftDialog :show="aiOpen" @close="aiOpen = false" @imported="onAiImported" />

    <BaseDialog :show="importing" :title="t('shihua.import.title')" width="wide" @close="importing = false">
      <p class="text-sm text-gray-500">{{ t('shihua.import.hint') }}</p>
      <input ref="fileInput" type="file" accept=".json,application/json" class="mt-3 text-sm" data-testid="shihua-import-file" @change="readImport" />
      <div v-if="preview" class="mt-4 space-y-2 text-sm">
        <p class="font-medium">{{ t('shihua.import.preview', { created: preview.created, updated: preview.updated, unchanged: preview.unchanged }) }}</p>
        <p v-if="preview.reset.length" class="text-amber-600">{{ t('shihua.import.reset', { slugs: preview.reset.join('、') }) }}</p>
        <div v-if="preview.errors.length" class="text-red-600">
          <p>{{ t('shihua.import.errors', { n: preview.errors.length }) }}</p>
          <p v-for="e in preview.errors" :key="e" class="text-xs">{{ e }}</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="importing = false">{{ t('shihua.edit.cancel') }}</button>
        <button class="btn btn-primary" :disabled="!pending.length || saving" data-testid="shihua-import-confirm" @click="confirmImport">{{ t('shihua.import.confirm') }}</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/shihua'
import ShihuaAiDraftDialog from './ShihuaAiDraftDialog.vue'

const props = defineProps<{ works: api.ShihuaWork[]; loading: boolean }>()
const emit = defineEmits<{ (e: 'reload'): void; (e: 'refresh-queue'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const stages = ['primary', 'junior', 'senior', 'legacy'] as const
const statuses = ['draft', 'verified_1', 'verified_2'] as const
const genres = ['shi', 'ci', 'qu', 'wen', 'fu'] as const
const filters = reactive({ q: '', stage: '', grade: '', status: '' })
const selected = ref(new Set<number>())
const candidates = ref(2)
const drawing = ref(false)
const busy = ref(new Set<number>())
const themeBusy = ref(false)
const themeProgress = reactive({ done: 0, total: 0 })

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

const filtered = computed(() => {
  const q = filters.q.trim()
  return props.works.filter(
    (w) =>
      (!filters.stage || w.stage === filters.stage) &&
      (!filters.grade || w.grade.includes(filters.grade.trim())) &&
      (!filters.status || w.text_status === filters.status) &&
      (!q || w.title.includes(q) || w.author.includes(q) || w.slug.includes(q) || w.full_text.includes(q))
  )
})
const allSelected = computed(() => filtered.value.length > 0 && filtered.value.every((w) => selected.value.has(w.id)))
function toggle(id: number) {
  const s = new Set(selected.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selected.value = s
}
function toggleAll() {
  selected.value = allSelected.value ? new Set() : new Set(filtered.value.map((w) => w.id))
}
function statusClass(s: api.ShihuaTextStatus) {
  if (s === 'verified_2') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (s === 'verified_1') return 'bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

async function verify(w: api.ShihuaWork, level: 0 | 1 | 2) {
  try {
    await api.verifyWork(w.id, level)
    emit('reload')
  } catch (e) {
    fail(e)
  }
}

async function generate(w: api.ShihuaWork) {
  busy.value = new Set(busy.value).add(w.id)
  try {
    await api.generateTheme(w.id)
    emit('reload')
  } catch (e) {
    fail(e)
  } finally {
    const s = new Set(busy.value)
    s.delete(w.id)
    busy.value = s
  }
}

/** Drafts scenes one after another (two at a time) for the selected works that have none yet. */
async function themeSelected() {
  const todo = props.works.filter((w) => selected.value.has(w.id) && !w.theme.prompt_theme)
  if (!todo.length) return
  themeBusy.value = true
  Object.assign(themeProgress, { done: 0, total: todo.length })
  const queue = [...todo]
  const worker = async () => {
    for (let w = queue.shift(); w; w = queue.shift()) {
      busy.value = new Set(busy.value).add(w.id)
      try {
        await api.generateTheme(w.id)
      } catch (e) {
        fail(e)
      }
      const s = new Set(busy.value)
      s.delete(w.id)
      busy.value = s
      themeProgress.done++
    }
  }
  await Promise.all([worker(), worker()])
  themeBusy.value = false
  emit('reload')
}

async function drawSelected() {
  drawing.value = true
  try {
    const ids = [...selected.value]
    const job = await api.createJob(ids, candidates.value)
    appStore.showSuccess(t('shihua.works.jobCreated', { n: job.counts.total }))
    emit('reload')
    emit('refresh-queue')
  } catch (e) {
    fail(e)
  } finally {
    drawing.value = false
  }
}

async function remove(w: api.ShihuaWork) {
  if (!confirm(t('shihua.works.confirmDelete', { title: w.title }))) return
  try {
    await api.deleteWork(w.id)
    appStore.showSuccess(t('shihua.works.deleted'))
    emit('reload')
  } catch (e) {
    fail(e)
  }
}

// ---- edit ----
const editing = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const theme = ref<api.ShihuaTheme>({ prompt_theme: '' })
const form = reactive({
  slug: '', title: '', author: '', dynasty: '', genre: 'shi', stage: 'junior', grade: '', textbooks: '[]',
  full_text: '', display_text: '', excerpt_note: '', famous_lines: '', sources: '', variant_note: '', sort: 0, prompt_theme: ''
})
const themeNotes = computed(() => {
  const th = theme.value
  const lines: string[] = []
  if (th.subject) lines.push(`主体：${th.subject}`)
  if (th.scene) lines.push(`场景：${th.scene}`)
  if (th.season || th.time) lines.push(`时令：${[th.season, th.time].filter(Boolean).join('，')}`)
  if (th.figures) lines.push(`人物：${th.figures}`)
  if (th.key_imagery?.length) lines.push(`意象：${th.key_imagery.join('；')}`)
  if (th.palette) lines.push(`色调：${th.palette}`)
  if (th.avoid?.length) lines.push(`避免：${th.avoid.join('、')}`)
  if (th.model) lines.push(`模型：${th.model}`)
  return lines
})

function openEdit(w: api.ShihuaWork | null) {
  editingId.value = w?.id ?? null
  theme.value = w ? { ...w.theme } : { prompt_theme: '' }
  Object.assign(form, {
    slug: w?.slug ?? '', title: w?.title ?? '', author: w?.author ?? '', dynasty: w?.dynasty ?? '', genre: w?.genre ?? 'shi',
    stage: w?.stage ?? 'junior', grade: w?.grade ?? '', textbooks: JSON.stringify(w?.textbooks ?? [], null, 1),
    full_text: w?.full_text ?? '', display_text: w?.display_text ?? '', excerpt_note: w?.excerpt_note ?? '', famous_lines: w?.famous_lines ?? '',
    sources: w?.sources ?? '', variant_note: w?.variant_note ?? '', sort: w?.sort ?? 0, prompt_theme: w?.theme.prompt_theme ?? ''
  })
  editing.value = true
}

async function save() {
  let textbooks: api.ShihuaTextbook[]
  try {
    textbooks = JSON.parse(form.textbooks || '[]')
    if (!Array.isArray(textbooks)) throw new Error()
  } catch {
    appStore.showError(t('shihua.edit.badTextbooks'))
    return
  }
  const { prompt_theme, textbooks: _raw, ...rest } = form
  const input: api.ShihuaWorkInput = { ...rest, textbooks, theme: { ...theme.value, prompt_theme } }
  saving.value = true
  try {
    if (editingId.value) await api.updateWork(editingId.value, input)
    else await api.createWork(input)
    appStore.showSuccess(t('shihua.works.saved'))
    editing.value = false
    emit('reload')
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

// ---- AI 添加篇目: after an import, select the new works so 「为所选生成画面描述」 is one click ----
const aiOpen = ref(false)
const pendingSelect = ref<string[]>([])
function onAiImported(slugs: string[]) {
  pendingSelect.value = slugs
  emit('reload')
}
watch(
  () => props.works,
  (works) => {
    if (!pendingSelect.value.length) return
    const ids = works.filter((w) => pendingSelect.value.includes(w.slug)).map((w) => w.id)
    if (ids.length) {
      selected.value = new Set(ids)
      pendingSelect.value = []
    }
  }
)

// ---- import ----
const importing = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const pending = ref<api.ShihuaWorkInput[]>([])
const preview = ref<api.ShihuaImportResult | null>(null)

function openImport() {
  pending.value = []
  preview.value = null
  if (fileInput.value) fileInput.value.value = ''
  importing.value = true
}

async function readImport(ev: Event) {
  const file = (ev.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const parsed = JSON.parse(await file.text())
    const works = Array.isArray(parsed) ? parsed : parsed.works
    if (!Array.isArray(works)) throw new Error()
    pending.value = works
  } catch {
    appStore.showError(t('shihua.import.badFile'))
    return
  }
  try {
    preview.value = await api.importWorks(pending.value, true)
  } catch (e) {
    fail(e)
  }
}

async function confirmImport() {
  saving.value = true
  try {
    const res = await api.importWorks(pending.value, false)
    appStore.showSuccess(t('shihua.import.done', { created: res.created, updated: res.updated }))
    importing.value = false
    emit('reload')
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}
</script>
