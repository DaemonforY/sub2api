<template>
  <div class="mx-auto flex max-w-[1440px] gap-6 px-4 py-8 sm:px-6">
    <aside class="sticky top-20 hidden w-52 shrink-0 self-start lg:block">
      <h2 class="px-3 text-xs font-semibold tracking-wide text-ink-400">分类</h2>
      <nav class="mt-2 space-y-0.5">
        <RouterLink v-for="c in navCats" :key="c.id" :to="c.id === 'featured' ? '/gallery' : `/gallery/${c.id}`" class="block rounded-xl px-3 py-2 text-sm transition hover:bg-white dark:hover:bg-ink-900" :class="current === c.id ? 'bg-white font-medium text-brand-600 shadow-sm dark:bg-ink-900' : 'text-ink-600 dark:text-ink-300'">
          {{ c.name }}
        </RouterLink>
      </nav>
    </aside>
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold">{{ heading }}</h1>
          <p class="mt-1 text-sm text-ink-500">{{ subheading }}</p>
        </div>
        <div class="flex items-center gap-2">
          <div class="relative">
            <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-400" />
            <input v-model="q" class="input w-56 pl-9" placeholder="搜索案例" @keydown.enter="reload" />
          </div>
          <select v-model="sort" class="input w-28" @change="reload">
            <option value="">推荐</option>
            <option value="new">最新</option>
            <option value="hot">最热</option>
          </select>
        </div>
      </div>
      <div class="mt-4 flex gap-2 overflow-x-auto pb-1 lg:hidden">
        <RouterLink v-for="c in navCats" :key="c.id" :to="c.id === 'featured' ? '/gallery' : `/gallery/${c.id}`" class="chip shrink-0" :class="{ 'chip-on': current === c.id }">{{ c.name }}</RouterLink>
      </div>

      <div v-if="loading && !items.length" class="mt-16 flex justify-center"><Loader2 class="h-6 w-6 animate-spin text-ink-400" /></div>
      <div v-else-if="!items.length" class="card mt-8 p-12 text-center">
        <p class="text-ink-500">这里还没有案例</p>
        <RouterLink :to="createLink" class="btn-primary mt-4">去做第一个</RouterLink>
      </div>
      <div v-else class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <template v-for="cell in cells" :key="cell.key">
          <AdCard v-if="cell.ad" :ad="cell.ad" />
          <WorkCard v-else :work="cell.work" :categories="cats" />
        </template>
      </div>
      <div v-if="items.length < total" class="mt-8 flex justify-center">
        <button class="btn-ghost" :disabled="loading" @click="more">加载更多</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Loader2, Search } from 'lucide-vue-next'
import WorkCard from '../components/WorkCard.vue'
import AdCard from '../components/AdCard.vue'
import { api, catalog } from '../lib/api'

const route = useRoute()
const cats = ref([])
const items = ref([])
const total = ref(0)
const page = ref(1)
const q = ref('')
const sort = ref('')
const loading = ref(false)
const ads = ref([])
const adEvery = ref(8)

// Works with an ad after every adEvery of them (ads rotate).
const cells = computed(() => {
  const out = []
  items.value.forEach((w, i) => {
    out.push({ key: w.id, work: w })
    if (ads.value.length && (i + 1) % adEvery.value === 0) {
      const n = (i + 1) / adEvery.value - 1
      out.push({ key: `ad-${n}`, ad: ads.value[n % ads.value.length] })
    }
  })
  return out
})

const current = computed(() => String(route.params.category || 'featured'))
const navCats = computed(() => [{ id: 'featured', name: '精选推荐' }, { id: 'all', name: '全部' }, ...cats.value])
const currentCat = computed(() => cats.value.find((c) => c.id === current.value))
const heading = computed(() => (current.value === 'featured' ? '精选案例' : current.value === 'all' ? '全部案例' : `${currentCat.value?.name || ''}案例`))
const subheading = computed(() => currentCat.value?.summary || '看看大家用 AI 做出的讲解视频和网页动画，喜欢就「制作同款」')
const createLink = computed(() => (currentCat.value ? { path: '/', query: { mode: currentCat.value.mode, category: currentCat.value.id } } : '/'))

async function fetchPage() {
  loading.value = true
  try {
    const params = new URLSearchParams({ category: current.value, page: String(page.value), size: '24' })
    if (q.value.trim()) params.set('q', q.value.trim())
    if (sort.value) params.set('sort', sort.value)
    const r = await api(`/gallery?${params}`, { auth: false })
    items.value = page.value === 1 ? r.items : [...items.value, ...r.items]
    total.value = r.total
  } catch {
    if (page.value === 1) items.value = []
  } finally {
    loading.value = false
  }
}
function reload() {
  page.value = 1
  fetchPage()
}
function more() {
  page.value++
  fetchPage()
}
watch(current, reload)
onMounted(() => {
  catalog().then((c) => (cats.value = c.categories))
  api('/ads', { auth: false })
    .then((r) => {
      ads.value = r.items || []
      adEvery.value = r.every || 8
    })
    .catch(() => {})
  reload()
})
</script>
