<script setup lang="ts">
// A lesson's quiz: questions come from the server without answers; answers are graded there.
// Passing (80% or more) also marks the lesson done.
import { computed, onMounted, ref } from 'vue'
import { ApiError, getQuiz, loginUrl, progress, submitQuiz, token, type QuizGrade, type QuizView } from '../api'

const props = defineProps<{ lesson: string }>()
const quiz = ref<QuizView | null>(null)
const picks = ref<number[][]>([])
const grade = ref<QuizGrade | null>(null)
const sending = ref(false)
const error = ref('')
const signedIn = ref(false)

onMounted(async () => {
  signedIn.value = !!token()
  try {
    quiz.value = await getQuiz(props.lesson)
    picks.value = quiz.value.questions.map(() => [])
  } catch {
    quiz.value = null // no quiz for this lesson
  }
})

const best = computed(() => progress.quizzes[props.lesson])
const complete = computed(() => picks.value.length > 0 && picks.value.every((p) => p.length > 0))

function toggle(qi: number, oi: number, multi: boolean) {
  if (grade.value) return
  const cur = picks.value[qi]
  picks.value[qi] = multi ? (cur.includes(oi) ? cur.filter((x) => x !== oi) : [...cur, oi].sort()) : [oi]
}

function optionClass(qi: number, oi: number) {
  const picked = picks.value[qi]?.includes(oi)
  if (!grade.value) return { picked }
  const right = grade.value.items[qi].answer.includes(oi)
  return { picked, right, wrong: picked && !right }
}

async function submit() {
  if (!complete.value || sending.value) return
  sending.value = true
  error.value = ''
  try {
    grade.value = await submitQuiz(props.lesson, picks.value)
  } catch (e) {
    error.value = (e as ApiError).message
  } finally {
    sending.value = false
  }
}

function retry() {
  grade.value = null
  picks.value = quiz.value!.questions.map(() => [])
}
</script>

<template>
  <div v-if="quiz" class="quiz" data-testid="quiz">
    <div class="quiz-head">
      <strong>📝 本课小测验</strong>
      <span class="runbox-note">{{ quiz.questions.length }} 题 · 答对 80% 算通过，结业证书看各课的最好成绩</span>
      <span v-if="best" class="quiz-best">最好成绩 {{ best.correct }} / {{ best.total }}</span>
    </div>
    <ol class="quiz-list">
      <li v-for="(q, qi) in quiz.questions" :key="qi">
        <p class="quiz-q">{{ q.q }}<span v-if="q.multi" class="quiz-multi">多选</span></p>
        <button
          v-for="(o, oi) in q.options"
          :key="oi"
          type="button"
          :class="['quiz-opt', optionClass(qi, oi)]"
          :disabled="!!grade"
          @click="toggle(qi, oi, q.multi)"
        >
          <span class="quiz-mark">{{ String.fromCharCode(65 + oi) }}</span>{{ o }}
        </button>
        <p v-if="grade" :class="['quiz-explain', grade.items[qi].correct ? 'ok' : 'bad']">
          {{ grade.items[qi].correct ? '✓ 答对了' : '✗ 正确答案：' + grade.items[qi].answer.map((a) => String.fromCharCode(65 + a)).join('、') }}
          · {{ grade.items[qi].explain }}
        </p>
      </li>
    </ol>
    <div class="runbox-actions">
      <template v-if="!signedIn">
        <a class="runbox-btn" :href="loginUrl()">登录后提交</a>
        <span class="runbox-note">测验成绩记在账号里，结业证书会用到</span>
      </template>
      <template v-else-if="!grade">
        <button class="runbox-btn" :disabled="!complete || sending" data-testid="quiz-submit" @click="submit">{{ sending ? '提交中…' : '提交答案' }}</button>
        <span v-if="!complete" class="runbox-note">每题都选了才能提交</span>
      </template>
      <template v-else>
        <span :class="['quiz-score', grade.passed ? 'ok' : 'bad']" data-testid="quiz-score">
          {{ grade.correct }} / {{ grade.total }} {{ grade.passed ? '· 通过，本课已记为完成' : '· 还差一点，看看解析再试一次' }}
        </span>
        <button class="runbox-btn ghost" @click="retry">再做一次</button>
      </template>
    </div>
    <div v-if="error" class="runbox-error">{{ error }}</div>
  </div>
</template>
