export default {
  sites: {
    title: '网站托管',
    description: '订阅用户发布的网站：查看和下线违规网站、处理访客举报、设置配额和价格。',
    tabs: { sites: '网站', reports: '举报', settings: '设置' },
    search: '搜索网站名、名称或用户邮箱',
    allStatuses: '全部状态',
    status: { active: '访问中', disabled: '已下线', unpaid: '欠费暂停', lapsed: '订阅到期暂停' },
    columns: { site: '网站', owner: '用户', status: '状态', size: '大小 · 文件数', billing: '计费', updated: '更新时间' },
    free: '赠送',
    paidUntil: '付费至 {date}',
    empty: '还没有网站',
    actions: { disable: '下线', enable: '恢复上线', delete: '删除' },
    disableHint: '下线后网站立即无法访问，用户不能再更新；原因会显示给用户。',
    disablePlaceholder: '下线原因，例如：仿冒银行登录页',
    disabled: '已下线',
    enabled: '已恢复上线',
    deleteConfirm: '删除 {url} 和它的全部文件，不能恢复。确定吗？',
    reports: {
      time: '时间',
      site: '网站',
      reason: '原因',
      contact: '举报人',
      resolve: '已处理',
      dismiss: '忽略',
      gone: '已删除',
      empty: '没有举报',
      statuses: { open: '待处理', resolved: '已处理', dismissed: '已忽略' }
    },
    settings: {
      hint: '网站地址为 「网站名」.{domain}。只有订阅用户能发布；每人前几个网站随订阅赠送，超出的按 30 天从余额扣费，余额不足时暂停访问。订阅到期后先保留若干天再暂停，暂停超过保留天数后删除。保存后 30 秒内生效。',
      noDomain: '服务器没有配置托管域名（SITES_DOMAIN），网站发布不可用。',
      enabled: '开放网站发布',
      max_per_user: '每人最多网站数',
      free_per_user: '订阅赠送网站数',
      extra_price: '超出部分价格（元 / 个 / 30 天）',
      max_mb: '单个网站大小上限（MB）',
      max_files: '单个网站文件数上限',
      grace_days: '订阅到期后保留天数',
      retention_days: '暂停后保留天数（之后删除）',
      saved: '已保存'
    }
  }
}
