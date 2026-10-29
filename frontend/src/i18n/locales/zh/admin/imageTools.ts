export default {
  imageTools: {
    aspect: {
      title: '生图画幅提示',
      hint: '上游把 size 当作参考，常常返回别的方向（要竖图给横图）。打开后，网关会在 gpt-image 的提示词末尾补一句画幅说明（如「【画幅】竖版构图，画幅宽:高 = 2:3」），对所有用户和客户端生效；提示词里已经写了【画幅】的不再补。下面是按请求尺寸统计的「方向不对」次数，打开前后对比看效果。保存后 1 分钟内生效。',
      enabled: '自动补画幅说明',
      day: '最近 24 小时',
      week: '最近 7 天',
      none: '没有带尺寸的生图请求',
      asked: { portrait: '要竖图', landscape: '要横图', square: '要方图' },
      wrong: '方向不对 {n} 次（{pct}%）',
      saved: '已保存，1 分钟内生效'
    },
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
