<template>
  <div class="space-y-3">
    <label class="block text-sm">
      <span class="input-label">{{ t('admin.courses.fields.link') }}</span>
      <input v-model="model.link" class="input" placeholder="https://pan.baidu.com/s/..." :data-testid="`${prefix}-link`" />
    </label>
    <div class="grid gap-3 md:grid-cols-2">
      <label class="text-sm">
        <span class="input-label">{{ t('admin.courses.fields.code') }}</span>
        <input v-model="model.code" class="input" maxlength="32" :data-testid="`${prefix}-code`" />
      </label>
      <label class="text-sm">
        <span class="input-label">{{ t('admin.courses.fields.password') }}</span>
        <input v-model="model.password" class="input" maxlength="64" />
      </label>
    </div>
    <label class="block text-sm">
      <span class="input-label">{{ t('admin.courses.fields.note') }}</span>
      <textarea v-model="model.note" rows="2" class="input" maxlength="2000"></textarea>
    </label>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { DeliveryInput } from '@/api/admin/courses'

// Netdisk link / code / unzip password / note (v-model: the parent's delivery input).
const model = defineModel<DeliveryInput>({ required: true })
defineProps<{ prefix: string }>()
const { t } = useI18n()
</script>
