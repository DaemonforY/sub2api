<template>
  <section class="mb-20" data-testid="home-steps">
    <div class="mb-10 text-center">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.steps.title') }}</h2>
      <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('home.v2.steps.subtitle') }}</p>
    </div>
    <ol class="grid gap-5 md:grid-cols-3">
      <li
        v-for="(step, i) in steps"
        :key="step.key"
        class="relative rounded-2xl border border-gray-200/70 bg-white/70 p-6 backdrop-blur-sm dark:border-dark-700 dark:bg-dark-800/60"
      >
        <span class="text-4xl font-black text-primary-500/20 dark:text-primary-400/20">0{{ i + 1 }}</span>
        <h3 class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ t(`home.v2.steps.${step.key}.title`) }}</h3>
        <p class="mt-2 min-h-[2.5rem] text-sm text-gray-600 dark:text-dark-300">{{ t(`home.v2.steps.${step.key}.desc`) }}</p>
        <component
          :is="step.external ? 'a' : 'router-link'"
          v-bind="step.external ? { href: step.to, target: '_blank', rel: 'noopener noreferrer' } : { to: step.to }"
          class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
        >
          {{ step.cta }} →
        </component>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { canvasUrl } from '@/constants/crossSites'

const props = defineProps<{ isAuthenticated: boolean; dashboardPath: string }>()
const { t } = useI18n()

const steps = computed(() => [
  {
    key: 'register',
    to: props.isAuthenticated ? props.dashboardPath : '/register',
    cta: props.isAuthenticated ? t('home.goToDashboard') : t('home.v2.steps.registerCta')
  },
  { key: 'key', to: '/keys', cta: t('home.v2.steps.keyCta') },
  {
    key: 'use',
    to: canvasUrl({ medium: 'home-steps', baseUrl: window.location.origin }),
    cta: t('home.v2.steps.useCta'),
    external: true
  }
])
</script>
