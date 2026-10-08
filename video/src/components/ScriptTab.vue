<template>
  <div class="mx-auto max-w-4xl">
    <div v-if="!spec" class="card p-8 text-center text-sm text-ink-500">{{ project.status === 'running' ? '脚本正在写…' : '还没有脚本' }}</div>
    <template v-else>
      <div class="card flex flex-wrap items-center gap-4 px-5 py-3">
        <span class="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-50 text-brand-600 dark:bg-brand-900/30"><FileText class="h-5 w-5" /></span>
        <span class="font-semibold">{{ project.mode === 'film' ? '视频脚本' : '动画说明' }}</span>
        <span class="flex items-center gap-1 text-sm text-ink-500"><LayoutGrid class="h-4 w-4" />{{ spec.scenes.length }} 个分镜</span>
        <span class="flex items-center gap-1 text-sm text-ink-500"><Clock class="h-4 w-4" />{{ total.toFixed(1) }}s</span>
        <span v-if="chars" class="flex items-center gap-1 text-sm text-ink-500"><Type class="h-4 w-4" />{{ chars }} 字</span>
        <span class="flex-1"></span>
        <div class="flex rounded-xl bg-ink-100 p-0.5 text-sm dark:bg-ink-800">
          <button class="rounded-lg px-3 py-1" :class="view === 'md' ? 'bg-white text-brand-600 shadow-sm dark:bg-ink-900' : ''" @click="view = 'md'">文档</button>
          <button class="rounded-lg px-3 py-1" :class="view === 'json' ? 'bg-white text-brand-600 shadow-sm dark:bg-ink-900' : ''" @click="view = 'json'">JSON</button>
        </div>
        <button v-if="project.mode === 'film' && view === 'md' && !editing" class="btn-ghost btn-sm" :disabled="busy || project.status !== 'ready'" @click="startEdit"><Pencil class="h-3.5 w-3.5" />编辑脚本</button>
      </div>

      <pre v-if="view === 'json'" class="thin-scroll card mt-4 max-h-[70vh] overflow-auto p-4 text-xs leading-5">{{ json }}</pre>

      <article v-else class="card mt-4 overflow-hidden">
        <div class="h-1.5 bg-gradient-to-r from-brand-500 via-amber-400 to-rose-400"></div>
        <div class="p-6 sm:p-8">
          <h2 class="text-2xl font-bold">{{ spec.title }}</h2>
          <p v-if="spec.summary" class="mt-3 leading-7 text-ink-600 dark:text-ink-300">{{ spec.summary }}</p>

          <template v-if="project.mode === 'film'">
            <h3 class="mt-8 text-lg font-semibold">视频定位</h3>
            <table class="mt-3 w-full overflow-hidden rounded-xl text-sm">
              <tbody>
                <tr v-for="row in facts" :key="row[0]" class="border-b border-ink-100 last:border-0 dark:border-ink-800">
                  <td class="w-32 bg-ink-50 px-4 py-2.5 text-ink-500 dark:bg-ink-950">{{ row[0] }}</td>
                  <td class="px-4 py-2.5">{{ row[1] }}</td>
                </tr>
              </tbody>
            </table>

            <h3 class="mt-8 text-lg font-semibold">配音脚本</h3>
            <div class="mt-3 whitespace-pre-wrap rounded-xl border border-brand-200 bg-brand-50/50 p-4 leading-8 dark:border-brand-900/40 dark:bg-brand-900/10">“{{ fullNarration }}”</div>
          </template>

          <h3 class="mt-8 text-lg font-semibold">分镜</h3>
          <div class="mt-3 space-y-3">
            <div v-for="(sc, i) in spec.scenes" :key="sc.id" class="rounded-xl border border-ink-200 p-4 dark:border-ink-800">
              <div class="flex items-center gap-2">
                <span class="flex h-6 w-6 items-center justify-center rounded-md bg-ink-900 text-xs font-semibold text-white dark:bg-white dark:text-ink-900">{{ i + 1 }}</span>
                <input v-if="editing" v-model="draft[i].title" class="input py-1" maxlength="40" />
                <span v-else class="font-medium">{{ sc.title }}</span>
                <span class="ml-auto shrink-0 text-xs text-ink-500">{{ sc.duration.toFixed(1) }}s</span>
              </div>
              <template v-if="project.mode === 'film'">
                <p class="mt-3 text-xs font-medium text-ink-500">旁白</p>
                <textarea v-if="editing" v-model="draft[i].narration" rows="3" class="input mt-1 leading-6"></textarea>
                <p v-else class="mt-1 leading-7">{{ sc.narration }}</p>
              </template>
              <p class="mt-3 text-xs font-medium text-ink-500">画面</p>
              <textarea v-if="editing" v-model="draft[i].visual" rows="4" class="input mt-1 leading-6"></textarea>
              <p v-else class="mt-1 whitespace-pre-wrap text-sm leading-6 text-ink-600 dark:text-ink-300">{{ sc.visual }}</p>
            </div>
          </div>
          <div v-if="editing" class="sticky bottom-0 mt-4 flex justify-end gap-2 bg-white/90 py-3 dark:bg-ink-900/90">
            <span class="mr-auto self-center text-xs text-ink-500">只有改动的分镜会重新配音和制作</span>
            <button class="btn-ghost" @click="editing = false">取消</button>
            <button class="btn-primary" :disabled="!changes.length" @click="$emit('save', changes)">保存并重新生成（{{ changes.length }}）</button>
          </div>
        </div>
      </article>
    </template>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Clock, FileText, LayoutGrid, Pencil, Type } from 'lucide-vue-next'

const props = defineProps({ project: { type: Object, required: true }, busy: Boolean })
defineEmits(['save'])
const view = ref('md')
const editing = ref(false)
const draft = ref([])

const spec = computed(() => props.project.spec)
const total = computed(() => spec.value.scenes.reduce((s, sc) => s + sc.duration, 0))
const fullNarration = computed(() => spec.value.scenes.map((sc) => sc.narration).filter(Boolean).join(''))
const chars = computed(() => fullNarration.value.length)
const json = computed(() => JSON.stringify({ ...spec.value, scenes: spec.value.scenes.map(({ code, audio, ...rest }) => ({ ...rest, audio: audio ? { file: audio.file, duration: audio.duration } : undefined })) }, null, 2))
const facts = computed(() => [
  ['目标受众', spec.value.audience || '—'],
  ['表达目的', spec.value.goal || '—'],
  ['语言', spec.value.language === 'en' ? '英文' : '简体中文'],
  ['画幅', `${props.project.width > props.project.height ? '横屏' : '竖屏'} ${props.project.width}×${props.project.height}`],
  ['时长', `约 ${Math.round(total.value)} 秒`]
])
const changes = computed(() =>
  draft.value.filter((d, i) => {
    const sc = spec.value.scenes[i]
    return d.title !== sc.title || d.narration !== (sc.narration || '') || d.visual !== sc.visual
  })
)
function startEdit() {
  draft.value = spec.value.scenes.map((sc) => ({ id: sc.id, title: sc.title, narration: sc.narration || '', visual: sc.visual }))
  editing.value = true
}
</script>
