<template>
  <BaseDialog :show="show" :title="songId ? t('inspiration.editor.titleEdit') : t('inspiration.editor.titleNew')" width="extra-wide" :close-on-click-outside="false" @close="emit('close')">
    <p v-if="loading" class="py-10 text-center text-sm text-gray-400">{{ t('inspiration.loading') }}</p>
    <div v-else class="grid gap-4 lg:grid-cols-2">
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-3">
          <label class="col-span-2 text-xs text-gray-500">{{ t('inspiration.editor.name') }}<input v-model="form.title" class="input mt-1" data-testid="song-title" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.artist') }}<input v-model="form.artist" class="input mt-1" data-testid="song-artist" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.year') }}<input v-model.number="year" type="number" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.lyricist') }}<input v-model="form.lyricist" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.composer') }}<input v-model="form.composer" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.album') }}<input v-model="form.album" class="input mt-1" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.link') }}<input v-model="form.link_url" class="input mt-1" placeholder="https://" /></label>
        </div>
        <label class="block text-xs text-gray-500">
          🔒 {{ t('inspiration.editor.lyrics') }}
          <textarea v-model="form.lyrics" rows="16" class="input mt-1 font-serif text-sm leading-6" data-testid="song-lyrics" />
          <span class="mt-1 block text-amber-600">{{ t('inspiration.editor.lyricsHint') }}</span>
        </label>
      </div>

      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="!songId || analyzing || !form.lyrics?.trim()" data-testid="song-analyze" @click="analyze">
            ✨ {{ analyzing ? t('inspiration.editor.analyzing') : t('inspiration.editor.analyze') }}
          </button>
          <span v-if="!songId" class="text-xs text-gray-400">{{ t('inspiration.editor.analyzeSaveFirst') }}</span>
        </div>
        <p v-if="dropped" class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">{{ t('inspiration.editor.dropped') }}</p>
        <label class="block text-xs text-gray-500">{{ t('inspiration.editor.summary') }}<textarea v-model="form.summary" rows="3" class="input mt-1 text-sm" data-testid="song-summary" /></label>
        <label class="block text-xs text-gray-500">{{ t('inspiration.editor.imagery') }}<input v-model="lists.imagery" class="input mt-1" :placeholder="t('inspiration.editor.listHint')" data-testid="song-imagery" /></label>
        <div class="grid grid-cols-2 gap-3">
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.moods') }}<input v-model="lists.moods" class="input mt-1" :placeholder="t('inspiration.editor.listHint')" /></label>
          <label class="text-xs text-gray-500">{{ t('inspiration.editor.tags') }}<input v-model="lists.tags" class="input mt-1" :placeholder="t('inspiration.editor.listHint')" /></label>
        </div>
        <div class="text-xs text-gray-500">
          <div class="mb-1 flex items-center justify-between">
            {{ t('inspiration.editor.allusions') }}
            <button class="text-primary-600 hover:underline" @click="form.allusions.push({ title: '', author: '', note: '' })">＋ {{ t('inspiration.editor.addAllusion') }}</button>
          </div>
          <div v-for="(a, i) in form.allusions" :key="i" class="mb-2 flex flex-wrap items-center gap-1.5">
            <input v-model="a.title" class="input w-36 py-1 text-sm" :placeholder="t('inspiration.editor.allusionTitle')" />
            <input v-model="a.author" class="input w-20 py-1 text-sm" :placeholder="t('inspiration.editor.allusionAuthor')" />
            <input v-model="a.note" class="input min-w-[8rem] flex-1 py-1 text-sm" :placeholder="t('inspiration.editor.allusionNote')" />
            <button v-if="a.title" class="text-primary-600 hover:underline" :title="t('inspiration.songs.openPoem')" @click="emit('search-poem', `${a.title} ${a.author}`.trim())">↗</button>
            <button class="text-gray-400 hover:text-red-500" @click="form.allusions.splice(i, 1)">✕</button>
          </div>
        </div>
        <label class="block text-xs text-gray-500">{{ t('inspiration.editor.notes') }}<textarea v-model="form.notes" rows="4" class="input mt-1 text-sm" /></label>
      </div>
    </div>

    <template #footer>
      <button v-if="songId" class="btn btn-danger mr-auto" data-testid="song-delete" @click="confirmDelete = true">{{ t('inspiration.editor.delete') }}</button>
      <button class="btn btn-secondary" @click="emit('close')">{{ t('inspiration.editor.cancel') }}</button>
      <button class="btn btn-primary" :disabled="saving || !form.title.trim()" data-testid="song-save" @click="save">{{ t('inspiration.editor.save') }}</button>
    </template>
  </BaseDialog>
  <ConfirmDialog
    :show="confirmDelete"
    :title="t('inspiration.editor.delete')"
    :message="t('inspiration.editor.deleteConfirm')"
    danger
    @confirm="remove"
    @cancel="confirmDelete = false"
  />
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'

