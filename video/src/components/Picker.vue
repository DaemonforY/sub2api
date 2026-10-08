<template>
  <div ref="root" class="relative">
    <button type="button" class="chip max-w-[14rem]" :aria-label="label" @click="open = !open">
      <component :is="iconComp" class="h-3.5 w-3.5 shrink-0 text-brand-500" />
      <span class="truncate">{{ current?.label || label }}</span>
      <ChevronDown class="h-3.5 w-3.5 shrink-0 text-ink-400" />
    </button>
    <div v-if="open" class="card absolute left-0 top-full z-30 mt-1 max-h-80 w-64 overflow-y-auto p-1 shadow-xl thin-scroll">
      <p class="px-3 pb-1 pt-2 text-[11px] font-medium text-ink-400">{{ label }}</p>
      <button
        v-for="o in options"
        :key="o.value"
        type="button"
        class="flex w-full items-start gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-ink-50 dark:hover:bg-ink-800"
        :class="{ 'bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300': o.value === modelValue }"
        @click="pick(o.value)"
      >
        <span class="min-w-0 flex-1">
          <span class="block truncate">{{ o.label }}</span>
          <span v-if="o.hint" class="block truncate text-xs text-ink-400">{{ o.hint }}</span>
        </span>
        <Check v-if="o.value === modelValue" class="mt-0.5 h-4 w-4 shrink-0" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Check, ChevronDown, Clock, Cpu, Mic } from 'lucide-vue-next'

const props = defineProps({ modelValue: { type: [String, Number], default: '' }, options: { type: Array, default: () => [] }, label: { type: String, default: '' }, icon: { type: String, default: 'cpu' } })
const emit = defineEmits(['update:modelValue'])
const open = ref(false)
const root = ref(null)
const current = computed(() => props.options.find((o) => o.value === props.modelValue))
const iconComp = computed(() => ({ cpu: Cpu, mic: Mic, clock: Clock })[props.icon] || Cpu)
function pick(v) {
  emit('update:modelValue', v)
  open.value = false
}
const outside = (e) => {
  if (root.value && !root.value.contains(e.target)) open.value = false
}
onMounted(() => document.addEventListener('mousedown', outside))
onBeforeUnmount(() => document.removeEventListener('mousedown', outside))
</script>
