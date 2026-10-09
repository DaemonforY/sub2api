<template>
  <a :href="ad.link" target="_blank" rel="sponsored noopener" class="group block overflow-hidden rounded-2xl border border-ink-200/80 bg-white transition hover:-translate-y-0.5 hover:shadow-soft dark:border-ink-800 dark:bg-ink-900">
    <div class="relative bg-ink-100 dark:bg-ink-800" style="aspect-ratio: 16 / 9">
      <img :src="ad.image" :alt="ad.title" loading="lazy" class="absolute inset-0 h-full w-full object-cover" />
      <span class="absolute left-2 top-2 rounded-md bg-black/55 px-1.5 py-0.5 text-[11px] text-white">广告</span>
    </div>
    <div class="p-3">
      <p class="line-clamp-2 min-h-[2.5rem] text-sm font-medium leading-5 group-hover:text-brand-600">{{ ad.title }}</p>
      <div class="mt-2 flex items-center gap-3 text-xs text-ink-500">
        <span class="truncate">{{ ad.text || '了解更多' }}</span>
        <span class="flex-1"></span>
        <span class="text-ink-400">Ads</span>
      </div>
    </div>
  </a>
</template>

<script setup>
// A gallery ad (admin settings on /review): image, title and link, marked 广告.
defineProps({ ad: { type: Object, required: true } })
</script>
