<template>
  <div class="card p-4 sm:p-6" data-testid="contest-auto-card">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.auto.title') }}</h3>
          <span
            v-if="settings"
            :class="[
              'rounded-full px-2.5 py-0.5 text-xs font-semibold',
              settings.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
            ]"
            data-testid="contest-auto-state"
          >{{ settings.enabled ? t('admin.contests.auto.running') : t('admin.contests.auto.stopped') }}</span>
        </div>
        <p v-if="settings" class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-dark-300">
          {{ t('admin.contests.auto.summary', {
            lead: settings.template.lead_days,
            day: settings.template.submission_end_day,
            time: settings.template.voting_end_time
          }) }}
        </p>
      </div>
      <div v-if="settings" class="flex flex-shrink-0 gap-2">
        <button type="button" class="btn btn-secondary" data-testid="contest-auto-edit" @click="openEditor">{{ t('admin.contests.auto.editTemplate') }}</button>
        <button
          type="button"
          :class="['btn', settings.enabled ? 'btn-danger' : 'btn-primary']"
          :disabled="saving"
          data-testid="contest-auto-toggle"
          @click="confirmToggle = true"
        >{{ settings.enabled ? t('admin.contests.auto.stop') : t('admin.contests.auto.start') }}</button>
      </div>
    </div>

    <div v-if="loading && !settings" class="mt-4 h-16 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
    <ul v-else-if="settings" class="mt-4 grid gap-3 sm:grid-cols-2">
      <li v-for="m in settings.months" :key="m.period" class="rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700" data-testid="contest-auto-month">
        <div class="flex items-center justify-between gap-2">
          <span class="truncate font-medium text-gray-900 dark:text-white">{{ m.title }}</span>
          <span :class="['flex-shrink-0 rounded px-1.5 py-0.5 text-xs font-medium', statusClass(m.status)]">{{ t(`admin.contests.auto.status.${m.status}`) }}</span>
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.contests.auto.window', { start: formatDateTime(m.submission_start_at), submissionEnd: formatDateTime(m.submission_end_at), votingEnd: formatDateTime(m.voting_end_at) }) }}
        </p>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          <RouterLink v-if="m.contest_id" :to="`/admin/contests/${m.contest_id}`" class="text-primary-600 hover:underline dark:text-primary-400">
            {{ t('admin.contests.auto.manage') }} →
          </RouterLink>
          <template v-else-if="m.status === 'scheduled'">{{ t('admin.contests.auto.createsAt', { time: formatDateTime(m.create_at) }) }}</template>
          <template v-else>{{ t(`admin.contests.auto.statusHint.${m.status}`) }}</template>
        </p>
      </li>
    </ul>

    <!-- Template editor -->
    <BaseDialog :show="editorOpen" :title="t('admin.contests.auto.editTemplate')" width="wide" @close="editorOpen = false">
      <form v-if="form" class="space-y-6" @submit.prevent="saveTemplate">
        <section class="grid gap-4 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.auto.fields.titleTemplate') }}</label>
            <input v-model.trim="form.title_template" type="text" maxlength="200" class="input" data-testid="contest-auto-title" />
            <p class="mt-1 text-xs text-gray-500">{{ t('admin.contests.auto.fields.placeholders') }}</p>
          </div>
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.description') }}</label>
            <textarea v-model="form.description" rows="3" class="input"></textarea>
          </div>
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.rules') }}</label>
            <textarea v-model="form.rules" rows="4" class="input"></textarea>
          </div>
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.coverImage') }}</label>
            <input v-model.trim="form.cover_image" type="text" class="input" placeholder="https://" />
          </div>
        </section>

        <section>
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.fields.schedule') }}</h4>
          <div class="grid gap-4 sm:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.contests.auto.fields.submissionEndDay') }}</label>
              <input v-model.number="form.submission_end_day" type="number" min="1" max="28" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.contests.auto.fields.votingEndTime') }}</label>
              <input v-model="form.voting_end_time" type="time" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.contests.auto.fields.leadDays') }}</label>
              <input v-model.number="form.lead_days" type="number" min="0" max="10" class="input" />
            </div>
          </div>
          <p class="mt-2 text-xs text-gray-500">{{ t('admin.contests.auto.fields.scheduleHint') }}</p>
        </section>

        <section>
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.fields.limits') }}</h4>
          <div class="grid gap-4 sm:grid-cols-4">
            <div>
              <label class="input-label">{{ t('admin.contests.fields.maxEntriesPerUser') }}</label>
              <input v-model.number="form.max_entries_per_user" type="number" min="1" max="50" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.contests.fields.votesPerUser') }}</label>
              <input v-model.number="form.votes_per_user" type="number" min="1" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.contests.fields.minVotesForPrize') }}</label>
              <input v-model.number="form.min_votes_for_prize" type="number" min="0" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.contests.fields.minAccountAgeHours') }}</label>
              <input v-model.number="form.min_account_age_hours" type="number" min="0" class="input" />
            </div>
          </div>
          <div class="mt-4 grid gap-2 sm:grid-cols-3">
            <label v-for="f in toggleFields" :key="f.key" class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form[f.key]" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
              {{ f.label }}
            </label>
          </div>
        </section>

        <section>
          <div class="mb-3 flex items-center justify-between">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.fields.prizes') }}</h4>
            <button type="button" class="btn btn-secondary btn-sm" @click="addPrize">{{ t('admin.contests.fields.addPrize') }}</button>
          </div>
          <div class="space-y-3">
            <div
              v-for="(p, i) in form.prizes"
              :key="i"
              class="grid items-end gap-3 rounded-lg border border-gray-200 p-3 sm:grid-cols-[90px_90px_1fr_120px_1.2fr_auto] dark:border-dark-700"
            >
              <div>
                <label class="input-label">{{ t('admin.contests.fields.rankFrom') }}</label>
                <input v-model.number="p.rank_from" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.contests.fields.rankTo') }}</label>
                <input v-model.number="p.rank_to" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.contests.fields.prizeType') }}</label>
                <Select v-model="p.type" :options="prizeTypeOptions" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.contests.fields.amount') }}</label>
                <input v-model.number="p.amount" type="number" min="0" step="0.01" class="input" :disabled="p.type !== 'balance'" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.contests.fields.label') }}</label>
                <input v-model.trim="p.label" type="text" maxlength="200" class="input" />
              </div>
              <button type="button" class="mb-2 text-gray-400 hover:text-red-500" :title="t('common.delete')" @click="form.prizes.splice(i, 1)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </div>
        </section>
        <p class="text-xs text-gray-500">{{ t('admin.contests.auto.fields.saveHint') }}</p>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="saving" data-testid="contest-auto-save" @click="saveTemplate">{{ t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="confirmToggle"
      :title="settings?.enabled ? t('admin.contests.auto.stop') : t('admin.contests.auto.start')"
      :message="settings?.enabled ? t('admin.contests.auto.stopConfirm') : t('admin.contests.auto.startConfirm')"
      :danger="settings?.enabled"
      @confirm="toggle"
      @cancel="confirmToggle = false"
    />
  </div>
</template>

<script setup lang="ts">
// 每月自动比赛: the switch and template of the monthly contest. While it runs, the backend creates and
// publishes each month's contest a few days before the 1st; stopping only affects future months.
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { adminAPI, type ContestAutoMonth, type ContestAutoSettings, type ContestAutoTemplate } from '@/api/admin'
import { formatDateTime } from '@/utils/contest'

const emit = defineEmits<{ (e: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const settings = ref<ContestAutoSettings | null>(null)
const loading = ref(false)
const saving = ref(false)
const editorOpen = ref(false)
const confirmToggle = ref(false)
const form = ref<ContestAutoTemplate | null>(null)

type ToggleKey = 'allow_self_vote' | 'require_review' | 'one_prize_per_user'
const toggleFields = computed<{ key: ToggleKey; label: string }[]>(() => [
  { key: 'one_prize_per_user', label: t('admin.contests.fields.onePrizePerUser') },
  { key: 'require_review', label: t('admin.contests.fields.requireReview') },
  { key: 'allow_self_vote', label: t('admin.contests.fields.allowSelfVote') }
])
const prizeTypeOptions = computed(() => [
  { value: 'balance', label: t('admin.contests.fields.typeBalance') },
  { value: 'custom', label: t('admin.contests.fields.typeCustom') }
])

function statusClass(status: ContestAutoMonth['status']): string {
  switch (status) {
    case 'created':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'scheduled':
      return 'bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    case 'paused':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
  }
}

async function load() {
  loading.value = true
  try {
    settings.value = await adminAPI.contests.getAuto()
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    loading.value = false
  }
}

function cloneTemplate(tpl: ContestAutoTemplate): ContestAutoTemplate {
  return { ...tpl, prizes: tpl.prizes.map((p) => ({ ...p })) }
}

function openEditor() {
  if (!settings.value) return
  form.value = cloneTemplate(settings.value.template)
  editorOpen.value = true
}

function addPrize() {
  if (!form.value) return
  const last = form.value.prizes[form.value.prizes.length - 1]
  const next = last ? last.rank_to + 1 : 1
  form.value.prizes.push({ rank_from: next, rank_to: next, type: 'balance', amount: 5, label: '' })
}

async function save(enabled: boolean, template: ContestAutoTemplate): Promise<boolean> {
  saving.value = true
  try {
    const before = settings.value?.months.filter((m) => m.status === 'created').length || 0
    settings.value = await adminAPI.contests.saveAuto(enabled, {
      ...template,
      prizes: template.prizes.map((p) => ({ ...p, amount: Number(p.amount) || 0 }))
    })
    const after = settings.value.months.filter((m) => m.status === 'created').length
    if (after > before) emit('changed')
    return true
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
    return false
  } finally {
    saving.value = false
  }
}

async function saveTemplate() {
  if (!form.value || !settings.value) return
  if (await save(settings.value.enabled, form.value)) {
    editorOpen.value = false
    appStore.showSuccess(t('admin.contests.saveSuccess'))
  }
}

async function toggle() {
  confirmToggle.value = false
  if (!settings.value) return
  const enabled = !settings.value.enabled
  if (await save(enabled, settings.value.template)) {
    appStore.showSuccess(enabled ? t('admin.contests.auto.started') : t('admin.contests.auto.stoppedToast'))
  }
}

onMounted(load)
defineExpose({ load })
</script>
