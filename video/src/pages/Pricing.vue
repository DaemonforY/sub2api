<template>
  <div class="mx-auto max-w-5xl px-4 py-12 sm:px-6">
    <section class="text-center">
      <h1 class="text-3xl font-bold sm:text-4xl">不卖积分，按用量计费</h1>
      <p class="mx-auto mt-4 max-w-2xl leading-7 text-ink-600 dark:text-ink-300">HiveGPT 视频用你的 HiveGPT 账号登录。写脚本和做动画调用的模型按实际用量从账户余额扣费，每一笔都记在主站「用量明细」里。配音、预览、导出 HTML 和发布到案例库都不另收费。</p>
    </section>
    <div class="mt-10 grid gap-4 md:grid-cols-3">
      <div v-for="c in cards" :key="c.title" class="card p-6">
        <component :is="c.icon" class="h-7 w-7 text-brand-500" />
        <h2 class="mt-4 font-semibold">{{ c.title }}</h2>
        <p class="mt-2 text-sm leading-6 text-ink-500">{{ c.text }}</p>
      </div>
    </div>
    <section class="card mt-10 p-6 sm:p-8">
      <h2 class="text-lg font-semibold">大概要花多少？</h2>
      <p class="mt-2 text-sm leading-7 text-ink-600 dark:text-ink-300">一个约 1 分钟、6 个分镜的讲解视频，通常调用模型 8–10 次、合计约 6–10 万 tokens；一个单场景动画约 1–2 万 tokens。实际花费取决于所选模型和视频长度，每个作品右上角会显示累计用量，明细在主站「用量明细」里能看到。</p>
      <div class="mt-5 flex flex-wrap gap-3">
        <a :href="`${MAIN_SITE_URL}/pricing?utm_source=video&utm_medium=pricing`" target="_blank" rel="noopener" class="btn-primary">查看 HiveGPT 价格</a>
        <a :href="`${MAIN_SITE_URL}/purchase?utm_source=video&utm_medium=pricing`" target="_blank" rel="noopener" class="btn-ghost">充值</a>
      </div>
    </section>
    <section class="mt-10">
      <h2 class="text-lg font-semibold">常见问题</h2>
      <div class="mt-4 space-y-3">
        <details v-for="f in faq" :key="f.q" class="card p-5">
          <summary class="cursor-pointer font-medium">{{ f.q }}</summary>
          <p class="mt-3 text-sm leading-7 text-ink-600 dark:text-ink-300">{{ f.a }}</p>
        </details>
      </div>
    </section>
  </div>
</template>

<script setup>
import { Coins, Gift, ShieldCheck } from 'lucide-vue-next'
import { MAIN_SITE_URL } from '../lib/api'

const cards = [
  { icon: Coins, title: '用多少付多少', text: '没有月费门槛，也没有会过期的积分。按模型实际用量从余额扣费。' },
  { icon: Gift, title: '配音和导出免费', text: '12 种中文音色配音、在线预览、导出 HTML、发布案例都不额外收费。' },
  { icon: ShieldCheck, title: '额度在你手里', text: '生成用的是你账号下的 Key，可以在主站随时停用或删除；作品默认只有自己能看。' }
]
const faq = [
  { q: '需要单独注册吗？', a: '不需要。点右上角「登录」，用 HiveGPT 账号授权一个 Key 就能用；还没有账号可以在登录页注册。' },
  { q: '用哪个模型最好？', a: '在创作页的「模型」里选择。不同模型的画面效果、速度和价格不同，写动画代码对模型要求高，建议先用默认推荐的模型。' },
  { q: '生成要多久？可以关掉页面吗？', a: '一个 1 分钟的视频通常 2–5 分钟。生成在服务器上进行，可以关掉页面，回来在「历史创作」里打开。' },
  { q: '能导出 MP4 吗？', a: '目前可以导出单个 HTML 文件（带配音和字幕，浏览器直接播放）。MP4 视频导出即将上线。' },
  { q: '作品版权归谁？', a: '你生成的作品归你所有。发布到案例库后，其他人可以观看并「制作同款」（基于你的描述重新生成），不会拿到你的作品文件以外的信息。' }
]
</script>
