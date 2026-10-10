<script setup lang="ts">
// 「分享」 at the top and bottom of a page: opens the summary card.
import { computed } from 'vue'
import { useData } from 'vitepress'
import { openSummaryCard } from '../share'

defineProps<{ place: 'top' | 'bottom' }>()
const { page } = useData()
const enabled = computed(() => !!page.value.shareCard)
</script>

<template>
  <div v-if="enabled && place === 'top'" class="share-top">
    <button class="share-chip" data-testid="share-open" @click="openSummaryCard">
      <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" d="M4 12v7a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-7M16 6l-4-4-4 4M12 2v13" /></svg>
      分享
    </button>
  </div>
  <div v-else-if="enabled" class="share-bottom">
    <span>觉得有用？生成一张卡片发给朋友，选中正文里的段落也能生成摘录卡。</span>
    <button class="runbox-btn small" data-testid="share-open-bottom" @click="openSummaryCard">生成分享卡片</button>
  </div>
</template>
