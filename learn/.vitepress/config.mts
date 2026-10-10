import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { defineConfig, type DefaultTheme } from 'vitepress'
import guideSidebar from './guide-sidebar.json'
import bigdataSidebar from './bigdata-sidebar.json'
import codexSidebar from './codex-sidebar.json'
import { findLesson, lessonHref, trackHref, tracks } from './theme/tracks'
import { SECTIONS, seoHead } from './seo-head'
import { pageShareCard } from './share-data'

// AI 学习 at hivegpt.cn/learn. Pages carry the main site's CSP (script-src 'self' + nonce), so
// nothing may be inlined: no appearance script, site data in a separate chunk (metaChunk).

function trackSidebar(id: string): DefaultTheme.SidebarItem[] {
  const track = tracks.find((t) => t.id === id)!
  return [
    {
      text: `${track.letter} · ${track.title}`,
      items: [
        { text: '路线介绍', link: `/${id}/` },
        ...track.lessons.map((l, i) =>
          l.ready ? { text: `${i + 1}. ${l.title}`, link: `/${id}/${l.id}` } : { text: `${i + 1}. ${l.title}（即将上线）` },
        ),
      ],
    },
  ]
}

export default defineConfig({
  base: '/learn/',
  lang: 'zh-CN',
  title: 'HiveGPT AI 学习',
  titleTemplate: ':title · HiveGPT AI 学习',
  description: '边学边做：AI 应用开发、AI 绘画与视频创作。每节课的代码都能在页面里直接运行。',
  appearance: false,
  metaChunk: true,
  cleanUrls: false,
  lastUpdated: false,
  srcExclude: ['scripts/**', 'README.md'],
  sitemap: { hostname: 'https://hivegpt.cn/learn/' },
  head: [
    ['link', { rel: 'icon', href: '/learn/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#7c3aed' }],
  ],
  markdown: {
    lineNumbers: false,
    container: { tipLabel: '提示', warningLabel: '注意', dangerLabel: '警告', infoLabel: '说明', detailsLabel: '详情' },
  },
  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'HiveGPT AI 学习',
    nav: [
      { text: '学习首页', link: '/' },
      {
        text: '学习路线',
        activeMatch: '^/[a-dh]/',
        items: tracks.filter((t) => !t.link).map((t) => ({ text: `${t.letter} · ${t.title}`, link: `/${t.id}/`, activeMatch: `^/${t.id}/` })),
      },
      { text: '场景玩法', link: '/scenes/', activeMatch: '^/scenes/' },
      { text: '生图提示词', link: '/prompts/', activeMatch: '^/prompts/' },
      { text: '接入教程', link: '/connect/', activeMatch: '^/connect/' },
      { text: 'Codex 教程', link: '/codex/', activeMatch: '^/codex/' },
      { text: '大数据', link: '/bigdata/', activeMatch: '^/bigdata/' },
      { text: '学员作品', link: '/showcase', activeMatch: '^/showcase' },
      { text: '校园', link: '/campus', activeMatch: '^/campus' },
      { text: '延伸阅读', link: '/guide/', activeMatch: '^/guide/' },
      { text: '公众号排版', link: 'https://hivegpt.cn/editor/?utm_source=learn&utm_medium=nav', target: '_self' },
      {
        text: 'HiveGPT',
        items: [
          { text: '回到 HiveGPT', link: 'https://hivegpt.cn/', target: '_self' },
          { text: '价格：订阅还是按量', link: 'https://hivegpt.cn/pricing', target: '_self' },
          { text: '付费课程', link: 'https://hivegpt.cn/courses', target: '_self' },
        ],
      },
    ],
    sidebar: {
      '/a/': trackSidebar('a'),
      '/b/': trackSidebar('b'),
      '/c/': trackSidebar('c'),
      '/d/': trackSidebar('d'),
      '/h/': trackSidebar('h'),
      '/bigdata/': [
        { text: '大数据', items: [{ text: '概览', link: '/bigdata/' }] },
        ...bigdataSidebar,
        {
          text: '面试题与模拟面试',
          items: [
            { text: 'Spark 面试题', link: '/bigdata/interview/spark' },
            { text: 'Flink 面试题', link: '/bigdata/interview/flink' },
            { text: 'Paimon 面试题', link: '/bigdata/interview/paimon' },
          ],
        },
      ],
      '/codex/': [{ text: 'Codex 教程', items: [{ text: '教程首页', link: '/codex/' }] }, ...codexSidebar],
      '/connect/': [
        {
          text: '接入教程',
          items: [
            { text: '开始之前', link: '/connect/' },
            { text: 'Codex 接入 HiveGPT', link: '/codex/hivegpt' },
            { text: '在常用工具里配置', link: '/connect/tools' },
            { text: 'Cherry Studio', link: '/connect/cherry-studio' },
            { text: 'Chatbox', link: '/connect/chatbox' },
            { text: '沉浸式翻译', link: '/connect/immersive-translate' },
            { text: 'OpenCode', link: '/connect/opencode' },
            { text: 'Cline / Roo Code', link: '/connect/cline' },
            { text: 'Continue', link: '/connect/continue' },
            { text: 'Dify', link: '/connect/dify' },
            { text: 'Obsidian Copilot', link: '/connect/obsidian' },
            { text: 'Zotero', link: '/connect/zotero' },
            { text: 'n8n', link: '/connect/n8n' },
            { text: 'LobeChat / NextChat', link: '/connect/lobechat-nextchat' },
            { text: 'Open WebUI', link: '/connect/open-webui' },
            { text: 'Bob / Pot 划词翻译', link: '/connect/bob-pot' },
            { text: 'AstrBot（QQ / 微信 / 飞书机器人）', link: '/connect/astrbot' },
            { text: 'Base URL 要不要加 /v1', link: '/connect/base-url' },
            { text: 'GPT 模型怎么选', link: '/connect/models' },
            { text: 'token 怎么算钱', link: '/connect/tokens' },
            { text: 'API 和 ChatGPT Plus', link: '/connect/api-vs-plus' },
            { text: '中转站怎么选', link: '/connect/choose-provider' },
            { text: '用 SDK 调用', link: '/connect/sdk' },
            { text: 'Python 完整示例', link: '/connect/python' },
            { text: 'Node.js 完整示例', link: '/connect/nodejs' },
            { text: 'LangChain', link: '/connect/langchain' },
            { text: 'Java / Spring AI', link: '/connect/java' },
            { text: 'Go', link: '/connect/go' },
            { text: 'Responses API', link: '/connect/responses-api' },
            { text: '看图识图 / 读 PDF', link: '/connect/vision' },
            { text: '图片 API（gpt-image-2）', link: '/connect/image-api' },
            { text: '批量处理 Excel', link: '/connect/excel' },
            { text: '批量翻译字幕', link: '/connect/subtitle-translate' },
            { text: '批量总结文档', link: '/connect/summarize-docs' },
            { text: '飞书 / 企业微信机器人', link: '/connect/feishu-wecom-bot' },
          ],
        },
        {
          text: '报错排查',
          items: [
            { text: '报错速查', link: '/connect/errors/' },
            { text: '401 Key 无效 / 未授权', link: '/connect/errors/401' },
            { text: '余额不足 / 额度用完', link: '/connect/errors/quota' },
            { text: '429 请求太频繁', link: '/connect/errors/429' },
            { text: '模型不存在 / 不支持', link: '/connect/errors/model' },
            { text: '超时 / 流式中断', link: '/connect/errors/timeout' },
            { text: '上下文太长', link: '/connect/errors/context-length' },
          ],
        },
        { text: 'HiveGPT', items: [{ text: '订阅还是按量', link: 'https://hivegpt.cn/pricing' }] },
      ],
      '/prompts/': [
        {
          text: '生图提示词',
          items: [
            { text: '提示词大全', link: '/prompts/' },
            { text: '中文文字不乱码', link: '/prompts/chinese-text' },
            { text: '电商产品图', link: '/prompts/product' },
            { text: '小红书 / 公众号 / 抖音封面', link: '/prompts/cover' },
            { text: '头像与风格化写真', link: '/prompts/avatar' },
          ],
        },
        {
          text: '系统学习',
          items: [
            { text: 'B1 · 提示词的结构', link: '/b/b1' },
            { text: 'B3 · 参考图和图生图', link: '/b/b3' },
            { text: 'B4 · 局部编辑、扩图和抠图', link: '/b/b4' },
          ],
        },
      ],
      '/scenes/': [
        {
          text: '场景玩法',
          items: [
            { text: 'AI 能帮你做什么', link: '/scenes/' },
            { text: '提示词心法', link: '/scenes/prompting' },
          ],
        },
        {
          text: '按身份找用法',
          items: [
            { text: '办公室日常', link: '/scenes/office' },
            { text: '销售与运营', link: '/scenes/sales' },
            { text: '老师', link: '/scenes/teacher' },
            { text: '学生', link: '/scenes/student' },
            { text: '财务与数据', link: '/scenes/data' },
            { text: '自媒体与内容', link: '/scenes/content' },
            { text: '个人生活', link: '/scenes/life' },
            { text: '小老板与个体户', link: '/scenes/business' },
          ],
        },
        {
          text: '让 AI 帮你动手',
          items: [
            { text: '自动化入门', link: '/scenes/automation' },
            { text: '批量处理文件和表格', link: '/scenes/automation-files' },
            { text: '网页整理与浏览器操作', link: '/scenes/automation-web' },
            { text: '定时任务和一键小工具', link: '/scenes/automation-schedule' },
          ],
        },
        {
          text: '准备工作',
          items: [
            { text: '接入教程', link: '/connect/' },
            { text: '安装 Codex 并接入 HiveGPT', link: '/c/c1' },
          ],
        },
      ],
      '/guide/': [{ text: '延伸阅读（JavaGuide）', items: [{ text: '目录与来源', link: '/guide/' }] }, ...guideSidebar],
    },
    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '目录',
    darkModeSwitchLabel: '外观',
    notFound: { title: '页面不存在', quote: '这个地址没有内容，可能已经换了位置。', linkLabel: '回到学习首页', linkText: '回到学习首页' },
    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索', buttonAriaLabel: '搜索' },
          modal: {
            noResultsText: '没有找到结果',
            resetButtonTitle: '清除',
            displayDetails: '显示详情',
            footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' },
          },
        },
      },
    },
    footer: {
      message: '代码示例在页面里运行时使用 HiveGPT 的模型接口。延伸阅读来自 JavaGuide（Apache-2.0），版权归原作者。',
      copyright: '© HiveGPT',
    },
  },
  // The default theme inlines a tiny script for the ⌘ hint in search; the CSP would block it.
  // The main site's 「我的学习」 page reads the tracks from here.
  buildEnd(site) {
    const data = tracks.map((t) => ({
      id: t.id,
      letter: t.letter,
      title: t.title,
      project: t.project,
      href: `/learn${trackHref(t)}`,
      lessons: t.lessons.filter((l) => l.ready).map((l) => ({ id: l.id, title: l.title, minutes: l.minutes, href: `/learn${lessonHref(t, l)}.html` })),
    }))
    writeFileSync(join(site.outDir, 'tracks.json'), JSON.stringify(data))
  },
  transformHead: ({ pageData, siteConfig }) => seoHead(pageData, siteConfig.srcDir),
  // Share card data (title, key points, reading time, reprint source) for the 分享 button.
  transformPageData(pageData, { siteConfig }) {
    if (pageData.isNotFound) return
    let md = ''
    try {
      md = readFileSync(join(siteConfig.srcDir, pageData.relativePath), 'utf8')
    } catch {
      return
    }
    const fm = pageData.frontmatter
    const lesson = fm.lesson ? findLesson(String(fm.lesson)) : null
    const label = lesson ? `${lesson.track.letter} · ${lesson.track.title}` : SECTIONS[pageData.relativePath.split('/')[0]] || ''
    pageData.shareCard = pageShareCard(md, fm, pageData.title, label, lesson?.lesson.minutes || 0)
  },
  transformHtml: (code) => code.replace(/<script id="check-mac-os">[\s\S]*?<\/script>/, ''),
  vite: {
    server: {
      // Local preview: the API comes from a mock or a local backend.
      proxy: { '/api': process.env.LEARN_API_PROXY || 'http://localhost:8092' },
    },
  },
})
