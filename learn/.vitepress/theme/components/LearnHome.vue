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

// 场景玩法 (/scenes/): what AI is good for at work and at home, by role — for people who don't code.
const scenes = [
  { href: '/scenes/office', icon: '💼', title: '办公室日常', text: '邮件、会议纪要、周报、整理表格、对比两版合同' },
  { href: '/scenes/sales', icon: '📈', title: '销售与运营', text: '客户跟进话术、竞品整理、活动方案、数据复盘' },
  { href: '/scenes/teacher', icon: '🍎', title: '老师', text: '教案、出题与批改、家长沟通、课件大纲' },
  { href: '/scenes/student', icon: '🎓', title: '学生', text: '学习计划、读论文、练口语、简历和面试' },
  { href: '/scenes/data', icon: '📊', title: '财务与数据', text: 'Excel 公式、数据清洗、合并几十张表、报表分析' },
  { href: '/scenes/content', icon: '✍️', title: '自媒体与内容', text: '选题、公众号排版、配图、短视频脚本' },
  { href: '/scenes/life', icon: '🏠', title: '个人生活', text: '旅行攻略、记账、看懂体检单、家里的琐事' },
  { href: '/scenes/automation', icon: '🤖', title: '让 AI 帮你动手', text: '批量整理文件、合并表格、抓网页、定时汇总' },
]

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

    <section class="hu-section" data-testid="home-scenes">
      <div class="hu-section-head">
        <div>
          <h2>不写代码，AI 也能帮你做很多事</h2>
          <p>不只是写代码和做 PPT。按你的身份找用法，提示词复制就能用，还能让 AI 帮你批量处理文件。</p>
        </div>
        <a class="hu-btn hu-btn-flat" :href="withBase('/scenes/')">场景玩法 →</a>
      </div>
      <div class="hu-grid hu-grid-4">
        <a v-for="s in scenes" :key="s.href" class="hu-card hu-more" :href="withBase(s.href)">
          <span class="hu-more-icon violet">{{ s.icon }}</span>
          <h3>{{ s.title }}</h3>
          <p>{{ s.text }}</p>
        </a>
      </div>
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
