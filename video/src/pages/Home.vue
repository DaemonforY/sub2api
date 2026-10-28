<template>
  <div class="relative">
    <div class="pointer-events-none absolute inset-x-0 top-0 h-[520px] bg-gradient-to-b from-brand-100/70 via-brand-50/40 to-transparent dark:from-brand-900/20 dark:via-transparent"></div>
    <div class="bg-grid pointer-events-none absolute inset-x-0 top-0 h-[520px] [mask-image:linear-gradient(to_bottom,black,transparent)]"></div>

    <div class="relative mx-auto flex max-w-[1440px] gap-6 px-4 pb-10 pt-10 sm:px-6">
      <HistorySidebar class="hidden w-60 shrink-0 xl:block" />

      <div class="min-w-0 flex-1">
        <section class="text-center">
          <h1 class="text-3xl font-extrabold tracking-tight sm:text-5xl">
            一句话生成<span class="whitespace-nowrap bg-gradient-to-r from-brand-500 to-rose-500 bg-clip-text italic text-transparent"> 讲解视频 </span>与<span class="whitespace-nowrap bg-gradient-to-r from-brand-500 to-amber-500 bg-clip-text italic text-transparent"> 网页动画</span>
          </h1>
          <p class="mx-auto mt-4 max-w-2xl text-[15px] leading-7 text-ink-600 dark:text-ink-300">不用剪辑、不用写代码。写下主题，AI 写脚本、配音、做动画，带字幕的视频几分钟就好；Logo、图表、流程图这类动画，一句话就能生成。</p>
          <div class="mt-6 inline-flex items-center gap-1 rounded-full border border-ink-200 bg-white p-1 shadow-sm dark:border-ink-700 dark:bg-ink-900">
            <button v-for="m in modes" :key="m.id" class="rounded-full px-5 py-2 text-sm font-medium transition" :class="mode === m.id ? 'bg-gradient-to-r from-brand-500 to-brand-600 text-white shadow' : 'text-ink-600 hover:text-brand-600 dark:text-ink-300'" @click="mode = m.id">
              {{ m.name }}
            </button>
            <span class="group relative ml-1 mr-1 flex h-7 w-7 cursor-help items-center justify-center rounded-full text-ink-400 hover:bg-ink-100 dark:hover:bg-ink-800">
              <HelpCircle class="h-4 w-4" />
              <span class="invisible absolute left-1/2 top-full z-20 mt-2 w-72 -translate-x-1/2 rounded-xl bg-ink-900 p-3 text-left text-xs leading-5 text-white opacity-0 shadow-xl transition group-hover:visible group-hover:opacity-100">
                <b>HTML 视频</b>：有旁白配音和字幕，分多个分镜，适合知识讲解、课程、故事和宣传。<br /><b>AI 动画</b>：一个画面循环播放，没有配音，适合 Logo、文字、图表、加载动画。
              </span>
            </span>
          </div>
        </section>

        <!-- prompt box -->
        <section class="card mx-auto mt-6 max-w-5xl overflow-visible shadow-soft" :class="{ 'ring-2 ring-brand-400': dropping }" @dragover.prevent="dropping = true" @dragleave.self="dropping = false" @drop.prevent="onDrop">
          <div class="flex items-center gap-2 px-5 pt-4 text-xs font-medium tracking-wide text-ink-500">
            <Clapperboard v-if="mode === 'film'" class="h-4 w-4 text-brand-500" />
            <Wand2 v-else class="h-4 w-4 text-brand-500" />
            {{ mode === 'film' ? '描述你的视频' : '描述你的动画' }}
            <span v-if="remixTitle" class="ml-2 rounded-md bg-brand-50 px-2 py-0.5 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300">制作同款：{{ remixTitle }}</span>
          </div>
          <textarea
            ref="promptEl"
            v-model="prompt"
            rows="5"
            maxlength="5000"
            class="block w-full resize-none border-0 bg-transparent px-5 py-3 text-[15px] leading-7 outline-none placeholder:text-ink-400"
            :placeholder="placeholder"
            data-testid="create-prompt"
            @keydown.meta.enter="create"
            @keydown.ctrl.enter="create"
            @paste="onPaste"
          ></textarea>
          <div class="px-5 pb-2">
            <ImageAttach ref="attach" :items="images" :max="9" />
            <p v-if="images.length" class="mt-1.5 text-xs text-ink-400">AI 会先看图再写脚本，把这些图做成视频的主要画面（推拉、标注、分屏）。写清楚想讲什么、给谁看，效果更好。</p>
          </div>
          <div class="flex items-center justify-between px-5 pb-1 text-xs text-ink-400">
            <div class="flex flex-wrap gap-1.5">
              <button v-for="ex in examples" :key="ex" class="max-w-[22rem] truncate rounded-lg bg-ink-100 px-2 py-1 text-ink-600 hover:bg-brand-50 hover:text-brand-700 dark:bg-ink-800 dark:text-ink-300" :title="ex" @click="prompt = ex">{{ ex }}</button>
            </div>
            <span class="shrink-0 tabular-nums">{{ prompt.length }}/5000</span>
          </div>
          <div class="flex flex-wrap items-center gap-2 border-t border-ink-100 px-4 py-3 dark:border-ink-800">
            <div class="flex overflow-hidden rounded-xl border border-ink-200 dark:border-ink-700" role="radiogroup" aria-label="画面比例">
              <button v-for="r in ratioChoices" :key="r.id" class="flex items-center gap-1 px-2.5 py-1.5 text-xs" :class="ratio === r.id ? 'bg-brand-500 text-white' : 'text-ink-600 hover:bg-ink-50 dark:text-ink-300 dark:hover:bg-ink-800'" :title="r.title" @click="ratio = r.id">
                <component :is="r.icon" class="h-3.5 w-3.5" />{{ r.label }}
              </button>
            </div>
            <Picker v-model="model" :options="modelOptions" icon="cpu" label="模型" />
            <Picker v-if="mode === 'film'" v-model="voice" :options="voiceOptions" icon="mic" label="音色" />
            <Picker v-if="mode === 'film'" v-model="length" :options="lengthOptions" icon="clock" label="时长" />
            <Picker v-else v-model="seconds" :options="secondOptions" icon="clock" label="循环时长" />
            <button class="chip" @click="scrollToStyles"><Palette class="h-3.5 w-3.5 text-brand-500" />{{ styleName }}</button>
            <label v-if="mode === 'film'" class="chip cursor-pointer" title="开始前先问 1–3 个问题确认讲解深度、时长和语气">
              <input v-model="ask" type="checkbox" class="h-3.5 w-3.5 accent-brand-500" />先确认方向
            </label>
            <span class="flex-1"></span>
            <button class="btn-primary px-5 py-2.5" :disabled="creating || !prompt.trim() || imagesBusy" :title="imagesBusy ? '图片还在上传' : ''" data-testid="create-submit" @click="create">
              <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
              <Sparkles v-else class="h-4 w-4" />
              {{ mode === 'film' ? '一键生成视频' : '生成动画' }}
            </button>
          </div>
        </section>
        <p v-if="lowBalanceTip" class="mx-auto mt-3 max-w-5xl rounded-xl bg-amber-50 px-4 py-2.5 text-sm leading-6 text-amber-800 dark:bg-amber-900/20 dark:text-amber-200" data-testid="low-balance-tip">
          <template v-if="balance <= 0">余额不足，Claude 暂时用不了。</template>
          <template v-else>
            这个{{ mode === 'film' ? '视频' : '动画' }}用 Claude 预计要 {{ costText(cost) }}，你的余额是 ${{ balance.toFixed(2) }}，{{ balance < cost.lo ? '做到一半就会停' : '可能不够做完' }}。
          </template>
          <button v-if="cheapModel" class="ml-1 font-medium underline" data-testid="low-balance-cheap" @click="model = cheapModel.id">换成 {{ cheapModel.name }}（约 {{ costText(cheapCost) }}）</button>
          <a :href="`${MAIN_SITE_URL}/purchase`" target="_blank" class="ml-2 font-medium underline">去充值</a>
        </p>
        <Modal v-if="costConfirm" title="余额可能不够做完" @close="costConfirm = false">
          <div data-testid="cost-confirm">
            <p class="text-sm leading-6 text-ink-600 dark:text-ink-300">
              用 Claude 做这个{{ mode === 'film' ? '视频' : '动画' }}预计要 <b>{{ costText(cost) }}</b>（Claude 按 3 倍计费），你的余额是 <b>${{ balance.toFixed(2) }}</b>。<template v-if="balance > 0">余额用完时会停在半路，已经花掉的不会退回；充值后可以点「继续生成」接着做。</template><template v-else>余额不足时 Claude 不能开始。</template>
            </p>
            <div class="mt-5 flex flex-wrap justify-end gap-2">
              <a :href="`${MAIN_SITE_URL}/purchase`" target="_blank" class="btn-ghost btn-sm">去充值</a>
              <button v-if="balance > 0" class="btn-ghost btn-sm" data-testid="cost-confirm-claude" @click="createWith('')">仍用 Claude</button>
              <button v-if="cheapModel" class="btn-primary btn-sm" data-testid="cost-confirm-cheap" @click="createWith(cheapModel.id)">换成 {{ cheapModel.name }} 生成（约 {{ costText(cheapCost) }}）</button>
            </div>
          </div>
        </Modal>
        <p class="mx-auto mt-2 max-w-5xl text-center text-xs text-ink-400">用 HiveGPT 账号登录，按实际用量从账户余额扣费。一个 1 分钟的视频通常需要 2–5 分钟生成，期间可以关掉页面。</p>
        <p class="mx-auto mt-1 max-w-5xl text-center text-xs text-ink-500">已经有做好的视频或 SVG 动画？<RouterLink to="/upload" class="font-medium text-brand-600 hover:underline" data-testid="home-upload">上传作品 →</RouterLink></p>

        <!-- styles / categories -->
        <section ref="stylesEl" class="mx-auto mt-12 max-w-6xl scroll-mt-24">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 class="text-xl font-bold">{{ mode === 'film' ? '选择影片类型与视觉风格' : '选择动画类型与视觉风格' }}</h2>
              <p class="mt-1 text-sm text-ink-500">风格决定配色、字体和画面语言；也可以交给 AI 根据主题决定。</p>
            </div>
            <button v-if="!showAllStyles && filteredStyles.length > 10" class="btn-ghost btn-sm" @click="showAllStyles = true">展开全部 {{ filteredStyles.length }} 种风格</button>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <template v-if="mode === 'film'">
              <button class="chip" :class="{ 'chip-on': genre === '' }" @click="genre = ''">全部类型</button>
              <button v-for="g in cat.genres" :key="g.id" class="chip" :class="{ 'chip-on': genre === g.id }" :title="g.summary" @click="genre = g.id">{{ g.name }}</button>
            </template>
            <template v-else>
              <button v-for="c in motionCategories" :key="c.id" class="chip" :class="{ 'chip-on': category === c.id }" :title="c.summary" @click="pickCategory(c)">{{ c.name }}</button>
            </template>
          </div>
          <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            <StyleCard v-for="s in shownStyles" :key="s.id" :item="s" :selected="style === s.id" @pick="style = $event" />
          </div>
        </section>

        <!-- featured works -->
        <section v-if="featured.length" class="mx-auto mt-14 max-w-6xl">
          <div class="flex items-end justify-between">
            <div>
              <h2 class="text-xl font-bold">大家在做什么</h2>
              <p class="mt-1 text-sm text-ink-500">鼠标移上去就会播放，点「制作同款」直接套用。</p>
            </div>
            <RouterLink to="/gallery" class="text-sm text-brand-600 hover:underline">查看全部案例 →</RouterLink>
          </div>
          <div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <WorkCard v-for="w in featured" :key="w.id" :work="w" :categories="cat.categories" />
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Clapperboard, HelpCircle, Loader2, Monitor, Palette, Smartphone, Sparkles, Square, Wand2 } from 'lucide-vue-next'
import HistorySidebar from '../components/HistorySidebar.vue'
import ImageAttach from '../components/ImageAttach.vue'
import Modal from '../components/Modal.vue'
import Picker from '../components/Picker.vue'
import StyleCard from '../components/StyleCard.vue'
import WorkCard from '../components/WorkCard.vue'
import { MAIN_SITE_URL, api, catalog, myClones, session, signIn, textModels } from '../lib/api'
import { toast, toastError } from '../lib/toast'
import { prefs, savePrefs } from '../lib/prefs'

