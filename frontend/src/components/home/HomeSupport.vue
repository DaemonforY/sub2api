<template>
  <section class="mb-20" data-testid="home-support">
    <div class="mb-10 text-center">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('home.v2.support.title') }}</h2>
      <p class="mt-3 text-gray-600 dark:text-dark-300">{{ t('home.v2.support.subtitle') }}</p>
    </div>
    <div class="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
      <component
        :is="item.href ? 'a' : 'div'"
        v-for="item in items"
        :key="item.key"
        v-bind="item.href ? { href: item.href, target: '_blank', rel: 'noopener noreferrer' } : {}"
        :class="[
          'group flex flex-col rounded-2xl border border-gray-200/70 bg-white/70 p-6 backdrop-blur-sm dark:border-dark-700 dark:bg-dark-800/60',
          item.href ? 'transition hover:-translate-y-0.5 hover:shadow-lg' : ''
        ]"
      >
        <span class="text-3xl">{{ item.emoji }}</span>
        <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ item.partner ? t('home.v2.support.partner.title', { site: item.partner.name }) : t(`home.v2.support.${item.key}.title`) }}
        </h3>
        <p class="mt-2 flex-1 text-sm text-gray-600 dark:text-dark-300">
          {{ item.partner ? t('home.v2.support.partner.desc', { site: item.partner.name }) : t(`home.v2.support.${item.key}.desc`) }}
        </p>
        <p v-if="item.partner" class="mt-4 text-sm font-medium text-primary-600 dark:text-primary-400">
          {{ t('home.v2.support.partner.cta', { site: item.partner.name }) }} →
        </p>
        <p v-else-if="item.key === 'contact'" class="mt-4 font-mono text-sm font-medium text-gray-800 [overflow-wrap:anywhere] dark:text-gray-200">
          {{ t('home.v2.support.contact.label', { info: contactInfo }) }}
        </p>
        <span v-else class="mt-4 text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('home.v2.support.open') }} →</span>
      </component>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { otherPartnerSites, partnerSiteUrl, type PartnerSite } from '@/constants/crossSites'

const props = defineProps<{ docUrl: string; learnHref: string; contactInfo: string }>()
const { t } = useI18n()

const items = computed(() => {
  const list: { key: string; emoji: string; href?: string; partner?: PartnerSite }[] = []
  if (props.docUrl) list.push({ key: 'docs', emoji: '📖', href: props.docUrl })
  list.push({ key: 'learn', emoji: '📚', href: props.learnHref })
  for (const site of otherPartnerSites()) {
    list.push({ key: `partner-${site.key}`, emoji: '🤝', href: partnerSiteUrl(site, '/purchase', 'home-support'), partner: site })
  }
  if (props.contactInfo) list.push({ key: 'contact', emoji: '💬' })
  return list
})
</script>
