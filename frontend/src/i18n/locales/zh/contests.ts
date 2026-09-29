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
      submitting_voting: '投稿与投票中',
      waiting_vote: '等待投票',
      voting: '投票中',
      tallying: '计票中',
      settled: '已结束',
      cancelled: '已取消'
    },
    schedule: {
      submission: '投稿时间',
      voting: '投票时间',
      deadline: '投票截止',
      endsIn: '距截止还有 {time}',
      startsIn: '距开始还有 {time}',
      days: '{n} 天',
      hours: '{n} 小时',
      minutes: '{n} 分钟'
    },
    countdown: {
      starts: '距开始',
      submissionEnds: '距投稿截止',
      votingStarts: '距投票开始',
      votingEnds: '距投票截止'
    },
    stats: {
      entries: '{n} 件作品',
      votes: '{n} 票'
    },
    prizes: {
      title: '奖项设置',
      place: '第 {n} 名',
      places: '第 {from}–{to} 名',
      balance: '${amount} 账户余额',
      rulesNote: '按投票截止时的票数排名，同票时先达到该票数者优先。'
    },
    rules: '活动规则',
    entries: {
      title: '参赛作品',
      sortVotes: '按票数',
      sortNew: '最新',
      empty: '还没有作品，快来第一个投稿吧',
      votes: '{n} 票',
      vote: '投票',
      voted: '已投票',
      unvote: '取消投票',
      by: '作者 {name}',
      mine: '我的',
      prompt: '提示词',
      withdraw: '撤回作品',
      withdrawConfirm: '确定撤回这件作品吗？撤回后它将退出排名，已获得的票数会退还给投票者。',
      status: {
        pending: '审核中',
        approved: '已通过',
        rejected: '未通过',
        withdrawn: '已撤回',
        disqualified: '已取消资格'
      }
    },
    viewer: {
      votesLeft: '你还剩 {left} / {total} 票',
      loginToVote: '登录后参与投票',
      loginToSubmit: '登录后投稿',
      accountTooNew: '账号注册时间不足 {hours} 小时，暂不能投票',
      noVotesLeft: '你的票已用完，可以取消已投的票再改投',
      myEntries: '我的投稿'
    },
    submit: {
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
    winners: {
      title: '获奖名单'
    },
    toast: {
      voted: '投票成功',
      unvoted: '已取消投票',
      withdrawn: '作品已撤回'
    }
  }
}
