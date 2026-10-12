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
    register: '免费注册',
    signupBonus: '新用户注册即送 ${amount} 体验额度，画图、对话都能用',
    trust: {
      metered: { title: '用多少扣多少', desc: '每次调用的模型、Token 和费用都记在「使用记录」里，随时可查。' },
      failures: { title: '出错不扣费', desc: '上游报错、超时的请求不收钱；流式输出中途断开的，只按已生成的部分计费。' },
      payment: { title: '付款自动到账', desc: '在线支付，付款完成后余额或订阅自动到账，不用等人工处理。' },
      support: { title: '有问题随时问', desc: '首页右下角的智能客服随时可以问，账号、充值问题都能答。', descContact: '首页右下角智能客服随时可问；也可以直接联系：{info}' },
      total: '已累计为用户处理 {n} 次请求'
    },
    pricing: {
      title: '价格：订阅更划算',
      subtitle: '按量计费 ¥1 = ${rate} 额度，用多少扣多少；常用的话订阅按天给额度，用满比按量便宜得多。',
      recommended: '推荐',
      days: '{n} 天',
      saving: '用满相当于按量 ¥{value}，省 {percent}%',
      daily: '每天 ${usd} 额度',
      perDay: '（约 {n} 次请求）',
      weekly: '每周最多 ${usd}',
      monthly: '每月最多 ${usd}',
      group: '适用分组：{group}',
      subscribe: '立即订阅',
      paygTitle: '按量计费',
      paygDesc: '偶尔用一用，用多少付多少',
      paygFrom: '起充',
      paygRate: '¥1 = ${rate} 额度，余额用完再充',
      paygAvg: '全站平均每次请求约 ${usd}',
      paygBonus: '注册就送 ${amount} 体验额度',
      topUp: '去充值',
      tryFree: '免费注册试试',
      note: '额度按各模型官方价格折算；订阅在额度内不扣余额。',
      more: '查看完整价格和用量估算'
    },
    getStarted: '立即开始',
    goToDashboard: '进入控制台',
    // 新增：面向用户的价值主张
    heroSubtitle: '一个密钥，畅用多个 AI 模型',
    heroDescription: '兼容 OpenAI 接口，一个 API Key 调用 GPT、Codex、gpt-image-2 等模型，按实际用量计费，出错不扣费',
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
        desc: '边学边做：AI 应用开发和 AI 绘画课程，每节课的代码都能在页面里直接运行，登录后每天有免费次数。',
        tags: '大模型基础,AI Agent,RAG,系统设计,面试题',
        cta: '免费阅读'
      }
    },
    // 首页 v2（参考 mxai.cn 的首屏即创作、场景入口、案例墙、上手路径、FAQ 结构）
    // 首页 2026-10 改版：一句 slogan、送多少说成具体的东西、先写提示词再注册
    v3: {
      nav: {
        draw: '画图',
        write: '写作',
        code: '编程',
        works: '作品',
        pricing: '价格',
        signupTag: '送${amount}'
      },
      hero: {
        news: 'GPT-6 Astra 已上线',
        newsTail: '微信付款，按次计费',
        titleLead: '你当老板，',
        titleHighlight: 'AI 打工。',
        subtitleLead: '画海报、写公众号、改代码、合并表格——',
        subtitleEm: '用中文说一句，它就开干。',
        subtitleTail: '不用会编程，不用海外信用卡。',
        offerTitle: '注册就送 ${amount} 体验额度',
        offerBoth: '够画约 {images} 张图，或和 GPT-6 对话约 {chats} 次',
        offerImages: '够画约 {images} 张图',
        offerChats: '够和 GPT-6 对话约 {chats} 次',
        offerGeneric: '画图、对话都能用，用多少扣多少',
        cta: '免费注册，领 ${amount}',
        ctaNoBonus: '免费注册',
        ctaSecondary: '拆开看看它怎么干活',
        ctaDashboard: '进入控制台',
        trustPay: '微信支付，自动到账',
        trustMetered: '用多少扣多少，每笔可查',
        trustTotal: '已为用户处理 {n} 次请求'
      },
      demo: {
        tabs: { draw: '画图', write: '写作', code: '编程' },
        you: '你说',
        placeholder: '描述你想画的画面，例如：月光下的古风庭院，水墨风格',
        generate: '生成',
        generating: '正在生成…',
        more: '看更多作品',
        draw: {
          prompt: '一只橘色短毛猫趴在老式木窗的窗台上，侧脸望向窗外下雨的街道。傍晚，室内暖黄台灯，窗外冷蓝，玻璃上挂着雨滴，水彩插画风格。',
          caption: '真实用户作品 · gpt-image-2 · 提示词节选',
          status: '已出图 · 2K',
          alt: '用户作品：雨天窗台上的橘猫'
        },
        write: {
          prompt: '把这篇稿子排成公众号样式，配一张封面，标题改得更抓人一点。',
          caption: '公众号排版 · 真实界面',
          status: '已排好版',
          alt: '公众号排版界面',
          go: '去排版'
        },
        code: {
          prompt: '把桌面「报表」文件夹里 30 个 Excel 合并成一张总表，去掉重复的行。',
          caption: 'Codex · 示意',
          status: '已完成',
          go: '看教程',
          cmd: '把「报表」里的 30 个 Excel 合并成一张总表',
          line1: '读取 30 个文件，共 4,812 行',
          line2: '统一表头：日期 / 门店 / 品类 / 销售额',
          line3: '去掉 26 行重复记录',
          line4: '写入 汇总.xlsx',
          done: '完成',
          mock: '示意'
        }
      },
      jobs: {
        eyebrow: '它能替你干的活',
        title: '你只管说要什么，剩下的它来。',
        subtitle: '一个账号，画图、写作、编程都能用；同一份余额，用哪个扣哪个。',
        say: '你说：',
        draw: {
          title: '画图',
          say: '拍一张中国老街「今昔对照」的纪实照片，左边旧时代，右边新时代',
          caption: '真实用户作品',
          go: '去画一张',
          more: '改图 · 抠图 · 扩图'
        },
        write: {
          title: '写作',
          say: '把这篇稿子排成公众号样式，配好封面，一键发到草稿箱',
          caption: '公众号排版',
          go: '公众号排版',
          more: '测写作风格'
        },
        code: {
          title: '编程',
          say: '把桌面上 30 个 Excel 合并成一张总表',
          cmd: '合并「报表」里的 Excel',
          line1: '30 个文件，4,812 行',
          line2: '统一表头，去重 26 行',
          done: '已写入 汇总.xlsx',
          go: '5 分钟接入 Codex',
          more: '让 AI 整理文件和表格'
        }
      },
      sheet: {
        title: '注册就能生成这张图',
        prompt: '你的提示词：',
        promptHint: '注册完会带着这句话去画布出图。',
        perkBonus: '送 ${amount} 体验额度',
        perkImages: '，约 {images} 张图',
        perkModels: 'GPT-6、gpt-image-2、Codex 一个账号全用上',
        register: '免费注册',
        wechat: '微信一键注册',
        login: '已有账号？登录',
        close: '关闭'
      },
      mobileBar: {
        title: '注册送 ${amount}',
        images: '先免费画约 {images} 张图',
        generic: '画图、对话都能用',
        cta: '免费注册'
      },
      finalCta: {
        bonus: '注册送 ${amount} · 微信付款 · 用多少扣多少',
        noBonus: '微信付款 · 用多少扣多少'
      },
      footer: {
        editor: '公众号排版',
        style: '测写作风格'
      },
      motion: {
        hud: {
          input: '你的一句话',
          inputSpec: '→ 9 tokens',
          model: 'GPT-6 Astra',
          modelSpec: '画图 · 写作 · 编程'
        },
        teardown: {
          eyebrow: '拆开看看 / 05 layers',
          title: '一句话，是怎么变成一张图的',
          subtitle: '从你打出的字，到画好的图，中间这五步都在 {site} 里跑完。',
          timeline: '拆解过程',
          layers: {
            prompt: '你说的一句话',
            promptSpec: '中文 · {n} 个字',
            tokens: '切成 token',
            tokensSpec: '≈ 70 tokens',
            modelSpec: '读懂你要画什么',
            imageSpec: '2K 出图',
            result: '成品',
            resultSpec: '约 {price} 一张',
            resultSpecPlain: '画好的图'
          },
          phases: { p0: '合在一起', p1: '拆开', p2: '读懂你要什么', p3: '出图', p4: '成品', p5: '合成' }
        },
        stats: {
          eyebrow: '数据在流动',
          titlePre: '已经帮用户处理了 ',
          titlePost: ' 个 token',
          titlePlain: '每一次请求，都在这里流动',
          requests: '次请求',
          minTopup: '1 元',
          payg: '起充，用多少扣多少',
          chat: '对话',
          image: '画图',
          code: '编程',
          legend: '每一次亮起，代表一个请求（示意）',
          source: '数字来自本站计费记录，含缓存读取的 token'
        }
      },
      firstImage: {
        title: '准备好了，去画第一张图',
        account: '账号注册好了',
        balance: '余额 ${amount} 已到账',
        key: '已为你准备好能画图的密钥',
        keyFailed: '密钥还没建好，到画布连接时可以再建一个',
        prompt: '你的提示词：',
        next: '到画布后点「开始生成」，再点「用 {site} 账号一键连接」，在弹出的小窗口里点「授权」，就能出图。',
        go: '去画布出图 →',
        auto: '{n} 秒后自动打开',
        dashboard: '先去控制台看看'
      }
    },
    v2: {
      nav: {
        start: '开始创作',
        explore: '发现',
        tools: '工具'
      },
      hero: {
        eyebrow: '{site} · AI 创作与 API 一站式平台',
        titleLead: '一个 Key，',
        titleHighlight: '畅用主流 AI',
        subtitle: '画图、视频、编程助手一站搞定。按量计费，也可以订阅。',
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
          points: 'Codex / OpenCode 一键配置,复制即用的环境变量与配置文件,按 Token 精确计费'
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
        },
        scenes: {
          title: 'AI 能帮你做什么',
          badge: '场景玩法',
          points: '办公、销售、老师、学生、自媒体等按身份找用法,可直接复制的提示词,让 AI 帮你批量处理文件'
        }
      },
      community: {
        title: '这些，都是用户在 HiveGPT 上做的',
        subtitle: '大家在 HiveGPT 无限画布上用 AI 创作的作品，喜欢的可以一键做同款',
        more: '去发现页看更多',
        siteLabel: '网页',
        videoLabel: '视频',
        aiLabel: 'AI 生成'
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
          desc: '在无限画布里画图，或把 Key 配置到 Codex / OpenCode'
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
          noCode: {
            q: '不会编程能用吗？',
            a: '能。画图、写公众号都在网页里点几下就行；只有接 Codex 这类编程工具时才要装软件，学习站有一步一步的教程。'
          },
          bonus: {
            q: '送的体验额度能用多少？',
            a: '按实际用量扣，画一张图、问一次各扣多少都记在「使用记录」里；首页写的「够画约 N 张图」是按最近 30 天的实际扣费估的。用完再充，1 元起充。'
          },
          vsChatgpt: {
            q: '和直接买 ChatGPT 会员有什么区别？',
            a: '不用海外信用卡，微信就能付；不用包月，用多少扣多少；画图、对话和 Codex 用的是同一份余额。'
          },
          canvas: {
            q: '画布和 API Key 是什么关系？',
            a: '画布是免费的创作界面，出图用的是你账号里的 Key。在画布点「开始生成」时选「用 {site} 账号一键连接」，授权一下就连好了，不用手动复制 Key。'
          },
          privacy: {
            q: '我的数据安全吗？',
            a: '画布里的作品、历史和 Key 默认只保存在你自己的浏览器中。网关只转发请求并记录计费所需的用量信息。'
          },
          contact: {
            q: '遇到问题怎么办？',
            a: '先问右下角的智能客服，或者看学习站的教程；还没解决可以通过页面上的联系方式找到我们。'
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
        desc: '画图、写作、编程，用中文说一句，AI 帮你做完。',
        product: '产品',
        support: '支持',
        canvas: '无限画布',
        contests: '创作比赛',
        plaza: '模型广场',
        batch: '批量生图',
        docs: '使用文档',
        learn: 'AI 学习站',
        scenes: 'AI 场景玩法',
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
