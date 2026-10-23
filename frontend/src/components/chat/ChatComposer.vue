<template>
  <div
    class="rounded-3xl border bg-white shadow-sm transition dark:bg-dark-900"
    :class="dragging ? 'border-primary-400 ring-2 ring-primary-200 dark:ring-primary-800' : 'border-gray-200 dark:border-dark-700'"
    @dragover.prevent="dragging = true"
    @dragleave.prevent="dragging = false"
    @drop.prevent="onDrop"
  >
    <div v-if="attachments.length" class="flex flex-wrap gap-2 px-3 pt-3">
      <div v-for="a in attachments" :key="a.key" class="group relative h-16 w-16 overflow-hidden rounded-xl bg-gray-100 dark:bg-dark-800">
        <img v-if="a.preview" :src="a.preview" class="h-full w-full object-cover" alt="" />
        <div v-if="a.uploading" class="absolute inset-0 flex items-center justify-center bg-black/30">
          <span class="h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
        </div>
        <button
          type="button"
          class="absolute right-0.5 top-0.5 rounded-full bg-black/60 p-0.5 text-white opacity-90 hover:bg-black/80"
          :aria-label="t('chat.removeImage')"
          @click="remove(a.key)"
        >
          <Icon name="x" size="xs" />
        </button>
      </div>
    </div>
    <textarea
      ref="box"
      v-model="text"
      rows="1"
      class="block max-h-52 w-full resize-none border-0 bg-transparent px-4 pb-1 pt-3.5 text-[15px] leading-relaxed text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-0 dark:text-dark-50"
      :placeholder="mode === 'image' ? t('chat.drawPlaceholder') : t('chat.placeholder')"
      data-testid="chat-input"
      @input="autosize"
      @keydown.enter="onEnter"
      @paste="onPaste"
    ></textarea>
    <div class="flex items-center gap-1.5 px-2.5 pb-2.5 pt-1">
      <button type="button" class="composer-btn" :title="t('chat.attach')" :disabled="attachments.length >= maxFiles" data-testid="chat-attach" @click="picker?.click()">
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
          <path stroke-linecap="round" stroke-linejoin="round" d="M18.375 12.739l-7.693 7.693a4.5 4.5 0 01-6.364-6.364l10.94-10.94A3 3 0 1119.5 7.372L8.552 18.32m.009-.01l-.01.01m5.699-9.941l-7.81 7.81a1.5 1.5 0 002.112 2.13" />
        </svg>
      </button>
      <input ref="picker" type="file" accept="image/png,image/jpeg,image/webp,image/gif" multiple class="hidden" @change="onPick" />
      <div class="flex rounded-full bg-gray-100 p-0.5 text-xs dark:bg-dark-800">
        <button type="button" class="mode-btn" :class="mode === 'text' ? 'mode-on' : ''" data-testid="chat-mode-text" @click="setMode('text')">{{ t('chat.modeText') }}</button>
        <button
          type="button"
          class="mode-btn"
          :class="[mode === 'image' ? 'mode-on' : '', canDraw ? '' : 'cursor-not-allowed opacity-50']"
          :title="canDraw ? '' : t('chat.drawNeedsGpt')"
          data-testid="chat-mode-image"
          @click="setMode('image')"
        >
          {{ t('chat.modeImage') }}
        </button>
      </div>
      <select
        v-if="mode === 'image'"
        :value="imageSize"
        class="rounded-full border-0 bg-gray-100 py-1 pl-3 pr-7 text-xs text-gray-600 focus:ring-1 focus:ring-primary-400 dark:bg-dark-800 dark:text-dark-300"
        data-testid="chat-image-size"
        @change="$emit('update:imageSize', ($event.target as HTMLSelectElement).value)"
      >
        <option value="1024x1024">{{ t('chat.sizeSquare') }}</option>
        <option value="1536x1024">{{ t('chat.sizeLandscape') }}</option>
        <option value="1024x1536">{{ t('chat.sizePortrait') }}</option>
      </select>
      <div class="flex-1"></div>
      <button
        v-if="busy"
        type="button"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-gray-900 text-white transition hover:bg-gray-700 dark:bg-white dark:text-gray-900"
        :title="t('chat.stop')"
        data-testid="chat-stop"
        @click="$emit('stop')"
      >
        <span class="h-3 w-3 rounded-sm bg-current"></span>
      </button>
      <button
        v-else
        type="button"
        class="flex h-9 w-9 items-center justify-center rounded-full bg-primary-600 text-white transition hover:bg-primary-700 disabled:bg-gray-200 disabled:text-gray-400 dark:disabled:bg-dark-700"
        :disabled="!canSend"
        :title="t('chat.send')"
        data-testid="chat-send"
        @click="submit"
      >
        <Icon name="arrowUp" size="sm" :stroke-width="2.4" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { fileURL, uploadFile } from '@/api/chat'

