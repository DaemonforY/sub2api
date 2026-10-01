<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-10 dark:bg-dark-950">
    <div class="card w-full max-w-lg space-y-4 p-6">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('siteReport.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('siteReport.description') }}</p>
      </div>
      <div v-if="done" class="rounded-lg bg-green-50 p-4 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-300" data-testid="report-done">{{ t('siteReport.done') }}</div>
      <template v-else>
        <div>
          <label class="input-label">{{ t('siteReport.site') }}</label>
          <input v-model="form.site" class="input" placeholder="abc1234" />
        </div>
        <div>
          <label class="input-label">{{ t('siteReport.reason') }}</label>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            <label v-for="reason in SITE_REPORT_REASONS" :key="reason" class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.reason" type="radio" :value="reason" />
              {{ t(`siteReport.reasons.${reason}`) }}
            </label>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('siteReport.detail') }}</label>
          <textarea v-model="form.detail" class="input min-h-24" maxlength="1000" :placeholder="t('siteReport.detailPlaceholder')" />
        </div>
        <div>
          <label class="input-label">{{ t('siteReport.contact') }}</label>
          <input v-model="form.contact" class="input" maxlength="200" :placeholder="t('siteReport.contactPlaceholder')" />
        </div>
        <button class="btn btn-primary w-full" :disabled="busy || !form.site || !form.reason" @click="submit">{{ t('siteReport.submit') }}</button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { reportSite, SITE_REPORT_REASONS, type SiteReportReason } from '@/api/sites'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const busy = ref(false)
const done = ref(false)
const form = reactive({ site: String(route.query.site || ''), reason: '' as SiteReportReason | '', detail: '', contact: '' })

async function submit() {
  if (!form.reason) return
  busy.value = true
  try {
    await reportSite({ site: form.site.trim().toLowerCase(), reason: form.reason, detail: form.detail, contact: form.contact })
    done.value = true
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    busy.value = false
  }
}
</script>
