<template>
  <Teleport to="body">
    <div
      v-if="visible"
      :class="[
        'fixed z-[60] flex items-center rounded-xl bg-[#1f1238] p-1 shadow-lg',
        mobile ? 'inset-x-3 bottom-[calc(12px+env(safe-area-inset-bottom))] justify-center' : '-translate-x-1/2'
      ]"
      :style="mobile ? undefined : { top: `${pos.top}px`, left: `${pos.left}px` }"
      data-testid="selection-share"
      @mousedown.prevent
    >
      <button type="button" :class="btnClass" @click="makeCard">
        <Icon name="sparkles" size="sm" />
        {{ t('shareCard.makeCard') }}
      </button>
      <span class="h-4 w-px bg-white/20"></span>
      <button type="button" :class="btnClass" @click="copy">{{ copied ? t('shareCard.copied') : t('shareCard.copy') }}</button>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
// Select text inside `root` → a small toolbar: 「生成卡片」 emits the passage for a quote card,
// 「复制」 copies it with the page title and link. Floats above the selection on desktop; a bar at
// the bottom on phones, where the system selection menu sits above the text.
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { track } from '@/utils/analytics'
import { blocksFromRange, clampBlocks, headingBefore, QUOTE_MIN_CHARS, type ShareBlock } from '@/utils/shareCard'

const props = defineProps<{ root: HTMLElement | null; title: string; skip?: string }>()
const emit = defineEmits<{ (e: 'quote', blocks: ShareBlock[], truncated: boolean, anchor: string): void }>()

const { t } = useI18n()
const visible = ref(false)
const mobile = ref(false)
const copied = ref(false)
const pos = ref({ top: 0, left: 0 })
let blocks: ShareBlock[] = []
let anchor = ''
let timer = 0

const btnClass = computed(() => [
  'inline-flex items-center gap-1.5 whitespace-nowrap rounded-lg font-semibold text-white hover:bg-white/10',
  mobile.value ? 'flex-1 justify-center px-3 py-2.5 text-[15px]' : 'px-3 py-1.5 text-sm'
])

function isCoarse() {
  return window.matchMedia('(pointer: coarse)').matches || window.innerWidth < 768
}

function read() {
  const sel = window.getSelection()
  if (!props.root || !sel || sel.isCollapsed || !sel.rangeCount) {
    visible.value = false
    return
  }
  const range = sel.getRangeAt(0)
  const found = blocksFromRange(range, props.root, props.skip || '')
  const chars = found.reduce((n, b) => n + b.text.replace(/\s/g, '').length, 0)
  if (chars < QUOTE_MIN_CHARS) {
    visible.value = false
    return
  }
  blocks = found
  anchor = headingBefore(range, props.root)
  mobile.value = isCoarse()
  const rect = range.getBoundingClientRect()
  pos.value = {
    top: Math.max(64, rect.top - 52),
    left: Math.min(window.innerWidth - 110, Math.max(110, rect.left + rect.width / 2))
  }
  copied.value = false
  visible.value = true
}

function schedule(delay: number) {
  window.clearTimeout(timer)
  timer = window.setTimeout(read, delay)
}

const onSelection = () => schedule(isCoarse() ? 300 : 120)
const onMouseUp = () => schedule(10)
const onScroll = () => {
  if (visible.value && !mobile.value) read()
}

function makeCard() {
  const kept = clampBlocks(blocks)
  emit('quote', kept.blocks, kept.truncated, anchor)
  visible.value = false
  window.getSelection()?.removeAllRanges()
}

async function copy() {
  const text = blocks.map((b) => b.text).join('\n\n')
  try {
    await navigator.clipboard.writeText(`${text}\n\n——《${props.title}》${window.location.origin}${window.location.pathname}`)
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
