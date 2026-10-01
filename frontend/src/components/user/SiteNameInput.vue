<template>
  <div>
    <div class="flex items-stretch overflow-hidden rounded-lg border border-gray-300 focus-within:border-primary-500 dark:border-dark-600">
      <span class="flex items-center bg-gray-50 px-2 text-sm text-gray-500 dark:bg-dark-800 dark:text-dark-400">https://</span>
      <input
        :value="modelValue"
        class="min-w-0 flex-1 bg-transparent px-2 py-2 text-sm text-gray-900 outline-none dark:text-white"
        maxlength="30"
        autocomplete="off"
        spellcheck="false"
        :placeholder="placeholder"
        data-testid="site-name-input"
        @input="onInput"
      />
      <span class="flex items-center bg-gray-50 px-2 text-sm text-gray-500 dark:bg-dark-800 dark:text-dark-400">.{{ domain }}</span>
    </div>
    <p v-if="message" :class="['mt-1 text-xs', state === 'ok' ? 'text-emerald-600' : state === 'checking' ? 'text-gray-500' : 'text-red-600']" data-testid="site-name-status">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { checkSiteName, normalizeSiteName, SITE_NAME_PATTERN } from '@/api/sites'

const props = withDefaults(defineProps<{ modelValue: string; domain: string; siteId?: number; placeholder?: string; current?: string }>(), { siteId: 0, placeholder: '', current: '' })
const emit = defineEmits<{ 'update:modelValue': [value: string]; 'update:valid': [valid: boolean] }>()

const { t } = useI18n()
type State = 'empty' | 'same' | 'invalid' | 'checking' | 'ok' | 'taken'
const state = ref<State>('empty')
const reason = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
let seq = 0

const message = computed(() => {
  switch (state.value) {
    case 'invalid':
      return t('sites.name.rule')
    case 'checking':
      return t('sites.name.checking')
    case 'ok':
      return t('sites.name.available')
    case 'taken':
      return reason.value || t('sites.name.taken')
    default:
      return ''
  }
})

function onInput(event: Event) {
  emit('update:modelValue', normalizeSiteName((event.target as HTMLInputElement).value))
}

watch(
  () => props.modelValue,
  (name) => {
    clearTimeout(timer)
    const current = ++seq
    if (!name) return set('empty')
    if (name === props.current) return set('same')
    if (!SITE_NAME_PATTERN.test(name) || name.includes('--')) return set('invalid')
    set('checking')
    timer = setTimeout(async () => {
      try {
        const res = await checkSiteName(name, props.siteId)
        if (current !== seq) return
        reason.value = res.reason || ''
        set(res.available ? 'ok' : 'taken')
      } catch {
        if (current === seq) set('empty')
      }
    }, 350)
  },
  { immediate: true }
)

function set(next: State) {
  state.value = next
  // Empty is valid when creating (a random name is picked); otherwise only a confirmed free name is.
  emit('update:valid', next === 'ok' || (next === 'empty' && !props.current))
}

onBeforeUnmount(() => clearTimeout(timer))
</script>
