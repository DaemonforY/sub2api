<script setup lang="ts">
// 看名画 (学画画): an open-access painting with the tools painters use to look at a picture —
// grey scale (values), blur (big shapes), mirror (fresh eyes), composition lines, a colour palette
// taken from the picture, and numbered notes. Artworks come from scripts/import_artworks.py.
import { computed, ref } from 'vue'
import { withBase } from 'vitepress'
import artworks from '../artworks.json'

interface Note {
  x: number
  y: number
  t: string
}

const props = withDefaults(
  defineProps<{
    art: string
    /** JSON array of { x, y, t }: x / y in percent of the picture. */
    notes?: string
    /** Comma-separated composition lines to offer: thirds, phi, diagonal, center. */
    lines?: string
    caption?: string
  }>(),
  { lines: 'thirds,phi,diagonal,center' },
)

const work = (artworks as Record<string, { src: string; width: number; height: number; title: string; artist: string; date: string; credit: string; link: string }>)[props.art]
const noteList = computed<Note[]>(() => {
  try {
    return props.notes ? JSON.parse(props.notes) : []
  } catch {
    return []
  }
})

type Mode = 'color' | 'gray' | 'blur' | 'mirror'
const MODES: { id: Mode; label: string; tip: string }[] = [
  { id: 'color', label: '原图', tip: '' },
  { id: 'gray', label: '灰度', tip: '去掉颜色只看明暗：哪里最亮、哪里最暗，明暗是怎么分布的。' },
  { id: 'blur', label: '眯眼看', tip: '模糊掉细节只看大的形状和色块，画面的骨架一下就出来了。' },
  { id: 'mirror', label: '镜像', tip: '左右翻转后，眼睛会像第一次看到它一样，更容易发现构图的问题。' },
]
const LINES: Record<string, string> = { thirds: '三分线', phi: '黄金分割线', diagonal: '对角线', center: '中线' }
const mode = ref<Mode>('color')
const shown = ref<Record<string, boolean>>({})
const active = ref<number | null>(null)
const palette = ref<{ hex: string; share: number; hsb: string }[] | null>(null)
const picking = ref(false)
const copied = ref('')
const img = ref<HTMLImageElement | null>(null)

const lineKeys = computed(() => props.lines.split(',').map((s) => s.trim()).filter((k) => LINES[k]))
const filter = computed(() => (mode.value === 'gray' ? 'grayscale(1)' : mode.value === 'blur' ? 'blur(6px)' : 'none'))
const tip = computed(() => MODES.find((m) => m.id === mode.value)?.tip || '')

function toggle(k: string) {
  shown.value = { ...shown.value, [k]: !shown.value[k] }
}

