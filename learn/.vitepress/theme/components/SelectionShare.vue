<script setup lang="ts">
// Select text in the article → a small toolbar: 「生成卡片」 opens the quote card, 「复制」 copies the
// passage with its source. Floats above the selection on desktop; a bar at the bottom on phones,
// where the system selection menu sits above the text.
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useData } from 'vitepress'
import { blocksFromRange, clampBlocks, headingBefore, QUOTE_MIN_CHARS, type ShareBlock } from '../shareCard'
import { openQuoteCard } from '../share'
import { track } from '../analytics'

// Interactive widgets and page chrome: selecting inside them is not quoting the article.
const SKIP = '.runbox, .quiz, .checkpoint, .interview, .canvas-try, .cert-panel, .cert-view, .ownkey, .showcase, .mermaid-box, .lesson-head, .lesson-foot, .share-top, .share-bottom, .track-page, .hu-home, .campus-perks, .home-tracks'

const { page } = useData()
const visible = ref(false)
const mobile = ref(false)
const pos = ref({ top: 0, left: 0 })
const copied = ref(false)
let blocks: ShareBlock[] = []
let anchor = ''
let timer = 0

const style = computed(() => (mobile.value ? {} : { top: `${pos.value.top}px`, left: `${pos.value.left}px` }))

function read() {
  const sel = window.getSelection()
  const root = document.querySelector('.vp-doc')
  if (!page.value.shareCard || !sel || sel.isCollapsed || !sel.rangeCount || !root) {
    visible.value = false
    return
  }
  const range = sel.getRangeAt(0)
  const found = blocksFromRange(range, root, SKIP)
  const chars = found.reduce((n, b) => n + b.text.replace(/\s/g, '').length, 0)
  if (chars < QUOTE_MIN_CHARS) {
    visible.value = false
    return
  }
  blocks = found
  anchor = headingBefore(range, root)
  mobile.value = window.matchMedia('(pointer: coarse)').matches || window.innerWidth < 768
  const rect = range.getBoundingClientRect()
  pos.value = {
    top: Math.max(64, rect.top - 52),
    left: Math.min(window.innerWidth - 110, Math.max(110, rect.left + rect.width / 2)),
  }
  copied.value = false
  visible.value = true
}

function schedule(delay: number) {
  window.clearTimeout(timer)
  timer = window.setTimeout(read, delay)
}

const onSelection = () => schedule(mobile.value || window.matchMedia('(pointer: coarse)').matches ? 300 : 120)
const onMouseUp = () => schedule(10)
const onScroll = () => {
  if (visible.value && !mobile.value) read()
}

function makeCard() {
  const { blocks: kept, truncated } = clampBlocks(blocks)
  openQuoteCard(kept, truncated, anchor)
  visible.value = false
  window.getSelection()?.removeAllRanges()
}

async function copy() {
  const text = blocks.map((b) => b.text).join('\n\n')
  const url = window.location.origin + window.location.pathname
  try {
    await navigator.clipboard.writeText(`${text}\n\n——《${page.value.shareCard?.title || page.value.title}》${url}`)
    copied.value = true
    track('share_link_copy', { kind: 'quote_text' })
    window.setTimeout(() => (visible.value = false), 900)
  } catch {
    // clipboard unavailable: the native copy still works
  }
}

onMounted(() => {
  document.addEventListener('selectionchange', onSelection)
  document.addEventListener('mouseup', onMouseUp)
  window.addEventListener('scroll', onScroll, { passive: true })
})
onUnmounted(() => {
  window.clearTimeout(timer)
  document.removeEventListener('selectionchange', onSelection)
  document.removeEventListener('mouseup', onMouseUp)
  window.removeEventListener('scroll', onScroll)
})
</script>

<template>
  <div v-if="visible" :class="['sel-share', { 'is-mobile': mobile }]" :style="style" data-testid="selection-share" @mousedown.prevent>
    <button @click="makeCard">
      <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="M4 5h16v14H4zM4 15l4-4 4 4 3-3 5 5M15 9h.01" /></svg>
      生成卡片
    </button>
    <span class="sel-share-sep"></span>
    <button @click="copy">{{ copied ? '已复制' : '复制' }}</button>
  </div>
</template>
