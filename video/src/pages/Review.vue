<template>
  <div class="mx-auto max-w-5xl px-4 py-10 sm:px-6">
    <h1 class="text-2xl font-bold">案例审核</h1>
    <p v-if="!session.me?.admin" class="card mt-6 p-8 text-center text-ink-500">只有管理员可以审核案例</p>
    <template v-else>
      <p class="mt-1 text-sm text-ink-500">用户提交的作品审核通过后出现在案例库；可以设为精选。</p>
      <p v-if="loaded && !items.length" class="card mt-6 p-8 text-center text-ink-500">没有待审核的作品</p>
      <div class="mt-6 space-y-3">
        <div v-for="w in items" :key="w.id" class="card flex flex-wrap items-center gap-4 p-4">
          <div class="min-w-0 flex-1">
            <a :href="`/w/${w.id}`" target="_blank" class="font-medium hover:text-brand-600">{{ w.title }}</a>
            <p class="mt-1 line-clamp-2 text-sm text-ink-500">{{ w.prompt }}</p>
            <p class="mt-1 text-xs text-ink-400">{{ w.author }} · {{ w.mode === 'film' ? 'HTML 视频' : '动画' }} · {{ Math.round(w.duration) }} 秒</p>
          </div>
          <select v-model="w.category" class="input w-32">
            <option v-for="c in cats" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
          <label class="flex items-center gap-1 text-sm"><input v-model="w.featured" type="checkbox" class="accent-brand-500" />精选</label>
          <button class="btn-ghost btn-sm" @click="preview(w)">预览</button>
          <button class="btn-primary btn-sm" @click="review(w, 'approve')">通过</button>
          <button class="btn-ghost btn-sm text-red-600" @click="review(w, 'reject')">拒绝</button>
        </div>
      </div>
      <div v-if="previewing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6" @click.self="previewing = null">
        <div class="w-full max-w-4xl overflow-hidden rounded-2xl bg-black" :style="{ aspectRatio: `${previewing.width} / ${previewing.height}` }">
          <FilmPlayer :spec="previewing.spec" :audio-url="(sc) => (sc.audio?.file ? `/api/v1/video/projects/${previewing.id}/audio/${sc.audio.file}` : '')" with-key />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import FilmPlayer from '../components/FilmPlayer.vue'
import { api, catalog, session } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const items = ref([])
const cats = ref([])
const loaded = ref(false)
const previewing = ref(null)
async function load() {
  if (!session.me?.admin) return
  try {
    items.value = (await api('/admin/pending')).items
  } catch (err) {
    toastError(err)
  } finally {
    loaded.value = true
  }
}
async function preview(w) {
  // Admins preview the full project through the review endpoint's owner check-free path.
  try {
    previewing.value = await api(`/admin/works/${w.id}`)
  } catch (err) {
    toastError(err)
  }
}
async function review(w, decision) {
  try {
    await api(`/admin/works/${w.id}/review`, { method: 'POST', body: { decision, category: w.category, featured: !!w.featured } })
    toast(decision === 'approve' ? '已通过' : '已拒绝')
    items.value = items.value.filter((x) => x.id !== w.id)
  } catch (err) {
    toastError(err)
  }
}
watch(() => session.me, load)
onMounted(() => {
  catalog().then((c) => (cats.value = c.categories))
  load()
})
</script>
