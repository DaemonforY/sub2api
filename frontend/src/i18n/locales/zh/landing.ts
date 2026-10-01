export default {
  batchImageGuide: {
    title: '图片批量生成',
    description: '一次提交多条提示词，任务完成后可统一下载图片结果'
  },
  // Home Page
  home: {
    viewOnGithub: '在 GitHub 上查看',
    viewDocs: '查看文档',
    docs: '文档',
    switchToLight: '切换到浅色模式',
    switchToDark: '切换到深色模式',
    dashboard: '控制台',
    login: '登录',
    getStarted: '立即开始',
    goToDashboard: '进入控制台',
    // 新增：面向用户的价值主张
    heroSubtitle: '一个密钥，畅用多个 AI 模型',
    heroDescription: '无需管理多个订阅账号，一站式接入 Claude、GPT、Gemini 等主流 AI 服务',
    tags: {
      subscriptionToApi: '订阅转 API',
      stickySession: '会话保持',
      realtimeBilling: '按量计费'
    },
    // 用户痛点区块
    painPoints: {
      title: '你是否也遇到这些问题？',
      items: {
        expensive: {
          title: '订阅费用高',
          desc: '每个 AI 服务都要单独订阅，每月支出越来越多'
        },
        complex: {
          title: '多账号难管理',
          desc: '不同平台的账号、密钥分散各处，管理起来很麻烦'
        },
        unstable: {
          title: '服务不稳定',
          desc: '单一账号容易触发限制，影响正常使用'
        },
        noControl: {
          title: '用量无法控制',
          desc: '不知道钱花在哪了，也无法限制团队成员的使用'
        }
      }
    },
    // 解决方案区块
    solutions: {
      title: '我们帮你解决',
      subtitle: '简单三步，开始省心使用 AI'
    },
    features: {
      unifiedGateway: '一键接入',
      unifiedGatewayDesc: '获取一个 API 密钥，即可调用所有已接入的 AI 模型，无需分别申请。',
      multiAccount: '稳定可靠',
      multiAccountDesc: '智能调度多个上游账号，自动切换和负载均衡，告别频繁报错。',
      balanceQuota: '用多少付多少',
      balanceQuotaDesc: '按实际使用量计费，支持设置配额上限，团队用量一目了然。'
    },
    // 优势对比
    comparison: {
      title: '为什么选择我们？',
      headers: {
        feature: '对比项',
        official: '官方订阅',
        us: '本平台'
      },
      items: {
        pricing: {
          feature: '付费方式',
          official: '固定月费，用不完也付',
          us: '按量付费，用多少付多少'
        },
        models: {
          feature: '模型选择',
          official: '单一服务商',
          us: '多模型随意切换'
        },
        management: {
          feature: '账号管理',
          official: '每个服务单独管理',
          us: '统一密钥，一站管理'
        },
        stability: {
          feature: '服务稳定性',
          official: '单账号易触发限制',
          us: '多账号池，自动切换'
        },
        control: {
          feature: '用量控制',
          official: '无法限制',
          us: '可设配额、查明细'
        }
      }
    },
    providers: {
      title: '已支持的 AI 模型',
      description: '一个 API，多种选择',
      supported: '已支持',
      pending: '待支持',
      soon: '即将推出',
      claude: 'Claude',
      gemini: 'Gemini',
      antigravity: 'Antigravity',
      more: '更多'
    },
    // CTA 区块
    cta: {
      title: '准备好开始了吗？',
      description: '注册即可获得免费试用额度，体验一站式 AI 服务',
      button: '免费注册'
    },
    // 无限画布（互推站点，子域名部署）
    canvas: {
      navLabel: '无限画布',
      card: {
        title: 'HiveGPT 无限画布 · AI 创作工作台',
        desc: '在无限画布上用本站 Key 直接文生图、图生图、参考图编辑与视频生成，节点连线编排完整创作流程。',
        tags: '文生图,图生图,参考图编辑,视频生成,画布编排',
        cta: '打开画布'
      }
    },
    // 配套学习资源（互推站点）
    learn: {
      badge: '配套学习',
      title: '配套学习资源',
      subtitle: '一边学 AI 应用开发，一边用一个 Key 动手实践',
      navLabel: 'AI 学习',
      card: {
        title: 'AI 应用开发知识体系',
        desc: '33 篇系统文章：大模型基础、Agent、RAG、AI 系统设计，配套高频面试题与学习路线，免费阅读。',
        tags: '大模型基础,AI Agent,RAG,系统设计,面试题',
        cta: '免费阅读'
      }
    },
    // 首页 v2（参考 mxai.cn 的首屏即创作、场景入口、案例墙、上手路径、FAQ 结构）
    v2: {
      nav: {
        start: '开始创作'
      },
      hero: {
        eyebrow: '{site} · AI 创作与 API 一站式平台',
        titleLead: '一个 Key，',
        titleHighlight: '畅用顶级 AI',
        subtitle: '画图、视频、编程助手一站搞定。按量计费，稳定不掉线。',
        chips: {
          models: '多模型聚合',
          billing: '按量计费',
          stable: '稳定高可用'
        },
        promptPlaceholder: '描述你想画的画面，例如：月光下的古风庭院，水墨风格，细节丰富',
        examplesLabel: '试试：',
        examples: '赛博朋克风格的城市夜景,水彩风格的熊猫在竹林里,电商白底主图：一双白色运动鞋,极简扁平风格的咖啡店 logo',
        generate: '去画布生成',
        getKey: '获取 API Key',
        hint: '在画布中粘贴你的 API Key 即可出图，网关地址已自动填好'
      },
      why: {
        title: '为什么选择 {site}',
        subtitle: '把分散的 AI 订阅，变成一个稳定、好用、可控的入口'
      },
      scenarios: {
        title: '覆盖高频 AI 使用场景',
        subtitle: '从写代码到出图出片，一个账号全部搞定',
        cta: '立即使用',
        coding: {
          title: 'AI 编程助手',
          badge: '开发者',
          points: 'Claude Code / Codex / OpenCode 一键配置,复制即用的环境变量与配置文件,按 Token 精确计费'
        },
        image: {
          title: 'AI 绘画',
          badge: '热门',
          points: '文生图、图生图、参考图编辑,无限画布自由编排,内置海量提示词库'
        },
        video: {
          title: 'AI 视频',
          badge: '新',
          points: '文生视频与图生视频,首尾帧与参考模式,在画布中串联完整流程'
        },
        batch: {
          title: '批量生图',
          badge: '效率',
          points: '一次提交多条提示词,后台排队自动生成,结果统一打包下载'
        },
        plaza: {
          title: '模型广场',
          badge: '透明',
          points: '查看可用模型与倍率,按分组了解价格,选择最适合的模型'
        },
        contest: {
          title: '创作比赛',
          badge: '有奖',
          points: '投稿 AI 作品赢取奖励,社区投票公开透明,截止时刻冻结排名'
        }
      },
      showcase: {
        title: '精选作品',
        subtitle: '来自社区创作比赛的高票作品',
        more: '查看全部活动',
        votes: '{n} 票'
      },
      steps: {
        title: '三步开始使用',
        subtitle: '从注册到出图，最快一分钟',
        register: {
          title: '注册账号',
          desc: '邮箱注册，登录后进入控制台'
        },
        key: {
          title: '充值并创建 API Key',
          desc: '按需充值，一个 Key 通用所有已开放的模型'
        },
        use: {
          title: '开始创作',
          desc: '在无限画布里画图，或把 Key 配置到 Claude Code / Codex'
        },
        registerCta: '免费注册',
        keyCta: '管理 API Key',
        partnerCta: '也可在 {site} 购买',
        useCta: '打开画布'
      },
      support: {
        title: '帮助与支持',
        subtitle: '上手指南、学习资源和联系方式都在这里',
        docs: {
          title: '使用文档',
          desc: '接入教程、客户端配置和常见问题'
        },
        learn: {
          title: 'AI 学习站',
          desc: '大模型、Agent、RAG 系统文章，边学边用'
        },
        contact: {
          title: '联系我们',
          desc: '账号、充值、合作问题可直接联系',
          label: '联系方式：{info}'
        },
        partner: {
          title: '{site} 购买 Key',
          desc: '{site} 是我们的合作站点，也可以在那里注册并购买 API Key。',
          cta: '去 {site} 购买'
        },
        open: '查看'
      },
      faq: {
        title: '常见问题',
        subtitle: '先看看这些，也许就能找到答案',
        items: {
          what: {
            q: '{site} 是什么？',
            a: '{site} 是一个 AI 服务聚合平台。你只需要一个账号和一个 API Key，就能在画布里画图、生成视频，也能接入 Claude Code、Codex 等编程工具。'
          },
          models: {
            q: '支持哪些模型和工具？',
            a: '已开放的模型以「模型广场」和首页的支持列表为准，并会持续增加。编程工具支持 Claude Code、Codex CLI、OpenCode 等，控制台「使用密钥」里有一键复制的配置。'
          },
          billing: {
            q: '如何收费？',
            a: '按实际用量计费，用多少扣多少。你可以为每个 Key 设置额度上限，随时在控制台查看用量明细。'
          },
          canvas: {
            q: '画布和 API Key 是什么关系？',
            a: '画布是一个免费的创作界面，真正的生图、生视频由你的 API Key 调用本站完成。从本站跳转到画布时网关地址会自动填好，只需粘贴 Key。'
          },
          privacy: {
            q: '我的数据安全吗？',
            a: '画布里的作品、历史和 Key 默认只保存在你自己的浏览器中。网关只转发请求并记录计费所需的用量信息。'
          },
          contact: {
            q: '遇到问题怎么办？',
            a: '先查看使用文档，仍未解决可以通过页面上的联系方式找到我们。'
          }
        }
      },
      finalCta: {
        title: '准备好用 AI 提升效率了吗？',
        subtitle: '注册账号，一个 Key 畅用画图、视频和编程助手',
        primary: '立即开始',
        secondary: '先去画布看看'
      },
      footer: {
        desc: '一个 Key 畅用顶级 AI：画图、视频、编程助手一站搞定。',
        product: '产品',
        support: '支持',
        canvas: '无限画布',
        contests: '创作比赛',
        plaza: '模型广场',
        batch: '批量生图',
        docs: '使用文档',
        learn: 'AI 学习站',
        partner: '合作站点：{site}',
        keys: 'API Key 管理'
      }
    },
    footer: {
      allRightsReserved: '保留所有权利。'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key 用量查询',
    subtitle: '输入您的 API Key 以查看实时消费金额与使用状态',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: '查询',
    querying: '查询中...',
    privacyNote: '您的 Key 仅在浏览器本地处理，不会被存储',
    dateRange: '统计范围:',
    dateRangeToday: '今日',
    dateRange7d: '7 天',
    dateRange30d: '30 天',
    dateRange90d: '90 天',
    dateRangeCustom: '自定义',
    apply: '应用',
    used: '已使用',
    detailInfo: '详细信息',
    tokenStats: 'Token 统计',
    dailyDetail: '按日明细',
    modelStats: '模型用量统计',
    // Table headers
    date: '日期',
    model: '模型',
    requests: '请求数',
    inputTokens: '输入 Tokens',
    outputTokens: '输出 Tokens',
    cacheCreationTokens: '缓存创建',
    cacheReadTokens: '缓存读取',
    cacheWriteTokens: '缓存写入',
    totalTokens: '总 Tokens',
    cost: '费用',
    // Status
    quotaMode: 'Key 限额模式',
    walletBalance: '钱包余额',
    // Ring card titles
    totalQuota: '总额度',
    limit5h: '5 小时限额',
    limitDaily: '日限额',
    limit7d: '7 天限额',
    limitWeekly: '周限额',
    limitMonthly: '月限额',
    // Detail rows
    remainingQuota: '剩余额度',
    expiresAt: '过期时间',
    todayExpires: '(今日到期)',
    daysLeft: '({days} 天)',
    usedQuota: '已用额度',
    resetNow: '即将重置',
    subscriptionType: '订阅类型',
    subscriptionExpires: '订阅到期',
    // Usage stat cells
    todayRequests: '今日请求',
    todayInputTokens: '今日输入',
    todayOutputTokens: '今日输出',
    todayTokens: '今日 Tokens',
    todayCacheCreation: '今日缓存创建',
    todayCacheRead: '今日缓存读取',
    todayCost: '今日费用',
    rpmTpm: 'RPM / TPM',
    totalRequests: '累计请求',
    totalInputTokens: '累计输入',
    totalOutputTokens: '累计输出',
    totalTokensLabel: '累计 Tokens',
    totalCacheCreation: '累计缓存创建',
    totalCacheRead: '累计缓存读取',
    totalCost: '累计费用',
    avgDuration: '平均耗时',
    // Messages
    enterApiKey: '请输入 API Key',
    querySuccess: '查询成功',
    queryFailed: '查询失败',
    queryFailedRetry: '查询失败，请稍后重试',
    noDailyUsage: '暂无按日用量数据',
  },

  // Setup Wizard
  setup: {
    title: 'Sub2API 安装向导',
    description: '配置您的 Sub2API 实例',
    database: {
      title: '数据库配置',
      description: '连接到您的 PostgreSQL 数据库',
      host: '主机',
      port: '端口',
      username: '用户名',
      password: '密码',
      databaseName: '数据库名称',
      sslMode: 'SSL 模式',
      passwordPlaceholder: '密码',
      ssl: {
        disable: '禁用',
        require: '要求',
        verifyCa: '验证 CA',
        verifyFull: '完全验证'
      }
    },
    redis: {
      title: 'Redis 配置',
      description: '连接到您的 Redis 服务器',
      host: '主机',
      port: '端口',
      username: '用户名（可选）',
      password: '密码（可选）',
      database: '数据库',
      usernamePlaceholder: '默认用户留空',
      passwordPlaceholder: '密码',
      enableTls: '启用 TLS',
      enableTlsHint: '连接 Redis 时使用 TLS（公共 CA 证书）'
    },
    admin: {
      title: '管理员账户',
      description: '创建您的管理员账户',
      email: '邮箱',
      password: '密码',
      confirmPassword: '确认密码',
      passwordPlaceholder: '至少 8 个字符',
      confirmPasswordPlaceholder: '确认密码',
      passwordMismatch: '密码不匹配'
    },
    ready: {
      title: '准备安装',
      description: '检查您的配置并完成安装',
      database: '数据库',
      redis: 'Redis',
      adminEmail: '管理员邮箱'
    },
    status: {
      testing: '测试中...',
      success: '连接成功',
      testConnection: '测试连接',
      installing: '安装中...',
      completeInstallation: '完成安装',
      completed: '安装完成！',
      redirecting: '正在跳转到登录页面...',
      restarting: '服务正在重启，请稍候...',
      timeout: '服务重启时间超出预期，请手动刷新页面。'
    }
  },

  // Common
}
