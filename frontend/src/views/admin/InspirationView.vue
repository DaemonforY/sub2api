<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card flex flex-wrap items-center gap-1 p-3">
        <button
          v-for="tab in tabs"
          :key="tab"
          class="rounded-lg px-3 py-1.5 text-sm"
          :class="active === tab ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
          :data-testid="`inspiration-tab-${tab}`"
          @click="active = tab"
        >
          {{ t(`inspiration.tabs.${tab}`) }}
        </button>
      </div>
      <!-- Panels stay mounted so a search survives switching tabs. -->
      <InspirationPoemsPanel v-show="active === 'poems'" ref="poemsPanel" />
      <InspirationFindPanel v-show="active === 'find'" />
      <InspirationSongsPanel v-show="active === 'songs'" @search-poem="searchPoem" />
      <InspirationPublicPanel v-if="active === 'public'" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import InspirationPoemsPanel from './inspiration/InspirationPoemsPanel.vue'
import InspirationFindPanel from './inspiration/InspirationFindPanel.vue'
import InspirationSongsPanel from './inspiration/InspirationSongsPanel.vue'
import InspirationPublicPanel from './inspiration/InspirationPublicPanel.vue'

const { t } = useI18n()
const tabs = ['poems', 'find', 'songs', 'public'] as const
const active = ref<(typeof tabs)[number]>('poems')
const poemsPanel = ref<InstanceType<typeof InspirationPoemsPanel> | null>(null)

/** From a song's allusion: look the poem up in the corpus. */
function searchPoem(q: string) {
  active.value = 'poems'
  poemsPanel.value?.searchFor(q)
}
</script>
