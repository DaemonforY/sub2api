<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { invoke } from '@tauri-apps/api/core'
import SetupView from './components/SetupView.vue'
import ChatView from './components/ChatView.vue'

// Setup (engine, sign-in, folder) hands over the chosen working folder; the chat runs in it.
const workspace = ref('')

// 埋点: desktop_installed (first launch) and desktop_open (every launch).
onMounted(() => invoke('track_launch').catch(() => {}))
</script>

<template>
  <div class="app">
    <SetupView v-if="!workspace" @ready="(dir: string) => (workspace = dir)" />
    <ChatView v-else :workspace="workspace" @change-folder="workspace = ''" @relogin="workspace = ''" />
  </div>
</template>

<style scoped>
.app {
  height: 100vh;
}
</style>
