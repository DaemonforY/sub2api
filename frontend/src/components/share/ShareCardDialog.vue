<template>
  <BaseDialog :show="show" :title="input?.kind === 'quote' ? t('shareCard.quoteTitle') : t('shareCard.title')" width="narrow" @close="emit('close')">
    <div class="space-y-4" data-testid="share-dialog">
      <!-- An <img> so phones can long-press to save it. -->
      <div class="flex justify-center rounded-xl bg-gray-100 p-2.5 dark:bg-dark-900">
        <img v-if="images[theme]" :src="images[theme]" :alt="t('shareCard.title')" class="block max-h-[56vh] max-w-full rounded-lg" data-testid="share-image" @contextmenu="onLongPress" />
        <div v-else class="flex h-72 items-center justify-center text-sm text-gray-500 dark:text-dark-400">
          <span v-if="failed" class="px-6 text-center text-amber-600 dark:text-amber-400">{{ t('shareCard.failed') }}</span>
          <span v-else class="flex items-center gap-2"><Icon name="refresh" size="sm" class="animate-spin" />{{ t('shareCard.generating') }}</span>
        </div>
      </div>
      <p v-if="input?.truncated" class="text-xs text-gray-500 dark:text-dark-400">{{ t('shareCard.truncated', { n: QUOTE_MAX_CHARS }) }}</p>

      <div class="grid grid-cols-4 gap-2" role="radiogroup">
        <button
          v-for="id in SHARE_THEMES"
          :key="id"
          type="button"
          role="radio"
          :aria-checked="theme === id"
          :class="[
            'flex flex-col items-center gap-1 rounded-xl border px-1 py-1.5 text-xs transition',
            theme === id ? 'border-primary-500 font-semibold text-primary-700 ring-2 ring-primary-500/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 hover:border-primary-300 dark:border-dark-700 dark:text-dark-300'
          ]"
          @click="theme = id"
        >
          <span class="h-4 w-full rounded-md" :style="{ background: SWATCHES[id] }"></span>
          {{ t(`shareCard.themes.${id}`) }}
        </button>
      </div>

      <label v-if="userName" class="flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
        <input v-model="showName" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
        {{ t('shareCard.showName', { name: userName }) }}
      </label>

      <p v-if="isMobile" class="text-center text-sm font-semibold text-primary-700 dark:text-primary-300">{{ t('shareCard.longPress') }}</p>
      <div class="flex flex-wrap gap-2">
        <button v-if="!isMobile" type="button" class="btn btn-primary" :disabled="!images[theme]" data-testid="share-download" @click="download">
          <Icon name="download" size="sm" />
          <span>{{ t('shareCard.download') }}</span>
        </button>
        <button v-if="!isMobile" type="button" class="btn btn-secondary" :disabled="!images[theme]" @click="copyImage">
          <Icon name="copy" size="sm" />
          <span>{{ t('shareCard.copyImage') }}</span>
        </button>
        <button v-if="isMobile && canShareFiles" type="button" class="btn btn-primary" :disabled="!images[theme]" @click="shareFile">{{ t('shareCard.shareTo') }}</button>
        <button type="button" class="btn btn-secondary" @click="copyLink">{{ t('shareCard.copyLink') }}</button>
      </div>

      <p v-if="!authStore.isAuthenticated" class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('shareCard.loginHint') }}
        <RouterLink :to="{ path: '/login', query: { redirect: route.fullPath } }" class="text-primary-600 underline dark:text-primary-400">{{ t('shareCard.login') }}</RouterLink>
      </p>
      <p v-else-if="aff" class="text-xs text-gray-500 dark:text-dark-400">{{ t('shareCard.affHint') }}</p>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
// Share card dialog for main-site pages (course detail, model plaza): preview, colour themes,
// optional sharer name, save / copy / share. The card itself is drawn by utils/shareCard.
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { getAffiliateDetail } from '@/api/user'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { track } from '@/utils/analytics'
import { dataUrlToBlob, drawShareCard, QUOTE_MAX_CHARS, SHARE_THEMES, shareUrl, type ShareBlock, type ShareCard, type ShareTheme } from '@/utils/shareCard'

export interface ShareCardInput {
  kind: 'quote' | 'summary'
  brand: string
  title: string
  label?: string
  summary?: string
  points?: string[]
  blocks?: ShareBlock[]
  truncated?: boolean
  meta?: string
  cta?: string
  anchor?: string
}

