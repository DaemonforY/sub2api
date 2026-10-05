import { writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { defineConfig, type DefaultTheme } from 'vitepress'
import guideSidebar from './guide-sidebar.json'
import bigdataSidebar from './bigdata-sidebar.json'
import { lessonHref, trackHref, tracks } from './theme/tracks'

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
        activeMatch: '^/[a-d]/',
        items: tracks.filter((t) => !t.link).map((t) => ({ text: `${t.letter} · ${t.title}`, link: `/${t.id}/`, activeMatch: `^/${t.id}/` })),
      },
      { text: '大数据', link: '/bigdata/', activeMatch: '^/bigdata/' },
      { text: '学员作品', link: '/showcase', activeMatch: '^/showcase' },
      { text: '校园', link: '/campus', activeMatch: '^/campus' },
      { text: '延伸阅读', link: '/guide/', activeMatch: '^/guide/' },
      { text: '付费课程', link: 'https://hivegpt.cn/courses', target: '_self' },
      { text: '回到 HiveGPT', link: 'https://hivegpt.cn/', target: '_self' },
    ],
    sidebar: {
      '/a/': trackSidebar('a'),
      '/b/': trackSidebar('b'),
      '/c/': trackSidebar('c'),
      '/d/': trackSidebar('d'),
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
  transformHtml: (code) => code.replace(/<script id="check-mac-os">[\s\S]*?<\/script>/, ''),
  vite: {
    server: {
      // Local preview: the API comes from a mock or a local backend.
      proxy: { '/api': process.env.LEARN_API_PROXY || 'http://localhost:8092' },
    },
  },
})
