<template>
  <div class="glb" data-testid="game-leaderboard">
    <div class="glb-head">
      <b>全站排行榜</b>
      <span v-if="board">{{ board.players }} 人上榜</span>
    </div>

    <p v-if="mine && mine.improved" class="glb-note good">新纪录！全站第 {{ mine.rank }} 名</p>
    <p v-else-if="mine" class="glb-note">你的最好成绩 {{ mine.best.toLocaleString() }} 分，全站第 {{ mine.rank }} 名</p>

    <ol v-if="board && board.top.length" class="glb-list">
      <li v-for="e in board.top.slice(0, limit)" :key="e.rank + e.display_name" :class="{ me: e.is_current_user }">
        <span class="n">{{ e.rank }}</span>
        <span class="name">{{ e.display_name }}{{ e.is_current_user ? '（你）' : '' }}</span>
        <span v-if="describe && e.detail" class="d">{{ describe(e.detail) }}</span>
        <b>{{ e.score.toLocaleString() }}</b>
      </li>
      <li v-if="board.me && board.me.rank > limit" class="me">
        <span class="n">{{ board.me.rank }}</span>
        <span class="name">{{ board.me.display_name }}（你）</span>
        <span v-if="describe && board.me.detail" class="d">{{ describe(board.me.detail) }}</span>
        <b>{{ board.me.score.toLocaleString() }}</b>
      </li>
    </ol>
    <p v-else-if="board" class="glb-empty">还没有人上榜，第一名等你来拿</p>
    <p v-else class="glb-empty">排行榜加载中…</p>

    <div v-if="!account.loggedIn.value && score > 0" class="glb-login">
      <span>这局 {{ score.toLocaleString() }} 分{{ wouldRank ? `，能排第 ${wouldRank} 名` : '' }}，登录就能上榜</span>
      <a :href="account.loginHref()" class="glb-btn" data-testid="game-board-login" @click="account.trackCta('result_login')">微信登录上榜</a>
    </div>

    <a :href="siteHref" class="glb-site" data-testid="game-site-cta" @click="account.trackCta(account.loggedIn.value ? 'draw' : 'register')">
      🎨 这个游戏的画面是 AI 画的。{{ siteLine }} →
    </a>
  </div>
</template>

<script setup lang="ts">
// Result-screen leaderboard for 小游戏可选登录. On mount it submits this run (or parks it for a visitor,
// who gets it submitted after signing in), then shows the top of the site-wide board.
import { computed, onMounted, ref } from 'vue'
import { useGameAccount } from '@/composables/useGameAccount'
import { getGameLeaderboard, type GameLeaderboard, type GameScoreResult } from '@/api/gameProgress'
import { canvasUrl } from '@/constants/crossSites'

const props = withDefaults(
  defineProps<{
    game: string
    score: number
    detail?: Record<string, unknown>
    describe?: (d: Record<string, unknown>) => string
    limit?: number
  }>(),
  { detail: () => ({}), describe: undefined, limit: 5 }
)

const account = useGameAccount(props.game)
const board = ref<GameLeaderboard | null>(null)
const mine = ref<GameScoreResult | null>(null)

// Where this run would land for a visitor who hasn't signed in yet (0 = outside the top 20).
const wouldRank = computed(() => {
  const b = board.value
  if (!b || props.score <= 0) return 0
  const idx = b.top.findIndex((e) => props.score > e.score)
  if (idx >= 0) return idx + 1
  return b.top.length < 20 ? b.top.length + 1 : 0
})

const siteLine = computed(() => {
  if (account.loggedIn.value) return '你也来画一张自己的游戏角色'
  const bonus = account.signupBonus.value
  return bonus > 0 ? `注册送 $${Math.round(bonus * 100) / 100}，画一张你自己的游戏角色` : '注册后也能画一张你自己的游戏角色'
})

const siteHref = computed(() =>
  account.loggedIn.value
    ? canvasUrl({ medium: `game-${props.game}`, baseUrl: location.origin, path: '/image' })
    : `/register?utm_source=game&utm_medium=${props.game}`
)

onMounted(async () => {
  mine.value = await account.submit(props.score, props.detail)
  try {
    board.value = await getGameLeaderboard(props.game)
  } catch {
    board.value = { game: props.game, board: 'main', players: 0, top: [] }
  }
})
</script>

<style scoped>
.glb {
  margin-top: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(10, 6, 24, 0.5);
  color: #e8dcff;
  font-size: 13px;
  text-align: left;
  text-shadow: none;
}
.glb-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 6px;
}
.glb-head b {
  font-size: 14px;
  color: #fff;
}
.glb-head span,
.glb-empty {
  color: #a89cc8;
}
.glb-note {
  margin: 0 0 6px;
  color: #ffd98a;
}
.glb-note.good {
  color: #86efac;
  font-weight: 700;
}
.glb-list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.glb-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 0;
}
.glb-list li.me {
  color: #86efac;
  font-weight: 700;
}
.glb-list .n {
  width: 22px;
  color: #ffd98a;
  font-weight: 800;
  text-align: right;
}
.glb-list .name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.glb-list .d {
  color: #a89cc8;
  font-size: 12px;
  white-space: nowrap;
}
.glb-list b {
  min-width: 64px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.glb-login {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed rgba(232, 220, 255, 0.25);
}
.glb-btn {
  padding: 6px 14px;
  border-radius: 999px;
  background: #22c55e;
  color: #052e12;
  font-weight: 800;
  text-decoration: none;
  white-space: nowrap;
}
.glb-btn:hover {
  background: #4ade80;
}
.glb-site {
  display: block;
  margin-top: 10px;
  color: #c4b5fd;
  font-size: 12px;
  text-decoration: none;
}
.glb-site:hover {
  color: #fff;
  text-decoration: underline;
}
</style>
