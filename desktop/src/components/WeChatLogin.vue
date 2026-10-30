<script setup lang="ts">
// Shows a QR of <site>/desktop-login?t=…; the phone signs in on the site and confirms, and we poll
// (src-tauri/src/login.rs) until the device key arrives. The QR refreshes itself when it expires.
import { onMounted, onUnmounted, ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import QRCode from 'qrcode'

const emit = defineEmits<{ done: [] }>()

interface LoginStarted {
  login_url: string
  expires_in: number
  poll_interval: number
}

const qr = ref('')
const status = ref<'loading' | 'pending' | 'approved' | 'rejected' | 'error'>('loading')
const error = ref('')
let timer: ReturnType<typeof setTimeout> | null = null
let expiresAt = 0
let interval = 2000
let stopped = false

function clear() {
  if (timer) clearTimeout(timer)
  timer = null
}

async function start() {
  clear()
  status.value = 'loading'
  error.value = ''
  try {
    const res = await invoke<LoginStarted>('login_start')
    qr.value = await QRCode.toDataURL(res.login_url, { width: 220, margin: 1 })
    expiresAt = Date.now() + res.expires_in * 1000
    interval = res.poll_interval * 1000
    status.value = 'pending'
    schedule()
  } catch (e) {
    status.value = 'error'
    error.value = String(e)
  }
}

function schedule() {
  if (stopped) return
  timer = setTimeout(poll, interval)
}

async function poll() {
  if (stopped) return
  // Refresh a little before the server drops the ticket.
  if (Date.now() > expiresAt - 5000) return start()
  try {
    const res = await invoke<{ status: string }>('login_poll')
    switch (res.status) {
      case 'done':
        stopped = true
        emit('done')
        return
      case 'approved':
        status.value = 'approved'
        break
      case 'rejected':
        status.value = 'rejected'
        return
      case 'expired':
        return start()
    }
  } catch (e) {
    // A network blip shouldn't kill the QR; show it and keep polling.
    error.value = String(e)
  }
  schedule()
}

onMounted(start)
onUnmounted(() => {
  stopped = true
  clear()
})
</script>

<template>
  <div class="login">
    <div class="qr">
      <img v-if="qr && status !== 'loading'" :src="qr" alt="登录二维码" :class="{ dim: status === 'rejected' }" />
      <span v-else class="placeholder">{{ status === 'error' ? '二维码获取失败' : '正在获取二维码…' }}</span>
    </div>
    <div class="text">
      <template v-if="status === 'approved'">
        <p class="big">已确认，正在登录…</p>
      </template>
      <template v-else-if="status === 'rejected'">
        <p class="big">这次登录已被拒绝</p>
        <button class="primary" @click="start">重新获取二维码</button>
      </template>
      <template v-else-if="status === 'error'">
        <p class="error">{{ error }}</p>
        <button class="primary" @click="start">重试</button>
      </template>
      <template v-else>
        <p class="big">用微信扫一扫</p>
        <p class="hint">扫码后在手机上登录 HiveGPT（新用户会自动注册），再点「确认登录」。</p>
        <p v-if="error" class="error small">{{ error }}</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.login {
  display: flex;
  gap: 20px;
  align-items: center;
}

.qr {
  width: 180px;
  height: 180px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fff;
}

.qr img {
  width: 170px;
  height: 170px;
}

.qr img.dim {
  opacity: 0.25;
}

.placeholder {
  color: var(--muted);
  font-size: 13px;
}

.big {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 6px;
}

.hint {
  color: var(--muted);
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
}

.error {
  color: var(--danger);
  white-space: pre-wrap;
}

.small {
  font-size: 12px;
}
</style>
