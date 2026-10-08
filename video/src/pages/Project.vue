<template>
  <div class="flex h-[calc(100vh-4rem)] min-h-[560px] gap-0 bg-ink-50 dark:bg-ink-950">
    <HistorySidebar ref="history" :current="id" class="hidden w-60 shrink-0 border-r border-ink-200 px-3 py-4 xl:block dark:border-ink-800" />

    <!-- centre: preview / script / code -->
    <section class="flex min-w-0 flex-1 flex-col">
      <div class="flex flex-wrap items-center gap-2 border-b border-ink-200 bg-white px-4 py-2.5 dark:border-ink-800 dark:bg-ink-900">
        <h1 class="mr-2 truncate text-[15px] font-semibold">{{ p?.mode === 'motion' ? '动画预览' : 'HTML 视频预览' }}</h1>
        <div class="flex rounded-xl bg-ink-100 p-0.5 dark:bg-ink-800">
          <button v-for="tb in tabs" :key="tb.id" class="flex items-center gap-1 rounded-lg px-3 py-1 text-sm" :class="tab === tb.id ? 'bg-white text-brand-600 shadow-sm dark:bg-ink-900' : 'text-ink-600 dark:text-ink-300'" @click="tab = tb.id">
            <component :is="tb.icon" class="h-3.5 w-3.5" />{{ tb.label }}
          </button>
        </div>
        <span class="flex-1"></span>
        <template v-if="p">
          <span v-if="p.visibility === 'pending'" class="rounded-lg bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">案例库审核中</span>
          <a v-else-if="p.visibility === 'public'" :href="`/w/${p.id}`" target="_blank" class="rounded-lg bg-emerald-50 px-2 py-1 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">已公开</a>
          <button class="btn-ghost btn-sm" :disabled="!isReady" title="复制分享链接" @click="share"><Link2 class="h-3.5 w-3.5" /></button>
          <button class="btn-ghost btn-sm" title="版本历史" @click="versionsOpen = true"><History class="h-3.5 w-3.5" /><span class="hidden 2xl:inline">版本</span></button>
          <button class="btn-ghost btn-sm" title="导出 HTML（带配音和字幕的单个网页文件）" :disabled="!isReady || exporting" @click="doExport"><Download class="h-3.5 w-3.5" /><span class="hidden 2xl:inline">导出 HTML</span></button>
          <button class="btn-ghost btn-sm" disabled title="视频导出（MP4）即将上线"><Film class="h-3.5 w-3.5" /><span class="hidden 2xl:inline">导出视频</span></button>
          <button v-if="p.visibility === 'private' || p.visibility === 'rejected'" class="btn-primary btn-sm" :disabled="!isReady" @click="publishOpen = true"><Send class="h-3.5 w-3.5" />发布</button>
          <button v-else class="btn-ghost btn-sm" @click="unpublish">撤回</button>
        </template>
      </div>

      <div class="thin-scroll min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
        <div v-if="!p" class="flex h-full items-center justify-center text-ink-400"><Loader2 class="h-6 w-6 animate-spin" /></div>

        <!-- preview -->
        <template v-else-if="tab === 'preview'">
          <div class="mx-auto" :style="{ maxWidth: previewMax }">
            <div class="relative w-full overflow-hidden rounded-xl bg-ink-900 shadow-soft" :style="{ aspectRatio: `${p.width} / ${p.height}` }">
              <FilmPlayer v-if="hasCode" ref="player" :spec="playable" :audio-url="audioUrl" with-key @error="onPlayerError" @time="(x) => (now = x)" />
              <div v-else class="absolute inset-0 flex flex-col items-center justify-center gap-3 px-6 text-center text-ink-300">
                <template v-if="p.status === 'running'">
                  <Loader2 class="h-8 w-8 animate-spin text-brand-400" />
                  <p class="text-sm">{{ stageText }}</p>
                  <p class="text-xs text-ink-500">可以离开这个页面，生成在服务器上继续，完成后在「历史创作」里找到它。</p>
                </template>
                <template v-else-if="p.status === 'questions'">
                  <MessageCircleQuestion class="h-8 w-8 text-brand-400" />
                  <p class="text-sm">AI 有几个问题想先确认，请在右边回答</p>
                </template>
                <template v-else>
                  <AlertCircle class="h-8 w-8 text-red-400" />
                  <p class="text-sm">{{ p.error || '还没有画面' }}</p>
                </template>
              </div>
            </div>

            <AgentPanel
              class="mt-4 flex h-[560px] overflow-hidden rounded-2xl border border-ink-200 lg:hidden dark:border-ink-800"
              :project="p"
              :events="events"
              :busy="busy"
              :voices="voices"
              :focus-scene="p?.spec?.scenes?.[currentScene]?.id || ''"
              @send="send"
              @answer="answer"
              @stop="stop"
              @resume="resume"
              @voice="setVoice"
            />

            <div v-if="p.spec?.scenes?.length > 1" class="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
              <button
                v-for="(sc, i) in p.spec.scenes"
                :key="sc.id"
                class="rounded-xl border bg-white p-2.5 text-left text-xs transition hover:border-brand-300 dark:bg-ink-900"
                :class="currentScene === i ? 'border-brand-500 ring-1 ring-brand-500/40' : 'border-ink-200 dark:border-ink-800'"
                @click="seekScene(i)"
              >
                <span class="flex items-center gap-1.5">
                  <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-ink-100 font-semibold dark:bg-ink-800">{{ i + 1 }}</span>
                  <span class="truncate font-medium">{{ sc.title }}</span>
                </span>
                <span class="mt-1 flex items-center gap-1 text-ink-500">
                  {{ sc.duration.toFixed(1) }}s
                  <Loader2 v-if="!sc.code && p.status === 'running'" class="h-3 w-3 animate-spin text-brand-500" />
                  <span v-else-if="sc.error || sceneErrors[sc.id]" class="text-red-500">出错</span>
                </span>
              </button>
            </div>
          </div>
        </template>

        <!-- script -->
        <ScriptTab v-else-if="tab === 'script'" :project="p" :busy="busy" @save="saveScript" />

        <!-- code -->
        <div v-else class="mx-auto max-w-4xl space-y-3">
          <p v-if="!p.spec" class="text-sm text-ink-500">还没有代码。</p>
          <details v-for="(sc, i) in p.spec?.scenes || []" :key="sc.id" class="card overflow-hidden" :open="i === 0">
            <summary class="flex cursor-pointer items-center gap-2 px-4 py-3 text-sm font-medium">
              <FileCode2 class="h-4 w-4 text-brand-500" />scenes/{{ sc.id }}.js
              <span class="text-xs font-normal text-ink-500">{{ sc.title }} · {{ (sc.code || '').length }} 字符</span>
              <span class="flex-1"></span>
              <button class="btn-ghost btn-sm" @click.prevent="copy(sc.code)">复制</button>
            </summary>
            <pre class="thin-scroll max-h-[60vh] overflow-auto border-t border-ink-100 bg-ink-950 p-4 text-xs leading-5 text-ink-100 dark:border-ink-800"><code>{{ sc.code || '// 还在生成…' }}</code></pre>
          </details>
        </div>
      </div>
    </section>

    <!-- right: the agent -->
    <AgentPanel
      class="hidden w-[400px] shrink-0 border-l border-ink-200 lg:flex dark:border-ink-800"
      :project="p"
      :events="events"
      :busy="busy"
      :voices="voices"
      :focus-scene="p?.spec?.scenes?.[currentScene]?.id || ''"
      @send="send"
      @answer="answer"
      @stop="stop"
      @resume="resume"
      @voice="setVoice"
    />

    <PublishDialog v-if="publishOpen && p" :project="p" :categories="categories" @close="publishOpen = false" @published="(x) => ((p = x), (publishOpen = false))" />
    <VersionsDialog v-if="versionsOpen && p" :project-id="p.id" :busy="busy" @close="versionsOpen = false" @restored="onRestored" />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircle, Code2, Download, FileCode2, FileText, Film, History, Link2, Loader2, MessageCircleQuestion, Play, Send } from 'lucide-vue-next'
