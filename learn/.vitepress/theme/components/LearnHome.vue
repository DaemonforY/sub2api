<script setup lang="ts">
// /learn home in the HeroUI style: hero with the next step, numbers, the tracks (tabs: all / AI / 大数据),
// the learner wall and more to read.
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { lessonHref, trackHref, tracks, type Track } from '../tracks'
import { learnConfig, loadProgress, progress, token } from '../api'
import Showcase from './Showcase.vue'

const freeRuns = ref(0)
const runEnabled = ref(true)
const signedIn = ref(false)
onMounted(async () => {
  signedIn.value = !!token()
  void loadProgress()
  const cfg = await learnConfig()
  freeRuns.value = cfg.free_runs_per_day || 0
  runEnabled.value = cfg.run_enabled
})

function doneIn(track: Track) {
  return track.lessons.filter((l) => progress.completed[l.id]).length
}

// The next ready lesson in a track that has been started.
const resume = computed(() => {
  for (const track of tracks) {
    if (!doneIn(track)) continue
    const idx = track.lessons.findIndex((l) => l.ready && !progress.completed[l.id])
    if (idx >= 0) return { track, lesson: track.lessons[idx], index: idx }
  }
  return null
})

const BIGDATA = new Set(['e', 'f', 'g'])
const tabs = [
  { id: 'all', label: '全部' },
  { id: 'ai', label: 'AI 应用' },
  { id: 'bigdata', label: '大数据' },
]
const tab = ref('all')
const shown = computed(() =>
  tracks.filter((t) => tab.value === 'all' || (tab.value === 'bigdata') === BIGDATA.has(t.id)),
)

const lessonCount = tracks.reduce((n, t) => n + t.lessons.length, 0)
const hours = Math.round(tracks.reduce((n, t) => n + t.lessons.reduce((m, l) => m + l.minutes, 0), 0) / 60)
const totalMinutes = (t: Track) => t.lessons.reduce((n, l) => n + l.minutes, 0)
const pct = (t: Track) => Math.round((doneIn(t) / t.lessons.length) * 100)

const runChip = computed(() =>
  !runEnabled.value ? '代码可复制到本地运行' : freeRuns.value > 0 ? `登录后每天 ${freeRuns.value} 次免费运行` : '用自己的 Key 在线运行',
)

const more = [
  { href: '/codex/', icon: '⌘', tint: 'violet', title: 'Codex 教程', text: 'CLI、桌面应用、IDE 扩展和云端任务，配置、沙箱、AGENTS.md、Skills、MCP、子代理，接入 HiveGPT 就能用。', cta: '去学' },
  { href: '/bigdata/', icon: '◆', tint: 'blue', title: '大数据专区', text: 'Spark、Flink、Paimon 源码学习：调度、Shuffle、Checkpoint、状态和 LSM 合并，每个结论都有源码位置和实验。', cta: '去学' },
  { href: '/guide/', icon: '❖', tint: 'teal', title: '延伸阅读', text: 'JavaGuide《AI 应用开发》专题 33 篇：大模型基础、Agent、RAG、系统设计和面试题，学完动手课后系统补理论。', cta: '去读' },
  { href: '/campus', icon: '✦', tint: 'amber', title: '大学生专区', text: '按课程设计、简历作品、实习面试推荐路线；学校邮箱认证享教育优惠，结业证书可以写进简历。', cta: '去看看' },
]
</script>

