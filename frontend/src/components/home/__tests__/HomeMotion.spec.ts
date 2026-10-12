import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import HomeMotion from '../HomeMotion.vue'
import { detectMotion } from '../motion/tier'

describe('home motion layer', () => {
  it('stays an empty box where there is no WebGL (and never loads three.js)', async () => {
    expect(detectMotion()).toBeNull() // jsdom has no WebGL
    const w = mount(HomeMotion, { props: { kind: 'hive' } })
    await new Promise((r) => setTimeout(r, 400))
    expect(w.find('canvas').exists()).toBe(false)
    expect(w.emitted('started')).toBeUndefined()
  })
})