const route = useRoute()
const router = useRouter()
const modes = [
  { id: 'film', name: 'HTML 视频' },
  { id: 'motion', name: 'AI 动画' }
]
const cat = ref({ styles: [], genres: [], categories: [], ratios: [], voices: [] })
const models = ref([])
const featured = ref([])
const clones = ref([]) // 我的音色
const cloneOn = ref(false)

const mode = ref(route.query.mode === 'motion' ? 'motion' : prefs.mode || 'film')
const prompt = ref(typeof route.query.prompt === 'string' ? route.query.prompt : '')
const ratio = ref(prefs.ratio || '16:9')
const model = ref(prefs.model || 'gpt-5.5')
const voice = ref(prefs.voice || 'zh-CN-XiaoxiaoNeural')
const length = ref(prefs.length || 'standard')
const seconds = ref(prefs.seconds || 8)
const style = ref(prefs.style || 'auto')
const genre = ref('')
const category = ref(typeof route.query.category === 'string' ? route.query.category : prefs.category || 'other')
const ask = ref(prefs.ask ?? true)
const showAllStyles = ref(false)
const creating = ref(false)
const remixOf = ref('')
const remixTitle = ref('')
const stylesEl = ref(null)
const promptEl = ref(null)

const ratioChoices = computed(() => {
  const all = [
    { id: '16:9', label: '横屏', title: '16:9', icon: Monitor },
    { id: '9:16', label: '竖屏', title: '9:16', icon: Smartphone },
    { id: '1:1', label: '方形', title: '1:1', icon: Square }
  ]
  return mode.value === 'film' ? all.slice(0, 2) : all
})
// The site's own model list (admin settings); without one, the models of the signed-in key's group.
const siteModels = ref([])
const modelOptions = computed(() => {
  if (siteModels.value.length) {
    const what = mode.value === 'motion' ? '一个动画' : (lengthOptions.find((o) => o.value === length.value)?.label || '一个视频').replace(/^约\s*/, '')
    return siteModels.value.map((m) => ({ value: m.id, label: m.name, hint: [m.note, `${what}约 ${costText(runCost(m.id))}`].filter(Boolean).join(' · ') }))
  }
  const list = models.value.length ? [...models.value] : ['gpt-5.5']
  if (!list.includes(model.value)) list.unshift(model.value)
  return list.map((m) => ({ value: m, label: m }))
})
watch(siteModels, (list) => {
  if (list.length && !list.some((m) => m.id === model.value)) model.value = (list.find((m) => m.default) || list[0]).id
})
// Paid voices (豆包语音 2.0) are charged per character from the balance; the price is per 10,000 characters.
const voiceOptions = computed(() => [
  { value: 'none', label: '无配音', hint: '只有画面和屏幕文字，方便自己配音或加音乐' },
  ...cat.value.voices.map((v) =>
    v.paid
      ? { value: v.id, label: `${v.name}（${v.gender}）· 付费`, hint: `${v.style} · 豆包语音 $${cat.value.voice_price_per_10k}/万字` }
      : { value: v.id, label: `${v.name}（${v.gender}）`, hint: `${v.style} · 免费` }
  ),
  // The user's own cloned voices (ready ones), then a way to make one.
  ...clones.value
    .filter((c) => c.status === 'ready' || c.voice === voice.value)
    .map((c) => ({ value: c.voice, label: `${c.name} · 我的音色`, hint: c.status === 'ready' ? `你复刻的声音 · $${cat.value.voice_price_per_10k}/万字` : '还没复刻好，请换一个音色' })),
  ...(cloneOn.value ? [{ value: '__clone__', label: '＋ 复刻我的声音', hint: '录 10–30 秒，用自己的声音配音' }] : [])
])
watch(voice, (v, old) => {
  if (v !== '__clone__') return
  voice.value = old
  router.push('/voices')
})
const lengthOptions = [
  { value: 'short', label: '约 30 秒', hint: '3–4 个分镜' },
  { value: 'standard', label: '约 1 分钟', hint: '5–7 个分镜' },
  { value: 'long', label: '2–3 分钟', hint: '8–10 个分镜' }
]
// What a run usually costs from the balance, in dollars, from prod usage (first run up to the last
// scene, 2026-10-14..28): Claude is billed at 3× and writes ~3× the code, so a 1-minute film is
// $4–7 against GPT-5.5's ~$0.8, and the $2 sign-up credit stops a Claude film halfway.
const RUN_COST = {
  claude: { short: [3, 5], standard: [4, 7], long: [8, 11], motion: [0.5, 1.5] },
  other: { short: [0.5, 1], standard: [0.7, 1], long: [1, 2], motion: [0.1, 0.3] }
}
function runCost(id, m = mode.value, len = length.value) {
  const [lo, hi] = RUN_COST[id.startsWith('claude') ? 'claude' : 'other'][m === 'motion' ? 'motion' : len] || RUN_COST.other.standard
  return { lo, hi }
}
const costText = (c) => `$${c.lo}–${c.hi}`
const balance = computed(() => Number(session.me?.balance ?? 0))
const cost = computed(() => runCost(model.value))
// The cheaper model to offer instead: the first non-Claude one on the site.
const cheapModel = computed(() => {
  const list = siteModels.value.length ? siteModels.value : [{ id: 'gpt-5.5', name: 'GPT-5.5' }]
  return list.find((m) => !m.id.startsWith('claude')) || null
})
const cheapCost = computed(() => runCost(cheapModel.value?.id || 'gpt-5.5'))
const lowBalanceTip = computed(() => !!session.me && model.value.startsWith('claude') && balance.value < cost.value.hi)
// Pictures as material (uploaded while the user writes; the create request sends their tokens).
const images = ref([])
const attach = ref(null)
const dropping = ref(false)
const imagesBusy = computed(() => images.value.some((x) => x.status === 'uploading'))
function onDrop(e) {
  dropping.value = false
  attach.value?.add(Array.from(e.dataTransfer?.files || []))
}
function onPaste(e) {
  const files = Array.from(e.clipboardData?.files || [])
  if (files.some((f) => f.type.startsWith('image/'))) {
    e.preventDefault()
    attach.value?.add(files)
  }
}
const secondOptions = [6, 8, 12, 16].map((s) => ({ value: s, label: `${s} 秒` }))
const motionCategories = computed(() => cat.value.categories.filter((c) => c.mode === 'motion'))
const filteredStyles = computed(() => cat.value.styles.filter((s) => !genre.value || s.id === 'auto' || s.genre === genre.value))
const shownStyles = computed(() => (showAllStyles.value ? filteredStyles.value : filteredStyles.value.slice(0, 10)))
const styleName = computed(() => cat.value.styles.find((s) => s.id === style.value)?.name || '视觉风格')
const examples = computed(() => {
  if (mode.value === 'film') return ['用 60 秒讲清楚区块链为什么不能篡改', '给大学生介绍狭义相对论', '为一款番茄钟 App 做 30 秒宣传片']
  const c = cat.value.categories.find((x) => x.id === category.value)
  return c ? [c.example] : []
})
const placeholder = computed(() => (mode.value === 'film' ? '输入视频主题或完整口播稿，例如：用科技博主的风格讲解牛顿三大定律…' : '描述你想要的动画，例如：三个圆点依次跳动，像波浪一样'))

