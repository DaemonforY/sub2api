<script setup lang="ts">
// The share card dialog: preview (an <img>, so phones can long-press to save it), colour themes,
// optional sharer name, and save / copy / share.
import { computed, reactive, ref, watch } from 'vue'
import { useData } from 'vitepress'
import { dataUrlToBlob, drawShareCard, SHARE_THEME_NAMES, SHARE_THEMES, shareUrl, type ShareCard, type ShareTheme } from '../shareCard'
import { myInviteCode, shareState } from '../share'
import { currentUser, loginUrl } from '../api'
import { track } from '../analytics'
import type { PageShareCard } from '../../share-data'

const THEME_KEY = 'share_card_theme'
const { page } = useData()

const theme = ref<ShareTheme>('violet')
const showName = ref(false)
const images = reactive<Partial<Record<ShareTheme, string>>>({})
const failed = ref(false)
const copied = ref('')
const aff = ref('')
const signedIn = ref(false)
const userName = ref('')
const isMobile = ref(false)
const canShareFiles = ref(false)

const data = computed(() => page.value.shareCard as PageShareCard | null)

function pageLink(): string {
  return shareUrl(window.location.origin + window.location.pathname, {
    aff: aff.value,
    medium: shareState.kind,
    anchor: shareState.kind === 'quote' ? shareState.anchor : '',
  })
}

function card(t: ShareTheme): ShareCard | null {
  const d = data.value
  if (!d) return null
  const base = {
    theme: t,
    brand: 'HiveGPT AI 学习',
    label: d.label,
    title: d.title,
    url: pageLink(),
    source: d.source,
    sharer: showName.value && userName.value ? `${userName.value} 推荐` : '',
    footer: 'hivegpt.cn/learn · 边学边做',
  }
  if (shareState.kind === 'quote') {
    return { ...base, kind: 'quote', blocks: shareState.blocks, meta: `直达这段原文 · 全文约 ${d.minutes} 分钟读完` }
  }
  return { ...base, kind: 'summary', summary: d.summary, points: d.points, meta: `约 ${d.minutes} 分钟读完` }
}

let generation = 0
async function render(t: ShareTheme) {
  const c = card(t)
  if (!c) return
  const run = generation
  try {
    const canvas = await drawShareCard(c)
    if (run !== generation) return
    images[t] = canvas.toDataURL('image/png')
  } catch {
    if (run === generation && t === theme.value) failed.value = true
  }
}

async function renderAll() {
  generation++
  failed.value = false
  for (const t of SHARE_THEMES) delete images[t]
  await render(theme.value)
  for (const t of SHARE_THEMES) if (t !== theme.value) await render(t)
}

async function openDialog() {
  const user = currentUser()
  signedIn.value = !!user
  userName.value = user?.username?.trim() || ''
  isMobile.value = window.matchMedia('(pointer: coarse)').matches || window.innerWidth < 768
  const saved = localStorage.getItem(THEME_KEY) as ShareTheme | null
  if (saved && SHARE_THEMES.includes(saved)) theme.value = saved
  aff.value = await myInviteCode()
  try {
    canShareFiles.value = !!navigator.canShare?.({ files: [new File([''], 'a.png', { type: 'image/png' })] })
  } catch {
    canShareFiles.value = false
  }
  track('share_card_open', { kind: shareState.kind, chars: shareState.blocks.reduce((n, b) => n + b.text.length, 0) })
  await renderAll()
}

watch(
  () => shareState.open,
  (open) => {
    if (open) void openDialog()
    else generation++
  },
)
watch(theme, (t) => localStorage.setItem(THEME_KEY, t))
watch(showName, () => void renderAll())

function close() {
  shareState.open = false
}

