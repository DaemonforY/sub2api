<template>
  <AppLayout>
    <div class="space-y-4" data-testid="goods-admin">
      <!-- List -->
      <template v-if="!editing">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('goods.admin.hint') }}</p>
          <button class="btn btn-primary" data-testid="goods-new" @click="startNew">{{ t('goods.admin.new') }}</button>
        </div>
        <div v-if="!rows.length && !loading" class="card py-16 text-center text-sm text-gray-500">{{ t('goods.admin.empty') }}</div>
        <div v-else class="card overflow-x-auto">
          <table class="w-full whitespace-nowrap text-sm" data-testid="goods-table">
            <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-2">{{ t('goods.admin.fields.name') }}</th>
                <th class="px-4 py-2">{{ t('goods.admin.fields.status') }}</th>
                <th class="px-4 py-2">{{ t('goods.admin.fields.price') }}</th>
                <th class="px-4 py-2" :title="t('goods.admin.forSaleHint')">{{ t('goods.admin.forSale') }}</th>
                <th class="px-4 py-2">{{ t('goods.admin.sold') }}</th>
                <th class="px-4 py-2" :title="t('goods.admin.waitingHint')">{{ t('goods.admin.waiting') }}</th>
                <th class="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="g in rows" :key="g.id">
                <td class="min-w-[12rem] whitespace-normal px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ g.name }}</div>
                  <a :href="`/goods/${g.slug}`" target="_blank" rel="noopener" class="text-xs text-primary-600 hover:underline dark:text-primary-400">/goods/{{ g.slug }}</a>
                </td>
                <td class="px-4 py-3">
                  <span :class="['badge', g.enabled ? 'badge-success' : 'badge-gray']">{{ g.enabled ? t('goods.admin.onSale') : t('goods.admin.offSale') }}</span>
                </td>
                <td class="px-4 py-3">¥{{ g.price }}</td>
                <td class="px-4 py-3" :class="forSale(g) <= 0 ? 'font-medium text-rose-600' : forSale(g) <= 5 ? 'text-amber-600' : ''">{{ forSale(g) }}</td>
                <td class="px-4 py-3">{{ g.stock.sold }}</td>
                <td class="px-4 py-3" :class="g.waiting ? 'font-medium text-rose-600' : ''">{{ g.waiting }}</td>
                <td class="space-x-2 px-4 py-3 text-right">
                  <button class="btn btn-secondary btn-sm" @click="open(g, 'stock')">{{ t('goods.admin.manageStock') }}</button>
                  <button class="btn btn-secondary btn-sm" @click="open(g, 'basic')">{{ t('common.edit') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- Editor -->
      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <button class="btn btn-secondary btn-sm" @click="closeEditor">← {{ t('goods.admin.backToList') }}</button>
          <div v-if="form.id" class="flex gap-1">
            <button v-for="tab in ['basic', 'stock'] as const" :key="tab" :class="['btn btn-sm', panel === tab ? 'btn-primary' : 'btn-secondary']" :data-testid="`goods-tab-${tab}`" @click="switchPanel(tab)">
              {{ t(`goods.admin.tabs.${tab}`) }}
            </button>
          </div>
        </div>

        <!-- Basic -->
        <form v-if="panel === 'basic'" class="card space-y-4 p-5" data-testid="goods-form" @submit.prevent="save">
          <div class="grid gap-4 md:grid-cols-2">
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.name') }}</span>
              <input v-model="form.name" class="input" maxlength="60" required data-testid="goods-name" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.slug') }}</span>
              <input v-model="form.slug" class="input" maxlength="40" pattern="[a-z0-9][a-z0-9\-]*" required data-testid="goods-slug" />
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('goods.admin.slugHint', { slug: form.slug || 'muse' }) }}</span>
            </label>
            <label class="text-sm md:col-span-2">
              <span class="input-label">{{ t('goods.admin.fields.subtitle') }}</span>
              <input v-model="form.subtitle" class="input" maxlength="120" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.price') }}</span>
              <input v-model.number="form.price" type="number" min="0.01" step="0.01" class="input" required data-testid="goods-price" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.originalPrice') }}</span>
              <input v-model.number="form.original_price" type="number" min="0" step="0.01" class="input" />
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.perUserLimit') }}</span>
              <input v-model.number="form.per_user_limit" type="number" min="0" max="1000" class="input" />
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('goods.admin.perUserLimitHint') }}</span>
            </label>
            <label class="text-sm">
              <span class="input-label">{{ t('goods.admin.fields.sortOrder') }}</span>
              <input v-model.number="form.sort_order" type="number" class="input" />
            </label>
          </div>
          <label class="block text-sm">
            <span class="input-label">{{ t('goods.admin.fields.description') }}</span>
            <textarea v-model="form.description" rows="8" class="input font-mono text-xs" maxlength="5000" :placeholder="t('goods.admin.descriptionPlaceholder')"></textarea>
          </label>
          <label class="block text-sm">
            <span class="input-label">{{ t('goods.admin.fields.usageNote') }}</span>
            <textarea v-model="form.usage_note" rows="4" class="input font-mono text-xs" maxlength="5000" :placeholder="t('goods.admin.usageNotePlaceholder')"></textarea>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('goods.admin.usageNoteHint') }}</span>
          </label>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4" data-testid="goods-enabled" />
            {{ t('goods.admin.fields.enabled') }}
          </label>
          <div class="flex flex-wrap gap-2">
            <button type="submit" class="btn btn-primary" :disabled="busy" data-testid="goods-save">{{ t('common.save') }}</button>
            <button v-if="form.id" type="button" class="btn btn-danger" :disabled="busy" @click="confirmDelete = true">{{ t('common.delete') }}</button>
          </div>
        </form>

        <!-- Stock -->
        <template v-else>
          <div v-if="current" class="grid gap-3 sm:grid-cols-4">
            <div v-for="key in ['forSale', 'available', 'reserved', 'sold'] as const" :key="key" class="card p-4">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t(`goods.admin.stat.${key}`) }}</p>
              <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ key === 'forSale' ? forSale(current) : current.stock[key] }}</p>
            </div>
          </div>
          <p v-if="current?.waiting" class="rounded-xl border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700 dark:border-rose-800/60 dark:bg-rose-900/20 dark:text-rose-200">{{ t('goods.admin.waitingBanner', { n: current.waiting }) }}</p>

          <form class="card space-y-3 p-5" data-testid="goods-import" @submit.prevent="importKeys">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('goods.admin.importTitle') }}</h3>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('goods.admin.importHint') }}</p>
            <textarea v-model="importText" rows="8" class="input font-mono text-xs" :placeholder="t('goods.admin.importPlaceholder')" data-testid="goods-import-keys"></textarea>
            <div class="flex flex-wrap items-end gap-3">
              <label class="text-sm">
                <span class="input-label">{{ t('goods.admin.batchNote') }}</span>
                <input v-model="batchNote" class="input w-64" maxlength="60" :placeholder="t('goods.admin.batchNotePlaceholder')" />
              </label>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('goods.admin.importCount', { n: importLines }) }}</span>
              <button type="submit" class="btn btn-primary" :disabled="busy || !importLines" data-testid="goods-import-submit">{{ t('goods.admin.import') }}</button>
            </div>
          </form>

          <div class="card p-5" data-testid="goods-items">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div class="flex gap-1">
                <button v-for="s in ['', 'available', 'sold', 'void'] as const" :key="s" :class="['btn btn-sm', itemStatus === s ? 'btn-primary' : 'btn-secondary']" @click="setItemStatus(s)">
                  {{ t(`goods.admin.itemStatus.${s || 'all'}`) }}
                </button>
              </div>
              <button class="btn btn-secondary btn-sm" :disabled="!selected.length || busy" data-testid="goods-void" @click="voidSelected">{{ t('goods.admin.voidSelected', { n: selected.length }) }}</button>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full whitespace-nowrap text-sm">
                <thead class="text-left text-xs text-gray-500">
                  <tr>
                    <th class="w-8 py-2 pr-2"></th>
                    <th class="py-2 pr-3">ID</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.key') }}</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.fields.status') }}</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.buyer') }}</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.order') }}</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.batchNote') }}</th>
                    <th class="py-2 pr-3">{{ t('goods.admin.time') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="it in items" :key="it.id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-2 pr-2"><input v-if="it.status === 'available'" v-model="selected" :value="it.id" type="checkbox" class="h-4 w-4" /></td>
                    <td class="py-2 pr-3 text-gray-500">{{ it.id }}</td>
                    <td class="py-2 pr-3 font-mono text-xs text-gray-700 dark:text-gray-300">{{ it.preview }}</td>
                    <td class="py-2 pr-3"><span :class="['badge', itemBadge(it.status)]">{{ t(`goods.admin.itemStatus.${it.status}`) }}</span></td>
                    <td class="py-2 pr-3 text-gray-700 dark:text-gray-300">{{ it.user_id ? `${it.username || displayEmail(it.user_email, t)} (#${it.user_id})` : '—' }}</td>
                    <td class="py-2 pr-3 text-gray-500">{{ it.order_id ? `#${it.order_id}` : '—' }}</td>
                    <td class="py-2 pr-3 text-gray-500">{{ it.batch_note || '—' }}</td>
                    <td class="py-2 pr-3 text-xs text-gray-500">{{ dateText(it.delivered_at || it.created_at) }}</td>
                  </tr>
                  <tr v-if="!items.length">
                    <td colspan="8" class="py-8 text-center text-gray-500">{{ t('goods.admin.noItems') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-if="itemTotal > pageSize" class="mt-3 flex items-center justify-end gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="itemPage <= 1" @click="loadItems(itemPage - 1)">‹</button>
              <span class="text-sm text-gray-500">{{ itemPage }} / {{ Math.ceil(itemTotal / pageSize) }}</span>
              <button class="btn btn-secondary btn-sm" :disabled="itemPage * pageSize >= itemTotal" @click="loadItems(itemPage + 1)">›</button>
            </div>
          </div>
        </template>
      </template>
    </div>

    <ConfirmDialog
      :show="confirmDelete"
      :title="t('goods.admin.deleteTitle')"
      :message="t('goods.admin.deleteMessage', { name: form.name })"
      danger
      @confirm="remove"
      @cancel="confirmDelete = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { displayEmail } from '@/utils/userDisplay'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import {
  adminDeleteGoods,
  adminImportGoodsKeys,
  adminListGoods,
  adminListGoodsItems,
  adminSaveGoods,
  adminVoidGoodsItems,
  type Goods,
  type GoodsAdminRow,
  type GoodsItem,
  type GoodsItemStatus,
} from '@/api/goods'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const pageSize = 50
const rows = ref<GoodsAdminRow[]>([])
const loading = ref(true)
const busy = ref(false)
const editing = ref(false)
const panel = ref<'basic' | 'stock'>('basic')
const form = ref<Goods>(emptyGoods())
const confirmDelete = ref(false)

const importText = ref('')
const batchNote = ref('')
const items = ref<GoodsItem[]>([])
const itemTotal = ref(0)
const itemPage = ref(1)
const itemStatus = ref<'' | GoodsItemStatus>('')
const selected = ref<number[]>([])

const current = computed(() => rows.value.find((r) => r.id === form.value.id) ?? null)
const importLines = computed(() => importText.value.split('\n').filter((l) => l.trim()).length)

function emptyGoods(): Goods {
  return { id: 0, slug: '', name: '', subtitle: '', description: '', usage_note: '', price: 0, original_price: 0, per_user_limit: 0, enabled: false, sort_order: 0 }
}

function forSale(g: GoodsAdminRow): number {
  return Math.max(0, g.stock.available - g.stock.reserved)
}

function itemBadge(status: GoodsItemStatus): string {
  return status === 'available' ? 'badge-success' : status === 'sold' ? 'badge-primary' : 'badge-gray'
}

function dateText(value?: string): string {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}

function showError(err: unknown) {
  appStore.showError(extractApiErrorMessage(err, t('common.error')))
}

async function load() {
  loading.value = true
  try {
    rows.value = await adminListGoods()
  } catch (err: unknown) {
    showError(err)
  } finally {
    loading.value = false
  }
}

function startNew() {
  form.value = emptyGoods()
  panel.value = 'basic'
  editing.value = true
}

function open(g: GoodsAdminRow, tab: 'basic' | 'stock') {
  const { stock: _stock, waiting: _waiting, ...goods } = g
  form.value = { ...goods }
  editing.value = true
  switchPanel(tab)
}

function switchPanel(tab: 'basic' | 'stock') {
  panel.value = tab
  if (tab === 'stock') {
    itemStatus.value = ''
    void loadItems(1)
  }
}

function closeEditor() {
  editing.value = false
  void load()
}

async function save() {
  busy.value = true
  try {
    const saved = await adminSaveGoods(form.value)
    const isNew = !form.value.id
    form.value = { ...form.value, ...saved }
    appStore.showSuccess(t('goods.admin.saved'))
    await load()
    // A new product goes straight to its stock, where the keys are imported.
    if (isNew) switchPanel('stock')
  } catch (err: unknown) {
    showError(err)
  } finally {
    busy.value = false
  }
}

async function remove() {
  confirmDelete.value = false
  busy.value = true
  try {
    await adminDeleteGoods(form.value.id)
    appStore.showSuccess(t('goods.admin.deleted'))
    closeEditor()
  } catch (err: unknown) {
    showError(err)
  } finally {
    busy.value = false
  }
}

async function importKeys() {
  if (!form.value.id || !importLines.value) return
  busy.value = true
  try {
    const res = await adminImportGoodsKeys(form.value.id, importText.value, batchNote.value)
    appStore.showSuccess(t('goods.admin.imported', { added: res.added, duplicates: res.duplicates, retried: res.retried }))
    importText.value = ''
    await Promise.all([load(), loadItems(1)])
  } catch (err: unknown) {
    showError(err)
  } finally {
    busy.value = false
  }
}

async function loadItems(page: number) {
  if (!form.value.id) return
  try {
    const res = await adminListGoodsItems(form.value.id, { status: itemStatus.value || undefined, page, page_size: pageSize })
    items.value = res.items
    itemTotal.value = res.total
    itemPage.value = page
    selected.value = []
  } catch (err: unknown) {
    showError(err)
  }
}

function setItemStatus(s: '' | GoodsItemStatus) {
  itemStatus.value = s
  void loadItems(1)
}

async function voidSelected() {
  if (!selected.value.length) return
  busy.value = true
  try {
    const res = await adminVoidGoodsItems(form.value.id, selected.value)
    appStore.showSuccess(t('goods.admin.voided', { n: res.voided }))
    await Promise.all([load(), loadItems(itemPage.value)])
  } catch (err: unknown) {
    showError(err)
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>
