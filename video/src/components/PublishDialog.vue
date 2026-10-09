<template>
  <Modal title="发布到案例库" @close="$emit('close')">
    <p v-if="project.mode === 'upload'" class="text-sm leading-6 text-ink-600 dark:text-ink-300">发布后经过审核会出现在案例库，所有人都能观看（会看到标题和作品介绍）。可以随时撤回。</p>
    <p v-else class="text-sm leading-6 text-ink-600 dark:text-ink-300">发布后经过审核会出现在案例库，所有人都能观看和「制作同款」（会看到你的描述，看不到你的 Key 和用量）。可以随时撤回。</p>
    <label class="mt-4 block text-sm font-medium">标题</label>
    <input v-model="title" maxlength="60" class="input mt-1" />
    <label class="mt-4 block text-sm font-medium">分类</label>
    <select v-model="category" class="input mt-1">
      <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
    </select>
    <p class="mt-3 text-xs text-ink-400">请勿发布侵权、违法或含个人隐私的内容。</p>
    <div class="mt-6 flex justify-end gap-2">
      <button class="btn-ghost" @click="$emit('close')">取消</button>
      <button class="btn-primary" :disabled="saving || !title.trim()" @click="publish">提交审核</button>
    </div>
  </Modal>
</template>

<script setup>
import { ref } from 'vue'
import Modal from './Modal.vue'
import { api } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const props = defineProps({ project: { type: Object, required: true }, categories: { type: Array, default: () => [] } })
const emit = defineEmits(['close', 'published'])
const title = ref(props.project.title)
const category = ref(props.project.category || (props.project.mode === 'film' ? 'film' : 'other'))
const saving = ref(false)
async function publish() {
  saving.value = true
  try {
    const p = await api(`/projects/${props.project.id}/publish`, { method: 'POST', body: { title: title.value.trim(), category: category.value } })
    toast('已提交，审核通过后会出现在案例库', 'success')
    emit('published', p)
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}
</script>
