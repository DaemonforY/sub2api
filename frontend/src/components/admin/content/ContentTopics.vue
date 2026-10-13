<template>
  <div class="space-y-4" data-testid="content-topics">
    <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
      <div class="flex flex-wrap gap-2">
        <button
          v-for="f in FILTERS"
          :key="f"
          type="button"
          :class="['btn btn-sm', filter === f ? 'btn-primary' : 'btn-secondary']"
          @click="filter = f"
        >{{ t(`admin.content.topics.filter.${f}`) }}</button>
      </div>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-secondary btn-sm" @click="openForm()">{{ t('admin.content.topics.add') }}</button>
        <button class="btn btn-primary btn-sm" :disabled="suggesting" data-testid="content-suggest" @click="suggest">
          {{ suggesting ? t('admin.content.topics.suggesting') : t('admin.content.topics.suggest') }}
        </button>
      </div>
    </div>

    <div v-if="formOpen" class="card space-y-3 p-4">
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.content.topics.titleLabel') }}</span>
        <input v-model.trim="form.title" class="input" maxlength="200" :placeholder="t('admin.content.topics.titlePlaceholder')" data-testid="content-topic-title" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.content.topics.angle') }}</span>
        <input v-model.trim="form.angle" class="input" maxlength="500" :placeholder="t('admin.content.topics.anglePlaceholder')" />
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.content.topics.notes') }}</span>
        <textarea v-model="form.notes" class="input min-h-[120px] text-sm" maxlength="6000" :placeholder="t('admin.content.topics.notesPlaceholder')" />
      </label>
      <label class="block w-40 text-sm">
        <span class="input-label">{{ t('admin.content.topics.score') }}</span>
        <input v-model.number="form.score" type="number" min="0" max="100" class="input" />
      </label>
      <div class="flex gap-2">
        <button class="btn btn-primary btn-sm" :disabled="saving || !form.title" @click="saveForm">{{ t('common.save') }}</button>
        <button class="btn btn-secondary btn-sm" @click="formOpen = false">{{ t('admin.content.common.cancel') }}</button>
      </div>
    </div>

    <p v-if="!topics.length && !loading" class="card p-8 text-center text-sm text-gray-400">{{ t('admin.content.topics.empty') }}</p>
    <ul class="space-y-2">
      <li v-for="tp in topics" :key="tp.id" class="card p-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t(`admin.content.topics.source.${tp.source}`) }}</span>
              <span class="text-xs font-semibold text-primary-600">{{ tp.score }}</span>
              <span class="font-medium text-gray-900 dark:text-white">{{ tp.title }}</span>
            </div>
            <p v-if="tp.angle" class="mt-1 text-sm text-gray-600 dark:text-dark-300">{{ tp.angle }}</p>
            <p v-if="tp.reason" class="mt-0.5 text-xs text-gray-400">{{ tp.reason }}</p>
            <p v-if="tp.notes" class="mt-1 line-clamp-2 text-xs text-gray-500">📎 {{ tp.notes }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <template v-if="tp.status === 'pending'">
              <button class="btn btn-primary btn-sm" :disabled="busyId === tp.id" @click="start(tp)">{{ t('admin.content.topics.start') }}</button>
              <button class="btn btn-secondary btn-sm" @click="openForm(tp)">{{ t('admin.content.topics.edit') }}</button>
              <button class="btn btn-secondary btn-sm" @click="setStatus(tp, 'dismissed')">{{ t('admin.content.topics.dismiss') }}</button>
            </template>
            <button v-if="tp.status === 'dismissed'" class="btn btn-secondary btn-sm" @click="setStatus(tp, 'pending')">{{ t('admin.content.topics.restore') }}</button>
            <button v-if="tp.status === 'used' && tp.item_id" class="btn btn-secondary btn-sm" @click="emit('open-item', tp.item_id)">{{ t('admin.content.topics.openItem') }}</button>
            <button v-if="tp.status !== 'used'" class="btn btn-secondary btn-sm text-red-600" @click="remove(tp)">{{ t('admin.content.topics.delete') }}</button>
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentTopic, TopicStatus } from '@/api/admin/content'

const emit = defineEmits<{ (e: 'open-item', id: number): void; (e: 'changed'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const FILTERS = ['pending', 'used', 'dismissed', 'all'] as const
const filter = ref<(typeof FILTERS)[number]>('pending')
const topics = ref<ContentTopic[]>([])
const loading = ref(false)
const suggesting = ref(false)
const saving = ref(false)
const busyId = ref(0)
const formOpen = ref(false)
const editingId = ref(0)
const form = reactive({ title: '', angle: '', notes: '', score: 70 })

async function load() {
  loading.value = true
  try {
    topics.value = await contentAPI.listTopics(filter.value as TopicStatus | 'all')
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    loading.value = false
  }
}
watch(filter, load)
onMounted(load)

function openForm(tp?: ContentTopic) {
  editingId.value = tp?.id || 0
  Object.assign(form, { title: tp?.title || '', angle: tp?.angle || '', notes: tp?.notes || '', score: tp?.score ?? 70 })
  formOpen.value = true
}

async function saveForm() {
  saving.value = true
  try {
    const input = { title: form.title, angle: form.angle, notes: form.notes, score: form.score }
    if (editingId.value) await contentAPI.updateTopic(editingId.value, input)
    else await contentAPI.createTopic(input)
    formOpen.value = false
    emit('changed')
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function suggest() {
  suggesting.value = true
  try {
    const added = await contentAPI.suggestTopics()
    appStore.showSuccess(t('admin.content.topics.suggested', { n: added.length }))
    filter.value = 'pending'
    emit('changed')
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    suggesting.value = false
  }
}

async function start(tp: ContentTopic) {
  busyId.value = tp.id
  try {
    const it = await contentAPI.startTopic(tp.id)
    appStore.showSuccess(t('admin.content.topics.started'))
    emit('changed')
    emit('open-item', it.id)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busyId.value = 0
  }
}

async function setStatus(tp: ContentTopic, status: 'pending' | 'dismissed') {
  try {
    await contentAPI.setTopicStatus(tp.id, status)
    emit('changed')
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function remove(tp: ContentTopic) {
  try {
    await contentAPI.deleteTopic(tp.id)
    emit('changed')
    await load()
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
</script>
