<template>
  <div class="gab" data-testid="game-account-bar">
    <template v-if="account.loggedIn.value">
      <span v-if="flushed" class="gab-ok">上一局 {{ flushed.best.toLocaleString() }} 分已上榜，全站第 {{ flushed.rank }} 名 🎉</span>
      <span v-else class="gab-ok">✓ 已登录，成绩会进全站排行榜{{ hasSave ? '，进度换设备也在' : '' }}</span>
    </template>
    <template v-else>
      <span>登录后成绩进全站排行榜{{ hasSave ? '，进度换设备也在' : '' }}</span>
      <a :href="account.loginHref()" class="gab-btn" data-testid="game-login" @click="account.trackCta('title_login')">微信登录</a>
    </template>
  </div>
</template>

<script setup lang="ts">
// Title-screen line for 小游戏可选登录: says what signing in gets you. After a sign-in that started from
// a result screen it submits the waiting score and says where it landed.
import { onMounted, ref } from 'vue'
import { useGameAccount } from '@/composables/useGameAccount'
import type { GameScoreResult } from '@/api/gameProgress'

const props = defineProps<{ game: string; hasSave?: boolean }>()
const account = useGameAccount(props.game)
const flushed = ref<GameScoreResult | null>(null)

onMounted(async () => {
  if (!account.loggedIn.value) return
  flushed.value = await account.flushPending()
})
</script>

<style scoped>
.gab {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 8px 12px;
  margin: 10px auto 0;
  max-width: 520px;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(10, 6, 24, 0.55);
  color: #e8dcff;
  font-size: 13px;
  line-height: 1.4;
  text-shadow: none;
}
.gab-ok {
  color: #b8f5c8;
}
.gab-btn {
  padding: 4px 12px;
  border-radius: 999px;
  background: #22c55e;
  color: #052e12;
  font-weight: 700;
  text-decoration: none;
  white-space: nowrap;
}
.gab-btn:hover {
  background: #4ade80;
}
</style>
