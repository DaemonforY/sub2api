<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap gap-1">
            <button
              v-for="tab in tabs"
              :key="tab"
              class="rounded-lg px-3 py-1.5 text-sm"
              :class="active === tab ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
              :data-testid="`desktop-tab-${tab}`"
              @click="active = tab"
            >
              {{ t(`desktopAdmin.tabs.${tab}`) }}
            </button>
          </div>
          <div v-if="active === 'overview'" class="flex items-center gap-2 text-sm">
            <Select v-model="days" :options="dayOptions" class="w-32" />
            <button class="btn btn-secondary btn-sm" @click="loadOverview">{{ t('common.refresh') }}</button>
          </div>
          <button v-else class="btn btn-primary btn-sm" data-testid="desktop-skill-new" @click="openEditor(null)">
            {{ t('desktopAdmin.skills.new') }}
          </button>
        </div>
        <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t(`desktopAdmin.hint.${active}`) }}</p>
      </div>

      <!-- 概览 -->
      <template v-if="active === 'overview'">
        <div v-if="overview" class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <div v-for="c in cards" :key="c.key" class="card p-4">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`desktopAdmin.cards.${c.key}`) }}</div>
            <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ c.value }}</div>
          </div>
        </div>

        <div v-if="overview" class="grid gap-4 lg:grid-cols-2">
          <div class="card p-4">
            <h3 class="mb-3 font-medium text-gray-900 dark:text-white">{{ t('desktopAdmin.funnel.title', { days: overview.days }) }}</h3>
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                  <th class="py-1">{{ t('desktopAdmin.funnel.step') }}</th>
                  <th class="py-1 text-right">{{ t('desktopAdmin.funnel.computers') }}</th>
                  <th class="py-1 text-right">{{ t('desktopAdmin.funnel.events') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in overview.funnel" :key="s.event" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-1.5">{{ eventLabel(s.event) }}</td>
                  <td class="py-1.5 text-right font-medium">{{ s.visitors }}</td>
                  <td class="py-1.5 text-right text-gray-500">{{ s.events }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="card p-4">
            <h3 class="mb-3 font-medium text-gray-900 dark:text-white">{{ t('desktopAdmin.templates.title') }}</h3>
            <p v-if="!overview.templates.length" class="text-sm text-gray-500">{{ t('desktopAdmin.empty') }}</p>
            <table v-else class="w-full text-sm" data-testid="desktop-template-stats">
              <thead>
                <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                  <th class="py-1">{{ t('desktopAdmin.templates.name') }}</th>
                  <th class="py-1 text-right">{{ t('desktopAdmin.templates.starts') }}</th>
                  <th class="py-1 text-right">{{ t('desktopAdmin.templates.doneRate') }}</th>
                  <th class="py-1 text-right">{{ t('desktopAdmin.templates.avg') }}</th>
                  <th class="py-1">{{ t('desktopAdmin.templates.topFailure') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in overview.templates" :key="s.template" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="py-1.5">{{ templateLabel(s.template) }}</td>
                  <td class="py-1.5 text-right">{{ s.starts }}</td>
                  <td class="py-1.5 text-right">{{ s.starts ? Math.round((s.done / s.starts) * 100) + '%' : '—' }}</td>
                  <td class="py-1.5 text-right">{{ s.avg_done_sec ? Math.round(s.avg_done_sec) + ' s' : '—' }}</td>
                  <td class="py-1.5 text-gray-500">{{ s.top_failure ? t(`desktopAdmin.reasons.${s.top_failure}`, s.top_failure) : '' }}</td>
                </tr>
              </tbody>
            </table>
            <template v-if="overview.failures.length">
              <h4 class="mb-2 mt-4 text-sm font-medium text-gray-900 dark:text-white">{{ t('desktopAdmin.failures') }}</h4>
              <ul class="space-y-1 text-sm text-gray-600 dark:text-dark-300">
                <li v-for="f in overview.failures" :key="f.step + f.reason">{{ f.step }} · {{ f.reason || '—' }} × {{ f.count }}</li>
              </ul>
            </template>
          </div>
        </div>

        <div v-if="overview" class="card p-4">
          <h3 class="mb-3 font-medium text-gray-900 dark:text-white">{{ t('desktopAdmin.devices.title') }}</h3>
          <p v-if="!overview.recent_devices.length" class="text-sm text-gray-500">{{ t('desktopAdmin.empty') }}</p>
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                <th class="py-1">{{ t('desktopAdmin.devices.name') }}</th>
                <th class="py-1">{{ t('desktopAdmin.devices.user') }}</th>
                <th class="py-1">{{ t('desktopAdmin.devices.status') }}</th>
                <th class="py-1">{{ t('desktopAdmin.devices.lastUsed') }}</th>
                <th class="py-1">{{ t('desktopAdmin.devices.created') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="d in overview.recent_devices" :key="d.key_id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="py-1.5">{{ d.name }}</td>
                <td class="py-1.5">{{ d.user_email || '#' + d.user_id }}</td>
                <td class="py-1.5">{{ d.status }}</td>
                <td class="py-1.5">{{ d.last_used_at ? formatTime(d.last_used_at) : t('desktopAdmin.devices.never') }}</td>
                <td class="py-1.5">{{ formatTime(d.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="loading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
      </template>

      <!-- 技能 -->
      <div v-else class="card overflow-x-auto p-4">
        <p v-if="!skills.length && !loading" class="text-sm text-gray-500">{{ t('desktopAdmin.empty') }}</p>
        <table v-else class="w-full text-sm" data-testid="desktop-skills-table">
          <thead>
            <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
              <th class="py-1">{{ t('desktopAdmin.skills.title') }}</th>
              <th class="py-1">{{ t('desktopAdmin.skills.slug') }}</th>
              <th class="py-1">{{ t('desktopAdmin.skills.status') }}</th>
              <th class="whitespace-nowrap py-1 text-right">{{ t('desktopAdmin.skills.sort') }}</th>
              <th class="whitespace-nowrap py-1 text-right">{{ t('desktopAdmin.skills.version') }}</th>
              <th class="py-1">{{ t('desktopAdmin.skills.updated') }}</th>
              <th class="py-1"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in skills" :key="s.id" class="border-t border-gray-100 dark:border-dark-700">
              <td class="py-2">
                <span class="mr-1">{{ s.icon }}</span>{{ s.title }}
                <div class="max-w-md truncate text-xs text-gray-500">{{ s.summary }}</div>
              </td>
              <td class="whitespace-nowrap py-2 font-mono text-xs">{{ s.slug }}</td>
              <td class="whitespace-nowrap py-2">
                <span class="rounded px-2 py-0.5 text-xs" :class="statusClass[s.status]">{{ t(`desktopAdmin.status.${s.status}`) }}</span>
              </td>
              <td class="py-2 text-right">{{ s.sort_order }}</td>
              <td class="whitespace-nowrap py-2 text-right">v{{ s.version }}</td>
              <td class="whitespace-nowrap py-2 text-xs text-gray-500">{{ formatTime(s.updated_at) }}</td>
              <td class="whitespace-nowrap py-2 text-right">
                <button class="btn btn-secondary btn-sm" @click="openEditor(s)">{{ t('common.edit') }}</button>
                <button
                  v-if="s.status !== 'published'"
                  class="btn btn-primary btn-sm ml-1"
                  :data-testid="`desktop-skill-publish-${s.slug}`"
                  @click="changeStatus(s, 'published')"
                >
                  {{ t('desktopAdmin.skills.publish') }}
                </button>
                <button v-else class="btn btn-secondary btn-sm ml-1" @click="changeStatus(s, 'hidden')">{{ t('desktopAdmin.skills.hide') }}</button>
                <button class="btn btn-danger btn-sm ml-1" @click="toDelete = s">{{ t('common.delete') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <BaseDialog
      :show="editorOpen"
      :title="editing ? t('desktopAdmin.editor.editTitle', { title: editing.title }) : t('desktopAdmin.editor.newTitle')"
      width="extra-wide"
      @close="editorOpen = false"
    >
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-3">
          <div class="grid grid-cols-[4rem_1fr] gap-2">
            <label class="block">
              <span class="input-label">{{ t('desktopAdmin.editor.icon') }}</span>
              <input v-model="form.icon" class="input" maxlength="8" />
            </label>
            <label class="block">
              <span class="input-label">{{ t('desktopAdmin.editor.title') }}</span>
              <input v-model="form.title" class="input" maxlength="40" data-testid="desktop-skill-title" />
            </label>
          </div>
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.slug') }}</span>
            <input v-model="form.slug" class="input font-mono" maxlength="47" placeholder="invoices" data-testid="desktop-skill-slug" />
            <span class="input-hint">{{ t('desktopAdmin.editor.slugHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.summary') }}</span>
            <input v-model="form.summary" class="input" maxlength="200" data-testid="desktop-skill-summary" />
            <span class="input-hint">{{ t('desktopAdmin.editor.summaryHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.prepare') }}</span>
            <textarea v-model="form.prepare" class="input" rows="2" />
          </label>
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.prompt') }}</span>
            <textarea v-model="form.prompt" class="input" rows="5" data-testid="desktop-skill-prompt" />
            <span class="input-hint">{{ t('desktopAdmin.editor.promptHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.samplePrompt') }}</span>
            <textarea v-model="form.sample_prompt" class="input" rows="4" />
            <span class="input-hint">{{ t('desktopAdmin.editor.samplePromptHint') }}</span>
          </label>
          <div class="flex flex-wrap items-center gap-4">
            <label class="flex items-center gap-2 text-sm">
              <Toggle v-model="form.needs_office" />{{ t('desktopAdmin.editor.needsOffice') }}
            </label>
            <label class="flex items-center gap-2 text-sm">
              {{ t('desktopAdmin.editor.sort') }}
              <input v-model.number="form.sort_order" type="number" class="input w-24" />
            </label>
          </div>
        </div>
        <div class="space-y-3">
          <label class="block">
            <span class="input-label">{{ t('desktopAdmin.editor.instructions') }}</span>
            <textarea v-model="form.instructions" class="input font-mono text-xs" rows="18" data-testid="desktop-skill-instructions" />
            <span class="input-hint">{{ t('desktopAdmin.editor.instructionsHint') }}</span>
          </label>
          <div v-if="skillMD">
            <span class="input-label">{{ t('desktopAdmin.editor.preview') }}</span>
            <pre class="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ skillMD }}</pre>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="editorOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="saving" data-testid="desktop-skill-save" @click="save">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!toDelete"
      :title="t('desktopAdmin.skills.deleteTitle')"
      :message="t('desktopAdmin.skills.deleteMessage', { title: toDelete?.title ?? '' })"
      danger
      @confirm="remove"
      @cancel="toDelete = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'
import * as api from '@/api/admin/desktop'

const { t } = useI18n()
const appStore = useAppStore()

const tabs = ['overview', 'skills'] as const
type Tab = (typeof tabs)[number]
const initial = new URLSearchParams(location.search).get('tab') as Tab
const active = ref<Tab>(tabs.includes(initial) ? initial : 'overview')
watch(active, (tab) => {
  const url = new URL(location.href)
  url.searchParams.set('tab', tab)
  history.replaceState(history.state, '', url)
  if (tab === 'skills' && !skills.value.length) void loadSkills()
})

const loading = ref(false)
const errorMessage = (err: unknown) => (err as { message?: string })?.message || t('common.error')
const formatTime = (s: string) => new Date(s).toLocaleString()

// ---- overview ----
const days = ref(30)
const dayOptions = [7, 30, 90].map((d) => ({ value: d, label: t('desktopAdmin.days', { n: d }) }))
const overview = ref<api.DesktopOverview | null>(null)
watch(days, () => void loadOverview())
const cards = computed(() => {
  const o = overview.value
  if (!o) return []
  const step = (ev: string) => o.funnel.find((s) => s.event === ev)?.visitors ?? 0
  return [
    { key: 'devices', value: o.devices },
    { key: 'active', value: o.active_devices },
    { key: 'firstTask', value: step('desktop_task_start') },
    { key: 'returned', value: o.returned }
  ]
})

async function loadOverview() {
  loading.value = true
  try {
    overview.value = await api.getOverview(days.value)
    if (!skills.value.length) skills.value = await api.listSkills()
  } catch (err) {
    appStore.showError(errorMessage(err))
  } finally {
    loading.value = false
  }
}

/** The analytics page labels say 「桌面助手：…」; here that prefix is redundant. */
const eventLabel = (ev: string) => t(`admin.analytics.events.${ev}`).replace(/^(桌面助手：|Desktop: )/, '')

/** Analytics carry the slug (or sample:<slug> / custom); show the skill's title. */
function templateLabel(tpl: string) {
  if (tpl === 'custom') return t('desktopAdmin.templates.custom')
  const sample = tpl.startsWith('sample:')
  const slug = sample ? tpl.slice(7) : tpl
  const title = skills.value.find((s) => s.slug === slug)?.title ?? slug
  return sample ? t('desktopAdmin.templates.sample', { title }) : title
}

// ---- skills ----
const skills = ref<api.DesktopSkill[]>([])
const statusClass: Record<api.DesktopSkillStatus, string> = {
  published: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
  draft: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300',
  hidden: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

async function loadSkills() {
  loading.value = true
  try {
    skills.value = await api.listSkills()
  } catch (err) {
    appStore.showError(errorMessage(err))
  } finally {
    loading.value = false
  }
}

const emptyForm = (): api.DesktopSkillInput => ({
  slug: '',
  title: '',
  icon: '',
  summary: '',
  prepare: '',
  prompt: '',
  sample_prompt: '',
  instructions: '',
  needs_office: true,
  status: 'draft',
  sort_order: (skills.value.at(-1)?.sort_order ?? 0) + 10
})
const editorOpen = ref(false)
const editing = ref<api.DesktopSkill | null>(null)
const form = reactive<api.DesktopSkillInput>(emptyForm())
const skillMD = ref('')
const saving = ref(false)

async function openEditor(s: api.DesktopSkill | null) {
  editing.value = s
  skillMD.value = ''
  Object.assign(form, emptyForm())
  if (s) {
    const { id: _id, version: _v, created_at: _c, updated_at: _u, ...rest } = s
    Object.assign(form, rest)
    api
      .getSkill(s.id)
      .then((r) => (skillMD.value = r.skill_md))
      .catch(() => {})
  }
  editorOpen.value = true
}

async function save() {
  saving.value = true
  try {
    if (editing.value) {
      // Status is changed with the publish / hide buttons; saving keeps it.
      await api.updateSkill(editing.value.id, { ...form, status: '' as api.DesktopSkillStatus })
    } else {
      await api.createSkill(form)
    }
    editorOpen.value = false
    appStore.showSuccess(t('desktopAdmin.editor.saved'))
    await loadSkills()
  } catch (err) {
    appStore.showError(errorMessage(err))
  } finally {
    saving.value = false
  }
}

async function changeStatus(s: api.DesktopSkill, status: api.DesktopSkillStatus) {
  try {
    await api.setSkillStatus(s.id, status)
    appStore.showSuccess(t(`desktopAdmin.skills.${status === 'published' ? 'published' : 'hidden'}`))
    await loadSkills()
  } catch (err) {
    appStore.showError(errorMessage(err))
  }
}

const toDelete = ref<api.DesktopSkill | null>(null)
async function remove() {
  const s = toDelete.value
  toDelete.value = null
  if (!s) return
  try {
    await api.deleteSkill(s.id)
    await loadSkills()
  } catch (err) {
    appStore.showError(errorMessage(err))
  }
}

onMounted(() => {
  if (active.value === 'overview') void loadOverview()
  else void loadSkills()
})
</script>
