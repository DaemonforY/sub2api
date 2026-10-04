<script setup lang="ts">
// The AI tutor on lesson pages: a floating button opening a chat that knows this lesson.
import { computed, nextTick, ref, watch } from 'vue'
import { useData } from 'vitepress'
import { ApiError, QUOTA_REASONS, askTutor, chooseOwnKey, learnConfig, loginUrl, ownKey, progress, token, type RunMessage } from '../api'
import OwnKeyPicker from './OwnKeyPicker.vue'

const { frontmatter } = useData()
const lesson = computed(() => (frontmatter.value.lesson ? String(frontmatter.value.lesson) : ''))
const open = ref(false)
const enabled = ref<boolean | null>(null)
const signedIn = ref(false)
const messages = ref<RunMessage[]>([])
const input = ref('')
const busy = ref(false)
const error = ref('')
const quotaOut = ref(false)
const box = ref<HTMLElement | null>(null)
const MAX_QUESTIONS = 5
const asked = computed(() => messages.value.filter((m) => m.role === 'user').length)

watch(lesson, () => {
  messages.value = []
  error.value = ''
  quotaOut.value = false
})

async function toggle() {
  open.value = !open.value
  if (open.value && enabled.value === null) {
    signedIn.value = !!token()
    enabled.value = (await learnConfig()).run_enabled
  }
}

function scroll() {
  void nextTick(() => box.value?.scrollTo({ top: box.value.scrollHeight }))
}

async function send(retry = false) {
  const text = retry ? '' : input.value.trim()
  if ((!text && !retry) || busy.value || !lesson.value) return
  if (!retry) {
    messages.value = [...messages.value, { role: 'user', content: text }]
    input.value = ''
  }
  busy.value = true
  error.value = ''
  quotaOut.value = false
  const history = messages.value.slice()
  const reply: RunMessage = { role: 'assistant', content: '' }
  const at = messages.value.length
  messages.value = [...messages.value, reply]
  scroll()
  try {
    const keyId = progress.tutorLeft === 0 ? ownKey.id : 0
    await askTutor(lesson.value, history, keyId, (d) => {
      reply.content += d
      messages.value = [...messages.value]
      scroll()
    })
  } catch (e) {
    const err = e as ApiError
    if (!reply.content) {
      messages.value = messages.value.slice(0, at)
      if (QUOTA_REASONS.includes(err.reason)) {
        progress.tutorLeft = 0
        if (ownKey.id) {
          busy.value = false
          return send(true)
        }
        quotaOut.value = true
      } else {
        if (err.reason === 'LEARN_KEY_INVALID') chooseOwnKey(null)
        error.value = err.status === 401 ? '登录已过期，请重新登录' : err.message
      }
      // Leave the question in place to send again.
      const last = messages.value[messages.value.length - 1]
      if (last?.role === 'user' && !quotaOut.value) {
        messages.value = messages.value.slice(0, -1)
        input.value = last.content
      }
    } else {
      error.value = err.message
    }
  } finally {
    busy.value = false
  }
}

function restart() {
  messages.value = []
  error.value = ''
  quotaOut.value = false
}
</script>

<template>
  <template v-if="lesson">
    <button class="tutor-fab" data-testid="tutor-open" @click="toggle">{{ open ? '×' : '💬 问助教' }}</button>
    <div v-if="open" class="tutor" data-testid="tutor">
      <div class="tutor-head">
        <strong>AI 助教</strong>
        <span class="runbox-note">只回答和这节课相关的问题</span>
        <span v-if="signedIn && enabled && progress.tutorLeft !== null" class="runbox-quota">
          {{ progress.tutorLeft === 0 && ownKey.id ? `用你的 Key「${ownKey.name}」（正常计费）` : `今日免费 ${progress.tutorLeft} 问` }}
        </span>
      </div>
      <div ref="box" class="tutor-body">
        <p v-if="!messages.length" class="runbox-note">
          看不懂哪一段、代码报错、想知道怎么用到自己的项目里，都可以问。助教会结合这节课的内容回答。
        </p>
        <div v-for="(m, i) in messages" :key="i" :class="['runbox-msg', m.role]">{{ m.content || (busy ? '思考中…' : '') }}</div>
        <OwnKeyPicker v-if="quotaOut" what="提问" @chosen="send(true)" />
        <div v-if="error" class="runbox-error">{{ error }}</div>
      </div>
      <div class="tutor-foot">
        <template v-if="enabled === false"><span class="runbox-note">助教暂未开放。</span></template>
        <template v-else-if="!signedIn"><a class="runbox-btn" :href="loginUrl()">登录后提问</a></template>
        <template v-else-if="asked >= MAX_QUESTIONS">
          <span class="runbox-note">这轮对话问得比较多了，开一个新对话继续吧。</span>
          <button class="runbox-btn ghost" @click="restart">新对话</button>
        </template>
        <template v-else>
          <textarea v-model="input" rows="2" maxlength="1000" placeholder="输入你的问题，Ctrl / ⌘ + Enter 发送" @keydown.ctrl.enter="send()" @keydown.meta.enter="send()"></textarea>
          <div class="runbox-actions">
            <button class="runbox-btn" :disabled="busy || !input.trim()" data-testid="tutor-send" @click="send()">{{ busy ? '回答中…' : '发送' }}</button>
            <button v-if="messages.length" class="runbox-btn ghost" :disabled="busy" @click="restart">新对话</button>
          </div>
        </template>
      </div>
    </div>
  </template>
</template>
