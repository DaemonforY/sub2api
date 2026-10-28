<template>
  <div class="space-y-4">
    <div class="card space-y-2 p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span class="text-sm text-gray-700 dark:text-gray-200" data-testid="materials-seed-progress">
          {{ t('materials.imagery.progress', { done: candidates.filter((c) => c.in_library).length, total: candidates.length }) }}
        </span>
        <button class="btn btn-primary btn-sm" :disabled="drafting || !candidatesLeft" data-testid="materials-seed" @click="draftSeeds">
          {{ drafting ? t('materials.imagery.drafting') : t('materials.imagery.draft', { n: Math.min(30, candidatesLeft) }) }}
        </button>
      </div>
      <details class="text-xs text-gray-500">
        <summary class="cursor-pointer">{{ t('materials.imagery.candidates') }}</summary>
        <div class="mt-2 flex flex-wrap gap-1.5">
          <span
            v-for="c in candidates"
            :key="c.name"
            class="rounded px-1.5 py-0.5"
            :class="c.in_library ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
            :title="t('materials.imagery.candidateRank', { rank: c.rank, poems: c.poems })"
          >
            {{ c.name }}
          </span>
        </div>
      </details>
    </div>

    <div class="card flex flex-wrap items-center gap-2 p-3">
      <input v-model="q" class="input w-56 text-sm" :placeholder="t('materials.imagery.search')" @keyup.enter="reload" />
      <select v-model="status" class="input w-36 text-sm" data-testid="materials-imagery-status" @change="reload">
        <option value="">{{ t('materials.imagery.allStatuses') }}</option>
        <option v-for="s in statuses" :key="s" :value="s">{{ t(`materials.statuses.${s}`) }}</option>
      </select>
      <select v-model="category" class="input w-36 text-sm" @change="reload">
        <option value="">{{ t('materials.imagery.allCategories') }}</option>
        <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
      </select>
      <span class="text-xs text-gray-500">{{ t('materials.imagery.total', { n: total }) }}</span>
      <div v-if="selected.size" class="ml-auto flex items-center gap-2">
        <span class="text-xs text-gray-500">{{ t('materials.imagery.selected', { n: selected.size }) }}</span>
        <button class="btn btn-primary btn-sm" data-testid="materials-imagery-publish" @click="setStatus('published')">{{ t('materials.imagery.publish') }}</button>
        <button class="btn btn-secondary btn-sm" @click="setStatus('draft')">{{ t('materials.imagery.toDraft') }}</button>
        <button class="btn btn-secondary btn-sm" @click="setStatus('hidden')">{{ t('materials.imagery.hide') }}</button>
      </div>
    </div>

    <div v-if="!items.length && !loading" class="card p-10 text-center text-sm text-gray-400">{{ t('materials.imagery.empty') }}</div>
    <div v-else class="card divide-y divide-gray-100 dark:divide-dark-700">
      <label class="flex items-center gap-2 px-4 py-2 text-xs text-gray-500">
        <input type="checkbox" :checked="allSelected" @change="toggleAll" />
      </label>
      <div v-for="im in items" :key="im.id" class="flex gap-3 px-4 py-3" :data-testid="`materials-imagery-${im.name}`">
        <input v-model="selectedList" type="checkbox" class="mt-1" :value="im.id" />
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-baseline gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ im.name }}</span>
            <span class="text-xs text-gray-500">{{ im.category || '—' }}</span>
            <span v-if="im.aliases.length" class="text-xs text-gray-400">{{ im.aliases.join('、') }}</span>
            <span class="rounded px-1.5 text-[10px]" :class="statusClass(im.status)">{{ t(`materials.statuses.${im.status}`) }}</span>
            <span class="text-[10px] text-gray-400">{{ t('materials.imagery.segmentsUsed', { n: im.segment_count }) }}</span>
          </div>
          <p class="text-sm text-gray-700 dark:text-gray-200">{{ im.visual_zh }}</p>
          <p class="text-xs text-gray-500">{{ im.meaning }}</p>
          <p class="text-xs italic text-gray-400">{{ im.visual_en }}</p>
          <div class="flex flex-wrap gap-1">
            <span v-for="e in im.emotions" :key="e" class="rounded bg-violet-50 px-1.5 text-[10px] text-violet-700 dark:bg-violet-900/20 dark:text-violet-300">{{ e }}</span>
            <span v-for="s in [...im.seasons, ...im.times, ...im.weathers]" :key="s" class="rounded bg-gray-100 px-1.5 text-[10px] text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ s }}</span>
          </div>
        </div>
        <button class="self-start text-xs text-primary-600 hover:underline" :data-testid="`materials-imagery-edit-${im.name}`" @click="openEdit(im)">
          {{ t('materials.imagery.edit') }}
        </button>
      </div>
    </div>
    <button v-if="items.length < total" class="btn btn-secondary btn-sm w-full" :disabled="loading" @click="more">…</button>

    <BaseDialog :show="!!editing" :title="t('materials.imagery.edit')" width="wide" @close="editing = null">
      <form v-if="editing" class="grid gap-3 sm:grid-cols-2" data-testid="materials-imagery-form" @submit.prevent="saveEdit">
        <label class="text-xs text-gray-500">{{ t('materials.imagery.fields.name') }}<input v-model="editing.name" class="input mt-1 text-sm" maxlength="8" required /></label>
        <label class="text-xs text-gray-500">
          {{ t('materials.imagery.fields.category') }}
          <select v-model="editing.category" class="input mt-1 text-sm">
            <option value="">—</option>
            <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
          </select>
        </label>
        <label class="text-xs text-gray-500 sm:col-span-2">{{ t('materials.imagery.fields.aliases') }}<input v-model="editing.aliases" class="input mt-1 text-sm" /></label>
        <label class="text-xs text-gray-500 sm:col-span-2">{{ t('materials.imagery.fields.meaning') }}<textarea v-model="editing.meaning" rows="2" class="input mt-1 text-sm" /></label>
        <label class="text-xs text-gray-500 sm:col-span-2">{{ t('materials.imagery.fields.visualZh') }}<textarea v-model="editing.visual_zh" rows="2" class="input mt-1 text-sm" /></label>
        <label class="text-xs text-gray-500 sm:col-span-2">
          {{ t('materials.imagery.fields.visualEn') }}
          <textarea v-model="editing.visual_en" rows="3" class="input mt-1 text-sm" />
          <span class="text-[11px] text-amber-600">{{ t('materials.imagery.visualEnHint') }}</span>
        </label>
        <label class="text-xs text-gray-500">{{ t('materials.imagery.fields.emotions') }}<input v-model="editing.emotions" class="input mt-1 text-sm" /></label>
        <label class="text-xs text-gray-500">{{ t('materials.imagery.fields.eras') }}<input v-model="editing.eras" class="input mt-1 text-sm" /></label>
        <fieldset v-for="group in checkGroups" :key="group.key" class="text-xs text-gray-500">
          <legend>{{ t(`materials.imagery.fields.${group.key}`) }}</legend>
          <label v-for="v in group.values" :key="v" class="mr-3 inline-flex items-center gap-1">
            <input v-model="editing[group.key]" type="checkbox" :value="v" />{{ v }}
          </label>
        </fieldset>
        <fieldset class="text-xs text-gray-500 sm:col-span-2">
          <legend>{{ t('materials.imagery.fields.styleIds') }}</legend>
          <label v-for="s in styles" :key="s.id" class="mr-3 inline-flex items-center gap-1">
            <input v-model="editing.style_ids" type="checkbox" :value="s.id" />{{ s.name }}
          </label>
        </fieldset>
        <label class="text-xs text-gray-500">
          {{ t('materials.imagery.fields.status') }}
          <select v-model="editing.status" class="input mt-1 text-sm">
            <option v-for="s in statuses" :key="s" :value="s">{{ t(`materials.statuses.${s}`) }}</option>
          </select>
        </label>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="editing = null">{{ t('materials.cancel') }}</button>
        <button class="btn btn-primary" :disabled="savingEdit" data-testid="materials-imagery-save" @click="saveEdit">{{ t('materials.save') }}</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'

