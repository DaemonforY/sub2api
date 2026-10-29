<script setup lang="ts">
// The AI critique of a practice attempt (学画画): the total against the pass mark, a score and a note
// per rubric item, what went well, the one change to make, and (drawings) a rewritten prompt.
import type { Critique } from '../api'

defineProps<{ critique: Critique; sample?: boolean }>()
const emit = defineEmits<{ useRevised: [prompt: string] }>()

function tone(score: number, pass: number): string {
  return score >= Math.max(pass, 85) ? 'good' : score >= pass ? 'ok' : 'low'
}
</script>

<template>
  <div class="crit" data-testid="critique">
    <div class="crit-top">
      <div :class="['crit-score', tone(critique.score, critique.pass)]">
        <b>{{ critique.score }}</b><small>/ 100</small>
      </div>
      <div class="crit-verdict">
        <span v-if="sample" class="crit-sample">示例点评</span>
        <b>{{ critique.passed ? '达标' : '还没达标' }}</b>
        <span class="runbox-note">及格线 {{ critique.pass }} 分 · AI 老师按下面几项打分</span>
      </div>
    </div>
    <ul class="crit-items">
      <li v-for="it in critique.items" :key="it.dim">
        <span class="crit-dim">{{ it.dim }}</span>
        <span class="crit-bar"><i :class="tone(it.score, critique.pass)" :style="{ width: `${it.score}%` }"></i></span>
        <span class="crit-num">{{ it.score }}</span>
        <span class="crit-note">{{ it.note }}</span>
      </li>
    </ul>
    <p v-if="critique.praise" class="crit-praise">👍 {{ critique.praise }}</p>
    <p v-if="critique.suggestion" class="crit-suggest">✏️ <b>下一步：</b>{{ critique.suggestion }}</p>
    <div v-if="critique.revised_prompt" class="crit-revised">
      <div class="crit-k">按建议改写的提示词</div>
      <p>{{ critique.revised_prompt }}</p>
      <button v-if="!sample" class="runbox-btn small" @click="emit('useRevised', critique.revised_prompt!)">填进输入框，改改再画</button>
    </div>
  </div>
</template>
