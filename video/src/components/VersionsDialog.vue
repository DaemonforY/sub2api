<template>
  <Modal title="版本历史" @close="$emit('close')">
    <p class="text-sm text-ink-500">每次修改前都会自动保存一个版本，可以回退到任意版本（当前版本也会先保存）。</p>
    <div class="thin-scroll mt-4 max-h-80 space-y-1 overflow-y-auto">
      <p v-if="loaded && !items.length" class="py-6 text-center text-sm text-ink-400">还没有历史版本</p>
      <div v-for="v in items" :key="v.id" class="flex items-center gap-3 rounded-xl px-3 py-2 hover:bg-ink-50 dark:hover:bg-ink-800">
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm">{{ v.note }}</span>
          <span class="text-xs text-ink-400">{{ new Date(v.created_at).toLocaleString() }}</span>
        </span>
        <button class="btn-ghost btn-sm" :disabled="busy || restoring" @click="restore(v)">回退到此版本</button>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import Modal from './Modal.vue'
import { api } from '../lib/api'
import { toastError } from '../lib/toast'

const props = defineProps({ projectId: { type: String, required: true }, busy: Boolean })
const emit = defineEmits(['close', 'restored'])
const items = ref([])
const loaded = ref(false)
const restoring = ref(false)
onMounted(async () => {
  try {
    items.value = (await api(`/projects/${props.projectId}/versions`)).items
  } catch (err) {
    toastError(err)
  } finally {
    loaded.value = true
  }
})
async function restore(v) {
  restoring.value = true
  try {
    emit('restored', await api(`/projects/${props.projectId}/versions/${v.id}/restore`, { method: 'POST', body: {} }))
  } catch (err) {
    toastError(err)
  } finally {
    restoring.value = false
  }
}
</script>
