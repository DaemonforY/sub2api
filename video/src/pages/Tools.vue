<template>
  <div class="mx-auto max-w-[1280px] px-4 py-10 sm:px-6">
    <section class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-brand-500 to-rose-500 p-8 text-white sm:p-10">
      <div class="bg-grid absolute inset-0 opacity-30"></div>
      <div class="relative max-w-2xl">
        <p class="text-sm font-medium opacity-80">热门推荐</p>
        <h1 class="mt-2 text-3xl font-bold">{{ hero.name }}</h1>
        <p class="mt-3 leading-7 opacity-90">{{ hero.summary }}</p>
        <RouterLink :to="`/tools/${hero.slug}`" class="mt-6 inline-flex items-center gap-1.5 rounded-xl bg-white px-5 py-2.5 text-sm font-semibold text-brand-700 shadow">立即使用 <ArrowRight class="h-4 w-4" /></RouterLink>
      </div>
    </section>
    <div class="mt-10 flex items-end justify-between">
      <h2 class="text-xl font-bold">全部工具 <span class="ml-1 text-sm font-normal text-ink-400">{{ TOOLS.length }} 款</span></h2>
    </div>
    <div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink v-for="t in TOOLS" :key="t.slug" :to="`/tools/${t.slug}`" class="card group flex gap-4 p-5 transition hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-soft">
        <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-brand-50 text-brand-600 transition group-hover:bg-brand-500 group-hover:text-white dark:bg-brand-900/30">
          <component :is="TOOL_ICONS[t.icon]" class="h-6 w-6" />
        </span>
        <span class="min-w-0">
          <span class="block font-semibold">{{ t.name }}</span>
          <span class="mt-1 line-clamp-2 block text-sm leading-6 text-ink-500">{{ t.headline }}</span>
        </span>
      </RouterLink>
    </div>
    <div class="card mt-10 flex flex-wrap items-center justify-between gap-4 p-6">
      <div>
        <h3 class="font-semibold">找不到合适的工具？</h3>
        <p class="mt-1 text-sm text-ink-500">直接进入创作页，用自然语言描述你的需求。</p>
      </div>
      <RouterLink to="/" class="btn-primary">自由创作</RouterLink>
    </div>
  </div>
</template>

<script setup>
import { ArrowRight } from 'lucide-vue-next'
import { TOOLS } from '../lib/tools'
import { TOOL_ICONS } from '../lib/icons'

const hero = TOOLS[0]
</script>
