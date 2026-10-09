<template>
  <footer class="mt-16 border-t border-ink-200 bg-white dark:border-ink-800 dark:bg-ink-950">
    <div class="mx-auto grid max-w-[1440px] gap-8 px-6 py-12 sm:grid-cols-2 lg:grid-cols-5">
      <div class="lg:col-span-2">
        <Logo />
        <p class="mt-3 max-w-sm text-sm leading-6 text-ink-500">输入一个主题，AI 写脚本、配音、做动画，几分钟得到带字幕的讲解视频；也能一句话做出 Logo、图表、流程图等网页动画。用 HiveGPT 账号登录，按用量从余额扣费。</p>
      </div>
      <div>
        <h3 class="text-sm font-semibold">产品</h3>
        <ul class="mt-3 space-y-2 text-sm text-ink-500">
          <li><RouterLink to="/?mode=film" class="hover:text-brand-600">HTML 视频生成</RouterLink></li>
          <li><RouterLink to="/?mode=motion" class="hover:text-brand-600">AI 动画生成</RouterLink></li>
          <li><RouterLink to="/gallery" class="hover:text-brand-600">案例库</RouterLink></li>
          <li><RouterLink to="/pricing" class="hover:text-brand-600">价格说明</RouterLink></li>
        </ul>
      </div>
      <div>
        <h3 class="text-sm font-semibold">工具</h3>
        <ul class="mt-3 space-y-2 text-sm text-ink-500">
          <li v-for="t in tools" :key="t.slug"><RouterLink :to="`/tools/${t.slug}`" class="hover:text-brand-600">{{ t.name }}</RouterLink></li>
        </ul>
      </div>
      <div>
        <h3 class="text-sm font-semibold">HiveGPT</h3>
        <ul class="mt-3 space-y-2 text-sm text-ink-500">
          <li><a :href="MAIN_SITE_URL" target="_blank" rel="noopener" class="hover:text-brand-600">HiveGPT 主站</a></li>
          <li><a href="https://canvas.hivegpt.cn" target="_blank" rel="noopener" class="hover:text-brand-600">无限画布</a></li>
          <li><a :href="`${MAIN_SITE_URL}/learn/`" target="_blank" rel="noopener" class="hover:text-brand-600">AI 学习</a></li>
        </ul>
      </div>
    </div>
    <p class="border-t border-ink-100 py-4 text-center text-xs text-ink-400 dark:border-ink-900">© {{ year }} HiveGPT · AI 生成内容仅供参考，发布前请自行核实</p>
  </footer>
</template>

<script setup>
import Logo from './Logo.vue'
import { MAIN_SITE_URL } from '../lib/api'
import { TOOLS } from '../lib/tools'

const tools = TOOLS.slice(0, 6)
const year = new Date().getFullYear()
</script>
