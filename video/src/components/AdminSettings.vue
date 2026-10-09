<template>
  <div class="space-y-8">
    <section class="card p-5">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="font-semibold">创作页的模型</h2>
          <p class="mt-1 text-sm text-ink-500">每个模型走所选分组计费：用户第一次用某个模型时，系统在他的账号下建一个「HiveGPT 视频」Key 并允许他使用这个分组，从余额扣费。建议给视频站单独建专属分组，在主站后台设置倍率和账号。</p>
        </div>
        <button class="btn-ghost btn-sm shrink-0" @click="addModel">添加模型</button>
      </div>
      <p v-if="!st.models.length" class="mt-4 rounded-xl bg-amber-50 p-3 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">还没有配置模型：现在用户用自己登录时的 Key 和它分组里的模型。</p>
      <div v-for="(m, i) in st.models" :key="i" class="mt-3 grid items-center gap-2 sm:grid-cols-[1.2fr_1fr_1fr_1.4fr_auto_auto]">
        <input v-model.trim="m.id" class="input" placeholder="模型名，如 claude-opus-5-5" />
        <input v-model.trim="m.name" class="input" placeholder="显示名" />
        <input v-model.trim="m.note" class="input" placeholder="提示（可选）" />
        <select v-model.number="m.group_id" class="input">
          <option :value="0" disabled>选择分组</option>
          <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }} · {{ g.platform }} · {{ g.rate_multiplier }}x{{ g.exclusive ? ' · 专属' : '' }}</option>
        </select>
        <label class="flex items-center gap-1 text-sm"><input type="radio" :checked="m.default" class="accent-brand-500" @change="setDefault(i)" />默认</label>
        <button class="btn-ghost btn-sm text-red-600" @click="st.models.splice(i, 1)">删除</button>
      </div>
    </section>

    <section class="card p-5">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="font-semibold">案例库广告</h2>
          <p class="mt-1 text-sm text-ink-500">启用的广告按顺序轮流插在案例之间，卡片上标「广告」，点击在新窗口打开链接。</p>
        </div>
        <button class="btn-ghost btn-sm shrink-0" @click="addAd">添加广告</button>
      </div>
      <label class="mt-4 flex items-center gap-2 text-sm">每隔 <input v-model.number="st.ad_every" type="number" min="3" max="30" class="input w-20" /> 个案例插一个广告</label>
      <div v-for="(a, i) in st.ads" :key="a.id || i" class="mt-4 flex flex-wrap gap-4 rounded-xl border border-ink-200 p-3 dark:border-ink-800">
        <label class="relative flex h-24 w-40 shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-ink-100 text-xs text-ink-500 dark:bg-ink-800">
          <img v-if="a.image" :src="a.image" class="absolute inset-0 h-full w-full object-cover" />
          <span v-else>上传图片（16:9）</span>
          <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" class="hidden" @change="upload(a, $event)" />
        </label>
        <div class="grid min-w-[16rem] flex-1 gap-2">
          <input v-model.trim="a.title" class="input" placeholder="标题" />
          <input v-model.trim="a.text" class="input" placeholder="一句话说明（可选）" />
          <input v-model.trim="a.link" class="input" placeholder="链接 https://…" />
        </div>
        <div class="flex flex-col items-end justify-between gap-2">
          <label class="flex items-center gap-1 text-sm"><input v-model="a.enabled" type="checkbox" class="accent-brand-500" />启用</label>
          <button class="btn-ghost btn-sm text-red-600" @click="st.ads.splice(i, 1)">删除</button>
        </div>
      </div>
    </section>

    <div class="flex justify-end"><button class="btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存设置' }}</button></div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api, session } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const st = reactive({ models: [], ads: [], ad_every: 8 })
const groups = ref([])
const saving = ref(false)

onMounted(async () => {
  try {
    const r = await api('/admin/settings')
    Object.assign(st, { models: r.settings.models || [], ads: r.settings.ads || [], ad_every: r.settings.ad_every || 8 })
    groups.value = r.groups || []
  } catch (err) {
    toastError(err)
  }
})
const addModel = () => st.models.push({ id: '', name: '', note: '', group_id: 0, default: !st.models.length })
const setDefault = (i) => st.models.forEach((m, j) => (m.default = i === j))
const addAd = () => st.ads.push({ id: '', title: '', text: '', image: '', link: '', enabled: true })

async function upload(ad, e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (file.size > 2 * 1024 * 1024) return toast('图片不能超过 2 MB')
  const form = new FormData()
  form.append('file', file)
  try {
    const res = await fetch('/api/v1/video/admin/ads/image', { method: 'POST', headers: { Authorization: `Bearer ${session.key}` }, body: form })
    const data = await res.json()
    if (data.code !== 0) throw new Error(data.message || '上传失败')
    ad.image = data.data.url
  } catch (err) {
    toastError(err)
  }
}

async function save() {
  saving.value = true
  try {
    const r = await api('/admin/settings', { method: 'PUT', body: { ...st } })
    Object.assign(st, r.settings)
    toast('已保存')
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}
</script>
