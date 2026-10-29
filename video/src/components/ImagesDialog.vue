<template>
  <Modal title="更换图片" wide @close="$emit('close')">
    <p class="text-sm leading-6 text-ink-500">
      换掉、删掉或新加图片后，AI 会看新图，只重做用到这些图片的分镜；旁白和配音不变。换之前的样子会自动存进「版本」，随时可以回退。
    </p>

    <p v-if="!rows.length" class="mt-4 rounded-xl bg-ink-50 px-4 py-3 text-sm text-ink-500 dark:bg-ink-800/60">这个作品还没有图片。可以加几张产品图、截图或照片，AI 会把它们放进合适的分镜。</p>
    <div v-else class="mt-4 space-y-2" data-testid="images-current">
      <div v-for="r in rows" :key="r.id" class="flex gap-3 rounded-xl border p-2.5" :class="r.removed ? 'border-dashed border-ink-200 opacity-60 dark:border-ink-700' : r.next ? 'border-brand-300 bg-brand-50/40 dark:border-brand-700 dark:bg-brand-900/10' : 'border-ink-200 dark:border-ink-800'">
        <div class="relative h-20 w-28 shrink-0 overflow-hidden rounded-lg bg-ink-100 dark:bg-ink-800">
          <img :src="r.next?.preview || r.url" alt="" class="h-full w-full object-contain" :class="{ 'opacity-40': r.next && r.next.status !== 'done' }" />
          <span class="absolute left-1 top-1 rounded bg-black/60 px-1 text-[10px] leading-4 text-white">{{ label(r.id) }}</span>
          <span v-if="r.next" class="absolute right-1 top-1 rounded bg-brand-500 px-1 text-[10px] leading-4 text-white">新图</span>
          <div v-if="r.next?.status === 'uploading'" class="absolute inset-x-1 bottom-1 h-1 overflow-hidden rounded-full bg-white/60">
            <div class="h-full bg-brand-500 transition-all" :style="{ width: `${Math.round(r.next.progress * 100)}%` }"></div>
          </div>
          <div v-if="r.next?.status === 'error'" class="absolute inset-0 flex items-center justify-center bg-red-500/70 text-[11px] font-medium text-white">上传失败</div>
        </div>
        <div class="min-w-0 flex-1 text-xs leading-5">
          <p class="font-medium text-ink-700 dark:text-ink-200">
            {{ r.scenes.length ? `用在分镜 ${r.scenes.join('、')}` : '没有分镜用到它' }}
            <span v-if="r.removed" class="ml-1 text-red-500">· 将删除</span>
            <span v-else-if="r.next" class="ml-1 text-brand-600">· 将换成新图</span>
          </p>
          <p v-if="r.desc" class="mt-0.5 line-clamp-2 text-ink-500" :title="r.desc">{{ r.desc }}</p>
          <div class="mt-1.5 flex flex-wrap gap-1.5">
            <template v-if="!r.removed">
              <button class="btn-ghost btn-sm" data-testid="image-replace" @click="pickFor(r)"><RefreshCw class="h-3.5 w-3.5" />{{ r.next ? '重选' : '换一张' }}</button>
              <button v-if="r.next" class="btn-ghost btn-sm" @click="r.next = null">不换了</button>
              <button class="btn-ghost btn-sm text-red-600" @click="r.removed = true"><Trash2 class="h-3.5 w-3.5" />删除</button>
            </template>
            <button v-else class="btn-ghost btn-sm" @click="r.removed = false"><Undo2 class="h-3.5 w-3.5" />恢复</button>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-4">
      <p class="mb-2 text-sm font-medium">新加图片</p>
      <ImageAttach :items="added" :max="room" :first="nextIndex" />
    </div>

    <label class="mt-4 block text-sm font-medium" for="images-note">怎么用新图（可选）</label>
    <textarea id="images-note" v-model="note" rows="2" maxlength="500" class="input mt-1.5 w-full resize-none text-sm" placeholder="比如：新图用在开场和结尾；只露出笔身，不要截图里的文字标签"></textarea>

    <p class="mt-3 rounded-xl bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
      <template v-if="changed">预计重做 {{ redoText }}，按作品所用模型从余额扣费（Claude 每个分镜约 $0.5–1.5，GPT 约 $0.1–0.3）。</template>
      <template v-else>换一张、删除或新加图片后，这里会显示要重做几个分镜。</template>
    </p>

    <div class="mt-5 flex justify-end gap-2">
      <button class="btn-ghost" @click="$emit('close')">取消</button>
      <button class="btn-primary" :disabled="!ready || saving" data-testid="images-submit" @click="submit"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" />开始换图</button>
    </div>
    <input ref="input" type="file" accept="image/jpeg,image/png,image/webp,image/gif,image/heic,image/heif" class="sr-only" @change="onPick" />
  </Modal>
