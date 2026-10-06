<script setup lang="ts">
// A6 / H7: runs a small agent loop for real. The model picks tools; the tools (the coffee shop notes,
// a calculator and, with `confirm`, a pretend order) run here in the browser; each step is one run on
// HiveGPT. `confirm` adds human-in-the-loop: an order waits for the learner to approve or reject it.
import { onMounted, reactive, ref } from 'vue'
import { ApiError, QUOTA_REASONS, chooseOwnKey, learnConfig, loginUrl, ownKey, progress, runExample, token, type RunMessage } from '../api'
import OwnKeyPicker from './OwnKeyPicker.vue'

const props = withDefaults(
  defineProps<{
    lesson: string
    prompt: string
    maxSteps?: number
    /** Instructions for the agent (default: use the tools, don't do arithmetic in your head). */
    system?: string
    /** Add place_order, which only runs after the learner approves it. */
    confirm?: boolean
    /** Recorded run shown before running. */
    sample?: string
  }>(),
  { maxSteps: 5 },
)

const DEFAULT_SYSTEM = '你可以使用工具。需要信息就先查，需要计算就用计算器，不要自己心算。'
const DEFAULT_SAMPLE = `第 1 步 · search_docs({"query":"拿铁 大杯"}) → 菜单：拿铁 28 元……大杯加 4 元。
第 2 步 · search_docs({"query":"积分 抵扣"}) → 会员规则：……每 100 分可抵扣 5 元。
第 3 步 · calculator({"expression":"(28+4)*3-200/100*5"}) → 86
第 4 步 · 最终回答：3 杯大杯拿铁共 96 元，200 积分抵 10 元，还需支付 86 元。`

const DOCS: Record<string, string> = {
  菜单: '拿铁 28 元，美式 22 元，燕麦拿铁 32 元；大杯加 4 元。',
  会员规则: '会员每消费 1 元积 1 分，每 100 分可抵扣 5 元。',
  营业时间: '周一到周五 8:00–21:00，周末 9:00–22:00。',
}

const TOOLS = [
  {
    type: 'function',
    function: {
      name: 'search_docs',
      description: '查询小蜂咖啡的菜单、会员规则、营业时间等资料',
      parameters: { type: 'object', properties: { query: { type: 'string', description: '关键词，空格分隔' } }, required: ['query'] },
    },
  },
  {
    type: 'function',
    function: {
      name: 'calculator',
      description: '计算四则运算表达式，例如 28*3-10',
      parameters: { type: 'object', properties: { expression: { type: 'string' } }, required: ['expression'] },
    },
  },
]

const ORDER_TOOL = {
  type: 'function',
  function: {
    name: 'place_order',
    description: '为顾客下单（会真实扣款，所以执行前会请顾客确认）。items 是饮品和杯型，例如「大杯拿铁 x2」',
    parameters: {
      type: 'object',
      properties: { items: { type: 'string', description: '饮品、杯型和数量' }, total: { type: 'number', description: '应付金额（元）' } },
      required: ['items', 'total'],
    },
  },
}
const tools = props.confirm ? [...TOOLS, ORDER_TOOL] : TOOLS

function searchDocs(query: string): string {
  const words = String(query || '').split(/\s+/).filter(Boolean)
  const hits = Object.entries(DOCS)
    .filter(([k, v]) => words.some((w) => (k + v).includes(w)))
    .map(([k, v]) => `${k}：${v}`)
  return hits.join('\n') || '没有找到相关资料'
}

// A small arithmetic parser (no eval): numbers, + - * / and parentheses.
function calculate(expr: string): number {
  const src = String(expr).replace(/\s+/g, '').replace(/×/g, '*').replace(/÷/g, '/')
  if (!/^[\d+\-*/().]+$/.test(src)) throw new Error('只支持数字和 + - * / ( )')
  let i = 0
  const peek = () => src[i]
  function factor(): number {
    if (peek() === '-') {
      i++
      return -factor()
    }
    if (peek() === '(') {
      i++
      const v = sum()
      if (src[i++] !== ')') throw new Error('括号不匹配')
      return v
    }
    const m = /^\d+(\.\d+)?/.exec(src.slice(i))
    if (!m) throw new Error('表达式不完整')
    i += m[0].length
    return Number(m[0])
  }
  function product(): number {
    let v = factor()
    while (peek() === '*' || peek() === '/') {
      const op = src[i++]
      const r = factor()
      v = op === '*' ? v * r : v / r
    }
    return v
  }
  function sum(): number {
    let v = product()
    while (peek() === '+' || peek() === '-') {
      const op = src[i++]
      const r = product()
      v = op === '+' ? v + r : v - r
    }
    return v
  }
  const v = sum()
  if (i !== src.length) throw new Error('表达式有多余内容')
  return Math.round(v * 1e6) / 1e6
}

// Human-in-the-loop: the order waits here until the learner clicks 批准 or 拒绝.
const pending = ref<{ items: string; total: string; resolve: (ok: boolean) => void } | null>(null)

function askApproval(args: Record<string, string>): Promise<boolean> {
  return new Promise((resolve) => {
    pending.value = { items: String(args.items ?? ''), total: String(args.total ?? ''), resolve }
  })
}

function decide(ok: boolean) {
  const p = pending.value
  pending.value = null
  p?.resolve(ok)
}

async function callTool(name: string, args: Record<string, string>): Promise<string> {
  if (name !== 'place_order' || !props.confirm) return runTool(name, args)
  const ok = await askApproval(args)
  if (!ok) return '顾客拒绝了这次下单，没有扣款。请询问顾客想怎么调整。'
  return `下单成功（演示，不会真的扣款）：${args.items}，共 ${args.total} 元，订单号 DEMO-${Math.floor(1000 + Math.random() * 9000)}`
}

