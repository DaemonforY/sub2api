export default {
  assistant: {
    title: '智能客服',
    description: '首页右下角的智能客服：用你选的 Key 调模型，根据站内说明和 AI 学习站教程回答。',
    settingsTitle: '客服设置',
    openHome: '去首页试一下',
    hint: '每个回答是一次模型调用，费用记在下面选的 Key 上（在「使用记录」里能看到）。建议在「API 密钥」页给这个 Key 设置额度上限，作为总开销的硬上限。',
    enabled: '开启智能客服',
    key: '使用的 Key',
    keyPlaceholder: '— 选择你的一个 Key —',
    keyHint: '只列出你自己账号下、GPT 分组、已启用的 Key。服务端只保存 Key 的编号，不保存 Key 本身。',
    keyCreate: '去新建一个 Key',
    keyOther: '当前用的是另一位管理员的 Key（编号 {id}），保持不变',
    keyProblem: '当前 Key 不可用：{msg}',
    noKeys: '你还没有可用的 GPT 分组 Key',
    model: '模型',
    modelHint: '默认 gpt-5.6-terra（每问约 $0.01）；gpt-5.6-luna 便宜约 10 倍，回答简单一些。',
    userPerDay: '登录用户每人每天',
    guestPerDay: '未登录访客每 IP 每天',
    guestHint: '设为 0 时，未登录访客看不到客服入口。',
    dailyCap: '全站每天上限',
    dailyCapHint: '所有人加起来每天最多回答多少次，0 表示不限制。',
    today: '今天已回答',
    pages: '知识库页面',
    pagesHint: 'AI 学习站页面 + 站点常见问题',
    saved: '已保存'
  }
}
