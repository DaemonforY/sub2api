<script setup lang="ts">
// The public certificate page (/learn/cert.html?c=<code>): anyone can verify a certificate.
import { onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { ApiError, publicCert, type Certificate } from '../api'
import { MAIN_SITE } from '../tracks'
import CertPoster from './CertPoster.vue'

const cert = ref<Certificate | null>(null)
const error = ref('')
const loading = ref(true)

onMounted(async () => {
  const code = new URLSearchParams(window.location.search).get('c') || ''
  if (!code) {
    error.value = '链接里缺少证书编号'
    loading.value = false
    return
  }
  try {
    cert.value = await publicCert(code)
  } catch (e) {
    error.value = (e as ApiError).status === 404 ? '没有找到这张证书，可能编号有误或已被撤销' : (e as ApiError).message
  } finally {
    loading.value = false
  }
})

function date(s: string) {
  return new Date(s).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
}

function joinUrl(c: Certificate) {
  return c.invite_code ? `${MAIN_SITE}/register?aff=${encodeURIComponent(c.invite_code)}` : `${MAIN_SITE}/register`
}
</script>

<template>
  <div class="cert-view" data-testid="cert-view">
    <p v-if="loading" class="runbox-note">正在查询证书…</p>
    <div v-else-if="error" class="runbox-error">{{ error }}</div>
    <template v-else-if="cert">
      <div class="cert-card">
        <div class="cert-card-brand">HiveGPT AI 学习</div>
        <div class="cert-card-title">结业证书</div>
        <p class="cert-card-small">兹证明</p>
        <p class="cert-card-name">{{ cert.display_name }}</p>
        <p class="cert-card-small">完成了学习路线</p>
        <p class="cert-card-track">「{{ cert.track_title }}」</p>
        <p class="cert-card-meta">
          <span v-if="cert.quiz_score">测验正确率 {{ cert.quiz_score }}% · </span>{{ date(cert.issued_at) }} · 证书编号 {{ cert.code }}
        </p>
        <p v-if="cert.project_url" class="cert-card-meta">
          结业项目：<a :href="cert.project_url" target="_blank" rel="noopener nofollow ugc">{{ cert.project_url }}</a>
        </p>
        <p class="cert-card-verified">✓ 已验证：这张证书由 HiveGPT AI 学习签发</p>
      </div>
      <div class="cert-cta">
        <a class="runbox-btn" :href="joinUrl(cert)">我也来学（注册 HiveGPT）</a>
        <a class="runbox-btn ghost" :href="withBase(`/${cert.track}/`)">看看这条路线</a>
      </div>
      <CertPoster :cert="cert" />
    </template>
  </div>
</template>
