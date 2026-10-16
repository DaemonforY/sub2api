<template>
  <div class="rounded-lg border-l-4 px-3 py-2" :class="roleClass" :data-testid="`ca-block-${block.kind}`">
    <div class="mb-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-gray-500 dark:text-dark-400">
      <span class="font-medium text-gray-700 dark:text-dark-200">{{ roleLabel }}</span>
      <span v-if="block.kind !== 'text' && block.kind !== 'other'" class="rounded bg-white/70 px-1.5 py-0.5 dark:bg-dark-700">{{ t(`admin.conversationArchive.transcript.kinds.${block.kind}`) }}</span>
      <span v-if="block.title" class="break-all font-mono">{{ block.title }}</span>
      <button
        v-if="collapsible"
        type="button"
        class="ml-auto text-primary-600 hover:underline dark:text-primary-400"
        data-testid="ca-block-toggle"
        @click="collapsed = !collapsed"
      >
        {{ collapsed ? expandLabel : t('admin.conversationArchive.transcript.collapse') }}
      </button>
    </div>

    <template v-if="block.kind === 'image'">
      <button v-if="block.imageSrc" type="button" class="block" @click="emit('open-image', block.imageSrc)">
        <img :src="block.imageSrc" :alt="block.title" class="max-h-48 max-w-xs rounded border border-gray-200 object-contain dark:border-dark-600" />
      </button>
      <div v-else class="break-all font-mono text-xs text-gray-700 dark:text-dark-200">{{ block.body || t('admin.conversationArchive.transcript.noContent') }}</div>
    </template>

    <template v-else-if="lineMode">
      <pre v-if="!collapsed" class="max-h-[32rem] overflow-auto whitespace-pre-wrap break-words font-mono text-xs text-gray-800 dark:text-dark-100" data-testid="ca-block-body">{{ block.body || t('admin.conversationArchive.transcript.noContent') }}</pre>
      <div v-else class="truncate font-mono text-xs text-gray-500 dark:text-dark-400">{{ firstLine }}</div>
    </template>

    <pre
      v-else
      class="whitespace-pre-wrap break-words text-sm text-gray-900 dark:text-dark-100"
      :class="block.kind === 'other' ? 'font-mono text-xs' : 'font-sans'"
      data-testid="ca-block-body"
    >{{ shownText || t('admin.conversationArchive.transcript.noContent') }}</pre>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { COLLAPSE_LINES, collapsedByDefault, lineCount, type TranscriptBlock } from '@/utils/archiveTranscript'

/** startCollapsed: always start folded to a one-line preview (used for 系统提示). */
const props = defineProps<{ block: TranscriptBlock; startCollapsed?: boolean }>()
const emit = defineEmits<{ (e: 'open-image', src: string): void }>()
const { t } = useI18n()

const collapsed = ref(props.startCollapsed || collapsedByDefault(props.block))
const isTool = computed(() => props.block.kind === 'tool_call' || props.block.kind === 'tool_output')
/** folds to a single preview line instead of the first 12 lines */
const lineMode = computed(() => isTool.value || !!props.startCollapsed)
const lines = computed(() => lineCount(props.block.body))
const collapsible = computed(() => props.block.kind !== 'image' && (lineMode.value || lines.value > COLLAPSE_LINES))

const expandLabel = computed(() =>
  lineMode.value ? t('admin.conversationArchive.transcript.expand') : t('admin.conversationArchive.transcript.moreLines', { n: lines.value })
)

const shownText = computed(() => {
  if (!collapsed.value) return props.block.body
  return props.block.body.split('\n').slice(0, COLLAPSE_LINES).join('\n') + '\n…'
})

const firstLine = computed(() => {
  const s = props.block.body.trim()
  const i = s.indexOf('\n')
  return (i >= 0 ? s.slice(0, i) : s).slice(0, 200)
})

const KNOWN_ROLES = ['user', 'assistant', 'system', 'developer', 'tool']
const roleLabel = computed(() =>
  KNOWN_ROLES.includes(props.block.role) ? t(`admin.conversationArchive.transcript.roles.${props.block.role}`) : props.block.role
)

const roleClass = computed(() => {
  switch (props.block.role) {
    case 'user':
      return 'border-primary-400 bg-primary-50/70 dark:bg-primary-900/20'
    case 'assistant':
      return 'border-emerald-400 bg-gray-50 dark:bg-dark-800'
    case 'tool':
      return 'border-amber-400 bg-amber-50/60 dark:bg-amber-900/10'
    default:
      return 'border-purple-400 bg-purple-50/60 dark:bg-purple-900/10'
  }
})
</script>
