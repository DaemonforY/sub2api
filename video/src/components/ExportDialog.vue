<template>
  <Modal title="导出视频 · 发布到平台" wide @close="$emit('close')">
    <!-- 1. the MP4 files -->
    <section>
      <h3 class="text-sm font-semibold">视频文件</h3>
      <p class="mt-1 text-xs leading-5 text-ink-500">在服务器上逐帧渲染：1080p 30 帧、高码率 H.264 + AAC，带配音和字幕。全站按顺序排队，可以关掉窗口，回来再下载。</p>
      <p class="mt-1 text-xs leading-5 text-ink-500">字幕：{{ subtitleLabel }}（在播放器右下角的「字幕」里调整，改了之后重新导出）</p>
      <div class="mt-3 grid gap-3 sm:grid-cols-2">
        <div v-for="l in layouts" :key="l.id" class="rounded-xl border border-ink-200 p-3 dark:border-ink-700">
          <div class="flex items-start gap-3">
            <div class="flex shrink-0 items-center justify-center overflow-hidden rounded-lg bg-ink-100 dark:bg-ink-800" :class="l.id.startsWith('portrait') ? 'h-24 w-[54px]' : 'h-[54px] w-24'">
              <img v-if="exp(l.id)?.cover" :src="exp(l.id).cover + '&inline=1'" class="h-full w-full object-cover" alt="封面" />
              <component :is="l.icon" v-else class="h-5 w-5 text-ink-400" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium">{{ l.name }} <span class="font-normal text-ink-400">{{ l.size }} · 30fps</span></p>
              <p class="mt-0.5 text-xs text-ink-500">适合 {{ l.for }}</p>
              <template v-if="exp(l.id)">
                <p v-if="exp(l.id).status === 'queued'" class="mt-2 text-xs text-ink-500">排队中，前面还有 {{ Math.max(0, exp(l.id).position - 1) }} 个</p>
                <div v-else-if="exp(l.id).status === 'running'" class="mt-2">
                  <div class="h-1.5 overflow-hidden rounded-full bg-ink-100 dark:bg-ink-800"><div class="h-full rounded-full bg-brand-500 transition-all" :style="{ width: `${Math.round(exp(l.id).progress * 100)}%` }"></div></div>
                  <p class="mt-1 text-xs text-ink-500">渲染中 {{ Math.round(exp(l.id).progress * 100) }}%</p>
                </div>
                <p v-else-if="exp(l.id).status === 'ready'" class="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
                  已导出 · {{ fmtBytes(exp(l.id).size) }} · {{ fmtTime(exp(l.id).duration) }}<span v-if="!exp(l.id).current" class="ml-1 text-amber-600">（作品有改动，可重新导出）</span>
                </p>
                <p v-else-if="exp(l.id).status === 'failed'" class="mt-2 text-xs text-red-600">{{ exp(l.id).error }}</p>
              </template>
            </div>
          </div>
          <div class="mt-3 flex flex-wrap gap-2">
            <template v-if="exp(l.id)?.status === 'ready'">
              <a class="btn-primary btn-sm" :href="exp(l.id).url" download><Download class="h-3.5 w-3.5" />下载 MP4</a>
              <a v-if="exp(l.id).cover" class="btn-ghost btn-sm" :href="exp(l.id).cover" download><ImageIcon class="h-3.5 w-3.5" />封面</a>
            </template>
            <button v-if="canStart(l.id)" class="btn-ghost btn-sm" :disabled="starting === l.id" @click="start(l.id)">
              <Loader2 v-if="starting === l.id" class="h-3.5 w-3.5 animate-spin" /><Film v-else class="h-3.5 w-3.5" />{{ exp(l.id) ? '重新导出' : '导出' }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- 2. per-platform copy and the one-click hand-off -->
    <section class="mt-6">
      <div class="flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold">发布到平台</h3>
        <button class="btn-ghost btn-sm" :disabled="writing" @click="writeCopy">
          <Loader2 v-if="writing" class="h-3.5 w-3.5 animate-spin" /><Sparkles v-else class="h-3.5 w-3.5" />{{ copy ? '重新生成文案' : 'AI 写发布文案' }}
        </button>
      </div>
      <p class="mt-1 text-xs leading-5 text-ink-500">各平台都不开放第三方直接发布，所以「一键发布」会：下载对应版本的视频、把标题简介和话题复制到剪贴板、打开平台的上传页，你在页面里选视频、粘贴文案就能发。手机上会直接调起系统分享，选对应 App。文案由作品所用的模型生成，按用量从余额扣费。</p>
      <div class="mt-3 flex flex-wrap gap-1.5">
        <button v-for="pf in platforms" :key="pf.id" class="chip" :class="{ 'chip-on': platform === pf.id }" @click="platform = pf.id">{{ pf.name }}</button>
      </div>
      <div class="mt-3 space-y-3">
        <label class="block text-xs text-ink-500">
          <span class="flex justify-between"><span>{{ current.titleLabel }}</span><span :class="{ 'text-red-600': titleLen > current.titleMax }">{{ titleLen }}/{{ current.titleMax }}</span></span>
          <input v-model="draft[platform].title" class="input mt-1 w-full" :placeholder="copy ? '' : '点「AI 写发布文案」，或自己填写'" />
        </label>
        <label class="block text-xs text-ink-500">
          简介
          <textarea v-model="draft[platform].desc" rows="4" class="input mt-1 w-full resize-y leading-6"></textarea>
        </label>
        <label class="block text-xs text-ink-500">
          话题（空格分隔，最多 {{ current.tagMax }} 个）
          <input v-model="draft[platform].tags" class="input mt-1 w-full" placeholder="大模型 AI科普" />
        </label>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <button class="btn-primary btn-sm" :disabled="handing" @click="oneClick">
          <Loader2 v-if="handing" class="h-3.5 w-3.5 animate-spin" /><Send v-else class="h-3.5 w-3.5" />一键发布到{{ current.name }}
        </button>
        <button class="btn-ghost btn-sm" @click="copyText(draft[platform].title, '标题')"><Copy class="h-3.5 w-3.5" />复制标题</button>
        <button class="btn-ghost btn-sm" @click="copyText(bodyText(), '简介和话题')"><Copy class="h-3.5 w-3.5" />复制简介+话题</button>
        <a class="btn-ghost btn-sm" :href="current.upload" target="_blank" rel="noopener"><ExternalLink class="h-3.5 w-3.5" />打开上传页</a>
      </div>
      <p class="mt-2 text-xs text-ink-400">推荐用{{ current.layout === 'portrait' ? '竖屏' : '横屏' }}版。{{ current.tip }}</p>
    </section>
  </Modal>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Copy, Download, ExternalLink, Film, Image as ImageIcon, Loader2, Monitor, Send, Smartphone, Sparkles } from 'lucide-vue-next'