// The edit form keeps the free-text lists as comma-separated strings.
interface EditForm {
  id: number
  name: string
  category: string
  aliases: string
  meaning: string
  visual_zh: string
  visual_en: string
  emotions: string
  eras: string
  seasons: string[]
  times: string[]
  weathers: string[]
  style_ids: number[]
  status: api.MaterialStatus
}

const { t } = useI18n()
const appStore = useAppStore()
const statuses = ['draft', 'review', 'published', 'hidden'] as const
const checkGroups = [
  { key: 'seasons', values: ['春', '夏', '秋', '冬'] },
  { key: 'times', values: ['清晨', '白天', '黄昏', '夜晚'] },
  { key: 'weathers', values: ['晴', '雨', '雪', '雾', '风'] }
] as const
const PAGE = 100

const q = ref('')
const status = ref('')
const category = ref('')
const categories = ref<string[]>(['天象', '气候', '地理', '植物', '动物', '器物', '建筑', '人物动作'])
const items = ref<api.MaterialImagery[]>([])
const total = ref(0)
const loading = ref(false)
const selectedList = ref<number[]>([])
const selected = computed(() => new Set(selectedList.value))
const allSelected = computed(() => items.value.length > 0 && items.value.every((im) => selected.value.has(im.id)))
const candidates = ref<api.MaterialImageryCandidate[]>([])
const candidatesLeft = computed(() => candidates.value.filter((c) => !c.in_library).length)
const drafting = ref(false)
const styles = ref<api.MaterialStyle[]>([])
const editing = ref<EditForm | null>(null)
const savingEdit = ref(false)

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

