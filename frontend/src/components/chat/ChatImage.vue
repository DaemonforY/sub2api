<template>
  <div class="relative overflow-hidden rounded-xl bg-gray-100 dark:bg-dark-800" :class="src && !failed ? boxClass : placeholderClass || boxClass">
    <img
      v-if="src && !failed"
      :src="src"
      :alt="alt"
      class="h-full w-full cursor-zoom-in object-cover"
      @click="$emit('open', name)"
      @load="$emit('loaded', src)"
      @error="onError"
    />
    <button
      v-else-if="failed"
      type="button"
      class="flex h-full w-full flex-col items-center justify-center gap-1 p-2 text-center text-xs text-gray-400 hover:text-gray-600"
      @click="load"
    >
      {{ t('chat.imageGone') }}
      <span class="text-primary-600 dark:text-primary-400">{{ t('chat.retry') }}</span>
    </button>
    <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2 text-xs text-gray-400">
      <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-400 border-t-transparent"></span>
      <span v-if="placeholderClass">{{ t('chat.imageLoading') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { fileURL, forgetFileURL } from '@/api/chat'

const props = withDefaults(
  defineProps<{
    name: string
    alt?: string
    boxClass?: string
    /** Size of the box while loading (a big picture needs a big placeholder; it has no size yet). */
    placeholderClass?: string
  }>(),
  { alt: '', boxClass: 'h-40 w-40', placeholderClass: '' }
)
defineEmits<{ (e: 'open', name: string): void; (e: 'loaded', src: string): void }>()
const { t } = useI18n()
const src = ref('')
const failed = ref(false)

async function load() {
  const name = props.name
  src.value = ''
  failed.value = false
  try {
    const url = await fileURL(name)
    if (name === props.name) src.value = url
  } catch {
    if (name === props.name) failed.value = true
  }
}

function onError() {
  forgetFileURL(props.name)
  failed.value = true
}

watch(() => props.name, load, { immediate: true })
</script>
