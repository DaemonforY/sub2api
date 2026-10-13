<template>
  <AppLayout>
    <div class="space-y-4" data-testid="content-view">
      <!-- An article: the 写作台 while it is being written, the 待发布 review after -->
      <template v-if="openId">
        <DeskView v-if="openKind === 'desk'" :item-id="openId" @close="closeItem" @review="toReview" />
        <ContentItemPanel
          v-else-if="openKind === 'review'"
          :item-id="openId"
          :platforms="overview?.platforms || []"
          :brand="brand"
          @close="closeItem"
          @changed="loadOverview"
          @desk="toDesk"
        />
        <p v-else class="py-8 text-center text-sm text-gray-400">{{ t('admin.content.common.loading') }}</p>
      </template>

      <template v-else>
        <div class="card space-y-3 p-4">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.desk.intro') }}</p>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <div v-for="s in summary" :key="s.key" class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.desk.summary.${s.key}`) }}</div>
              <div class="text-xl font-semibold text-gray-900 dark:text-white">{{ s.value }}</div>
            </div>
          </div>
          <p v-if="settingsLoaded && !hasKey" class="text-sm text-amber-600">
            {{ t('admin.content.overview.noKey') }}
            <button class="ml-1 underline" @click="setTab('settings')">{{ t('admin.desk.summary.goSettings') }}</button>
          </p>
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
            {{ t(`admin.desk.tabs.${tb}`) }}
            <span v-if="tb === 'review' && overview?.pending" class="ml-1 rounded-full bg-amber-400 px-1.5 text-xs text-white">{{ overview.pending }}</span>
          </button>
        </div>

        <DeskList v-if="tab === 'desk'" @open="openItem" @persona="setTab('persona')" />

        <template v-else-if="tab === 'review'">
          <p v-if="!items.length && !loadingItems" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.desk.review.empty') }}</p>
          <ul class="space-y-2">
            <li v-for="it in items" :key="it.id">
              <button type="button" class="card block w-full p-4 text-left hover:ring-1 hover:ring-primary-200" :data-testid="`content-item-${it.id}`" @click="openItem(it.id)">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ it.title || it.brief.topic }}</span>
                  <span class="text-xs text-gray-400">{{ fmtTime(it.updated_at) }}</span>
                </div>
                <div class="mt-2 flex flex-wrap items-center gap-1.5 text-xs">
                  <span v-for="v in it.variants || []" :key="v.platform" :class="['rounded px-1.5 py-0.5', variantClass(v.status)]">
                    {{ t(`admin.content.platform.${v.platform}`) }} · {{ t(`admin.content.versionStatus.${v.status}`) }}
                  </span>
                </div>
              </button>
            </li>
          </ul>
        </template>

        <ContentTopics v-else-if="tab === 'topics'" @open-item="openItem" @changed="loadOverview" />
        <PersonaPanel v-else-if="tab === 'persona'" />
        <ContentStats v-else-if="tab === 'stats'" @open-item="openItem" />
        <ContentSettingsPanel v-else-if="tab === 'settings'" :platforms="overview?.platforms || []" @saved="onSettingsSaved" />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ContentItemPanel from '@/components/admin/content/ContentItemPanel.vue'
import ContentTopics from '@/components/admin/content/ContentTopics.vue'
import ContentStats from '@/components/admin/content/ContentStats.vue'
import ContentSettingsPanel from '@/components/admin/content/ContentSettingsPanel.vue'
import DeskList from '@/components/admin/content/desk/DeskList.vue'
import DeskView from '@/components/admin/content/desk/DeskView.vue'
import PersonaPanel from '@/components/admin/content/desk/PersonaPanel.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentItem, ContentOverview, ContentSettings, VersionStatus } from '@/api/admin/content'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const TABS = ['desk', 'review', 'topics', 'persona', 'stats', 'settings'] as const
type Tab = (typeof TABS)[number]

const tab = ref<Tab>('desk')
const overview = ref<ContentOverview | null>(null)
const items = ref<ContentItem[]>([])
const loadingItems = ref(false)
const hasKey = ref(false)
const settingsLoaded = ref(false)
const openKind = ref<'' | 'desk' | 'review'>('')
const brand = computed(() => appStore.siteName || 'HiveGPT')

const openId = computed(() => {
  const n = Number(route.query.item)
  return Number.isInteger(n) && n > 0 ? n : 0
})

const summary = computed(() => [
  { key: 'desk', value: overview.value?.desk ?? 0 },
  { key: 'pending', value: overview.value?.pending ?? 0 },
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
  if (tab.value !== 'review') return
  loadingItems.value = true
  try {
    items.value = await contentAPI.listItems('pending_review')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loadingItems.value = false
  }
}

// Which screen an article opens in: ?view=desk forces the 写作台, otherwise by its status.
async function resolveOpen() {
  openKind.value = ''
  if (!openId.value) return
  if (route.query.view === 'desk') {
    openKind.value = 'desk'
    return
  }
  try {
    const it = await contentAPI.getItem(openId.value)
    openKind.value = it.status === 'desk' ? 'desk' : 'review'
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

function setTab(tb: Tab) {
  tab.value = tb
  router.replace({ query: { tab: tb } })
}

function openItem(id: number) {
  router.push({ query: { ...route.query, item: String(id), view: undefined } })
}

function closeItem() {
  router.push({ query: { tab: tab.value } })
}

function toReview(id: number) {
  router.replace({ query: { tab: 'review', item: String(id) } })
}

function toDesk() {
  router.replace({ query: { ...route.query, view: 'desk' } })
}

function onSettingsSaved(s: ContentSettings) {
  hasKey.value = s.key_id > 0 && !s.key_problem
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

watch(tab, loadItems)
watch([openId, () => route.query.view], (cur, old) => {
  resolveOpen()
  if (!cur[0] && old?.[0]) {
    loadOverview()
    loadItems()
  }
})

onMounted(() => {
  const q = String(route.query.tab || '')
  if ((TABS as readonly string[]).includes(q)) tab.value = q as Tab
  loadOverview()
  loadSettings()
  loadItems()
  resolveOpen()
})
</script>