import AgentPanel from '../components/AgentPanel.vue'
import FilmPlayer from '../components/FilmPlayer.vue'
import HistorySidebar from '../components/HistorySidebar.vue'
import PublishDialog from '../components/PublishDialog.vue'
import ScriptTab from '../components/ScriptTab.vue'
import VersionsDialog from '../components/VersionsDialog.vue'
import { api, catalog, session, signIn } from '../lib/api'
import { exportHtml } from '../lib/exportHtml'
import { toast, toastError } from '../lib/toast'

const route = useRoute()
const router = useRouter()
const id = computed(() => String(route.params.id))
const p = ref(null)
const events = ref([])
const tab = ref('preview')
const now = ref(0)
const player = ref(null)
const history = ref(null)
const publishOpen = ref(false)
const versionsOpen = ref(false)
const exporting = ref(false)
const voices = ref([])
const categories = ref([])
const sceneErrors = reactive({})
const repaired = new Set()

const tabs = [
  { id: 'preview', label: '预览', icon: Play },
  { id: 'script', label: '脚本', icon: FileText },
  { id: 'code', label: '代码', icon: Code2 }
]
const busy = computed(() => p.value?.status === 'running')
const isReady = computed(() => p.value?.status === 'ready')
const hasCode = computed(() => p.value?.spec?.scenes?.some((sc) => sc.code))
// Play the scenes that already have code (a project can be previewed while the rest is being made).
const playable = computed(() => {
  const spec = p.value?.spec
  if (!spec) return null
  return { ...spec, scenes: spec.scenes.filter((sc) => sc.code) }
})
const previewMax = computed(() => (p.value && p.value.height > p.value.width ? '420px' : '1100px'))
const currentScene = computed(() => {
  const scenes = playable.value?.scenes || []
  let at = 0
  let idx = 0
  scenes.forEach((sc, i) => {
    if (now.value >= at) idx = i
    at += sc.duration
  })
  const sc = scenes[idx]
  return sc ? p.value.spec.scenes.findIndex((x) => x.id === sc.id) : 0
})
const stageText = computed(() => ({ questions: '正在理解你的需求…', script: '正在写脚本和分镜…', voice: '正在配音…', code: '正在制作动画…', plan: '正在理解修改意见…', repair: '正在修复…' })[p.value?.stage] || 'AI 正在工作…')

