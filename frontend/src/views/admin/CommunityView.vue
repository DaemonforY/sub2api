<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-wrap gap-2">
        <button v-for="tabKey in tabs" :key="tabKey" :class="['btn btn-sm', tab === tabKey ? 'btn-primary' : 'btn-secondary']" @click="switchTab(tabKey)">
          {{ t(`admin.community.tabs.${tabKey}`) }}
        </button>
      </div>

      <div v-if="tab === 'settings'" class="card space-y-3 p-5" data-testid="community-settings">
        <label class="flex items-center gap-2 text-sm">
          <input v-model="settings.review_all" type="checkbox" class="h-4 w-4" />
          {{ t('admin.community.reviewAll') }}
        </label>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.community.reviewAllHint') }}</p>
        <div class="space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
          <div class="text-sm font-medium">{{ t('admin.community.commentSettings.title') }}</div>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="settings.comments_enabled" type="checkbox" class="h-4 w-4" data-testid="comments-enabled" />
            {{ t('admin.community.commentSettings.enabled') }}
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input v-model="settings.comments_review_all" type="checkbox" class="h-4 w-4" :disabled="!settings.comments_enabled" data-testid="comments-review-all" />
            {{ t('admin.community.commentSettings.reviewAll') }}
          </label>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.community.commentSettings.hint') }}</p>
        </div>
        <div class="border-t border-gray-100 pt-3 dark:border-dark-700">
          <div class="text-sm font-medium">{{ t('admin.community.cloud.title') }}</div>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.community.cloud.hint') }}</p>
          <div class="mt-2 grid max-w-md grid-cols-2 gap-3">
            <label class="text-xs text-gray-600 dark:text-dark-300">
              {{ t('admin.community.cloud.free') }}
              <input v-model.number="settings.cloud_quota_mb" type="number" min="0" class="input mt-1" data-testid="cloud-quota" />
            </label>
            <label class="text-xs text-gray-600 dark:text-dark-300">
              {{ t('admin.community.cloud.subscriber') }}
              <input v-model.number="settings.cloud_subscriber_quota_mb" type="number" min="0" class="input mt-1" data-testid="cloud-subscriber-quota" />
            </label>
          </div>
        </div>
        <button class="btn btn-primary btn-sm" :disabled="busy" @click="saveSettings">{{ t('common.save') }}</button>
      </div>

      <div v-else-if="tab === 'restricted'" class="card overflow-x-auto" data-testid="community-restricted">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.author') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.email') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.works') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.since') }}</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="a in restrictedList" :key="a.user_id" data-testid="community-restricted-row">
              <td class="px-4 py-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ a.display_name || a.handle }}</span>
                <span class="ml-1 text-xs text-gray-500">@{{ a.handle }}</span>
              </td>
              <td class="px-4 py-2 text-gray-600 dark:text-dark-300">{{ a.email }}</td>
              <td class="px-4 py-2">{{ a.works_count }}</td>
              <td class="px-4 py-2 text-gray-500">{{ formatDateTime(a.updated_at) }}</td>
              <td class="px-4 py-2 text-right">
                <button class="btn btn-primary btn-sm" :disabled="busy" @click="unrestrict(a)">{{ t('admin.community.actions.unban') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!restrictedList.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.community.noRestricted') }}</p>
      </div>

      <div v-else-if="tab === 'comments'" class="space-y-3" data-testid="community-comments">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="f in commentFilters"
            :key="f"
            :class="['btn btn-sm', commentFilter === f ? 'btn-primary' : 'btn-secondary']"
            @click="switchCommentFilter(f)"
          >
            {{ t(`admin.community.commentFilters.${f || 'all'}`) }}
          </button>
        </div>
        <div class="card overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead class="text-left text-xs text-gray-500 dark:text-dark-400">
              <tr>
                <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.time') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.author') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.comment') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.work') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.status') }}</th>
                <th class="px-4 py-2" />
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="c in commentList" :key="c.id" data-testid="community-comment">
                <td class="whitespace-nowrap px-4 py-2 text-gray-500">{{ formatDateTime(c.created_at) }}</td>
                <td class="whitespace-nowrap px-4 py-2">{{ c.author?.display_name || c.author?.handle }}<span class="ml-1 text-xs text-gray-500">@{{ c.author?.handle }}</span></td>
                <td class="max-w-md px-4 py-2">
                  <span class="whitespace-pre-wrap break-words">{{ c.body }}</span>
                  <span v-if="c.review_flags?.length" class="block text-xs text-amber-700 dark:text-amber-300">{{ t('admin.community.flags') }}：{{ c.review_flags.join('、') }}</span>
                  <span v-if="c.report_count" class="block text-xs text-red-600">{{ t('admin.community.reports', { count: c.report_count }) }}</span>
                </td>
                <td class="px-4 py-2"><a :href="workUrl(c.work_id) + '#comments'" target="_blank" rel="noopener" class="text-primary-600 hover:underline">#{{ c.work_id }} {{ c.work_title }}</a></td>
                <td class="whitespace-nowrap px-4 py-2"><span :class="['badge', commentBadge(c.status)]">{{ t(`admin.community.commentStatus.${c.status}`) }}</span></td>
                <td class="space-x-1 whitespace-nowrap px-4 py-2 text-right">
                  <button v-if="c.status === 'pending' || c.status === 'hidden'" class="btn btn-primary btn-sm" :disabled="busy" @click="actComment(c, 'approve')">{{ t('admin.community.actions.approve') }}</button>
                  <button v-if="c.status === 'pending' || c.status === 'approved'" class="btn btn-secondary btn-sm" :disabled="busy" @click="actComment(c, 'hide')">{{ t('admin.community.actions.hide') }}</button>
                  <button class="btn btn-danger btn-sm" :disabled="busy" @click="banUser(c.owner_id, c.author?.handle || '')">{{ t('admin.community.actions.ban') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-if="!commentList.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.community.noComments') }}</p>
        </div>
        <div v-if="commentList.length >= 50 || page > 1" class="flex justify-center gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="go(page - 1)">{{ t('admin.community.prev') }}</button>
          <button class="btn btn-secondary btn-sm" :disabled="commentList.length < 50" @click="go(page + 1)">{{ t('admin.community.next') }}</button>
        </div>
      </div>

      <div v-else-if="tab === 'reports'" class="card overflow-x-auto" data-testid="community-reports">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs text-gray-500 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.time') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.work') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.reason') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('admin.community.columns.status') }}</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="r in reportList" :key="r.id">
              <td class="px-4 py-2 text-gray-500">{{ formatDateTime(r.created_at) }}</td>
              <td class="px-4 py-2">
                <a :href="workUrl(r.work_id) + (r.comment_id ? '#comments' : '')" target="_blank" rel="noopener" class="text-primary-600 hover:underline">#{{ r.work_id }} {{ r.work_title }}</a>
                <span v-if="r.comment_id" class="mt-0.5 block max-w-md text-xs text-gray-600 dark:text-dark-300" data-testid="report-comment">
                  {{ t('admin.community.reportedComment') }}：{{ r.comment_body }}
                  <span v-if="r.comment_status && r.comment_status !== 'approved'" class="text-gray-400">（{{ t(`admin.community.commentStatus.${r.comment_status}`) }}）</span>
                </span>
              </td>
              <td class="px-4 py-2">{{ t(`admin.community.reasons.${r.reason}`) }}<span v-if="r.detail" class="block text-xs text-gray-500">{{ r.detail }}</span></td>
              <td class="px-4 py-2">{{ t(`admin.community.reportStatus.${r.status}`) }}</td>
              <td class="space-x-1 whitespace-nowrap px-4 py-2 text-right">
                <button v-if="r.status === 'open'" class="btn btn-secondary btn-sm" @click="setReport(r.id, 'resolved')">{{ t('admin.community.resolve') }}</button>
                <button v-if="r.status === 'open'" class="btn btn-secondary btn-sm" @click="setReport(r.id, 'dismissed')">{{ t('admin.community.dismiss') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!reportList.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.community.noReports') }}</p>
      </div>

      <template v-else>
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3" data-testid="community-works">
          <div v-for="w in workList" :key="w.id" class="card overflow-hidden" data-testid="community-work">
            <a :href="workUrl(w.id)" target="_blank" rel="noopener" class="block bg-gray-100 dark:bg-dark-800">
              <img :src="w.cover_thumb_url" alt="" class="h-48 w-full object-cover" loading="lazy" />
            </a>
            <div class="space-y-1.5 p-3 text-sm">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ w.title || t('admin.community.untitled') }}</span>
                <span :class="['badge', statusBadge(w.status)]">{{ t(`admin.community.status.${w.status}`) }}</span>
                <span v-if="w.featured" class="badge badge-purple">{{ t('admin.community.featured') }}</span>
              </div>
              <div class="text-xs text-gray-500">
                @{{ w.author.handle }} · {{ formatDateTime(w.created_at) }} · {{ w.image_count }} {{ t('admin.community.images') }} · ♥ {{ w.like_count }}
                <span v-if="w.report_count" class="text-red-600"> · {{ t('admin.community.reports', { count: w.report_count }) }}</span>
              </div>
              <p v-if="w.review_flags?.length" class="text-xs text-amber-700 dark:text-amber-300">{{ t('admin.community.flags') }}：{{ w.review_flags.join('、') }}</p>
              <p v-if="w.review_reason" class="text-xs text-gray-500">{{ w.review_reason }}</p>
              <p v-if="w.prompt" class="line-clamp-3 text-xs text-gray-600 dark:text-dark-300">{{ w.prompt }}</p>
              <div class="flex flex-wrap gap-1 pt-1">
                <button v-if="w.status !== 'approved'" class="btn btn-primary btn-sm" :disabled="busy" @click="act(w, 'approve')">{{ t('admin.community.actions.approve') }}</button>
                <button v-if="w.status === 'pending'" class="btn btn-secondary btn-sm" :disabled="busy" @click="ask(w, 'reject')">{{ t('admin.community.actions.reject') }}</button>
                <button v-if="w.status === 'approved'" class="btn btn-secondary btn-sm" :disabled="busy" @click="ask(w, 'hide')">{{ t('admin.community.actions.hide') }}</button>
                <button v-if="w.status === 'approved'" class="btn btn-secondary btn-sm" :disabled="busy" @click="act(w, w.featured ? 'unfeature' : 'feature')">{{ w.featured ? t('admin.community.actions.unfeature') : t('admin.community.actions.feature') }}</button>
                <button class="btn btn-danger btn-sm" :disabled="busy" @click="banAuthor(w)">{{ t('admin.community.actions.ban') }}</button>
              </div>
            </div>
          </div>
        </div>
        <p v-if="!workList.length" class="card p-8 text-center text-sm text-gray-500">{{ t('admin.community.empty') }}</p>
        <div v-if="workList.length >= 30 || page > 1" class="flex justify-center gap-2">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="go(page - 1)">{{ t('admin.community.prev') }}</button>
          <button class="btn btn-secondary btn-sm" :disabled="workList.length < 30" @click="go(page + 1)">{{ t('admin.community.next') }}</button>
        </div>
      </template>
    </div>

    <BaseDialog :show="!!reasonFor" :title="reasonFor?.action === 'hide' ? t('admin.community.actions.hide') : t('admin.community.actions.reject')" width="narrow" @close="reasonFor = null">
      <textarea v-model="reason" class="input min-h-[80px]" maxlength="200" :placeholder="t('admin.community.reasonPlaceholder')" data-testid="community-reason" />
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="reasonFor = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="busy" data-testid="community-reason-confirm" @click="confirmReason">{{ t('common.confirm') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { AdminComment, AdminCommentStatus, AdminCommunityWork, AdminWorkReport, CommunitySettings, CommunityWorkStatus, ModerateAction, RestrictedAuthor } from '@/api/admin/community'
import { CANVAS_SITE_URL } from '@/constants/crossSites'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

type Tab = 'pending' | 'reported' | 'approved' | 'hidden' | 'comments' | 'reports' | 'restricted' | 'settings'
const tabs: Tab[] = ['pending', 'reported', 'approved', 'hidden', 'comments', 'reports', 'restricted', 'settings']
const commentFilters = ['pending', 'reported', 'hidden', ''] as const
type CommentFilter = (typeof commentFilters)[number]

const { t } = useI18n()
const appStore = useAppStore()
const tab = ref<Tab>('pending')
const page = ref(1)
const busy = ref(false)
const workList = ref<AdminCommunityWork[]>([])
const reportList = ref<AdminWorkReport[]>([])
const restrictedList = ref<RestrictedAuthor[]>([])
const commentList = ref<AdminComment[]>([])
const commentFilter = ref<CommentFilter>('pending')
const settings = ref<CommunitySettings>({ review_all: false, comments_enabled: true, comments_review_all: false, cloud_quota_mb: 200, cloud_subscriber_quota_mb: 2048 })
const reasonFor = ref<{ work: AdminCommunityWork; action: 'reject' | 'hide' } | null>(null)
const reason = ref('')

const canvasBase = CANVAS_SITE_URL.replace(/\/+$/, '')
const workUrl = (id: number) => `${canvasBase}/w/${id}`

function statusBadge(status: CommunityWorkStatus) {
  return { approved: 'badge-success', pending: 'badge-warning', rejected: 'badge-danger', hidden: 'badge-gray' }[status]
}

function commentBadge(status: AdminCommentStatus) {
  return { approved: 'badge-success', pending: 'badge-warning', hidden: 'badge-gray', deleted: 'badge-gray' }[status]
}

function showError(error: unknown) {
  appStore.showError(extractApiErrorMessage(error, t('common.error')))
}

async function load() {
  try {
    if (tab.value === 'settings') settings.value = await adminAPI.community.getSettings()
    else if (tab.value === 'comments') commentList.value = await adminAPI.community.comments(commentFilter.value, page.value)
    else if (tab.value === 'reports') reportList.value = await adminAPI.community.reports('', page.value)
    else if (tab.value === 'restricted') restrictedList.value = await adminAPI.community.restricted()
    else workList.value = await adminAPI.community.works(tab.value, page.value)
  } catch (error) {
    showError(error)
  }
}

function switchTab(next: Tab) {
  tab.value = next
  page.value = 1
  void load()
}

function switchCommentFilter(next: CommentFilter) {
  commentFilter.value = next
  page.value = 1
  void load()
}

async function actComment(comment: AdminComment, action: 'approve' | 'hide') {
  busy.value = true
  try {
    await adminAPI.community.moderateComment(comment.id, action)
    appStore.showSuccess(t('admin.community.done'))
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

function go(next: number) {
  page.value = next
  void load()
}

async function act(work: AdminCommunityWork, action: ModerateAction, why = '') {
  busy.value = true
  try {
    await adminAPI.community.moderate(work.id, action, why)
    appStore.showSuccess(t('admin.community.done'))
    await load()
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

function ask(work: AdminCommunityWork, action: 'reject' | 'hide') {
  reasonFor.value = { work, action }
  reason.value = ''
}

async function confirmReason() {
  if (!reasonFor.value) return
  const { work, action } = reasonFor.value
  reasonFor.value = null
  await act(work, action, reason.value)
}

function banAuthor(work: AdminCommunityWork) {
  return banUser(work.owner_id, work.author.handle)
}

async function banUser(userId: number, handle: string) {
  if (!window.confirm(t('admin.community.banConfirm', { handle }))) return
  busy.value = true
  try {
    await adminAPI.community.ban(userId, true)
    appStore.showSuccess(t('admin.community.banned'))
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function unrestrict(author: RestrictedAuthor) {
  busy.value = true
  try {
    await adminAPI.community.ban(author.user_id, false)
    restrictedList.value = restrictedList.value.filter((a) => a.user_id !== author.user_id)
    appStore.showSuccess(t('admin.community.unrestricted', { handle: author.handle }))
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function setReport(id: number, status: AdminWorkReport['status']) {
  try {
    await adminAPI.community.setReport(id, status)
    await load()
  } catch (error) {
    showError(error)
  }
}

async function saveSettings() {
  busy.value = true
  try {
    settings.value = await adminAPI.community.saveSettings(settings.value)
    appStore.showSuccess(t('admin.community.done'))
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>
