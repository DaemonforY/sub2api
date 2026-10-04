<script setup lang="ts">
// A track's certificate: what is still missing, claiming it, and the share poster.
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { ApiError, certStatus, claimCert, currentUser, loginUrl, refreshMe, setShowcase, token, type CertStatus } from '../api'
import { findTrack } from '../tracks'
import CertPoster from './CertPoster.vue'

const props = defineProps<{ track: string }>()
const status = ref<CertStatus | null>(null)
const signedIn = ref(false)
const loading = ref(false)
const name = ref('')
const project = ref('')
const claiming = ref(false)
const showOnWall = ref(true)
const savingWall = ref(false)
const error = ref('')
const track = computed(() => findTrack(props.track))

async function load() {
  loading.value = true
  error.value = ''
  try {
    status.value = await certStatus(props.track)
    if (!project.value && status.value.projects.length) project.value = status.value.projects[0].url
  } catch (e) {
    error.value = (e as ApiError).message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  signedIn.value = !!token()
  name.value = currentUser()?.username || ''
  if (signedIn.value) void load()
})

async function toggleWall() {
  const cert = status.value?.certificate
  if (!cert) return
  savingWall.value = true
  error.value = ''
  try {
    const res = await setShowcase(props.track, !cert.showcase)
    cert.showcase = res.showcase
  } catch (e) {
    error.value = (e as ApiError).message
  } finally {
    savingWall.value = false
  }
}

async function claim() {
  claiming.value = true
  error.value = ''
  try {
    await claimCert(props.track, name.value, project.value, showOnWall.value)
    await Promise.all([load(), refreshMe()])
  } catch (e) {
    error.value = (e as ApiError).message
  } finally {
    claiming.value = false
  }
}
</script>

<template>
  <div class="cert-panel" data-testid="cert-panel">
    <div class="cert-panel-head">
      <span class="cert-badge">🎓</span>
      <div>
        <strong>结业证书</strong>
        <p class="runbox-note">完成下面的要求就能领取，证书有公开的验证页，可以生成分享海报。</p>
      </div>
    </div>

    <template v-if="!signedIn">
      <ul class="cert-items">
        <li>学完这条路线的全部课程</li>
        <li v-if="track && track.id !== 'd'">各课测验总正确率不低于 80%</li>
        <li>{{ track?.project }}</li>
      </ul>
      <a class="runbox-btn" :href="loginUrl()">登录后查看我的进度</a>
    </template>

    <template v-else-if="status">
      <template v-if="status.certificate">
        <p class="cert-got">
          ✓ 已领取 · 证书编号 {{ status.certificate.code }} ·
          <a :href="withBase(`/cert.html?c=${status.certificate.code}`)">查看证书页</a>
        </p>
        <p class="runbox-note cert-wall">
          <template v-if="status.certificate.showcase">
            ✓ 已展示在 <a :href="withBase('/showcase')">学员作品墙</a>
            <button class="runbox-btn ghost small" :disabled="savingWall" @click="toggleWall">不展示了</button>
          </template>
          <template v-else>
            没有展示在学员作品墙
            <button class="runbox-btn ghost small" :disabled="savingWall" data-testid="cert-wall-on" @click="toggleWall">展示到作品墙</button>
          </template>
        </p>
        <CertPoster :cert="status.certificate" />
      </template>
      <template v-else>
        <ul class="cert-items">
          <li v-for="(it, i) in status.items" :key="i" :class="{ done: it.done }">
            <span class="cert-check">{{ it.done ? '✓' : '○' }}</span>
            <span>{{ it.title }}<small v-if="it.detail"> — {{ it.detail }}</small></span>
          </li>
        </ul>
        <div v-if="status.eligible" class="cert-claim">
          <label>证书上显示的名字<input v-model="name" maxlength="20" placeholder="昵称或真名，最多 20 个字" /></label>
          <label v-if="status.projects.length > 1 || (status.projects.length && status.project_link)">
            结业项目
            <select v-model="project">
              <option v-for="p in status.projects" :key="p.url" :value="p.url">{{ p.title }}（{{ p.url }}）</option>
              <option v-if="status.project_link" value="">填写其他链接…</option>
            </select>
          </label>
          <label v-if="status.project_link && !status.projects.some((p) => p.url === project)">
            结业项目链接<input v-model="project" placeholder="https://github.com/你的名字/项目" />
          </label>
          <label class="cert-check-line">
            <input v-model="showOnWall" type="checkbox" data-testid="cert-wall" />
            把我的名字和结业项目展示在<a :href="withBase('/showcase')" target="_blank">学员作品墙</a>（随时可以关闭）
          </label>
          <div class="runbox-actions">
            <button class="runbox-btn" :disabled="claiming || !name.trim()" data-testid="cert-claim" @click="claim">{{ claiming ? '领取中…' : '领取结业证书' }}</button>
            <span class="runbox-note">名字和项目链接会显示在公开的证书页上</span>
          </div>
        </div>
        <button v-else class="runbox-btn ghost" :disabled="loading" @click="load">{{ loading ? '核对中…' : '刷新进度' }}</button>
      </template>
    </template>
    <p v-else-if="loading" class="runbox-note">正在核对你的进度…</p>
    <div v-if="error" class="runbox-error">{{ error }}</div>
  </div>
</template>
