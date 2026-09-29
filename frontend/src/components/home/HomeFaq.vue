<template>
  <section class="mx-auto mb-20 max-w-3xl" data-testid="home-faq">
    <div class="mb-10 text-center">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.faq.title') }}</h2>
      <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('home.v2.faq.subtitle') }}</p>
    </div>
    <div class="space-y-3">
      <div
        v-for="key in keys"
        :key="key"
        class="overflow-hidden rounded-xl border border-gray-200/70 bg-white/70 backdrop-blur-sm dark:border-dark-700 dark:bg-dark-800/60"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between gap-4 px-5 py-4 text-left text-sm font-medium text-gray-900 dark:text-white"
          :aria-expanded="open === key"
          @click="open = open === key ? '' : key"
        >
          <span>{{ t(`home.v2.faq.items.${key}.q`, { site: siteName }) }}</span>
          <span :class="['flex-shrink-0 text-gray-400 transition-transform', open === key ? 'rotate-180' : '']" aria-hidden="true">⌄</span>
        </button>
        <div v-show="open === key" class="px-5 pb-4 text-sm leading-relaxed text-gray-600 dark:text-dark-300">
          {{ t(`home.v2.faq.items.${key}.a`, { site: siteName }) }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{ siteName: string }>()
const { t } = useI18n()

const keys = ['what', 'models', 'billing', 'canvas', 'privacy', 'contact'] as const
const open = ref<string>('what')
</script>
