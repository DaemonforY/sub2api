<template>
  <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px]" data-testid="desk-editor">
    <!-- Editor -->
    <div class="card flex min-w-0 flex-col">
      <div class="flex flex-wrap items-center gap-1 border-b border-gray-100 px-3 py-2 dark:border-dark-700">
        <button v-for="b in toolbar" :key="b.key" type="button" :title="b.title" :class="['rounded px-2 py-1 text-sm', b.active() ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700']" @click="b.run()">
          {{ b.label }}
        </button>
        <span class="mx-1 h-4 w-px bg-gray-200 dark:bg-dark-600" />
        <button type="button" class="rounded px-2 py-1 text-sm text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700" @click="editor?.chain().focus().undo().run()">↶</button>
        <button type="button" class="rounded px-2 py-1 text-sm text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700" @click="editor?.chain().focus().redo().run()">↷</button>
        <div class="ml-auto flex items-center gap-2 text-xs text-gray-400">
          <span>{{ t('admin.desk.editor.count', { chars: stats.chars, paras: stats.paras }) }}</span>
          <span v-if="saveState === 'saving'">{{ t('admin.desk.editor.saving') }}</span>
          <span v-else-if="saveState === 'saved'">{{ t('admin.desk.editor.saved', { time: savedAt }) }}</span>
          <span v-else-if="saveState === 'error'" class="text-red-600">{{ t('admin.desk.editor.saveFailed') }}</span>
        </div>
      </div>

      <BubbleMenu v-if="editor" :editor="editor" :options="{ placement: 'top-start', offset: 8, flip: true, shift: { padding: 8 } }">
        <div class="flex max-w-[420px] flex-wrap gap-1 rounded-lg border border-gray-200 bg-white p-1 shadow-lg dark:border-dark-600 dark:bg-dark-800" data-testid="desk-bubble">
          <button v-for="a in ACTIONS" :key="a" type="button" class="rounded px-2 py-1 text-xs text-gray-700 hover:bg-primary-50 hover:text-primary-700 dark:text-dark-200 dark:hover:bg-primary-900/30" @mousedown.prevent="ask(a)">
            {{ t(`admin.desk.assist.${a}`) }}
          </button>
          <button type="button" class="rounded px-2 py-1 text-xs text-gray-700 hover:bg-primary-50 dark:text-dark-200 dark:hover:bg-primary-900/30" @mousedown.prevent="customOpen = true">{{ t('admin.desk.assist.custom') }}</button>
          <button type="button" class="rounded px-2 py-1 text-xs text-gray-700 hover:bg-primary-50 dark:text-dark-200 dark:hover:bg-primary-900/30" @mousedown.prevent="startComment">💬 {{ t('admin.desk.assist.comment') }}</button>
        </div>
      </BubbleMenu>

      <EditorContent :editor="editor" class="desk-prose min-h-[60vh] flex-1 px-5 py-4" />

      <div class="flex flex-wrap items-center gap-2 border-t border-gray-100 px-3 py-2 dark:border-dark-700">
        <button class="btn btn-secondary btn-sm" @click="snapshotOpen = !snapshotOpen">{{ t('admin.desk.editor.snapshot') }}</button>
        <div v-if="snapshotOpen" class="flex items-center gap-1">
          <input v-model.trim="snapshotLabel" class="input h-8 w-48 text-xs" :placeholder="t('admin.desk.editor.snapshotPlaceholder')" @keyup.enter="saveSnapshot" />
          <button class="btn btn-secondary btn-sm" @click="saveSnapshot">{{ t('admin.desk.common.ok') }}</button>
        </div>
        <button class="btn btn-secondary btn-sm" @click="emit('stage', 'drafts')">{{ t('admin.desk.editor.backDrafts') }}</button>
        <button class="btn btn-primary btn-sm ml-auto" :disabled="busy" data-testid="desk-finalize" @click="doFinalize">{{ t('admin.desk.editor.finalize') }}</button>
      </div>
    </div>

    <!-- Side panel (sticky next to the editor; the suggestion floats at the bottom on small screens) -->
    <div class="space-y-3 lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] lg:self-start lg:overflow-y-auto">
      <!-- An AI suggestion waiting for a decision -->
      <div
        v-if="pending"
        class="card fixed inset-x-2 bottom-2 z-30 max-h-[55vh] space-y-2 overflow-y-auto p-3 shadow-2xl ring-2 ring-primary-300 lg:static lg:inset-auto lg:max-h-none lg:shadow-none"
        data-testid="desk-suggestion"
      >
        <div class="flex items-center justify-between text-sm font-medium">
          <span>{{ t(`admin.desk.assist.${pending.action}`) }}</span>
          <button class="text-xs text-gray-400 hover:underline" @click="pending = null">{{ t('admin.desk.common.close') }}</button>
        </div>
        <p v-if="pending.loading" class="animate-pulse text-sm text-gray-500">{{ t('admin.desk.assist.thinking') }}</p>
        <template v-else-if="pending.result?.text">
          <div class="max-h-72 overflow-y-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-2 text-sm leading-relaxed dark:bg-dark-800">
            <span v-for="(p, i) in diffParts" :key="i" :class="p.kind === 'add' ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200' : p.kind === 'del' ? 'bg-red-100 text-red-700 line-through dark:bg-red-900/40 dark:text-red-300' : ''">{{ p.text }}</span>
          </div>
          <p v-if="pending.result.note" class="text-xs text-gray-500">{{ pending.result.note }}</p>
          <div class="flex gap-2">
            <button class="btn btn-primary btn-sm" data-testid="desk-accept" @click="accept(pending.result.text)">{{ t('admin.desk.assist.accept') }}</button>
            <button class="btn btn-secondary btn-sm" @click="retry">{{ t('admin.desk.assist.retry') }}</button>
            <button class="btn btn-secondary btn-sm" @click="pending = null">{{ t('admin.desk.assist.reject') }}</button>
          </div>
        </template>
        <template v-else-if="pending.result?.options?.length">
          <div v-for="(o, i) in pending.result.options" :key="i" class="space-y-1 rounded-lg bg-gray-50 p-2 text-sm dark:bg-dark-800">
            <div>{{ o.text }}</div>
            <div v-if="o.note" class="text-xs text-gray-500">{{ o.note }}</div>
            <a v-if="o.source" :href="o.source" target="_blank" rel="noopener" class="block truncate text-xs text-primary-600 hover:underline">{{ o.source_title || o.source }}</a>
            <div class="flex gap-2">
              <button v-if="pending.action === 'analogy'" class="text-xs text-primary-600 hover:underline" @click="insertAfter(o.text)">{{ t('admin.desk.assist.insertAfter') }}</button>
              <button v-if="pending.action === 'evidence'" class="text-xs text-primary-600 hover:underline" @click="keepFact(o)">{{ t('admin.desk.assist.keepFact') }}</button>
            </div>
          </div>
        </template>
        <p v-else class="text-sm text-gray-500">{{ t('admin.desk.assist.nothing') }}</p>
      </div>

      <div v-if="customOpen" class="card space-y-2 p-3">
        <div class="text-sm font-medium">{{ t('admin.desk.assist.custom') }}</div>
        <input v-model="customText" class="input" maxlength="500" :placeholder="t('admin.desk.assist.customPlaceholder')" @keyup.enter="askCustom" />
        <div class="flex gap-2">
          <button class="btn btn-primary btn-sm" :disabled="!customText.trim()" @click="askCustom">{{ t('admin.desk.common.ok') }}</button>
          <button class="btn btn-secondary btn-sm" @click="customOpen = false">{{ t('admin.desk.common.cancel') }}</button>
        </div>
      </div>

      <div v-if="commentDraft !== null" class="card space-y-2 p-3">
        <div class="text-xs text-gray-500">「{{ commentQuote }}」</div>
        <textarea v-model="commentDraft" class="input min-h-[64px] text-sm" :placeholder="t('admin.desk.comments.placeholder')" />
        <div class="flex gap-2">
          <button class="btn btn-primary btn-sm" :disabled="!commentDraft.trim()" @click="addComment">{{ t('admin.desk.common.ok') }}</button>
          <button class="btn btn-secondary btn-sm" @click="commentDraft = null">{{ t('admin.desk.common.cancel') }}</button>
        </div>
      </div>

      <div class="card p-3">
        <div class="mb-3 flex gap-1">
          <button v-for="tb in TABS" :key="tb" :class="['btn btn-sm flex-1', tab === tb ? 'btn-primary' : 'btn-secondary']" @click="tab = tb">
            {{ t(`admin.desk.side.${tb}`) }}<span v-if="tb === 'check' && openIssues.length" class="ml-1 text-xs">{{ openIssues.length }}</span>
            <span v-if="tb === 'comments' && openComments.length" class="ml-1 text-xs">{{ openComments.length }}</span>
          </button>
        </div>

        <!-- Check -->
        <div v-if="tab === 'check'" class="space-y-2" data-testid="desk-check">
          <button class="btn btn-secondary btn-sm w-full" :disabled="busy" @click="runCheck">{{ desk.check ? t('admin.desk.check.again') : t('admin.desk.check.run') }}</button>
          <p class="text-xs text-gray-500">{{ t('admin.desk.check.hint') }}</p>
          <template v-if="desk.check">
            <div class="grid grid-cols-4 gap-1 text-center text-xs">
              <div v-for="(v, k) in desk.check.scores || {}" :key="k" class="rounded bg-gray-50 py-1 dark:bg-dark-800">
                <div class="text-gray-500">{{ t(`admin.desk.check.score.${k}`) }}</div>
                <div class="font-semibold">{{ v }}</div>
              </div>
            </div>
            <p v-if="desk.check.summary" class="text-sm text-gray-700 dark:text-dark-200">{{ desk.check.summary }}</p>
            <p class="text-xs text-gray-400">
              {{ t('admin.desk.check.stats', { single: Math.round(desk.check.stats.single_ratio * 100), headings: desk.check.stats.headings, placeholders: desk.check.stats.placeholders }) }}
            </p>
            <ul class="space-y-2">
              <li v-for="is in openIssues" :key="is.id || is.problem" class="space-y-1 rounded-lg border border-gray-100 p-2 text-sm dark:border-dark-700">
                <div class="flex items-center gap-1 text-xs">
                  <span :class="['rounded px-1.5', is.level === 'must' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300']">{{ t(`admin.desk.check.level.${is.level}`) }}</span>
                  <span class="rounded bg-gray-100 px-1.5 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t(`admin.desk.check.layer.${is.layer}`) }}</span>
                </div>
                <div class="text-gray-800 dark:text-dark-100">{{ is.problem }}</div>
                <div v-if="is.quote" class="text-xs text-gray-500">「{{ is.quote }}」</div>
                <div v-if="is.fix" class="rounded bg-emerald-50 p-1.5 text-xs text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-200">→ {{ is.fix }}</div>
                <div class="flex gap-3 text-xs">
                  <button v-if="is.quote" class="text-primary-600 hover:underline" @click="locate(is.quote)">{{ t('admin.desk.check.locate') }}</button>
                  <button v-if="is.quote && is.fix" class="text-primary-600 hover:underline" @click="applyFix(is)">{{ t('admin.desk.check.apply') }}</button>
                  <button class="text-gray-400 hover:underline" @click="hidden.add(issueKey(is))">{{ t('admin.desk.check.ignore') }}</button>
                </div>
              </li>
            </ul>
          </template>
        </div>

        <!-- Fact cards -->
        <div v-else-if="tab === 'facts'" class="space-y-2">
          <p class="text-xs text-gray-500">{{ t('admin.desk.side.factsHint') }}</p>
          <ul class="max-h-[60vh] space-y-1.5 overflow-y-auto">
            <li v-for="f in usableFacts" :key="f.id" class="rounded bg-gray-50 p-2 text-xs dark:bg-dark-800">
              <span class="font-mono text-gray-400">{{ f.id }}</span> {{ f.text }}
              <a v-if="f.source?.startsWith('http')" :href="f.source" target="_blank" rel="noopener" class="block truncate text-primary-600 hover:underline">{{ f.source_title || f.source }}</a>
            </li>
          </ul>
        </div>

        <!-- Versions -->
        <div v-else-if="tab === 'versions'" class="space-y-2">
          <ul class="max-h-[40vh] space-y-1 overflow-y-auto">
            <li v-for="v in versions" :key="v.id">
              <button :class="['w-full rounded px-2 py-1.5 text-left text-xs', preview?.id === v.id ? 'bg-primary-50 dark:bg-primary-900/30' : 'hover:bg-gray-50 dark:hover:bg-dark-800']" @click="openVersion(v.id)">
                <span class="font-medium text-gray-800 dark:text-dark-100">{{ v.label || t(`admin.desk.versions.kind.${v.kind}`) }}</span>
                <span class="ml-1 text-gray-400">{{ fmtTime(v.created_at) }} · {{ v.chars }} {{ t('admin.desk.common.chars') }}</span>
              </button>
            </li>
          </ul>
          <p v-if="!versions.length" class="text-xs text-gray-400">{{ t('admin.desk.versions.empty') }}</p>
          <div v-if="preview" class="space-y-2">
            <div class="prose prose-sm max-h-[40vh] max-w-none overflow-y-auto rounded-lg border border-gray-100 p-2 dark:prose-invert dark:border-dark-700" v-html="renderMd(preview.markdown || '')" />
            <button class="btn btn-secondary btn-sm" @click="restore(preview.id)">{{ t('admin.desk.versions.restore') }}</button>
          </div>
        </div>

        <!-- Comments -->
        <div v-else class="space-y-2">
          <p class="text-xs text-gray-500">{{ t('admin.desk.comments.hint') }}</p>
          <ul class="space-y-2">
            <li v-for="c in desk.comments" :key="c.id" :class="['rounded-lg border border-gray-100 p-2 text-sm dark:border-dark-700', c.resolved ? 'opacity-50' : '']">
              <button v-if="c.quote" class="block text-left text-xs text-gray-500 hover:underline" @click="locate(c.quote)">「{{ c.quote }}」</button>
              <div class="text-gray-800 dark:text-dark-100">{{ c.text }}</div>
              <div class="mt-1 flex gap-3 text-xs">
                <button class="text-primary-600 hover:underline" @click="toggleComment(c.id)">{{ c.resolved ? t('admin.desk.comments.reopen') : t('admin.desk.comments.resolve') }}</button>
                <button class="text-gray-400 hover:underline" @click="removeComment(c.id)">{{ t('admin.desk.common.delete') }}</button>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import { BubbleMenu } from '@tiptap/vue-3/menus'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from '@tiptap/markdown'
