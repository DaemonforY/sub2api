export default {
  imageTools: {
    title: '图片工具',
    description: '无限画布的 AI 抠图和 AI 超分：设置价格、订阅用户每天的免费次数和开关，查看每一次处理和扣费。只有处理成功才扣费。',
    settings: {
      title: '价格与免费额度',
      hint: '价格按余额扣除（当前 1 元充值 = 1 余额）。订阅用户每天先用免费次数（两种工具合计），用完后按次扣余额；没有订阅的用户每次都扣余额。保存后 30 秒内生效。',
      enabled: '开放图片工具',
      priceRemoveBg: 'AI 抠图（每张）',
      priceUpscale: 'AI 超分（每张）',
      freeDaily: '订阅用户每天免费次数',
      save: '保存',
      saved: '已保存',
      notConfigured: '服务器没有配置图片处理服务（IMAGE_TOOLS_BASE_URL），开关打开后也无法使用。'
    },
    stats: {
      today: '今天',
      week: '近 7 天',
      month: '本月',
      runs: '处理次数',
      free: '免费',
      revenue: '收入',
      users: '用户'
    },
    tools: { remove_bg: 'AI 抠图', upscale: 'AI 超分' },
    filters: { search: '搜索用户邮箱', allTools: '全部工具', from: '开始日期', to: '结束日期' },
    columns: { time: '时间', user: '用户', tool: '工具', charge: '扣费', size: '图片大小', duration: '耗时', key: 'API Key' },
    free: '免费',
    empty: '还没有处理记录'
  }
}
