<script setup lang="ts">
// Right of the nav: the HiveGPT account (or sign in).
import { onMounted, ref } from 'vue'
import { currentUser, loginUrl, type LearnUser } from '../api'

const user = ref<LearnUser | null>(null)
const login = ref('/login')
onMounted(() => {
  user.value = currentUser()
  login.value = loginUrl()
})
</script>

<template>
  <div class="nav-user">
    <template v-if="user">
      <a href="/my-learning" class="nav-user-link">我的学习</a>
      <a href="/dashboard" class="nav-user-link" :title="user.email">{{ user.username || user.email || '我的账户' }}</a>
    </template>
    <a v-else :href="login" class="runbox-btn small">登录</a>
  </div>
</template>
