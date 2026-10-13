<template>
  <div class="space-y-4" data-testid="content-settings">
    <div class="card space-y-4 p-5">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="text-sm">
          <span class="input-label">{{ t('admin.content.settings.key') }}</span>
          <select v-model.number="form.key_id" class="input" data-testid="content-key">
            <option :value="0">{{ t('admin.content.settings.keyPlaceholder') }}</option>
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
          </select>
          <span class="input-hint">
            {{ t('admin.content.settings.keyHint') }}
            <a href="/keys" target="_blank" class="text-primary-600 hover:underline">{{ t('admin.content.settings.keyCreate') }} ↗</a>
          </span>
          <span v-if="form.key_problem" class="input-hint text-red-600">{{ t('admin.content.settings.keyProblem', { msg: form.key_problem }) }}</span>
        </label>
        <label class="text-sm">
          <span class="input-label">{{ t('admin.content.settings.model') }}</span>
          <input v-model.trim="form.model" class="input" placeholder="gpt-5.5" />
          <span class="input-hint">{{ t('admin.content.settings.modelHint') }}</span>
        </label>
      </div>

      <div class="text-sm">
        <span class="input-label">{{ t('admin.content.settings.platforms') }}</span>
        <div class="flex flex-wrap gap-3">
          <label v-for="p in platforms" :key="p.id" class="flex items-center gap-1.5">
            <input v-model="form.platforms" type="checkbox" :value="p.id" class="h-4 w-4" />
            {{ t(`admin.content.platform.${p.id}`) }}
          </label>
        </div>
      </div>

      <label class="flex items-start gap-2 text-sm">
        <input v-model="form.search" type="checkbox" class="mt-0.5 h-4 w-4" />
        <span>
          {{ t('admin.content.settings.search') }}
          <span v-if="!form.search_available" class="block text-xs text-amber-600">{{ t('admin.content.settings.searchUnavailable') }}</span>
        </span>
      </label>

      <label class="block text-sm">
        <span class="input-label">{{ t('admin.content.settings.product') }}</span>
        <textarea v-model="form.product" class="input min-h-[72px] text-sm" maxlength="1000" />
        <span class="input-hint">{{ t('admin.content.settings.productHint') }}</span>
      </label>
      <label class="block text-sm">
        <span class="input-label">{{ t('admin.content.settings.style') }}</span>
        <textarea v-model="form.style_prompt" class="input min-h-[96px] text-sm" maxlength="2000" />
        <span class="input-hint">
          {{ t('admin.content.settings.styleHint') }}
          <a href="/style" target="_blank" class="text-primary-600 hover:underline">/style ↗</a>
        </span>
      </label>
      <label class="block w-60 text-sm">
        <span class="input-label">{{ t('admin.content.settings.promoBonus') }}</span>
        <input v-model.number="form.promo_bonus" type="number" min="0" max="20" step="0.5" class="input" />
        <span class="input-hint">{{ t('admin.content.settings.promoBonusHint') }}</span>
      </label>

      <div class="space-y-3 rounded-xl border border-gray-100 p-4 dark:border-dark-700">
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.content.settings.notify') }}</div>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.content.settings.notifyHint') }}</p>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="text-sm">
            <span class="input-label">{{ t('admin.content.settings.notifyType') }}</span>
            <select v-model="form.notify_type" class="input" data-testid="content-notify-type">
              <option value="">{{ t('admin.content.settings.notifyNone') }}</option>
              <option value="serverchan">{{ t('admin.content.settings.notifyServerChan') }}</option>
              <option value="pushplus">{{ t('admin.content.settings.notifyPushPlus') }}</option>
              <option value="wecom">{{ t('admin.content.settings.notifyWecom') }}</option>
            </select>
          </label>
          <label v-if="form.notify_type" class="text-sm">
            <span class="input-label">{{ t('admin.content.settings.notifyKey') }}</span>
            <input
              v-model.trim="notifyKey"
              type="password"
              autocomplete="off"
              class="input"
              :placeholder="form.notify_key_set ? t('admin.content.settings.notifyKeySet') : ''"
              data-testid="content-notify-key"
            />
          </label>
        </div>
        <button v-if="form.notify_type && form.notify_key_set" class="btn btn-secondary btn-sm" :disabled="testing" @click="test">{{ t('admin.content.settings.notifyTest') }}</button>
      </div>

      <button class="btn btn-primary" :disabled="saving" data-testid="content-save" @click="save">{{ t('common.save') }}</button>
    </div>

    <div class="card space-y-3 p-5" data-testid="content-rules">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.content.settings.rules') }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.content.settings.rulesHint') }}</p>
      </div>
      <div class="flex gap-2">
        <input v-model.trim="newRule" class="input min-w-0 flex-1" maxlength="300" :placeholder="t('admin.content.settings.rulePlaceholder')" @keyup.enter="addRule" />
        <button class="btn btn-secondary" :disabled="!newRule" @click="addRule">{{ t('admin.content.settings.addRule') }}</button>
      </div>
      <ul class="divide-y divide-gray-100 dark:divide-dark-700">
        <li v-for="r in rules" :key="r.id" class="flex items-center gap-3 py-2 text-sm">
          <input type="checkbox" class="h-4 w-4" :checked="r.enabled" @change="toggleRule(r)" />
          <span :class="['flex-1', r.enabled ? '' : 'text-gray-400 line-through']">{{ r.rule }}</span>
          <span class="text-xs text-gray-400">{{ t(`admin.content.settings.ruleOrigin.${r.origin}`) }}</span>
          <button class="text-xs text-red-600 hover:underline" @click="removeRule(r)">{{ t('admin.content.topics.delete') }}</button>
        </li>
      </ul>
    </div>

    <div class="card p-5">
      <h3 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.content.settings.specs') }}</h3>
      <ul class="space-y-2 text-xs text-gray-600 dark:text-dark-300">
        <li v-for="p in platforms" :key="p.id">
          <span class="font-medium text-gray-900 dark:text-white">{{ t(`admin.content.platform.${p.id}`) }}</span>：
          {{ t('admin.content.settings.specTitle', { n: p.title_max }) }} · {{ t('admin.content.settings.specBody', { min: p.body_min, max: p.body_max }) }} ·
          {{ p.links ? t('admin.content.settings.specLinks') : p.read_more ? t('admin.content.settings.specReadMore') : t('admin.content.settings.specPromo') }}
          <span v-if="p.cards"> · {{ t('admin.content.settings.specCards') }}</span>
          <span v-if="p.micro_max"> · {{ t('admin.content.settings.specMicro', { n: p.micro_max }) }}</span> · {{ p.ai_label }}
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as contentAPI from '@/api/admin/content'
import type { ContentRule, ContentSettings, KeyOption, PlatformSpec } from '@/api/admin/content'

