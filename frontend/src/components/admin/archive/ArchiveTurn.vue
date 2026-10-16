<template>
  <div class="space-y-2" :data-testid="`ca-turn-${turn.id}`">
    <div v-if="turn.reset" class="flex items-center gap-3 py-1 text-xs font-medium text-amber-600 dark:text-amber-400" data-testid="ca-reset">
      <span class="h-px flex-1 bg-amber-300 dark:bg-amber-700"></span>
      <span>{{ t('admin.conversationArchive.transcript.reset') }}</span>
      <span class="h-px flex-1 bg-amber-300 dark:bg-amber-700"></span>
    </div>

    <div
      class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-dark-400"
      :class="highlighted ? 'rounded bg-yellow-50 px-2 py-1 dark:bg-yellow-900/20' : ''"
      data-testid="ca-turn-meta"
    >
      <span class="whitespace-nowrap font-medium text-gray-700 dark:text-dark-200">{{ formatDateTime(turn.created_at) }}</span>
      <span class="font-mono text-gray-900 dark:text-white">{{ turn.model || '–' }}</span>
      <span class="font-mono">{{ turn.inbound_endpoint || '–' }}</span>
      <span>{{ t('admin.conversationArchive.transcript.tokens', { in: turn.input_tokens, out: turn.output_tokens, cached: turn.cache_read_tokens }) }}</span>
      <span v-if="turn.prefix_items > 0">{{ t('admin.conversationArchive.transcript.skipped', { n: turn.prefix_items }) }}</span>
      <span v-if="turn.request_id" class="inline-flex items-center gap-1">
        <span class="max-w-[16rem] truncate font-mono" :title="turn.request_id">{{ turn.request_id }}</span>
        <button
          type="button"
          class="text-primary-600 hover:underline dark:text-primary-400"
          :title="t('admin.conversationArchive.transcript.copyId')"
          data-testid="ca-copy-request-id"
          @click="copyToClipboard(turn.request_id)"
        >
          {{ t('admin.conversationArchive.transcript.copyId') }}
        </button>
      </span>
    </div>

    <div v-if="turn.fetch_error" class="rounded bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300" data-testid="ca-fetch-error">
      {{ t('admin.conversationArchive.transcript.fetchError', { error: turn.fetch_error }) }}
    </div>

    <ArchiveBlock v-if="turn.instructions" :block="systemBlock" start-collapsed data-testid="ca-instructions" />

    <ArchiveBlock v-for="(b, i) in inputBlocks" :key="`in-${i}`" :block="b" @open-image="emit('open-image', $event)" />

    <ArchiveBlock v-for="(b, i) in responseBlocks" :key="`out-${i}`" :block="b" @open-image="emit('open-image', $event)" />

    <div v-if="turn.images?.length" class="flex flex-wrap gap-2 pl-1">
      <ArchiveImage v-for="img in turn.images" :key="img.id" :image-id="img.id" @open="emit('open-image', $event)" />
    </div>

    <!-- Images returned as links: text links only, so opening a transcript never contacts other hosts -->
    <ul v-if="linkImages.length" class="space-y-0.5 pl-1 text-xs" data-testid="ca-image-urls">
      <li v-for="(img, i) in linkImages" :key="`url-${i}`" class="break-all">
        <span v-if="img.source" class="mr-1 text-gray-500 dark:text-dark-400">{{ img.source }}</span>
        <a
          :href="img.url"
          target="_blank"
          rel="noopener noreferrer"
          class="font-mono text-primary-600 hover:underline dark:text-primary-400"
          data-testid="ca-image-url"
        >{{ img.url }}</a>
      </li>
    </ul>

    <div
      v-if="turn.response?.status && turn.response.status !== 'completed'"
      class="text-xs text-amber-600 dark:text-amber-400"
    >
      {{ t('admin.conversationArchive.transcript.responseStatus', { status: turn.response.status }) }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ArchiveTurn } from '@/api/admin/conversationArchive'
import { useClipboard } from '@/composables/useClipboard'
import { normalizeItems, normalizeResponse, type TranscriptBlock } from '@/utils/archiveTranscript'
import { formatDateTime } from '@/utils/format'
import ArchiveBlock from './ArchiveBlock.vue'
import ArchiveImage from './ArchiveImage.vue'

const props = defineProps<{ turn: ArchiveTurn; highlighted?: boolean }>()
const emit = defineEmits<{ (e: 'open-image', src: string): void }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const systemBlock = computed<TranscriptBlock>(() => ({
  role: 'system',
  kind: 'other',
  title: t('admin.conversationArchive.transcript.systemPrompt'),
  body: props.turn.instructions || ''
}))

const inputBlocks = computed(() => normalizeItems(props.turn.new_items))

/** only http(s) links; anything else (javascript:, data: …) is dropped */
const linkImages = computed(() =>
  (props.turn.image_urls || []).filter((u) => !!u && typeof u.url === 'string' && /^https?:\/\//i.test(u.url.trim()))
)

const responseBlocks = computed(() => {
  const blocks = normalizeResponse(props.turn.response)
  // Generated images are archived as separate objects and shown as thumbnails
  // below; don't also inline the base64 copy a raw output item may carry.
  if (props.turn.images?.length) return blocks.filter((b) => b.kind !== 'image' || !b.imageSrc)
  return blocks
})
</script>