import Modal from './Modal.vue'
import { api } from '../lib/api'
import { fmtBytes, fmtTime } from '../lib/media'
import { toast, toastError } from '../lib/toast'
import { subtitleStyle, SUBTITLE_POSITIONS, SUBTITLE_SIZES } from '../lib/subtitles'

const props = defineProps({ project: { type: Object, required: true } })
const subtitleLabel = computed(() => {
  const st = subtitleStyle(props.project.spec?.subtitles)
  if (st.off) return '不显示'
  return `${SUBTITLE_POSITIONS.find((x) => x.id === st.position).name} · ${SUBTITLE_SIZES.find((x) => x.id === st.size).name}号`
})
defineEmits(['close'])

const layouts = [
  { id: 'portrait', name: '竖屏', size: '1080×1920', for: '抖音、视频号、小红书', icon: Smartphone },
  { id: 'landscape', name: '横屏', size: '1920×1080', for: 'B站、YouTube、西瓜视频', icon: Monitor }
]
const platforms = [
  { id: 'douyin', name: '抖音', titleLabel: '标题', titleMax: 30, tagMax: 5, layout: 'portrait', upload: 'https://creator.douyin.com/creator-micro/content/upload', tip: '在抖音创作者中心上传，作品描述里粘贴文案。' },
  { id: 'channels', name: '视频号', titleLabel: '短标题', titleMax: 16, tagMax: 5, layout: 'portrait', upload: 'https://channels.weixin.qq.com/platform/post/create', tip: '视频号助手需要用微信扫码登录。' },
  { id: 'xiaohongshu', name: '小红书', titleLabel: '标题', titleMax: 20, tagMax: 10, layout: 'portrait', upload: 'https://creator.xiaohongshu.com/publish/publish?source=official', tip: '在小红书创作服务平台选「上传视频」，可用导出的封面图做封面。' },
  { id: 'bilibili', name: 'B站', titleLabel: '标题', titleMax: 80, tagMax: 10, layout: 'landscape', upload: 'https://member.bilibili.com/platform/upload/video/frame', tip: '在B站创作中心投稿，标签逐个添加。' }
]

const exports = ref([])
const starting = ref('')
const writing = ref(false)
const handing = ref(false)
const copy = ref(null)
const platform = ref('douyin')
const draft = reactive(Object.fromEntries(platforms.map((p) => [p.id, { title: '', desc: '', tags: '' }])))
const current = computed(() => platforms.find((p) => p.id === platform.value))
const titleLen = computed(() => Array.from(draft[platform.value].title || '').length)

