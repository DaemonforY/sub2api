<template>
  <div class="mx-auto max-w-3xl px-4 pb-16 pt-8 sm:px-6">
    <h1 class="text-2xl font-bold">我的音色</h1>
    <p class="mt-1 text-sm leading-6 text-ink-500">录一段 10–30 秒的声音，复刻成你自己的 AI 音色，用它给视频配音。只有你自己能用；复刻用的是豆包声音复刻 2.0，录音交给豆包训练，本站不保存原始录音。</p>

    <div v-if="!session.key" class="card mt-6 p-10 text-center">
      <p class="text-ink-500">登录后才能复刻音色</p>
      <button class="btn-primary mt-4" @click="doSignIn"><LogIn class="h-4 w-4" />登录 HiveGPT 账号</button>
    </div>

    <template v-else>
      <div v-if="loading" class="flex justify-center py-16 text-ink-400"><Loader2 class="h-5 w-5 animate-spin" /></div>
      <template v-else>
        <p v-if="!info.enabled && !items.length" class="card mt-6 p-6 text-sm text-ink-500">声音复刻暂未开通，请先用创作页里的预置音色。</p>

        <!-- my voices -->
        <div v-for="c in items" :key="c.id" class="card mt-4 p-4" :data-testid="`clone-${c.id}`">
          <div class="flex flex-wrap items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-600 dark:bg-brand-900/30"><Mic class="h-5 w-5" /></div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <input v-if="renaming === c.id" v-model.trim="newName" maxlength="20" class="input w-40 py-1" @keydown.enter="rename(c)" @blur="rename(c)" />
                <button v-else class="truncate font-medium hover:text-brand-600" title="改名" @click="startRename(c)">{{ c.name }}</button>
                <span class="rounded-full px-2 py-0.5 text-xs" :class="badge(c).cls">{{ badge(c).text }}</span>
              </div>
              <p class="mt-0.5 text-xs text-ink-400">
                {{ new Date(c.created_at).toLocaleDateString() }} 创建 · 还能重录 {{ c.retrains_left }} 次<template v-if="c.charged"> · 已付 ${{ c.price.toFixed(2) }}</template>
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button v-if="c.status === 'ready'" class="btn-ghost btn-sm" :disabled="previewing === c.id" @click="preview(c)">
                <Loader2 v-if="previewing === c.id" class="h-3.5 w-3.5 animate-spin" /><Play v-else class="h-3.5 w-3.5" />试听
              </button>
              <button v-if="c.status === 'ready'" class="btn-primary btn-sm" @click="useVoice(c)">用它做视频</button>
              <button v-if="c.status !== 'training' && c.status !== 'disabled' && c.retrains_left > 0 && info.enabled" class="btn-ghost btn-sm" @click="openRecorder(c)">重录</button>
              <button class="btn-ghost btn-sm text-red-600" @click="remove(c)">删除</button>
            </div>
          </div>
          <p v-if="c.status === 'training'" class="mt-3 flex items-center gap-2 text-sm text-ink-500"><Loader2 class="h-4 w-4 animate-spin" />正在复刻，通常几十秒到几分钟，可以先离开这个页面</p>
          <p v-if="c.error" class="mt-3 text-sm text-red-600">{{ c.error }}</p>
          <div v-if="c.status === 'ready'" class="mt-3 flex gap-2">
            <input v-model="previewText" maxlength="60" class="input py-1.5 text-xs" placeholder="试听的句子（可选，60 字以内）" />
          </div>
        </div>

        <!-- new voice / re-record -->
        <button v-if="info.enabled && !recorderOpen && items.length < info.max_per_user" class="btn-primary mt-6" data-testid="clone-new" @click="openRecorder(null)"><Plus class="h-4 w-4" />复刻我的声音</button>
        <p v-else-if="info.enabled && !recorderOpen && items.length >= info.max_per_user" class="mt-6 text-sm text-ink-500">每人最多 {{ info.max_per_user }} 个音色，删掉一个不用的才能新建。</p>

        <section v-if="recorderOpen" class="card mt-6 p-5" data-testid="clone-recorder">
          <div class="flex items-center justify-between">
            <h2 class="font-semibold">{{ target ? `重录「${target.name}」` : '复刻我的声音' }}</h2>
            <button class="btn-ghost btn-sm" @click="closeRecorder">取消</button>
          </div>

          <template v-if="!target">
            <label class="mt-4 block text-sm font-medium">音色名字</label>
            <input v-model.trim="name" maxlength="20" class="input mt-1 max-w-xs" placeholder="比如：我的声音" data-testid="clone-name" />
          </template>

          <div class="mt-5 rounded-xl bg-ink-50 p-4 text-sm leading-7 dark:bg-ink-800/60">
            <p class="text-xs font-medium text-ink-500">请用平时说话的语气朗读：</p>
            <p class="mt-1 text-base">{{ info.script }}</p>
          </div>
          <ul class="mt-3 list-disc space-y-1 pl-5 text-xs leading-5 text-ink-500">
            <li>找个安静的房间，手机或电脑离嘴 20–30 厘米，不要开背景音乐。</li>
            <li>录 10–30 秒，只能有你一个人的声音；读错了可以接着读，不用重来。</li>
            <li>也可以上传已有的录音（wav / mp3 / m4a，10 MB 以内），但必须是你本人或已获本人同意的声音。</li>
          </ul>

          <div class="mt-5 flex flex-wrap items-center gap-3">
            <button v-if="!rec" class="btn-primary" :disabled="submitting" data-testid="clone-record" @click="record"><Mic class="h-4 w-4" />{{ audio ? '重新录' : '开始录音' }}</button>
            <button v-else class="btn-primary !from-red-500 !to-red-600" @click="stopRecording"><Square class="h-4 w-4" />停止（{{ elapsed.toFixed(0) }} 秒）</button>
            <div v-if="rec" class="h-2 w-32 overflow-hidden rounded-full bg-ink-200 dark:bg-ink-700"><div class="h-full bg-brand-500 transition-[width] duration-75" :style="{ width: `${Math.min(100, level * 140)}%` }"></div></div>
            <label v-if="!rec" class="btn-ghost cursor-pointer"><UploadCloud class="h-4 w-4" />上传录音<input type="file" accept=".wav,.mp3,.m4a,audio/wav,audio/mpeg,audio/mp4,audio/x-m4a" class="hidden" @change="pickFile" /></label>
          </div>
          <p v-if="rec && elapsed < 10" class="mt-2 text-xs text-ink-400">至少录 10 秒</p>
          <div v-if="audio && !rec" class="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <audio :src="audioURL" controls class="h-9"></audio>
            <span class="text-ink-500">{{ audioLabel }}</span>
          </div>

          <label v-if="!target" class="mt-5 flex items-start gap-2 text-sm leading-6">
            <input v-model="consent" type="checkbox" class="mt-1.5 accent-brand-500" data-testid="clone-consent" />
            <span>{{ info.consent }}</span>
          </label>

          <div class="mt-5 flex flex-wrap items-center gap-3">
            <button class="btn-primary" :disabled="!canSubmit" data-testid="clone-submit" @click="submit">
              <Loader2 v-if="submitting" class="h-4 w-4 animate-spin" />{{ target ? '用新录音重新复刻' : '开始复刻' }}
            </button>
            <span v-if="!target" class="text-xs text-ink-500">复刻成功后从余额扣 ${{ info.price.toFixed(2) }}，失败不扣；之后用它配音按字数计费（每万字 ${{ voicePrice }}）</span>
            <span v-else class="text-xs text-ink-500">重录不另收费</span>
          </div>
        </section>
      </template>
    </template>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Loader2, LogIn, Mic, Play, Plus, Square, UploadCloud } from 'lucide-vue-next'
