// Learning tracks and lessons. Lesson pages live at /<track>/<lesson>.md (the 大数据 tracks' at
// their article pages, see `link`); lessons not ready yet are listed as 「即将上线」. Shared by the
// config (sidebars) and the theme (home, track pages).
import bigdataTracks from '../bigdata-tracks.json'

export interface Lesson {
  id: string
  title: string
  minutes: number
  ready: boolean
  /** Page path when not /<track>/<id> (大数据 articles). */
  link?: string
}

export interface Track {
  id: string
  letter: string
  title: string
  tagline: string
  audience: string
  project: string
  color: string
  ready: boolean
  /** Lessons have quizzes (track D practises with mock interviews instead). */
  quizzes: boolean
  /** Track page when not /<id>/ (大数据 tracks). */
  link?: string
  lessons: Lesson[]
}

export const tracks: Track[] = [
  {
    id: 'a',
    letter: 'A',
    title: 'AI 应用开发入门',
    tagline: '从第一次调用 API 到结构化输出、Function Calling、RAG、Agent、MCP，最后上线一个聊天机器人。',
    audience: '后端 / 前端开发、计算机专业学生',
    project: '做一个网页聊天 / 问答机器人，用「网站托管」发布出去',
    color: 'linear-gradient(135deg,#8b5cf6,#6d28d9)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'a1', title: '第一次调用大模型 API', minutes: 15, ready: true },
      { id: 'a2', title: '流式输出和多轮对话', minutes: 15, ready: true },
      { id: 'a3', title: '结构化输出：让模型稳定返回 JSON', minutes: 15, ready: true },
      { id: 'a4', title: 'Function Calling：让模型调用你的函数', minutes: 20, ready: true },
      { id: 'a5', title: 'Embedding 和 RAG 问答', minutes: 20, ready: true },
      { id: 'a6', title: 'Agent 循环：让模型自己一步步完成任务', minutes: 20, ready: true },
      { id: 'a7', title: 'MCP 入门：把工具做成标准插件', minutes: 15, ready: true },
      { id: 'a8', title: '上线：成本、限流和错误重试', minutes: 15, ready: true },
    ],
  },
  {
    id: 'b',
    letter: 'B',
    title: 'AI 绘画与视频创作',
    tagline: '提示词、模板、参考图、局部编辑、视频生成，到发布作品和参加比赛，在 HiveGPT 无限画布里边学边画。',
    audience: '设计、运营、自媒体，零基础也可以',
    project: '在画布社区发布一个作品集（至少 3 张）',
    color: 'linear-gradient(135deg,#2dd4bf,#0d9488)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'b1', title: '提示词的结构', minutes: 15, ready: true },
      { id: 'b2', title: '用提示词模板，少走弯路', minutes: 10, ready: true },
      { id: 'b3', title: '参考图和图生图', minutes: 15, ready: true },
      { id: 'b4', title: '局部编辑、扩图和抠图', minutes: 15, ready: true },
      { id: 'b5', title: '视频生成', minutes: 15, ready: true },
      { id: 'b6', title: '发布作品和做同款', minutes: 10, ready: true },
      { id: 'b7', title: '参加比赛', minutes: 10, ready: true },
    ],
  },
  {
    id: 'c',
    letter: 'C',
    title: 'AI 编程助手上手',
    tagline: '把 Codex CLI 接入 HiveGPT，让 AI 帮你写一个网页、改需求、发布上线。',
    audience: '想用 AI 写代码的人，有没有编程基础都可以',
    project: '用编程助手做一个小网站并发布',
    color: 'linear-gradient(135deg,#f59e0b,#d97706)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'c1', title: '安装 Codex CLI 并接入 HiveGPT', minutes: 15, ready: true },
      { id: 'c2', title: '让 AI 写一个网页', minutes: 20, ready: true },
      { id: 'c3', title: '调试和改需求', minutes: 15, ready: true },
      { id: 'c4', title: '发布你的网站', minutes: 10, ready: true },
    ],
  },
  {
    id: 'd',
    letter: 'D',
    title: 'AI 面试训练',
    tagline: '大模型、Agent、RAG、系统设计题库，AI 扮演面试官给你打分。',
    audience: '准备 AI 应用开发岗位面试的同学',
    project: '4 组模拟面试都拿到 60 分以上',
    color: 'linear-gradient(135deg,#f472b6,#db2777)',
    ready: true,
    quizzes: false,
    lessons: [
      { id: 'd1', title: '大模型基础', minutes: 30, ready: true },
      { id: 'd2', title: 'Agent', minutes: 30, ready: true },
      { id: 'd3', title: 'RAG', minutes: 30, ready: true },
      { id: 'd4', title: 'AI 应用系统设计', minutes: 30, ready: true },
    ],
  },
  {
    id: 'h',
    letter: 'H',
    title: 'AI 应用开发进阶',
    tagline: '学完 A 之后：提示词工程、评估、上下文工程、RAG 进阶、工作流与多智能体、安全护栏、Agent 进阶和生产化。',
    audience: '学完 A 路线、准备把 AI 应用做到能上线的开发者',
    project: '做一个带评估集、安全护栏、日志和一项成本优化的 AI 应用，发布出去',
    color: 'linear-gradient(135deg,#0ea5e9,#4f46e5)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'h1', title: '提示词工程进阶：系统提示词、少样本和自检', minutes: 15, ready: true },
      { id: 'h2', title: '评估：给 AI 应用写测试', minutes: 20, ready: true },
      { id: 'h3', title: '上下文工程：长对话、记忆和压缩', minutes: 20, ready: true },
      { id: 'h4', title: 'RAG 进阶：查询改写、混合检索和重排', minutes: 20, ready: true },
      { id: 'h5', title: '工作流与多智能体：路由、并行和评审', minutes: 20, ready: true },
      { id: 'h6', title: '安全护栏：提示注入、越权和输出检查', minutes: 20, ready: true },
      { id: 'h7', title: 'Agent 进阶：规划、反思和人工确认', minutes: 20, ready: true },
      { id: 'h8', title: '生产化：可观测、缓存、模型路由和成本', minutes: 20, ready: true },
    ],
  },
]

