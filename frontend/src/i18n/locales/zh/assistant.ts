export default {
  supportAssistant: {
    open: '智能客服',
    title: '{site} 智能客服',
    subtitle: '根据站内说明和 AI 学习站教程回答',
    close: '关闭',
    clear: '清空对话',
    greeting: '你好，我是 {site} 的智能客服。可以问我注册、API Key、充值订阅、编程工具接入，或者 AI 学习站教程里的问题。',
    greetingTools: '我还能帮你查你自己的余额、API Key 和最近的报错。',
    suggestions: {
      diagnose: '我的 Key 最近为什么报错？',
      q1: '怎么创建 API Key？',
      q2: 'Codex 怎么接入 HiveGPT？',
      q3: '怎么充值，订阅和按量有什么区别？',
      q4: '公众号排版的 AppID 在哪获取？'
    },
    placeholder: '输入问题，Enter 发送，Shift+Enter 换行',
    send: '发送',
    stop: '停止',
    thinking: '正在查资料…',
    checked: '已查看：{list}',
    sources: '相关页面',
    left: '今天还可以问 {n} 次',
    none: '今天的提问次数用完了',
    guestHint: '登录后每天可以问 {n} 次',
    login: '登录',
    disclaimer: 'AI 回答仅供参考，价格和规则以页面为准。不要在这里发送 Key 或密码。',
    disclaimerTools: 'AI 回答仅供参考。需要时会查询你的账户摘要（不含 Key 本身）并发送给模型服务，对话保存 30 天。不要在这里发送 Key 或密码。',
    tooLong: '问题最多 {n} 字',
    error: '出错了：{msg}'
  }
}
