<template>
  <div class="space-y-4" data-testid="desk-angles">
    <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.angles.title') }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.angles.hint') }}</p>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="busy" @click="more">{{ desk.angles.length ? t('admin.desk.angles.again') : t('admin.desk.angles.make') }}</button>
    </div>

    <div class="grid gap-3 lg:grid-cols-2">
      <button
        v-for="a in desk.angles"
        :key="a.id"
        type="button"
        :class="['card space-y-2 p-4 text-left transition', picked?.id === a.id ? 'ring-2 ring-primary-500' : 'hover:ring-1 hover:ring-primary-200']"
        :data-testid="`desk-angle-${a.id}`"
        @click="pick(a)"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="font-semibold text-gray-900 dark:text-white">{{ a.title }}</div>
          <span class="flex-shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ a.archetype }}</span>
        </div>
        <p class="text-sm text-gray-700 dark:text-dark-200">{{ a.angle }}</p>
        <p class="rounded-lg bg-gray-50 p-2 text-sm italic text-gray-600 dark:bg-dark-800 dark:text-dark-300">「{{ a.hook }}」</p>
        <div class="grid grid-cols-3 gap-2 text-xs">
          <div v-for="(v, k) in { H: a.h, K: a.k, R: a.r }" :key="k">
            <div class="flex justify-between text-gray-500"><span>{{ t(`admin.desk.angles.hkr.${k}`) }}</span><span>{{ v }}</span></div>
            <div class="mt-0.5 h-1.5 rounded-full bg-gray-100 dark:bg-dark-700">
              <div class="h-1.5 rounded-full bg-primary-500" :style="{ width: `${v * 20}%` }" />
            </div>
          </div>
        </div>
        <p class="text-xs text-gray-500">{{ a.why }}</p>
        <p v-if="a.reader" class="text-xs text-gray-400">{{ t('admin.desk.angles.reader', { reader: a.reader }) }}</p>
      </button>
    </div>
    <p v-if="!desk.angles.length && !busy" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.desk.angles.empty') }}</p>

    <!-- The pick, editable -->
    <div v-if="picked" class="card space-y-3 p-4" data-testid="desk-angle-form">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.angles.edit') }}</h4>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.angles.fTitle') }}</span>
        <input v-model="picked.title" class="input" maxlength="60" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.angles.fAngle') }}</span>
        <input v-model="picked.angle" class="input" maxlength="300" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.angles.fHook') }}</span>
        <textarea v-model="picked.hook" class="input min-h-[72px] text-sm" maxlength="600" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.desk.angles.fNote') }}</span>
        <input v-model="picked.note" class="input" maxlength="500" :placeholder="t('admin.desk.angles.fNotePlaceholder')" />
      </label>
    </div>

    <!-- Reference article -->
    <div class="card space-y-3 p-4">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.desk.ref.title') }}</h4>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.desk.ref.hint') }}</p>
      </div>
      <div v-if="desk.skeleton" class="space-y-1 rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
        <div class="flex items-center justify-between">
          <span class="font-medium">{{ t('admin.desk.ref.from', { source: desk.skeleton.source }) }}</span>
          <button class="text-xs text-red-600 hover:underline" @click="clearRef">{{ t('admin.desk.common.delete') }}</button>
        </div>
        <p><span class="text-gray-500">{{ t('admin.desk.ref.opening') }}</span>{{ desk.skeleton.opening }}</p>
        <ol class="list-decimal space-y-0.5 pl-5">
          <li v-for="(p, i) in desk.skeleton.parts" :key="i"><span class="font-medium">{{ p.role }}</span>：{{ p.how }}</li>
        </ol>
        <p><span class="text-gray-500">{{ t('admin.desk.ref.emotion') }}</span>{{ desk.skeleton.emotion }}</p>
        <p><span class="text-gray-500">{{ t('admin.desk.ref.escalation') }}</span>{{ desk.skeleton.escalation }}</p>
        <p v-if="desk.skeleton.callbacks"><span class="text-gray-500">{{ t('admin.desk.ref.callbacks') }}</span>{{ desk.skeleton.callbacks }}</p>
      </div>
      <template v-else>
        <div class="flex flex-wrap gap-2">
          <input v-model.trim="refUrl" class="input min-w-0 flex-1" :placeholder="t('admin.desk.ref.urlPlaceholder')" />
          <button class="btn btn-secondary" :disabled="busy || !refUrl" @click="addRef({ url: refUrl })">{{ t('admin.desk.ref.analyze') }}</button>
        </div>
        <details>
          <summary class="cursor-pointer text-xs text-gray-500">{{ t('admin.desk.ref.paste') }}</summary>
          <textarea v-model="refText" class="input mt-2 min-h-[120px] text-sm" />
          <button class="btn btn-secondary btn-sm mt-2" :disabled="busy || refText.trim().length < 300" @click="addRef({ text: refText })">{{ t('admin.desk.ref.analyze') }}</button>
        </details>
      </template>
    </div>

    <div class="card flex flex-wrap items-center justify-end gap-2 p-4">
      <button class="btn btn-secondary" @click="emit('stage', 'materials')">{{ t('admin.desk.common.back') }}</button>
      <button class="btn btn-primary" :disabled="busy || !picked" data-testid="desk-write-drafts" @click="write">{{ t('admin.desk.angles.write') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { Angle, ContentItem, DeskStage } from '@/api/admin/content'

const props = defineProps<{ item: ContentItem; busy: boolean }>()
const emit = defineEmits<{ (e: 'updated', it: ContentItem): void; (e: 'stage', s: DeskStage): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const desk = computed(() => props.item.desk!)
const picked = ref<Angle | null>(null)
const refUrl = ref('')
const refText = ref('')

watch(
  () => props.item.id,
  () => {
    picked.value = desk.value.angle ? { ...desk.value.angle } : null
  },
  { immediate: true }
)

function pick(a: Angle) {
  picked.value = { ...a, note: picked.value?.note || '' }
}

async function call(fn: () => Promise<ContentItem>) {
  try {
    emit('updated', await fn())
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

const more = () => call(() => contentAPI.makeAngles(props.item.id))
const clearRef = () => call(() => contentAPI.clearReference(props.item.id))
async function addRef(input: { url?: string; text?: string }) {
  await call(() => contentAPI.addReference(props.item.id, input))
  refUrl.value = refText.value = ''
}

async function write() {
  if (!picked.value) return
  const angle = picked.value
  await call(async () => {
    await contentAPI.chooseAngle(props.item.id, angle)
    return contentAPI.makeDrafts(props.item.id)
  })
}
</script>
