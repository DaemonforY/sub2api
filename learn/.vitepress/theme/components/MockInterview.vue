<script setup lang="ts">
// A mock interview on one topic: five questions from the bank, one at a time; the AI interviewer
// scores each answer (0–10) with comments and what a strong answer covers.
import { computed, onMounted, ref } from 'vue'
import {
  ApiError,
  QUOTA_REASONS,
  answerInterview,
  chooseOwnKey,
  getInterview,
  learnConfig,
  listInterviews,
  loginUrl,
  ownKey,
  progress,
  startInterview,
  token,
  type Interview,
} from '../api'
import OwnKeyPicker from './OwnKeyPicker.vue'

const props = defineProps<{ topic: string }>()
const signedIn = ref(false)
const enabled = ref<boolean | null>(null)
const best = ref(0)
const iv = ref<Interview | null>(null)
const answer = ref('')
const busy = ref(false)
const error = ref('')
const quotaOut = ref(false)
const PASS = 60

const answers = computed(() => iv.value?.answers || [])
const current = computed(() => (iv.value && iv.value.status === 'active' ? iv.value.questions[answers.value.length] : null))

onMounted(async () => {
  signedIn.value = !!token()
  enabled.value = (await learnConfig()).run_enabled
  if (!signedIn.value) return
  try {
    const data = await listInterviews()
    best.value = data.topics.find((t) => t.id === props.topic)?.best || 0
    const active = data.recent.find((r) => r.topic === props.topic && r.status === 'active')
    if (active) iv.value = await getInterview(active.id)
  } catch {
    // shown as not started
  }
})

async function start() {
  busy.value = true
  error.value = ''
  quotaOut.value = false
  try {
    const keyId = progress.interviewsLeft === 0 ? ownKey.id : 0
    iv.value = await startInterview(props.topic, keyId)
    answer.value = ''
  } catch (e) {
    const err = e as ApiError
    if (QUOTA_REASONS.includes(err.reason)) {
      progress.interviewsLeft = 0
      if (ownKey.id) {
        busy.value = false
        return start()
      }
      quotaOut.value = true
    } else {
      if (err.reason === 'LEARN_KEY_INVALID') chooseOwnKey(null)
      error.value = err.message
    }
  } finally {
    busy.value = false
  }
}

async function submit() {
  if (!iv.value || !answer.value.trim() || busy.value) return
  busy.value = true
  error.value = ''
  try {
    iv.value = await answerInterview(iv.value.id, answer.value)
    answer.value = ''
    if (iv.value.status === 'finished') best.value = Math.max(best.value, iv.value.score)
  } catch (e) {
    error.value = (e as ApiError).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="interview" data-testid="mock-interview">
    <div class="interview-head">
      <strong>🎤 模拟面试</strong>
      <span class="runbox-note">每场 5 题，AI 面试官逐题打分点评，总分 = 平均分 × 10</span>
      <span v-if="best" :class="['quiz-best', { ok: best >= PASS }]">最好成绩 {{ best }} 分{{ best >= PASS ? ' · 已通过' : '' }}</span>
    </div>

    <template v-if="!signedIn">
      <a class="runbox-btn" :href="loginUrl()">登录后开始模拟面试</a>
    </template>
    <p v-else-if="enabled === false" class="runbox-note">模拟面试暂未开放。</p>
    <template v-else>
      <div v-for="(a, i) in answers" :key="i" class="interview-turn">
        <p class="interview-q">第 {{ i + 1 }} 题：{{ iv!.questions[i].q }}</p>
        <div class="runbox-msg user">{{ a.answer }}</div>
        <div class="interview-grade">
          <span :class="['interview-score', a.score >= 6 ? 'ok' : 'bad']">{{ a.score }} / 10</span>
          <p>{{ a.comment }}</p>
          <details><summary>好回答应该覆盖什么</summary><p class="interview-better">{{ a.better }}</p></details>
        </div>
      </div>

      <div v-if="current" class="interview-turn">
        <p class="interview-q">第 {{ answers.length + 1 }} / 5 题：{{ current.q }}</p>
        <textarea v-model="answer" class="runbox-input" rows="5" maxlength="2000" placeholder="像真实面试一样作答：先给结论，再讲原理和你的实践。不会可以写「不会」，看看参考要点。"></textarea>
        <div class="runbox-actions">
          <button class="runbox-btn" :disabled="busy || !answer.trim()" data-testid="interview-answer" @click="submit">{{ busy ? '面试官评分中…' : '提交回答' }}</button>
          <span v-if="iv?.own_key" class="runbox-note">这场用你的 Key 评分（正常计费）</span>
        </div>
      </div>

      <div v-if="iv && iv.status === 'finished'" :class="['interview-result', iv.score >= PASS ? 'ok' : 'bad']" data-testid="interview-result">
        本场得分 <strong>{{ iv.score }}</strong> 分 · {{ iv.score >= PASS ? '通过！' : `还差 ${PASS - iv.score} 分通过，看看每题的参考要点再来一场` }}
      </div>

      <div v-if="!current" class="runbox-actions">
        <button class="runbox-btn" :disabled="busy" data-testid="interview-start" @click="start">
          {{ busy ? '准备题目…' : iv ? '再来一场' : '开始模拟面试' }}
        </button>
        <span v-if="progress.interviewsLeft !== null" class="runbox-note">
          {{ progress.interviewsLeft === 0 && ownKey.id ? `免费场次已用完，将用你的 Key「${ownKey.name}」（正常计费）` : `今天还能免费面试 ${progress.interviewsLeft} 场` }}
        </span>
      </div>
      <OwnKeyPicker v-if="quotaOut" what="模拟面试" @chosen="start" />
      <div v-if="error" class="runbox-error">{{ error }}</div>
    </template>
  </div>
</template>
