<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- Rules -->
      <div class="card space-y-3 p-4" data-testid="creator-settings">
        <div class="flex flex-wrap items-end gap-3">
          <label class="flex items-center gap-2 text-sm">
            <input v-model="settings.creator_enabled" type="checkbox" class="h-4 w-4" data-testid="creator-enabled" />
            {{ t('admin.creators.settings.enabled') }}
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.creators.settings.commission') }}</span>
            <input v-model.number="settings.creator_commission_percent" type="number" min="0" max="100" step="0.5" class="input w-28" data-testid="creator-commission" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.creators.settings.settleDays') }}</span>
            <input v-model.number="settings.creator_settle_days" type="number" min="0" max="365" class="input w-28" />
          </label>
          <label class="text-sm">
            <span class="input-label">{{ t('admin.creators.settings.withdrawMin') }}</span>
            <input v-model.number="settings.creator_withdraw_min_cny" type="number" min="1" step="1" class="input w-28" />
          </label>
          <button class="btn btn-secondary" :disabled="busy" data-testid="creator-settings-save" @click="saveSettings">{{ t('common.save') }}</button>
        </div>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.creators.settings.hint') }}</p>
      </div>

      <div class="flex flex-wrap gap-1">
        <button v-for="tab in tabs" :key="tab" :class="['btn btn-sm', panel === tab ? 'btn-primary' : 'btn-secondary']" :data-testid="`creators-tab-${tab}`" @click="switchPanel(tab)">
          {{ t(`admin.creators.tabs.${tab}`) }}<span v-if="badges[tab]" class="ml-1 rounded-full bg-rose-500 px-1.5 text-xs text-white">{{ badges[tab] }}</span>
        </button>
      </div>

      <!-- Creators -->
      <template v-if="panel === 'creators'">
        <div v-if="!creators.length" class="card py-12 text-center text-sm text-gray-500">{{ t('admin.creators.empty') }}</div>
        <div v-for="c in creators" :key="c.user_id" class="card space-y-3 p-4" data-testid="creator-row">
          <div class="flex flex-wrap items-center gap-3">
            <div class="min-w-0 flex-1">
              <div class="font-medium text-gray-900 dark:text-white">{{ c.display_name }} <span class="text-xs font-normal text-gray-400">#{{ c.user_id }} {{ c.user_email }}</span></div>
              <div class="text-xs text-gray-500">
                {{ t('admin.creators.appliedAt', { time: formatDateTime(c.created_at) }) }} ·
                {{ t('admin.creators.courseCount', { onSale: c.on_sale_count, total: c.course_count }) }}
                <template v-if="c.balance"> · {{ t('admin.creators.money', { net: c.balance.net.toFixed(2), available: c.balance.available.toFixed(2), paid: c.balance.paid.toFixed(2) }) }}</template>
              </div>
            </div>
            <span :class="['badge', creatorBadge(c.status)]">{{ t(`admin.creators.status.${c.status}`) }}</span>
            <span class="text-xs text-gray-500">{{ t('admin.creators.rate', { percent: c.commission_percent ?? settings.creator_commission_percent }) }}<template v-if="c.commission_percent === null">{{ t('admin.creators.rateDefault') }}</template></span>
            <button class="btn btn-secondary btn-sm" @click="toggleCreator(c)">{{ openCreator === c.user_id ? t('admin.creators.close') : t('admin.creators.handle') }}</button>
          </div>
          <div v-if="openCreator === c.user_id" class="space-y-3 border-t border-gray-100 pt-3 text-sm dark:border-dark-700">
            <dl class="grid gap-2 md:grid-cols-2">
              <div><dt class="text-xs text-gray-500">{{ t('creator.apply.contact') }}</dt><dd>{{ c.contact }}</dd></div>
              <div><dt class="text-xs text-gray-500">{{ t('creator.apply.bio') }}</dt><dd class="whitespace-pre-wrap">{{ c.bio || '—' }}</dd></div>
              <div class="md:col-span-2"><dt class="text-xs text-gray-500">{{ t('creator.apply.plan') }}</dt><dd class="whitespace-pre-wrap">{{ c.plan }}</dd></div>
            </dl>
            <div class="flex flex-wrap items-end gap-3">
              <label>
                <span class="input-label">{{ t('admin.creators.fields.status') }}</span>
                <select v-model="creatorForm.status" class="input" data-testid="creator-status">
                  <option v-for="s in ['pending', 'approved', 'rejected', 'suspended']" :key="s" :value="s">{{ t(`admin.creators.status.${s}`) }}</option>
                </select>
              </label>
              <label>
                <span class="input-label">{{ t('admin.creators.fields.commission') }}</span>
                <input v-model="creatorForm.commission" type="number" min="0" max="100" step="0.5" class="input w-32" :placeholder="String(settings.creator_commission_percent)" data-testid="creator-own-rate" />
              </label>
              <label class="min-w-48 flex-1">
                <span class="input-label">{{ t('admin.creators.fields.note') }}</span>
                <input v-model="creatorForm.note" class="input" maxlength="500" :placeholder="t('admin.creators.notePlaceholder')" />
              </label>
              <button class="btn btn-primary" :disabled="busy" data-testid="creator-save" @click="saveCreator(c)">{{ t('common.save') }}</button>
            </div>
            <p class="text-xs text-gray-500">{{ t('admin.creators.statusHint') }}</p>
          </div>
        </div>
      </template>

      <!-- Course reviews -->
      <template v-else-if="panel === 'reviews'">
        <div v-if="!reviews.length" class="card py-12 text-center text-sm text-gray-500">{{ t('admin.creators.noReviews') }}</div>
        <div v-for="c in reviews" :key="c.id" class="card space-y-3 p-4" data-testid="review-row">
          <div class="flex flex-wrap items-center gap-3">
            <img v-if="c.draft?.cover_url" :src="c.draft.cover_url" alt="" class="h-12 w-20 rounded object-cover" />
            <div class="min-w-0 flex-1">
              <div class="font-medium text-gray-900 dark:text-white">{{ c.draft?.title || c.title }}</div>
              <div class="text-xs text-gray-500">
                {{ c.creator_name }} · {{ t('admin.creators.submittedAt', { time: formatDateTime(c.submitted_at || '') }) }} ·
                {{ c.approved_at ? t('admin.creators.changeOfLive') : t('admin.creators.firstListing') }}
              </div>
            </div>
            <span class="text-sm">¥{{ c.draft?.price }}<span v-if="c.approved_at && c.draft && c.draft.price !== c.price" class="ml-1 text-xs text-gray-400 line-through">¥{{ c.price }}</span></span>
            <button class="btn btn-secondary btn-sm" @click="inspect(c)">{{ openReview === c.id ? t('admin.creators.close') : t('admin.creators.inspect') }}</button>
          </div>
          <div v-if="openReview === c.id && c.draft" class="space-y-3 border-t border-gray-100 pt-3 text-sm dark:border-dark-700" data-testid="review-detail">
            <ul class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
              <li v-for="line in reviewLines(c)" :key="line">{{ line }}</li>
            </ul>
            <div v-for="field in markdownFields" :key="field">
              <div class="text-xs font-medium text-gray-500">{{ t(`admin.courses.fields.${field}`) }}</div>
              <div class="course-md rounded-lg border border-gray-200 p-3 dark:border-dark-700" v-html="renderMarkdown(c.draft[field]) || '—'"></div>
            </div>
            <div>
              <div class="text-xs font-medium text-gray-500">{{ t('admin.courses.fields.outline') }}</div>
              <div v-for="(sec, si) in c.draft.outline" :key="si" class="text-xs">
                <span class="font-medium">{{ sec.title }}</span>：{{ sec.lessons.map((l) => l.title).join('、') }}
              </div>
            </div>
            <div v-if="reviewDelivery" class="rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800" data-testid="review-delivery">
              <div class="font-medium text-gray-500">{{ t('admin.creators.deliveryCheck') }}</div>
              <a :href="reviewDelivery.link" target="_blank" rel="noopener noreferrer" class="break-all font-mono text-primary-600">{{ reviewDelivery.link }}</a>
              <template v-if="reviewDelivery.code"> · {{ t('admin.courses.fields.code') }} {{ reviewDelivery.code }}</template>
              <template v-if="reviewDelivery.password"> · {{ t('admin.courses.fields.password') }} {{ reviewDelivery.password }}</template>
            </div>
            <p v-else class="text-amber-600">{{ t('admin.courses.missingDelivery') }}</p>
            <div class="flex flex-wrap items-end gap-2">
              <label class="min-w-48 flex-1">
                <span class="input-label">{{ t('admin.creators.reviewNote') }}</span>
                <input v-model="reviewNote" class="input" maxlength="500" :placeholder="t('admin.creators.reviewNotePlaceholder')" data-testid="review-note" />
              </label>
              <button class="btn btn-primary" :disabled="busy" data-testid="review-approve" @click="decide(c, true)">{{ t('admin.creators.approve') }}</button>
              <button class="btn btn-danger" :disabled="busy || !reviewNote.trim()" data-testid="review-reject" @click="decide(c, false)">{{ t('admin.creators.reject') }}</button>
            </div>
          </div>
        </div>
      </template>

      <!-- Withdrawals -->
      <template v-else>
        <div class="flex flex-wrap gap-1">
          <button v-for="s in ['pending', 'paid', 'rejected', 'cancelled', 'all']" :key="s" :class="['btn btn-sm', withdrawStatus === s ? 'btn-primary' : 'btn-secondary']" @click="loadWithdrawals(s)">
            {{ t(`admin.creators.withdrawStatus.${s}`) }}
          </button>
        </div>
        <div v-if="!withdrawals.length" class="card py-12 text-center text-sm text-gray-500">{{ t('admin.creators.noWithdrawals') }}</div>
        <div v-for="w in withdrawals" :key="w.id" class="card space-y-2 p-4 text-sm" data-testid="creator-withdrawal">
          <div class="flex flex-wrap items-center gap-3">
            <span class="text-lg font-semibold">¥{{ w.cny_amount.toFixed(2) }}</span>
            <span :class="['badge', w.status === 'paid' ? 'badge-success' : w.status === 'pending' ? 'badge-warning' : 'badge-gray']">{{ t(`admin.creators.withdrawStatus.${w.status}`) }}</span>
            <span>{{ w.display_name }} <span class="text-xs text-gray-400">#{{ w.user_id }} {{ w.user_email }}</span></span>
            <span class="ml-auto text-xs text-gray-400">{{ formatDateTime(w.created_at) }}</span>
          </div>
          <div class="font-mono text-xs">{{ t(`creator.withdraw.${w.method}`) }} · {{ w.account }} · {{ w.real_name }}</div>
          <div v-if="w.user_note" class="text-xs text-gray-500">{{ t('admin.creators.userNote') }}{{ w.user_note }}</div>
          <div v-if="w.admin_note" class="text-xs text-gray-500">{{ t('creator.withdraw.adminNote') }}{{ w.admin_note }}</div>
          <div v-if="w.status === 'pending'" class="flex flex-wrap items-end gap-2">
            <input v-model="withdrawNotes[w.id]" class="input min-w-48 flex-1" maxlength="500" :placeholder="t('admin.creators.payNotePlaceholder')" />
            <button class="btn btn-primary btn-sm" :disabled="busy" data-testid="withdrawal-paid" @click="resolve(w.id, true)">{{ t('admin.creators.markPaid') }}</button>
            <button class="btn btn-danger btn-sm" :disabled="busy || !withdrawNotes[w.id]?.trim()" @click="resolve(w.id, false)">{{ t('admin.creators.reject') }}</button>
          </div>
        </div>
        <div v-if="withdrawTotal > 20" class="flex gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="withdrawPage <= 1" @click="loadWithdrawals(withdrawStatus, withdrawPage - 1)">{{ t('creator.sales.prev') }}</button>
          <button class="btn btn-secondary btn-sm" :disabled="withdrawPage * 20 >= withdrawTotal" @click="loadWithdrawals(withdrawStatus, withdrawPage + 1)">{{ t('creator.sales.next') }}</button>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { CourseCreator, CourseSettings, CreatorStatus, CreatorWithdrawal } from '@/api/admin/courses'
