<template>
  <!-- The home page's dark stage on the left (slogan, what it does, the live token counter), the form
       on the right in the page theme. Phones get a short dark band with the logo and the slogan instead. -->
  <div class="relative flex min-h-screen flex-col bg-gray-50 lg:flex-row dark:bg-dark-950">
    <aside class="auth-stage relative overflow-hidden bg-[#050a14] text-white lg:sticky lg:top-0 lg:h-screen lg:w-[46%] lg:max-w-[720px] lg:shrink-0 lg:border-r lg:border-white/10">
      <div class="stage-glow pointer-events-none absolute inset-0" aria-hidden="true"></div>
      <div class="stage-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>

      <div class="relative z-[1] flex h-full flex-col px-5 pb-6 pt-5 sm:px-8 lg:px-12 lg:py-10">
        <router-link to="/home" class="flex w-fit items-center gap-2.5" data-testid="auth-home-link">
          <span class="h-9 w-9 shrink-0 overflow-hidden rounded-xl shadow-md shadow-blue-500/20">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-base font-semibold">{{ siteName }}</span>
        </router-link>

        <!-- phones: the slogan in one line -->
        <p class="mt-5 text-[26px] font-extrabold leading-tight tracking-tight lg:hidden">
          {{ t('home.v3.hero.titleLead') }}{{ titleGap }}<span class="flow-text">{{ t('home.v3.hero.titleHighlight') }}</span>
        </p>
        <p class="mt-2 text-sm text-slate-400 lg:hidden">{{ t('home.v3.hero.subtitleEm') }}{{ t('home.v3.hero.subtitleTail') }}</p>

        <div class="hidden flex-1 flex-col justify-center py-10 lg:flex">
          <p class="mb-8 inline-flex w-fit items-center gap-2 rounded-full border border-white/15 bg-[#050a14]/60 py-1 pl-1 pr-3 text-xs text-slate-400">
            <span class="rounded-full bg-cyan-300 px-2 py-0.5 text-[11px] font-bold text-cyan-950">NEW</span>
            <span class="text-white">{{ t('home.v3.hero.news') }}</span>· {{ t('home.v3.hero.newsTail') }}
          </p>
          <h1 class="stage-title">
            <span>{{ t('home.v3.hero.titleLead') }}</span>{{ titleGap }}<span class="flow-text">{{ t('home.v3.hero.titleHighlight') }}</span>
          </h1>
          <p class="mt-6 max-w-md text-base leading-relaxed text-slate-400">
            {{ t('home.v3.hero.subtitleLead') }}<span class="text-white">{{ t('home.v3.hero.subtitleEm') }}</span>{{ t('home.v3.hero.subtitleTail') }}
          </p>

          <!-- what people ask it for, one per line -->
          <ul class="mt-9 max-w-md space-y-3">
            <li v-for="job in jobs" :key="job.key" class="job-row">
              <span class="job-icon"><Icon :name="job.icon" size="md" /></span>
              <span class="min-w-0">
                <span class="block text-sm font-semibold text-white">{{ t(`home.v3.jobs.${job.key}.title`) }}</span>
                <span class="mt-0.5 block truncate text-[13px] text-slate-400">{{ t('home.v3.jobs.say') }}{{ t(`home.v3.jobs.${job.key}.say`) }}</span>
              </span>
            </li>
          </ul>

          <HomeTokenCounter class="mt-10 self-start" />
        </div>

        <p class="hidden text-xs text-slate-500 lg:block">&copy; {{ currentYear }} {{ siteName }}. All rights reserved.</p>
      </div>
    </aside>

    <main class="relative flex min-w-0 flex-1 flex-col">
      <div class="form-glow pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true"></div>
      <div class="relative z-[1] hidden justify-end px-8 pt-6 lg:flex">
        <router-link to="/home" class="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white">
          <Icon name="arrowLeft" size="sm" />
          {{ t('auth.layout.backHome') }}
        </router-link>
      </div>

      <div class="relative z-[1] flex flex-1 items-start justify-center px-4 py-6 sm:py-10 lg:items-center lg:px-8">
        <div class="w-full max-w-md">
          <div class="auth-card rounded-3xl border border-gray-200/80 bg-white p-6 shadow-xl shadow-slate-900/[0.06] sm:p-8 dark:border-white/10 dark:bg-dark-900 dark:shadow-black/30">
            <slot />
          </div>

          <div class="mt-6 text-center text-sm">
            <slot name="footer" />
          </div>

          <p class="mt-8 text-center text-xs text-gray-400 lg:hidden dark:text-dark-500">&copy; {{ currentYear }} {{ siteName }}. All rights reserved.</p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import Icon from '@/components/icons/Icon.vue'
