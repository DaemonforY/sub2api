<template>
  <Modal title="存为我的风格" @close="$emit('close')">
    <div data-testid="save-style">
      <p class="text-sm leading-6 text-ink-600 dark:text-ink-300">
        把这部片子的<b>配色、字体、圆角、背景和页眉、角色造型</b>存成风格卡。以后新建视频选它，整个系列看起来是同一个账号做的；画幅也跟这部一样（{{ project.width > project.height ? '横屏' : '竖屏' }}）。
      </p>
      <label class="mt-4 block text-xs font-medium text-ink-500">风格卡名字</label>
      <input v-model="name" class="input mt-1" maxlength="20" placeholder="例如：科技频道主风格" />
      <label class="mt-4 block text-xs font-medium text-ink-500">系列规则（可选，一行一条，以后同风格的片子都会遵守）</label>
      <textarea v-model="rulesText" rows="3" class="input mt-1 leading-6" placeholder="例如：屏幕上的正文不小于 28px，一屏最多 3 行"></textarea>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn-ghost" @click="$emit('close')">取消</button>
        <button class="btn-primary" :disabled="saving || !name.trim()" @click="save"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" />保存风格卡</button>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { ref } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import Modal from './Modal.vue'
import { api } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const props = defineProps({ project: { type: Object, required: true } })
const emit = defineEmits(['close', 'saved'])
const name = ref((props.project.title || '').slice(0, 20))
const rulesText = ref('')
const saving = ref(false)

async function save() {
  saving.value = true
  try {
    const rules = rulesText.value.split('\n').map((x) => x.trim()).filter(Boolean)
    const card = await api('/styles', { method: 'POST', body: { project_id: props.project.id, name: name.value.trim(), rules } })
    toast(`已存为风格卡「${card.name}」，新建视频时在「我的风格」里选它`, 'success')
    emit('saved', card)
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}
</script>
