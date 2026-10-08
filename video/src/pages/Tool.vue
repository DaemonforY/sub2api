<template>
  <div v-if="tool" class="mx-auto max-w-[1280px] px-4 py-10 sm:px-6">
    <RouterLink to="/tools" class="text-sm text-ink-500 hover:text-brand-600">← 全部工具</RouterLink>
    <section class="mt-4 grid items-center gap-8 lg:grid-cols-2">
      <div>
        <span class="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-50 text-brand-600 dark:bg-brand-900/30"><component :is="TOOL_ICONS[tool.icon]" class="h-6 w-6" /></span>
        <h1 class="mt-4 text-3xl font-bold sm:text-4xl">{{ tool.name }}</h1>
        <p class="mt-2 text-lg text-brand-700 dark:text-brand-300">{{ tool.headline }}</p>
        <p class="mt-4 leading-7 text-ink-600 dark:text-ink-300">{{ tool.summary }}</p>
        <ul class="mt-5 space-y-2">
          <li v-for="pt in tool.points" :key="pt" class="flex items-start gap-2 text-sm"><CircleCheck class="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />{{ pt }}</li>
        </ul>
        <RouterLink :to="{ path: '/', query: { mode: tool.mode, category: tool.category } }" class="btn-primary mt-6 px-6 py-3"><Sparkles class="h-4 w-4" />立即使用</RouterLink>
      </div>
      <div class="card p-5">
        <h2 class="text-sm font-semibold">试试这些描述</h2>
        <div class="mt-3 space-y-2">
          <RouterLink v-for="ex in tool.prompts" :key="ex" :to="{ path: '/', query: { mode: tool.mode, category: tool.category, prompt: ex } }" class="block rounded-xl border border-ink-200 px-4 py-3 text-sm leading-6 transition hover:border-brand-300 hover:bg-brand-50/50 dark:border-ink-700 dark:hover:bg-brand-900/10">
            {{ ex }}
          </RouterLink>
        </div>
      </div>
    </section>
    <section class="mt-14">
      <div class="flex items-end justify-between">
        <h2 class="text-xl font-bold">{{ tool.name }}案例</h2>
        <RouterLink :to="`/gallery/${tool.category}`" class="text-sm text-brand-600 hover:underline">更多 →</RouterLink>
      </div>
      <p v-if="loaded && !works.length" class="card mt-4 p-8 text-center text-sm text-ink-500">还没有公开案例，来做第一个吧</p>
      <div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <WorkCard v-for="w in works" :key="w.id" :work="w" :categories="cats" />
      </div>
    </section>
  </div>
  <Missing v-else />
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CircleCheck, Sparkles } from 'lucide-vue-next'
import WorkCard from '../components/WorkCard.vue'
import Missing from './Missing.vue'
import { api, catalog } from '../lib/api'
import { findTool } from '../lib/tools'
import { TOOL_ICONS } from '../lib/icons'

const route = useRoute()
const tool = computed(() => findTool(String(route.params.slug)))
const works = ref([])
const cats = ref([])
const loaded = ref(false)
async function load() {
  if (!tool.value) return
  document.title = `${tool.value.name} · HiveGPT 视频`
  try {
    works.value = (await api(`/gallery?category=${tool.value.category}&size=8`, { auth: false })).items
  } catch {
    works.value = []
  } finally {
    loaded.value = true
  }
}
watch(tool, load)
onMounted(() => {
  catalog().then((c) => (cats.value = c.categories))
  load()
})
</script>
