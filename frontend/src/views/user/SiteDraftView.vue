<template>
  <AppLayout>
    <div v-if="draft" class="flex flex-col gap-4 lg:h-[calc(100dvh-8rem)] lg:flex-row">
      <!-- 左：需求、修改记录、修改输入 -->
      <div class="card flex min-h-0 flex-col p-4 lg:w-[24rem] lg:shrink-0">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <router-link to="/sites/ai" class="text-xs text-primary-600 hover:underline">← {{ t('siteBuilder.backToList') }}</router-link>
            <h2 class="mt-1 truncate font-semibold text-gray-900 dark:text-white">{{ draft.title || t('siteBuilder.title') }}</h2>
          </div>
          <button class="shrink-0 text-xs text-gray-400 hover:text-red-500" :disabled="active" @click="confirmDelete = true">{{ t('common.delete') }}</button>
        </div>

        <div class="mt-3 flex-1 space-y-2 overflow-y-auto text-sm" data-testid="site-draft-turns">
          <div v-for="(turn, i) in draft.turns" :key="i" class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <div class="text-[11px] text-gray-400">{{ i === 0 ? t('siteBuilder.firstTurn') : formatDateTime(turn.at) }}</div>
            <div class="whitespace-pre-wrap break-words text-gray-800 dark:text-dark-100">{{ turn.instruction }}</div>
            <div v-if="turn.status === 'failed'" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ t('siteBuilder.failedTurn', { error: turn.error }) }}</div>
          </div>
          <div v-if="active || lastEvent" :class="['flex items-center gap-2 px-1 text-xs', lastEvent?.kind === 'error' ? 'text-red-600' : 'text-gray-500 dark:text-dark-400']" data-testid="site-draft-progress">
            <span v-if="active" class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>
            {{ active ? t(`siteBuilder.status.${draft.status}`) + ' · ' + (lastEvent?.text || '') : lastEvent?.text }}
          </div>
        </div>

        <div v-if="draft.error && !active" class="mt-2 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="site-draft-error">{{ draft.error }}</div>

        <div class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
          <div class="flex flex-wrap gap-2">
            <button v-if="active" class="btn btn-secondary btn-sm" data-testid="site-draft-stop" @click="stop">{{ t('siteBuilder.stop') }}</button>
            <button v-if="!active && draft.status === 'failed'" class="btn btn-primary btn-sm" :disabled="busy" data-testid="site-draft-retry" @click="retry">{{ t('siteBuilder.retry') }}</button>
            <button v-if="!active && draft.html && missingImages" class="btn btn-secondary btn-sm" :disabled="busy" data-testid="site-draft-redraw" @click="retry">{{ t('siteBuilder.redraw') }}</button>
            <button v-if="draft.can_undo" class="btn btn-secondary btn-sm" :disabled="busy" data-testid="site-draft-undo" @click="undo">{{ t('siteBuilder.undo') }}</button>
          </div>
          <template v-if="draft.html">
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="chip in chips"
                :key="chip"
                type="button"
                class="rounded-full border border-gray-200 px-2.5 py-0.5 text-xs text-gray-600 hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-dark-300"
                :disabled="active"
                @click="instruction = chip"
              >
                {{ chip }}
              </button>
            </div>
            <textarea
              v-model="instruction"
              rows="3"
              maxlength="1000"
              class="input resize-none"
              :placeholder="t('siteBuilder.revisePlaceholder')"
              :disabled="active"
              data-testid="site-draft-instruction"
              @keydown.enter.exact.prevent="revise"
            ></textarea>
            <div class="flex items-center justify-between gap-2">
              <span class="text-[11px] text-gray-400">{{ t('siteBuilder.tokens', { tokens: (draft.prompt_tokens + draft.completion_tokens).toLocaleString(), images: draft.images_drawn }) }}</span>
              <button class="btn btn-primary btn-sm" :disabled="active || busy || !instruction.trim()" data-testid="site-draft-revise" @click="revise">{{ t('siteBuilder.send') }}</button>
            </div>
          </template>
        </div>
      </div>

      <!-- 右：预览 -->
      <div class="card flex min-h-[32rem] flex-1 flex-col overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 px-4 py-2 dark:border-dark-700">
          <div class="flex rounded-lg bg-gray-100 p-0.5 text-xs dark:bg-dark-800">
            <button
              v-for="d in (['desktop', 'mobile'] as const)"
              :key="d"
              :class="['rounded-md px-3 py-1', device === d ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white' : 'text-gray-500']"
              @click="device = d"
            >
              {{ t(`siteBuilder.device.${d}`) }}
            </button>
          </div>
          <div class="flex items-center gap-2">
            <a v-if="draft.site_url" :href="draft.site_url" target="_blank" rel="noopener" class="hidden max-w-[14rem] truncate sm:inline text-xs text-emerald-600 hover:underline">{{ draft.site_url }}</a>
            <button class="btn btn-secondary btn-sm whitespace-nowrap" :disabled="!pageSource" @click="openNew">{{ t('siteBuilder.openNew') }}</button>
            <button class="btn btn-primary btn-sm whitespace-nowrap" :disabled="active || !draft.html" data-testid="site-draft-publish" @click="openPublish">{{ t('siteBuilder.publish') }}</button>
          </div>
        </div>
        <div class="relative flex flex-1 justify-center overflow-auto bg-gray-100 dark:bg-dark-900">
          <div
            v-if="active && draft.html"
            class="absolute left-1/2 top-3 z-10 flex -translate-x-1/2 items-center gap-2 rounded-full bg-gray-900/80 px-3 py-1 text-xs text-white"
            data-testid="site-draft-revising"
          >
            <span class="h-2 w-2 animate-pulse rounded-full bg-primary-400"></span>{{ t(`siteBuilder.status.${draft.status}`) }}
          </div>
          <iframe
            v-if="pageSource"
            :srcdoc="srcdoc"
            sandbox="allow-scripts allow-popups"
            :class="['h-full min-h-[30rem] border-0 bg-white', device === 'mobile' ? 'my-3 w-[390px] rounded-xl shadow' : 'w-full']"
            title="preview"
            data-testid="site-draft-preview"
          ></iframe>
          <div v-else class="m-auto px-6 text-center text-sm text-gray-400">{{ active ? t('siteBuilder.previewWriting') : t('siteBuilder.previewEmpty') }}</div>
        </div>
      </div>
    </div>

    <BaseDialog :show="publishOpen" :title="t('siteBuilder.publishTitle')" width="normal" @close="publishOpen = false">
      <div v-if="sites" class="space-y-4 text-sm" data-testid="site-draft-publish-dialog">
        <p v-if="!sites.quota.available" class="text-gray-600 dark:text-dark-300">{{ t('siteBuilder.unavailable') }}</p>
        <div v-else-if="!sites.quota.subscribed" class="space-y-3">
          <p class="text-amber-700 dark:text-amber-300">{{ t('siteBuilder.subscribeFirst') }}</p>
          <router-link to="/purchase" class="btn btn-primary">{{ t('siteBuilder.subscribe') }}</router-link>
        </div>
        <template v-else>
          <div class="flex gap-4">
            <label class="flex items-center gap-1.5"><input v-model="publishMode" type="radio" value="new" :disabled="siteFull" /> {{ t('siteBuilder.publishNew') }}</label>
            <label class="flex items-center gap-1.5"><input v-model="publishMode" type="radio" value="update" :disabled="!sites.sites.length" /> {{ t('siteBuilder.publishUpdate') }}</label>
          </div>
          <p v-if="siteFull && publishMode === 'new'" class="text-xs text-amber-700">{{ t('siteBuilder.siteFull', { max: sites.quota.max_sites }) }}</p>
          <template v-if="publishMode === 'new'">
            <label class="block">
              <span class="input-label">{{ t('siteBuilder.publishNameTitle') }}</span>
              <input v-model="publishTitle" class="input" maxlength="60" />
            </label>
            <div>
              <span class="input-label">{{ t('siteBuilder.publishName') }}</span>
              <SiteNameInput v-model="publishName" v-model:valid="publishNameValid" :domain="sites.quota.domain" :placeholder="t('sites.name.placeholder')" />
            </div>
          </template>
          <template v-else>
            <label class="block">
              <span class="input-label">{{ t('siteBuilder.publishSite') }}</span>
              <select v-model.number="publishSiteId" class="input" data-testid="site-draft-publish-site">
                <option v-for="s in sites.sites" :key="s.id" :value="s.id">{{ s.title || s.name }} · {{ s.url }}</option>
              </select>
            </label>
            <p class="text-xs text-gray-500">{{ t('siteBuilder.publishUpdateHint') }}</p>
          </template>
        </template>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="publishOpen = false">{{ t('common.cancel') }}</button>
          <button v-if="canPublish" class="btn btn-primary" :disabled="busy || (publishMode === 'new' && (siteFull || !publishNameValid))" data-testid="site-draft-publish-go" @click="publish">
            {{ busy ? t('siteBuilder.publishing') : t('siteBuilder.publishGo') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog :show="confirmDelete" :title="t('common.delete')" :message="t('siteBuilder.deleteConfirm')" danger @confirm="remove" @cancel="confirmDelete = false" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import SiteNameInput from '@/components/user/SiteNameInput.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { mySites, type MySites } from '@/api/sites'
import {
  cancelSiteDraft,
  deleteSiteDraft,
  getSiteDraft,
  previewHtml,
  publishSiteDraft,
  retrySiteDraft,
  reviseSiteDraft,
  siteDraftActive,
  siteDraftImage,
  undoSiteDraft,
  type SiteDraft
} from '@/api/siteDrafts'

const { t, tm, rt } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const id = Number(route.params.id)

const draft = ref<SiteDraft | null>(null)
const busy = ref(false)
const instruction = ref('')
const device = ref<'desktop' | 'mobile'>('desktop')
const confirmDelete = ref(false)
const imageUrls = ref<Record<number, string>>({})
// What each picture URL was fetched for, so a redrawn picture is fetched again.
const imageKeys: Record<number, string> = {}
let timer: ReturnType<typeof setTimeout> | null = null

const chips = computed(() => (tm('siteBuilder.chips') as unknown[]).map((c) => rt(c as Parameters<typeof rt>[0])))
const active = computed(() => !!draft.value && siteDraftActive(draft.value.status))
const lastEvent = computed(() => draft.value?.events[draft.value.events.length - 1])
const missingImages = computed(() => !!draft.value?.images.some((im) => im.status !== 'ok'))
// The first page grows in the preview as it is written; a change keeps showing the current page until it is done.
const pageSource = computed(() => (draft.value ? draft.value.html || draft.value.draft_html || '' : ''))
const srcdoc = computed(() =>
  draft.value ? previewHtml(pageSource.value, imageUrls.value, { pending: t('siteBuilder.imgPending'), failed: t('siteBuilder.imgFailed') }, draft.value.images) : ''
)

async function syncImages(d: SiteDraft) {
  for (const im of d.images) {
    const key = `${im.prompt}|${im.size}|${im.bytes || 0}`
    if (im.status !== 'ok' || imageKeys[im.n] === key) continue
    imageKeys[im.n] = key
    try {
      // Data URLs: the sandboxed preview has an opaque origin and cannot load this page's blob: URLs.
      const url = await blobToDataURL(await siteDraftImage(d.id, im.n))
      imageUrls.value = { ...imageUrls.value, [im.n]: url }
    } catch {
      delete imageKeys[im.n]
    }
  }
}

function blobToDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(String(r.result))
    r.onerror = () => reject(r.error)
    r.readAsDataURL(blob)
  })
}

