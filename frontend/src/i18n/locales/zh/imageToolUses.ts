export default {
  imageToolUses: {
    title: '图片工具记录',
    description: '在无限画布使用 AI 抠图、AI 超分的每一次处理和扣费。订阅用户每天有免费次数，用完后按次从余额扣费；处理失败不扣费。',
    today: '今天的免费次数',
    todayValue: '剩 {left} / {daily} 次',
    noSubscription: '订阅后每天可免费用 {daily} 次',
    prices: '价格',
    priceValue: '抠图 ¥{removeBg} / 张，超分 ¥{upscale} / 张',
    monthRuns: '本月处理',
    monthCost: '本月扣费',
    runsValue: '{runs} 次（免费 {free} 次）',
    tools: { remove_bg: 'AI 抠图', upscale: 'AI 超分' },
    allTools: '全部工具',
    columns: { time: '时间', tool: '工具', charge: '扣费', size: '图片大小', duration: '耗时', key: 'API Key' },
    free: '免费',
    empty: '还没有使用记录：在无限画布的「图片工具」里使用 AI 抠图或 AI 超分后，会在这里看到每一次的扣费。',
    openCanvas: '去画布使用'
  }
}
