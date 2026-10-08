<template>
  <div class="space-y-4" data-testid="creator-editor">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <button class="btn btn-secondary btn-sm" @click="emit('close')">← {{ t('creator.backToCourses') }}</button>
      <div v-if="current" class="flex flex-wrap gap-2">
        <a v-if="current.status === 'published'" :href="`/courses/${current.slug}`" target="_blank" class="btn btn-secondary btn-sm">{{ t('creator.viewPage') }}</a>
        <button v-if="current.approved_at && current.status === 'published'" class="btn btn-secondary btn-sm" :disabled="busy" data-testid="creator-unlist" @click="setOnSale(false)">{{ t('creator.unlist') }}</button>
        <button v-else-if="current.approved_at" class="btn btn-secondary btn-sm" :disabled="busy" data-testid="creator-relist" @click="setOnSale(true)">{{ t('creator.relist') }}</button>
        <button v-if="!current.approved_at && !current.student_count" class="btn btn-danger btn-sm" :disabled="busy" @click="remove">{{ t('common.delete') }}</button>
      </div>
    </div>

    <!-- Where the review stands -->
    <div v-if="current" :class="['rounded-lg border p-3 text-sm', reviewTone]" data-testid="creator-review-state">
      <div class="font-medium">{{ t(`creator.review.${current.review_status || 'approved'}`) }}</div>
      <p class="mt-1 text-xs opacity-80">{{ t(`creator.reviewHint.${current.review_status || 'approved'}`) }}</p>
      <p v-if="current.review_status === 'rejected' && current.review_note" class="mt-1 whitespace-pre-wrap text-xs" data-testid="creator-review-note">{{ t('creator.rejectReason') }}{{ current.review_note }}</p>
      <p v-if="current.approved_at" class="mt-1 text-xs opacity-80">{{ t('creator.liveVersion', { title: current.title, price: current.price }) }}</p>
    </div>

    <!-- Course -->
    <div class="card space-y-4 p-5" data-testid="creator-course-form">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="text-sm">
          <span class="input-label">{{ t('admin.courses.fields.title') }}</span>
          <input v-model="form.title" class="input" maxlength="120" data-testid="creator-title" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.courses.fields.slug') }}</span>
          <input v-model="form.slug" class="input" maxlength="64" placeholder="ai-office-101" :disabled="!!current" data-testid="creator-slug" />
          <span class="input-hint">{{ current ? t('creator.slugFixed') : t('admin.courses.slugHint') }}</span>
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
          <span class="input-label">{{ t('admin.courses.fields.price') }}</span>
          <input v-model.number="form.price" type="number" min="0.01" step="0.01" class="input" data-testid="creator-price" />
          <span v-if="form.price > 0" class="input-hint" data-testid="creator-share-hint">{{ t('creator.shareHint', { percent: commissionPercent, share: share(form.price) }) }}</span>
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.courses.fields.originalPrice') }}</span>
          <input v-model.number="form.original_price" type="number" min="0" step="0.01" class="input" />
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.courses.fields.salePrice') }}</span>
          <input v-model.number="form.sale_price" type="number" min="0" step="0.01" class="input" />
          <span class="input-hint">{{ t('admin.courses.salePriceHint') }}</span>
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.courses.fields.saleEndsAt') }}</span>
          <input v-model="saleEndsLocal" type="datetime-local" class="input" />
        </label>
        <label class="flex items-center gap-2 text-sm md:col-span-2">
          <input v-model="form.edu_discount" type="checkbox" class="h-4 w-4" />
          {{ t('admin.courses.fields.eduDiscount') }}
        </label>
        <label class="text-sm md:col-span-2">
          <span class="input-label">{{ t('admin.courses.fields.trialVideo') }}</span>
          <input v-model="form.trial_video_url" class="input" placeholder="https://www.bilibili.com/video/BV... / https://.../trial.mp4" />
          <span class="input-hint">{{ t('admin.courses.trialVideoHint') }}</span>
        </label>
        <div class="text-sm md:col-span-2">
          <span class="input-label">{{ t('admin.courses.fields.cover') }}</span>
          <div v-if="current" class="flex items-center gap-3">
            <img v-if="coverURL" :src="coverURL" alt="" class="h-16 w-28 rounded object-cover" />
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
            <label v-if="current" class="btn btn-secondary btn-sm cursor-pointer">
              {{ t('admin.courses.insertImage') }}
              <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" class="hidden" @change="(e) => onInsertImage(e, field)" />
            </label>
            <button class="btn btn-secondary btn-sm" @click="preview[field] = !preview[field]">{{ preview[field] ? t('admin.courses.edit') : t('admin.courses.preview') }}</button>
          </div>
        </div>
        <div v-if="preview[field]" class="course-md min-h-24 rounded-lg border border-gray-200 p-3 dark:border-dark-700" v-html="renderMarkdown(form[field])"></div>
        <textarea v-else v-model="form[field]" rows="8" class="input font-mono text-xs" :placeholder="t(`admin.courses.placeholders.${field}`)"></textarea>
      </div>

      <!-- New course: the netdisk link goes with it -->
      <div v-if="!current" class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" data-testid="creator-new-delivery">
        <div>
          <span class="text-sm font-medium">{{ t('admin.courses.newDeliveryTitle') }}</span>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('creator.deliveryHint') }}</p>
        </div>
        <DeliveryFields :model-value="delivery" prefix="creator-new" />
      </div>

      <div class="flex flex-wrap gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
        <button class="btn btn-primary" :disabled="busy" data-testid="creator-save" @click="save">{{ current ? t('creator.saveDraft') : t('creator.createCourse') }}</button>
        <button
          v-if="current"
          class="btn btn-secondary"
          :disabled="busy || !canSubmit"
          :title="submitBlocker"
          data-testid="creator-submit"
          @click="submit"
        >
          {{ t('creator.submit') }}
        </button>
        <span v-if="current && submitBlocker" class="self-center text-xs text-gray-500">{{ submitBlocker }}</span>
      </div>
    </div>

    <!-- Netdisk delivery: live at once, no review -->
    <div v-if="current" class="card space-y-3 p-5" data-testid="creator-delivery">
      <div>
        <h3 class="text-sm font-medium">{{ t('creator.deliveryTitle') }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('creator.deliveryHint') }}</p>
      </div>
      <DeliveryFields :model-value="delivery" prefix="creator-delivery" />
      <label v-if="deliveries.length && current.student_count" class="flex items-center gap-2 text-sm">
        <input v-model="delivery.notify" type="checkbox" class="h-4 w-4" />
        {{ t('admin.courses.notifyStudents', { count: current.student_count }) }}
      </label>
      <button class="btn btn-primary btn-sm" :disabled="busy || !delivery.link" data-testid="creator-delivery-save" @click="saveDelivery">{{ t('admin.courses.saveDelivery') }}</button>
      <ul v-if="deliveries.length" class="space-y-2 text-sm">
        <li v-for="d in deliveries" :key="d.version" class="rounded-lg border border-gray-100 p-3 dark:border-dark-700">
          <div class="flex flex-wrap items-center gap-2">
            <span class="badge badge-gray">v{{ d.version }}</span>
            <span class="text-xs text-gray-400">{{ formatDateTime(d.updated_at) }}</span>
            <span v-if="d.version === deliveries[0].version" class="badge badge-success">{{ t('admin.courses.current') }}</span>
          </div>
          <div class="mt-1 break-all font-mono text-xs">{{ d.link }}<template v-if="d.code"> · {{ t('admin.courses.fields.code') }} {{ d.code }}</template><template v-if="d.password"> · {{ t('admin.courses.fields.password') }} {{ d.password }}</template></div>
        </li>
      </ul>
      <p v-else class="text-sm text-amber-600" data-testid="creator-no-delivery">{{ t('admin.courses.missingDelivery') }}</p>
    </div>

    <!-- Buyers -->
    <div v-if="current && current.student_count" class="card p-5">
      <h3 class="mb-3 text-sm font-medium">{{ t('creator.students', { count: studentTotal }) }}</h3>
      <table class="w-full text-sm">
        <thead class="text-left text-xs text-gray-500">
          <tr><th class="py-1">{{ t('creator.buyer') }}</th><th class="py-1">{{ t('creator.boughtAt') }}</th><th class="py-1">{{ t('creator.viewed') }}</th></tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
          <tr v-for="s in students" :key="s.user_id">
            <td class="py-2">{{ s.user_email || '***' }}<span v-if="s.refunded" class="badge badge-gray ml-2">{{ t('creator.refunded') }}</span></td>
            <td class="py-2 text-xs text-gray-500">{{ formatDateTime(s.created_at) }}</td>
            <td class="py-2 text-xs text-gray-500">{{ s.view_count }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import creatorAPI, { type CreatorCourseInput } from '@/api/creator'
import DeliveryFields from './DeliveryFields.vue'
import type { DeliveryInput } from '@/api/admin/courses'
import type { Course, CourseDelivery, CourseEnrollment } from '@/api/courses'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'

type MarkdownField = 'intro_md' | 'trial_md' | 'faq_md'

const props = defineProps<{ courseId: number | null; commissionPercent: number }>()
const emit = defineEmits<{ close: []; saved: [] }>()

const { t } = useI18n()
const appStore = useAppStore()

const markdownFields: MarkdownField[] = ['intro_md', 'trial_md', 'faq_md']
const busy = ref(false)
const current = ref<Course | null>(null)
const preview = reactive<Record<MarkdownField, boolean>>({ intro_md: false, trial_md: false, faq_md: false })
const emptyForm = (): CreatorCourseInput => ({
  slug: '', title: '', subtitle: '', category: '', price: 0, original_price: 0,
  intro_md: '', outline: [], trial_md: '', faq_md: '', sale_price: 0, sale_ends_at: null, edu_discount: true, trial_video_url: '',
})
const form = reactive<CreatorCourseInput>(emptyForm())
const emptyDelivery = (): DeliveryInput => ({ link: '', code: '', password: '', note: '', notify: true })
const delivery = reactive<DeliveryInput>(emptyDelivery())
const deliveries = ref<CourseDelivery[]>([])
const students = ref<CourseEnrollment[]>([])
const studentTotal = ref(0)

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

const coverURL = computed(() => current.value?.draft?.cover_url || current.value?.cover_url || '')
const reviewTone = computed(() => {
  switch (current.value?.review_status) {
    case 'pending':
      return 'border-blue-200 bg-blue-50 text-blue-800 dark:border-blue-900/50 dark:bg-blue-900/20 dark:text-blue-200'
    case 'rejected':
      return 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/50 dark:bg-rose-900/20 dark:text-rose-200'
    case 'draft':
      return 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200'
    default:
      return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-200'
  }
})
const submitBlocker = computed(() => {
  const c = current.value
  if (!c) return ''
  if (c.review_status === 'pending') return t('creator.blocker.pending')
  if (c.review_status === 'approved' || !c.review_status) return t('creator.blocker.noChanges')
  if (!c.delivery_version) return t('creator.blocker.noDelivery')
  return ''
})
const canSubmit = computed(() => !!current.value && !submitBlocker.value)

function fail(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

function formatDateTime(value: string) {
  return value ? new Date(value).toLocaleString() : ''
}

function share(price: number) {
  return (price * (1 - props.commissionPercent / 100)).toFixed(2)
}

function fillForm(c: Course) {
  const src = c.draft || c
  Object.assign(form, emptyForm(), {
    slug: c.slug, title: src.title, subtitle: src.subtitle, category: src.category, price: src.price, original_price: src.original_price,
    intro_md: src.intro_md || '', trial_md: src.trial_md || '', faq_md: src.faq_md || '',
    outline: JSON.parse(JSON.stringify(src.outline || [])),
    sale_price: src.sale_price || 0, sale_ends_at: src.sale_ends_at || null, edu_discount: src.edu_discount, trial_video_url: src.trial_video_url || '',
  })
}

async function load(id: number) {
  current.value = await creatorAPI.get(id)
  fillForm(current.value)
  deliveries.value = await creatorAPI.deliveries(id)
  if (current.value.student_count) {
    const res = await creatorAPI.students(id)
    students.value = res.items
    studentTotal.value = res.total
  }
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

function save() {
  return run(async () => {
    if (current.value) {
      current.value = await creatorAPI.update(current.value.id, { ...form })
    } else {
      const created = await creatorAPI.create({ ...form })
      if (delivery.link.trim()) {
        await creatorAPI.saveDelivery(created.id, { ...delivery, notify: false })
        Object.assign(delivery, emptyDelivery())
      }
      await load(created.id)
    }
    fillForm(current.value!)
    appStore.showSuccess(t('creator.draftSaved'))
    emit('saved')
  })
}

function submit() {
  return run(async () => {
    if (!current.value) return
    current.value = await creatorAPI.update(current.value.id, { ...form })
    current.value = await creatorAPI.submit(current.value.id)
    appStore.showSuccess(t('creator.submitted'))
    emit('saved')
  })
}

function setOnSale(onSale: boolean) {
  return run(async () => {
    if (!current.value) return
    current.value = await creatorAPI.setOnSale(current.value.id, onSale)
    emit('saved')
  })
}

function remove() {
  if (!current.value || !window.confirm(t('creator.deleteConfirm', { title: current.value.title }))) return
  return run(async () => {
    await creatorAPI.remove(current.value!.id)
    emit('saved')
    emit('close')
  })
}

function saveDelivery() {
  return run(async () => {
    if (!current.value) return
    await creatorAPI.saveDelivery(current.value.id, { ...delivery })
    Object.assign(delivery, emptyDelivery())
    deliveries.value = await creatorAPI.deliveries(current.value.id)
    current.value = await creatorAPI.get(current.value.id)
    appStore.showSuccess(t('admin.courses.deliverySaved'))
  })
}

function pickFile(e: Event): File | null {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0] || null
  input.value = ''
  return file
}

function onCover(e: Event) {
  const file = pickFile(e)
  if (!file || !current.value) return
  return run(async () => {
    current.value = await creatorAPI.uploadCover(current.value!.id, file)
  })
}

function onInsertImage(e: Event, field: MarkdownField) {
  const file = pickFile(e)
  if (!file) return
  return run(async () => {
    const url = await creatorAPI.uploadImage(file)
    form[field] = `${form[field]}${form[field] && !form[field].endsWith('\n') ? '\n' : ''}![](${url})\n`
    preview[field] = false
  })
}

onMounted(() => {
  if (props.courseId) void run(() => load(props.courseId!))
})
</script>
