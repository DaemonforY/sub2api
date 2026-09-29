<template>
  <section class="mb-20" data-testid="home-scenarios">
    <div class="mb-10 text-center">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.scenarios.title') }}</h2>
      <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('home.v2.scenarios.subtitle') }}</p>
    </div>
    <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <component
        :is="card.external ? 'a' : 'router-link'"
        v-for="card in cards"
        :key="card.key"
        v-bind="card.external ? { href: card.to, target: '_blank', rel: 'noopener noreferrer' } : { to: card.to }"
        class="group relative flex flex-col overflow-hidden rounded-2xl border border-gray-200/70 bg-white/70 p-6 shadow-sm backdrop-blur-sm transition hover:-translate-y-1 hover:shadow-xl hover:shadow-primary-500/10 dark:border-dark-700 dark:bg-dark-800/60"
      >
        <div :class="['pointer-events-none absolute -right-10 -top-10 h-32 w-32 rounded-full opacity-20 blur-2xl transition group-hover:opacity-40', card.glow]"></div>
        <div class="flex items-center justify-between">
          <div :class="['flex h-12 w-12 items-center justify-center rounded-xl text-2xl shadow-lg', card.tile]">{{ card.emoji }}</div>
          <span class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300">
            {{ t(`home.v2.scenarios.${card.key}.badge`) }}
          </span>
        </div>
        <h3 class="mt-5 text-lg font-semibold text-gray-900 dark:text-white">{{ t(`home.v2.scenarios.${card.key}.title`) }}</h3>
        <ul class="mt-3 flex-1 space-y-1.5 text-sm text-gray-600 dark:text-dark-300">
          <li v-for="point in points(card.key)" :key="point" class="flex items-start gap-2">
            <span class="mt-1.5 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary-500"></span>
            <span>{{ point }}</span>
          </li>
        </ul>
        <span class="mt-5 inline-flex items-center gap-1 text-sm font-medium text-primary-600 group-hover:gap-2 dark:text-primary-400">
          {{ t('home.v2.scenarios.cta') }} <span aria-hidden="true">→</span>
        </span>
      </component>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { canvasUrl } from '@/constants/crossSites'

const props = defineProps<{ isAuthenticated: boolean; showModelPlaza: boolean }>()
const { t } = useI18n()

interface Card {
  key: 'coding' | 'image' | 'video' | 'batch' | 'plaza' | 'contest'
  emoji: string
  tile: string
  glow: string
  to: string
  external?: boolean
}

const cards = computed<Card[]>(() => {
  const origin = window.location.origin
  const list: Card[] = [
    { key: 'coding', emoji: '💻', tile: 'bg-gradient-to-br from-sky-400 to-blue-600', glow: 'bg-sky-400', to: props.isAuthenticated ? '/keys' : '/register' },
    { key: 'image', emoji: '🎨', tile: 'bg-gradient-to-br from-fuchsia-400 to-violet-600', glow: 'bg-fuchsia-400', to: canvasUrl({ medium: 'home-scenario', baseUrl: origin, path: '/image' }), external: true },
    { key: 'video', emoji: '🎬', tile: 'bg-gradient-to-br from-rose-400 to-orange-500', glow: 'bg-rose-400', to: canvasUrl({ medium: 'home-scenario', baseUrl: origin, path: '/video' }), external: true },
    { key: 'batch', emoji: '🗂️', tile: 'bg-gradient-to-br from-emerald-400 to-teal-600', glow: 'bg-emerald-400', to: '/batch-image' },
    { key: 'plaza', emoji: '🧭', tile: 'bg-gradient-to-br from-amber-400 to-yellow-500', glow: 'bg-amber-400', to: '/model-plaza' },
    { key: 'contest', emoji: '🏆', tile: 'bg-gradient-to-br from-violet-500 to-indigo-600', glow: 'bg-violet-500', to: '/contests' }
  ]
  return props.showModelPlaza ? list : list.filter((c) => c.key !== 'plaza')
})

function points(key: Card['key']): string[] {
  return String(t(`home.v2.scenarios.${key}.points`)).split(',').map((s) => s.trim()).filter(Boolean)
}
</script>
