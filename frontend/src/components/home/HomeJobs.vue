<template>
  <section id="jobs" class="mb-20 scroll-mt-20" data-testid="home-jobs">
    <p class="text-sm font-semibold tracking-widest text-primary-600 dark:text-primary-300">{{ t('home.v3.jobs.eyebrow') }}</p>
    <h2 class="mt-2 text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl dark:text-white">{{ t('home.v3.jobs.title') }}</h2>
    <p class="mt-3 max-w-2xl text-gray-600 dark:text-dark-300">{{ t('home.v3.jobs.subtitle') }}</p>

    <div class="mt-10 grid gap-4 lg:grid-cols-[1.25fr_1fr_1fr]">
      <article v-for="job in jobs" :key="job.key" class="flex flex-col overflow-hidden rounded-3xl border border-gray-200/80 bg-white/80 dark:border-white/10 dark:bg-dark-900/60">
        <div class="px-6 pt-6">
          <div class="flex items-baseline gap-2.5">
            <h3 class="text-xl font-bold text-gray-900 dark:text-white">{{ t(`home.v3.jobs.${job.key}.title`) }}</h3>
            <span class="font-mono text-xs text-gray-400">{{ job.model }}</span>
          </div>
          <p class="mt-3 text-sm leading-relaxed text-gray-600 dark:text-dark-300">
            {{ t('home.v3.jobs.say') }}<span class="text-gray-900 dark:text-white">「{{ t(`home.v3.jobs.${job.key}.say`) }}」</span>
          </p>
        </div>
        <div class="relative mx-3.5 mt-5 min-h-[220px] flex-1 overflow-hidden rounded-2xl bg-[#0b0913]">
          <template v-if="job.image">
            <img :src="job.image" alt="" loading="lazy" :class="['absolute inset-0 h-full w-full object-cover', job.position]" />
            <span class="absolute bottom-2.5 left-2.5 rounded-md bg-black/55 px-2 py-0.5 text-[11px] text-white">{{ t(`home.v3.jobs.${job.key}.caption`) }}</span>
          </template>
          <div v-else class="absolute inset-0 p-5 font-mono text-[12.5px] leading-7 text-gray-300">
            <div><span class="text-teal-400">codex ›</span> {{ t('home.v3.jobs.code.cmd') }}</div>
            <div class="text-gray-500">• {{ t('home.v3.jobs.code.line1') }}</div>
            <div class="text-gray-500">• {{ t('home.v3.jobs.code.line2') }}</div>
            <div class="text-emerald-400">✓ {{ t('home.v3.jobs.code.done') }}</div>
            <span class="absolute right-3 top-3 rounded border border-dashed border-white/20 px-1.5 text-[11px] text-gray-500">{{ t('home.v3.demo.code.mock') }}</span>
          </div>
        </div>
        <div class="flex flex-wrap gap-x-5 gap-y-1 px-6 pb-5 pt-4 text-sm">
          <a :href="job.go" v-bind="job.external ? external : {}" class="font-medium text-primary-600 hover:underline dark:text-primary-300">{{ t(`home.v3.jobs.${job.key}.go`) }} →</a>
          <a :href="job.more" v-bind="job.external ? external : {}" class="text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200">{{ t(`home.v3.jobs.${job.key}.more`) }}</a>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { canvasUrl } from '@/constants/crossSites'

/** 它能替你干的活: each card is 「你说：…」 and what came back, with the way in. */
const { t } = useI18n()
const external = { target: '_blank', rel: 'noopener noreferrer' }
const utm = 'utm_source=home&utm_medium=jobs'

const jobs = [
  {
    key: 'draw',
    model: 'gpt-image-2',
    image: '/home/jinxi.jpg',
    position: 'object-center',
    go: canvasUrl({ medium: 'home-jobs', baseUrl: window.location.origin, path: '/image' }),
    more: canvasUrl({ medium: 'home-jobs', path: '/tools' }),
    external: true
  },
  { key: 'write', model: 'GPT-6', image: '/home/editor.jpg', position: 'object-[60%_0]', go: `/editor/?${utm}`, more: `/style?${utm}`, external: false },
  { key: 'code', model: 'Codex', image: '', position: '', go: `/learn/codex/hivegpt?${utm}`, more: `/learn/scenes/automation?${utm}`, external: false }
] as const
</script>
