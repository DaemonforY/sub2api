<script setup lang="ts">
// 眼力练习 (学画画): look at a picture and answer — which light, which composition, which shot.
// Practice only (answers live in the page, nothing is recorded); the lesson quiz is what counts.
import { computed, ref } from 'vue'
import { withBase } from 'vitepress'

interface Question {
  img?: string
  q: string
  options: string[]
  answer: number
  explain: string
}

const props = withDefaults(defineProps<{ questions: string; title?: string }>(), { title: '眼力练习' })
const list = computed<Question[]>(() => {
  try {
    return JSON.parse(props.questions)
  } catch {
    return []
  }
})
const idx = ref(0)
const picked = ref<number | null>(null)
const right = ref(0)
const finished = ref(false)
const cur = computed(() => list.value[idx.value])

function pick(i: number) {
  if (picked.value !== null) return
  picked.value = i
  if (i === cur.value.answer) right.value++
}

function next() {
  if (idx.value + 1 >= list.value.length) {
    finished.value = true
    return
  }
  idx.value++
  picked.value = null
}

function again() {
  idx.value = 0
  picked.value = null
  right.value = 0
  finished.value = false
}
</script>

<template>
  <div class="eyeq" data-testid="eye-quiz">
    <div class="eyeq-head">
      <span class="eyeq-title">👁 {{ title }}</span>
      <span class="runbox-note">{{ finished ? `答对 ${right} / ${list.length}` : `第 ${idx + 1} / ${list.length} 题` }}</span>
    </div>
    <template v-if="!finished && cur">
      <img v-if="cur.img" class="eyeq-img" :src="withBase(cur.img)" alt="看图回答" loading="lazy" />
      <p class="eyeq-q">{{ cur.q }}</p>
      <div class="eyeq-opts">
        <button
          v-for="(o, i) in cur.options"
          :key="i"
          :class="{ right: picked !== null && i === cur.answer, wrong: picked === i && i !== cur.answer }"
          :disabled="picked !== null"
          @click="pick(i)"
        >
          {{ o }}
        </button>
      </div>
      <div v-if="picked !== null" class="eyeq-explain">
        <b>{{ picked === cur.answer ? '答对了。' : `答案是「${cur.options[cur.answer]}」。` }}</b>{{ cur.explain }}
        <button class="runbox-btn small" @click="next">{{ idx + 1 >= list.length ? '看结果' : '下一题' }}</button>
      </div>
    </template>
    <div v-else class="eyeq-done">
      <p>{{ right === list.length ? '全对！眼睛已经开始变尖了。' : right * 2 >= list.length ? '不错，错的那几题回去对照正文再看一眼。' : '多看几遍上面的对比图，再来一次。' }}</p>
      <button class="runbox-btn ghost" @click="again">再练一次</button>
    </div>
  </div>
</template>
