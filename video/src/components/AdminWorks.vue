<template>
  <div>
    <div class="flex flex-wrap items-center gap-2">
      <input v-model="q" class="input w-56" placeholder="搜索标题" @keydown.enter="reload" />
      <select v-model="category" class="input w-36" @change="reload">
        <option value="all">全部分类</option>
        <option value="featured">只看精选</option>
        <option v-for="c in cats" :key="c.id" :value="c.id">{{ c.name }}</option>
      </select>
      <button class="btn-ghost btn-sm" @click="reload">搜索</button>
      <RouterLink to="/upload" class="btn-primary btn-sm ml-auto"><UploadCloud class="h-4 w-4" />上传作品</RouterLink>
    </div>
    <p class="mt-2 text-sm text-ink-500">案例库里已公开的作品，共 {{ total }} 个。可以改标题、介绍、分类、封面，设精选或撤回。</p>

    <div class="mt-4 space-y-3">
      <div v-for="w in items" :key="w.id" class="card flex flex-wrap items-center gap-4 p-4">
        <img v-if="w.media?.poster" :src="w.media.poster" alt="" class="h-14 w-24 shrink-0 rounded-lg bg-black object-cover" />
        <div class="min-w-0 flex-1">
          <a :href="`/w/${w.id}`" target="_blank" class="font-medium hover:text-brand-600">{{ w.title }}</a>
          <span v-if="w.featured" class="ml-2 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">精选</span>
          <p class="mt-1 text-xs text-ink-400">
            {{ catName(w.category) }} · {{ modeLabel(w) }}<template v-if="w.duration"> · {{ fmtTime(w.duration) }}</template> · {{ w.views }} 次观看 · {{ w.author }}
          </p>
        </div>
        <button class="btn-ghost btn-sm" @click="edit(w)"><Pencil class="h-4 w-4" />编辑</button>
      </div>
      <p v-if="loaded && !items.length" class="card p-8 text-center text-ink-500">没有找到作品</p>
      <button v-if="items.length < total" class="btn-ghost mx-auto block" :disabled="loading" @click="more">加载更多</button>
    </div>

    <Modal v-if="editing" title="编辑作品" @close="editing = null">
      <div class="space-y-4">
        <label class="block">
          <span class="text-sm font-medium">标题</span>
          <input v-model="form.title" maxlength="60" class="input mt-1" />
        </label>
        <label class="block">
          <span class="text-sm font-medium">作品介绍</span>
          <textarea v-model="form.description" rows="4" maxlength="1000" class="input mt-1 resize-none"></textarea>
        </label>
        <div class="flex flex-wrap items-center gap-4">
          <label class="block">
            <span class="text-sm font-medium">分类</span>
            <select v-model="form.category" class="input mt-1 w-40">
              <option v-for="c in cats" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>
          <label class="mt-6 flex items-center gap-1 text-sm"><input v-model="form.featured" type="checkbox" class="accent-brand-500" />精选（排在案例库前面）</label>
        </div>
        <div v-if="editing.mode === 'upload' && editing.spec?.upload?.kind === 'video' && editing.media">
          <div class="flex items-center gap-3">
            <span class="text-sm font-medium">封面</span>
            <button v-if="!pickCover" class="text-sm text-brand-600 hover:underline" @click="pickCover = true">更换封面</button>
          </div>
          <img v-if="!pickCover && editing.media.poster" :src="editing.media.poster" alt="封面" class="mt-1 w-48 rounded-lg" />
          <PosterPicker v-if="pickCover" class="mt-1" :src="editing.media.url" @poster="(p) => (poster = p.blob)" />
        </div>
        <div class="flex flex-wrap gap-2 pt-2">
          <button class="btn-primary flex-1" :disabled="saving || !form.title.trim()" @click="save">
            <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />保存
          </button>
          <button class="btn-ghost text-red-600" :disabled="saving" @click="hide">从案例库撤回</button>
        </div>
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Loader2, Pencil, UploadCloud } from 'lucide-vue-next'
import Modal from './Modal.vue'
import PosterPicker from './PosterPicker.vue'
import { api, apiUpload, catalog } from '../lib/api'
import { fmtTime, modeLabel } from '../lib/media'
import { toast, toastError } from '../lib/toast'

const items = ref([])
const total = ref(0)
const page = ref(1)
const q = ref('')
const category = ref('all')
const cats = ref([])
const loaded = ref(false)
const loading = ref(false)
const editing = ref(null)
const saving = ref(false)
const poster = ref(null)
const pickCover = ref(false)
const form = reactive({ title: '', description: '', category: 'other', featured: false })

const catName = (id) => cats.value.find((c) => c.id === id)?.name || id

async function fetchPage() {
  loading.value = true
  try {
    const params = new URLSearchParams({ category: category.value, page: String(page.value), size: '48', sort: 'new' })
    if (q.value.trim()) params.set('q', q.value.trim())
    const r = await api(`/gallery?${params}`, { auth: false })
    items.value = page.value === 1 ? r.items : [...items.value, ...r.items]
    total.value = r.total
  } catch (err) {
    toastError(err)
  } finally {
    loading.value = false
    loaded.value = true
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

async function edit(w) {
  try {
    const p = await api(`/admin/works/${w.id}`)
    Object.assign(form, { title: p.title, description: p.prompt || '', category: p.category || 'other', featured: !!p.featured })
    poster.value = null
    pickCover.value = false
    editing.value = p
  } catch (err) {
    toastError(err)
  }
}

async function save() {
  const id = editing.value.id
  saving.value = true
  try {
    if (pickCover.value && poster.value) {
      const fd = new FormData()
      fd.append('poster', poster.value, 'poster.jpg')
      await apiUpload(`/admin/works/${id}/poster`, fd)
    }
    await api(`/admin/works/${id}`, { method: 'PUT', body: { title: form.title, description: form.description, category: form.category, featured: form.featured } })
    toast('已保存', 'success')
    editing.value = null
    reload()
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}

async function hide() {
  if (!confirm('撤回后作品不再出现在案例库，作者仍能在「历史创作」里看到。确定撤回？')) return
  saving.value = true
  try {
    await api(`/admin/works/${editing.value.id}`, { method: 'PUT', body: { visibility: 'private' } })
    toast('已撤回')
    editing.value = null
    reload()
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  catalog().then((c) => (cats.value = c.categories))
  reload()
})
</script>
