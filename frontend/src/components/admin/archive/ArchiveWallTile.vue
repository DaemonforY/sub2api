<template>
  <button
    ref="root"
    type="button"
    class="group relative block aspect-square w-full overflow-hidden rounded-lg border border-gray-200 bg-gray-100 dark:border-dark-600 dark:bg-dark-800"
    :class="active ? 'ring-2 ring-primary-500' : ''"
    :title="item.prompt"
    :data-testid="`ca-wall-tile-${item.id}`"
    @click="emit('open')"
  >
    <img v-if="src" :src="src" :alt="t('admin.conversationArchive.transcript.imageAlt', { id: item.id })" class="h-full w-full object-cover" />
    <span v-else-if="failed" class="absolute inset-0 flex items-center justify-center px-1 text-center text-xs text-red-600">
      {{ t('admin.conversationArchive.transcript.imageFailed') }}
    </span>
    <span v-else class="absolute inset-0 animate-pulse bg-gray-200 dark:bg-dark-700" />
    <span
      class="pointer-events-none absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/70 to-transparent px-1.5 pb-1 pt-4 text-left text-[11px] text-white opacity-0 transition-opacity group-hover:opacity-100"
    >
      {{ item.email ? displayEmail(item.email, t) : `#${item.user_id}` }} · {{ formatDateTimeToMinute(item.created_at) }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getImageThumb, type ArchiveWallImage } from '@/api/admin/conversationArchive'
import { displayEmail } from '@/utils/userDisplay'
import { formatDateTimeToMinute } from '@/utils/format'

/**
 * One tile of the image wall: the thumbnail is fetched (with the admin auth header) once the
 * tile scrolls near the viewport, shown as an object URL and revoked on unmount.
 */
const props = defineProps<{ item: ArchiveWallImage; active?: boolean }>()
const emit = defineEmits<{ (e: 'open'): void; (e: 'loaded', src: string): void }>()
const { t } = useI18n()

const root = ref<HTMLElement | null>(null)
const src = ref('')
const failed = ref(false)
let alive = true
let observer: IntersectionObserver | null = null

async function load() {
  try {
    const blob = await getImageThumb(props.item.id)
    const url = URL.createObjectURL(blob)
    if (!alive) {
      URL.revokeObjectURL(url)
      return
    }
    src.value = url
    emit('loaded', url)
  } catch {
    failed.value = true
  }
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined' || !root.value) {
    void load()
    return
  }
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        observer?.disconnect()
        observer = null
        void load()
      }
    },
    { rootMargin: '400px' }
  )
  observer.observe(root.value)
})

onBeforeUnmount(() => {
  alive = false
  observer?.disconnect()
  if (src.value) URL.revokeObjectURL(src.value)
  src.value = ''
})
</script>