import type { Course, CourseDelivery } from '@/api/courses'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'

type Panel = 'creators' | 'reviews' | 'withdrawals'
const tabs: Panel[] = ['creators', 'reviews', 'withdrawals']
const markdownFields = ['intro_md', 'trial_md', 'faq_md'] as const

const { t } = useI18n()
const appStore = useAppStore()
const api = adminAPI.courses

const busy = ref(false)
const panel = ref<Panel>('creators')
const settings = reactive<CourseSettings>({ affiliate_rate_percent: 0, creator_enabled: false, creator_commission_percent: 20, creator_settle_days: 7, creator_withdraw_min_cny: 50 })
const creators = ref<CourseCreator[]>([])
const openCreator = ref(0)
const creatorForm = reactive<{ status: CreatorStatus; commission: string | number; note: string }>({ status: 'pending', commission: '', note: '' })
const reviews = ref<Course[]>([])
const openReview = ref(0)
const reviewNote = ref('')
const reviewDelivery = ref<CourseDelivery | null>(null)
const withdrawals = ref<CreatorWithdrawal[]>([])
const withdrawStatus = ref('pending')
const withdrawPage = ref(1)
const withdrawTotal = ref(0)
const pendingWithdrawals = ref(0)
const withdrawNotes = reactive<Record<number, string>>({})