<template>
  <div class="hu-home">
    <section class="hu-hero">
      <div class="hu-hero-glow" aria-hidden="true"></div>
      <a class="hu-announce" :href="withBase('/h/')">
        <span class="hu-chip hu-chip-solid">新课</span>
        H · AI 应用开发进阶上线：评估、护栏、Agent 进阶和生产化 <span aria-hidden="true">→</span>
      </a>
      <h1 class="hu-title">
        边学边做，学会<br />
        <span class="hu-grad violet">AI 应用开发</span>与<span class="hu-grad blue">大数据</span>
      </h1>
      <p class="hu-subtitle">
        {{ tracks.length }} 条路线、{{ lessonCount }} 节课。示例代码在页面里直接运行，看不懂随时问 AI 助教，学完领一张能查验的结业证书。
      </p>
      <div class="hu-hero-actions">
        <a v-if="resume" class="hu-btn hu-btn-primary hu-btn-lg" :href="withBase(lessonHref(resume.track, resume.lesson))">
          继续学习 {{ resume.track.letter }}{{ resume.index + 1 }}
        </a>
        <a v-else class="hu-btn hu-btn-primary hu-btn-lg" :href="withBase('/a/a1')">开始第一课</a>
        <a class="hu-btn hu-btn-bordered hu-btn-lg" href="#tracks">浏览学习路线</a>
      </div>
      <div class="hu-chips">
        <span class="hu-chip">▶ 页面内直接运行</span>
        <span class="hu-chip">⚡ {{ runChip }}</span>
        <span class="hu-chip">✓ 测验 + 结业证书</span>
        <span class="hu-chip">☁ 进度跟着账号走</span>
      </div>

      <div v-if="resume" class="hu-card hu-resume">
        <div class="hu-resume-icon" :style="{ background: resume.track.color }">{{ resume.track.letter }}</div>
        <div class="hu-resume-body">
          <div class="hu-caption">继续上次的学习 · {{ resume.track.title }}</div>
          <div class="hu-resume-title">{{ resume.track.letter }}{{ resume.index + 1 }} · {{ resume.lesson.title }}</div>
          <div class="hu-progress"><b :style="{ width: pct(resume.track) + '%' }"></b></div>
        </div>
        <span class="hu-chip hu-chip-primary">{{ doneIn(resume.track) }} / {{ resume.track.lessons.length }}</span>
      </div>
      <a v-else-if="!signedIn" class="hu-link" href="/login?redirect=%2Flearn%2F">登录后进度和成绩会同步到账号 →</a>
    </section>

    <section class="hu-stats">
      <div class="hu-stat"><b>{{ tracks.length }}</b><span>学习路线</span></div>
      <div class="hu-stat"><b>{{ lessonCount }}</b><span>节课</span></div>
      <div class="hu-stat"><b>{{ hours }}<small>小时</small></b><span>学习时长</span></div>
      <div class="hu-stat"><b>100%</b><span>课程免费阅读</span></div>
    </section>

    <section id="tracks" class="hu-section">
      <div class="hu-section-head">
        <div>
          <h2>选择学习路线</h2>
          <p>每条路线按顺序学效果更好；学完做一个结业项目，领取结业证书。</p>
        </div>
        <div class="hu-tabs" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.id"
            role="tab"
            :aria-selected="tab === t.id"
            :class="['hu-tab', { on: tab === t.id }]"
            data-testid="home-tab"
            @click="tab = t.id"
          >
            {{ t.label }}
          </button>
        </div>
      </div>

      <div class="hu-grid">
        <component
          :is="t.ready ? 'a' : 'div'"
          v-for="t in shown"
          :key="t.id"
          :href="t.ready ? withBase(trackHref(t)) : undefined"
          :class="['hu-card hu-track', { soon: !t.ready }]"
          data-testid="home-track"
        >
          <div class="hu-track-head">
            <div class="hu-track-icon" :style="{ background: t.color }">{{ t.letter }}</div>
            <div>
              <h3>{{ t.title }}</h3>
              <div class="hu-caption">{{ t.lessons.length }} 课 · 约 {{ Math.round(totalMinutes(t) / 6) / 10 }} 小时</div>
            </div>
            <span v-if="!t.ready" class="hu-chip">即将上线</span>
            <span v-else-if="pct(t) === 100" class="hu-chip hu-chip-success">已学完</span>
            <span v-else-if="t.id === 'h'" class="hu-chip hu-chip-primary">新</span>
          </div>
          <p class="hu-track-text">{{ t.tagline }}</p>
          <div class="hu-track-project"><span>结业项目</span>{{ t.project }}</div>
          <div v-if="t.ready" class="hu-track-foot">
            <div class="hu-progress"><b :style="{ width: pct(t) + '%' }"></b></div>
            <span class="hu-caption">{{ doneIn(t) }} / {{ t.lessons.length }}</span>
          </div>
        </component>
      </div>
    </section>

    <section class="hu-section">
      <Showcase compact :limit="6">
        <template #head>
          <div class="hu-section-head">
            <div>
              <h2>学员作品</h2>
              <p>结业学员公开的项目和证书，点开可以查验。</p>
            </div>
            <a class="hu-btn hu-btn-flat" :href="withBase('/showcase')">看全部 →</a>
          </div>
        </template>
      </Showcase>
    </section>

    <section class="hu-section">
      <div class="hu-section-head">
        <div>
          <h2>更多内容</h2>
          <p>教程、专区和延伸阅读，按需要挑着看。</p>
        </div>
      </div>
      <div class="hu-grid hu-grid-4">
        <a v-for="m in more" :key="m.href" class="hu-card hu-more" :href="withBase(m.href)">
          <span :class="['hu-more-icon', m.tint]">{{ m.icon }}</span>
          <h3>{{ m.title }}</h3>
          <p>{{ m.text }}</p>
          <span class="hu-more-cta">{{ m.cta }} →</span>
        </a>
      </div>
    </section>
  </div>
</template>