import { api, apiUpload, catalog, loadMe, session, signIn } from '../lib/api'
import { startRecording } from '../lib/recorder'
import { savePrefs } from '../lib/prefs'
import { toast, toastError } from '../lib/toast'

const router = useRouter()
const loading = ref(true)
const items = ref([])
const info = ref({ enabled: false, price: 0, max_per_user: 3, script: '', consent: '', retrains: 5 })
const voicePrice = ref(0)

const recorderOpen = ref(false)
const target = ref(null) // the clone being re-recorded
const name = ref('')
const consent = ref(false)
const audio = ref(null) // Blob or File
const audioURL = ref('')
const audioSeconds = ref(0)
const rec = ref(null)
const elapsed = ref(0)
const level = ref(0)
const submitting = ref(false)
const previewing = ref(0)
const previewText = ref('')
const renaming = ref(0)
const newName = ref('')
let tick = null
let poll = null
let alive = true
let player = null

const MAX_SECONDS = 30

const canSubmit = computed(() => !!audio.value && !rec.value && !submitting.value && (target.value || (name.value && consent.value)))
const audioLabel = computed(() => (audio.value instanceof File ? audio.value.name : `录音 ${audioSeconds.value.toFixed(0)} 秒`))

function badge(c) {
  return (
    {
      training: { text: '复刻中', cls: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' },
      ready: { text: '可以使用', cls: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' },
      failed: { text: '复刻失败', cls: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300' },
      disabled: { text: '已停用', cls: 'bg-ink-100 text-ink-500 dark:bg-ink-800' }
    }[c.status] || { text: c.status, cls: 'bg-ink-100 text-ink-500' }
  )
}

async function load() {
  try {
    const r = await api('/clones')
    const wasTraining = new Set(items.value.filter((c) => c.status === 'training').map((c) => c.id))
    items.value = r.items || []
    // the fee leaves the balance when training succeeds
    if (items.value.some((c) => c.status === 'ready' && wasTraining.has(c.id))) loadMe()
    info.value = r.info || info.value
  } catch (err) {
    toastError(err)
  } finally {
    loading.value = false
  }
  clearTimeout(poll)
  if (alive && items.value.some((c) => c.status === 'training')) poll = setTimeout(load, 4000)
}

async function doSignIn() {
  try {
    await signIn()
    await load()
  } catch (err) {
    toastError(err)
  }
}

function setAudio(blob, seconds) {
  if (audioURL.value) URL.revokeObjectURL(audioURL.value)
  audio.value = blob
  audioSeconds.value = seconds
  audioURL.value = blob ? URL.createObjectURL(blob) : ''
}

function openRecorder(c) {
  target.value = c
  name.value = ''
  consent.value = false
  setAudio(null, 0)
  recorderOpen.value = true
}

function closeRecorder() {
  rec.value?.cancel()
  rec.value = null
  clearInterval(tick)
  setAudio(null, 0)
  recorderOpen.value = false
}

async function record() {
  if (!navigator.mediaDevices?.getUserMedia) return toast('这个浏览器不能录音，请换 Chrome / Safari，或上传录音文件', 'error', 5000)
  try {
    rec.value = await startRecording({ onLevel: (v) => (level.value = v) })
  } catch (err) {
    return toast(err?.name === 'NotAllowedError' ? '没有麦克风权限：请在浏览器地址栏允许使用麦克风后再试' : '打不开麦克风，请检查设备后重试，或上传录音文件', 'error', 6000)
  }
  elapsed.value = 0
  tick = setInterval(() => {
    elapsed.value = rec.value?.seconds() || 0
    if (elapsed.value >= MAX_SECONDS) stopRecording()
  }, 200)
}

function stopRecording() {
  clearInterval(tick)
  if (!rec.value) return
  const { blob, seconds } = rec.value.stop()
  rec.value = null
  level.value = 0
  if (seconds < 10) {
    setAudio(null, 0)
    return toast('录音不到 10 秒，请再录长一点', 'error')
  }
  setAudio(blob, seconds)
}

function pickFile(e) {
  const f = e.target.files?.[0]
  e.target.value = ''
  if (!f) return
  if (f.size > 10 * 1024 * 1024) return toast('录音文件不能超过 10 MB', 'error')
  setAudio(f, 0)
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  const form = new FormData()
  form.append('audio', audio.value, audio.value instanceof File ? audio.value.name : 'voice.wav')
  try {
    if (target.value) {
      await apiUpload(`/clones/${target.value.id}/retrain`, form)
      toast('已重新提交，复刻好后就能用')
    } else {
      form.append('name', name.value)
      form.append('consent', consent.value ? '1' : '0')
      await apiUpload('/clones', form)
      toast('已提交复刻，通常几十秒到几分钟')
    }
    closeRecorder()
    await load()
    loadMe()
  } catch (err) {
    toastError(err)
  } finally {
    submitting.value = false
  }
}

async function preview(c) {
  previewing.value = c.id
  try {
    const res = await fetch(`/api/v1/video/clones/${c.id}/preview`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${session.key}` },
      body: JSON.stringify({ text: previewText.value })
    })
    if (!res.ok || !(res.headers.get('Content-Type') || '').startsWith('audio/')) {
      const data = await res.json().catch(() => null)
      throw new Error((data?.message || '试听失败，请稍后再试').replace(/（[A-Za-z][^（）]*）\s*$/, ''))
    }
    const url = URL.createObjectURL(await res.blob())
    player?.pause()
    player = new Audio(url)
    player.onended = () => URL.revokeObjectURL(url)
    await player.play()
  } catch (err) {
    toastError(err)
  } finally {
    previewing.value = 0
  }
}

function useVoice(c) {
  savePrefs({ voice: c.voice, mode: 'film' })
  router.push('/')
}

function startRename(c) {
  renaming.value = c.id
  newName.value = c.name
}

async function rename(c) {
  if (renaming.value !== c.id) return
  renaming.value = 0
  if (!newName.value || newName.value === c.name) return
  try {
    Object.assign(c, await api(`/clones/${c.id}`, { method: 'PUT', body: { name: newName.value } }))
  } catch (err) {
    toastError(err)
  }
}

async function remove(c) {
  if (!confirm(`删除音色「${c.name}」？已经配好音的作品不受影响，但以后不能再用它配音，复刻费用不退。`)) return
  try {
    await api(`/clones/${c.id}`, { method: 'DELETE' })
    items.value = items.value.filter((x) => x.id !== c.id)
  } catch (err) {
    toastError(err)
  }
}

onMounted(() => {
  catalog().then((c) => (voicePrice.value = c.voice_price_per_10k || 0)).catch(() => {})
  if (session.key) load()
  else loading.value = false
})
onBeforeUnmount(() => {
  alive = false
  clearTimeout(poll)
  clearInterval(tick)
  rec.value?.cancel()
  player?.pause()
  if (audioURL.value) URL.revokeObjectURL(audioURL.value)
})
</script>
