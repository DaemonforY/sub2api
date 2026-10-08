<template>
  <div class="flex min-h-screen flex-col">
    <AppHeader v-if="!route.meta.bare" />
    <main class="flex-1" :class="route.meta.full ? 'min-h-0' : ''">
      <RouterView />
    </main>
    <AppFooter v-if="!route.meta.full && !route.meta.bare" />
    <Toasts />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from './components/AppHeader.vue'
import AppFooter from './components/AppFooter.vue'
import Toasts from './components/Toasts.vue'
import { loadMe } from './lib/api'
import { initTheme } from './lib/theme'

const route = useRoute()
onMounted(() => {
  initTheme()
  loadMe()
})
</script>
