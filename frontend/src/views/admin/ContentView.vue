<template>
  <AppLayout>
    <div class="space-y-4" data-testid="content-view">
      <!-- An article opened from a list, a topic or the WeChat push (?item=) -->
      <ContentItemPanel
        v-if="openId"
        :item-id="openId"
        :platforms="overview?.platforms || []"
        :brand="brand"
        @close="closeItem"
        @changed="refresh"
      />

      <template v-else>
        <div class="card space-y-3 p-4">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.content.intro') }}</p>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="grid flex-1 grid-cols-2 gap-2 sm:grid-cols-4">
              <div v-for="s in summary" :key="s.key" class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.content.overview.${s.key}`) }}</div>
                <div class="text-xl font-semibold text-gray-900 dark:text-white">{{ s.value }}</div>
              </div>
            </div>
            <button class="btn btn-primary" :disabled="running || !hasKey" data-testid="content-run" @click="runNow">
              {{ running ? t('admin.content.overview.running_') : t('admin.content.overview.runNow') }}
            </button>
          </div>
          <p v-if="settingsLoaded && !hasKey" class="text-sm text-amber-600">{{ t('admin.content.overview.noKey') }}</p>
        </div>

        <div class="flex flex-wrap gap-2" role="tablist">
          <button
            v-for="tb in TABS"
            :key="tb"
            type="button"
            role="tab"
            :data-testid="`content-tab-${tb}`"
            :class="['btn btn-sm', tab === tb ? 'btn-primary' : 'btn-secondary']"
            @click="setTab(tb)"
          >
            {{ t(`admin.content.tabs.${tb}`) }}
            <span v-if="tb === 'review' && overview?.pending" class="ml-1 rounded-full bg-amber-400 px-1.5 text-xs text-white">{{ overview.pending }}</span>
          </button>
        </div>

        <!-- 待审核 / 全部稿件 -->
        <template v-if="tab === 'review' || tab === 'items'">
          <div v-if="tab === 'items'" class="flex flex-wrap gap-2">
            <button
              v-for="f in ITEM_FILTERS"
              :key="f"
              type="button"
              :class="['btn btn-sm', itemFilter === f ? 'btn-primary' : 'btn-secondary']"
              @click="itemFilter = f"
            >{{ f ? t(`admin.content.itemStatus.${f}`) : t('admin.content.items.filterAll') }}</button>
          </div>
          <p v-if="!items.length && !loadingItems" class="card p-8 text-center text-sm text-gray-400">
            {{ tab === 'review' ? t('admin.content.review.empty') : t('admin.content.items.empty') }}
          </p>
          <ul class="space-y-2">
            <li v-for="it in items" :key="it.id">
              <button type="button" class="card block w-full p-4 text-left hover:ring-1 hover:ring-primary-200" :data-testid="`content-item-${it.id}`" @click="openItem(it.id)">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ it.title || it.brief.topic }}</span>
                  <span class="text-xs text-gray-400">{{ fmtTime(it.created_at) }}</span>
                </div>
                <div class="mt-2 flex flex-wrap items-center gap-1.5 text-xs">
                  <span class="rounded bg-gray-100 px-1.5 py-0.5 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t(`admin.content.itemStatus.${it.status}`) }}</span>
                  <span
                    v-for="v in it.variants || []"
                    :key="v.platform"
                    :class="['rounded px-1.5 py-0.5', variantClass(v.status)]"
                  >{{ t(`admin.content.platform.${v.platform}`) }} · {{ t(`admin.content.versionStatus.${v.status}`) }}</span>
                  <span v-if="warnings(it)" class="text-amber-600">⚠ {{ warnings(it) }}</span>
                  <span v-if="it.error" class="truncate text-red-600">{{ it.error }}</span>
                </div>
              </button>
            </li>
          </ul>
        </template>

        <ContentTopics v-else-if="tab === 'topics'" @open-item="openItem" @changed="loadOverview" />
        <ContentStats v-else-if="tab === 'stats'" @open-item="openItem" />
        <ContentSettingsPanel v-else-if="tab === 'settings'" :platforms="overview?.platforms || []" @saved="onSettingsSaved" />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ContentItemPanel from '@/components/admin/content/ContentItemPanel.vue'
import ContentTopics from '@/components/admin/content/ContentTopics.vue'
import ContentStats from '@/components/admin/content/ContentStats.vue'
import ContentSettingsPanel from '@/components/admin/content/ContentSettingsPanel.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, ContentOverview, ContentSettings, ItemStatus, VersionStatus } from '@/api/admin/content'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const TABS = ['review', 'items', 'topics', 'stats', 'settings'] as const
type Tab = (typeof TABS)[number]
const ITEM_FILTERS: (ItemStatus | '')[] = ['', 'pending_review', 'done', 'failed', 'rejected']

const tab = ref<Tab>('review')
const overview = ref<ContentOverview | null>(null)
const items = ref<ContentItem[]>([])
const loadingItems = ref(false)
const itemFilter = ref<ItemStatus | ''>('')
const running = ref(false)
const hasKey = ref(false)
const settingsLoaded = ref(false)
const brand = computed(() => appStore.siteName || 'HiveGPT')
let timer: ReturnType<typeof setInterval> | null = null

const openId = computed(() => {
  const n = Number(route.query.item)
  return Number.isInteger(n) && n > 0 ? n : 0
})

const summary = computed(() => [
  { key: 'pending', value: overview.value?.pending ?? 0 },
  { key: 'running', value: overview.value?.running ?? 0 },
  { key: 'topics', value: overview.value?.topics ?? 0 },
  { key: 'published', value: overview.value?.published ?? 0 }
])

async function loadOverview() {
  try {
    overview.value = await contentAPI.getOverview()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function loadSettings() {
  try {
    const s = await contentAPI.getSettings()
    hasKey.value = s.key_id > 0 && !s.key_problem
  } catch {
    hasKey.value = false
  } finally {
    settingsLoaded.value = true
  }
}

async function loadItems() {
  if (tab.value !== 'review' && tab.value !== 'items') return
  loadingItems.value = true
  try {
    items.value = await contentAPI.listItems(tab.value === 'review' ? 'pending_review' : itemFilter.value)
    if (tab.value === 'items') {
      // Running articles first so their progress is easy to find.
      items.value.sort((a, b) => Number(b.running) - Number(a.running))
    }
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loadingItems.value = false
  }
}

function refresh() {
  loadOverview()
  loadItems()
}

function setTab(tb: Tab) {
  tab.value = tb
  router.replace({ query: { ...route.query, tab: tb, item: undefined } })
}

function openItem(id: number) {
  router.push({ query: { ...route.query, item: String(id) } })
}

function closeItem() {
  router.push({ query: { ...route.query, item: undefined } })
}

function onSettingsSaved(s: ContentSettings) {
  hasKey.value = s.key_id > 0 && !s.key_problem
}

async function runNow() {
  running.value = true
  try {
    await contentAPI.runNow()
    appStore.showSuccess(t('admin.content.overview.runStarted'))
    tab.value = 'items'
    refresh()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    running.value = false
  }
}

function warnings(it: ContentItem) {
  return (it.checks || []).filter((c) => c.level !== 'info').length
}

function variantClass(s: VersionStatus) {
  if (s === 'pending_review') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
  if (s === 'approved') return 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
  if (s === 'published') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (s === 'failed') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

watch([tab, itemFilter], loadItems)
watch(openId, (id, old) => {
  if (!id && old) refresh()
})

onMounted(() => {
  const q = String(route.query.tab || '')
  if ((TABS as readonly string[]).includes(q)) tab.value = q as Tab
  loadOverview()
  loadSettings()
  loadItems()
  // Keep the counts and the list fresh while articles are being written.
  timer = setInterval(() => {
    if (!openId.value && (overview.value?.running || 0) > 0) refresh()
  }, 10000)
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>
