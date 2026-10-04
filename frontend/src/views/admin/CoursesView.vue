<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- List -->
      <template v-if="!editing">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.courses.hint') }}</p>
          <button class="btn btn-primary" data-testid="course-new" @click="startNew">{{ t('admin.courses.new') }}</button>
        </div>
        <div class="card flex flex-wrap items-end gap-3 p-4" data-testid="course-settings">
          <label class="text-sm">
            <span class="input-label">{{ t('admin.courses.rebateRate') }}</span>
            <input v-model.number="settings.affiliate_rate_percent" type="number" min="0" max="50" step="0.5" class="input w-32" data-testid="course-rebate-rate" />
          </label>
          <button class="btn btn-secondary" :disabled="busy" data-testid="course-settings-save" @click="saveSettings">{{ t('common.save') }}</button>
          <p class="min-w-0 flex-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.courses.rebateHint') }}</p>
        </div>
        <div v-if="!courses.length && !loading" class="card py-16 text-center text-sm text-gray-500">{{ t('admin.courses.empty') }}</div>
        <div v-else class="card overflow-x-auto">
          <table class="w-full whitespace-nowrap text-sm" data-testid="course-table">
            <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-2">{{ t('admin.courses.fields.title') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.fields.status') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.fields.price') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.students') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.revenue') }}</th>
                <th class="px-4 py-2" :title="t('admin.courses.funnelHint')">{{ t('admin.courses.funnel') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.refundsCol') }}</th>
                <th class="px-4 py-2">{{ t('admin.courses.delivery') }}</th>
                <th class="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="c in courses" :key="c.id">
                <td class="min-w-[12rem] whitespace-normal px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ c.title }}</div>
                  <div class="text-xs text-gray-400">/courses/{{ c.slug }}</div>
                </td>
                <td class="px-4 py-3"><span :class="['badge', statusBadge(c.status)]">{{ t(`admin.courses.status.${c.status}`) }}</span></td>
                <td class="px-4 py-3">
                  ¥{{ c.price }}
                  <div v-if="c.sale_active" class="text-xs text-rose-500">{{ t('admin.courses.saleNow', { price: c.sale_price }) }}</div>
                </td>
                <td class="px-4 py-3">{{ c.student_count }}</td>
                <td class="px-4 py-3">¥{{ (c.revenue || 0).toFixed(2) }}</td>
                <td class="px-4 py-3 text-xs text-gray-600 dark:text-gray-300" data-testid="course-funnel">{{ c.views_30d || 0 }} / {{ c.orders_30d || 0 }} / {{ c.paid_30d || 0 }}</td>
                <td class="px-4 py-3">{{ c.refunds || 0 }}</td>
                <td class="px-4 py-3">
                  <span v-if="c.delivery_version" class="text-gray-600 dark:text-gray-300">{{ t('admin.courses.version', { v: c.delivery_version }) }}</span>
                  <span v-else class="text-amber-600">{{ t('admin.courses.noDelivery') }}</span>
                </td>
                <td class="px-4 py-3 text-right">
                  <button class="btn btn-secondary btn-sm" @click="open(c)">{{ t('common.edit') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- Editor -->
      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <button class="btn btn-secondary btn-sm" @click="closeEditor">← {{ t('admin.courses.backToList') }}</button>
          <div v-if="current" class="flex gap-1">
            <button v-for="tab in ['basic', 'delivery', 'students'] as const" :key="tab" :class="['btn btn-sm', panel === tab ? 'btn-primary' : 'btn-secondary']" :data-testid="`course-tab-${tab}`" @click="switchPanel(tab)">
              {{ t(`admin.courses.tabs.${tab}`) }}
            </button>
          </div>
        </div>

        <!-- Basic -->
        <div v-if="panel === 'basic'" class="card space-y-4 p-5" data-testid="course-form">
          <div class="grid gap-4 md:grid-cols-2">
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.title') }}</span>
              <input v-model="form.title" class="input" maxlength="120" data-testid="course-title" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.slug') }}</span>
              <input v-model="form.slug" class="input" maxlength="64" placeholder="ai-agent-101" data-testid="course-slug" />
              <span class="input-hint">{{ t('admin.courses.slugHint') }}</span>
            </label>
            <label class="text-sm md:col-span-2">
              <span class="input-label">{{ t('admin.courses.fields.subtitle') }}</span>
              <input v-model="form.subtitle" class="input" maxlength="200" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.category') }}</span>
              <input v-model="form.category" class="input" maxlength="32" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.status') }}</span>
              <select v-model="form.status" class="input" data-testid="course-status">
                <option v-for="s in ['draft', 'published', 'archived']" :key="s" :value="s">{{ t(`admin.courses.status.${s}`) }}</option>
              </select>
              <span class="input-hint">{{ t('admin.courses.statusHint') }}</span>
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.price') }}</span>
              <input v-model.number="form.price" type="number" min="0.01" step="0.01" class="input" data-testid="course-price" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.originalPrice') }}</span>
              <input v-model.number="form.original_price" type="number" min="0" step="0.01" class="input" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.salePrice') }}</span>
              <input v-model.number="form.sale_price" type="number" min="0" step="0.01" class="input" data-testid="course-sale-price" />
              <span class="input-hint">{{ t('admin.courses.salePriceHint') }}</span>
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.saleEndsAt') }}</span>
              <input v-model="saleEndsLocal" type="datetime-local" class="input" data-testid="course-sale-ends" />
            </label>
            <label class="flex items-center gap-2 text-sm md:col-span-2">
              <input v-model="form.edu_discount" type="checkbox" class="h-4 w-4" data-testid="course-edu" />
              {{ t('admin.courses.fields.eduDiscount') }}
            </label>
            <label class="text-sm md:col-span-2">
              <span class="input-label">{{ t('admin.courses.fields.trialVideo') }}</span>
              <input v-model="form.trial_video_url" class="input" placeholder="https://www.bilibili.com/video/BV... / https://.../trial.mp4" data-testid="course-trial-url" />
              <span class="input-hint">{{ t('admin.courses.trialVideoHint') }}</span>
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.sortOrder') }}</span>
              <input v-model.number="form.sort_order" type="number" class="input" />
            </label>
            <div class="text-sm">
              <span class="input-label">{{ t('admin.courses.fields.cover') }}</span>
              <div v-if="current" class="flex items-center gap-3">
                <img v-if="current.cover_url" :src="current.cover_url" alt="" class="h-16 w-28 rounded object-cover" />
                <label class="btn btn-secondary btn-sm cursor-pointer">
                  {{ t('admin.courses.uploadCover') }}
                  <input type="file" accept="image/png,image/jpeg,image/webp" class="hidden" @change="onCover" />
                </label>
              </div>
              <span v-else class="input-hint">{{ t('admin.courses.coverAfterSave') }}</span>
            </div>
          </div>

          <!-- Outline -->
          <div class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700">
            <div class="flex items-center justify-between">
              <span class="text-sm font-medium">{{ t('admin.courses.fields.outline') }}</span>
              <button class="btn btn-secondary btn-sm" @click="form.outline.push({ title: '', lessons: [{ title: '', duration: '', trial: false }] })">{{ t('admin.courses.addSection') }}</button>
            </div>
            <div v-for="(section, si) in form.outline" :key="si" class="space-y-2 rounded-xl border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex gap-2">
                <input v-model="section.title" class="input" :placeholder="t('admin.courses.sectionTitle')" />
                <button class="btn btn-secondary btn-sm shrink-0 whitespace-nowrap" @click="form.outline.splice(si, 1)">{{ t('common.delete') }}</button>
              </div>
              <div v-for="(lesson, li) in section.lessons" :key="li" class="flex flex-wrap items-center gap-2 pl-4">
                <input v-model="lesson.title" class="input min-w-0 flex-1" :placeholder="t('admin.courses.lessonTitle')" />
                <input v-model="lesson.duration" class="input w-24" :placeholder="t('admin.courses.duration')" />
                <label class="flex items-center gap-1 text-xs"><input v-model="lesson.trial" type="checkbox" />{{ t('admin.courses.trial') }}</label>
                <button class="text-xs text-gray-400 hover:text-red-500" @click="section.lessons.splice(li, 1)">✕</button>
              </div>
              <button class="pl-4 text-xs text-primary-600" @click="section.lessons.push({ title: '', duration: '', trial: false })">+ {{ t('admin.courses.addLesson') }}</button>
            </div>
          </div>

          <!-- Markdown -->
          <div v-for="field in markdownFields" :key="field" class="space-y-1 border-t border-gray-100 pt-4 dark:border-dark-700">
            <div class="flex items-center justify-between">
              <span class="text-sm font-medium">{{ t(`admin.courses.fields.${field}`) }}</span>
              <div class="flex gap-2">
                <label class="btn btn-secondary btn-sm cursor-pointer">
                  {{ t('admin.courses.insertImage') }}
                  <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" class="hidden" @change="(e) => onInsertImage(e, field)" />
                </label>
                <button class="btn btn-secondary btn-sm" @click="preview[field] = !preview[field]">{{ preview[field] ? t('admin.courses.edit') : t('admin.courses.preview') }}</button>
              </div>
            </div>
            <div v-if="preview[field]" class="course-md min-h-24 rounded-lg border border-gray-200 p-3 dark:border-dark-700" v-html="renderMarkdown(form[field])"></div>
            <textarea v-else v-model="form[field]" rows="8" class="input font-mono text-xs" :placeholder="t(`admin.courses.placeholders.${field}`)"></textarea>
          </div>

          <div class="flex flex-wrap gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
            <button class="btn btn-primary" :disabled="busy" data-testid="course-save" @click="save">{{ t('common.save') }}</button>
            <a v-if="current && current.status !== 'draft'" :href="`/courses/${current.slug}`" target="_blank" class="btn btn-secondary">{{ t('admin.courses.viewPage') }}</a>
            <button v-if="current && !current.student_count" class="btn btn-danger ml-auto" :disabled="busy" @click="confirmDelete = true">{{ t('common.delete') }}</button>
          </div>
        </div>

        <!-- Delivery -->
        <div v-else-if="panel === 'delivery' && current" class="space-y-4">
          <div class="card space-y-3 p-5" data-testid="course-delivery-form">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.courses.deliveryHint') }}</p>
            <label class="block text-sm">
              <span class="input-label">{{ t('admin.courses.fields.link') }}</span>
              <input v-model="delivery.link" class="input" placeholder="https://pan.baidu.com/s/..." data-testid="delivery-link" />
            </label>
            <div class="grid gap-3 md:grid-cols-2">
              <label class="text-sm">
                <span class="input-label">{{ t('admin.courses.fields.code') }}</span>
                <input v-model="delivery.code" class="input" maxlength="32" data-testid="delivery-code" />
              </label>
              <label class="text-sm">
                <span class="input-label">{{ t('admin.courses.fields.password') }}</span>
                <input v-model="delivery.password" class="input" maxlength="64" />
              </label>
            </div>
            <label class="block text-sm">
              <span class="input-label">{{ t('admin.courses.fields.note') }}</span>
              <textarea v-model="delivery.note" rows="3" class="input" maxlength="2000"></textarea>
            </label>
            <label v-if="deliveries.length" class="flex items-center gap-2 text-sm">
              <input v-model="delivery.notify" type="checkbox" class="h-4 w-4" data-testid="delivery-notify" />
              {{ t('admin.courses.notifyStudents', { count: current.student_count }) }}
            </label>
            <button class="btn btn-primary" :disabled="busy || !delivery.link" data-testid="delivery-save" @click="saveDelivery">{{ t('admin.courses.saveDelivery') }}</button>
          </div>
          <div class="card p-5">
            <h3 class="mb-3 text-sm font-medium">{{ t('admin.courses.history') }}</h3>
            <p v-if="!deliveries.length" class="text-sm text-gray-500">{{ t('admin.courses.noDelivery') }}</p>
            <ul class="space-y-2 text-sm">
              <li v-for="d in deliveries" :key="d.version" class="rounded-lg border border-gray-100 p-3 dark:border-dark-700">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="badge badge-gray">v{{ d.version }}</span>
                  <span class="text-xs text-gray-400">{{ formatDateTime(d.updated_at) }}</span>
                  <span v-if="d.version === deliveries[0].version" class="badge badge-success">{{ t('admin.courses.current') }}</span>
                </div>
                <div class="mt-1 break-all font-mono text-xs">{{ d.link }}<template v-if="d.code"> · {{ t('admin.courses.fields.code') }} {{ d.code }}</template><template v-if="d.password"> · {{ t('admin.courses.fields.password') }} {{ d.password }}</template></div>
                <div v-if="d.note" class="mt-1 whitespace-pre-wrap text-xs text-gray-500">{{ d.note }}</div>
              </li>
            </ul>
          </div>
        </div>

        <!-- Students -->
        <div v-else-if="panel === 'students' && current" class="space-y-4">
          <div class="card flex flex-wrap items-end gap-2 p-5">
            <label class="min-w-48 flex-1 text-sm">
              <span class="input-label">{{ t('admin.courses.grantUser') }}</span>
              <input v-model="grantUser" class="input" :placeholder="t('admin.courses.grantUserPlaceholder')" data-testid="grant-user" />
            </label>
            <label class="min-w-48 flex-1 text-sm">
              <span class="input-label">{{ t('admin.courses.fields.note') }}</span>
              <input v-model="grantNote" class="input" maxlength="200" />
            </label>
            <button class="btn btn-primary" :disabled="busy || !grantUser.trim()" data-testid="grant-submit" @click="grant">{{ t('admin.courses.grant') }}</button>
          </div>
          <div class="card overflow-x-auto">
            <table class="w-full whitespace-nowrap text-sm" data-testid="student-table">
              <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-2">{{ t('admin.courses.user') }}</th>
                  <th class="px-4 py-2">{{ t('admin.courses.source') }}</th>
                  <th class="px-4 py-2">{{ t('admin.courses.enrolledAt') }}</th>
                  <th class="px-4 py-2">{{ t('admin.courses.firstViewed') }}</th>
                  <th class="px-4 py-2">{{ t('admin.courses.views') }}</th>
                  <th class="px-4 py-2">{{ t('admin.courses.recentIps') }}</th>
                  <th class="px-4 py-2"></th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="s in students" :key="s.user_id">
                  <td class="px-4 py-2">
                    <div>{{ s.username || s.user_email }}</div>
                    <div class="text-xs text-gray-400">#{{ s.user_id }} {{ s.user_email }}</div>
                  </td>
                  <td class="px-4 py-2">
                    {{ t(`admin.courses.sources.${s.source}`) }}<span v-if="s.order_id" class="text-xs text-gray-400"> #{{ s.order_id }}</span>
                    <div v-if="s.note" class="text-xs text-gray-400">{{ s.note }}</div>
                  </td>
                  <td class="px-4 py-2 text-xs">{{ formatDateTime(s.created_at) }}</td>
                  <td class="px-4 py-2 text-xs">{{ s.first_viewed_at ? formatDateTime(s.first_viewed_at) : t('admin.courses.notViewed') }}</td>
                  <td class="px-4 py-2">{{ s.view_count }}</td>
                  <td class="px-4 py-2">
                    <span :class="s.recent_ips >= 4 ? 'font-semibold text-red-600' : ''" :title="s.recent_ips >= 4 ? t('admin.courses.ipWarning') : ''">{{ s.recent_ips }}</span>
                  </td>
                  <td class="px-4 py-2 text-right">
                    <span v-if="s.refunded" class="badge badge-gray">{{ t('admin.courses.refunded') }}</span>
                    <span v-else-if="s.status === 'revoked'" class="badge badge-gray">{{ t('admin.courses.revoked') }}</span>
                    <button v-else class="btn btn-secondary btn-sm" :disabled="busy" @click="revoke(s.user_id)">{{ t('admin.courses.revoke') }}</button>
                  </td>
                </tr>
              </tbody>
            </table>
            <p v-if="!students.length" class="p-6 text-center text-sm text-gray-500">{{ t('admin.courses.noStudents') }}</p>
          </div>
          <div v-if="studentTotal > 30" class="flex justify-end gap-2">
            <button class="btn btn-secondary btn-sm" :disabled="studentPage <= 1" @click="loadStudents(studentPage - 1)">‹</button>
            <span class="text-sm text-gray-500">{{ studentPage }} / {{ Math.ceil(studentTotal / 30) }}</span>
            <button class="btn btn-secondary btn-sm" :disabled="studentPage * 30 >= studentTotal" @click="loadStudents(studentPage + 1)">›</button>
          </div>
        </div>
      </template>
    </div>

    <ConfirmDialog
      :show="confirmDelete"
      :title="t('admin.courses.deleteTitle')"
      :message="t('admin.courses.deleteMessage', { title: current?.title || '' })"
      danger
      @confirm="remove"
      @cancel="confirmDelete = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { adminAPI } from '@/api/admin'
import type { CourseInput, CourseSettings, DeliveryInput } from '@/api/admin/courses'
import type { Course, CourseDelivery, CourseEnrollment } from '@/api/courses'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'

type MarkdownField = 'intro_md' | 'trial_md' | 'faq_md'
type Panel = 'basic' | 'delivery' | 'students'

const { t } = useI18n()
const appStore = useAppStore()
const api = adminAPI.courses

const markdownFields: MarkdownField[] = ['intro_md', 'trial_md', 'faq_md']
const courses = ref<Course[]>([])
const loading = ref(true)
const busy = ref(false)
const editing = ref(false)
const current = ref<Course | null>(null)
const panel = ref<Panel>('basic')
const preview = reactive<Record<MarkdownField, boolean>>({ intro_md: false, trial_md: false, faq_md: false })
const confirmDelete = ref(false)

const emptyForm = (): CourseInput => ({
  slug: '', title: '', subtitle: '', category: '', price: 0, original_price: 0,
  intro_md: '', outline: [], trial_md: '', faq_md: '', status: 'draft', sort_order: 0,
  sale_price: 0, sale_ends_at: null, edu_discount: true, trial_video_url: '',
})
const form = reactive<CourseInput>(emptyForm())
const delivery = reactive<DeliveryInput>({ link: '', code: '', password: '', note: '', notify: true })
const settings = reactive<CourseSettings>({ affiliate_rate_percent: 0 })

// <input type="datetime-local"> works in local time without a zone; the API takes ISO times.
const saleEndsLocal = computed({
  get: () => {
    if (!form.sale_ends_at) return ''
    const d = new Date(form.sale_ends_at)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
  },
  set: (v: string) => {
    form.sale_ends_at = v ? new Date(v).toISOString() : null
  },
})
const deliveries = ref<CourseDelivery[]>([])
const students = ref<CourseEnrollment[]>([])
const studentTotal = ref(0)
const studentPage = ref(1)
const grantUser = ref('')
const grantNote = ref('')

function fail(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

function statusBadge(status: string) {
  return status === 'published' ? 'badge-success' : status === 'archived' ? 'badge-gray' : 'badge-warning'
}

function formatDateTime(value: string) {
  return value ? new Date(value).toLocaleString() : ''
}

async function loadList() {
  loading.value = true
  try {
    courses.value = await api.list()
  } catch (err) {
    fail(err)
  } finally {
    loading.value = false
  }
}

function fillForm(c: Course | null) {
  Object.assign(form, emptyForm())
  if (!c) return
  Object.assign(form, {
    slug: c.slug, title: c.title, subtitle: c.subtitle, category: c.category, price: c.price, original_price: c.original_price,
    intro_md: c.intro_md || '', trial_md: c.trial_md || '', faq_md: c.faq_md || '', status: c.status, sort_order: c.sort_order,
    outline: JSON.parse(JSON.stringify(c.outline || [])),
    sale_price: c.sale_price || 0, sale_ends_at: c.sale_ends_at || null, edu_discount: c.edu_discount, trial_video_url: c.trial_video_url || '',
  })
}

function startNew() {
  current.value = null
  fillForm(null)
  panel.value = 'basic'
  editing.value = true
}

async function open(c: Course) {
  try {
    current.value = await api.get(c.id)
    fillForm(current.value)
    panel.value = 'basic'
    editing.value = true
  } catch (err) {
    fail(err)
  }
}

function closeEditor() {
  editing.value = false
  current.value = null
  void loadList()
}

async function switchPanel(next: Panel) {
  panel.value = next
  if (!current.value) return
  if (next === 'delivery') {
    try {
      deliveries.value = await api.deliveries(current.value.id)
    } catch (err) {
      fail(err)
    }
  } else if (next === 'students') {
    await loadStudents(1)
  }
}

async function save() {
  busy.value = true
  try {
    current.value = current.value ? await api.update(current.value.id, { ...form }) : await api.create({ ...form })
    fillForm(current.value)
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function remove() {
  confirmDelete.value = false
  if (!current.value) return
  busy.value = true
  try {
    await api.remove(current.value.id)
    closeEditor()
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

function pickFile(e: Event): File | null {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0] || null
  input.value = ''
  return file
}

async function onCover(e: Event) {
  const file = pickFile(e)
  if (!file || !current.value) return
  busy.value = true
  try {
    current.value = await api.uploadCover(current.value.id, file)
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function onInsertImage(e: Event, field: MarkdownField) {
  const file = pickFile(e)
  if (!file) return
  busy.value = true
  try {
    const url = await api.uploadImage(file)
    form[field] = `${form[field]}${form[field] && !form[field].endsWith('\n') ? '\n' : ''}![](${url})\n`
    preview[field] = false
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function saveDelivery() {
  if (!current.value) return
  busy.value = true
  try {
    await api.saveDelivery(current.value.id, { ...delivery })
    Object.assign(delivery, { link: '', code: '', password: '', note: '', notify: true })
    deliveries.value = await api.deliveries(current.value.id)
    appStore.showSuccess(t('admin.courses.deliverySaved'))
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function loadStudents(page: number) {
  if (!current.value) return
  try {
    const res = await api.enrollments(current.value.id, page)
    students.value = res.items
    studentTotal.value = res.total
    studentPage.value = page
  } catch (err) {
    fail(err)
  }
}

async function grant() {
  if (!current.value) return
  busy.value = true
  try {
    await api.grant(current.value.id, grantUser.value.trim(), grantNote.value.trim())
    grantUser.value = ''
    grantNote.value = ''
    await loadStudents(1)
    appStore.showSuccess(t('admin.courses.granted'))
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function revoke(userId: number) {
  if (!current.value) return
  busy.value = true
  try {
    await api.revoke(current.value.id, userId)
    await loadStudents(studentPage.value)
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

async function loadSettings() {
  try {
    Object.assign(settings, await api.getSettings())
  } catch {
    // keep defaults
  }
}

async function saveSettings() {
  busy.value = true
  try {
    Object.assign(settings, await api.saveSettings({ ...settings }))
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    fail(err)
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  void loadList()
  void loadSettings()
})
</script>

<style scoped>
.course-md :deep(img) {
  @apply my-2 max-w-full rounded;
}
.course-md :deep(h2),
.course-md :deep(h3) {
  @apply mb-1 mt-3 font-semibold;
}
.course-md :deep(ul) {
  @apply list-disc pl-5;
}
.course-md :deep(p) {
  @apply text-sm leading-6;
}
</style>
