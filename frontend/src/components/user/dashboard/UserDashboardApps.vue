<template>
  <section v-if="modules.length" class="card p-5" data-testid="dashboard-apps">
    <div class="mb-4 flex items-baseline justify-between">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('appModules.sectionTitle') }}</h3>
      <span class="text-xs text-gray-400">{{ t('appModules.sectionHint') }}</span>
    </div>
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <component
        :is="link(m).to ? 'router-link' : 'a'"
        v-for="m in modules"
        :key="m.key"
        v-bind="link(m).to ? { to: link(m).to } : { href: link(m).href, target: '_blank', rel: 'noopener' }"
        class="group flex items-start gap-3 rounded-xl border border-gray-200 p-4 transition hover:border-primary-400 hover:shadow-sm dark:border-dark-700"
        :data-testid="`dashboard-app-${m.key}`"
      >
        <img :src="m.iconUrl" alt="" class="h-10 w-10 flex-shrink-0 rounded-lg bg-gray-50 p-1.5 dark:bg-dark-800" />
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="font-medium text-gray-900 group-hover:text-primary-600 dark:text-white">{{ t(m.labelKey) }}</span>
            <span v-if="m.badgeKey" class="rounded bg-primary-50 px-1.5 py-0.5 text-[11px] font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ t(m.badgeKey) }}</span>
            <span v-if="!link(m).to" class="text-xs text-gray-400" aria-hidden="true">↗</span>
          </div>
          <p class="mt-1 text-sm leading-5 text-gray-500 dark:text-dark-400">{{ t(m.descriptionKey) }}</p>
        </div>
      </component>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { APP_MODULES, appModuleLink, type AppModule } from '@/constants/appModules'

const { t } = useI18n()
const modules = APP_MODULES
const link = (m: AppModule) => appModuleLink(m, 'dashboard-apps')
</script>