// 学画画 · 从 0 到 1 (docs/DRAWING_COURSE_PLAN.md): three tracks with in-lesson drawing practice.
// Their certificates open once every lesson is online (cert_closed in the backend catalog).
tracks.push(
  {
    id: 'i',
    letter: 'I',
    title: '视觉基本功',
    tagline: '学会看画面：形、透视、光影、色彩、构图、镜头、质感和风格。每课先看名画，再学在提示词里怎么说，最后当场画一张请 AI 老师点评。',
    audience: '想把画画好的所有人，零基础可以',
    project: '每个课内练习的 AI 点评都达到及格分',
    color: 'linear-gradient(135deg,#f97316,#db2777)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'i1', title: '学会看：形与剪影', minutes: 20, ready: false },
      { id: 'i2', title: '透视与空间', minutes: 25, ready: false },
      { id: 'i3', title: '光影与明暗', minutes: 25, ready: true },
      { id: 'i4', title: '色彩', minutes: 25, ready: false },
      { id: 'i5', title: '构图', minutes: 25, ready: true },
      { id: 'i6', title: '镜头语言', minutes: 20, ready: false },
      { id: 'i7', title: '质感与材质', minutes: 20, ready: false },
      { id: 'i8', title: '媒介与风格', minutes: 25, ready: false },
      { id: 'i9', title: '综合创作：从想法到成图', minutes: 30, ready: false },
    ],
  },
  {
    id: 'j',
    letter: 'J',
    title: '题材实战',
    tagline: '人像、动物、风景、建筑、美食、产品、海报、绘本、动漫、漫画、国风、Logo 与信息图：每个题材一张配方卡和一道命题练习。',
    audience: '学过视觉基本功，或者手上有具体要画的东西',
    project: '每个课内练习的 AI 点评都达到及格分',
    color: 'linear-gradient(135deg,#eab308,#ea580c)',
    ready: true,
    quizzes: true,
    lessons: [
      { id: 'j1', title: '人像与人物', minutes: 25, ready: true },
      { id: 'j2', title: '动物与宠物', minutes: 20, ready: false },
      { id: 'j3', title: '风景与自然', minutes: 20, ready: false },
      { id: 'j4', title: '城市、建筑与室内', minutes: 20, ready: false },
      { id: 'j5', title: '美食与静物', minutes: 20, ready: false },
      { id: 'j6', title: '产品图与电商主图', minutes: 25, ready: false },
      { id: 'j7', title: '海报与文字', minutes: 25, ready: false },
      { id: 'j8', title: '插画与绘本', minutes: 25, ready: false },
      { id: 'j9', title: '动漫角色与设定图', minutes: 25, ready: false },
      { id: 'j10', title: '漫画分镜', minutes: 25, ready: false },
      { id: 'j11', title: '国风与诗画', minutes: 25, ready: false },
      { id: 'j12', title: 'Logo、图标与信息图', minutes: 25, ready: false },
    ],
  },
  {
    id: 'k',
    letter: 'K',
    title: '艺术修养',
    tagline: '怎么看一幅画；中国画、诗书画印；从文艺复兴到印象派、现代艺术；东亚视觉；设计与摄影的眼光；AI 时代的审美与伦理。',
    audience: '想提高审美、看得懂画展的人',
    project: '测验和赏析写作达标',
    color: 'linear-gradient(135deg,#a16207,#57534e)',
    ready: false,
    quizzes: true,
    lessons: [
      { id: 'k1', title: '怎么看一幅画', minutes: 20, ready: false },
      { id: 'k2', title: '中国画入门', minutes: 25, ready: false },
      { id: 'k3', title: '诗书画印', minutes: 20, ready: false },
      { id: 'k4', title: '从文艺复兴到巴洛克', minutes: 25, ready: false },
      { id: 'k5', title: '印象派与后印象派', minutes: 25, ready: false },
      { id: 'k6', title: '现代艺术为什么「看不懂」', minutes: 25, ready: false },
      { id: 'k7', title: '东亚视觉：浮世绘、动画与国漫', minutes: 20, ready: false },
      { id: 'k8', title: '设计与摄影的眼光', minutes: 20, ready: false },
      { id: 'k9', title: 'AI 时代的审美与伦理', minutes: 20, ready: false },
    ],
  },
)