watch([mode, ratio, model, voice, length, seconds, style, category, ask], () =>
  savePrefs({ mode: mode.value, ratio: ratio.value, model: model.value, voice: voice.value, length: length.value, seconds: seconds.value, style: style.value, category: category.value, ask: ask.value })
)
watch(mode, (m) => {
  if (m === 'film' && ratio.value === '1:1') ratio.value = '16:9'
})

function pickCategory(c) {
  category.value = c.id
  if (!prompt.value.trim()) prompt.value = c.example
}
function scrollToStyles() {
  stylesEl.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// A Claude run the balance can't pay for asks first (the tip above the button is easy to miss).
const costConfirm = ref(false)
let costAccepted = false
function createWith(id) {
  costConfirm.value = false
  if (id) model.value = id
  else costAccepted = true
  create()
}

async function create() {
  if (creating.value || !prompt.value.trim()) return
  if (!session.key) {
    try {
      if (!(await signIn())) return
    } catch (err) {
      toastError(err)
      return
    }
  }
  if (!costAccepted && session.me && model.value.startsWith('claude') && balance.value < cost.value.lo) {
    costConfirm.value = true
    return
  }
  costAccepted = false
  creating.value = true
  try {
    const p = await api('/projects', {
      method: 'POST',
      body: {
        mode: mode.value,
        prompt: prompt.value.trim(),
        remix_of: remixOf.value || undefined,
        images: images.value.filter((x) => x.status === 'done').map((x) => x.token),
        options: {
          ratio: ratio.value, style: style.value, genre: genre.value || undefined, category: mode.value === 'motion' ? category.value : 'film',
          voice: voice.value, model: model.value, length: length.value, ask: mode.value === 'film' && ask.value, seconds: seconds.value
        }
      }
    })
    images.value.splice(0)
    router.push(`/p/${p.id}`)
  } catch (err) {
    toastError(err)
  } finally {
    creating.value = false
  }
}

async function loadRemix(id) {
  try {
    const w = await api(`/works/${id}`, { auth: false })
    remixOf.value = w.id
    remixTitle.value = w.title
    mode.value = w.mode
    prompt.value = w.prompt
    const o = w.options || {}
    if (o.ratio) ratio.value = o.ratio
    if (o.style) style.value = o.style
    if (o.voice && !o.voice.startsWith('clone:')) voice.value = o.voice // a cloned voice is its owner's only
    if (o.length) length.value = o.length
    if (o.category && w.mode === 'motion') category.value = o.category
    promptEl.value?.focus()
  } catch {
    toast('这个案例已经不在了')
  }
}

onMounted(async () => {
  catalog().then((c) => (cat.value = c)).catch(toastError)
  if (typeof route.query.remix === 'string') loadRemix(route.query.remix)
  if (typeof route.query.style === 'string') style.value = route.query.style
  api('/gallery?category=featured&size=8', { auth: false }).then((r) => (featured.value = r.items)).catch(() => {})
  api('/models', { auth: false }).then((r) => (siteModels.value = r.items || [])).catch(() => {})
  api('/clone-info', { auth: false }).then((r) => (cloneOn.value = !!r.enabled)).catch(() => {})
  loadClones()
  models.value = await textModels()
})
watch(() => session.key, async () => {
  loadClones()
  models.value = await textModels()
})

// A remembered cloned voice that is gone (deleted, or another account) falls back to the default.
async function loadClones() {
  clones.value = await myClones()
  if (session.key && voice.value.startsWith('clone:') && !clones.value.some((c) => c.voice === voice.value)) voice.value = 'zh-CN-XiaoxiaoNeural'
}
</script>
