<script setup lang="ts">
// /learn home: what this is, where you left off, the tracks, further reading.
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { tracks, type Track } from '../tracks'
import { learnConfig, loadProgress, progress, token } from '../api'
import Showcase from './Showcase.vue'

const freeRuns = ref(20)
const signedIn = ref(false)
onMounted(async () => {
  signedIn.value = !!token()
  void loadProgress()
  const cfg = await learnConfig()
  if (cfg.free_runs_per_day) freeRuns.value = cfg.free_runs_per_day
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

const readyLessons = (t: Track) => t.lessons.filter((l) => l.ready).length
const totalMinutes = (t: Track) => t.lessons.reduce((n, l) => n + l.minutes, 0)
</script>

<template>
  <div class="learn-home">
    <section class="home-hero">
      <div class="home-hero-text">
        <h1>边学边做，<br />用一个 Key 学会 <em>AI 应用开发</em>和 <em>AI 创作</em></h1>
        <p>每节课的代码和提示词都能在页面里直接运行，看到真实结果。学完一条路线，做一个能发布出去的结业项目。</p>
        <div class="home-chips">
          <span>页面内直接运行</span>
          <span>登录后每天 {{ freeRuns }} 次免费运行</span>
          <span>进度跟着账号走</span>
        </div>
      </div>
      <div class="home-resume">
        <template v-if="resume">
          <h4>继续学习</h4>
          <div class="home-resume-title">{{ resume.track.letter }}{{ resume.index + 1 }} · {{ resume.lesson.title }}</div>
          <div class="runbox-note">{{ resume.track.title }} · 已完成 {{ doneIn(resume.track) }} / {{ resume.track.lessons.length }} 课</div>
          <div class="home-bar"><b :style="{ width: (doneIn(resume.track) / resume.track.lessons.length) * 100 + '%' }"></b></div>
          <a class="runbox-btn" :href="withBase(`/${resume.track.id}/${resume.lesson.id}`)">继续</a>
        </template>
        <template v-else>
          <h4>从这里开始</h4>
          <div class="home-resume-title">A1 · 第一次调用大模型 API</div>
          <div class="runbox-note">15 分钟：拿到 Key，用 curl / Python / JavaScript 调通第一个请求。</div>
          <a class="runbox-btn" :href="withBase('/a/a1')">开始第一课</a>
          <a v-if="!signedIn" class="runbox-note home-login" href="/login?redirect=%2Flearn%2F">登录后进度会同步到账号 →</a>
        </template>
      </div>
    </section>

    <h2 class="home-h2">选择学习路线 <small>每课 10–20 分钟，按顺序学效果更好</small></h2>
    <div class="home-tracks">
      <component
        :is="t.ready ? 'a' : 'div'"
        v-for="t in tracks"
        :key="t.id"
        :href="t.ready ? withBase(`/${t.id}/`) : undefined"
        :class="['home-track', { soon: !t.ready }]"
        data-testid="home-track"
      >
        <div class="home-track-icon" :style="{ background: t.color }">{{ t.letter }}</div>
        <h3>{{ t.title }}</h3>
        <p>{{ t.tagline }}</p>
        <div class="home-track-meta">
          <template v-if="t.ready">
            <span>{{ readyLessons(t) }} / {{ t.lessons.length }} 课已上线</span>
            <span>约 {{ Math.round(totalMinutes(t) / 6) / 10 }} 小时</span>
          </template>
          <span v-else class="home-soon">即将上线</span>
        </div>
        <div class="home-track-project">结业项目：{{ t.project }}</div>
        <div v-if="t.ready" class="home-track-foot">
          <div class="home-bar"><b :style="{ width: (doneIn(t) / t.lessons.length) * 100 + '%' }"></b></div>
          <span class="runbox-note">{{ doneIn(t) }} / {{ t.lessons.length }}</span>
        </div>
      </component>
    </div>

    <Showcase compact :limit="6">
      <template #head>
        <h2 class="home-h2">学员作品 <small><a :href="withBase('/showcase')">看全部 →</a></small></h2>
      </template>
    </Showcase>

    <div class="home-two">
      <a class="home-card" :href="withBase('/bigdata/')">
        <h3>大数据</h3>
        <p>Apache Spark、Flink、Paimon 源码学习：调度、Shuffle、Checkpoint、状态、LSM 合并和提交，每个结论都有源码位置和实验。</p>
        <span class="lesson-next">去学 →</span>
      </a>
      <a class="home-card" :href="withBase('/guide/')">
        <h3>延伸阅读</h3>
        <p>JavaGuide《AI 应用开发》专题 33 篇：大模型基础、Agent、RAG、系统设计和面试题。适合学完动手课后系统补理论。</p>
        <span class="lesson-next">去读 →</span>
      </a>
      <a class="home-card edu" :href="withBase('/campus')">
        <h3>大学生专区</h3>
        <p>按课程设计、简历作品、实习面试推荐路线；学校邮箱认证享教育优惠，结业证书可以写进简历。</p>
        <span class="lesson-next">去看看 →</span>
      </a>
    </div>
  </div>
</template>
