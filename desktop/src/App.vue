<script setup lang="ts">
import { ref } from 'vue'
import SetupView from './components/SetupView.vue'
import ChatView from './components/ChatView.vue'

// Setup (engine, sign-in, folder) hands over the chosen working folder; the chat runs in it.
const workspace = ref('')
</script>

<template>
  <div class="app">
    <SetupView v-if="!workspace" @ready="(dir: string) => (workspace = dir)" />
    <ChatView v-else :workspace="workspace" @change-folder="workspace = ''" />
  </div>
</template>

<style scoped>
.app {
  height: 100vh;
}
</style>
