<template>
  <button
    type="button"
    class="flex h-28 w-28 items-center justify-center overflow-hidden rounded border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800"
    :disabled="!src"
    :data-testid="`ca-image-${imageId}`"
    @click="src && emit('open', src)"
  >
    <img v-if="src" :src="src" :alt="t('admin.conversationArchive.transcript.imageAlt', { id: imageId })" class="h-full w-full object-cover" />
    <span v-else-if="failed" class="px-1 text-center text-xs text-red-600">{{ t('admin.conversationArchive.transcript.imageFailed') }}</span>
    <span v-else class="text-xs text-gray-400">…</span>
  </button>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getImage } from '@/api/admin/conversationArchive'

/**
 * An archived image, fetched through the admin-authenticated proxy as a blob and
 * shown as an object URL (the auth token never goes into a URL). The object URL
 * is revoked when the thumbnail unmounts, i.e. when the session changes or the
 * page is left.
 */
const props = defineProps<{ imageId: number }>()
const emit = defineEmits<{ (e: 'open', src: string): void }>()
const { t } = useI18n()

const src = ref('')
const failed = ref(false)
let alive = true

onMounted(async () => {
  try {
    const blob = await getImage(props.imageId)
    const url = URL.createObjectURL(blob)
    if (!alive) {
      URL.revokeObjectURL(url)
      return
    }
    src.value = url
  } catch {
    failed.value = true
  }
})

onBeforeUnmount(() => {
  alive = false
  if (src.value) URL.revokeObjectURL(src.value)
  src.value = ''
})
</script>
