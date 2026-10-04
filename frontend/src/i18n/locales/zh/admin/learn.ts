export default {
  learn: {
    title: 'AI 学习',
    description: '学习站 /learn 的在线运行设置和学习数据。',
    runTitle: '在线运行',
    openSite: '打开学习站',
    runHint: '课时里的「动手试试」会用下面这个学习专用 Key 调用本站网关，费用记在这个 Key 上（在「使用记录」里能看到）。登录用户每天有免费次数，用完后提示用自己的 Key 运行。',
    enabled: '开启在线运行',
    apiKey: '学习专用 Key',
    apiKeySet: '已设置，留空表示不修改',
    apiKeyPlaceholder: '粘贴一个 sk- 开头的 Key',
    apiKeyHint: '用管理员账号在「API 密钥」页新建一个 Key，分组选「GPT-按量」，并给它设一个使用额度上限。Key 加密保存，保存后不再显示。',
    model: '模型',
    modelHint: '这个 Key 的分组里可用的模型，建议选便宜、速度快的，例如 gpt-5.5。',
    freeRuns: '每人每天免费次数',
    dailyCap: '全站每天上限',
    dailyCapHint: '所有人加起来每天最多运行多少次，0 表示不限制。',
    stats: { learners: '学过的人', learnersToday: '今天运行的人', runsToday: '今天运行', runs7d: '近 7 天运行', failed7d: '近 7 天失败', tokens7d: '近 7 天 tokens' },
    lesson: '课时',
    completed: '完成人数',
    runs: '成功运行次数',
    noData: '还没有学习数据'
  }
}
