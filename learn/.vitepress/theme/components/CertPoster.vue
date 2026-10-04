<script setup lang="ts">
// Generates the share poster for a certificate (drawn in the browser) and offers it for saving.
import { ref } from 'vue'
import type { Certificate } from '../api'
import { certPageUrl, drawPoster } from '../poster'

const props = defineProps<{ cert: Certificate }>()
const image = ref('')
const busy = ref(false)
const copied = ref(false)

async function make() {
  busy.value = true
  try {
    image.value = await drawPoster(props.cert)
  } finally {
    busy.value = false
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(certPageUrl(props.cert.code))
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    // clipboard unavailable
  }
}
</script>

<template>
  <div class="cert-poster">
    <div class="runbox-actions">
      <button class="runbox-btn" :disabled="busy" data-testid="cert-poster" @click="make">{{ busy ? '生成中…' : '生成分享海报' }}</button>
      <button class="runbox-btn ghost" @click="copyLink">{{ copied ? '已复制' : '复制证书链接' }}</button>
    </div>
    <template v-if="image">
      <img :src="image" alt="结业证书分享海报" class="cert-poster-img" />
      <p class="runbox-note">
        手机上长按图片保存，电脑上 <a :href="image" :download="`HiveGPT-证书-${cert.code}.png`">下载海报</a>。
        扫码打开的证书页上，「我也来学」的注册链接带着你的邀请码。
      </p>
    </template>
  </div>
</template>
