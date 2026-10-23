<template>
  <div class="relative overflow-hidden rounded-xl bg-gray-100 dark:bg-dark-800" :class="boxClass">
    <img v-if="src" :src="src" :alt="alt" class="h-full w-full cursor-zoom-in object-cover" @click="$emit('open', src)" @load="$emit('loaded', src)" />
    <div v-else-if="failed" class="flex h-full w-full items-center justify-center p-2 text-center text-xs text-gray-400">{{ t('chat.imageGone') }}</div>
    <div v-else class="h-full w-full animate-pulse"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { fileURL } from '@/api/chat'

const props = withDefaults(defineProps<{ name: string; alt?: string; boxClass?: string }>(), { alt: '', boxClass: 'h-40 w-40' })
defineEmits<{ (e: 'open', src: string): void; (e: 'loaded', src: string): void }>()
const { t } = useI18n()
const src = ref('')
const failed = ref(false)

watch(
  () => props.name,
  async (name) => {
    src.value = ''
    failed.value = false
    try {
      src.value = await fileURL(name)
    } catch {
      failed.value = true
    }
  },
  { immediate: true }
)
</script>
