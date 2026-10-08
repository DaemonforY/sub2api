<template>
  <header class="sticky top-0 z-40 border-b border-ink-200/70 bg-white/85 backdrop-blur dark:border-ink-800 dark:bg-ink-950/85">
    <div class="mx-auto flex h-16 max-w-[1440px] items-center gap-6 px-4 sm:px-6">
      <Logo />
      <nav class="hidden flex-1 items-center justify-center gap-1 md:flex">
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="rounded-lg px-3 py-2 text-[15px] text-ink-600 transition hover:text-brand-600 dark:text-ink-300" :class="{ '!text-brand-600 font-medium': isActive(item) }">
          {{ item.label }}
        </RouterLink>
      </nav>
      <div class="ml-auto flex items-center gap-2 md:ml-0">
        <button class="rounded-lg p-2 text-ink-500 hover:bg-ink-100 dark:hover:bg-ink-800" :aria-label="dark ? '浅色模式' : '深色模式'" @click="toggleTheme">
          <Sun v-if="dark" class="h-5 w-5" />
          <Moon v-else class="h-5 w-5" />
        </button>
        <template v-if="session.me">
          <div class="relative" @mouseleave="menu = false">
            <button class="flex items-center gap-2 rounded-xl px-2 py-1.5 hover:bg-ink-100 dark:hover:bg-ink-800" @click="menu = !menu" @mouseenter="menu = true">
              <span class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-full bg-brand-100 text-sm font-semibold text-brand-700">
                <img v-if="session.me.avatar" :src="session.me.avatar" alt="" class="h-full w-full object-cover" />
                <template v-else>{{ (session.me.name || 'H').slice(0, 1).toUpperCase() }}</template>
              </span>
              <span class="hidden text-left leading-tight sm:block">
                <span class="block max-w-[8rem] truncate text-sm font-medium">{{ session.me.name }}</span>
                <span class="block text-xs text-ink-500">余额 ${{ (session.me.balance || 0).toFixed(2) }}</span>
              </span>
            </button>
            <div v-if="menu" class="absolute right-0 top-full w-56 pt-1">
              <div class="card overflow-hidden py-1 text-sm">
                <p class="truncate px-4 py-2 text-xs text-ink-500">Key：{{ session.me.key_name }}<template v-if="session.me.group"> · {{ session.me.group }}</template></p>
                <a :href="mainSite('/purchase')" target="_blank" rel="noopener" class="block px-4 py-2 hover:bg-ink-50 dark:hover:bg-ink-800">充值 / 订阅</a>
                <a :href="mainSite('/usage')" target="_blank" rel="noopener" class="block px-4 py-2 hover:bg-ink-50 dark:hover:bg-ink-800">用量明细</a>
                <RouterLink v-if="session.me.admin" to="/review" class="block px-4 py-2 hover:bg-ink-50 dark:hover:bg-ink-800" @click="menu = false">案例审核</RouterLink>
                <button class="block w-full px-4 py-2 text-left text-red-600 hover:bg-ink-50 dark:hover:bg-ink-800" @click="signOut(); menu = false">退出登录</button>
              </div>
            </div>
          </div>
        </template>
        <button v-else class="btn-primary" :disabled="signingIn" @click="doSignIn">
          <LogIn class="h-4 w-4" />登录
        </button>
        <button class="rounded-lg p-2 md:hidden" aria-label="菜单" @click="mobile = !mobile"><Menu class="h-5 w-5" /></button>
      </div>
    </div>
    <nav v-if="mobile" class="border-t border-ink-200 px-4 py-2 md:hidden dark:border-ink-800">
      <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="block rounded-lg px-3 py-2" @click="mobile = false">{{ item.label }}</RouterLink>
    </nav>
  </header>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { LogIn, Menu, Moon, Sun } from 'lucide-vue-next'
import Logo from './Logo.vue'
import { MAIN_SITE_URL, session, signIn, signOut } from '../lib/api'
import { dark, toggleTheme } from '../lib/theme'
import { toast, toastError } from '../lib/toast'

const route = useRoute()
const menu = ref(false)
const mobile = ref(false)
const signingIn = ref(false)
const nav = [
  { to: '/', label: '开始创作', match: ['home', 'project'] },
  { to: '/gallery', label: '案例', match: ['gallery', 'work'] },
  { to: '/tools', label: '工具', match: ['tools', 'tool'] },
  { to: '/pricing', label: '价格', match: ['pricing'] }
]
const isActive = (item) => item.match.includes(route.name)
const mainSite = (path) => `${MAIN_SITE_URL}${path}?utm_source=video&utm_medium=header`

async function doSignIn() {
  signingIn.value = true
  try {
    if (await signIn()) toast('登录成功', 'success')
  } catch (err) {
    toastError(err)
  } finally {
    signingIn.value = false
  }
}
</script>
