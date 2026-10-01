export default {
  community: {
    title: '画布社区',
    description: '无限画布社区的作品审核、举报处理、精选和发布限制。',
    tabs: { pending: '待审核', reported: '被举报', approved: '已公开', hidden: '已隐藏', reports: '举报记录', settings: '设置' },
    status: { approved: '已公开', pending: '待审核', rejected: '未通过', hidden: '已隐藏' },
    untitled: '未命名作品',
    featured: '精选',
    images: '张图',
    reports: '{count} 次举报',
    flags: '命中',
    actions: { approve: '通过', reject: '不通过', hide: '隐藏', feature: '设为精选', unfeature: '取消精选', ban: '限制作者发布' },
    reasonPlaceholder: '原因（会通知作者），例如：涉嫌侵权',
    banConfirm: '限制 @{handle} 发布作品？对方的主页和作品会对其他人隐藏。',
    banned: '已限制该作者发布',
    done: '已处理',
    empty: '这里没有作品',
    noReports: '还没有举报',
    prev: '上一页',
    next: '下一页',
    resolve: '已处理',
    dismiss: '忽略',
    columns: { time: '时间', work: '作品', reason: '原因', status: '状态' },
    reasons: { porn: '色情低俗', violence: '暴力血腥', politics: '违法违规', copyright: '侵犯版权', fraud: '诈骗广告', spam: '垃圾内容', other: '其他' },
    reportStatus: { open: '待处理', resolved: '已处理', dismissed: '已忽略' },
    reviewAll: '所有新作品都需要人工审核后才公开',
    reviewAllHint: '关闭时只有命中敏感词的作品进入待审核，其余直接公开；任何作品都可以被举报。'
  }
}
