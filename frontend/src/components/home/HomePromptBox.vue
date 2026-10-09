<template>
  <div
    class="rounded-2xl border border-gray-200/70 bg-white/80 p-3 shadow-xl shadow-primary-500/5 backdrop-blur-md dark:border-dark-700/70 dark:bg-dark-800/70"
    data-testid="home-prompt-box"
  >
    <div class="relative">
      <textarea
        v-model="prompt"
        rows="3"
        maxlength="1000"
        class="block w-full resize-none rounded-xl border-0 bg-transparent px-3 py-2.5 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-0 dark:text-white dark:placeholder:text-dark-400"
        :placeholder="t('home.v2.hero.promptPlaceholder')"
        data-testid="home-prompt-input"
        @keydown.enter.meta.prevent="generate"
        @keydown.enter.ctrl.prevent="generate"
      ></textarea>
      <span class="pointer-events-none absolute bottom-1 right-3 text-[11px] text-gray-400">{{ prompt.length }} / 1000</span>
    </div>

    <div class="mt-2 flex flex-wrap items-center gap-2 px-1">
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('home.v2.hero.examplesLabel') }}</span>
      <button
        v-for="ex in examples"
        :key="ex"
        type="button"
        class="rounded-full border border-gray-200 px-2.5 py-1 text-xs text-gray-600 transition hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-dark-300 dark:hover:border-primary-500 dark:hover:text-primary-400"
        @click="prompt = ex"
      >
        {{ ex }}
      </button>
    </div>

    <div class="mt-3 flex flex-col gap-2 border-t border-gray-100 pt-3 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700">
      <p class="px-1 text-xs text-gray-500 dark:text-dark-400">{{ t('home.v2.hero.hint') }}</p>
      <div class="flex shrink-0 gap-2">
        <router-link :to="keyTarget" class="btn btn-secondary px-4 py-2 text-sm" @click="track('cta_click', { where: 'hero-key' })">
          {{ t('home.v2.hero.getKey') }}
        </router-link>
        <a
          :href="canvasHref"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-primary px-5 py-2 text-sm shadow-lg shadow-primary-500/30"
          data-testid="home-prompt-generate"
        >
          ✨ {{ t('home.v2.hero.generate') }}
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { canvasUrl } from '@/constants/crossSites'
import { track } from '@/utils/analytics'

const props = defineProps<{ isAuthenticated: boolean }>()

const { t } = useI18n()
const prompt = ref('')

const examples = computed(() => String(t('home.v2.hero.examples')).split(',').map((s) => s.trim()).filter(Boolean))
const keyTarget = computed(() => (props.isAuthenticated ? '/keys' : '/register'))
const canvasHref = computed(() =>
  canvasUrl({ medium: 'home-hero', baseUrl: window.location.origin, path: '/image', prompt: prompt.value })
)

function generate() {
  window.open(canvasHref.value, '_blank', 'noopener,noreferrer')
}
</script>