function set(d: SiteDraft) {
  draft.value = d
  void syncImages(d)
  schedule()
}

function schedule() {
  if (timer) clearTimeout(timer)
  timer = null
  if (draft.value && siteDraftActive(draft.value.status)) timer = setTimeout(load, 2000)
}

async function load() {
  try {
    set(await getSiteDraft(id))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
    schedule()
  }
}

async function act(fn: () => Promise<SiteDraft | void>) {
  busy.value = true
  try {
    const d = await fn()
    if (d) set(d)
    else await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

function revise() {
  const text = instruction.value.trim()
  if (!text || active.value) return
  void act(async () => {
    const d = await reviseSiteDraft(id, text)
    instruction.value = ''
    return d
  })
}
const retry = () => act(() => retrySiteDraft(id))
const undo = () => act(() => undoSiteDraft(id))
const stop = () => act(() => cancelSiteDraft(id))

async function remove() {
  confirmDelete.value = false
  try {
    await deleteSiteDraft(id)
    router.replace('/sites/ai')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function openNew() {
  const url = URL.createObjectURL(new Blob([srcdoc.value], { type: 'text/html' }))
  window.open(url, '_blank', 'noopener')
  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}

// --- publish -----------------------------------------------------------------------------------
const publishOpen = ref(false)
const sites = ref<MySites | null>(null)
const publishMode = ref<'new' | 'update'>('new')
const publishName = ref('')
const publishNameValid = ref(true)
const publishTitle = ref('')
const publishSiteId = ref(0)
const siteFull = computed(() => !!sites.value && sites.value.quota.used >= sites.value.quota.max_sites)
const canPublish = computed(() => !!sites.value?.quota.available && !!sites.value?.quota.subscribed && (publishMode.value === 'update' ? publishSiteId.value > 0 : true))

async function openPublish() {
  publishOpen.value = true
  publishTitle.value = draft.value?.title || ''
  try {
    sites.value = await mySites()
    const own = sites.value.sites.find((s) => s.id === draft.value?.site_id)
    publishMode.value = own || (siteFull.value && sites.value.sites.length) ? 'update' : 'new'
    publishSiteId.value = own?.id || sites.value.sites[0]?.id || 0
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function publish() {
  busy.value = true
  try {
    const site = await publishSiteDraft(
      id,
      publishMode.value === 'update' ? { site_id: publishSiteId.value } : { name: publishName.value, title: publishTitle.value }
    )
    publishOpen.value = false
    appStore.showSuccess(site.status === 'pending' || site.pending_version ? t('siteBuilder.publishPending') : t('siteBuilder.publishedTo', { url: site.url }))
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

// The poll keeps going in a background tab; a tab coming back refreshes at once.
function onVisible() {
  if (document.visibilityState === 'visible' && active.value) void load()
}

watch(
  () => draft.value?.title,
  (title) => {
    if (title) document.title = title
  }
)

onMounted(() => {
  void load()
  document.addEventListener('visibilitychange', onVisible)
})
onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
  document.removeEventListener('visibilitychange', onVisible)
})
</script>
