<script setup lang="ts">
// 「动手试试」: runs a lesson's example on HiveGPT for the signed-in learner (free runs a day).
// The request can only carry messages plus an optional JSON schema / tools; the server picks
// the model. Signed out or when runs are off, a recorded sample output is shown instead.
import { computed, onMounted, ref } from 'vue'
import { ApiError, QUOTA_REASONS, chooseOwnKey, learnConfig, loginUrl, ownKey, progress, runExample, runMode, token, type RunMessage, type RunResult } from '../api'
import OwnKeyPicker from './OwnKeyPicker.vue'

const props = withDefaults(
  defineProps<{
    lesson: string
    prompt: string
    system?: string
    /** JSON of a json_schema response_format body: { name, schema, strict }. */
    schema?: string
    /** Ask for any JSON object. */
    json?: boolean
    /** JSON array of function tools. */
    tools?: string
    /** Multi-turn: keeps the conversation (up to 6 questions). */
    chat?: boolean
    /** Recorded output shown before running (signed out, runs off). */
    sample?: string
    rows?: number
    title?: string
  }>(),
  { rows: 3, title: '动手试试' },
)

const input = ref(props.prompt)
const running = ref(false)
const error = ref('')
const needLogin = ref(false)
const quotaOut = ref(false)
const result = ref<RunResult | null>(null)
const history = ref<RunMessage[]>([])
const enabled = ref<boolean | null>(null)
const signedIn = ref(false)
const MAX_TURNS = 6

onMounted(async () => {
  signedIn.value = !!token()
  enabled.value = (await learnConfig()).run_enabled
})

const structured = computed(() => !!props.schema || props.json)
const turns = computed(() => history.value.filter((m) => m.role === 'user').length)

function pretty(text: string): string {
  if (!structured.value) return text
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}

const toolCalls = computed(() => {
  const calls = result.value?.tool_calls as Array<{ function?: { name?: string; arguments?: string } }> | undefined
  if (!calls?.length) return ''
  return JSON.stringify(
    calls.map((c) => {
      let args: unknown = c.function?.arguments
      try {
        args = JSON.parse(String(c.function?.arguments || '{}'))
      } catch {
        // keep the raw string
      }
      return { name: c.function?.name, arguments: args }
    }),
    null,
    2,
  )
})

async function run() {
  const text = input.value.trim()
  if (!text || running.value) return
  running.value = true
  error.value = ''
  needLogin.value = false
  quotaOut.value = false
  const messages: RunMessage[] = []
  if (props.system) messages.push({ role: 'system', content: props.system })
  if (props.chat) messages.push(...history.value)
  messages.push({ role: 'user', content: text })
  try {
    const body: { lesson: string; messages: RunMessage[]; response_format?: unknown; tools?: unknown } = { lesson: props.lesson, messages }
    if (props.schema) body.response_format = { type: 'json_schema', json_schema: JSON.parse(props.schema) }
    else if (props.json) body.response_format = { type: 'json_object' }
    if (props.tools) body.tools = JSON.parse(props.tools)
    const res = await runExample(body)
    result.value = res
    if (props.chat) {
      history.value = [...history.value, { role: 'user', content: text }, { role: 'assistant', content: res.content }]
      input.value = ''
    }
  } catch (e) {
    const err = e as ApiError
    if (QUOTA_REASONS.includes(err.reason)) {
      progress.runsLeft = 0
      if (ownKey.id) {
        running.value = false
        return run()
      }
      quotaOut.value = true
    } else if (err.reason === 'LEARN_KEY_INVALID') {
      chooseOwnKey(null)
      error.value = err.message
    } else if (err.status === 401) {
      needLogin.value = true
      error.value = '登录已过期，请重新登录后再运行'
    } else {
      error.value = err.message || '运行失败，请稍后再试'
    }
  } finally {
    running.value = false
  }
}

function restart() {
  history.value = []
  result.value = null
  input.value = props.prompt
}
</script>

<template>
  <div class="runbox" data-testid="runbox">
    <div class="runbox-head">
      <span class="runbox-title">▶ {{ title }}</span>
      <span v-if="signedIn && enabled && (runMode.ownKeyOnly || progress.runsLeft === 0) && ownKey.id" class="runbox-quota">
        用你的 Key「{{ ownKey.name }}」运行（正常计费）<a href="javascript:void 0" @click="chooseOwnKey(null)">{{ runMode.ownKeyOnly ? '换一个' : '不用了' }}</a>
      </span>
      <span v-else-if="signedIn && enabled && runMode.ownKeyOnly" class="runbox-quota">用你自己的 Key 运行，按用量计费</span>
      <span v-else-if="signedIn && enabled && progress.runsLeft !== null" class="runbox-quota">今日免费运行剩余 {{ progress.runsLeft }} 次</span>
    </div>

    <details v-if="system" class="runbox-system">
      <summary>系统提示词（这次请求一起发送，点开查看）</summary>
      <div>{{ system }}</div>
    </details>

    <div v-if="chat && history.length" class="runbox-chat">
      <div v-for="(m, i) in history" :key="i" :class="['runbox-msg', m.role]">{{ m.content }}</div>
    </div>

    <textarea
      v-model="input"
      class="runbox-input"
      :rows="rows"
      :placeholder="chat && history.length ? '接着问一句…' : '在这里改提示词，再点运行'"
      :disabled="chat && turns >= MAX_TURNS"
      @keydown.ctrl.enter="run"
      @keydown.meta.enter="run"
    ></textarea>

    <div class="runbox-actions">
      <template v-if="enabled === false">
        <span class="runbox-note">在线运行暂未开放，可以复制上面的代码用自己的 Key 运行。</span>
      </template>
      <template v-else-if="!signedIn">
        <a class="runbox-btn" :href="loginUrl()">登录后运行</a>
        <span class="runbox-note">{{ runMode.ownKeyOnly ? '登录 HiveGPT 后用自己的 Key 运行' : '登录 HiveGPT 后每天有免费运行次数' }}</span>
      </template>
      <template v-else>
        <button class="runbox-btn" :disabled="running || !input.trim() || (chat && turns >= MAX_TURNS)" data-testid="runbox-run" @click="run">
          {{ running ? '运行中…' : chat && history.length ? '发送' : '运行' }}
        </button>
        <button v-if="chat && history.length" class="runbox-btn ghost" @click="restart">重新开始</button>
        <span class="runbox-note">Ctrl / ⌘ + Enter 也可以运行</span>
      </template>
    </div>

    <OwnKeyPicker v-if="quotaOut" @chosen="run" />

    <div v-if="error" class="runbox-error">
      {{ error }}
      <a v-if="needLogin" :href="loginUrl()">去登录</a>
    </div>

    <div v-if="result && !chat" class="runbox-out" data-testid="runbox-output">
      <div class="runbox-meta">
        <span>✓ 运行成功 · {{ (result.latency_ms / 1000).toFixed(1) }} 秒</span>
        <span>{{ result.prompt_tokens + result.completion_tokens }} tokens · {{ result.model }}</span>
      </div>
      <pre v-if="toolCalls"><code>{{ toolCalls }}</code></pre>
      <pre v-if="result.content"><code>{{ pretty(result.content) }}</code></pre>
      <p v-if="toolCalls && !result.content" class="runbox-note">模型没有直接回答，而是要求调用上面的函数（finish_reason: {{ result.finish_reason }}）。</p>
    </div>
    <div v-else-if="sample && !result" class="runbox-out sample">
      <div class="runbox-meta"><span>示例输出（之前运行的结果）</span></div>
      <pre><code>{{ sample }}</code></pre>
    </div>
  </div>
</template>
