<template>
  <button type="button" :class="btnClass" :disabled="busy" data-testid="game-result-card" @click="make">{{ busy ? '生成中…' : '🖼 生成战绩图' }}</button>
  <Teleport to="body">
    <div v-if="url" class="grc-mask" data-testid="game-result-card-preview" @click.self="url = ''">
      <div class="grc-box">
        <img :src="url" alt="战绩图" />
        <p>{{ isTouch ? '长按图片保存，发给朋友或朋友圈' : '图片已下载，也可以右键另存' }}</p>
        <button type="button" class="grc-close" @click="url = ''">关闭</button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
// 「生成战绩图」 for the co-op games: draws the shared 1080×1440 result card, previews it (long-press
// to save on phones) and downloads it on desktop.
import { ref } from 'vue'
import { drawGameResultCard, type ResultCardInput } from '@/games/resultCard'
import { track } from '@/utils/analytics'

const props = defineProps<{ build: () => ResultCardInput; btnClass?: string }>()
const busy = ref(false)
const url = ref('')
const isTouch = typeof window !== 'undefined' && !!window.matchMedia && window.matchMedia('(pointer: coarse)').matches

async function make(): Promise<void> {
  const input = props.build()
  busy.value = true
  try {
    url.value = await drawGameResultCard(input)
    if (!isTouch) {
      const a = document.createElement('a')
      a.href = url.value
      a.download = `${input.title}-${input.score}.png`
      a.click()
    }
    track('game_cta', { game: input.game, type: 'share', cta: 'result_card' })
  } catch (err) {
    console.error(err)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.grc-mask {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(5, 3, 12, 0.82);
}
.grc-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  max-height: 100%;
  color: #e8dcff;
  font-size: 14px;
  text-shadow: none;
}
.grc-box img {
  max-width: min(420px, 100%);
  max-height: calc(100vh - 120px);
  border-radius: 12px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
}
.grc-close {
  padding: 6px 18px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
}
</style>
