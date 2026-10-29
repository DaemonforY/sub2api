<script setup lang="ts">
// 课内生图练习 (学画画): the learner writes a prompt for the exercise, it is drawn right here, and an
// AI teacher scores the picture against the exercise's rubric with one concrete next step. Free
// drawings a day on the site's learning key, then the learner's own key. Pictures stay in this
// browser (a small history per exercise); the server keeps only the prompt and the scores.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { track } from '../analytics'
import {
  ApiError,
  QUOTA_REASONS,
  chooseOwnKey,
  drawExercise,
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
import { CANVAS_SITE } from '../tracks'
import CritiqueCard from './CritiqueCard.vue'
import OwnKeyPicker from './OwnKeyPicker.vue'

const props = withDefaults(
  defineProps<{
    exercise: string
    /** The starting prompt in the box (deliberately incomplete: the exercise is to finish it). */
    prompt: string
    size?: '1024x1024' | '1024x1536' | '1536x1024'
    /** A sample answer: picture path and its critique (JSON), shown before the learner draws. */
    sample?: string
    sampleCritique?: string
    samplePrompt?: string
    title?: string
  }>(),
  { size: '1024x1024', title: '动手画' },
)

interface Attempt {
  prompt: string
  thumb: string
  score?: number
  at: number
}

const SIZES = [
  { id: '1024x1024', label: '方形 1:1' },
  { id: '1024x1536', label: '竖版 2:3' },
  { id: '1536x1024', label: '横版 3:2' },
] as const
const HISTORY_MAX = 8
const historyKey = `learn_draw_${props.exercise}`

const input = ref(props.prompt)
const size = ref<string>(props.size)
const info = ref<ExerciseView | null>(null)
const enabled = ref<boolean | null>(null)
const signedIn = ref(false)
const running = ref(false)
const elapsed = ref(0)
const error = ref('')
const needLogin = ref(false)
const quotaOut = ref(false)
const image = ref('')
const critique = ref<Critique | null>(null)
const critiqueError = ref('')
const history = ref<Attempt[]>([])
const viewing = ref<Attempt | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const sampleCrit = computed<Critique | null>(() => {
  try {
    return props.sampleCritique ? JSON.parse(props.sampleCritique) : null
  } catch {
    return null
  }
})
const best = computed(() => progress.practice[props.exercise] || 0)
const canvasHref = computed(() => `${CANVAS_SITE}/image?prompt=${encodeURIComponent(input.value.trim())}&utm_source=learn`)

onMounted(async () => {
  signedIn.value = !!token()
  try {
    history.value = JSON.parse(localStorage.getItem(historyKey) || '[]')
  } catch {
    history.value = []
  }
  const [cfg, ex] = await Promise.all([learnConfig(), exerciseInfo(props.exercise)])
  enabled.value = cfg.run_enabled
  info.value = ex
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})

// A small JPEG of the drawing for the history (localStorage is only a few MB).
function thumbnail(src: string): Promise<string> {
  return new Promise((resolve) => {
    const im = new Image()
    im.onload = () => {
      const scale = 240 / Math.max(im.width, im.height)
      const c = document.createElement('canvas')
      c.width = Math.round(im.width * scale)
      c.height = Math.round(im.height * scale)
      c.getContext('2d')?.drawImage(im, 0, 0, c.width, c.height)
      resolve(c.toDataURL('image/jpeg', 0.72))
    }
    im.onerror = () => resolve('')
    im.src = src
  })
}

async function remember(prompt: string, src: string, score?: number) {
  const thumb = await thumbnail(src)
  if (!thumb) return
  history.value = [{ prompt, thumb, score, at: Date.now() }, ...history.value].slice(0, HISTORY_MAX)
  try {
    localStorage.setItem(historyKey, JSON.stringify(history.value))
  } catch {
    history.value = history.value.slice(0, 3)
    try {
      localStorage.setItem(historyKey, JSON.stringify(history.value))
    } catch {
      // storage full: the history just isn't kept
    }
  }
}

async function draw() {
  const text = input.value.trim()
  if (!text || running.value) return
  running.value = true
  error.value = ''
  needLogin.value = false
  quotaOut.value = false
  critiqueError.value = ''
  viewing.value = null
  elapsed.value = 0
  timer = setInterval(() => elapsed.value++, 1000)
  try {
    const res = await drawExercise(props.exercise, text, size.value)
    image.value = res.image
    critique.value = res.critique || null
    critiqueError.value = res.critique_error || ''
    track('learn_draw', { exercise: props.exercise, score: res.critique?.score ?? -1 })
    remember(text, res.image, res.critique?.score)
  } catch (e) {
    const err = e as ApiError
    if (QUOTA_REASONS.includes(err.reason)) {
      progress.drawsLeft = 0
      if (ownKey.id && err.reason !== 'LEARN_OWN_KEY_REQUIRED') {
        stop()
        return draw()
      }
      quotaOut.value = true
    } else if (err.reason === 'LEARN_KEY_INVALID') {
      chooseOwnKey(null)
      error.value = err.message
    } else if (err.status === 401) {
      needLogin.value = true
      error.value = '登录已过期，请重新登录后再画'
    } else {
      error.value = err.message || '生成失败，请稍后再试'
    }
  } finally {
    stop()
  }
}

function stop() {
  running.value = false
  if (timer) clearInterval(timer)
  timer = null
}

function useRevised(p: string) {
  input.value = p
}

function show(a: Attempt) {
  viewing.value = viewing.value === a ? null : a
}

function load(a: Attempt) {
  input.value = a.prompt
  viewing.value = null
}

function reset() {
  input.value = props.prompt
}
</script>

<template>
  <div class="drawbox" data-testid="drawbox">
    <div class="runbox-head">
      <span class="runbox-title">🎨 {{ title }}{{ info ? `：${info.title}` : '' }}</span>
      <span v-if="best" class="drawbox-best">最好成绩 {{ best }} 分</span>
      <span v-if="signedIn && enabled && (runMode.ownKeyOnly || progress.drawsLeft === 0) && ownKey.id" class="runbox-quota">
        用你的 Key「{{ ownKey.name }}」画（正常计费）<a href="javascript:void 0" @click="chooseOwnKey(null)">{{ runMode.ownKeyOnly ? '换一个' : '不用了' }}</a>
      </span>
      <span v-else-if="signedIn && enabled && progress.drawsLeft !== null && !runMode.ownKeyOnly" class="runbox-quota">今日免费生图剩余 {{ progress.drawsLeft }} 张</span>
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

    <textarea v-model="input" class="runbox-input" rows="4" placeholder="在这里写提示词" @keydown.ctrl.enter="draw" @keydown.meta.enter="draw"></textarea>

    <div class="drawbox-sizes">
      <button v-for="s in SIZES" :key="s.id" :class="{ on: size === s.id }" @click="size = s.id">{{ s.label }}</button>
      <a href="javascript:void 0" class="drawbox-reset" @click="reset">恢复起始提示词</a>
    </div>

    <div class="runbox-actions">
      <template v-if="enabled === false">
        <span class="runbox-note">课内生图暂未开放，可以先复制提示词去画布里画。</span>
        <a class="runbox-btn ghost" :href="canvasHref" target="_blank" rel="noopener">在画布里画 ↗</a>
      </template>
      <template v-else-if="!signedIn">
        <a class="runbox-btn" :href="loginUrl()">登录后画</a>
        <span class="runbox-note">{{ runMode.ownKeyOnly ? '登录 HiveGPT 后用自己的 Key 画' : '登录 HiveGPT 后每天有免费生图次数' }}</span>
      </template>
      <template v-else>
        <button class="runbox-btn" :disabled="running || !input.trim()" data-testid="drawbox-run" @click="draw">
          {{ running ? `正在画… ${elapsed} 秒` : image ? '改好了，再画一张' : '画出来，请 AI 老师点评' }}
        </button>
        <span class="runbox-note">{{ running ? '一张图大约 30–90 秒，画完会自动点评' : 'Ctrl / ⌘ + Enter 也可以' }}</span>
      </template>
    </div>

    <OwnKeyPicker v-if="quotaOut" what="生图" @chosen="draw" />
    <div v-if="error" class="runbox-error">
      {{ error }}
      <a v-if="needLogin" :href="loginUrl()">去登录</a>
    </div>

    <div v-if="image && !viewing" class="drawbox-out">
      <div class="drawbox-pic">
        <img :src="image" alt="你画的图" />
        <div class="drawbox-pic-actions">
          <a class="runbox-btn small ghost" :href="image" download="hivegpt-practice.jpg">下载</a>
          <a class="runbox-btn small ghost" :href="canvasHref" target="_blank" rel="noopener">在画布里继续改 ↗</a>
        </div>
        <span class="drawbox-ai">AI 生成</span>
      </div>
      <CritiqueCard v-if="critique" :critique="critique" @use-revised="useRevised" />
      <p v-else-if="critiqueError" class="runbox-error">图画好了，但点评没出来：{{ critiqueError }}</p>
    </div>
    <div v-else-if="viewing" class="drawbox-out">
      <div class="drawbox-pic">
        <img :src="viewing.thumb" alt="之前画的图" />
      </div>
      <div class="drawbox-old">
        <p class="runbox-note">{{ new Date(viewing.at).toLocaleString() }}{{ viewing.score !== undefined ? ` · ${viewing.score} 分` : '' }}</p>
        <p class="drawbox-old-prompt">{{ viewing.prompt }}</p>
        <button class="runbox-btn small" @click="load(viewing)">用这段提示词</button>
      </div>
    </div>
    <div v-else-if="sample && sampleCrit" class="drawbox-out sample">
      <div class="drawbox-pic">
        <img :src="withBase(sample)" alt="示例：一个达标的答案" loading="lazy" />
        <span class="drawbox-ai">示例 · AI 生成</span>
      </div>
      <div>
        <p v-if="samplePrompt" class="drawbox-old-prompt"><b>示例提示词：</b>{{ samplePrompt }}</p>
        <CritiqueCard :critique="sampleCrit" sample />
      </div>
    </div>

    <div v-if="history.length" class="drawbox-history">
      <span class="runbox-note">我画过的（只存在这台设备上）：</span>
      <button v-for="a in history" :key="a.at" :class="{ on: viewing === a }" :title="a.prompt" @click="show(a)">
        <img :src="a.thumb" alt="" />
        <small v-if="a.score !== undefined">{{ a.score }}</small>
      </button>
    </div>
  </div>
</template>
