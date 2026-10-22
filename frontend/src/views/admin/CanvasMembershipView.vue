<template>
  <AppLayout>
    <div class="space-y-4" data-testid="canvas-membership-view">
      <div class="card space-y-1 p-4">
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('canvasMembership.admin.description') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('canvasMembership.admin.hint') }}</p>
      </div>

      <!-- Site-wide watermark switch -->
      <div class="card space-y-2 p-4" data-testid="canvas-membership-watermark">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('canvasMembership.admin.watermarkOffTitle') }}</h3>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input :checked="watermarkOff" type="checkbox" class="h-4 w-4" :disabled="switching" data-testid="canvas-membership-watermark-off" @change="toggleWatermark" />
            {{ t('canvasMembership.admin.watermarkOff') }}
          </label>
        </div>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('canvasMembership.admin.watermarkOffHint') }}</p>
        <p class="text-xs text-amber-600 dark:text-amber-400">{{ t('canvasMembership.admin.watermarkOffRisk') }}</p>
      </div>

      <!-- Plans -->
      <form class="card space-y-4 p-4" data-testid="canvas-membership-config" @submit.prevent="save">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('canvasMembership.admin.plans') }}</h3>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="config.enabled" type="checkbox" class="h-4 w-4" data-testid="canvas-membership-enabled" />
            {{ t('canvasMembership.admin.enabled') }}
          </label>
        </div>
        <div v-for="(plan, index) in config.plans" :key="plan.id || `new-${index}`" class="grid grid-cols-2 items-end gap-3 md:grid-cols-[2fr_1fr_1fr_1fr_auto]">
          <div>
            <label class="input-label">{{ t('canvasMembership.admin.name') }}</label>
            <input v-model="plan.name" class="input" maxlength="20" required />
          </div>
          <div>
            <label class="input-label">{{ t('canvasMembership.admin.days') }}</label>
            <input v-model.number="plan.days" type="number" min="1" max="3660" class="input" required />
          </div>
          <div>
            <label class="input-label">{{ t('canvasMembership.admin.price') }}</label>
            <input v-model.number="plan.price" type="number" min="0.01" step="0.01" class="input" required />
          </div>
          <div>
            <label class="input-label">{{ t('canvasMembership.admin.originalPrice') }}</label>
            <input v-model.number="plan.original_price" type="number" min="0" step="0.01" class="input" />
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="config.plans.splice(index, 1)">{{ t('canvasMembership.admin.remove') }}</button>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="config.plans.length >= 6" @click="addPlan">{{ t('canvasMembership.admin.addPlan') }}</button>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="saving" data-testid="canvas-membership-save">{{ t('canvasMembership.admin.save') }}</button>
        </div>
      </form>

      <!-- Grant -->
      <form class="card flex flex-wrap items-end gap-3 p-4" @submit.prevent="grant">
        <h3 class="w-full font-semibold text-gray-900 dark:text-white">{{ t('canvasMembership.admin.grantTitle') }}</h3>
        <div>
          <label class="input-label">{{ t('canvasMembership.admin.grantUser') }}</label>
          <input v-model="grantUser" class="input w-64" required />
        </div>
        <div>
          <label class="input-label">{{ t('canvasMembership.admin.grantDays') }}</label>
          <input v-model.number="grantDays" type="number" class="input w-32" required />
        </div>
        <button type="submit" class="btn btn-secondary btn-sm" :disabled="granting">{{ t('canvasMembership.admin.grant') }}</button>
      </form>

      <!-- Members -->
      <div class="card p-4" data-testid="canvas-membership-members">
        <div class="mb-3 flex items-center justify-between">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('canvasMembership.admin.members') }} · {{ members.length }}</h3>
          <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="showAll" type="checkbox" class="h-4 w-4" @change="load" />
            {{ t('canvasMembership.admin.showAll') }}
          </label>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs text-gray-500">
                <th class="py-2 pr-3">ID</th>
                <th class="py-2 pr-3">{{ t('canvasMembership.admin.email') }}</th>
                <th class="py-2 pr-3">{{ t('canvasMembership.admin.until') }}</th>
                <th class="py-2 pr-3">{{ t('canvasMembership.admin.terms') }}</th>
                <th class="py-2 pr-3">{{ t('canvasMembership.admin.unmarkedSaves') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in members" :key="m.user_id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="py-2 pr-3 text-gray-500">{{ m.user_id }}</td>
                <td class="py-2 pr-3 text-gray-900 dark:text-white">{{ m.email || m.username }}</td>
                <td class="py-2 pr-3" :class="ended(m.member_until) ? 'text-gray-400' : 'text-emerald-600 dark:text-emerald-400'">{{ dateText(m.member_until) }}</td>
                <td class="py-2 pr-3 text-gray-600 dark:text-gray-300">{{ m.terms_accepted_at ? dateText(m.terms_accepted_at) : '—' }}</td>
                <td class="py-2 pr-3 text-gray-600 dark:text-gray-300">{{ m.unmarked_saves }}</td>
              </tr>
              <tr v-if="!members.length">
                <td colspan="5" class="py-8 text-center text-gray-500">{{ t('canvasMembership.admin.noMembers') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminGetCanvasMembership, adminGrantCanvasMembership, adminSaveCanvasMembership, type CanvasMember, type CanvasMembershipConfig } from '@/api/canvasMembership'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const config = ref<CanvasMembershipConfig>({ enabled: false, plans: [] })
const members = ref<CanvasMember[]>([])
const showAll = ref(false)
const saving = ref(false)
const granting = ref(false)
// The site-wide switch saves on its own, together with the plans as last saved (not unsaved edits in the form).
const savedConfig = ref<CanvasMembershipConfig>({ enabled: false, plans: [] })
const watermarkOff = ref(false)
const switching = ref(false)
const grantUser = ref('')
const grantDays = ref<number>(30)

function dateText(value: string): string {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}
function ended(value: string): boolean {
  return new Date(value).getTime() <= Date.now()
}
function addPlan() {
  config.value.plans.push({ id: 0, name: '', days: 30, price: 0 })
}

function apply(res: { config: CanvasMembershipConfig; members: CanvasMember[] }) {
  config.value = { enabled: res.config.enabled, plans: res.config.plans.map((p) => ({ ...p, original_price: p.original_price || undefined })) }
  savedConfig.value = res.config
  watermarkOff.value = res.config.watermark_off === true
  members.value = res.members
}

async function load() {
  try {
    apply(await adminGetCanvasMembership(showAll.value))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function save() {
  saving.value = true
  try {
    const plans = config.value.plans.map((p) => ({ ...p, original_price: Number(p.original_price) || 0 }))
    apply(await adminSaveCanvasMembership({ enabled: config.value.enabled, plans, watermark_off: watermarkOff.value }))
    appStore.showSuccess(t('canvasMembership.admin.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function toggleWatermark(event: Event) {
  const input = event.target as HTMLInputElement
  const off = input.checked
  if (off && !window.confirm(t('canvasMembership.admin.watermarkOffConfirm'))) {
    input.checked = false
    return
  }
  switching.value = true
  try {
    // Keep the form's unsaved plan edits; only the switch and the saved settings go out.
    const draft = config.value
    const res = await adminSaveCanvasMembership({ ...savedConfig.value, watermark_off: off })
    apply(res)
    config.value = draft
    appStore.showSuccess(t(off ? 'canvasMembership.admin.watermarkOffSaved' : 'canvasMembership.admin.watermarkOnSaved'))
  } catch (err) {
    input.checked = watermarkOff.value
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    switching.value = false
  }
}

async function grant() {
  const who = grantUser.value.trim()
  if (!who || !grantDays.value) return
  granting.value = true
  try {
    const id = Number(who)
    const res = await adminGrantCanvasMembership(Number.isInteger(id) && id > 0 ? { user_id: id, days: grantDays.value } : { email: who, days: grantDays.value })
    appStore.showSuccess(res.until ? t('canvasMembership.admin.granted', { date: dateText(res.until) }) : t('canvasMembership.admin.grantedEnded'))
    await load()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    granting.value = false
  }
}

onMounted(load)
</script>