function fileName() {
  const title = (data.value?.title || 'HiveGPT').slice(0, 30)
  return `HiveGPT-${title}.png`.replace(/[\\/:*?"<>|\s]+/g, '-')
}

function flash(what: string) {
  copied.value = what
  window.setTimeout(() => (copied.value = ''), 1500)
}

function download() {
  const url = images[theme.value]
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = fileName()
  document.body.appendChild(a)
  a.click()
  a.remove()
  track('share_card_save', { kind: shareState.kind, theme: theme.value, method: 'download' })
}

function blob(): Blob | null {
  const url = images[theme.value]
  return url ? dataUrlToBlob(url) : null
}

const canCopyImage = typeof window !== 'undefined' && typeof ClipboardItem !== 'undefined' && !!navigator.clipboard?.write

async function copyImage() {
  try {
    const b = blob()
    if (!b) return
    await navigator.clipboard.write([new ClipboardItem({ [b.type]: b })])
    flash('image')
    track('share_card_save', { kind: shareState.kind, theme: theme.value, method: 'copy' })
  } catch {
    flash('image-failed')
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(pageLink())
    flash('link')
    track('share_link_copy', { kind: shareState.kind })
  } catch {
    // clipboard unavailable
  }
}

async function shareFile() {
  try {
    const b = blob()
    if (!b) return
    await navigator.share({ files: [new File([b], fileName(), { type: 'image/png' })], title: data.value?.title })
    track('share_card_save', { kind: shareState.kind, theme: theme.value, method: 'share' })
  } catch {
    // cancelled
  }
}

function onLongPress() {
  if (isMobile.value) track('share_card_save', { kind: shareState.kind, theme: theme.value, method: 'longpress' })
}
</script>

<template>
  <Teleport to="body">
    <div v-if="shareState.open" class="share-mask" data-testid="share-dialog" @click.self="close">
      <div class="share-dialog" role="dialog" aria-modal="true" aria-label="分享卡片">
        <div class="share-dialog-head">
          <b>{{ shareState.kind === 'quote' ? '摘录卡片' : '分享卡片' }}</b>
          <button class="share-x" aria-label="关闭" @click="close">×</button>
        </div>

        <div class="share-preview">
          <img v-if="images[theme]" :src="images[theme]" alt="分享卡片" data-testid="share-image" @contextmenu="onLongPress" />
          <div v-else class="share-loading">{{ failed ? '卡片生成失败，请换个浏览器再试' : '生成中…' }}</div>
        </div>
        <p v-if="shareState.truncated" class="runbox-note share-tip">选中的内容较长，卡片里只放了前 300 字。</p>

        <div class="share-themes" role="radiogroup" aria-label="卡片风格">
          <button
            v-for="t in SHARE_THEMES"
            :key="t"
            role="radio"
            :aria-checked="theme === t"
            :class="['share-theme', `is-${t}`, { active: theme === t }]"
            @click="theme = t"
          >
            <span class="share-swatch"></span>{{ SHARE_THEME_NAMES[t] }}
          </button>
        </div>

        <label v-if="userName" class="share-name">
          <input v-model="showName" type="checkbox" />
          卡片上显示我的昵称（{{ userName }}）
        </label>

        <p v-if="isMobile" class="share-hint">长按上面的图片，保存到相册或直接发给朋友</p>
        <div class="share-actions">
          <button v-if="!isMobile" class="runbox-btn" :disabled="!images[theme]" data-testid="share-download" @click="download">下载图片</button>
          <button v-if="!isMobile && canCopyImage" class="runbox-btn ghost" :disabled="!images[theme]" @click="copyImage">
            {{ copied === 'image' ? '已复制图片' : copied === 'image-failed' ? '浏览器不支持复制' : '复制图片' }}
          </button>
          <button v-if="isMobile && canShareFiles" class="runbox-btn" :disabled="!images[theme]" @click="shareFile">分享到…</button>
          <button class="runbox-btn ghost" @click="copyLink">{{ copied === 'link' ? '已复制链接' : '复制链接' }}</button>
        </div>

        <p v-if="!signedIn" class="runbox-note share-tip">
          <a :href="loginUrl()">登录</a>后生成的二维码会带上你的邀请码，好友扫码注册后绑定为你邀请的用户。
        </p>
        <p v-else-if="aff" class="runbox-note share-tip">二维码已带上你的邀请码。</p>
      </div>
    </div>
  </Teleport>
</template>
