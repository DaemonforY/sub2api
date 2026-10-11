import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import HomeCommunityWall from '../HomeCommunityWall.vue'

const { recommendedWorks } = vi.hoisted(() => ({ recommendedWorks: vi.fn() }))

vi.mock('@/api/community', async (importOriginal) => ({ ...(await importOriginal<typeof import('@/api/community')>()), recommendedWorks }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key }) }))

const work = (id: number) => ({
  id, title: `作品 ${id}`, prompt: '', author: { handle: `u${id}`, display_name: '', avatar_url: '' }, cover_url: `/api/v1/community/media/${id}.jpg`,
  cover_thumb_url: `/api/v1/community/media/${id}_t.jpg`, cover_width: 640, cover_height: 800, image_count: 1, like_count: id, remix_count: 0
})

describe('home community works', () => {
  beforeEach(() => recommendedWorks.mockReset())

  it('links each work and the explore page to the canvas, with the AI label', async () => {
    recommendedWorks.mockResolvedValue([work(1), work(2)])
    const wrapper = mount(HomeCommunityWall)
    await flushPromises()
    const links = wrapper.findAll('a').map((a) => a.attributes('href'))
    expect(links[0]).toMatch(/\/explore\?utm_source=hivegpt&utm_medium=home-community$/)
    expect(links[1]).toMatch(/\/w\/1\?utm_source=hivegpt/)
    expect(wrapper.text()).toContain('home.v2.community.aiLabel')
    expect(wrapper.text()).toContain('@u1')
  })

  it('stays hidden without works', async () => {
    recommendedWorks.mockResolvedValue([])
    const wrapper = mount(HomeCommunityWall)
    await flushPromises()
    expect(wrapper.find('[data-testid="home-community-wall"]').exists()).toBe(false)
  })
})
