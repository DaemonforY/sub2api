<template>
  <!-- Hero art: the community's recommended works; the slot (terminal) shows until there are enough. -->
  <div v-if="works.length >= MIN_WORKS" class="grid w-full max-w-md grid-cols-2 gap-3" data-testid="home-hero-collage">
    <div v-for="(column, ci) in columns" :key="ci" class="flex flex-col gap-3" :class="ci === 1 ? 'pt-10' : ''">
      <a
        v-for="w in column"
        :key="w.id"
        :href="canvasUrl({ medium: 'home-hero', path: `/w/${w.id}` })"
        target="_blank"
        rel="noopener noreferrer"
        class="group relative block overflow-hidden rounded-2xl bg-gray-100 shadow-lg shadow-black/10 dark:bg-dark-800"
        :title="w.title || authorName(w.author)"
      >
        <img
          :src="w.cover_thumb_url"
          :alt="w.title"
          class="w-full object-cover transition duration-500 group-hover:scale-105"
          :style="{ aspectRatio: aspect(w) }"
        />
        <span class="absolute left-2 top-2 rounded bg-black/55 px-1.5 py-0.5 text-[10px] font-medium text-white">{{ t('home.v2.community.aiLabel') }}</span>
        <span class="absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/70 to-transparent px-3 pb-2 pt-6 text-xs text-white/90">@{{ w.author.handle }}</span>
      </a>
    </div>
  </div>
  <slot v-else />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { authorName, recommendedWorks, type CommunityWork } from '@/api/community'
import { canvasUrl } from '@/constants/crossSites'

const MIN_WORKS = 4
const { t } = useI18n()
const works = ref<CommunityWork[]>([])

// Two staggered columns of two works each (about the height of the hero text).
const columns = computed(() => {
  const picked = works.value.slice(0, MIN_WORKS)
  return [picked.filter((_, i) => i % 2 === 0), picked.filter((_, i) => i % 2 === 1)]
})

function aspect(w: CommunityWork): string {
  if (!w.cover_width || !w.cover_height) return '1 / 1'
  const ratio = Math.min(1.25, Math.max(0.75, w.cover_height / w.cover_width))
  return `1 / ${ratio.toFixed(3)}`
}

onMounted(async () => {
  works.value = await recommendedWorks()
})
</script>
