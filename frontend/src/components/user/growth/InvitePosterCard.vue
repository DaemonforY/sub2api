<template>
  <div class="card p-6" data-testid="invite-poster-card">
    <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliatePoster.title') }}</h3>
    <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliatePoster.description') }}</p>

    <div class="mt-5 grid gap-6 md:grid-cols-[minmax(0,340px)_minmax(0,1fr)]">
      <!-- Large preview: an <img> so phones can long-press to save it. -->
      <div class="relative mx-auto w-full max-w-[340px]">
        <div class="aspect-[3/4] overflow-hidden rounded-xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900">
          <img v-if="images[selected]" :src="images[selected]" :alt="t(`affiliatePoster.templates.${selected}`)" class="h-full w-full object-contain" data-testid="invite-poster-preview" />
          <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-dark-400">
            <span v-if="failed" class="px-6 text-center text-amber-600 dark:text-amber-400">{{ t('affiliatePoster.failed') }}</span>
            <span v-else class="flex items-center gap-2"><Icon name="refresh" size="sm" class="animate-spin" />{{ t('affiliatePoster.generating') }}</span>
          </div>
        </div>
      </div>

      <div class="min-w-0 space-y-5">
        <div class="grid grid-cols-4 gap-3 md:grid-cols-2 xl:grid-cols-4" role="radiogroup">
          <button
            v-for="id in POSTER_TEMPLATES"
            :key="id"
            type="button"
            role="radio"
            :aria-checked="selected === id"
            :data-testid="`invite-poster-template-${id}`"
            :class="[
              'group rounded-xl border p-1.5 text-left transition',
              selected === id ? 'border-primary-500 ring-2 ring-primary-500/40' : 'border-gray-200 hover:border-primary-300 dark:border-dark-700'
            ]"
            @click="selected = id"
          >
            <div class="aspect-[3/4] overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-800">
              <img v-if="images[id]" :src="images[id]" alt="" class="h-full w-full object-cover" />
            </div>
            <p class="mt-1.5 truncate text-center text-xs text-gray-600 dark:text-dark-300">{{ t(`affiliatePoster.templates.${id}`) }}</p>
          </button>
        </div>

        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-primary" :disabled="!images[selected]" data-testid="invite-poster-download" @click="download">
            <Icon name="download" size="sm" />
            <span>{{ t('affiliatePoster.download') }}</span>
          </button>
          <button type="button" class="btn btn-secondary" :disabled="!images[selected]" @click="copyImage">
            <Icon name="copy" size="sm" />
            <span>{{ t('affiliatePoster.copyImage') }}</span>
          </button>
        </div>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliatePoster.saveHint') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { POSTER_TEMPLATES, posterSiteName, type PosterData, type PosterTemplateId } from '@/utils/invitePoster'
import { drawInvitePoster } from '@/utils/invitePosterDraw'

const props = defineProps<{
  inviteLink: string
  affCode: string
  rebateRate: number
  inviteeBonusRate: number
  inviteeBonusCap: number
  eduDiscount: number
}>()

const { t, locale } = useI18n()
const appStore = useAppStore()
const selected = ref<PosterTemplateId>('invite')
const images = reactive<Partial<Record<PosterTemplateId, string>>>({})
const failed = ref(false)

const data = computed<PosterData>(() => ({
  siteName: posterSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName),
  siteHost: typeof window === 'undefined' ? '' : window.location.host,
  logoUrl: appStore.cachedPublicSettings?.site_logo || '/logo.svg',
  inviteLink: props.inviteLink,
  affCode: props.affCode,
  rebateRate: props.rebateRate,
  inviteeBonusRate: props.inviteeBonusRate,
  inviteeBonusCap: props.inviteeBonusCap,
  eduDiscount: props.eduDiscount
}))

let generation = 0
async function renderAll() {
  if (!props.inviteLink || !props.affCode) return
  const run = ++generation
  failed.value = false
  // Selected design first so the preview appears quickly.
  const order = [selected.value, ...POSTER_TEMPLATES.filter((id) => id !== selected.value)]
  for (const id of order) {
    try {
      const canvas = document.createElement('canvas')
      await drawInvitePoster(canvas, id, data.value, (key, params) => t(key, params ?? {}))
      if (run !== generation) return
      images[id] = canvas.toDataURL('image/png')
    } catch {
      if (run === generation && id === selected.value) failed.value = true
    }
  }
}

function fileName() {
  return `${data.value.siteName}-${t(`affiliatePoster.templates.${selected.value}`)}-${props.affCode}.png`.replace(/[\\/:*?"<>|\s]+/g, '-')
}

function download() {
  const url = images[selected.value]
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = fileName()
  document.body.appendChild(a)
  a.click()
  a.remove()
}

async function copyImage() {
  const url = images[selected.value]
  if (!url) return
  try {
    if (typeof ClipboardItem === 'undefined' || !navigator.clipboard?.write) throw new Error('unsupported')
    const blob = await (await fetch(url)).blob()
    await navigator.clipboard.write([new ClipboardItem({ [blob.type]: blob })])
    appStore.showSuccess(t('affiliatePoster.copied'))
  } catch {
    appStore.showWarning(t('affiliatePoster.copyUnsupported'))
  }
}

onMounted(() => void renderAll())
watch(() => [props.inviteLink, props.affCode, props.rebateRate, props.inviteeBonusRate, props.inviteeBonusCap, props.eduDiscount, locale.value], () => {
  for (const id of POSTER_TEMPLATES) delete images[id]
  void renderAll()
})
</script>
