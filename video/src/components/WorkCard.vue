<template>
  <article ref="el" class="group overflow-hidden rounded-2xl border border-ink-200/80 bg-white transition hover:-translate-y-0.5 hover:shadow-soft dark:border-ink-800 dark:bg-ink-900" @mouseenter="onEnter" @mouseleave="active = false">
    <RouterLink :to="`/w/${work.id}`" class="relative block bg-ink-900" :style="{ aspectRatio: ratio }">
      <UploadedMedia v-if="work.mode === 'upload' && work.media" card :media="work.media" :active="active" />
      <FilmPlayer v-else-if="visible" card :src="playSrc" :active="active" @ready="release()" />
      <div v-else class="absolute inset-0 flex items-center justify-center text-ink-500"><Clapperboard class="h-8 w-8 opacity-40" /></div>
      <span class="absolute left-2 top-2 rounded-md bg-black/55 px-1.5 py-0.5 text-[11px] text-white">{{ work.mode === 'motion' ? categoryName : modeLabel(work) }}</span>
      <span v-if="work.duration" class="absolute bottom-2 right-2 rounded-md bg-black/55 px-1.5 py-0.5 text-[11px] tabular-nums text-white">{{ fmt(work.duration) }}</span>
      <span v-if="active" class="absolute bottom-2 left-2 rounded-md bg-black/55 px-1.5 py-0.5 text-[11px] text-white">移开即停止</span>
    </RouterLink>
    <div class="p-3">
      <RouterLink :to="`/w/${work.id}`" class="line-clamp-2 min-h-[2.5rem] text-sm font-medium leading-5 hover:text-brand-600">{{ work.title || work.prompt }}</RouterLink>
      <div class="mt-2 flex items-center gap-3 text-xs text-ink-500">
        <span class="flex items-center gap-1"><Eye class="h-3.5 w-3.5" />{{ work.views }}</span>
        <span class="truncate">{{ work.author }}</span>
        <span class="flex-1"></span>
        <RouterLink v-if="work.mode !== 'upload'" :to="{ path: '/', query: { remix: work.id } }" class="rounded-lg bg-brand-50 px-2 py-1 font-medium text-brand-700 opacity-0 transition group-hover:opacity-100 dark:bg-brand-900/30 dark:text-brand-300">制作同款</RouterLink>
      </div>
    </div>
  </article>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Clapperboard, Eye } from 'lucide-vue-next'
import FilmPlayer from './FilmPlayer.vue'
import UploadedMedia from './UploadedMedia.vue'
import { enqueueCard } from '../lib/cardQueue'
import { modeLabel } from '../lib/media'

const props = defineProps({ work: { type: Object, required: true }, categories: { type: Array, default: () => [] } })
const active = ref(false)
const visible = ref(false)
const el = ref(null)
// The work's own player page (scene data inlined, cached); the version busts the cache after edits.
const playSrc = computed(() => `/play/${props.work.id}?u=${Math.floor(Date.parse(props.work.updated_at || 0) / 1000) || 0}`)
// Every card is 16:9 so the grid lines up; the player letterboxes other ratios.
const ratio = computed(() => '16 / 9')
const categoryName = computed(() => props.categories.find((c) => c.id === props.work.category)?.name || '动画')

// Cards start their player only when scrolled into view, through a queue (see cardQueue.js).
let observer = null
let cancel = null
let release = () => {}
onMounted(() => {
  if (props.work.mode === 'upload') return // the card has its media links
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        observer.disconnect()
        load()
      }
    },
    { rootMargin: '200px' }
  )
  if (el.value) observer.observe(el.value)
  else load()
})
onBeforeUnmount(() => {
  observer?.disconnect()
  cancel?.()
  release()
})

function load() {
  cancel = enqueueCard((done) => {
    release = done
    visible.value = true
  })
}
function onEnter() {
  active.value = true
}
const fmt = (s) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, '0')}`
</script>