function runTool(name: string, args: Record<string, string>): string {
  try {
    if (name === 'search_docs') return searchDocs(args.query)
    if (name === 'calculator') return String(calculate(args.expression))
    return `没有这个工具：${name}`
  } catch (e) {
    return `调用失败：${(e as Error).message}`
  }
}

interface Step {
  calls: { name: string; args: string; result: string }[]
  answer?: string
}

const input = ref(props.prompt)
const steps = ref<Step[]>([])
const running = ref(false)
const error = ref('')
const quotaOut = ref(false)
const enabled = ref<boolean | null>(null)
const signedIn = ref(false)

onMounted(async () => {
  signedIn.value = !!token()
  enabled.value = (await learnConfig()).run_enabled
})

async function run() {
  const q = input.value.trim()
  if (!q || running.value) return
  running.value = true
  error.value = ''
  quotaOut.value = false
  steps.value = []
  const messages: RunMessage[] = [
    { role: 'system', content: props.system || DEFAULT_SYSTEM },
    { role: 'user', content: q },
  ]
  try {
    for (let n = 0; n < props.maxSteps; n++) {
      let res
      try {
        res = await runExample({ lesson: props.lesson, messages, tools })
      } catch (e) {
        const err = e as ApiError
        if (QUOTA_REASONS.includes(err.reason) && ownKey.id) {
          progress.runsLeft = 0
          res = await runExample({ lesson: props.lesson, messages, tools })
        } else throw e
      }
      const calls = (res.tool_calls as { id: string; function: { name: string; arguments: string } }[] | undefined) || []
      if (!calls.length) {
        steps.value = [...steps.value, { calls: [], answer: res.content }]
        return
      }
      messages.push({ role: 'assistant', content: res.content || '', tool_calls: calls })
      const step = reactive<Step>({ calls: [] })
      steps.value = [...steps.value, step]
      for (const c of calls) {
        let args: Record<string, string> = {}
        try {
          args = JSON.parse(c.function.arguments || '{}')
        } catch {
          // keep empty
        }
        const call = reactive({ name: c.function.name, args: c.function.arguments, result: '等待执行…' })
        step.calls.push(call)
        const result = await callTool(c.function.name, args)
        call.result = result
        messages.push({ role: 'tool', tool_call_id: c.id, content: result })
      }
    }
    error.value = `已经走了 ${props.maxSteps} 步还没得到答案，循环被步数上限停止了。`
  } catch (e) {
    const err = e as ApiError
    if (QUOTA_REASONS.includes(err.reason)) {
      progress.runsLeft = 0
      quotaOut.value = true
    } else {
      if (err.reason === 'LEARN_KEY_INVALID') chooseOwnKey(null)
      error.value = err.status === 401 ? '登录已过期，请重新登录后再运行' : err.message
    }
  } finally {
    running.value = false
    pending.value = null
  }
}
</script>

<template>
  <div class="runbox agentloop" data-testid="agent-loop">
    <div class="runbox-head">
      <span class="runbox-title">▶ 运行 Agent 循环</span>
      <span v-if="signedIn && enabled && progress.runsLeft !== null" class="runbox-quota">今日免费运行剩余 {{ progress.runsLeft }} 次（每一步算一次）</span>
    </div>
    <textarea v-model="input" class="runbox-input" rows="2"></textarea>
    <div class="runbox-actions">
      <span v-if="enabled === false" class="runbox-note">在线运行暂未开放，可以复制上面的代码用自己的 Key 运行。</span>
      <a v-else-if="!signedIn" class="runbox-btn" :href="loginUrl()">登录后运行</a>
      <button v-else class="runbox-btn" :disabled="running || !input.trim()" data-testid="agent-run" @click="run">{{ running ? `运行中（第 ${steps.length + 1} 步）…` : '运行' }}</button>
    </div>
    <div v-if="pending" class="agent-confirm" data-testid="agent-confirm">
      <p>Agent 想要下单：<strong>{{ pending.items }}</strong>，共 <strong>{{ pending.total }}</strong> 元。要批准吗？</p>
      <button class="runbox-btn small" data-testid="agent-approve" @click="decide(true)">批准</button>
      <button class="runbox-btn small ghost" data-testid="agent-reject" @click="decide(false)">拒绝</button>
    </div>
    <ol v-if="steps.length" class="agent-steps">
      <li v-for="(s, i) in steps" :key="i">
        <template v-if="s.answer !== undefined">
          <strong>第 {{ i + 1 }} 步 · 最终回答</strong>
          <div class="runbox-msg assistant">{{ s.answer }}</div>
        </template>
        <template v-else>
          <strong>第 {{ i + 1 }} 步 · 调用工具</strong>
          <div v-for="(c, j) in s.calls" :key="j" class="agent-call">
            <code>{{ c.name }}({{ c.args }})</code>
            <span>→ {{ c.result }}</span>
          </div>
        </template>
      </li>
    </ol>
    <div v-else-if="!running" class="runbox-out sample">
      <div class="runbox-meta"><span>示例（之前运行的结果）</span></div>
      <pre><code>{{ sample || DEFAULT_SAMPLE }}</code></pre>
    </div>
    <OwnKeyPicker v-if="quotaOut" @chosen="run" />
    <div v-if="error" class="runbox-error">{{ error }}</div>
  </div>
</template>
