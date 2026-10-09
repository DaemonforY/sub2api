<template>
  <aside class="sticky top-20 self-start">
    <div class="flex items-center justify-between">
      <h2 class="text-sm font-semibold text-brand-600">历史创作</h2>
      <RouterLink to="/" class="rounded-lg p-1 text-ink-400 hover:bg-ink-100 hover:text-brand-600 dark:hover:bg-ink-800" title="新建创作"><Plus class="h-4 w-4" /></RouterLink>
    </div>
    <div class="thin-scroll mt-3 max-h-[calc(100vh-8rem)] space-y-1 overflow-y-auto pr-1">
      <p v-if="!session.key" class="rounded-xl border border-dashed border-ink-200 p-3 text-xs leading-5 text-ink-500 dark:border-ink-700">登录后查看历史作品</p>
      <p v-else-if="loaded && !items.length" class="rounded-xl border border-dashed border-ink-200 p-3 text-xs text-ink-500 dark:border-ink-700">还没有作品，写下第一个想法吧</p>
      <RouterLink
        v-for="p in items"
        :key="p.id"
        :to="`/p/${p.id}`"
        class="block rounded-xl px-3 py-2 transition hover:bg-white dark:hover:bg-ink-900"
        :class="{ 'bg-brand-50 dark:bg-brand-900/20': p.id === current }"
      >
        <span class="block truncate text-sm font-medium">{{ p.title || p.prompt }}</span>
        <span class="mt-0.5 flex items-center gap-1.5 text-[11px] text-ink-500">
          <span class="rounded px-1 py-px" :class="badge(p)">{{ modeLabel(p) }}</span>
          <span v-if="p.status === 'running'" class="text-brand-600">生成中…</span>
          <span v-else-if="p.status === 'questions'" class="text-brand-600">待确认</span>
          <span v-else-if="p.status === 'failed'" class="text-red-500">失败</span>
          <span v-else>{{ when(p.updated_at) }}</span>
        </span>
      </RouterLink>
    </div>
  </aside>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { Plus } from 'lucide-vue-next'
import { api, session } from '../lib/api'
import { modeLabel } from '../lib/media'

defineProps({ current: { type: String, default: '' } })
const items = ref([])
const loaded = ref(false)

async function load() {
  if (!session.key) {
    items.value = []
    return
  }
  try {
    items.value = (await api('/projects')).items
  } catch {
    items.value = []
  } finally {
    loaded.value = true
  }
}
defineExpose({ load })
onMounted(load)
watch(() => session.key, load)

function badge(p) {
  if (p.mode === 'film') return 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300'
  if (p.mode === 'upload') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
}
function when(iso) {
  const d = new Date(iso)
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${d.getMonth() + 1}月${d.getDate()}日`
}
</script>
