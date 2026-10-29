<script setup lang="ts">
// AI 点评 (学画画): the learner submits their own picture (a photo they took, something they drew)
// or a written appreciation, and an AI teacher scores it against the exercise's rubric. Pictures
// are shrunk in the browser before upload and are not stored.
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { track } from '../analytics'
import {
  ApiError,
  QUOTA_REASONS,
  chooseOwnKey,
  critiqueExercise,
  exerciseInfo,
  learnConfig,
  loginUrl,
  ownKey,
  progress,
  runMode,
  token,
  type Critique,
  type ExerciseView,
} from '../api'
import CritiqueCard from './CritiqueCard.vue'
import OwnKeyPicker from './OwnKeyPicker.vue'

const props = withDefaults(defineProps<{ exercise: string; placeholder?: string; title?: string; sample?: string; sampleCritique?: string }>(), {
  title: 'AI 点评',
})

const MAX_EDGE = 1280
const info = ref<ExerciseView | null>(null)
const enabled = ref<boolean | null>(null)
const signedIn = ref(false)
const image = ref('')
const text = ref('')
const running = ref(false)
const error = ref('')
const needLogin = ref(false)
const quotaOut = ref(false)
const critique = ref<Critique | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const isText = computed(() => info.value?.kind === 'text')
const ready = computed(() => (isText.value ? text.value.trim().length >= 10 : !!image.value))
const best = computed(() => progress.practice[props.exercise] || 0)
const sampleCrit = computed<Critique | null>(() => {
  try {
    return props.sampleCritique ? JSON.parse(props.sampleCritique) : null
  } catch {
    return null
  }
})

onMounted(async () => {
  signedIn.value = !!token()
  const [cfg, ex] = await Promise.all([learnConfig(), exerciseInfo(props.exercise)])
  enabled.value = cfg.run_enabled
  info.value = ex
})

// Shrinks the picture to at most 1280 px on the long edge, as a JPEG data URL.
function pick(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  error.value = ''
  if (!file.type.startsWith('image/')) {
    error.value = '请选择一张图片（PNG、JPG 或 WebP）'
    return
  }
  const url = URL.createObjectURL(file)
  const im = new Image()
  im.onload = () => {
    const scale = Math.min(1, MAX_EDGE / Math.max(im.width, im.height))
    const c = document.createElement('canvas')
    c.width = Math.round(im.width * scale)
    c.height = Math.round(im.height * scale)
    c.getContext('2d')?.drawImage(im, 0, 0, c.width, c.height)
    image.value = c.toDataURL('image/jpeg', 0.86)
    critique.value = null
    URL.revokeObjectURL(url)
  }
  im.onerror = () => {
    error.value = '这张图片读不出来，换一张试试'
    URL.revokeObjectURL(url)
  }
  im.src = url
}

async function submit() {
  if (!ready.value || running.value) return
  running.value = true
  error.value = ''
  needLogin.value = false
  quotaOut.value = false
  try {
    critique.value = await critiqueExercise(props.exercise, isText.value ? { text: text.value.trim() } : { image: image.value, text: text.value.trim() || undefined })
    track('learn_critique', { exercise: props.exercise, score: critique.value.score })
  } catch (e) {
    const err = e as ApiError
    if (QUOTA_REASONS.includes(err.reason)) {
      progress.critiquesLeft = 0
      if (ownKey.id && err.reason !== 'LEARN_OWN_KEY_REQUIRED') {
        running.value = false
        return submit()
      }
      quotaOut.value = true
    } else if (err.reason === 'LEARN_KEY_INVALID') {
      chooseOwnKey(null)
      error.value = err.message
    } else if (err.status === 401) {
      needLogin.value = true
      error.value = '登录已过期，请重新登录'
    } else {
      error.value = err.message || '点评失败，请稍后再试'
    }
  } finally {
    running.value = false
  }
}
</script>

<template>
  <div class="drawbox critbox" data-testid="critique-box">
    <div class="runbox-head">
      <span class="runbox-title">🧑‍🏫 {{ title }}{{ info ? `：${info.title}` : '' }}</span>
      <span v-if="best" class="drawbox-best">最好成绩 {{ best }} 分</span>
      <span v-if="signedIn && enabled && progress.critiquesLeft !== null && !runMode.ownKeyOnly" class="runbox-quota">今日免费点评剩余 {{ progress.critiquesLeft }} 次</span>
    </div>
    <div v-if="info" class="drawbox-task">
      <p><b>题目：</b>{{ info.task }}</p>
      <details>
        <summary>AI 老师看这几点（满分 100，{{ info.pass }} 分及格）</summary>
        <ul>
          <li v-for="r in info.rubric" :key="r.dim"><b>{{ r.dim }}</b>（{{ r.weight }}%）：{{ r.look }}</li>
        </ul>
      </details>
    </div>

    <template v-if="info && !isText">
      <div class="critbox-pick">
        <button class="runbox-btn ghost" @click="fileInput?.click()">{{ image ? '换一张' : '选择图片' }}</button>
        <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp" hidden @change="pick" />
        <span class="runbox-note">图片只用来点评，不会保存</span>
      </div>
      <img v-if="image" class="critbox-preview" :src="image" alt="要点评的图片" />
      <input v-model="text" class="critbox-note" maxlength="200" :placeholder="placeholder || '（可选）说说你拍的是什么、想表达什么'" />
    </template>
    <textarea v-else-if="info" v-model="text" class="runbox-input" rows="6" maxlength="1500" :placeholder="placeholder || '在这里写'"></textarea>

    <div class="runbox-actions">
      <template v-if="enabled === false">
        <span class="runbox-note">AI 点评暂未开放。</span>
      </template>
      <template v-else-if="!signedIn">
        <a class="runbox-btn" :href="loginUrl()">登录后请 AI 点评</a>
        <span class="runbox-note">登录 HiveGPT 后每天有免费点评次数</span>
      </template>
      <template v-else>
        <button class="runbox-btn" :disabled="running || !ready" data-testid="critique-run" @click="submit">{{ running ? '点评中…' : '请 AI 老师点评' }}</button>
      </template>
    </div>

    <OwnKeyPicker v-if="quotaOut" what="点评" @chosen="submit" />
    <div v-if="error" class="runbox-error">
      {{ error }}
      <a v-if="needLogin" :href="loginUrl()">去登录</a>
    </div>
    <CritiqueCard v-if="critique" :critique="critique" />
    <div v-else-if="sampleCrit && !image" :class="sample ? 'drawbox-out sample' : ''">
      <div v-if="sample" class="drawbox-pic">
        <img :src="withBase(sample)" alt="示例图片" loading="lazy" />
        <span class="drawbox-ai">示例 · AI 生成</span>
      </div>
      <CritiqueCard :critique="sampleCrit" sample />
    </div>
  </div>
</template>
