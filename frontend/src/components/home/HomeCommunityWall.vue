<template>
  <section v-if="works.length" class="mb-20" data-testid="home-community-wall">
    <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.community.title') }}</h2>
        <p class="mt-2 text-gray-600 dark:text-dark-300">{{ t('home.v2.community.subtitle') }}</p>
      </div>
      <a
        :href="canvasUrl({ medium: 'home-community', path: '/explore' })"
        target="_blank"
        rel="noopener noreferrer"
        class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
      >
        {{ t('home.v2.community.more') }} →
      </a>
    </div>
    <div class="columns-2 gap-4 md:columns-3 lg:columns-4">
      <a
        v-for="w in works"
        :key="w.id"
        :href="canvasUrl({ medium: 'home-community', path: `/w/${w.id}` })"
        target="_blank"
        rel="noopener noreferrer"
        class="group relative mb-4 block break-inside-avoid overflow-hidden rounded-2xl bg-gray-100 dark:bg-dark-800"
      >
        <img
          :src="w.cover_thumb_url"
          :alt="w.title"
          :width="w.cover_width || undefined"
          :height="w.cover_height || undefined"
          loading="lazy"
          class="h-auto w-full object-cover transition duration-500 group-hover:scale-105"
        />
        <span class="absolute left-3 top-3 rounded bg-black/55 px-1.5 py-0.5 text-[11px] font-medium text-white">{{ t('home.v2.community.aiLabel') }}</span>
        <span v-if="w.kind === 'site'" class="absolute right-3 top-3 rounded bg-black/55 px-1.5 py-0.5 text-[11px] font-medium text-white" data-testid="home-work-site">{{ t('home.v2.community.siteLabel') }}</span>
        <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/75 via-black/30 to-transparent p-3 pt-10 text-white">
          <div v-if="w.title" class="truncate text-sm font-semibold">{{ w.title }}</div>
          <div class="mt-0.5 flex items-center justify-between gap-2 text-xs text-white/80">
            <span class="truncate">{{ authorName(w.author) }}</span>
            <span v-if="w.like_count" class="flex-shrink-0">♥ {{ w.like_count }}</span>
          </div>
        </div>
      </a>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { authorName, recommendedWorks, type CommunityWork } from '@/api/community'
import { canvasUrl } from '@/constants/crossSites'

const { t } = useI18n()
const works = ref<CommunityWork[]>([])

onMounted(async () => {
  works.value = (await recommendedWorks()).slice(0, 12) // stays hidden when there are none
})
</script>
