import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import CourseDetailView from '../CourseDetailView.vue'

const { getCourse, push, auth } = vi.hoisted(() => ({ getCourse: vi.fn(), push: vi.fn(), auth: { isAuthenticated: false } }))

vi.mock('@/api/courses', () => ({ getCourse }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { slug: 'ai-agent' } }), useRouter: () => ({ push }) }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const stubs = { PlazaNavBar: true, RouterLink: { template: '<a><slot /></a>' } }
const course = {
  id: 5, slug: 'ai-agent', title: 'AI Agent 实战', subtitle: '从零做 Agent', category: 'AI 开发', cover_url: '', price: 199, original_price: 299,
  intro_md: '## 适合谁\n<script>alert(1)</script>后端开发', trial_md: '', faq_md: '',
  outline: [{ title: '第一章', lessons: [{ title: 'Agent 是什么', duration: '12:00', trial: true }, { title: '工具调用', duration: '', trial: false }] }],
  status: 'published', sort_order: 0, lesson_count: 2, student_count: 3, owned: false, created_at: '', updated_at: ''
}

describe('CourseDetailView', () => {
  beforeEach(() => {
    getCourse.mockReset()
    push.mockReset()
    auth.isAuthenticated = false
  })

  it('renders the page (sanitised Markdown, outline with previews) and sends guests to sign in before checkout', async () => {
    getCourse.mockResolvedValue(course)
    const wrapper = mount(CourseDetailView, { global: { stubs } })
    await flushPromises()
    expect(getCourse).toHaveBeenCalledWith('ai-agent')
    const page = wrapper.get('[data-testid="course-detail"]')
    expect(page.text()).toContain('AI Agent 实战')
    expect(page.html()).not.toContain('<script>')
    expect(page.find('h2').exists()).toBe(true)
    expect(wrapper.get('[data-testid="course-outline"]').text()).toContain('courses.trial')

    await wrapper.get('[data-testid="course-buy"]').trigger('click')
    expect(push).toHaveBeenCalledWith({ path: '/login', query: { redirect: '/purchase?course=ai-agent' } })
  })

  it('opens checkout for signed-in users and 我的课程 for owners', async () => {
    auth.isAuthenticated = true
    getCourse.mockResolvedValue(course)
    const wrapper = mount(CourseDetailView, { global: { stubs } })
    await flushPromises()
    await wrapper.get('[data-testid="course-buy"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/purchase?course=ai-agent')

    getCourse.mockResolvedValue({ ...course, owned: true })
    const owned = mount(CourseDetailView, { global: { stubs } })
    await flushPromises()
    expect(owned.get('[data-testid="course-buy"]').text()).toBe('courses.mine.go')
    await owned.get('[data-testid="course-buy"]').trigger('click')
    expect(push).toHaveBeenLastCalledWith('/my-courses')
  })
})