function statusClass(s: api.MaterialStatus) {
  return {
    draft: 'bg-amber-50 text-amber-700',
    review: 'bg-sky-50 text-sky-700',
    published: 'bg-emerald-50 text-emerald-700',
    hidden: 'bg-gray-100 text-gray-500'
  }[s]
}

async function fetchPage(offset: number) {
  loading.value = true
  try {
    const page = await api.listImagery({ q: q.value.trim() || undefined, status: status.value || undefined, category: category.value || undefined, limit: PAGE, offset })
    items.value = offset ? [...items.value, ...page.items] : page.items
    total.value = page.total
    if (page.categories?.length) categories.value = page.categories
    // Keep ticks made while the list was loading; drop only entries no longer listed.
    const listed = new Set(items.value.map((im) => im.id))
    selectedList.value = selectedList.value.filter((id) => listed.has(id))
  } catch (e) {
    fail(e)
  } finally {
    loading.value = false
  }
}
const reload = () => fetchPage(0)
const more = () => fetchPage(items.value.length)

async function loadCandidates() {
  try {
    candidates.value = await api.listCandidates()
  } catch (e) {
    fail(e)
  }
}

function toggleAll() {
  selectedList.value = allSelected.value ? [] : items.value.map((im) => im.id)
}

async function setStatus(s: api.MaterialStatus) {
  try {
    const n = await api.setImageryStatus([...selected.value], s)
    appStore.showSuccess(t('materials.imagery.updated', { n }))
    await reload()
  } catch (e) {
    fail(e)
  }
}

async function draftSeeds() {
  drafting.value = true
  try {
    const res = await api.draftSeeds(30)
    appStore.showSuccess(t('materials.imagery.drafted', { n: res.drafted.length, failed: res.failed.length, left: res.left }))
    await Promise.all([loadCandidates(), reload()])
  } catch (e) {
    fail(e)
  } finally {
    drafting.value = false
  }
}

const splitList = (s: string) =>
  s
    .split(/[,，、\s]+/)
    .map((x) => x.trim())
    .filter(Boolean)

function openEdit(im: api.MaterialImagery) {
  editing.value = {
    id: im.id,
    name: im.name,
    category: im.category,
    aliases: im.aliases.join('，'),
    meaning: im.meaning,
    visual_zh: im.visual_zh,
    visual_en: im.visual_en,
    emotions: im.emotions.join('，'),
    eras: im.eras.join('，'),
    seasons: [...im.seasons],
    times: [...im.times],
    weathers: [...im.weathers],
    style_ids: [...im.style_ids],
    status: im.status
  }
  if (!styles.value.length) void api.listStyles().then((s) => (styles.value = s), fail)
}

async function saveEdit() {
  const f = editing.value
  if (!f) return
  savingEdit.value = true
  try {
    await api.updateImagery(f.id, {
      name: f.name.trim(),
      category: f.category,
      aliases: splitList(f.aliases),
      meaning: f.meaning,
      visual_zh: f.visual_zh,
      visual_en: f.visual_en,
      emotions: splitList(f.emotions),
      eras: splitList(f.eras),
      seasons: f.seasons,
      times: f.times,
      weathers: f.weathers,
      style_ids: f.style_ids,
      status: f.status
    })
    editing.value = null
    await reload()
  } catch (e) {
    fail(e)
  } finally {
    savingEdit.value = false
  }
}

onMounted(() => {
  void reload()
  void loadCandidates()
})
</script>
