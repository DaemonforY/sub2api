import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import TutorStudentView from '../TutorStudentView.vue'

const { getPublic, joinTutor, streamStudentChat } = vi.hoisted(() => ({ getPublic: vi.fn(), joinTutor: vi.fn(), streamStudentChat: vi.fn() }))

vi.mock('@/api/tutors', () => ({ getPublic, joinTutor, streamStudentChat }))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { code: 'AbCd2345' } }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, args?: Record<string, unknown>) => (args ? `${key} ${JSON.stringify(args)}` : key) })
}))

const info = (over = {}) => ({ name: '数学助教', template: 'qa', subject: '数学', greeting: '同学你好', suggestions: ['这节课的重点是什么？'], needs_pass: true, enabled: true, ...over })

describe('tutor student page', () => {
  beforeEach(() => {
    getPublic.mockReset()
    joinTutor.mockReset()
    streamStudentChat.mockReset()
    localStorage.clear()
    sessionStorage.clear()
  })

  it('joins with a name and the class password, then asks', async () => {
    getPublic.mockResolvedValue(info())
    joinTutor.mockResolvedValue({ token: 'tok-1', name: '小明', left: 20 })
    streamStudentChat.mockImplementation(async (code, token, messages, onDelta) => {
      expect(code).toBe('abcd2345')
      expect(token).toBe('tok-1')
      expect(messages).toEqual([{ role: 'user', content: '这节课的重点是什么？' }])
      onDelta('重点是**勾股定理**。')
      return { left: 19 }
    })
    const wrapper = mount(TutorStudentView)
    await flushPromises()
    await wrapper.get('[data-testid="tutor-join-name"]').setValue('小明')
    await wrapper.get('[data-testid="tutor-join-pass"]').setValue('8023')
    await wrapper.get('[data-testid="tutor-join"]').trigger('submit')
    await flushPromises()
    expect(joinTutor).toHaveBeenCalledWith('abcd2345', '小明', '8023')
    expect(JSON.parse(localStorage.getItem('tutor_session_abcd2345') || '{}').token).toBe('tok-1')

    await wrapper.findAll('button').find((b) => b.text() === '这节课的重点是什么？')!.trigger('click')
    await flushPromises()
    expect(wrapper.html()).toContain('<strong>勾股定理</strong>')
    expect(wrapper.text()).toContain('tutors.left {"n":19}')
  })

  it('shows a closed assistant and goes back to joining when the session ends', async () => {
    getPublic.mockResolvedValueOnce(info({ enabled: false }))
    const closed = mount(TutorStudentView)
    await flushPromises()
    expect(closed.get('[data-testid="tutor-student-error"]').text()).toBe('tutors.closed')

    getPublic.mockResolvedValue(info())
    localStorage.setItem('tutor_session_abcd2345', JSON.stringify({ token: 'old', name: '小明', left: 5 }))
    streamStudentChat.mockRejectedValue(Object.assign(new Error('请先输入口令和名字进入'), { reason: 'TUTOR_SESSION' }))
    const wrapper = mount(TutorStudentView)
    await flushPromises()
    await wrapper.get('[data-testid="tutor-chat-input"]').setValue('问题')
    await wrapper.get('[data-testid="tutor-chat-send"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="tutor-join"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('tutors.sessionGone')
    expect(localStorage.getItem('tutor_session_abcd2345')).toBeNull()
  })
})
