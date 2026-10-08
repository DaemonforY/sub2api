import { describe, expect, it } from 'vitest'
import { parseWechatResumeRoute, stripWechatResumeQuery } from '../paymentWechatResume'

describe('parseWechatResumeRoute', () => {
  it('prefers the opaque resume token over legacy openid query params', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
    }, [], 88)).toEqual({
      wechatResumeToken: 'resume-token-123',
      paymentType: 'wxpay',
      orderType: 'subscription',
      orderAmount: 0,
      planId: 7,
    })
  })

  it('falls back to legacy openid-based resume when opaque token is absent', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'balance',
    }, [], 88)).toEqual({
      openid: 'openid-123',
      paymentType: 'wxpay',
      orderType: 'balance',
      orderAmount: 12.5,
      planId: undefined,
    })
  })
})

describe('parseWechatResumeRoute membership', () => {
  it('keeps a 创作会员 order a membership order through the WeChat sign-in', () => {
    const parsed = parseWechatResumeRoute({ wechat_resume_token: 'tok', order_type: 'membership', plan_id: '2', payment_type: 'wxpay' }, [], 10)
    expect(parsed?.orderType).toBe('membership')
    expect(parsed?.planId).toBe(2)
  })
})

describe('stripWechatResumeQuery', () => {
  it('removes both opaque-token and legacy resume params from the route query', () => {
    expect(stripWechatResumeQuery({
      foo: 'bar',
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
      state: 'state-123',
      scope: 'snsapi_base',
    })).toEqual({
      foo: 'bar',
    })
  })

  it('reads course orders back from the WeChat callback', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      wechat_resume_token: 'resume-course',
      payment_type: 'wxpay',
      order_type: 'course',
      course_id: '5',
      course: 'ai-agent',
    }, [], 0)).toEqual({
      wechatResumeToken: 'resume-course',
      paymentType: 'wxpay',
      orderType: 'course',
      orderAmount: 0,
      courseId: 5,
    })
    expect(stripWechatResumeQuery({ course: 'ai-agent', course_id: '5', wechat_resume: '1' })).toEqual({ course: 'ai-agent' })
  })
})