import { Placeholder } from '@tiptap/extensions'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import { textDiff } from '@/utils/textDiff'
import { findText } from '@/utils/editorFind'
import * as contentAPI from '@/api/admin/content'
import type { AssistAction, AssistResult, Comment, ContentItem, DeskStage, Issue, TextVersion } from '@/api/admin/content'

const props = defineProps<{ item: ContentItem; busy: boolean }>()
const emit = defineEmits<{ (e: 'updated', it: ContentItem): void; (e: 'stage', s: DeskStage): void; (e: 'reload'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const ACTIONS: AssistAction[] = ['voice', 'tighten', 'expand', 'sharper', 'lighter', 'plain', 'analogy', 'evidence']
const TABS = ['check', 'facts', 'versions', 'comments'] as const
const tab = ref<(typeof TABS)[number]>('check')
const desk = computed(() => props.item.desk!)

// --- editor ---
const saveState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const savedAt = ref('')
const stats = reactive({ chars: 0, paras: 0 })
let saveTimer: ReturnType<typeof setTimeout> | null = null
let lastSaved = props.item.desk!.body

const editor = useEditor({
  content: props.item.desk!.body,
  contentType: 'markdown',
  extensions: [StarterKit, Markdown, Placeholder.configure({ placeholder: () => t('admin.desk.editor.placeholder') })],
  onCreate: ({ editor: e }) => countText(e.state.doc.textContent, e.state.doc.childCount),
  onUpdate: ({ editor: e }) => {
    countText(e.state.doc.textContent, e.state.doc.childCount)
    scheduleSave()
  }
})

function countText(text: string, paras: number) {
  stats.chars = text.replace(/\s/g, '').length
  stats.paras = paras
}

function currentMarkdown() {
  return editor.value?.getMarkdown() || ''
}

function scheduleSave() {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => save(), 2000)
}

async function save(label = ''): Promise<boolean> {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = null
  const body = currentMarkdown()
  if (!body.trim() || (body === lastSaved && !label)) return true
  saveState.value = 'saving'
  try {
    await contentAPI.saveBody(props.item.id, body, label)
    lastSaved = body
    saveState.value = 'saved'
    const d = new Date()
    savedAt.value = `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
    return true
  } catch (e) {
    saveState.value = 'error'
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
    return false
  }
}

onBeforeUnmount(() => {
  if (saveTimer) save()
  editor.value?.destroy()
})

const toolbar = computed(() => {
  const e = editor.value
  return [
    { key: 'h2', label: 'H2', title: t('admin.desk.editor.h2'), active: () => !!e?.isActive('heading', { level: 2 }), run: () => e?.chain().focus().toggleHeading({ level: 2 }).run() },
    { key: 'bold', label: 'B', title: t('admin.desk.editor.bold'), active: () => !!e?.isActive('bold'), run: () => e?.chain().focus().toggleBold().run() },
    { key: 'quote', label: '❝', title: t('admin.desk.editor.quote'), active: () => !!e?.isActive('blockquote'), run: () => e?.chain().focus().toggleBlockquote().run() },
    { key: 'list', label: '•', title: t('admin.desk.editor.list'), active: () => !!e?.isActive('bulletList'), run: () => e?.chain().focus().toggleBulletList().run() }
  ]
})

// --- selection assistant ---
interface Pending {
  action: AssistAction
  from: number
  to: number
  original: string
  before: string
  after: string
  instruction?: string
  loading: boolean
  result?: AssistResult
}
const pending = ref<Pending | null>(null)
const customOpen = ref(false)
const customText = ref('')

const diffParts = computed(() => (pending.value?.result?.text ? textDiff(pending.value.original, pending.value.result.text) : []))

function selectionContext() {
  const e = editor.value
  if (!e) return null
  const { from, to } = e.state.selection
  if (from === to) return null
  const doc = e.state.doc
  return {
    from,
    to,
    original: doc.textBetween(from, to, '\n\n'),
    before: doc.textBetween(Math.max(0, from - 800), from, '\n\n'),
    after: doc.textBetween(to, Math.min(doc.content.size, to + 800), '\n\n')
  }
}

// Each request has a number; a reply only lands if no newer request (or a close) came since.
let requestSeq = 0

async function run(p: Pending) {
  const seq = ++requestSeq
  pending.value = p
  try {
    const result = await contentAPI.assist(props.item.id, { action: p.action, selection: p.original, before: p.before, after: p.after, instruction: p.instruction })
    if (seq === requestSeq && pending.value) pending.value = { ...p, loading: false, result }
  } catch (e) {
    if (seq === requestSeq) pending.value = null
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function ask(action: AssistAction, instruction?: string) {
  const ctx = selectionContext()
  if (!ctx) return
  run({ action, ...ctx, instruction, loading: true })
}

function askCustom() {
  if (!customText.value.trim()) return
  ask('custom', customText.value.trim())
  customOpen.value = false
  customText.value = ''
}

function retry() {
  if (pending.value) run({ ...pending.value, loading: true, result: undefined })
}

// The range the suggestion was for, if the text there is unchanged; otherwise find it again.
function rangeFor(p: Pending) {
  const e = editor.value!
  const doc = e.state.doc
  if (p.to <= doc.content.size && doc.textBetween(p.from, p.to, '\n\n') === p.original) return { from: p.from, to: p.to }
  return findText(doc, p.original)
}

function replaceRange(range: { from: number; to: number }, text: string) {
  const e = editor.value!
  if (text.includes('\n')) {
    e.chain().focus().insertContentAt(range, text, { contentType: 'markdown' }).run()
  } else {
    e.chain().focus().command(({ tr }) => {
      tr.insertText(text, range.from, range.to)
      return true
    }).run()
  }
}

function accept(text: string) {
  const p = pending.value
  if (!p) return
  const range = rangeFor(p)
  if (!range) {
    appStore.showError(t('admin.desk.assist.lost'))
    return
  }
  replaceRange(range, text)
  pending.value = null
}

function insertAfter(text: string) {
  const p = pending.value
  if (!p) return
  const range = rangeFor(p)
  if (!range) return
  editor.value!.chain().focus().insertContentAt(range.to, ' ' + text).run()
}

async function keepFact(o: { text: string; source?: string; source_title?: string }) {
  try {
    emit('updated', await contentAPI.addFact(props.item.id, { text: o.text, source: o.source, source_title: o.source_title }))
    appStore.showSuccess(t('admin.desk.assist.factKept'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

// --- check ---
const hidden = reactive(new Set<string>())
const issueKey = (is: Issue) => is.id || is.quote + is.problem
const openIssues = computed(() => (desk.value.check?.issues || []).filter((is) => !hidden.has(issueKey(is))))
watch(() => desk.value.check?.at, () => hidden.clear())

async function runCheck() {
  if (!(await save())) return
  try {
    emit('updated', await contentAPI.runCheck(props.item.id))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function locate(quote: string) {
  const e = editor.value
  if (!e) return
  const range = findText(e.state.doc, quote)
  if (!range) {
    appStore.showError(t('admin.desk.check.notFound'))
    return
  }
  e.chain().focus().setTextSelection(range).scrollIntoView().run()
}

function applyFix(is: Issue) {
  const e = editor.value
  if (!e || !is.fix) return
  const range = findText(e.state.doc, is.quote)
  if (!range) {
    appStore.showError(t('admin.desk.check.notFound'))
    return
  }
  replaceRange(range, is.fix)
  hidden.add(issueKey(is))
}

const usableFacts = computed(() => desk.value.facts.filter((f) => f.status !== 'rejected'))

// --- versions ---
const versions = ref<TextVersion[]>([])
const preview = ref<TextVersion | null>(null)
const snapshotOpen = ref(false)
const snapshotLabel = ref('')

async function loadVersions() {
  try {
    versions.value = await contentAPI.listVersions(props.item.id)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
watch(tab, (v) => {
  if (v === 'versions') loadVersions()
})

async function openVersion(vid: number) {
  try {
    preview.value = await contentAPI.getVersion(props.item.id, vid)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function restore(vid: number) {
  if (!(await save())) return
  try {
    await contentAPI.restoreVersion(props.item.id, vid)
    emit('reload')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function saveSnapshot() {
  if (await save(snapshotLabel.value || t('admin.desk.versions.manual'))) {
    snapshotOpen.value = false
    snapshotLabel.value = ''
    appStore.showSuccess(t('admin.desk.versions.saved'))
    if (tab.value === 'versions') loadVersions()
  }
}

// --- comments ---
const commentDraft = ref<string | null>(null)
const commentQuote = ref('')
const openComments = computed(() => desk.value.comments.filter((c) => !c.resolved))

function startComment() {
  const ctx = selectionContext()
  if (!ctx) return
  commentQuote.value = ctx.original.slice(0, 200)
  commentDraft.value = ''
}

async function saveComments(list: Comment[]) {
  try {
    emit('updated', await contentAPI.saveComments(props.item.id, list))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function addComment() {
  if (!commentDraft.value?.trim()) return
  await saveComments([...desk.value.comments, { id: '', quote: commentQuote.value, text: commentDraft.value.trim(), at: '' }])
  commentDraft.value = null
  tab.value = 'comments'
}
const toggleComment = (id: string) => saveComments(desk.value.comments.map((c) => (c.id === id ? { ...c, resolved: !c.resolved } : c)))
const removeComment = (id: string) => saveComments(desk.value.comments.filter((c) => c.id !== id))

// --- final ---
async function doFinalize() {
  if (!(await save())) return
  try {
    emit('updated', await contentAPI.finalize(props.item.id))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

const renderMd = (md: string) => renderMarkdown(md)

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

defineExpose({ save })
</script>

<style scoped>
.desk-prose :deep(.ProseMirror) {
  outline: none;
  min-height: 60vh;
  font-size: 16px;
  line-height: 1.85;
}
.desk-prose :deep(.ProseMirror p) {
  margin: 0 0 1em;
}
.desk-prose :deep(.ProseMirror h1) {
  font-size: 1.6em;
  font-weight: 700;
  margin: 0.2em 0 0.8em;
}
.desk-prose :deep(.ProseMirror h2) {
  font-size: 1.25em;
  font-weight: 700;
  margin: 1.4em 0 0.6em;
}
.desk-prose :deep(.ProseMirror h3) {
  font-size: 1.1em;
  font-weight: 600;
  margin: 1.2em 0 0.5em;
}
.desk-prose :deep(.ProseMirror blockquote) {
  border-left: 3px solid rgb(209 213 219);
  padding-left: 0.9em;
  color: rgb(107 114 128);
  margin: 0 0 1em;
}
.desk-prose :deep(.ProseMirror ul),
.desk-prose :deep(.ProseMirror ol) {
  padding-left: 1.4em;
  margin: 0 0 1em;
}
.desk-prose :deep(.ProseMirror ul) {
  list-style: disc;
}
.desk-prose :deep(.ProseMirror ol) {
  list-style: decimal;
}
.desk-prose :deep(.ProseMirror pre) {
  background: rgb(31 41 55);
  color: rgb(243 244 246);
  border-radius: 8px;
  padding: 0.8em 1em;
  font-size: 0.85em;
  overflow-x: auto;
  margin: 0 0 1em;
}
.desk-prose :deep(.ProseMirror code) {
  font-size: 0.9em;
}
.desk-prose :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  color: rgb(156 163 175);
  float: left;
  height: 0;
  pointer-events: none;
}
</style>
