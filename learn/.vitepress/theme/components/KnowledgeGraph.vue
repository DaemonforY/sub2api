<script setup lang="ts">
// <KnowledgeGraph topic="flink" />: the knowledge graph of a big-data track, from .vitepress/kg/<topic>.json.
// Three views of the same data: 学习路线 (layers in study order, prerequisite edges), 概念关系 (a grid of
// group × layer with every kind of relation) and 清单 (a plain list, the default on phones). Click a concept for
// its key points, its relations and the articles that explain it; #node=<id> links to a concept.
// Data rules are checked here and by scripts/check_kg.py.
import { computed, onMounted, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { withBase } from 'vitepress'
import { loadProgress, progress } from '../api'
import { lessonByPath } from '../tracks'

interface KLesson { title: string; link: string }
interface KNode {
  id: string; label: string; layer: string; group: string; summary: string; lessons: KLesson[]
  points?: string[]; code?: string[]; conf?: string[]; lab?: string; core?: boolean
}
interface KEdge { from: string; to: string; type: string; label?: string }
interface KLayer { id: string; title: string; desc?: string; goal?: string; weeks?: string }
interface KGroup { id: string; title: string; color?: string }
interface KData { topic: string; title: string; intro?: string; version?: string; groups: KGroup[]; layers: KLayer[]; nodes: KNode[]; edges: KEdge[] }

const props = defineProps<{ topic: string }>()

const files = import.meta.glob<{ default: KData }>('../../kg/*.json')
const data = ref<KData | null>(null)
const loadError = ref('')

const EDGE_TYPES: Record<string, { name: string; color: string; dash?: string; arrow: boolean; out: string; in: string }> = {
  prereq: { name: '前置', color: '#64748b', arrow: true, out: '学完它可以接着学', in: '先学' },
  part: { name: '组成', color: '#16a34a', arrow: true, out: '它是……的一部分', in: '由这些组成' },
  causes: { name: '影响', color: '#ea580c', arrow: true, out: '它会影响', in: '受这些影响' },
  contrast: { name: '对比', color: '#9333ea', dash: '6 4', arrow: false, out: '常和它对比', in: '常和它对比' },
  implements: { name: '实现', color: '#0891b2', dash: '2 3', arrow: true, out: '它实现了', in: '由这些实现' },
}
const PALETTE = ['#2563eb', '#d97706', '#16a34a', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#ca8a04', '#475569', '#0d9488', '#9333ea', '#ea580c', '#65a30d', '#be123c', '#4f46e5', '#0369a1']

const view = ref<'route' | 'graph' | 'list'>('route')
const selected = ref<string>('')
const query = ref('')
const coreOnly = ref(false)
const zoom = ref(1)
const canvas = ref<HTMLElement | null>(null)
const autoFit = ref(true)
const shown = reactive<Record<string, boolean>>({ prereq: true, part: true, causes: true, contrast: true, implements: true })

onMounted(async () => {
  if (window.innerWidth < 760) view.value = 'list'
  void loadProgress()
  const key = `../../kg/${props.topic}.json`
  try {
    if (!files[key]) throw new Error(`找不到 .vitepress/kg/${props.topic}.json`)
    data.value = (await files[key]()).default
  } catch (e) {
    loadError.value = String((e as Error).message || e)
  }
  readHash()
  window.addEventListener('hashchange', readHash)
})
onBeforeUnmount(() => {
  window.removeEventListener('hashchange', readHash)
  ro?.disconnect()
})

// Fit the drawing to the container width until the reader zooms by hand.
let ro: ResizeObserver | null = null
function fitZoom() {
  if (!autoFit.value || !canvas.value || !layout.value.w) return
  zoom.value = Math.max(0.5, Math.min(1, +((canvas.value.clientWidth - 4) / layout.value.w).toFixed(2)))
}
watch(canvas, (el) => {
  ro?.disconnect()
  if (el && typeof ResizeObserver !== 'undefined') {
    ro = new ResizeObserver(fitZoom)
    ro.observe(el)
  }
})
function setZoom(z: number) {
  autoFit.value = false
  zoom.value = Math.max(0.5, Math.min(1.6, +z.toFixed(2)))
}

function readHash() {
  const m = window.location.hash.match(/node=([\w-]+)/)
  if (m && nodeMap.value[m[1]]) selected.value = m[1]
}
watch(selected, (id) => {
  if (typeof window === 'undefined') return
  const url = new URL(window.location.href)
  url.hash = id ? `node=${id}` : ''
  history.replaceState(history.state, '', url.toString())
})

// --- Data and checks ------------------------------------------------------------------------------

const nodeMap = computed<Record<string, KNode>>(() => Object.fromEntries((data.value?.nodes || []).map((n) => [n.id, n])))
const layerIndex = computed<Record<string, number>>(() => Object.fromEntries((data.value?.layers || []).map((l, i) => [l.id, i])))
const groupIndex = computed<Record<string, number>>(() => Object.fromEntries((data.value?.groups || []).map((g, i) => [g.id, i])))
const groupColor = (gid: string) => {
  const g = data.value?.groups.find((x) => x.id === gid)
  return g?.color || PALETTE[(groupIndex.value[gid] ?? 0) % PALETTE.length]
}
const groupTitle = (gid: string) => data.value?.groups.find((x) => x.id === gid)?.title || gid
const layerOf = (lid: string) => data.value?.layers.find((x) => x.id === lid)

const problems = computed<string[]>(() => {
  const d = data.value
  if (!d) return []
  const out: string[] = []
  const seen = new Set<string>()
  for (const n of d.nodes) {
    if (seen.has(n.id)) out.push(`节点 id 重复：${n.id}`)
    seen.add(n.id)
    if (layerIndex.value[n.layer] === undefined) out.push(`${n.id}：阶段 ${n.layer} 不存在`)
    if (groupIndex.value[n.group] === undefined) out.push(`${n.id}：分组 ${n.group} 不存在`)
    if (!n.lessons?.length) out.push(`${n.id}：没有对应的文章`)
  }
  for (const e of d.edges) {
    if (!nodeMap.value[e.from] || !nodeMap.value[e.to]) out.push(`关系 ${e.from} → ${e.to}：端点不存在`)
    else if (!EDGE_TYPES[e.type]) out.push(`关系 ${e.from} → ${e.to}：未知类型 ${e.type}`)
    else if (e.type === 'prereq' && layerIndex.value[nodeMap.value[e.from].layer] > layerIndex.value[nodeMap.value[e.to].layer])
      out.push(`前置关系 ${e.from} → ${e.to} 从后面的阶段指向前面的阶段`)
  }
  if (order.value.length < d.nodes.length) out.push('前置关系里有环，学习顺序排不出来')
  return out
})
const edges = computed(() => (data.value?.edges || []).filter((e) => nodeMap.value[e.from] && nodeMap.value[e.to] && EDGE_TYPES[e.type]))

// Study order: layer by layer; inside a layer a topological order of the prerequisites, ties by group then file order.
const order = computed<string[]>(() => {
  const d = data.value
  if (!d) return []
  const pos = Object.fromEntries(d.nodes.map((n, i) => [n.id, i]))
  const out: string[] = []
  for (const layer of d.layers) {
    const ids = d.nodes.filter((n) => n.layer === layer.id).map((n) => n.id)
    const inLayer = new Set(ids)
    const indeg: Record<string, number> = Object.fromEntries(ids.map((i) => [i, 0]))
    for (const e of edges.value) if (e.type === 'prereq' && inLayer.has(e.from) && inLayer.has(e.to)) indeg[e.to]++
    const key = (i: string) => (groupIndex.value[nodeMap.value[i].group] ?? 0) * 10000 + pos[i]
    const ready = ids.filter((i) => indeg[i] === 0).sort((a, b) => key(a) - key(b))
    while (ready.length) {
      const u = ready.shift()!
      out.push(u)
      for (const e of edges.value)
        if (e.type === 'prereq' && e.from === u && inLayer.has(e.to) && --indeg[e.to] === 0) {
          ready.push(e.to)
          ready.sort((a, b) => key(a) - key(b))
        }
    }
  }
  return out
})
const seq = computed<Record<string, number>>(() => Object.fromEntries(order.value.map((id, i) => [id, i + 1])))

// --- Progress: a concept counts as learned when one of its course lessons is marked done ----------

const learned = (n: KNode) => n.lessons.some((l) => {
  const id = lessonByPath(l.link.split('#')[0])
  return !!(id && progress.completed[id])
})
const learnedCount = computed(() => (data.value?.nodes || []).filter(learned).length)

// --- Search, selection and neighbourhood ----------------------------------------------------------

const matches = computed<Set<string>>(() => {
  const q = query.value.trim().toLowerCase()
  if (!q || !data.value) return new Set()
  return new Set(data.value.nodes.filter((n) =>
    [n.label, n.summary, ...(n.code || []), ...(n.conf || []), ...(n.points || [])].join(' ').toLowerCase().includes(q)).map((n) => n.id))
})
const visibleNode = (n: KNode) => !coreOnly.value || n.core || n.id === selected.value
const sel = computed(() => (selected.value ? nodeMap.value[selected.value] : null))

const relations = computed(() => {
  const id = selected.value
  if (!id) return []
  const groups: { title: string; type: string; items: { id: string; label?: string }[] }[] = []
  const add = (title: string, type: string, items: { id: string; label?: string }[]) => items.length && groups.push({ title, type, items })
  const of = (t: string, dir: 'out' | 'in') => edges.value.filter((e) => e.type === t && (dir === 'out' ? e.from === id : e.to === id))
    .map((e) => ({ id: dir === 'out' ? e.to : e.from, label: e.label }))
  add('先学这些', 'prereq', of('prereq', 'in'))
  add('学完可以接着学', 'prereq', of('prereq', 'out'))
  add('由这些组成', 'part', of('part', 'in'))
  add('它是这些的一部分', 'part', of('part', 'out'))
  add('它会影响', 'causes', of('causes', 'out'))
  add('受这些影响', 'causes', of('causes', 'in'))
  add('它实现了', 'implements', of('implements', 'out'))
  add('由这些实现', 'implements', of('implements', 'in'))
  add('常和它对比', 'contrast', [...of('contrast', 'out'), ...of('contrast', 'in')])
  return groups
})

// Every prerequisite before the selected concept, in study order: "学到这里要先学什么".
const pathTo = computed<string[]>(() => {
  if (!selected.value) return []
  const need = new Set<string>()
  const stack = [selected.value]
  while (stack.length) {
    const u = stack.pop()!
    for (const e of edges.value) if (e.type === 'prereq' && e.to === u && !need.has(e.from)) { need.add(e.from); stack.push(e.from) }
  }
  return order.value.filter((i) => need.has(i))
})
const showPath = ref(false)
watch(selected, () => (showPath.value = false))

const lit = computed<Set<string>>(() => {
  const s = new Set<string>()
  if (!selected.value) return s
  s.add(selected.value)
  if (showPath.value) pathTo.value.forEach((i) => s.add(i))
  else for (const e of edges.value) if (shown[e.type] && (e.from === selected.value || e.to === selected.value)) { s.add(e.from); s.add(e.to) }
  return s
})
const edgeLit = (e: KEdge) => {
  if (!selected.value) return false
  if (showPath.value) return e.type === 'prereq' && lit.value.has(e.from) && lit.value.has(e.to)
  return e.from === selected.value || e.to === selected.value
}
function pick(id: string) {
  selected.value = selected.value === id ? '' : id
}
function jump(id: string) {
  selected.value = id
  if (view.value !== 'list') {
    const el = document.querySelector(`[data-kg-node="${id}"]`)
    el?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' })
  }
}

// --- Layout ---------------------------------------------------------------------------------------

const NW = 172, NH = 44, GX = 18, GY = 18
const LABEL_W = 210

interface Box { x: number; y: number }
interface Layout { w: number; h: number; pos: Record<string, Box>; bands: { y: number; h: number; layer: KLayer }[]; cards?: { x: number; y: number; h: number; group: KGroup }[] }

// 学习路线: one band per layer, concepts in study order, wrapping after `perRow`.
const routeLayout = computed<Layout>(() => {
  const d = data.value
  const pos: Record<string, Box> = {}
  const bands: Layout['bands'] = []
  if (!d) return { w: 0, h: 0, pos, bands }
  const perRow = 6
  let y = 16
  for (const layer of d.layers) {
    const ids = order.value.filter((i) => nodeMap.value[i].layer === layer.id && visibleNode(nodeMap.value[i]))
    const rows = Math.max(1, Math.ceil(ids.length / perRow))
    const h = rows * (NH + GY) + GY + 14
    ids.forEach((id, k) => {
      pos[id] = { x: LABEL_W + 20 + (k % perRow) * (NW + GX), y: y + GY + 8 + Math.floor(k / perRow) * (NH + GY) }
    })
    bands.push({ y, h, layer })
    y += h + 10
  }
  return { w: LABEL_W + 20 + perRow * (NW + GX) + 10, h: y + 10, pos, bands }
})

// 概念关系: a module map. Each group is a card with its concepts stacked in study order; cards wrap 6 to a row.
const graphLayout = computed<Layout>(() => {
  const d = data.value
  const pos: Record<string, Box> = {}
  const bands: Layout['bands'] = []
  const cards: { x: number; y: number; h: number; group: KGroup }[] = []
  if (!d) return { w: 0, h: 0, pos, bands, cards }
  const perRow = 6, CW = NW + 24, GAPX = 22, top = 10
  let y = top
  for (let r = 0; r < d.groups.length; r += perRow) {
    const row = d.groups.slice(r, r + perRow)
    let tallest = 0
    row.forEach((g, k) => {
      const x = 10 + k * (CW + GAPX)
      const ids = order.value.filter((i) => nodeMap.value[i].group === g.id && visibleNode(nodeMap.value[i]))
      ids.forEach((id, n) => (pos[id] = { x: x + 12, y: y + 40 + n * (NH + 12) }))
      tallest = Math.max(tallest, ids.length)
      cards.push({ x, y, h: 0, group: g })
    })
    const h = 40 + Math.max(1, tallest) * (NH + 12) + 4
    cards.slice(-row.length).forEach((c) => (c.h = h))
    y += h + 26
  }
  return { w: 10 + perRow * (CW + GAPX), h: y, pos, bands, cards }
})

const layout = computed(() => (view.value === 'graph' ? graphLayout.value : routeLayout.value))
watch(() => [layout.value.w, view.value], () => setTimeout(fitZoom))

function edgePath(e: KEdge, L: Layout) {
  const a = L.pos[e.from], b = L.pos[e.to]
  if (!a || !b) return ''
  const ax = a.x + NW / 2, bx = b.x + NW / 2
  if (Math.abs(a.y - b.y) < 2) {
    // same row: arc over the top
    const y = a.y, dx = Math.abs(bx - ax)
    const x1 = a.x + (bx > ax ? NW : 0), x2 = b.x + (bx > ax ? 0 : NW)
    const lift = Math.min(16, 8 + dx * 0.02)
    return `M${ax},${y} C${ax},${y - lift} ${bx},${y - lift} ${bx},${y}`
  }
  const down = b.y > a.y
  const y1 = down ? a.y + NH : a.y, y2 = down ? b.y : b.y + NH
  const c = Math.max(30, Math.abs(y2 - y1) * 0.45)
  return `M${ax},${y1} C${ax},${y1 + (down ? c : -c)} ${bx},${y2 + (down ? -c : c)} ${bx},${y2}`
}
function edgeMid(e: KEdge, L: Layout) {
  const a = L.pos[e.from], b = L.pos[e.to]
  if (!a || !b) return { x: 0, y: 0 }
  if (Math.abs(a.y - b.y) < 2) return { x: (a.x + b.x) / 2 + NW / 2, y: a.y - 12 }
  return { x: (a.x + b.x) / 2 + NW / 2, y: (a.y + b.y) / 2 + NH / 2 }
}
const drawnEdges = computed(() => {
  const L = layout.value
  return edges.value.filter((e) => shown[e.type] && L.pos[e.from] && L.pos[e.to] && (view.value === 'graph' || e.type === 'prereq' || edgeLit(e)))
})

const width = (s: string) => [...s].reduce((n, ch) => n + (/[\u0000-ÿ]/.test(ch) ? 0.56 : 1), 0)
const fit = (s: string, max = 11.5) => {
  if (width(s) <= max) return s
  let out = ''
  for (const ch of s) {
    if (width(out + ch) > max - 1) break
    out += ch
  }
  return out + '…'
}

const listByLayer = computed(() => (data.value?.layers || []).map((layer) => ({
  layer,
  nodes: order.value.map((i) => nodeMap.value[i]).filter((n) => n.layer === layer.id && visibleNode(n)),
})))
</script>

<template>
  <div class="kg" data-testid="knowledge-graph">
    <div v-if="loadError" class="kg-error">知识图谱加载失败：{{ loadError }}</div>
    <div v-else-if="!data" class="kg-loading">正在加载知识图谱……</div>
    <template v-else>
      <p v-if="data.intro" class="kg-intro">{{ data.intro }}</p>
      <div class="kg-stats">
        <span v-if="data.version">基于 {{ data.version }}</span>
        <span>{{ data.nodes.length }} 个概念</span>
        <span>{{ edges.length }} 条关系</span>
        <span>{{ data.layers.length }} 个学习阶段</span>
        <span v-if="progress.loaded">已学 {{ learnedCount }} / {{ data.nodes.length }}</span>
      </div>
      <div v-if="problems.length" class="kg-error">
        <b>数据有 {{ problems.length }} 处问题</b>（运行 <code>python3 scripts/check_kg.py {{ topic }}</code> 查看）：
        <ul><li v-for="p in problems.slice(0, 8)" :key="p">{{ p }}</li></ul>
      </div>

      <div class="kg-toolbar">
        <div class="kg-tabs" role="tablist">
          <button :class="{ on: view === 'route' }" role="tab" @click="view = 'route'">学习路线</button>
          <button :class="{ on: view === 'graph' }" role="tab" @click="view = 'graph'">概念关系</button>
          <button :class="{ on: view === 'list' }" role="tab" @click="view = 'list'">清单</button>
        </div>
        <input v-model="query" class="kg-search" type="search" placeholder="搜索概念、类名、配置项" aria-label="搜索概念" />
        <label class="kg-check"><input v-model="coreOnly" type="checkbox" /> 只看重点</label>
        <div v-if="view !== 'list'" class="kg-zoom">
          <button aria-label="缩小" @click="setZoom(zoom - 0.1)">−</button>
          <span>{{ Math.round(zoom * 100) }}%</span>
          <button aria-label="放大" @click="setZoom(zoom + 0.1)">＋</button>
          <button v-if="!autoFit" class="kg-fit" @click="autoFit = true; fitZoom()">适应宽度</button>
        </div>
      </div>
      <div class="kg-legend">
        <span class="kg-legend-title">关系：</span>
        <label v-for="(t, k) in EDGE_TYPES" :key="k" class="kg-chip" :class="{ off: !shown[k] }">
          <input v-model="shown[k]" type="checkbox" />
          <svg width="26" height="10"><line x1="1" y1="5" x2="25" y2="5" :stroke="t.color" stroke-width="2.5" :stroke-dasharray="t.dash" /></svg>
          {{ t.name }}
        </label>
        <span v-if="view === 'route'" class="kg-hint">学习路线默认只画「前置」，点一个概念显示它的全部关系</span>
        <span v-else-if="view === 'graph'" class="kg-hint">按主题分组，组内按学习顺序；S1、S2……是第几个学习阶段。跨分组的连线默认调淡，点一个概念看它的全部关系</span>
      </div>
      <div v-if="query && view !== 'list'" class="kg-hint kg-found">找到 {{ matches.size }} 个：
        <a v-for="id in [...matches].slice(0, 12)" :key="id" href="javascript:void 0" @click="jump(id)">{{ nodeMap[id].label }}</a>
      </div>

      <div class="kg-body" :class="{ 'has-panel': !!sel }">
        <!-- 清单 -->
        <div v-if="view === 'list'" class="kg-list">
          <section v-for="b in listByLayer" :key="b.layer.id" class="kg-list-layer">
            <h3>{{ b.layer.title }} <small v-if="b.layer.weeks">{{ b.layer.weeks }}</small></h3>
            <p v-if="b.layer.desc" class="kg-muted">{{ b.layer.desc }}</p>
            <p v-if="b.layer.goal" class="kg-goal">学完这一阶段：{{ b.layer.goal }}</p>
            <ol>
              <li v-for="n in b.nodes" :key="n.id" :value="seq[n.id]" :class="{ hit: matches.has(n.id), on: selected === n.id }">
                <a href="javascript:void 0" class="kg-list-name" @click="pick(n.id)">
                  <i :style="{ background: groupColor(n.group) }"></i>{{ n.label }}<b v-if="n.core" class="kg-core">重点</b><span v-if="learned(n)" class="kg-done">✓ 已学</span>
                </a>
                <span class="kg-muted"> {{ n.summary }}</span>
              </li>
            </ol>
          </section>
        </div>

        <!-- 学习路线 / 概念关系 -->
        <div v-else ref="canvas" class="kg-canvas">
          <svg :width="layout.w * zoom" :height="layout.h * zoom" :viewBox="`0 0 ${layout.w} ${layout.h}`" class="kg-svg" @click.self="selected = ''">
            <defs>
              <marker v-for="(t, k) in EDGE_TYPES" :id="`kg-arrow-${topic}-${k}`" :key="k" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                <path d="M0,0 L10,5 L0,10 z" :fill="t.color" />
              </marker>
            </defs>
            <!-- bands -->
            <g v-for="(b, i) in layout.bands" :key="b.layer.id">
              <rect :x="4" :y="b.y" :width="layout.w - 8" :height="b.h" rx="14" :class="['kg-band', i % 2 ? 'odd' : '']" />
              <template v-if="view === 'route'">
                <text :x="20" :y="b.y + 28" class="kg-band-title">{{ fit(b.layer.title, 14) }}</text>
                <text v-if="b.layer.weeks" :x="20" :y="b.y + 48" class="kg-band-sub">{{ b.layer.weeks }}</text>
                <text v-if="b.layer.goal" :x="20" :y="b.y + 68" class="kg-band-sub"><title>{{ b.layer.goal }}</title>验收：{{ fit(b.layer.goal, 13) }}</text>
              </template>
              <text v-else :x="16" :y="b.y + 24" class="kg-band-sub kg-band-left"><title>{{ b.layer.title }}</title>{{ fit(b.layer.title, 8.5) }}</text>
            </g>
            <g v-if="view === 'graph'">
              <g v-for="c in layout.cards" :key="c.group.id">
                <rect :x="c.x" :y="c.y" :width="NW + 24" :height="c.h" rx="14" class="kg-card" :stroke="groupColor(c.group.id)" />
                <rect :x="c.x" :y="c.y" :width="NW + 24" height="30" rx="14" :fill="groupColor(c.group.id)" opacity="0.14" />
                <text :x="c.x + 14" :y="c.y + 21" class="kg-col-title" :fill="groupColor(c.group.id)">{{ fit(c.group.title, 13) }}</text>
              </g>
            </g>
            <!-- edges -->
            <g class="kg-edges">
              <path v-for="(e, i) in drawnEdges" :key="i" :d="edgePath(e, layout)" fill="none"
                :stroke="EDGE_TYPES[e.type].color" :stroke-dasharray="EDGE_TYPES[e.type].dash"
                :marker-end="EDGE_TYPES[e.type].arrow ? `url(#kg-arrow-${topic}-${e.type})` : undefined"
                :class="['kg-edge', view, { lit: edgeLit(e), dim: !!selected && !edgeLit(e), cross: view === 'graph' && nodeMap[e.from].group !== nodeMap[e.to].group }]">
                <title>{{ nodeMap[e.from].label }} —{{ e.label || EDGE_TYPES[e.type].name }}→ {{ nodeMap[e.to].label }}</title>
              </path>
            </g>
            <!-- labels on the selected node's edges -->
            <g v-if="selected" class="kg-edge-labels">
              <template v-for="(e, i) in drawnEdges" :key="'l' + i">
                <text v-if="edgeLit(e) && e.label" :x="edgeMid(e, layout).x" :y="edgeMid(e, layout).y" text-anchor="middle" class="kg-edge-label">{{ fit(e.label, 16) }}</text>
              </template>
            </g>
            <!-- nodes -->
            <g v-for="(p, id) in layout.pos" :key="id" :transform="`translate(${p.x},${p.y})`" :data-kg-node="id"
              :class="['kg-node', { on: selected === id, dim: !!selected && !lit.has(id as string), hit: matches.has(id as string), core: nodeMap[id].core, done: learned(nodeMap[id]) }]"
              tabindex="0" role="button" :aria-label="nodeMap[id].label" @click.stop="pick(id as string)" @keydown.enter="pick(id as string)">
              <title>{{ nodeMap[id].label }}：{{ nodeMap[id].summary }}</title>
              <rect :width="NW" :height="NH" rx="10" class="kg-node-box" :stroke="groupColor(nodeMap[id].group)" />
              <rect width="6" :height="NH" rx="3" :fill="groupColor(nodeMap[id].group)" />
              <text x="16" :y="NH / 2 + 4" class="kg-node-seq"><title>{{ view === 'route' ? '学习顺序' : '学习阶段：' + layerOf(nodeMap[id].layer)?.title }}</title>{{ view === 'route' ? seq[id] : 'S' + ((layerIndex[nodeMap[id].layer] ?? 0) + 1) }}</text>
              <text x="38" :y="NH / 2 + 5" class="kg-node-label">{{ fit(nodeMap[id].label, 10.5) }}</text>
              <text v-if="learned(nodeMap[id])" :x="NW - 16" :y="NH / 2 + 5" class="kg-node-done">✓</text>
            </g>
          </svg>
        </div>

        <!-- detail -->
        <aside v-if="sel" class="kg-panel" aria-live="polite">
          <button class="kg-close" aria-label="关闭" @click="selected = ''">×</button>
          <div class="kg-panel-tags">
            <span :style="{ background: groupColor(sel.group) }">{{ groupTitle(sel.group) }}</span>
            <span class="soft">{{ layerOf(sel.layer)?.title }}</span>
            <span class="soft">第 {{ seq[sel.id] }} 个</span>
            <span v-if="learned(sel)" class="soft ok">✓ 已学</span>
          </div>
          <h3>{{ sel.label }}</h3>
          <p>{{ sel.summary }}</p>
          <ul v-if="sel.points?.length" class="kg-points"><li v-for="p in sel.points" :key="p">{{ p }}</li></ul>
          <div v-if="sel.code?.length" class="kg-kv"><b>源码</b><code v-for="c in sel.code" :key="c">{{ c }}</code></div>
          <div v-if="sel.conf?.length" class="kg-kv"><b>配置</b><code v-for="c in sel.conf" :key="c">{{ c }}</code></div>
          <p v-if="sel.lab" class="kg-lab"><b>实测</b>{{ sel.lab }}</p>
          <div class="kg-lessons">
            <b>读这些文章</b>
            <a v-for="l in sel.lessons" :key="l.link" :href="withBase(l.link)">{{ l.title }} →</a>
          </div>
          <div v-for="g in relations" :key="g.title" class="kg-rel">
            <b><i :style="{ background: EDGE_TYPES[g.type].color }"></i>{{ g.title }}</b>
            <a v-for="r in g.items" :key="r.id" href="javascript:void 0" @click="jump(r.id)">
              {{ nodeMap[r.id].label }}<small v-if="r.label">（{{ r.label }}）</small>
            </a>
          </div>
          <div v-if="pathTo.length" class="kg-rel">
            <b>学到这里，一共要先学 {{ pathTo.length }} 个概念</b>
            <button v-if="view !== 'list'" class="kg-link-btn" @click="showPath = !showPath">{{ showPath ? '取消高亮' : '在图上高亮这条路径' }}</button>
            <ol class="kg-path"><li v-for="id in pathTo" :key="id"><a href="javascript:void 0" @click="jump(id)">{{ nodeMap[id].label }}</a><span v-if="learned(nodeMap[id])" class="kg-done">✓</span></li></ol>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.kg { margin: 16px 0 24px; }
.kg-intro { color: var(--vp-c-text-2); }
.kg-stats { display: flex; flex-wrap: wrap; gap: 8px; margin: 8px 0 14px; font-size: 13px; }
.kg-stats span { padding: 3px 10px; border-radius: 999px; background: var(--vp-c-bg-soft); color: var(--vp-c-text-2); }
.kg-error { border: 1px solid #fca5a5; background: rgba(239, 68, 68, 0.08); border-radius: 12px; padding: 10px 14px; font-size: 14px; margin: 10px 0; }
.kg-loading { color: var(--vp-c-text-2); padding: 30px 0; }
.kg-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.kg-tabs { display: inline-flex; border: 1px solid var(--vp-c-divider); border-radius: 10px; overflow: hidden; }
.kg-tabs button { padding: 6px 14px; font-size: 14px; color: var(--vp-c-text-2); }
.kg-tabs button.on { background: var(--vp-c-brand-1); color: #fff; font-weight: 600; }
.kg-search { flex: 1; min-width: 180px; max-width: 300px; border: 1px solid var(--vp-c-divider); border-radius: 10px; padding: 6px 12px; font-size: 14px; background: var(--vp-c-bg); }
.kg-check { font-size: 14px; color: var(--vp-c-text-2); display: inline-flex; align-items: center; gap: 4px; cursor: pointer; }
.kg-zoom { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: var(--vp-c-text-2); }
.kg-zoom button { width: 28px; height: 28px; border-radius: 8px; border: 1px solid var(--vp-c-divider); }
.kg-zoom .kg-fit { width: auto; padding: 0 8px; font-size: 12.5px; }
.kg-legend { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; margin: 10px 0; font-size: 13px; }
.kg-legend-title { color: var(--vp-c-text-2); }
.kg-chip { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border: 1px solid var(--vp-c-divider); border-radius: 999px; cursor: pointer; user-select: none; }
.kg-chip input { display: none; }
.kg-chip.off { opacity: 0.4; }
.kg-hint { font-size: 12.5px; color: var(--vp-c-text-3); }
.kg-found { margin: -2px 0 8px; }
.kg-found a { margin-right: 8px; }
.kg-body { display: grid; grid-template-columns: minmax(0, 1fr); gap: 14px; align-items: start; }
.kg-canvas { overflow: auto; max-height: 78vh; border: 1px solid var(--vp-c-divider); border-radius: 16px; background: var(--vp-c-bg); }
.kg-svg { display: block; font-family: var(--vp-font-family-base); }
.kg-band { fill: var(--vp-c-bg-soft); opacity: 0.55; }
.kg-band.odd { opacity: 0.9; }
.kg-band-title { font-size: 15px; font-weight: 700; fill: var(--vp-c-text-1); }
.kg-band-sub { font-size: 12px; fill: var(--vp-c-text-2); }
.kg-col-title { font-size: 14px; font-weight: 700; }
.kg-card { fill: var(--vp-c-bg-soft); fill-opacity: 0.6; stroke-width: 1.2; stroke-opacity: 0.45; }
.kg-edge { stroke-width: 1.6; opacity: 0.55; transition: opacity 0.15s; }
.kg-edge.route { opacity: 0.32; }
.kg-edge.cross { opacity: 0.13; }
.kg-edge.lit { stroke-width: 2.6; opacity: 1; }
.kg-edge.dim { opacity: 0.08; }
.kg-edge-label { font-size: 11.5px; fill: var(--vp-c-text-1); paint-order: stroke; stroke: var(--vp-c-bg); stroke-width: 4px; }
.kg-node { cursor: pointer; transition: opacity 0.15s; }
.kg-node.dim { opacity: 0.22; }
.kg-node-box { fill: var(--vp-c-bg); stroke-width: 1.5; }
.kg-node.core .kg-node-box { stroke-width: 2.6; }
.kg-node.on .kg-node-box { fill: var(--vp-c-brand-soft); stroke-width: 3; }
.kg-node.hit .kg-node-box { stroke: #f59e0b !important; stroke-width: 3.2; }
.kg-node.done .kg-node-box { fill: rgba(22, 163, 74, 0.07); }
.kg-node:focus { outline: none; }
.kg-node:focus .kg-node-box { stroke-width: 3; }
.kg-node-seq { font-size: 11px; fill: var(--vp-c-text-3); font-weight: 600; }
.kg-node-label { font-size: 13.5px; fill: var(--vp-c-text-1); }
.kg-node.core .kg-node-label { font-weight: 700; }
.kg-node-done { font-size: 13px; fill: #16a34a; font-weight: 700; }
.kg-panel { position: fixed; z-index: 30; right: 16px; top: calc(var(--vp-nav-height) + 12px); width: 360px; max-height: calc(100vh - var(--vp-nav-height) - 24px); overflow: auto; border: 1px solid var(--vp-c-divider); border-radius: 16px; padding: 16px 18px; background: var(--vp-c-bg); font-size: 14px; box-shadow: 0 18px 50px rgba(15, 23, 42, 0.18); }
.kg-panel h3 { margin: 6px 0 6px; font-size: 19px; }
.kg-panel p { margin: 6px 0; line-height: 1.7; }
.kg-close { position: absolute; right: 12px; top: 8px; font-size: 22px; color: var(--vp-c-text-3); }
.kg-panel-tags { display: flex; flex-wrap: wrap; gap: 6px; padding-right: 20px; }
.kg-panel-tags span { font-size: 12px; padding: 2px 8px; border-radius: 999px; color: #fff; }
.kg-panel-tags span.soft { background: var(--vp-c-bg-soft); color: var(--vp-c-text-2); }
.kg-panel-tags span.ok { color: #16a34a; }
.kg-points { margin: 6px 0; padding-left: 18px; }
.kg-points li { line-height: 1.6; margin: 2px 0; }
.kg-kv { display: flex; flex-wrap: wrap; gap: 4px 6px; align-items: baseline; margin: 6px 0; }
.kg-kv b, .kg-lab b { font-size: 12px; color: var(--vp-c-text-2); margin-right: 4px; }
.kg-kv code { font-size: 12px; word-break: break-all; }
.kg-lab { background: rgba(22, 163, 74, 0.08); border-radius: 10px; padding: 6px 10px; }
.kg-lessons, .kg-rel { margin-top: 12px; display: flex; flex-direction: column; gap: 3px; }
.kg-lessons b, .kg-rel b { font-size: 12.5px; color: var(--vp-c-text-2); display: flex; align-items: center; gap: 6px; }
.kg-rel b i { width: 10px; height: 3px; border-radius: 2px; display: inline-block; }
.kg-rel a small { color: var(--vp-c-text-3); }
.kg-link-btn { align-self: flex-start; font-size: 12.5px; color: var(--vp-c-brand-1); }
.kg-path { margin: 4px 0 0; padding-left: 22px; }
.kg-done { color: #16a34a; font-size: 12px; margin-left: 6px; }
.kg-core { font-size: 11px; color: var(--vp-c-brand-1); border: 1px solid var(--vp-c-brand-1); border-radius: 6px; padding: 0 4px; margin-left: 6px; font-weight: 600; }
.kg-list-layer { border: 1px solid var(--vp-c-divider); border-radius: 16px; padding: 12px 18px; margin-bottom: 12px; }
.kg-list-layer h3 { margin: 4px 0; font-size: 17px; }
.kg-list-layer h3 small { font-size: 12px; color: var(--vp-c-text-3); font-weight: 400; margin-left: 6px; }
.kg-muted { color: var(--vp-c-text-2); font-size: 13.5px; }
.kg-goal { font-size: 13px; color: var(--vp-c-brand-3); background: var(--vp-c-brand-soft); border-radius: 8px; padding: 4px 10px; display: inline-block; }
.kg-list ol { padding-left: 26px; margin: 8px 0; }
.kg-list li { margin: 6px 0; line-height: 1.6; }
.kg-list li.hit { background: rgba(245, 158, 11, 0.14); border-radius: 6px; }
.kg-list li.on .kg-list-name { text-decoration: underline; }
.kg-list-name { font-weight: 600; display: inline-flex; align-items: center; }
.kg-list-name i { width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; display: inline-block; }
@media (max-width: 767px) {
  .kg-panel { left: 8px; right: 8px; top: auto; bottom: 8px; width: auto; max-height: 62vh; }
}
</style>

<style>
/* A page holding a knowledge graph uses the full content width (no outline column). */
.VPDoc:has(.kg) .aside { display: none; }
.VPDoc:has(.kg) .content-container,
.VPDoc:has(.kg) .container { max-width: none !important; }
.VPDoc.has-aside:has(.kg) .content-container { max-width: none !important; }
</style>