</template>

<script setup>
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { Loader2, RefreshCw, Trash2, Undo2 } from 'lucide-vue-next'
import Modal from './Modal.vue'
import ImageAttach from './ImageAttach.vue'
import { api } from '../lib/api'
import { isImageFile, stageImage } from '../lib/stageImage'
import { toast, toastError } from '../lib/toast'

const MAX = 9
const props = defineProps({ project: { type: Object, required: true } })
const emit = defineEmits(['close', 'changed'])

const label = (id) => `图${String(id).replace(/^img/, '')}`
const scenes = computed(() => props.project.spec?.scenes || [])
// The spec's pictures carry the descriptions; options.images is the list before a script exists.
const rows = reactive(
  (props.project.spec?.images?.length ? props.project.spec.images : props.project.options?.images || []).map((img) => ({
    id: img.id,
    url: img.url,
    desc: img.desc || '',
    scenes: scenes.value.map((sc, i) => ((sc.images || []).includes(img.id) ? i + 1 : 0)).filter(Boolean),
    next: null, // the replacement picture being uploaded
    removed: false
  }))
)
const added = reactive([])
const note = ref('')
const saving = ref(false)
const input = ref(null)
let target = null

const nextIndex = computed(() => Math.max(0, ...rows.map((r) => Number(String(r.id).replace(/^img/, '')) || 0)) + 1)
const room = computed(() => Math.max(0, MAX - rows.filter((r) => !r.removed).length))
const changed = computed(() => rows.some((r) => r.removed || r.next) || added.length > 0)
const uploading = computed(() => rows.some((r) => !r.removed && r.next && r.next.status !== 'done') || added.some((it) => it.status !== 'done'))
const ready = computed(() => changed.value && !uploading.value)
// Scenes that show a replaced or removed picture are certain; where new pictures go, the AI decides.
const redoText = computed(() => {
  const sure = new Set()
  rows.filter((r) => r.removed || r.next).forEach((r) => r.scenes.forEach((n) => sure.add(n)))
  const parts = []
  if (sure.size) parts.push(`${sure.size} 个分镜（${[...sure].sort((a, b) => a - b).join('、')}）`)
  if (added.length) parts.push(sure.size ? '加上 AI 放新图的分镜' : 'AI 放新图的几个分镜')
  return parts.join('，') || '少量分镜'
})

function pickFor(r) {
  target = r
  input.value?.click()
}
async function onPick(e) {
  const file = Array.from(e.target.files || []).find(isImageFile)
  e.target.value = ''
  const r = target
  if (!file || !r) return
  if (r.next) URL.revokeObjectURL(r.next.preview)
  r.next = { preview: URL.createObjectURL(file), status: 'uploading', progress: 0, token: '' }
  const it = r.next
  try {
    const res = await stageImage(file, (p) => (it.progress = p))
    Object.assign(it, { status: 'done', token: res.token, progress: 1 })
  } catch (err) {
    Object.assign(it, { status: 'error' })
    toast(err.message || '图片上传失败', 'error', 6000)
  }
}

async function submit() {
  const replace = {}
  rows.filter((r) => !r.removed && r.next?.token).forEach((r) => (replace[r.id] = r.next.token))
  const body = {
    replace,
    add: added.map((it) => it.token),
    remove: rows.filter((r) => r.removed).map((r) => r.id),
    note: note.value.trim()
  }
  saving.value = true
  try {
    const p = await api(`/projects/${props.project.id}/images`, { method: 'POST', body })
    toast('开始换图，AI 会重做相关分镜')
    emit('changed', p)
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}

onBeforeUnmount(() => rows.forEach((r) => r.next && URL.revokeObjectURL(r.next.preview)))
</script>