const props = defineProps<{
  busy: boolean
  canDraw: boolean
  mode: 'text' | 'image'
  imageSize: string
  /** No key or model yet. */
  disabled?: boolean
}>()
const emit = defineEmits<{
  (e: 'send', payload: { content: string; files: string[] }): void
  (e: 'stop'): void
  (e: 'update:mode', mode: 'text' | 'image'): void
  (e: 'update:imageSize', size: string): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const maxFiles = 4
const maxEdge = 2048

interface Attachment {
  key: number
  name: string
  preview: string
  uploading: boolean
  ownPreview: boolean
}

const text = ref('')
const attachments = ref<Attachment[]>([])
const box = ref<HTMLTextAreaElement>()
const picker = ref<HTMLInputElement>()
const dragging = ref(false)
let seq = 0

const uploading = computed(() => attachments.value.some((a) => a.uploading))
const canSend = computed(
  () => !props.disabled && !uploading.value && (text.value.trim() !== '' || (props.mode === 'text' && attachments.value.length > 0))
)

function setMode(m: 'text' | 'image') {
  if (m === 'image' && !props.canDraw) {
    appStore.showInfo(t('chat.drawNeedsGpt'))
    return
  }
  emit('update:mode', m)
}

function autosize() {
  const el = box.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 208)}px`
}

function onEnter(e: KeyboardEvent) {
  // Enter sends; Shift+Enter is a new line; Enter that confirms an IME candidate does nothing.
  if (e.shiftKey || e.isComposing || e.keyCode === 229) return
  e.preventDefault()
  if (!props.busy) submit()
}

function submit() {
  if (!canSend.value) return
  emit('send', { content: text.value.trim(), files: attachments.value.map((a) => a.name) })
}

/** Clears the box once the turn was accepted. */
function clear() {
  text.value = ''
  for (const a of attachments.value) if (a.ownPreview) URL.revokeObjectURL(a.preview)
  attachments.value = []
  void nextTick(autosize)
}

/** Puts text (and pictures already on the server) back, e.g. when a turn could not start. */
function restore(content: string, files: string[] = []) {
  text.value = content
  attachments.value = []
  files.forEach(addExisting)
  void nextTick(autosize)
}

function focus() {
  box.value?.focus()
}

function addExisting(name: string) {
  if (attachments.value.length >= maxFiles || attachments.value.some((a) => a.name === name)) return
  const a: Attachment = { key: ++seq, name, preview: '', uploading: false, ownPreview: false }
  attachments.value.push(a)
  fileURL(name)
    .then((url) => {
      const item = attachments.value.find((x) => x.key === a.key)
      if (item) item.preview = url
    })
    .catch(() => undefined)
}

function remove(key: number) {
  const a = attachments.value.find((x) => x.key === key)
  if (a?.ownPreview) URL.revokeObjectURL(a.preview)
  attachments.value = attachments.value.filter((x) => x.key !== key)
}

// Large photos are scaled to 2048px and sent as JPEG, which keeps uploads and model input small.
async function shrink(file: File): Promise<Blob> {
  if (file.type === 'image/gif') return file
  try {
    const bmp = await createImageBitmap(file)
    const scale = Math.min(1, maxEdge / Math.max(bmp.width, bmp.height))
    if (scale === 1 && file.size <= 3 << 20) {
      bmp.close()
      return file
    }
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(bmp.width * scale)
    canvas.height = Math.round(bmp.height * scale)
    const ctx = canvas.getContext('2d')!
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height)
    bmp.close()
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.9))
    return blob || file
  } catch {
    return file
  }
}

async function addFiles(files: File[]) {
  const images = files.filter((f) => f.type.startsWith('image/'))
  if (!images.length) return
  const room = maxFiles - attachments.value.length
  if (images.length > room) appStore.showInfo(t('chat.maxImages', { n: maxFiles }))
  for (const file of images.slice(0, Math.max(0, room))) {
    const a: Attachment = { key: ++seq, name: '', preview: URL.createObjectURL(file), uploading: true, ownPreview: true }
    attachments.value.push(a)
    try {
      const blob = await shrink(file)
      const res = await uploadFile(blob, blob === file ? file.name : 'image.jpg')
      const item = attachments.value.find((x) => x.key === a.key)
      if (item) {
        item.name = res.name
        item.uploading = false
      }
    } catch (err) {
      remove(a.key)
      appStore.showError((err as { message?: string })?.message || t('chat.uploadFailed'))
    }
  }
}

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  void addFiles(Array.from(input.files || []))
  input.value = ''
}

function onPaste(e: ClipboardEvent) {
  const files = Array.from(e.clipboardData?.files || [])
  if (files.some((f) => f.type.startsWith('image/'))) {
    e.preventDefault()
    void addFiles(files)
  }
}

function onDrop(e: DragEvent) {
  dragging.value = false
  void addFiles(Array.from(e.dataTransfer?.files || []))
}

onBeforeUnmount(() => {
  for (const a of attachments.value) if (a.ownPreview) URL.revokeObjectURL(a.preview)
})

defineExpose({ clear, restore, focus, addExisting, setText: (s: string) => ((text.value = s), nextTick(autosize)) })
</script>

<style scoped>
.composer-btn {
  @apply flex h-9 w-9 items-center justify-center rounded-full text-gray-500 transition hover:bg-gray-100 hover:text-gray-800 disabled:opacity-40 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-dark-100;
}
.mode-btn {
  @apply rounded-full px-3 py-1 text-gray-500 transition dark:text-dark-400;
}
.mode-on {
  @apply bg-white text-gray-900 shadow-sm dark:bg-dark-600 dark:text-white;
}
</style>
