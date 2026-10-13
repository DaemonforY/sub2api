export default {
  games: {
    title: '小游戏',
    subtitle: '几分钟玩一局，测完还能生成分享卡片发朋友圈。',
    play: '开始玩',
    minutes: '约 {n} 分钟',
    soon: '更多小游戏正在做',
    soonDesc: '想玩什么？在首页右下角的智能客服里告诉我们。',
    list: {
      abti: { title: 'ABTI · AI 打工人格测试', desc: '31 道题、15 个维度，测出你在 AI 时代是哪种打工人。', tag: '人格测试' },
      style: { title: '测测你的写作风格', desc: '贴几篇你写的文章，看看你的句子节奏、口头禅和 AI 味。', tag: '写作' }
    },
    abti: {
      title: 'ABTI · AI 打工人格测试',
      badge: '小游戏 · 约 3 分钟',
      subtitle: 'AI 来了以后，你是派活的、摸鱼的，还是校对 AI 的？31 道题，测出你的 AI 打工人格。',
      facts: { questions: '31 道题', dims: '15 个维度', types: '16 种人格 + 隐藏款' },
      start: '开始测试',
      restart: '再测一次',
      lastResult: '查看上次的结果（{code} {name}）',
      friend: '你的朋友测出了',
      friendAsk: '你是哪一种？',
      disclaimer: '本测试仅供娱乐，题目和人格都是我们自己编的，别拿它当心理诊断、面试或分手依据。',
      aiArt: '插图由 AI 生成。',
      quiz: {
        progress: '第 {n} / {total} 题',
        back: '上一题',
        quit: '退出',
        extra: '附加题'
      },
      result: {
        kicker: { normal: '你的 AI 打工人格', hidden: '隐藏人格已解锁', fallback: '系统兜底人格' },
        match: '匹配度 {n}% · 命中 {exact}/15 个维度',
        hiddenMatch: '隐藏人格 · 常规人格最接近 {code} {name}（{n}%）',
        fallbackMatch: '常规人格最高匹配只有 {n}%（{code} {name}）',
        dims: '你的 15 个维度',
        level: { L: '低', M: '中', H: '高' },
        close: '和你最像的另外几种',
        all: '全部人格',
        hiddenSlot: '隐藏款',
        tip: 'AI 打工建议',
        share: '生成分享卡片',
        restart: '再测一次',
        hall: '去游戏大厅',
        cta: {
          image: '让 AI 画一张你的人格头像',
          code: '看看 Codex 怎么接入',
          write: '打开公众号 AI 写作',
          learn: '去 AI 学习站',
          plaza: '看看都能用哪些模型',
          invite: '去拿我的邀请链接'
        },
        signup: '免费注册，送 ${n} 额度',
        signupPlain: '免费注册',
        imagePrompt: '画一个 Q 版卡通打工人形象，人设是「{name}」：{intro} 可爱贴纸风，粗描边，柔和的淡紫色和蜂蜜黄配色，干净的浅色背景，不要文字。'
      },
      share: {
        brand: 'ABTI · AI 打工人格测试',
        title: '我是 {code}「{name}」',
        match: '匹配度 {n}%',
        cta: '扫码测测你是哪种 AI 打工人'
      }
    }
  }
}