function hsb(r: number, g: number, b: number): string {
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const d = max - min
  let h = 0
  if (d) {
    if (max === r) h = ((g - b) / d) % 6
    else if (max === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
  }
  h = Math.round(((h * 60) + 360) % 360)
  return `色相 ${h}° · 纯度 ${max ? Math.round((d / max) * 100) : 0}% · 明度 ${Math.round((max / 255) * 100)}%`
}

// k-means on a small copy of the picture: the main colours and how much of the picture each covers.
async function pickPalette() {
  if (palette.value) {
    palette.value = null
    return
  }
  const el = img.value
  if (!el) return
  picking.value = true
  await new Promise((r) => setTimeout(r, 0))
  const w = 72
  const h = Math.max(1, Math.round((w * el.naturalHeight) / el.naturalWidth))
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return
  ctx.drawImage(el, 0, 0, w, h)
  const data = ctx.getImageData(0, 0, w, h).data
  const px: number[][] = []
  for (let i = 0; i < data.length; i += 4) px.push([data[i], data[i + 1], data[i + 2]])
  const k = 6
  let centers = Array.from({ length: k }, (_, i) => px[Math.floor(((i + 0.5) * px.length) / k)].slice())
  let assign = new Array(px.length).fill(0)
  for (let iter = 0; iter < 12; iter++) {
    assign = px.map((p) => {
      let best = 0
      let bestD = Infinity
      centers.forEach((c, j) => {
        const d = (p[0] - c[0]) ** 2 + (p[1] - c[1]) ** 2 + (p[2] - c[2]) ** 2
        if (d < bestD) {
          bestD = d
          best = j
        }
      })
      return best
    })
    centers = centers.map((c, j) => {
      const mine = px.filter((_, i) => assign[i] === j)
      if (!mine.length) return c
      return [0, 1, 2].map((ch) => mine.reduce((s, p) => s + p[ch], 0) / mine.length)
    })
  }
  const counts = centers.map((_, j) => assign.filter((a) => a === j).length)
  palette.value = centers
    .map((c, j) => {
      const [r, g, b] = c.map((v) => Math.round(v))
      return { hex: '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join(''), share: counts[j] / px.length, hsb: hsb(r, g, b) }
    })
    .filter((c) => c.share > 0.01)
    .sort((a, b) => b.share - a.share)
  picking.value = false
}

async function copy(hex: string) {
  try {
    await navigator.clipboard.writeText(hex)
    copied.value = hex
    setTimeout(() => (copied.value = ''), 1200)
  } catch {
    // clipboard unavailable
  }
}
</script>

<template>
  <figure v-if="work" class="art-viewer" data-testid="art-viewer">
    <div class="art-tools">
      <div class="art-seg">
        <button v-for="m in MODES" :key="m.id" :class="{ on: mode === m.id }" @click="mode = m.id">{{ m.label }}</button>
      </div>
      <div class="art-seg">
        <button v-for="k in lineKeys" :key="k" :class="{ on: shown[k] }" @click="toggle(k)">{{ LINES[k] }}</button>
      </div>
      <button class="art-chip" :class="{ on: palette }" :disabled="picking" @click="pickPalette">🎨 {{ picking ? '取色中…' : '取色板' }}</button>
    </div>

    <div class="art-stage" :style="{ aspectRatio: `${work.width} / ${work.height}`, maxWidth: `calc(70vh * ${work.width / work.height})` }">
      <img
        ref="img"
        :src="withBase(work.src)"
        :alt="`${work.artist}《${work.title}》`"
        :width="work.width"
        :height="work.height"
        loading="lazy"
        :style="{ filter, transform: mode === 'mirror' ? 'scaleX(-1)' : 'none' }"
      />
      <svg class="art-lines" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
        <g v-if="shown.thirds" class="l-thirds">
          <line x1="33.33" y1="0" x2="33.33" y2="100" /><line x1="66.67" y1="0" x2="66.67" y2="100" />
          <line x1="0" y1="33.33" x2="100" y2="33.33" /><line x1="0" y1="66.67" x2="100" y2="66.67" />
        </g>
        <g v-if="shown.phi" class="l-phi">
          <line x1="38.2" y1="0" x2="38.2" y2="100" /><line x1="61.8" y1="0" x2="61.8" y2="100" />
          <line x1="0" y1="38.2" x2="100" y2="38.2" /><line x1="0" y1="61.8" x2="100" y2="61.8" />
        </g>
        <g v-if="shown.diagonal" class="l-diag"><line x1="0" y1="0" x2="100" y2="100" /><line x1="100" y1="0" x2="0" y2="100" /></g>
        <g v-if="shown.center" class="l-center"><line x1="50" y1="0" x2="50" y2="100" /><line x1="0" y1="50" x2="100" y2="50" /></g>
      </svg>
      <button
        v-for="(n, i) in noteList"
        :key="i"
        class="art-dot"
        :class="{ on: active === i }"
        :style="{ left: `${n.x}%`, top: `${n.y}%` }"
        :aria-label="`注释 ${i + 1}`"
        @click="active = active === i ? null : i"
      >
        {{ i + 1 }}
      </button>
    </div>

    <p v-if="tip" class="art-tip">{{ tip }}</p>

    <div v-if="palette" class="art-palette">
      <button v-for="c in palette" :key="c.hex" class="art-swatch" :title="c.hsb" @click="copy(c.hex)">
        <span class="sw" :style="{ background: c.hex, flexGrow: Math.max(c.share * 100, 8) }"></span>
        <span class="hex">{{ copied === c.hex ? '已复制' : c.hex }}</span>
        <span class="hsb">{{ c.hsb }}</span>
      </button>
    </div>

    <ol v-if="noteList.length" class="art-notes">
      <li v-for="(n, i) in noteList" :key="i" :class="{ on: active === i }" @click="active = active === i ? null : i">{{ n.t }}</li>
    </ol>

    <figcaption>
      <span v-if="caption" class="cap">{{ caption }}</span>
      <span class="credit">
        {{ work.artist }}《{{ work.title }}》，{{ work.date }}。<a :href="work.link" target="_blank" rel="noopener">{{ work.credit }}</a>
      </span>
    </figcaption>
  </figure>
</template>