const props = defineProps<{ show: boolean; input: ShareCardInput | null }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const SWATCHES: Record<ShareTheme, string> = {
  violet: 'linear-gradient(135deg,#4c1d95,#7c3aed,#c026d3)',
  honey: 'linear-gradient(135deg,#f59e0b,#f97316,#e11d48)',
  ocean: 'linear-gradient(135deg,#0f766e,#0891b2,#2563eb)',
  ink: 'linear-gradient(135deg,#0b1023,#1e1b4b,#3b0764)'
}
const THEME_KEY = 'share_card_theme'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const theme = ref<ShareTheme>('violet')
const showName = ref(false)
const images = reactive<Partial<Record<ShareTheme, string>>>({})
const failed = ref(false)
const aff = ref('')
const isMobile = ref(false)
const canShareFiles = ref(false)
const userName = computed(() => (authStore.isAuthenticated ? (authStore.user?.username || '').trim() : ''))

let cachedAff: { uid: number; code: string } | null = null
async function inviteCode(): Promise<string> {
  const uid = authStore.user?.id
  if (!authStore.isAuthenticated || !uid) return ''
  if (cachedAff?.uid === uid) return cachedAff.code
  try {
    const detail = await getAffiliateDetail()
    cachedAff = { uid, code: detail.aff_code || '' }
    return cachedAff.code
  } catch {
    return ''
  }
}

function pageLink(): string {
  const input = props.input
  return shareUrl(window.location.origin + window.location.pathname, {
    aff: aff.value,
    medium: input?.kind || 'summary',
    anchor: input?.kind === 'quote' ? input.anchor : ''
  })
}

function card(id: ShareTheme): ShareCard | null {
  const input = props.input
  if (!input) return null
  const base = {
    theme: id,
    brand: input.brand,
    label: input.label,
    title: input.title,
    url: pageLink(),
    meta: input.meta,
    cta: input.cta || t('shareCard.cta'),
    sharer: showName.value && userName.value ? t('shareCard.recommendedBy', { name: userName.value }) : '',
    footer: window.location.host
  }
  return input.kind === 'quote'
    ? { ...base, kind: 'quote', blocks: input.blocks || [] }
    : { ...base, kind: 'summary', summary: input.summary, points: input.points || [] }
}

let generation = 0
async function renderAll() {
  const run = ++generation
  failed.value = false
  for (const id of SHARE_THEMES) delete images[id]
  for (const id of [theme.value, ...SHARE_THEMES.filter((x) => x !== theme.value)]) {
    const c = card(id)
    if (!c) return
    try {
      const canvas = await drawShareCard(c)
      if (run !== generation) return
      images[id] = canvas.toDataURL('image/png')
    } catch {
      if (run === generation && id === theme.value) failed.value = true
    }
  }
}

async function open() {
  isMobile.value = window.matchMedia('(pointer: coarse)').matches || window.innerWidth < 768
  const saved = localStorage.getItem(THEME_KEY) as ShareTheme | null
  if (saved && SHARE_THEMES.includes(saved)) theme.value = saved
  try {
    canShareFiles.value = !!navigator.canShare?.({ files: [new File([''], 'a.png', { type: 'image/png' })] })
  } catch {
    canShareFiles.value = false
  }
  aff.value = await inviteCode()
  track('share_card_open', { kind: props.input?.kind || 'summary', chars: (props.input?.blocks || []).reduce((n, b) => n + b.text.length, 0) })
  await renderAll()
}

watch(
  () => props.show,
  (show) => {
    if (show) void open()
    else generation++
  },
  { immediate: true }
)
watch(theme, (id) => localStorage.setItem(THEME_KEY, id))
watch(showName, () => void renderAll())

function fileName() {
  return `${props.input?.brand || 'HiveGPT'}-${(props.input?.title || '').slice(0, 30)}.png`.replace(/[\\/:*?"<>|\s]+/g, '-')
}

function saved(method: string) {
  track('share_card_save', { kind: props.input?.kind || 'summary', theme: theme.value, method })
}

function download() {
  const url = images[theme.value]
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = fileName()
  document.body.appendChild(a)
  a.click()
  a.remove()
  saved('download')
}

function imageBlob(): Blob | null {
  const url = images[theme.value]
  return url ? dataUrlToBlob(url) : null
}

async function copyImage() {
  try {
    if (typeof ClipboardItem === 'undefined' || !navigator.clipboard?.write) throw new Error('unsupported')
    const blob = imageBlob()
    if (!blob) return
    await navigator.clipboard.write([new ClipboardItem({ [blob.type]: blob })])
    appStore.showSuccess(t('shareCard.copyImageDone'))
    saved('copy')
  } catch {
    appStore.showWarning(t('shareCard.copyImageUnsupported'))
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(pageLink())
    appStore.showSuccess(t('shareCard.linkCopied'))
    track('share_link_copy', { kind: props.input?.kind || 'summary' })
  } catch {
    // clipboard unavailable
  }
}

async function shareFile() {
  try {
    const blob = imageBlob()
    if (!blob) return
    await navigator.share({ files: [new File([blob], fileName(), { type: 'image/png' })], title: props.input?.title })
    saved('share')
  } catch {
    // cancelled
  }
}

function onLongPress() {
  if (isMobile.value) saved('longpress')
}
</script>