const audioUrl = (sc) => (sc.audio?.file ? `/api/v1/video/projects/${p.value.id}/audio/${sc.audio.file}` : '')

let lastEvent = 0
let timer = 0
async function refresh() {
  try {
    const [proj, ev] = await Promise.all([api(`/projects/${id.value}`), api(`/projects/${id.value}/events?after=${lastEvent}`)])
    p.value = proj
    if (ev.events.length) {
      events.value = [...events.value, ...ev.events]
      lastEvent = ev.events[ev.events.length - 1].id
    }
  } catch (err) {
    if (err.status === 404) {
      toast('找不到这个作品')
      router.replace('/')
    } else if (err.status === 401) {
      if (!(await signIn().catch(() => false))) router.replace('/')
    }
  }
}
function schedule() {
  clearTimeout(timer)
  const delay = busy.value ? 2000 : 15000
  timer = setTimeout(async () => {
    if (document.visibilityState === 'visible') await refresh()
    schedule()
  }, delay)
}
watch(busy, (b, was) => {
  if (was && !b) history.value?.load()
  schedule()
})

async function act(path, body, okText) {
  try {
    p.value = await api(`/projects/${id.value}/${path}`, { method: 'POST', body: body || {} })
    if (okText) toast(okText)
    await refresh()
    schedule()
  } catch (err) {
    toastError(err)
  }
}
const send = (text) => act('message', { text, scene: p.value?.spec?.scenes?.[currentScene.value]?.id || '' })
const answer = (answers) => act('answer', { answers })
const resume = () => act('resume')
const setVoice = (voice) => act('voice', { voice }, '正在用新音色重新配音')
async function stop() {
  try {
    await api(`/projects/${id.value}/stop`, { method: 'POST', body: {} })
    setTimeout(refresh, 800)
  } catch (err) {
    toastError(err)
  }
}
async function saveScript(scenes) {
  try {
    p.value = await api(`/projects/${id.value}/script`, { method: 'PUT', body: { scenes } })
    toast('已保存，正在按新脚本重新配音和制作')
    tab.value = 'preview'
    await refresh()
    schedule()
  } catch (err) {
    toastError(err)
  }
}
async function unpublish() {
  try {
    p.value = await api(`/projects/${id.value}/unpublish`, { method: 'POST', body: {} })
    toast('已从案例库撤回')
  } catch (err) {
    toastError(err)
  }
}
function onRestored(next) {
  p.value = next
  versionsOpen.value = false
  toast('已回退')
  refresh()
}

// A scene that throws while playing is sent back to the agent once (per version of its code).
function onPlayerError(e) {
  if (!e.scene) return
  sceneErrors[e.scene] = e.message
  const sc = p.value?.spec?.scenes?.find((x) => x.id === e.scene)
  if (!sc || busy.value) return
  const sig = `${sc.id}:${(sc.code || '').length}`
  if (repaired.has(sig)) return
  repaired.add(sig)
  toast(`分镜「${sc.title}」播放出错，AI 正在自动修复`)
  act('repair', { scene: sc.id, error: e.message })
}
watch(
  () => p.value?.spec?.scenes?.map((s) => s.code).join('|'),
  () => Object.keys(sceneErrors).forEach((k) => delete sceneErrors[k])
)

function seekScene(i) {
  const scenes = playable.value?.scenes || []
  const target = p.value.spec.scenes[i]
  let at = 0
  for (const sc of scenes) {
    if (sc.id === target.id) break
    at += sc.duration
  }
  player.value?.seek(at + 0.01)
}
function share() {
  const url = p.value.visibility === 'public' ? `${location.origin}/w/${p.value.id}` : location.href
  navigator.clipboard?.writeText(url)
  toast(p.value.visibility === 'public' ? '分享链接已复制' : '链接已复制（作品发布到案例库后别人才能打开）')
}
function copy(text) {
  navigator.clipboard?.writeText(text || '')
  toast('已复制')
}
async function doExport() {
  exporting.value = true
  try {
    await exportHtml(p.value, (sc) => audioUrl(sc), true)
  } catch (err) {
    toastError(err)
  } finally {
    exporting.value = false
  }
}

watch(id, () => {
  p.value = null
  events.value = []
  lastEvent = 0
  refresh().then(schedule)
})
onMounted(async () => {
  if (!session.key && !(await signIn().catch(() => false))) {
    router.replace('/')
    return
  }
  catalog().then((c) => ((voices.value = c.voices), (categories.value = c.categories)))
  await refresh()
  schedule()
})
onBeforeUnmount(() => clearTimeout(timer))
</script>
