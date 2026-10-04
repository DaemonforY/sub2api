// Learning tracks and lessons. Lesson pages live at /<track>/<lesson>.md; lessons not ready yet
// are listed as 「即将上线」. Shared by the config (sidebars) and the theme (home, track pages).

export interface Lesson {
  id: string
  title: string
  minutes: number
  ready: boolean
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
]

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
  return `/${id[0]}/${id}`
}

export const MAIN_SITE = 'https://hivegpt.cn'
export const CANVAS_SITE = 'https://canvas.hivegpt.cn'
