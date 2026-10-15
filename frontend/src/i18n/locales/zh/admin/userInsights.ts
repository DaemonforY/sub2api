export default {
  userInsights: {
    title: '用户画像',
    description: '注册、用量和付款的风险视角：谁在白嫖免费额度、哪些账号是同一个人、谁在用中转工具。',
    tabs: {
      overview: '概览',
      profile: '用户画像',
      events: '风控事件'
    },
    range: '{n} 天',
    refresh: '刷新',
    since: '统计区间：{since} 起，共 {days} 天；不含管理员账号',
    loadFailed: '加载失败',
    empty: '暂无数据',
    total: '合计',
    segments: {
      inactive: '未激活',
      trial: '试用中',
      paying: '付费',
      churned: '流失',
      hint: {
        inactive: '注册后从没调用过 API',
        trial: '用过 API，从没付过款',
        paying: '付过款，最近 14 天有使用（或 30 天内付过款）',
        churned: '付过款，之后没再使用'
      }
    },
    signups: {
      title: '每日注册',
      hint: '按注册日统计，每个账号归入它现在所在的分层。'
    },
    spend: {
      title: '每日消耗与收入',
      hint: '免费消耗 = 从没付过款的账号的用量（美元）；总用量含所有账号；收入只算真实付款（人民币）。',
      free: '免费消耗 ($)',
      used: '总用量 ($)',
      paid: '收入 (¥)'
    },
    clients: {
      title: '客户端分布',
      hint: '按请求的 User-Agent 归类。'
    },
    clientClass: {
      codex_desktop: 'Codex 桌面版',
      codex_cli: 'Codex CLI',
      claude_code: 'Claude Code',
      cherry_studio: 'Cherry Studio',
      relay: '中转工具',
      browser: '浏览器',
      sdk: 'SDK/脚本',
      other: '其他'
    },
    clusters: {
      title: '关联账号群',
      hint: '同一个浏览器或同一个 IP 段注册了多个账号；免费消耗只算其中没付过款的账号。',
      kind: {
        visitor: '同浏览器',
        ip_prefix: '同 IP 段'
      },
      members: '{n} 个账号'
    },
    relayUsers: {
      title: '中转工具用户',
      hint: '用中转/转售工具（AxonHub、new-api 等）或裸 HTTP 库调用的账号，按消耗排序。'
    },
    badge: {
      disabled: '已封禁',
      shadowed: '静默',
      paid: '付费',
      free: '试用',
      admin: '管理员'
    },
    col: {
      date: '日期',
      class: '客户端',
      requests: '请求数',
      users: '用户数',
      cost: '消耗 ($)',
      user: '用户',
      userAgent: 'User-Agent',
      kind: '类型',
      key: '标识',
      members: '账号',
      freeUsd: '免费消耗 ($)',
      paidCny: '付款 (¥)',
      spendUsd: '区间消耗 ($)',
      lastSeen: '最近一次',
      count: '次数',
      ip: 'IP',
      prefix: 'IP 段',
      group: '分组',
      model: '模型',
      time: '时间',
      rule: '规则',
      detail: '详情',
      orderType: '订单类型',
      paymentType: '支付方式',
      amount: '金额',
      payAmount: '实付',
      status: '状态',
      createdAt: '创建时间',
      paidAt: '付款时间',
      startsAt: '开始',
      expiresAt: '到期',
      dailyUsage: '今日用量 / 日限额 ($)',
      via: '关联方式'
    },
    profile: {
      searchPlaceholder: '输入用户 ID 或邮箱',
      search: '查询',
      prompt: '输入用户 ID 或邮箱查看画像；在其他列表里点用户也会跳到这里。',
      id: 'ID',
      status: '状态',
      role: '角色',
      createdAt: '注册时间',
      signupSource: '注册方式',
      balance: '余额',
      riskScore: '风险分',
      riskLevel: {
        low: '低风险',
        medium: '中风险',
        high: '高风险'
      },
      noReasons: '没有发现风险信号',
      flags: '状态标记',
      banned: '已封禁',
      shadowed: '静默限制',
      shadowReason: '原因：{reason}',
      shadowSource: {
        auto: '注册规则自动',
        admin: '管理员'
      },
      shadowMeta: '{source} · {time}',
      noFlags: '无',
      signup: '注册信息',
      attribution: {
        source: '来源',
        medium: '媒介',
        campaign: '活动',
        aff_code: '邀请码',
        referrer_host: '来路域名',
        landing_path: '落地页'
      },
      visitorId: '浏览器 ID (visitor_id)',
      signupPrefix: '注册 IP 段',
      trialGranted: '发放试用',
      yes: '是',
      no: '否',
      webPrefixes: '网页访问 IP 段',
      apiIps: 'API 调用 IP',
      clients: 'API 客户端',
      groups: '使用的分组',
      models: '使用的模型',
      spend: '消耗与付款',
      spendTotal: '总消耗',
      spendFreeSub: '试用订阅消耗',
      spendPaidSub: '付费订阅消耗',
      spendBalance: '余额消耗',
      paidCny: '累计付款',
      everPaid: '付过款',
      spendWindow: '用量统计最近 90 天',
      payments: '付款记录',
      subscriptions: '订阅',
      invites: '邀请关系',
      inviteUp: '邀请人链',
      inviteDown: '邀请的人',
      noInviter: '没有邀请人',
      noInvitees: '没有邀请别人',
      linked: '关联账号',
      events: '风控事件'
    },
    via: {
      visitor: '同浏览器',
      ip_prefix: '同 IP 段',
      api_ip: '同 API IP',
      user_agent: '同中转 UA',
      inviter: '邀请人',
      invitee: '被邀请人'
    },
    rules: {
      linked_signup: '新注册关联到已封禁/静默账号',
      trial_day1: '试用账号首日用完额度',
      relay_free: '中转工具消耗免费额度',
      burst_signups: '同网段/同浏览器短时间多次注册',
      invitee_fingerprint: '被邀请人与邀请人同指纹'
    },
    actions: {
      shadow: '静默限制',
      unshadow: '解除静默',
      ban: '封禁',
      unban: '解除封禁',
      shadowTitle: '静默限制这个账号？',
      shadowMessage: '账号 {email} 的请求会被悄悄限制，用户不会看到封禁提示。请写明原因，方便以后复查。',
      reasonLabel: '原因',
      reasonPlaceholder: '例如：与已封禁账号同浏览器，注册当天用完试用额度',
      reasonRequired: '请填写静默原因',
      unshadowTitle: '解除静默限制？',
      unshadowMessage: '账号 {email} 将恢复正常请求。',
      banTitle: '封禁这个账号？',
      banMessage: '账号 {email} 将无法登录和调用 API。',
      unbanTitle: '解除封禁？',
      unbanMessage: '账号 {email} 将恢复登录和调用 API。',
      done: '已完成',
      failed: '操作失败'
    },
    events: {
      title: '风控事件',
      hint: '后台每 20 分钟检查一次新账号，命中规则会记一条事件并邮件通知运维告警收件人。',
      run: '立即检查',
      running: '检查中…',
      newEvents: '检查完成，新增 {n} 条风控事件',
      noNew: '检查完成，没有新的风控事件'
    }
  }
}