defineProps<{ platforms: PlatformSpec[] }>()
const emit = defineEmits<{ (e: 'saved', s: ContentSettings): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const form = reactive<ContentSettings>({
  enabled: false,
  key_id: 0,
  model: '',
  daily_time: '09:00',
  daily_count: 1,
  platforms: ['wechat', 'zhihu', 'csdn', 'toutiao', 'xiaohongshu'],
  search: true,
  style_prompt: '',
  product: '',
  promo_bonus: 1,
  notify_type: '',
  notify_key_set: false,
  search_available: false
})
const notifyKey = ref('')
const keys = ref<KeyOption[]>([])
const rules = ref<ContentRule[]>([])
const newRule = ref('')
const saving = ref(false)
const testing = ref(false)

async function load() {
  try {
    const [s, k, r] = await Promise.all([contentAPI.getSettings(), contentAPI.listKeys(), contentAPI.listRules()])
    Object.assign(form, s)
    keys.value = k
    rules.value = r
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    const out = await contentAPI.saveSettings({ ...form, notify_key: notifyKey.value })
    Object.assign(form, out)
    notifyKey.value = ''
    emit('saved', out)
    appStore.showSuccess(t('admin.content.settings.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  try {
    await contentAPI.testNotify()
    appStore.showSuccess(t('admin.content.settings.notifySent'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    testing.value = false
  }
}

async function addRule() {
  if (!newRule.value) return
  try {
    rules.value.push(await contentAPI.createRule(newRule.value))
    newRule.value = ''
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function toggleRule(r: ContentRule) {
  try {
    await contentAPI.setRuleEnabled(r.id, !r.enabled)
    r.enabled = !r.enabled
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}

async function removeRule(r: ContentRule) {
  try {
    await contentAPI.deleteRule(r.id)
    rules.value = rules.value.filter((x) => x.id !== r.id)
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  }
}
</script>
