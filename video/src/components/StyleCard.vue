<template>
  <button
    type="button"
    class="group relative overflow-hidden rounded-2xl border text-left transition"
    :class="selected ? 'border-brand-500 ring-2 ring-brand-500/40' : 'border-ink-200 hover:border-brand-300 dark:border-ink-700'"
    @click="$emit('pick', item.id)"
  >
    <!-- A small title card painted with the style's own theme. -->
    <div class="relative aspect-video w-full overflow-hidden" :style="{ background: th.bg, color: th.text, fontFamily: th.font }">
      <template v-if="item.id === 'auto'">
        <div class="absolute inset-0 bg-gradient-to-br from-orange-200 via-rose-100 to-sky-200 opacity-80 dark:opacity-30"></div>
        <Sparkles class="absolute left-1/2 top-1/2 h-8 w-8 -translate-x-1/2 -translate-y-1/2 text-brand-600" />
      </template>
      <template v-else>
        <div class="absolute left-[8%] top-[14%] h-[6%] w-[28%] rounded-full" :style="{ background: th.accent }"></div>
        <div class="absolute left-[8%] top-[26%] text-[15px] font-bold leading-tight" :style="{ fontFamily: th.display }">{{ sample }}</div>
        <div class="absolute left-[8%] top-[48%] h-[5%] w-[46%] rounded-full opacity-40" :style="{ background: th.sub }"></div>
        <div class="absolute left-[8%] top-[58%] h-[5%] w-[36%] rounded-full opacity-30" :style="{ background: th.sub }"></div>
        <div class="absolute bottom-[12%] right-[8%] flex h-[46%] items-end gap-[6%]" style="width: 34%">
          <span v-for="(h, i) in [45, 70, 55, 92]" :key="i" class="flex-1 transition-all duration-500 group-hover:h-full" :style="{ height: h + '%', background: i === 3 ? th.accent : th.accent2, borderRadius: Math.min(th.radius, 6) + 'px', opacity: i === 3 ? 1 : 0.55 }"></span>
        </div>
        <div class="absolute inset-x-[8%] bottom-[8%] h-px opacity-30" :style="{ background: th.border }"></div>
      </template>
    </div>
    <div class="flex items-center justify-between gap-2 bg-white px-3 py-2 dark:bg-ink-900">
      <span class="truncate text-sm font-medium">{{ item.name }}</span>
      <Check v-if="selected" class="h-4 w-4 shrink-0 text-brand-600" />
    </div>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { Check, Sparkles } from 'lucide-vue-next'

const props = defineProps({ item: { type: Object, required: true }, selected: Boolean })
defineEmits(['pick'])
const th = computed(() => props.item.theme)
const sample = computed(() => (props.item.genre === 'promo' ? '更快，更强' : props.item.genre === 'story' ? '很久很久以前' : '原理一览'))
</script>
