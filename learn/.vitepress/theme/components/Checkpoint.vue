<script setup lang="ts">
// A checkpoint the server verifies (has a key, called the API with it, published a work, ...).
import { computed, onMounted, ref } from 'vue'
import { ApiError, checkpointInfo, loginUrl, progress, token, verifyCheckpoint } from '../api'

const props = defineProps<{ id: string }>()
const title = ref('')
const hint = ref('')
const checking = ref(false)
const message = ref('')
const signedIn = ref(false)
const passed = computed(() => !!progress.checkpoints[props.id])

onMounted(async () => {
  signedIn.value = !!token()
  try {
    const info = await checkpointInfo(props.id)
    title.value = info.title
    hint.value = info.hint
  } catch {
    title.value = '检查点'
  }
})

async function check() {
  checking.value = true
  message.value = ''
  try {
    const res = await verifyCheckpoint(props.id)
    if (!res.passed) message.value = `还没有检测到：${res.hint || ''}`
  } catch (e) {
    message.value = (e as ApiError).message
  } finally {
    checking.value = false
  }
}
</script>

<template>
  <div :class="['checkpoint', { done: passed }]" data-testid="checkpoint">
    <span class="checkpoint-icon">{{ passed ? '✓' : '◎' }}</span>
    <div class="checkpoint-body">
      <strong>检查点：{{ title }}</strong>
      <p v-if="!passed && !message" class="runbox-note">{{ hint }}</p>
      <p v-if="message" class="checkpoint-msg">{{ message }}</p>
    </div>
    <span v-if="passed" class="lesson-done">已完成</span>
    <a v-else-if="!signedIn" class="runbox-btn ghost" :href="loginUrl()">登录后核对</a>
    <button v-else class="runbox-btn ghost" :disabled="checking" @click="check">{{ checking ? '核对中…' : '我做完了，核对' }}</button>
  </div>
</template>
