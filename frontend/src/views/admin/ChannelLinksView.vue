<template>
  <AppLayout>
    <div class="space-y-4" data-testid="channel-links-view">
      <div class="card flex flex-wrap items-start justify-between gap-3 p-4">
        <div class="max-w-3xl space-y-1">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.channelLinks.intro', { base: shortBase }) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.channelLinks.statsNote') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <router-link to="/admin/analytics" class="btn btn-secondary btn-sm">{{ t('admin.channelLinks.backToAnalytics') }}</router-link>
          <button type="button" class="btn btn-primary btn-sm" data-testid="channel-link-new" @click="startCreate">{{ t('admin.channelLinks.create') }}</button>
        </div>
      </div>

      <!-- Create / edit -->
      <form v-if="form" class="card space-y-4 p-4" data-testid="channel-link-form" @submit.prevent="save">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ editingId ? t('admin.channelLinks.editTitle') : t('admin.channelLinks.createTitle') }}
        </h3>
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label" for="cl-name">{{ t('admin.channelLinks.field.name') }}</label>
            <input id="cl-name" v-model="form.name" class="input" maxlength="100" :placeholder="t('admin.channelLinks.field.namePlaceholder')" required />
          </div>
          <div>
            <label class="input-label" for="cl-source">{{ t('admin.channelLinks.field.source') }}</label>
            <div class="flex gap-2">
              <select id="cl-source" v-model="sourceChoice" class="input">
                <option v-for="s in SOURCES" :key="s" :value="s">{{ sourceLabel(s) }}</option>
                <option value="__custom">{{ t('admin.channelLinks.custom') }}</option>
              </select>
              <input v-if="sourceChoice === '__custom'" v-model="form.source" class="input" maxlength="32" placeholder="e.g. weibo" data-testid="channel-link-source-custom" />
            </div>
          </div>
          <div>
            <label class="input-label" for="cl-medium">{{ t('admin.channelLinks.field.medium') }}</label>
            <select id="cl-medium" v-model="form.medium" class="input">
              <option value="">{{ t('admin.channelLinks.medium.none') }}</option>
              <option v-for="m in MEDIUMS" :key="m" :value="m">{{ t(`admin.channelLinks.medium.${m}`) }}</option>
            </select>
          </div>
          <div>
            <label class="input-label" for="cl-target">{{ t('admin.channelLinks.field.target') }}</label>
            <div class="flex gap-2">
              <select id="cl-target" v-model="targetChoice" class="input">
                <option v-for="p in TARGETS" :key="p.path" :value="p.path">{{ t(`admin.channelLinks.target.${p.key}`) }}（{{ p.path }}）</option>
                <option value="__custom">{{ t('admin.channelLinks.custom') }}</option>
              </select>
              <input v-if="targetChoice === '__custom'" v-model="form.target_path" class="input font-mono" maxlength="200" placeholder="/learn/..." />
            </div>
          </div>
          <div>
            <label class="input-label" for="cl-aff">{{ t('admin.channelLinks.field.aff') }}</label>
            <input id="cl-aff" v-model="form.aff_code" class="input font-mono uppercase" maxlength="32" :placeholder="t('admin.channelLinks.field.affPlaceholder')" />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.channelLinks.field.affHint') }}</p>
          </div>
          <div>
            <label class="input-label" for="cl-code">{{ t('admin.channelLinks.field.code') }}</label>
            <input
              id="cl-code"
              v-model="form.code"
              class="input font-mono lowercase"
              maxlength="32"
              :disabled="!!editingId"
              :placeholder="t('admin.channelLinks.field.codePlaceholder')"
            />
            <p class="mt-1 text-xs text-gray-400">{{ editingId ? t('admin.channelLinks.field.codeLocked') : t('admin.channelLinks.field.codeHint') }}</p>
          </div>
          <div class="md:col-span-2">
            <label class="input-label" for="cl-note">{{ t('admin.channelLinks.field.note') }}</label>
            <input id="cl-note" v-model="form.note" class="input" maxlength="300" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="form = null">{{ t('admin.channelLinks.cancel') }}</button>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="saving" data-testid="channel-link-save">{{ t('admin.channelLinks.save') }}</button>
        </div>
      </form>

      <div class="card p-4">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[960px] text-sm" data-testid="channel-links-table">
            <thead>
              <tr class="whitespace-nowrap text-left text-xs text-gray-500">
                <th class="py-1">{{ t('admin.channelLinks.col.name') }}</th>
                <th>{{ t('admin.channelLinks.col.link') }}</th>
                <th class="px-2 text-right">{{ t('admin.channelLinks.col.clicks') }}</th>
                <th class="px-2 text-right">{{ t('admin.analytics.col.visitors') }}</th>
                <th class="px-2 text-right">{{ t('admin.analytics.col.signups') }}</th>
                <th class="px-2 text-right">{{ t('admin.channelLinks.col.activated') }}</th>
                <th class="px-2 text-right">{{ t('admin.analytics.col.paidUsers') }}</th>
                <th class="px-2 text-right">{{ t('admin.analytics.col.revenue') }}</th>
                <th class="px-2 text-right">{{ t('admin.channelLinks.col.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="l in links" :key="l.id" class="border-t border-gray-100 align-top dark:border-dark-700">
                <td class="max-w-[280px] py-2 pr-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ l.name }}</div>
                  <div class="text-xs text-gray-500">
                    {{ sourceLabel(l.source) }}<template v-if="l.medium"> · {{ mediumLabel(l.medium) }}</template> · {{ l.target_path }}
                    <template v-if="l.aff_code"> · {{ t('admin.channelLinks.affTag', { code: l.aff_code }) }}</template>
                  </div>
                  <div v-if="l.note" class="text-xs text-gray-400">{{ l.note }}</div>
                </td>
                <td class="py-2 pr-3">
                  <button type="button" class="whitespace-nowrap font-mono text-xs text-primary-600 hover:underline dark:text-primary-400" :data-testid="`channel-link-copy-${l.code}`" @click="copy(shortUrl(l))">
                    {{ shortUrl(l) }}
                  </button>
                </td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.clicks }}</td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.visitors }}</td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.signups }}</td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.activated }}</td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.paid_users }}</td>
                <td class="whitespace-nowrap px-2 py-2 text-right tabular-nums">{{ l.revenue ? l.revenue.toFixed(2) : 0 }}</td>
                <td class="whitespace-nowrap py-2 pl-2 text-right">
                  <button type="button" class="btn btn-secondary btn-sm" @click="showQR(l)">{{ t('admin.channelLinks.qr') }}</button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1" @click="startEdit(l)">{{ t('admin.channelLinks.edit') }}</button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1 text-red-600" @click="remove(l)">{{ t('admin.channelLinks.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!loading && !links.length">
                <td colspan="9" class="py-4 text-center text-gray-500">{{ t('admin.channelLinks.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- QR code -->
      <div v-if="qr" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" data-testid="channel-link-qr" @click.self="qr = null">
        <div class="card w-full max-w-xs space-y-3 p-5 text-center">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ qr.link.name }}</h3>
          <img :src="qr.dataUrl" alt="QR" class="mx-auto h-56 w-56" />
          <p class="break-all font-mono text-xs text-gray-500">{{ shortUrl(qr.link) }}</p>
          <div class="flex justify-center gap-2">
            <a :href="qr.dataUrl" :download="`${qr.link.code}.png`" class="btn btn-primary btn-sm">{{ t('admin.channelLinks.downloadQR') }}</a>
            <button type="button" class="btn btn-secondary btn-sm" @click="qr = null">{{ t('admin.channelLinks.close') }}</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  createChannelLink,
  deleteChannelLink,
  listChannelLinks,
  updateChannelLink,
  type ChannelLink,
  type ChannelLinkInput
} from '@/api/admin/analytics'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t, te } = useI18n()
const appStore = useAppStore()

const SOURCES = ['xiaohongshu', 'bilibili', 'zhihu', 'v2ex', 'jike', 'juejin', 'wechat', 'guancha', 'douyin', 'school', 'kol']
const MEDIUMS = ['post', 'video', 'article', 'group', 'poster', 'partner']
const TARGETS = [
  { key: 'home', path: '/' },
  { key: 'register', path: '/register' },
  { key: 'pricing', path: '/pricing' },
  { key: 'connect', path: '/learn/connect/' },
  { key: 'codex', path: '/learn/codex/hivegpt' },
  { key: 'learn', path: '/learn/' },
  { key: 'editor', path: '/editor/' }
]

const links = ref<ChannelLink[]>([])
const loading = ref(false)
const saving = ref(false)
const form = ref<(ChannelLinkInput & { code: string }) | null>(null)
const editingId = ref<number | null>(null)
const sourceChoice = ref('xiaohongshu')
const targetChoice = ref('/')
const qr = ref<{ link: ChannelLink; dataUrl: string } | null>(null)

const origin = typeof window === 'undefined' ? '' : window.location.origin
const shortBase = computed(() => `${origin}/go/`)
const shortUrl = (l: ChannelLink) => `${origin}/go/${l.code}`

function sourceLabel(s: string): string {
  return te(`admin.channelLinks.source.${s}`) ? t(`admin.channelLinks.source.${s}`) : s
}
function mediumLabel(m: string): string {
  return te(`admin.channelLinks.medium.${m}`) ? t(`admin.channelLinks.medium.${m}`) : m
}

watch(sourceChoice, (v) => {
  if (form.value && v !== '__custom') form.value.source = v
})
watch(targetChoice, (v) => {
  if (form.value && v !== '__custom') form.value.target_path = v
})

async function load(): Promise<void> {
  loading.value = true
  try {
    links.value = await listChannelLinks()
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  } finally {
    loading.value = false
  }
}

function startCreate(): void {
  editingId.value = null
  form.value = { code: '', name: '', source: 'xiaohongshu', medium: 'post', target_path: '/', aff_code: '', note: '' }
  sourceChoice.value = 'xiaohongshu'
  targetChoice.value = '/'
}

function startEdit(l: ChannelLink): void {
  editingId.value = l.id
  form.value = { code: l.code, name: l.name, source: l.source, medium: l.medium, target_path: l.target_path, aff_code: l.aff_code, note: l.note }
  sourceChoice.value = SOURCES.includes(l.source) ? l.source : '__custom'
  targetChoice.value = TARGETS.some((p) => p.path === l.target_path) ? l.target_path : '__custom'
}

async function save(): Promise<void> {
  if (!form.value || saving.value) return
  saving.value = true
  try {
    const input: ChannelLinkInput = { ...form.value, aff_code: form.value.aff_code.trim().toUpperCase() }
    const saved = editingId.value ? await updateChannelLink(editingId.value, input) : await createChannelLink(input)
    appStore.showSuccess(t('admin.channelLinks.saved'))
    const created = !editingId.value
    form.value = null
    editingId.value = null
    await load()
    if (created) await copy(shortUrl(saved))
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  } finally {
    saving.value = false
  }
}

async function remove(l: ChannelLink): Promise<void> {
  if (!window.confirm(t('admin.channelLinks.confirmDelete', { name: l.name }))) return
  try {
    await deleteChannelLink(l.id)
    await load()
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  }
}

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess(t('admin.channelLinks.copied', { url: text }))
  } catch {
    window.prompt(t('admin.channelLinks.copyManually'), text)
  }
}

async function showQR(l: ChannelLink): Promise<void> {
  try {
    const dataUrl = await QRCode.toDataURL(shortUrl(l), { width: 640, margin: 2, errorCorrectionLevel: 'M' })
    qr.value = { link: l, dataUrl }
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.analytics.loadFailed')))
  }
}

onMounted(load)
</script>