import HomeTokenCounter from '@/components/home/HomeTokenCounter.vue'

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
// 「你当老板，」 ends in a full-width comma; "You're the boss." needs a space before the next part
const titleGap = computed(() => (/[，。、！？]$/.test(t('home.v3.hero.titleLead')) ? '' : ' '))

const jobs = [
  { key: 'draw', icon: 'sparkles' },
  { key: 'write', icon: 'edit' },
  { key: 'code', icon: 'terminal' }
] as const

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.stage-glow {
  background:
    radial-gradient(60% 50% at 30% 40%, rgba(37, 99, 235, 0.26), transparent 70%),
    radial-gradient(45% 40% at 90% 90%, rgba(94, 231, 255, 0.1), transparent 70%);
}
.stage-grid {
  opacity: 0.55;
  background-image:
    linear-gradient(rgba(94, 231, 255, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(94, 231, 255, 0.06) 1px, transparent 1px);
  background-size: 56px 56px;
  -webkit-mask-image: radial-gradient(75% 70% at 40% 45%, #000, transparent);
  mask-image: radial-gradient(75% 70% at 40% 45%, #000, transparent);
}
.form-glow {
  background: radial-gradient(50% 40% at 50% 30%, rgba(59, 130, 246, 0.08), transparent 70%);
}
.stage-title {
  font-size: clamp(44px, 4.6vw, 72px);
  font-weight: 800;
  line-height: 1.12;
}
.stage-title > span {
  display: inline-block;
  white-space: nowrap;
}
.flow-text {
  background: linear-gradient(100deg, #9fdcff 0%, #5ee7ff 30%, #3b82f6 70%, #5ee7ff 100%);
  background-size: 220% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  animation: flow 5s linear infinite;
}
@keyframes flow {
  to { background-position: -220% 0; }
}
.job-row {
  display: flex;
  align-items: center;
  gap: 14px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.03);
  padding: 12px 14px;
}
.job-icon {
  display: grid;
  height: 40px;
  width: 40px;
  flex-shrink: 0;
  place-items: center;
  border-radius: 12px;
  color: #5ee7ff;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.25), rgba(94, 231, 255, 0.08));
  box-shadow: inset 0 0 0 1px rgba(94, 231, 255, 0.2);
}

/* the form takes the home page's blue: the main button is the hero button, inputs focus in blue */
.auth-card :deep(.btn-primary) {
  border-radius: 14px;
  padding-top: 0.75rem;
  padding-bottom: 0.75rem;
  font-size: 15px;
  font-weight: 600;
  background: linear-gradient(100deg, #2563eb, #3b82f6 55%, #22b8e6);
  box-shadow: 0 10px 30px -12px rgba(59, 130, 246, 0.8), inset 0 1px 0 rgba(255, 255, 255, 0.25);
}
.auth-card :deep(.btn-primary:hover:not(:disabled)) {
  filter: brightness(1.08);
}
.auth-card :deep(.input:focus) {
  border-color: #3b82f6;
  --tw-ring-color: rgba(59, 130, 246, 0.3);
}
@media (prefers-reduced-motion: reduce) {
  .flow-text { animation: none; }
}
</style>