// The newest export of a layout.
const exp = (layout) => exports.value.find((e) => e.layout === layout)
const canStart = (layout) => {
  const e = exp(layout)
  return !e || e.status === 'failed' || (e.status === 'ready' && !e.current)
}

let timer = 0
let alive = true
async function load() {
  try {
    exports.value = (await api(`/projects/${props.project.id}/exports`)).exports || []
  } catch (err) {
    toastError(err)
  }
  clearTimeout(timer)
  if (alive && exports.value.some((e) => e.status === 'queued' || e.status === 'running')) timer = setTimeout(load, 4000)
}
async function start(layout) {
  starting.value = layout
  try {
    await api(`/projects/${props.project.id}/exports`, { method: 'POST', body: { layout } })
    await load()
  } catch (err) {
    toastError(err)
  } finally {
    starting.value = ''
  }
}

function fill(c) {
  copy.value = c
  for (const p of platforms) {
    const v = c?.[p.id]
    if (v) Object.assign(draft[p.id], { title: v.title || '', desc: v.desc || '', tags: (v.tags || []).join(' ') })
  }
}
async function writeCopy() {
  writing.value = true
  try {
    fill((await api(`/projects/${props.project.id}/publish-copy`, { method: 'POST', body: {} })).copy)
    toast('文案已生成，可以直接改')
  } catch (err) {
    toastError(err)
  } finally {
    writing.value = false
  }
}

const tags = () => (draft[platform.value].tags || '').split(/[\s,，#＃]+/).filter(Boolean).slice(0, current.value.tagMax)
const bodyText = () => [draft[platform.value].desc, tags().map((t) => `#${t}`).join(' ')].filter(Boolean).join('\n\n')
const fullText = () => [draft[platform.value].title, bodyText()].filter(Boolean).join('\n\n')

async function copyText(text, what) {
  try {
    await navigator.clipboard.writeText(text || '')
    toast(`已复制${what}`)
  } catch {
    toast('浏览器不允许自动复制，请手动选中复制')
  }
}

const mobile = () => /Android|iPhone|iPad|iPod|HarmonyOS/i.test(navigator.userAgent)

// One click: the right MP4 is downloaded (or shared on a phone), the copy goes to the clipboard and
// the platform's upload page opens. Nothing is posted on the user's behalf — no platform allows it.
async function oneClick() {
  const pf = current.value
  // A finished 1080p version of the platform's orientation, else a finished 720p one.
  const ready = (id) => (exp(id)?.status === 'ready' ? exp(id) : null)
  const e = ready(pf.layout) || ready(pf.layout + '-720') || exp(pf.layout)
  if (!e || e.status !== 'ready') {
    if (!e || canStart(pf.layout)) await start(pf.layout)
    toast(`${pf.layout === 'portrait' ? '竖屏' : '横屏'}版还在导出，完成后再点「一键发布」`)
    return
  }
  const text = fullText()
  if (mobile() && navigator.canShare) {
    handing.value = true
    try {
      const blob = await (await fetch(e.url)).blob()
      const file = new File([blob], `${props.project.title || 'video'}.mp4`, { type: 'video/mp4' })
      if (navigator.canShare({ files: [file] })) {
        await navigator.clipboard.writeText(text).catch(() => {})
        await navigator.share({ files: [file], title: draft[pf.id].title, text })
        return
      }
    } catch (err) {
      if (err?.name === 'AbortError') return
    } finally {
      handing.value = false
    }
  }
  // Desktop: open the upload page first (inside the click, so it isn't blocked), then download and copy.
  window.open(pf.upload, '_blank', 'noopener')
  const a = document.createElement('a')
  a.href = e.url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  a.remove()
  try {
    await navigator.clipboard.writeText(text)
    toast(`已下载视频并复制文案：在${pf.name}的上传页选择刚下载的视频，粘贴文案即可发布`)
  } catch {
    toast(`已下载视频：在${pf.name}的上传页选择它，文案可用「复制」按钮复制`)
  }
}

onMounted(async () => {
  load()
  try {
    const r = await api(`/projects/${props.project.id}/publish-copy`)
    if (r.copy) fill(r.copy)
  } catch {
    /* no copy yet */
  }
  if (!draft.douyin.title) {
    const t = props.project.title || ''
    for (const p of platforms) draft[p.id].title = Array.from(t).slice(0, p.titleMax).join('')
  }
})
onBeforeUnmount(() => {
  alive = false
  clearTimeout(timer)
})
</script>
