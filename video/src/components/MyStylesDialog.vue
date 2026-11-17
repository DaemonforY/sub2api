<template>
  <Modal title="我的风格" wide @close="$emit('close')">
    <div data-testid="my-styles">
      <p class="text-sm leading-6 text-ink-600 dark:text-ink-300">风格卡记下一个系列的外观（配色、字体、圆角、背景和页眉、角色）和你的系列规则。做好一部满意的视频后，在作品页点「存为我的风格」；以后新建视频选它，整个账号的片子保持一致。</p>
      <p v-if="!items.length" class="mt-6 rounded-xl bg-ink-50 p-6 text-center text-sm text-ink-500 dark:bg-ink-950">还没有风格卡。</p>
      <div v-for="st in items" :key="st.id" class="mt-4 rounded-xl border border-ink-200 p-4 dark:border-ink-800">
        <div class="flex items-center gap-2">
          <input v-model="st.name" class="input flex-1 py-1" maxlength="20" />
          <span class="text-xs text-ink-500">{{ st.ratio }}</span>
          <a v-if="st.source_project" :href="`/p/${st.source_project}`" class="text-xs text-brand-600 hover:underline">来源作品</a>
        </div>
        <p v-if="st.character" class="mt-2 text-xs leading-5 text-ink-500">角色：{{ st.character }}</p>
        <label class="mt-3 block text-xs font-medium text-ink-500">系列规则（一行一条，最多 12 条）</label>
        <textarea v-model="st.rulesText" rows="3" class="input mt-1 text-sm leading-6" placeholder="例如：屏幕上的正文不小于 28px"></textarea>
        <div class="mt-2 flex justify-end gap-2">
          <button class="btn-ghost btn-sm text-red-600" @click="remove(st)">删除</button>
          <button class="btn-primary btn-sm" @click="save(st)">保存</button>
        </div>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import Modal from './Modal.vue'
import { api, myStyles } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const emit = defineEmits(['close', 'changed'])
const items = ref([])

async function load() {
  items.value = (await myStyles()).map((st) => ({ ...st, rulesText: (st.rules || []).join('\n') }))
}
async function save(st) {
  try {
    const rules = st.rulesText.split('\n').map((x) => x.trim()).filter(Boolean)
    await api(`/styles/${st.id}`, { method: 'PUT', body: { name: st.name, rules } })
    toast('已保存', 'success')
    emit('changed')
    await load()
  } catch (err) {
    toastError(err)
  }
}
async function remove(st) {
  if (!confirm(`删除风格卡「${st.name}」？用它做过的作品不受影响。`)) return
  try {
    await api(`/styles/${st.id}`, { method: 'DELETE' })
    emit('changed')
    await load()
  } catch (err) {
    toastError(err)
  }
}
onMounted(load)
</script>
