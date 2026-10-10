<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('siteBuilder.title') }}</h2>
            <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-dark-400">{{ t('siteBuilder.intro') }}</p>
          </div>
          <router-link to="/sites" class="btn btn-secondary">{{ t('siteBuilder.back') }}</router-link>
        </div>
      </div>

      <form class="card space-y-4 p-5" data-testid="site-builder-form" @submit.prevent="start">
        <label class="block text-sm">
          <span class="input-label">{{ t('siteBuilder.describe') }}</span>
          <textarea
            v-model="description"
            class="input min-h-[7rem]"
            maxlength="2000"
            :placeholder="t('siteBuilder.describePlaceholder')"
            data-testid="site-builder-description"
          ></textarea>
          <span class="input-hint">{{ t('siteBuilder.describeTips') }}</span>
        </label>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="block text-sm">
            <span class="input-label">{{ t('siteBuilder.style') }}</span>
            <input v-model="style" class="input" maxlength="200" :placeholder="t('siteBuilder.stylePlaceholder')" />
          </label>
          <div class="text-sm">
            <span class="input-label">{{ t('siteBuilder.images') }}</span>
            <div class="flex flex-wrap items-center gap-2">
              <button
                v-for="n in imageChoices"
                :key="n"
                type="button"
                :class="['rounded-lg border px-3 py-1.5 text-sm', images === n ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300']"
                :data-testid="`site-builder-images-${n}`"
                @click="images = n"
              >
                {{ n === 0 ? t('siteBuilder.imagesNone') : t('siteBuilder.imagesCount', { n }) }}
              </button>
              <span v-if="config?.image_price" class="text-xs text-gray-400">{{ t('siteBuilder.imagePrice', { price: fmt(config.image_price) }) }}</span>
            </div>
          </div>
        </div>
        <label class="block text-sm">
          <span class="input-label">{{ t('siteBuilder.key') }}</span>
          <select v-model.number="keyId" class="input" data-testid="site-builder-key">
            <option :value="0">{{ t('siteBuilder.keyPlaceholder') }}</option>
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
          </select>
          <span class="input-hint">
            {{ t('siteBuilder.keyHint') }}
            <router-link to="/keys" class="text-primary-600 hover:underline">{{ t('siteBuilder.keyCreate') }}</router-link>
          </span>
        </label>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span v-if="config?.page_price" class="text-xs text-gray-500 dark:text-dark-400" data-testid="site-builder-cost">
            {{ t('siteBuilder.cost', { page: fmt(config.page_price), revise: fmt(config.revise_price || 0) }) }}<template v-if="images && config.image_price">{{ t('siteBuilder.costImages', { n: images, total: fmt(config.image_price * images) }) }}</template>
          </span>
          <span v-else></span>
          <button class="btn btn-primary" :disabled="starting || !description.trim() || !keyId" data-testid="site-builder-start">
            {{ starting ? t('siteBuilder.starting') : t('siteBuilder.start') }}
          </button>
        </div>
      </form>

      <div v-if="drafts.length" class="space-y-2">
        <h3 class="px-1 text-sm font-semibold text-gray-700 dark:text-dark-200">{{ t('siteBuilder.mine') }}</h3>
        <div class="grid gap-3 md:grid-cols-2">
          <router-link v-for="d in drafts" :key="d.id" :to="`/sites/ai/${d.id}`" class="card block p-4 transition hover:shadow-md" data-testid="site-draft-card">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">{{ d.title || d.brief.description }}</div>
                <div class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ d.brief.description }}</div>
              </div>
              <span :class="['shrink-0 rounded-full px-2 py-0.5 text-xs', badge(d)]">{{ d.site_url ? t('siteBuilder.published') : t(`siteBuilder.status.${d.status}`) }}</span>
            </div>
            <div class="mt-2 text-xs text-gray-400">{{ formatDateTime(d.updated_at) }}</div>
          </router-link>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { listKeys, type KeyOption } from '@/api/tutors'
import { createSiteDraft, listSiteDrafts, siteBuilderConfig, siteDraftActive, type SiteBuilderConfig, type SiteDraft } from '@/api/siteDrafts'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const description = ref('')
const style = ref('')
const images = ref(1)
const imageChoices = [0, 1, 2, 3, 4]
const keyId = ref(0)
const keys = ref<KeyOption[]>([])
const config = ref<SiteBuilderConfig | null>(null)
const drafts = ref<SiteDraft[]>([])
const starting = ref(false)

const fmt = (v: number) => (v >= 0.1 ? v.toFixed(2) : v.toFixed(3))

function badge(d: SiteDraft) {
  if (d.site_url) return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (siteDraftActive(d.status)) return 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
  if (d.status === 'failed') return 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
}

watch(keyId, async (id) => {
  try {
    config.value = await siteBuilderConfig(id)
  } catch {
    // prices are optional
  }
})

async function start() {
  starting.value = true
  try {
    const d = await createSiteDraft(keyId.value, { description: description.value.trim(), style: style.value.trim(), images: images.value })
    router.push(`/sites/ai/${d.id}`)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    starting.value = false
  }
}

onMounted(async () => {
  try {
    const [ks, list, cfg] = await Promise.all([listKeys(), listSiteDrafts(), siteBuilderConfig(0)])
    keys.value = ks
    drafts.value = list
    config.value = cfg
    images.value = cfg.default_images
    if (ks.length === 1) keyId.value = ks[0].id
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
})
</script>
