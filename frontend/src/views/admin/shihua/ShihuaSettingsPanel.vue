<template>
  <div class="card space-y-4 p-5">
    <div class="grid gap-4 md:grid-cols-3">
      <label>
        <span class="input-label">{{ t('shihua.settings.key') }}</span>
        <select v-model.number="form.key_id" class="input" data-testid="shihua-key">
          <option :value="0">{{ t('shihua.settings.noKey') }}</option>
          <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? ` · ${k.group.name}` : '' }}</option>
        </select>
      </label>
      <label>
        <span class="input-label">{{ t('shihua.settings.textModel') }}</span>
        <input v-model="form.text_model" class="input" />
      </label>
      <div class="grid grid-cols-2 gap-3">
        <label>
          <span class="input-label">{{ t('shihua.settings.size') }}</span>
          <select v-model="form.size" class="input">
            <option v-for="s in sizes" :key="s" :value="s">{{ s }}</option>
          </select>
        </label>
        <label>
          <span class="input-label">{{ t('shihua.settings.quality') }}</span>
          <select v-model="form.quality" class="input">
            <option v-for="q in qualities" :key="q" :value="q">{{ q }}</option>
          </select>
        </label>
      </div>
      <label>
        <span class="input-label">{{ t('shihua.settings.concurrency') }}</span>
        <input v-model.number="form.concurrency" type="number" min="1" max="4" class="input" />
      </label>
      <label>
        <span class="input-label">{{ t('shihua.settings.dailyLimit') }}</span>
        <input v-model.number="form.daily_limit" type="number" min="0" max="500" class="input" data-testid="shihua-daily-limit" />
      </label>
      <div class="space-y-2 pt-6 text-sm text-gray-700 dark:text-gray-300">
        <label class="flex items-center gap-2"><Toggle v-model="form.auto_upscale" /> {{ t('shihua.settings.autoUpscale') }}</label>
        <label class="flex items-center gap-2"><Toggle v-model="form.two_people_verify" /> {{ t('shihua.settings.twoPeople') }}</label>
      </div>
    </div>
    <p v-if="settings && !settings.upscale_available" class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
      {{ t('shihua.settings.upscaleUnavailable') }}
    </p>
    <label class="block">
      <span class="input-label flex items-center justify-between">
        {{ t('shihua.settings.stylePrompt') }}
        <button class="text-xs text-primary-600 hover:underline" @click="form.style_prompt = settings?.default_style_prompt ?? ''">{{ t('shihua.settings.resetStyle') }}</button>
      </span>
      <textarea v-model="form.style_prompt" rows="12" class="input text-xs" :placeholder="settings?.default_style_prompt" />
    </label>
    <div class="flex justify-end">
      <button class="btn btn-primary" :disabled="saving" data-testid="shihua-save-settings" @click="save">{{ t('shihua.settings.save') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import * as api from '@/api/admin/shihua'

const props = defineProps<{ settings: api.ShihuaSettingsView | null }>()
const emit = defineEmits<{ (e: 'saved', value: api.ShihuaSettingsView): void }>()
const { t } = useI18n()
const appStore = useAppStore()
const sizes = ['1024x1536', '1536x1024', '1024x1024']
const qualities = ['high', 'medium', 'low']
const keys = ref<ApiKey[]>([])
const saving = ref(false)
const form = reactive<api.ShihuaSettings>({
  key_id: 0, text_model: '', size: '1024x1536', quality: 'high', concurrency: 2, daily_limit: 50, style_prompt: '', auto_upscale: true, two_people_verify: false
})

function apply(v: api.ShihuaSettingsView | null) {
  if (!v) return
  Object.assign(form, {
    key_id: v.key_id, text_model: v.text_model, size: v.size, quality: v.quality, concurrency: v.concurrency, daily_limit: v.daily_limit,
    style_prompt: v.style_prompt, auto_upscale: v.auto_upscale, two_people_verify: v.two_people_verify
  })
}
watch(() => props.settings, apply, { immediate: true })

async function save() {
  saving.value = true
  try {
    const v = await api.saveSettings({ ...form })
    emit('saved', v)
    appStore.showSuccess(t('shihua.settings.saved'))
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    keys.value = (await keysAPI.list(1, 100, { status: 'active' })).items
  } catch {
    keys.value = []
  }
})
</script>
