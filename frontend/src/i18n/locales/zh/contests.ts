export default {
  contests: {
    title: '活动广场',
    subtitle: '用 AI 创作，参加比赛赢取奖励',
    empty: '暂时还没有活动，敬请期待',
    loadFailed: '加载活动失败',
    backToList: '返回活动列表',
    phase: {
      draft: '草稿',
      upcoming: '即将开始',
      submitting: '投稿中',
      submitting_voting: '投稿与点赞中',
      waiting_vote: '等待点赞开始',
      voting: '点赞中',
      tallying: '计票中',
      settled: '已结束',
      cancelled: '已取消'
    },
    schedule: {
      submission: '投稿时间',
      voting: '点赞时间',
      deadline: '点赞截止',
      endsIn: '距截止还有 {time}',
      startsIn: '距开始还有 {time}',
      days: '{n} 天',
      hours: '{n} 小时',
      minutes: '{n} 分钟'
    },
    countdown: {
      starts: '距开始',
      submissionEnds: '距投稿截止',
      votingStarts: '距点赞开始',
      votingEnds: '距点赞截止'
    },
    stats: {
      entries: '{n} 件作品',
      votes: '{n} 个赞'
    },
    prizes: {
      title: '奖项设置',
      place: '第 {n} 名',
      places: '第 {from}–{to} 名',
      balance: '${amount} 账户余额',
      rulesNote: '按点赞截止时的赞数排名，赞数相同时先达到该赞数者优先。'
    },
    rules: '活动规则',
    entries: {
      viewWork: '在画布查看作品',
      title: '参赛作品',
      sortVotes: '按赞数',
      sortRandom: '随机看看',
      sortNew: '最新',
      empty: '还没有作品，快来第一个投稿吧',
      votes: '{n} 个赞',
      vote: '点赞',
      voted: '已点赞',
      unvote: '取消点赞',
      share: '拉票',
      by: '作者 {name}',
      mine: '我的',
      prompt: '提示词',
      withdraw: '撤回作品',
      withdrawConfirm: '确定撤回这件作品吗？撤回后它将退出排名，已获得的赞会退还给点赞的人。',
      status: {
        pending: '审核中',
        approved: '已通过',
        rejected: '未通过',
        withdrawn: '已撤回',
        disqualified: '已取消资格'
      }
    },
    viewer: {
      votesLeft: '你还剩 {left} / {total} 个赞',
      loginToVote: '登录后点赞',
      loginToSubmit: '登录后投稿',
      accountTooNew: '账号注册时间不足 {hours} 小时，暂不能点赞',
      noVotesLeft: '你的赞已用完，可以取消一个再给别的作品',
      myEntries: '我的投稿'
    },
    submit: {
      fromUpload: '上传图片',
      fromWork: '从画布作品选择',
      pickWork: '选择你在 HiveGPT 无限画布发布的作品',
      noWorks: '你还没有在画布发布作品',
      goCreate: '去画布创作',
      workHint: '投稿使用作品的封面图；多图作品要投其他图片，可以在画布的作品页点「投稿参赛」。',
      workPromptHint: '提示词跟随作品：作品公开了提示词，投稿里也会显示。',
      workRequired: '请选择一个作品',
      blocked: { private: '私密作品', review: '审核中' },
      button: '投稿',
      title: '提交作品',
      image: '作品图片',
      imageHint: '支持 PNG / JPG / WebP / GIF，最大 10MB',
      chooseImage: '选择图片',
      titleLabel: '作品名称',
      titlePlaceholder: '给作品起个名字',
      description: '创作说明',
      descriptionPlaceholder: '可选：讲讲你的创作思路',
      prompt: '提示词',
      promptPlaceholder: '可选：公开你使用的提示词',
      submit: '提交',
      submitting: '提交中...',
      success: '投稿成功',
      pendingReview: '投稿成功，审核通过后将展示在作品区',
      imageRequired: '请选择作品图片',
      titleRequired: '请填写作品名称',
      imageTooLarge: '图片不能超过 10MB',
      limit: '每人最多投稿 {n} 件，你还可以投 {left} 件'
    },
    leaderboard: {
      title: '排行榜',
      live: '实时排名',
      final: '最终排名',
      empty: '暂无排名'
    },
    entry: {
      votesLabel: '个赞',
      back: '返回比赛',
      notFound: '作品不存在或已撤回',
      pendingNote: '作品还在审核中，通过后别人才能看到和点赞。你可以先把链接发给朋友。',
      rank: '当前第 {n} 名',
      voteCta: '为 TA 点赞',
      votedCta: '已点赞',
      votedAfterLogin: '已为 TA 点赞 ✓',
      votesLeftAfter: '你还有 {n} 个赞，可以去看看其他作品',
      noVotesLeft: '你的赞已用完',
      selfVote: '不能给自己的作品点赞',
      votingClosed: '现在不在点赞时间',
      shareMine: '邀请好友点赞',
      share: '分享给朋友',
      joinTitle: '你也可以参赛',
      joinPrize: '用 AI 画一张图投稿，{prize}',
      joinNoPrize: '用 AI 画一张图投稿，和大家比一比',
      goCreate: '去画一张',
      seeOthers: '看看其他作品',
      firstPrize: '第一名可得 {reward}',
      posterCaption: '作者 {author} · {contest}',
      posterCta: '扫码为 TA 点赞',
      posterMeta: '注册后即可点赞',
      shareTitle: '拉票分享',
      submittedShare: '投稿成功！分享给朋友，邀请他们来点赞吧',
      submittedPending: '投稿成功，审核通过后就能被点赞。现在就可以把链接分享出去'
    },
    winners: {
      title: '获奖名单'
    },
    toast: {
      voted: '点赞成功',
      unvoted: '已取消点赞',
      withdrawn: '作品已撤回'
    }
  }
}
