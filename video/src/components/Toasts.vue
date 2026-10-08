<template>
  <div class="pointer-events-none fixed inset-x-0 top-4 z-[100] flex flex-col items-center gap-2 px-4">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="pointer-events-auto max-w-md rounded-xl px-4 py-2.5 text-sm shadow-lg"
      :class="t.kind === 'error' ? 'bg-red-600 text-white' : t.kind === 'success' ? 'bg-emerald-600 text-white' : 'bg-ink-900 text-white dark:bg-white dark:text-ink-900'"
      role="status"
    >
      {{ t.text }}
    </div>
  </div>
</template>

<script setup>
import { toasts } from '../lib/toast'
</script>