const badges = computed<Record<Panel, number>>(() => ({
  creators: creators.value.filter((c) => c.status === 'pending').length,
  reviews: reviews.value.length,
  withdrawals: pendingWithdrawals.value,
}))

function fail(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

function formatDateTime(value: string) {
  return value ? new Date(value).toLocaleString() : ''
}

function creatorBadge(status: string) {
  return status === 'approved' ? 'badge-success' : status === 'pending' ? 'badge-warning' : status === 'suspended' ? 'badge-danger' : 'badge-gray'
}

/** What the reviewer should compare: price, limited-time price, trial video, and changes against the live version. */
function reviewLines(c: Course): string[] {
  const d = c.draft!
  const lines = [t('admin.creators.line.price', { price: d.price, original: d.original_price || '—' })]
  if (d.sale_price) lines.push(t('admin.creators.line.sale', { price: d.sale_price, until: formatDateTime(d.sale_ends_at || '') }))
  if (d.subtitle) lines.push(t('admin.creators.line.subtitle', { text: d.subtitle }))
  if (d.trial_video_url) lines.push(t('admin.creators.line.video', { url: d.trial_video_url }))
  if (c.approved_at) {
    const fields = ['title', 'subtitle', 'price', 'original_price', 'sale_price', 'intro_md', 'trial_md', 'faq_md', 'trial_video_url'] as const
    const changed: string[] = fields.filter((k) => (d[k] ?? '') !== (c[k] ?? ''))
    if (JSON.stringify(d.outline) !== JSON.stringify(c.outline)) changed.push('outline')
    if ((d.cover_url || '') !== (c.cover_url || '')) changed.push('cover')
    lines.push(t('admin.creators.line.changed', { fields: changed.length ? changed.map((k) => t(`admin.creators.field.${k}`)).join('、') : '—' }))
  }
  return lines
}

async function loadCreators() {
  creators.value = await api.creators()
}
async function loadReviews() {
  reviews.value = await api.reviews()
}
async function loadWithdrawals(status = withdrawStatus.value, page = 1) {
  try {
    const res = await api.creatorWithdrawals(status, page)
    withdrawals.value = res.items
    withdrawTotal.value = res.total
    withdrawStatus.value = status
    withdrawPage.value = page
    if (status === 'pending') pendingWithdrawals.value = res.total
  } catch (err) {
    fail(err)
  }
}

async function switchPanel(next: Panel) {
  panel.value = next
  try {
    if (next === 'creators') await loadCreators()
    else if (next === 'reviews') await loadReviews()
    else await loadWithdrawals()
  } catch (err) {
    fail(err)
  }
}

async function inspect(c: Course) {
  reviewDelivery.value = null
  if (openReview.value === c.id) {
    openReview.value = 0
    return
  }
  openReview.value = c.id
  reviewNote.value = ''
  try {
    reviewDelivery.value = (await api.deliveries(c.id))[0] || null
  } catch (err) {
    fail(err)
  }
}

function toggleCreator(c: CourseCreator) {
  if (openCreator.value === c.user_id) {
    openCreator.value = 0
    return
  }
  openCreator.value = c.user_id
  Object.assign(creatorForm, { status: c.status === 'pending' ? 'approved' : c.status, commission: c.commission_percent ?? '', note: c.admin_note })
}

async function run(fn: () => Promise<void>) {
  busy.value = true
  try {
    await fn()
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

function saveSettings() {
  return run(async () => {
    Object.assign(settings, await api.saveSettings({ ...settings }))
    appStore.showSuccess(t('common.saved'))
  })
}

function saveCreator(c: CourseCreator) {
  return run(async () => {
    const raw = String(creatorForm.commission).trim()
    await api.updateCreator(c.user_id, { status: creatorForm.status, commission_percent: raw === '' ? null : Number(raw), admin_note: creatorForm.note })
    openCreator.value = 0
    await loadCreators()
    appStore.showSuccess(t('common.saved'))
  })
}

function decide(c: Course, approve: boolean) {
  return run(async () => {
    await api.review(c.id, approve, reviewNote.value)
    reviewNote.value = ''
    openReview.value = 0
    await loadReviews()
    appStore.showSuccess(t(approve ? 'admin.creators.approved' : 'admin.creators.rejected'))
  })
}

function resolve(id: number, paid: boolean) {
  return run(async () => {
    const note = withdrawNotes[id] || ''
    if (paid) await api.payCreatorWithdrawal(id, note)
    else await api.rejectCreatorWithdrawal(id, note)
    await loadWithdrawals(withdrawStatus.value, withdrawPage.value)
    if (withdrawStatus.value !== 'pending') pendingWithdrawals.value = (await api.creatorWithdrawals('pending')).total
  })
}

onMounted(async () => {
  try {
    Object.assign(settings, await api.getSettings())
    await Promise.all([loadCreators(), loadReviews()])
    pendingWithdrawals.value = (await api.creatorWithdrawals('pending')).total
  } catch (err) {
    fail(err)
  }
})
</script>
