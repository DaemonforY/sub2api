<template>
  <BaseDialog :show="show" :title="t('chat.roleTitle')" width="wide" @close="$emit('close')">
    <p class="mb-3 text-sm text-gray-500 dark:text-dark-400">{{ t('chat.roleHint') }}</p>
    <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
      <button
        v-for="r in CHAT_ROLES"
        :key="r.id"
        type="button"
        class="flex flex-col items-start gap-1 rounded-xl border px-3 py-2.5 text-left text-sm transition"
        :class="draft === r.prompt ? 'border-primary-400 bg-primary-50 text-primary-800 dark:border-primary-600 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 hover:border-primary-300 dark:border-dark-700 dark:text-dark-200'"
        :data-testid="`chat-role-${r.id}`"
        @click="draft = r.prompt"
      >
        <span class="text-lg leading-none">{{ r.icon }}</span>
        <span class="font-medium">{{ t(`chat.roles.${r.id}`) }}</span>
      </button>
    </div>
    <label class="mb-1 mt-4 block text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('chat.systemPrompt') }}</label>
    <textarea
      v-model="draft"
      rows="6"
      maxlength="8000"
      class="input w-full resize-y text-sm"
      :placeholder="t('chat.systemPromptPlaceholder')"
      data-testid="chat-system-prompt"
    ></textarea>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" data-testid="chat-role-apply" @click="$emit('apply', draft.trim())">{{ t('chat.roleApply') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { CHAT_ROLES } from './chatRoles'

const props = defineProps<{ show: boolean; prompt: string }>()
defineEmits<{ (e: 'close'): void; (e: 'apply', prompt: string): void }>()
const { t } = useI18n()
const draft = ref(props.prompt)

watch(
  () => props.show,
  (open) => {
    if (open) draft.value = props.prompt
  }
)
</script>
