<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @mousedown.self="$emit('close')">
    <div class="card w-full max-w-lg p-6 shadow-2xl" role="dialog" aria-modal="true">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold">{{ title }}</h2>
        <button class="rounded-lg p-1 text-ink-400 hover:bg-ink-100 dark:hover:bg-ink-800" aria-label="关闭" @click="$emit('close')"><X class="h-5 w-5" /></button>
      </div>
      <div class="mt-4"><slot /></div>
    </div>
  </div>
</template>

<script setup>
import { X } from 'lucide-vue-next'
defineProps({ title: { type: String, default: '' } })
defineEmits(['close'])
</script>
