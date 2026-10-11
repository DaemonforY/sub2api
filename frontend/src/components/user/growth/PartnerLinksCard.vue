<template>
  <div v-if="report && (report.links.length > 0 || period !== 30)" class="card p-6" data-testid="partner-links">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">📣 {{ t('growth.partnerLinks.title') }}</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.partnerLinks.description') }}</p>
      </div>
      <div class="flex rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-800">
        <button
          v-for="p in periods"
          :key="p"
          type="button"
          :data-testid="`partner-links-period-${p}`"
          :class="['rounded-md px-3 py-1', period === p ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-400']"
          @click="changePeriod(p)"
        >
          {{ t(`growth.partnerLinks.periods.${p ? `d${p}` : 'all'}`) }}
        </button>
      </div>
    </div>

    <div v-for="l in report.links" :key="l.code" class="mt-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700" :data-testid="`partner-link-${l.code}`">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <p class="font-medium text-gray-900 dark:text-white">{{ l.name }}</p>
          <code class="break-all text-sm text-primary-600 dark:text-primary-400">{{ shortUrl(l) }}</code>
          <p v-if="l.promo_code" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.partnerLinks.promo', { code: l.promo_code }) }}</p>
        </div>
        <div class="flex shrink-0 gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="copy(shortUrl(l))">{{ t('growth.partnerLinks.copy') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="showQR(l)">{{ t('growth.partnerLinks.qr') }}</button>
        </div>
      </div>
      <div class="mt-3 grid grid-cols-3 gap-2 text-center sm:grid-cols-7">
        <div v-for="c in columns(l)" :key="c.key" class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t(`growth.partnerLinks.columns.${c.key}`) }}</p>
          <p :class="['mt-0.5 font-semibold tabular-nums', c.key === 'rebates' ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-900 dark:text-white']" :data-testid="`partner-link-${l.code}-${c.key}`">{{ c.value }}</p>
        </div>
      </div>
    </div>
    <p class="mt-3 text-xs text-gray-400 dark:text-dark-500">{{ t('growth.partnerLinks.note') }}</p>

    <div v-if="qr" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="qr = null">
      <div class="card w-full max-w-xs space-y-3 p-5 text-center">
        <h3 class="font-semibold text-gray-900 dark:text-white">{{ qr.link.name }}</h3>
        <img :src="qr.dataUrl" alt="QR" class="mx-auto h-56 w-56" />
        <p class="break-all font-mono text-xs text-gray-500">{{ shortUrl(qr.link) }}</p>
        <div class="flex justify-center gap-2">
          <a :href="qr.dataUrl" :download="`${qr.link.code}.png`" class="btn btn-primary btn-sm">{{ t('growth.partnerLinks.downloadQR') }}</a>
          <button type="button" class="btn btn-secondary btn-sm" @click="qr = null">{{ t('growth.partnerLinks.close') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { growthAPI, type PartnerLink, type PartnerLinkReport } from '@/api/growth'
import { useAppStore } from '@/stores/app'

// Shown only to users who have a partner short link (an admin bound their invite code to one).
const { t } = useI18n()
const appStore = useAppStore()
const periods = [7, 30, 0]
const period = ref(30)
const report = ref<PartnerLinkReport | null>(null)
const qr = ref<{ link: PartnerLink; dataUrl: string } | null>(null)

const origin = typeof window === 'undefined' ? '' : window.location.origin
const shortUrl = (l: PartnerLink) => `${origin}/go/${l.code}`
const money = (v: number) => (v ? v.toFixed(2) : '0')

function columns(l: PartnerLink) {
  return [
    { key: 'clicks', value: l.clicks },
    { key: 'visitors', value: l.visitors },
    { key: 'signups', value: l.signups },
    { key: 'activated', value: l.activated },
    { key: 'paid', value: l.paid_users },
    { key: 'revenue', value: money(l.revenue) },
    { key: 'rebates', value: money(l.rebates) }
  ]
}

async function load() {
  try {
    report.value = await growthAPI.getPartnerLinks(period.value)
  } catch {
    report.value = null
  }
}

function changePeriod(p: number) {
  if (p === period.value) return
  period.value = p
  void load()
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess(t('growth.partnerLinks.copied', { url: text }))
  } catch {
    window.prompt(t('growth.partnerLinks.copy'), text)
  }
}

async function showQR(l: PartnerLink) {
  try {
    qr.value = { link: l, dataUrl: await QRCode.toDataURL(shortUrl(l), { width: 640, margin: 2, errorCorrectionLevel: 'M' }) }
  } catch {
    qr.value = null
  }
}

onMounted(load)
</script>
