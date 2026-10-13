<template>
  <div class="space-y-4" :data-testid="`content-version-${version.platform}`">
    <div class="flex flex-wrap items-center gap-2">
      <span :class="['rounded px-2 py-0.5 text-xs font-medium', statusClass(version.status)]">{{ t(`admin.content.versionStatus.${version.status}`) }}</span>
      <span v-if="version.kind === 'answer'" class="rounded bg-sky-50 px-2 py-0.5 text-xs text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">{{ t('admin.content.version.answerBadge') }}</span>
      <span v-if="version.edited" class="text-xs text-gray-400">{{ t('admin.content.version.edited') }}</span>
      <span v-if="version.feedback" class="text-xs text-gray-400">· {{ version.feedback }}</span>
    </div>

    <p v-if="version.status === 'failed'" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
      {{ t('admin.content.version.failed', { msg: version.error || '—' }) }}
    </p>
    <p v-if="version.status === 'generating'" class="rounded-lg bg-primary-50 p-3 text-sm text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
      {{ t('admin.content.version.regenerating') }}
    </p>

    <!-- Checks -->
    <div v-if="version.checks?.length || editable" class="space-y-1">
      <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.content.version.checks') }}</div>
      <p v-if="!version.checks?.length" class="text-xs text-emerald-600">{{ t('admin.content.version.noChecks') }}</p>
      <ul class="space-y-1">
        <li v-for="(c, i) in version.checks" :key="i" :class="['rounded px-2 py-1 text-xs', checkClass(c.level)]">{{ c.text }}</li>
      </ul>
    </div>

    <!-- Editor (before approval) -->
    <template v-if="editable">
      <div class="grid gap-3">
        <label class="text-sm">
          <span class="input-label">{{ t('admin.content.version.title') }} <span class="text-xs text-gray-400">{{ form.title.length }} / {{ spec?.title_max }}</span></span>
          <input v-model="form.title" class="input" data-testid="content-version-title" />
        </label>
        <label v-if="version.kind === 'answer'" class="text-sm">
          <span class="input-label">{{ t('admin.content.version.question') }}</span>
          <input v-model="form.question" class="input" />
          <span class="input-hint">{{ t('admin.content.version.questionHint') }}</span>
        </label>
        <label v-if="version.platform === 'wechat'" class="text-sm">
          <span class="input-label">{{ t('admin.content.version.digest') }}</span>
          <input v-model="form.digest" class="input" maxlength="120" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.content.version.body') }} <span class="text-xs text-gray-400">{{ plainLength(form.body) }} / {{ spec?.body_max }}</span></span>
          <textarea v-model="form.body" class="input min-h-[320px] font-mono text-xs leading-relaxed" data-testid="content-version-body" />
          <span class="input-hint">{{ bodyHint }}</span>
        </label>
        <label v-if="spec?.micro_max" class="text-sm">
          <span class="input-label">{{ t('admin.content.version.micro') }} <span class="text-xs text-gray-400">{{ form.micro.length }} / {{ spec?.micro_max }}</span></span>
          <textarea v-model="form.micro" class="input min-h-[96px] text-sm" />
        </label>
        <div class="grid gap-3 sm:grid-cols-2">
          <label v-if="spec?.tags_max" class="text-sm">
            <span class="input-label">{{ t('admin.content.version.tags') }}</span>
            <input v-model="tagsText" class="input" />
            <span class="input-hint">{{ t('admin.content.version.tagsHint') }}</span>
          </label>
          <label v-if="version.platform === 'csdn'" class="text-sm">
            <span class="input-label">{{ t('admin.content.version.category') }}</span>
            <input v-model="form.category" class="input" />
          </label>
        </div>
        <div v-if="spec?.cards" class="space-y-2">
          <div class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.content.version.cards') }}</div>
          <div v-for="(card, i) in form.cards" :key="i" class="grid gap-2 rounded-lg border border-gray-100 p-2 sm:grid-cols-[1fr_2fr_auto] dark:border-dark-700">
            <input v-model="card.title" class="input" :placeholder="t('admin.content.version.cardTitle')" />
            <textarea
              :value="card.points.join('\n')"
              class="input min-h-[64px] text-sm"
              :placeholder="t('admin.content.version.cardPoints')"
              @input="card.points = ($event.target as HTMLTextAreaElement).value.split('\n')"
            />
            <button type="button" class="btn btn-secondary btn-sm" @click="form.cards.splice(i, 1)">{{ t('admin.content.version.removeCard') }}</button>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="form.cards.push({ title: '', points: [] })">{{ t('admin.content.version.addCard') }}</button>
        </div>
      </div>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-primary" :disabled="busy" data-testid="content-version-approve" @click="approve">{{ t('admin.content.version.approve') }}</button>
        <button class="btn btn-secondary" :disabled="busy || !dirty" @click="save">{{ t('admin.content.version.save') }}</button>
        <button class="btn btn-secondary" :disabled="busy" @click="regenOpen = true">{{ t('admin.content.version.regenerate') }}</button>
        <button v-if="version.status !== 'skipped'" class="btn btn-secondary" :disabled="busy" @click="skip">{{ t('admin.content.version.skip') }}</button>
      </div>
      <div v-if="regenOpen" class="space-y-2 rounded-lg border border-gray-100 p-3 dark:border-dark-700">
        <div class="text-sm font-medium">{{ t('admin.content.version.regenerateTitle') }}</div>
        <input v-model="feedback" class="input" maxlength="500" :placeholder="t('admin.content.version.feedbackPlaceholder')" />
        <div class="flex gap-2">
          <button class="btn btn-primary btn-sm" :disabled="busy" @click="regenerate">{{ t('admin.content.version.regenerate') }}</button>
          <button class="btn btn-secondary btn-sm" @click="regenOpen = false">{{ t('admin.content.common.cancel') }}</button>
        </div>
      </div>
    </template>

    <!-- Failed: only regenerate / skip -->
    <div v-else-if="version.status === 'failed'" class="flex flex-wrap gap-2">
      <button class="btn btn-primary" :disabled="busy" @click="regenerate">{{ t('admin.content.version.regenerate') }}</button>
      <button class="btn btn-secondary" :disabled="busy" @click="skip">{{ t('admin.content.version.skip') }}</button>
    </div>

    <!-- Publish package (after approval) -->
    <template v-if="version.status === 'approved' || version.status === 'published'">
      <div class="space-y-3 rounded-xl border border-primary-100 bg-primary-50/40 p-4 dark:border-primary-900/40 dark:bg-primary-900/10" data-testid="content-publish">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.content.publish.title') }}</h4>
          <a v-if="publishUrl" :href="publishUrl" target="_blank" rel="noopener" class="btn btn-secondary btn-sm">{{ t('admin.content.publish.openEditor') }} ↗</a>
        </div>
        <p class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
          {{ t('admin.content.publish.aiLabel', { label: spec?.ai_label || '' }) }}
        </p>

        <div v-if="version.kind === 'answer' && version.question" class="space-y-1 text-sm">
          <div class="text-xs text-gray-500">{{ t('admin.content.version.question') }}</div>
          <div class="font-medium">{{ version.question }}</div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary btn-sm" @click="copy(version.question)">{{ t('admin.content.publish.copyQuestion') }}</button>
            <a :href="`https://www.zhihu.com/search?type=content&q=${encodeURIComponent(version.question)}`" target="_blank" rel="noopener" class="btn btn-secondary btn-sm">
              {{ t('admin.content.publish.searchQuestion') }} ↗
            </a>
          </div>
        </div>

        <div class="space-y-1">
          <div class="text-xs text-gray-500">{{ t('admin.content.version.title') }}</div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ version.title }}</span>
            <button class="btn btn-secondary btn-sm" @click="copy(version.title)">{{ t('admin.content.publish.copyTitle') }}</button>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <button v-if="richPlatform" class="btn btn-primary btn-sm" :title="t('admin.content.publish.copyRichHint')" @click="copyRich">{{ t('admin.content.publish.copyRich') }}</button>
          <button class="btn btn-secondary btn-sm" @click="copy(rendered)">{{ version.platform === 'csdn' ? t('admin.content.publish.copyMarkdown') : t('admin.content.publish.copyBody') }}</button>
          <button v-if="version.rendered_micro" class="btn btn-secondary btn-sm" @click="copy(version.rendered_micro)">{{ t('admin.content.publish.copyMicro') }}</button>
          <button v-if="version.tags?.length" class="btn btn-secondary btn-sm" @click="copy(tagLine)">{{ t('admin.content.publish.copyTags') }}</button>
        </div>
        <p v-if="richPlatform" class="text-xs text-gray-500">{{ t('admin.content.publish.copyRichHint') }}</p>

        <div v-if="version.link_url" class="text-sm">
          <span class="text-xs text-gray-500">{{ spec?.read_more ? t('admin.content.publish.readMore') : spec?.links ? t('admin.content.publish.link') : t('admin.content.publish.bioLink') }}：</span>
          <code class="break-all text-xs">{{ version.link_url }}</code>
          <button class="ml-2 text-xs text-primary-600 hover:underline" @click="copy(version.link_url)">{{ t('admin.content.publish.copy') }}</button>
        </div>
        <div v-if="version.promo_code" class="text-sm">
          <span class="text-xs text-gray-500">{{ t('admin.content.publish.promo') }}：</span>
          <code class="font-semibold">{{ version.promo_code }}</code>
          <span class="ml-2 text-xs text-gray-500">{{ t('admin.content.publish.promoHint') }}</span>
        </div>

        <!-- 小红书 cards -->
        <div v-if="spec?.cards && version.cards?.length" class="space-y-2">
          <div class="flex flex-wrap items-center gap-2 text-xs">
            <span class="text-gray-500">{{ t('admin.content.publish.cardTheme') }}</span>
            <button
              v-for="th in NOTE_THEMES"
              :key="th"
              type="button"
              :class="['rounded px-2 py-0.5', theme === th ? 'bg-primary-600 text-white' : 'bg-white text-gray-600 dark:bg-dark-800 dark:text-dark-300']"
              @click="theme = th"
            >{{ NOTE_THEME_NAMES[th] }}</button>
            <button class="btn btn-secondary btn-sm" @click="downloadCards">{{ t('admin.content.publish.downloadCards') }}</button>
          </div>
          <div class="flex gap-2 overflow-x-auto pb-1">
            <img v-for="(src, i) in cardImages" :key="i" :src="src" class="h-48 flex-shrink-0 rounded-lg border border-gray-200 dark:border-dark-700" alt="" />
          </div>
        </div>

        <details class="text-sm">
          <summary class="cursor-pointer text-xs text-gray-500">{{ t('admin.content.version.body') }}</summary>
          <div class="prose prose-sm mt-2 max-w-none dark:prose-invert" v-html="renderedHtml" />
        </details>

        <div class="space-y-1">
          <label class="input-label">{{ t('admin.content.publish.postUrl') }}</label>
          <div class="flex flex-wrap gap-2">
            <input v-model="postUrl" class="input min-w-0 flex-1" :placeholder="t('admin.content.publish.postUrlPlaceholder')" data-testid="content-post-url" />
            <button class="btn btn-primary" :disabled="busy || !postUrl.trim()" @click="record">{{ t('admin.content.publish.record') }}</button>
          </div>
          <p v-if="version.published_at" class="text-xs text-emerald-600">
            {{ t('admin.content.publish.publishedAt', { time: fmtTime(version.published_at) }) }} ·
            <a :href="version.post_url" target="_blank" rel="noopener" class="hover:underline">{{ version.post_url }}</a>
          </p>
        </div>
      </div>
    </template>

    <!-- Skipped -->
    <div v-if="version.status === 'skipped'" class="flex gap-2">
      <button class="btn btn-secondary btn-sm" :disabled="busy" @click="approve">{{ t('admin.content.version.approve') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'
import { copyRichText } from '@/utils/richCopy'
import { drawNoteCard, NOTE_THEMES, NOTE_THEME_NAMES, type NoteTheme } from '@/utils/noteCards'
import * as contentAPI from '@/api/admin/content'
import type { ContentCheck, ContentVersion, NoteCard, PlatformSpec, VersionStatus } from '@/api/admin/content'

const props = defineProps<{ version: ContentVersion; spec?: PlatformSpec; brand: string }>()
const emit = defineEmits<{ (e: 'updated', v: ContentVersion): void }>()

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const busy = ref(false)
const regenOpen = ref(false)
const feedback = ref('')
const postUrl = ref(props.version.post_url || '')
const theme = ref<NoteTheme>('cream')
const cardImages = ref<string[]>([])

const editable = computed(() => props.version.status === 'pending_review' || props.version.status === 'skipped')
const richPlatform = computed(() => ['wechat', 'zhihu', 'toutiao'].includes(props.version.platform))
const rendered = computed(() => props.version.rendered || props.version.body)
const renderedHtml = computed(() => renderMarkdown(rendered.value))
const tagLine = computed(() => (props.version.tags || []).map((x) => (props.version.platform === 'xiaohongshu' ? `#${x}` : x)).join(' '))
const publishUrl = computed(() => props.spec?.publish_url || '')
// (Built here: "{{" inside a template interpolation would end it.)
const bodyHint = computed(() => t('admin.content.version.bodyHint', { link: '{{link}}', promo: '{{promo}}' }))

interface Form {
  title: string
  body: string
  question: string
  digest: string
  micro: string
  category: string
  cards: NoteCard[]
}
const form = reactive<Form>({ title: '', body: '', question: '', digest: '', micro: '', category: '', cards: [] })
const tagsText = ref('')

function load(v: ContentVersion) {
  Object.assign(form, {
    title: v.title,
    body: v.body,
    question: v.question || '',
    digest: v.digest || '',
    micro: v.micro || '',
    category: v.category || '',
    cards: (v.cards || []).map((c) => ({ title: c.title, points: [...c.points] }))
  })
  tagsText.value = (v.tags || []).join(', ')
  postUrl.value = v.post_url || ''
}
watch(() => props.version, load, { immediate: true })

const dirty = computed(() => {
  const v = props.version
  return (
    form.title !== v.title ||
    form.body !== v.body ||
    form.question !== (v.question || '') ||
    form.digest !== (v.digest || '') ||
    form.micro !== (v.micro || '') ||
    form.category !== (v.category || '') ||
    tagsText.value !== (v.tags || []).join(', ') ||
    JSON.stringify(form.cards) !== JSON.stringify(v.cards || [])
  )
})

function input(): contentAPI.VersionInput {
  return {
    title: form.title,
    body: form.body,
    question: form.question,
    digest: form.digest,
    micro: form.micro,
    category: form.category,
    tags: tagsText.value.split(/[,，\s]+/).filter(Boolean),
    cards: form.cards.map((c) => ({ title: c.title, points: c.points.map((p) => p.trim()).filter(Boolean) }))
  }
}

async function act(fn: () => Promise<ContentVersion>, ok?: string) {
  busy.value = true
  try {
    const v = await fn()
    emit('updated', v)
    if (ok) appStore.showSuccess(ok)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

async function save() {
  await act(() => contentAPI.updateVersion(props.version.id, input()), t('admin.content.version.saved'))
}

async function approve() {
  await act(async () => {
    if (dirty.value) await contentAPI.updateVersion(props.version.id, input())
    return contentAPI.approveVersion(props.version.id)
  }, t('admin.content.version.approved'))
}

async function skip() {
  await act(() => contentAPI.skipVersion(props.version.id))
}

async function regenerate() {
  regenOpen.value = false
  await act(() => contentAPI.regenerateVersion(props.version.id, feedback.value.trim()))
  feedback.value = ''
}

async function record() {
  await act(() => contentAPI.publishVersion(props.version.id, postUrl.value.trim()), t('admin.content.publish.recorded'))
}

function copy(text?: string) {
  if (text) copyToClipboard(text, t('admin.content.publish.copied'))
}

async function copyRich() {
  const ok = await copyRichText(renderedHtml.value, rendered.value)
  if (ok) appStore.showSuccess(t('admin.content.publish.copied'))
  else copy(rendered.value)
}

// 小红书 cards are drawn in the browser whenever the cards or the colours change.
async function drawCards() {
  const cards = props.version.cards || []
  if (!props.spec?.cards || !cards.length) {
    cardImages.value = []
    return
  }
  const out: string[] = []
  for (let i = 0; i < cards.length; i++) {
    try {
      const canvas = await drawNoteCard(cards[i], i, cards.length, { theme: theme.value, brand: props.brand })
      out.push(canvas.toDataURL('image/png'))
    } catch {
      return
    }
  }
  cardImages.value = out
}
watch(() => [props.version.cards, props.version.status, theme.value], drawCards, { immediate: true, deep: true })

function downloadCards() {
  cardImages.value.forEach((src, i) => {
    const a = document.createElement('a')
    a.href = src
    a.download = `xhs-${props.version.id}-${i + 1}.png`
    document.body.appendChild(a)
    a.click()
    a.remove()
  })
}

function plainLength(md: string) {
  return md.replace(/^#{1,6}\s+|[*_`>|]/gm, '').replace(/\s/g, '').length
}

function statusClass(s: VersionStatus) {
  switch (s) {
    case 'pending_review':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
    case 'approved':
      return 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
    case 'published':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
    case 'failed':
      return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

function checkClass(level: ContentCheck['level']) {
  if (level === 'error') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  if (level === 'warn') return 'bg-amber-50 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300'
  return 'bg-gray-50 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>
