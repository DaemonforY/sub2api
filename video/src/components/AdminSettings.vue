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
      <h2 class="font-semibold">画面检查</h2>
      <p class="mt-1 text-sm text-ink-500">新做的影片在分镜写完后，渲染服务给每个分镜截 4 张图（带字幕），交给同一个模型对照检查文字重叠、超出、留白太多、角色走样、数字前后不一致、播放报错，有问题就改一次代码。每条片子多一轮模型调用（实测约多 40%–60% 费用、多 1–3 分钟），费用记在用户自己的 Key 上。渲染服务正在导出 MP4 超过 2 分钟时，这条片子跳过检查。</p>
      <label class="mt-4 flex items-center gap-2 text-sm"><input v-model="reviewOn" type="checkbox" class="accent-brand-500" />开启画面检查</label>
    </section>

    <section class="card p-5">
      <h2 class="font-semibold">付费配音（豆包语音 2.0）</h2>
      <p class="mt-1 text-sm text-ink-500">用户选豆包音色时，按旁白字数从余额扣费，每段配好后扣一次，失败不扣；余额不够本次配音时不开始。豆包的成本约 ¥3/万字。服务器配置了 VIDEO_DOUBAO_API_KEY（豆包语音控制台 → API Key）后，创作页才会出现这些音色。</p>
      <label class="mt-4 flex items-center gap-2 text-sm">每万字 $ <input v-model.number="st.tts_price_per_10k" type="number" min="0.01" max="1000" step="0.5" class="input w-24" /></label>
    </section>

    <section class="card p-5">
      <h2 class="font-semibold">我的音色（豆包声音复刻 2.0）</h2>
      <p class="mt-1 text-sm leading-6 text-ink-500">
        用户在「我的音色」录 10–30 秒声音复刻自己的音色。音色位要先在火山引擎「豆包语音 → 声音复刻」里购买（≤50 个 ¥138/个，会到期，到期需续费），把控制台里的音色 ID（S_ 开头）每行一个填在下面；用户新建音色时自动分配一个空闲的位，删除后这个位可以给下一个用户重新训练（每个位豆包最多训练 15 次）。
        复刻费在训练成功后从余额扣一次，配音按上面的每万字价格扣。不要在控制台里「启用」这些音色，启用后就不能再重新训练。
      </p>
      <div class="mt-4 grid gap-4 sm:grid-cols-[1fr_auto]">
        <label class="text-sm">音色 ID（每行一个，共 {{ slotCount }} 个）
          <textarea v-model="slotsText" rows="5" class="input mt-1 font-mono text-xs" placeholder="S_xxxxxxxx"></textarea>
        </label>
        <div class="space-y-3 text-sm">
          <label class="flex items-center gap-2">复刻费 $ <input v-model.number="st.clone_price" type="number" min="0" max="10000" step="1" class="input w-24" /></label>
          <label class="flex items-center gap-2">每人最多 <input v-model.number="st.clone_max_per_user" type="number" min="1" max="20" class="input w-20" /> 个</label>
        </div>
      </div>
      <div v-if="clones.length" class="mt-5 overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="text-ink-400"><tr><th class="py-1 pr-3">#</th><th class="pr-3">用户</th><th class="pr-3">名字</th><th class="pr-3">状态</th><th class="pr-3">音色位</th><th class="pr-3">训练</th><th class="pr-3">授权时间 / IP</th><th></th></tr></thead>
          <tbody>
            <tr v-for="c in clones" :key="c.id" class="border-t border-ink-100 dark:border-ink-800">
              <td class="py-1.5 pr-3">{{ c.id }}</td>
              <td class="pr-3">{{ c.user_id }} {{ c.user_email }}</td>
              <td class="pr-3">{{ c.name }}</td>
              <td class="pr-3">{{ { training: '复刻中', ready: '可用', failed: '失败', disabled: '已停用' }[c.status] || c.status }}{{ c.charged ? ` · 已扣 $${c.price}` : '' }}</td>
              <td class="pr-3 font-mono">{{ c.speaker_id }}</td>
              <td class="pr-3">{{ c.trainings }}</td>
              <td class="pr-3">{{ new Date(c.consent_at).toLocaleString() }} · {{ c.consent_ip }}</td>
              <td><button class="btn-ghost btn-sm" @click="toggleClone(c)">{{ c.status === 'disabled' ? '恢复' : '停用' }}</button></td>
            </tr>
          </tbody>
        </table>
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
import { computed, onMounted, reactive, ref } from 'vue'
import { api, session } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const st = reactive({ models: [], ads: [], ad_every: 8, tts_price_per_10k: 6, clone_slots: [], clone_price: 0, clone_max_per_user: 3, review_off: false })
const reviewOn = computed({ get: () => !st.review_off, set: (v) => { st.review_off = !v } })
const slotsText = ref('')
const slotList = () => slotsText.value.split(/[\s,，]+/).map((x) => x.trim()).filter(Boolean)
const slotCount = computed(() => slotList().length)
const clones = ref([])
const groups = ref([])
const saving = ref(false)

onMounted(async () => {
  try {
    const r = await api('/admin/settings')
    Object.assign(st, {
      models: r.settings.models || [], ads: r.settings.ads || [], ad_every: r.settings.ad_every || 8, tts_price_per_10k: r.settings.tts_price_per_10k || 6,
      clone_slots: r.settings.clone_slots || [], clone_price: r.settings.clone_price || 138, clone_max_per_user: r.settings.clone_max_per_user || 3,
      review_off: !!r.settings.review_off
    })
    slotsText.value = st.clone_slots.join('\n')
    groups.value = r.groups || []
  } catch (err) {
    toastError(err)
  }
})
onMounted(loadClones)

async function loadClones() {
  try {
    clones.value = (await api('/admin/clones')).items || []
  } catch {
    clones.value = []
  }
}

async function toggleClone(c) {
  const disable = c.status !== 'disabled'
  if (disable && !confirm(`停用「${c.name}」？用户将不能再用它配音（已配好的作品不变）。`)) return
  try {
    await api(`/admin/clones/${c.id}/disable`, { method: 'POST', body: { disable } })
    await loadClones()
  } catch (err) {
    toastError(err)
  }
}
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
    st.clone_slots = slotList()
    const r = await api('/admin/settings', { method: 'PUT', body: { ...st } })
    Object.assign(st, r.settings)
    slotsText.value = (st.clone_slots || []).join('\n')
    toast('已保存')
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}
</script>