const props = defineProps<{ show: boolean; songId: number | null }>()
const emit = defineEmits<{ close: []; saved: [song: api.LibSong]; deleted: []; 'search-poem': [q: string] }>()
const { t } = useI18n()
const appStore = useAppStore()

const blank = (): api.LibSongInput => ({
  title: '', artist: '', lyricist: '', composer: '', year: null, album: '', lyrics: '', summary: '',
  imagery: [], moods: [], tags: [], allusions: [], notes: '', link_url: ''
})
const form = reactive<api.LibSongInput>(blank())
const lists = reactive({ imagery: '', moods: '', tags: '' })
const year = ref<number | ''>('')
const loading = ref(false)
const saving = ref(false)
const analyzing = ref(false)
const dropped = ref(false)
const confirmDelete = ref(false)

const splitList = (s: string) => s.split(/[,，、;；\n]+/).map((x) => x.trim()).filter(Boolean)
const joinList = (a: string[]) => a.join('、')

function fill(s: api.LibSongInput) {
  Object.assign(form, blank(), s, { allusions: (s.allusions ?? []).map((a) => ({ ...a })) })
  lists.imagery = joinList(s.imagery ?? [])
  lists.moods = joinList(s.moods ?? [])
  lists.tags = joinList(s.tags ?? [])
  year.value = s.year ?? ''
}

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

watch(
  () => [props.show, props.songId] as const,
  async ([show, id], old) => {
    if (!show) return
    // Saving a new song sets songId while the dialog is open: keep the form as it is.
    if (old && old[0] && old[1] === null && id !== null) return
    dropped.value = false
    if (id === null) {
      fill(blank())
      return
    }
    loading.value = true
    try {
      fill(await api.getSong(id))
    } catch (e) {
      fail(e)
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

function input(): api.LibSongInput {
  return {
    ...form,
    year: typeof year.value === 'number' && year.value > 0 ? year.value : null,
    imagery: splitList(lists.imagery),
    moods: splitList(lists.moods),
    tags: splitList(lists.tags),
    allusions: form.allusions.filter((a) => a.title.trim())
  }
}

async function save() {
  saving.value = true
  try {
    const song = props.songId ? await api.updateSong(props.songId, input()) : await api.createSong(input())
    appStore.showSuccess(t('inspiration.editor.saved'))
    emit('saved', song)
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

async function analyze() {
  if (!props.songId) return
  analyzing.value = true
  try {
    // Analyze what is saved: save the pasted lyrics first.
    await api.updateSong(props.songId, input())
    const a = await api.analyzeSong(props.songId)
    if (a.summary) form.summary = a.summary
    if (a.imagery.length) lists.imagery = joinList(a.imagery)
    if (a.moods.length) lists.moods = joinList(a.moods)
    if (a.tags.length) lists.tags = joinList(a.tags)
    if (a.allusions.length) form.allusions = a.allusions
    dropped.value = a.dropped.includes('summary')
    appStore.showSuccess(t('inspiration.editor.analyzed'))
  } catch (e) {
    fail(e)
  } finally {
    analyzing.value = false
  }
}

async function remove() {
  confirmDelete.value = false
  if (!props.songId) return
  try {
    await api.deleteSong(props.songId)
    appStore.showSuccess(t('inspiration.editor.deleted'))
    emit('deleted')
  } catch (e) {
    fail(e)
  }
}
</script>