// Spark / Flink / Paimon, generated by scripts/import_bigdata.py.
tracks.push(...(bigdataTracks as Track[]))

export function trackHref(track: Track): string {
  return track.link || `/${track.id}/`
}

export function lessonHref(track: Track, lesson: Lesson): string {
  return lesson.link || `/${track.id}/${lesson.id}`
}

/** Lesson id of a page path such as /a/a3 or /bigdata/spark/lecture-03-shuffle (with or without .html). */
export function lessonByPath(path: string): string | null {
  const clean = path.replace(/\.html$/, '').replace(/^\/learn/, '')
  for (const t of tracks) for (const l of t.lessons) if (lessonHref(t, l) === clean) return l.id
  return null
}

export function findTrack(id: string): Track | undefined {
  return tracks.find((t) => t.id === id)
}

export function findLesson(id: string): { track: Track; lesson: Lesson; index: number } | null {
  for (const track of tracks) {
    const index = track.lessons.findIndex((l) => l.id === id)
    if (index >= 0) return { track, lesson: track.lessons[index], index }
  }
  return null
}

export function lessonLink(id: string): string {
  const found = findLesson(id)
  return found ? lessonHref(found.track, found.lesson) : `/${id[0]}/${id}`
}

export const MAIN_SITE = 'https://hivegpt.cn'
export const CANVAS_SITE = 'https://canvas.hivegpt.cn'
