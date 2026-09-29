<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="card flex flex-wrap items-center justify-between gap-4 p-4 sm:p-6">
          <p class="max-w-3xl text-sm text-gray-600 dark:text-dark-300">{{ t('admin.contests.description') }}</p>
          <button type="button" class="btn btn-primary" data-testid="contest-create" @click="openEditor(null)">
            <Icon name="plus" size="sm" class="mr-1.5" />
            {{ t('admin.contests.create') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="contests" :loading="loading" row-key="id">
          <template #cell-title="{ row }">
            <div class="min-w-0 max-w-sm">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.title }}</div>
              <div class="mt-0.5 text-xs text-gray-400">#{{ row.id }}</div>
            </div>
          </template>
          <template #cell-phase="{ row }">
            <span :class="phaseBadgeClass(row.phase)">{{ t(`contests.phase.${row.phase}`) }}</span>
          </template>
          <template #cell-voting_end_at="{ value }">
            <span class="whitespace-nowrap text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center gap-3 text-sm font-medium">
              <RouterLink :to="`/admin/contests/${row.id}`" class="text-primary-600 hover:text-primary-700 dark:text-primary-400">
                {{ t('admin.contests.actions.manage') }}
              </RouterLink>
              <button v-if="editable(row)" type="button" class="text-gray-600 hover:text-gray-900 dark:text-gray-300" @click="openEditor(row)">
                {{ t('admin.contests.actions.edit') }}
              </button>
              <button v-if="row.phase === 'tallying'" type="button" class="text-amber-600 hover:text-amber-700" @click="ask('settle', row)">
                {{ t('admin.contests.actions.settle') }}
              </button>
              <button v-if="editable(row)" type="button" class="text-gray-500 hover:text-red-600" @click="ask('cancel', row)">
                {{ t('admin.contests.actions.cancel') }}
              </button>
              <button v-if="row.status === 'draft' || row.status === 'cancelled'" type="button" class="text-red-500 hover:text-red-700" @click="ask('delete', row)">
                {{ t('admin.contests.actions.delete') }}
              </button>
            </div>
          </template>
          <template #empty>
            <div class="py-10 text-center text-sm text-gray-500">{{ t('admin.contests.empty') }}</div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <!-- Editor -->
    <BaseDialog :show="editorOpen" :title="editingId ? t('admin.contests.edit') : t('admin.contests.create')" width="wide" @close="editorOpen = false">
      <form class="space-y-6" @submit.prevent="save">
        <section class="grid gap-4 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.title') }}</label>
            <input v-model.trim="form.title" type="text" maxlength="200" class="input" data-testid="contest-title-input" />
          </div>
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.description') }}</label>
            <textarea v-model="form.description" rows="3" class="input"></textarea>
          </div>
          <div class="sm:col-span-2">
            <label class="input-label">{{ t('admin.contests.fields.rules') }}</label>
            <textarea v-model="form.rules" rows="3" class="input"></textarea>
          </div>
          <div>
            <label class="input-label">{{ t('admin.contests.fields.coverImage') }}</label>
            <input v-model.trim="form.cover_image" type="text" class="input" placeholder="https://" />
            <p class="mt-1 text-xs text-gray-500">{{ t('admin.contests.fields.coverImageHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.contests.fields.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
        </section>

        <section>
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.fields.schedule') }}</h4>
          <div class="grid gap-4 sm:grid-cols-2">
            <div v-for="f in scheduleFields" :key="f.key">
              <label class="input-label">{{ f.label }}</label>
              <input v-model="times[f.key]" type="datetime-local" class="input" />
            </div>
          </div>
        </section>

        <section>
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.contests.fields.limits') }}</h4>
          <div class="grid gap-4 sm:grid-cols-3">
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
            <div class="sm:col-span-3">
              <label class="input-label">{{ t('admin.contests.fields.minAccountAgeHours') }}</label>
              <input v-model.number="form.min_account_age_hours" type="number" min="0" class="input sm:max-w-xs" />
              <p class="mt-1 text-xs text-gray-500">{{ t('admin.contests.fields.minAccountAgeHint') }}</p>
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
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="saving" data-testid="contest-save" @click="save">{{ t('common.save') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!pending"
      :title="pendingTitle"
      :message="pendingMessage"
      :danger="pending?.kind !== 'settle'"
      @confirm="runPending"
      @cancel="pending = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { adminAPI, type ContestInput } from '@/api/admin'
import type { Contest } from '@/api/contests'
import { formatDateTime, phaseBadgeClass } from '@/utils/contest'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const contests = ref<Contest[]>([])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.contests.columns.title') },
  { key: 'phase', label: t('admin.contests.columns.phase') },
  { key: 'voting_end_at', label: t('admin.contests.columns.deadline') },
  { key: 'entry_count', label: t('admin.contests.columns.entries') },
  { key: 'vote_count', label: t('admin.contests.columns.votes') },
  { key: 'actions', label: t('admin.contests.columns.actions') }
])

const statusOptions = computed(() => [
  { value: 'draft', label: t('admin.contests.fields.draft') },
  { value: 'published', label: t('admin.contests.fields.published') }
])
const prizeTypeOptions = computed(() => [
  { value: 'balance', label: t('admin.contests.fields.typeBalance') },
  { value: 'custom', label: t('admin.contests.fields.typeCustom') }
])

type TimeKey = 'submission_start_at' | 'submission_end_at' | 'voting_start_at' | 'voting_end_at'
type ToggleKey = 'allow_self_vote' | 'require_review' | 'one_prize_per_user'
const scheduleFields = computed<{ key: TimeKey; label: string }[]>(() => [
  { key: 'submission_start_at', label: t('admin.contests.fields.submissionStart') },
  { key: 'submission_end_at', label: t('admin.contests.fields.submissionEnd') },
  { key: 'voting_start_at', label: t('admin.contests.fields.votingStart') },
  { key: 'voting_end_at', label: t('admin.contests.fields.votingEnd') }
])
const toggleFields = computed<{ key: ToggleKey; label: string }[]>(() => [
  { key: 'one_prize_per_user', label: t('admin.contests.fields.onePrizePerUser') },
  { key: 'require_review', label: t('admin.contests.fields.requireReview') },
  { key: 'allow_self_vote', label: t('admin.contests.fields.allowSelfVote') }
])

function editable(c: Contest): boolean {
  return c.status === 'draft' || c.status === 'published'
}

function toLocalInput(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function blankForm(): ContestInput {
  const day = 24 * 3600 * 1000
  const start = new Date(Math.ceil(Date.now() / 3600000) * 3600000)
  return {
    title: '', description: '', rules: '', cover_image: '', status: 'draft',
    submission_start_at: start.toISOString(),
    submission_end_at: new Date(start.getTime() + 7 * day).toISOString(),
    voting_start_at: start.toISOString(),
    voting_end_at: new Date(start.getTime() + 10 * day).toISOString(),
    max_entries_per_user: 1, votes_per_user: 3, allow_self_vote: false, require_review: false,
    min_account_age_hours: 24, one_prize_per_user: true, min_votes_for_prize: 1,
    prizes: [
      { rank_from: 1, rank_to: 1, type: 'balance', amount: 20, label: t('admin.contests.defaults.first') },
      { rank_from: 2, rank_to: 3, type: 'balance', amount: 10, label: t('admin.contests.defaults.second') }
    ]
  }
}

const editorOpen = ref(false)
const saving = ref(false)
const editingId = ref(0)
const form = reactive<ContestInput>(blankForm())
const times = reactive<Record<TimeKey, string>>({ submission_start_at: '', submission_end_at: '', voting_start_at: '', voting_end_at: '' })

function openEditor(c: Contest | null) {
  editingId.value = c?.id || 0
  const src: ContestInput = c
    ? {
        title: c.title, description: c.description, rules: c.rules, cover_image: c.cover_image,
        status: c.status === 'published' ? 'published' : 'draft',
        submission_start_at: c.submission_start_at, submission_end_at: c.submission_end_at,
        voting_start_at: c.voting_start_at, voting_end_at: c.voting_end_at,
        max_entries_per_user: c.max_entries_per_user, votes_per_user: c.votes_per_user,
        allow_self_vote: c.allow_self_vote, require_review: c.require_review,
        min_account_age_hours: c.min_account_age_hours, one_prize_per_user: c.one_prize_per_user,
        min_votes_for_prize: c.min_votes_for_prize, prizes: c.prizes.map((p) => ({ ...p }))
      }
    : blankForm()
  Object.assign(form, src)
  for (const f of scheduleFields.value) times[f.key] = toLocalInput(src[f.key])
  editorOpen.value = true
}

function addPrize() {
  const last = form.prizes[form.prizes.length - 1]
  const next = last ? last.rank_to + 1 : 1
  form.prizes.push({ rank_from: next, rank_to: next, type: 'balance', amount: 5, label: '' })
}

async function save() {
  const payload: ContestInput = { ...form, prizes: form.prizes.map((p) => ({ ...p, amount: Number(p.amount) || 0 })) }
  for (const f of scheduleFields.value) {
    const d = new Date(times[f.key])
    if (!times[f.key] || Number.isNaN(d.getTime())) {
      appStore.showError(`${f.label}?`)
      return
    }
    payload[f.key] = d.toISOString()
  }
  saving.value = true
  try {
    if (editingId.value) await adminAPI.contests.update(editingId.value, payload)
    else await adminAPI.contests.create(payload)
    appStore.showSuccess(t('admin.contests.saveSuccess'))
    editorOpen.value = false
    await load()
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  } finally {
    saving.value = false
  }
}

// ---- confirm actions ----
type PendingKind = 'settle' | 'cancel' | 'delete'
const pending = ref<{ kind: PendingKind; contest: Contest } | null>(null)
const pendingTitle = computed(() => (pending.value ? t(`admin.contests.actions.${pending.value.kind}`) : ''))
const pendingMessage = computed(() => (pending.value ? t(`admin.contests.${pending.value.kind}Confirm`) : ''))

function ask(kind: PendingKind, contest: Contest) {
  pending.value = { kind, contest }
}

async function runPending() {
  const p = pending.value
  pending.value = null
  if (!p) return
  try {
    if (p.kind === 'settle') {
      await adminAPI.contests.settle(p.contest.id)
      appStore.showSuccess(t('admin.contests.settleSuccess'))
    } else if (p.kind === 'cancel') {
      await adminAPI.contests.cancel(p.contest.id)
    } else {
      await adminAPI.contests.remove(p.contest.id)
    }
    await load()
  } catch (err: any) {
    appStore.showError(err?.message || t('common.error'))
  }
}

async function load() {
  loading.value = true
  try {
    contests.value = await adminAPI.contests.list()
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.contests.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
