<template>
  <div class="mx-auto max-w-[1280px] px-4 py-8 sm:px-6">
    <div v-if="!w && !missing" class="flex justify-center py-24"><Loader2 class="h-6 w-6 animate-spin text-ink-400" /></div>
    <div v-else-if="missing" class="card p-12 text-center">
      <p class="text-ink-500">这个案例不存在或已下架</p>
      <RouterLink to="/gallery" class="btn-primary mt-4">去看看其他案例</RouterLink>
    </div>
    <template v-else>
      <RouterLink to="/gallery" class="text-sm text-ink-500 hover:text-brand-600">← 案例库</RouterLink>
      <div class="mt-3 grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
        <div>
          <div class="mx-auto overflow-hidden rounded-2xl bg-black shadow-soft" :style="{ aspectRatio: `${w.width} / ${w.height}`, maxWidth: w.height > w.width ? '440px' : 'none' }">
            <div v-if="w.mode === 'upload' && w.media" class="relative h-full w-full"><UploadedMedia :media="w.media" /></div>
            <FilmPlayer v-else :spec="w.spec" :audio-url="audioUrl" />
          </div>
          <h1 class="mt-5 text-2xl font-bold">{{ w.title }}</h1>
          <div class="mt-2 flex flex-wrap items-center gap-4 text-sm text-ink-500">
            <span class="flex items-center gap-1"><Eye class="h-4 w-4" />{{ w.views }} 次观看</span>
            <span>{{ modeLabel(w) }}<template v-if="w.duration"> · {{ Math.round(w.duration) }} 秒</template></span>
            <span>作者 {{ w.author }}</span>
          </div>
          <div v-if="w.prompt" class="card mt-5 p-5">
            <h2 class="text-sm font-semibold">{{ w.mode === 'upload' ? '作品介绍' : '提示词' }}</h2>
            <p class="mt-2 whitespace-pre-wrap text-sm leading-7 text-ink-600 dark:text-ink-300">{{ w.prompt }}</p>
          </div>
        </div>
        <aside class="space-y-4">
          <div class="card p-5">
            <template v-if="w.mode === 'upload'">
              <p class="text-sm leading-6 text-ink-600 dark:text-ink-300">这是作者上传的作品。想做一个类似的？用一句话描述，AI 帮你生成。</p>
              <RouterLink to="/" class="btn-primary mt-3 w-full py-3"><Sparkles class="h-4 w-4" />开始创作</RouterLink>
              <button class="btn-ghost btn-sm mt-3 w-full" @click="copyLink"><Link2 class="h-3.5 w-3.5" />复制链接</button>
            </template>
            <template v-else>
              <RouterLink :to="{ path: '/', query: { remix: w.id } }" class="btn-primary w-full py-3"><Copy class="h-4 w-4" />制作同款</RouterLink>
              <div class="mt-3 grid grid-cols-2 gap-2">
                <button class="btn-ghost btn-sm" @click="copyLink"><Link2 class="h-3.5 w-3.5" />复制链接</button>
                <button class="btn-ghost btn-sm" :disabled="exporting" @click="download"><Download class="h-3.5 w-3.5" />导出 HTML</button>
              </div>
            </template>
          </div>
          <div>
            <h2 class="text-sm font-semibold">相关推荐</h2>
            <div class="mt-3 grid gap-3">
              <WorkCard v-for="r in related" :key="r.id" :work="r" :categories="cats" />
            </div>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Copy, Download, Eye, Link2, Loader2, Sparkles } from 'lucide-vue-next'
import FilmPlayer from '../components/FilmPlayer.vue'
import UploadedMedia from '../components/UploadedMedia.vue'
import { modeLabel } from '../lib/media'
import WorkCard from '../components/WorkCard.vue'
import { api, catalog } from '../lib/api'
import { exportHtml } from '../lib/exportHtml'
import { toast, toastError } from '../lib/toast'

const route = useRoute()
const w = ref(null)
const missing = ref(false)
const related = ref([])
const cats = ref([])
const exporting = ref(false)
const audioUrl = (sc) => (sc.audio?.file ? `/api/v1/video/works/${w.value.id}/audio/${sc.audio.file}` : '')

async function load() {
  w.value = null
  missing.value = false
  try {
    w.value = await api(`/works/${route.params.id}`, { auth: false })
    document.title = `${w.value.title} · HiveGPT 视频`
    api(`/works/${w.value.id}/view`, { method: 'POST', auth: false, body: {} }).catch(() => {})
    const r = await api(`/gallery?category=${w.value.category || 'all'}&size=6`, { auth: false })
    related.value = r.items.filter((x) => x.id !== w.value.id).slice(0, 4)
  } catch {
    missing.value = !w.value
  }
}
function copyLink() {
  navigator.clipboard?.writeText(location.href)
  toast('链接已复制')
}
async function download() {
  exporting.value = true
  try {
    await exportHtml(w.value, audioUrl, false)
  } catch (err) {
    toastError(err)
  } finally {
    exporting.value = false
  }
}
watch(() => route.params.id, load)
onMounted(() => {
  catalog().then((c) => (cats.value = c.categories))
  load()
})
</script>
