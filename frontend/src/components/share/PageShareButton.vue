<template>
  <button
    type="button"
    class="inline-flex shrink-0 items-center gap-1 rounded-full border border-primary-200 bg-primary-50 px-3 py-1.5 text-xs font-semibold text-primary-700 hover:bg-primary-100 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-300"
    data-testid="page-share"
    @click="open = true"
  >
    <svg viewBox="0 0 24 24" class="h-3.5 w-3.5" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="M4 12v7a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-7M16 6l-4-4-4 4M12 2v13" /></svg>
    <span>{{ label || t('shareCard.share') }}</span>
  </button>
  <ShareCardDialog :show="open" :input="open ? input : null" @close="open = false" />
</template>

<script setup lang="ts">
// A "share this page" button: opens the share-card dialog with a summary card for the page.
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ShareCardDialog, { type ShareCardInput } from '@/components/share/ShareCardDialog.vue'

defineProps<{ input: ShareCardInput; label?: string }>()
const { t } = useI18n()
const open = ref(false)
</script>
