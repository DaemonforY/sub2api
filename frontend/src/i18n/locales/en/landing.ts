export default {
  batchImageGuide: {
    title: 'Batch Image Generation',
    description: 'Submit multiple prompts in one job and download the generated images when complete'
  },
  // Home Page
  home: {
    viewOnGithub: 'View on GitHub',
    viewDocs: 'View Documentation',
    docs: 'Docs',
    switchToLight: 'Switch to Light Mode',
    switchToDark: 'Switch to Dark Mode',
    dashboard: 'Dashboard',
    login: 'Login',
    register: 'Sign up free',
    signupBonus: 'New accounts get ${amount} of free credit for images and chat',
    trust: {
      metered: { title: 'Pay for what you use', desc: 'Every call is listed under Usage with its model, tokens and cost.' },
      failures: { title: 'Errors are free', desc: "Requests that fail upstream or time out aren't charged; a stream cut off midway is billed only for what was generated." },
      payment: { title: 'Credited automatically', desc: 'Pay online and your balance or subscription is active right after, no manual step.' },
      support: { title: 'Help when you need it', desc: 'The assistant at the bottom right answers account and payment questions any time.', descContact: 'Ask the assistant at the bottom right any time, or contact us: {info}' },
      total: '{n} requests served so far'
    },
    pricing: {
      title: 'Pricing: subscriptions go further',
      subtitle: 'Pay as you go at ¥1 = ${rate} of usage; if you use it daily, a subscription gives a daily allowance for far less.',
      recommended: 'Recommended',
      days: '{n} days',
      saving: 'Worth ¥{value} of pay-as-you-go if used fully, {percent}% less',
      daily: '${usd} of usage a day',
      perDay: ' (about {n} requests)',
      weekly: 'Up to ${usd} a week',
      monthly: 'Up to ${usd} a month',
      group: 'Group: {group}',
      subscribe: 'Subscribe',
      paygTitle: 'Pay as you go',
      paygDesc: 'For occasional use',
      paygFrom: 'minimum',
      paygRate: '¥1 = ${rate} of usage; top up again when it runs out',
      paygAvg: 'Average request about ${usd}',
      paygBonus: '${amount} free credit when you sign up',
      topUp: 'Top up',
      tryFree: 'Try it free',
      note: 'Usage is valued at each model\'s official price; a subscription does not touch your balance within its allowance.',
      more: 'Full pricing and usage estimator'
    },
    getStarted: 'Get Started',
    goToDashboard: 'Go to Dashboard',
    // User-focused value proposition
    heroSubtitle: 'One Key, All AI Models',
    heroDescription: 'OpenAI-compatible API: one key for GPT, Codex, gpt-image-2 and more, pay only for what you use, failed requests are free',
    tags: {
      subscriptionToApi: 'Subscription to API',
      stickySession: 'Session Persistence',
      realtimeBilling: 'Pay As You Go'
    },
    // Pain points section
    painPoints: {
      title: 'Sound Familiar?',
      items: {
        expensive: {
          title: 'High Subscription Costs',
          desc: 'Paying for multiple AI subscriptions that add up every month'
        },
        complex: {
          title: 'Account Chaos',
          desc: 'Managing scattered accounts and API keys across different platforms'
        },
        unstable: {
          title: 'Service Interruptions',
          desc: 'Single accounts hitting rate limits and disrupting your workflow'
        },
        noControl: {
          title: 'No Usage Control',
          desc: "Can't track where your money goes or limit team member usage"
        }
      }
    },
    // Solutions section
    solutions: {
      title: 'We Solve These Problems',
      subtitle: 'Three simple steps to stress-free AI access'
    },
    features: {
      unifiedGateway: 'One-Click Access',
      unifiedGatewayDesc: 'Get a single API key to call all connected AI models. No separate applications needed.',
      multiAccount: 'Always Reliable',
      multiAccountDesc: 'Smart routing across multiple upstream accounts with automatic failover. Say goodbye to errors.',
      balanceQuota: 'Pay What You Use',
      balanceQuotaDesc: 'Usage-based billing with quota limits. Full visibility into team consumption.'
    },
    // Comparison section
    comparison: {
      title: 'Why Choose Us?',
      headers: {
        feature: 'Comparison',
        official: 'Official Subscriptions',
        us: 'Our Platform'
      },
      items: {
        pricing: {
          feature: 'Pricing',
          official: 'Fixed monthly fee, pay even if unused',
          us: 'Pay only for what you use'
        },
        models: {
          feature: 'Model Selection',
          official: 'Single provider only',
          us: 'Switch between models freely'
        },
        management: {
          feature: 'Account Management',
          official: 'Manage each service separately',
          us: 'Unified key, one dashboard'
        },
        stability: {
          feature: 'Stability',
          official: 'Single account rate limits',
          us: 'Multi-account pool, auto-failover'
        },
        control: {
          feature: 'Usage Control',
          official: 'Not available',
          us: 'Quotas & detailed analytics'
        }
      }
    },
    providers: {
      title: 'Supported AI Models',
      description: 'One API, Multiple Choices',
      supported: 'Supported',
      pending: 'Pending',
      soon: 'Soon',
      claude: 'Claude',
      gemini: 'Gemini',
      antigravity: 'Antigravity',
      more: 'More'
    },
    // CTA section
    cta: {
      title: 'Ready to Get Started?',
      description: 'Sign up now and get free trial credits to experience seamless AI access',
      button: 'Sign Up Free'
    },
    // Infinite canvas (cross-site, deployed on a subdomain)
    canvas: {
      navLabel: 'Infinite Canvas',
      card: {
        title: 'HiveGPT Infinite Canvas · AI creative studio',
        desc: 'Generate, edit and remix images or videos on an infinite canvas with your key from this site, wiring nodes into a full creative workflow.',
        tags: 'Text to Image,Image to Image,Reference Edit,Video,Node Workflow',
        cta: 'Open canvas'
      }
    },
    // Learning resources (cross-site)
    learn: {
      badge: 'Learn',
      title: 'Learning Resources',
      subtitle: 'Learn AI application development, then practice with a single API key',
      navLabel: 'Learn AI',
      card: {
        title: 'AI Application Development Guide',
        desc: 'Learn by doing: AI app development and AI art lessons whose code runs right in the page, with free daily runs after signing in.',
        tags: 'LLM Basics,AI Agent,RAG,System Design,Interview',
        cta: 'Read for free'
      }
    },
    // Home v2 (inspired by mxai.cn: create-from-the-hero, scenario entry points, showcase wall, onboarding path, FAQ)
    // Home page, 2026-10 redesign: one slogan, what the sign-up credit buys, prompt first then sign up
    v3: {
      nav: {
        draw: 'Draw',
        write: 'Write',
        code: 'Code',
        works: 'Gallery',
        pricing: 'Pricing',
        signupTag: '+${amount}'
      },
      hero: {
        news: 'GPT-6 Astra is here',
        newsTail: 'Pay per use',
        titleLead: "You're the boss.",
        titleHighlight: 'AI does the work.',
        subtitleLead: 'Posters, articles, code fixes, merged spreadsheets — ',
        subtitleEm: 'say it in one sentence and it gets going.',
        subtitleTail: ' No coding and no foreign credit card needed.',
        offerTitle: 'Sign up and get ${amount} of credit',
        offerBoth: 'About {images} images, or about {chats} GPT-6 chats',
        offerImages: 'About {images} images',
        offerChats: 'About {chats} GPT-6 chats',
        offerGeneric: 'Works for images and chat, pay for what you use',
        cta: 'Sign up free, get ${amount}',
        ctaNoBonus: 'Sign up free',
        ctaSecondary: 'See how it works',
        ctaDashboard: 'Open console',
        trustPay: 'WeChat Pay, credited instantly',
        trustMetered: 'Pay for what you use, every charge listed',
        trustTotal: '{n} requests served'
      },
      demo: {
        tabs: { draw: 'Draw', write: 'Write', code: 'Code' },
        you: 'You',
        placeholder: 'Describe a picture, e.g. a moonlit classical courtyard in ink-wash style',
        generate: 'Generate',
        generating: 'Generating…',
        more: 'More works',
        draw: {
          prompt: 'A ginger shorthair cat lying on the sill of an old wooden window, looking out at a rainy street. Dusk, warm lamp light inside, cold blue outside, raindrops on the glass, watercolour illustration.',
          caption: "A member's work · gpt-image-2 · prompt excerpt",
          status: 'Done · 2K',
          alt: "A member's work: a ginger cat at a rainy window"
        },
        write: {
          prompt: 'Lay this draft out as a WeChat article, add a cover image and make the title catchier.',
          caption: 'WeChat article editor · real screen',
          status: 'Laid out',
          alt: 'The WeChat article editor',
          go: 'Open editor'
        },
        code: {
          prompt: 'Merge the 30 Excel files in the "Reports" folder on my desktop into one sheet and drop duplicate rows.',
          caption: 'Codex · illustration',
          status: 'Done',
          go: 'Read the guide',
          cmd: 'merge the 30 Excel files in "Reports" into one sheet',
          line1: 'read 30 files, 4,812 rows',
          line2: 'unify headers: date / store / category / sales',
          line3: 'drop 26 duplicate rows',
          line4: 'write summary.xlsx',
          done: 'done',
          mock: 'illustration'
        }
      },
      jobs: {
        eyebrow: 'What it does for you',
        title: 'Say what you want. It handles the rest.',
        subtitle: 'One account for images, writing and code; one balance, charged for whatever you use.',
        say: 'You say: ',
        draw: {
          title: 'Draw',
          say: 'A documentary photo of an old Chinese street, then and now: the old days on the left, today on the right',
          caption: "A member's work",
          go: 'Draw one',
          more: 'Edit · cut out · extend'
        },
        write: {
          title: 'Write',
          say: 'Lay this draft out as a WeChat article, add a cover and send it to my drafts',
          caption: 'WeChat article editor',
          go: 'Article editor',
          more: 'Writing style test'
        },
        code: {
          title: 'Code',
          say: 'Merge the 30 Excel files on my desktop into one sheet',
          cmd: 'merge the Excel files in "Reports"',
          line1: '30 files, 4,812 rows',
          line2: 'headers unified, 26 duplicates dropped',
          done: 'wrote summary.xlsx',
          go: 'Set up Codex in 5 minutes',
          more: 'Let AI sort files and sheets'
        }
      },
      sheet: {
        title: 'Sign up to generate this picture',
        prompt: 'Your prompt: ',
        promptHint: 'After signing up you go straight to the canvas with it.',
        perkBonus: '${amount} of free credit',
        perkImages: ', about {images} images',
        perkModels: 'GPT-6, gpt-image-2 and Codex in one account',
        register: 'Sign up free',
        wechat: 'Sign up with WeChat',
        login: 'Have an account? Log in',
        close: 'Close'
      },
      mobileBar: {
        title: 'Sign up, get ${amount}',
        images: 'About {images} free images',
        generic: 'For images and chat',
        cta: 'Sign up'
      },
      finalCta: {
        bonus: '${amount} on sign-up · WeChat Pay · pay for what you use',
        noBonus: 'WeChat Pay · pay for what you use'
      },
      footer: {
        editor: 'WeChat article editor',
        style: 'Writing style test'
      },
      motion: {
        hud: {
          input: 'Your one sentence',
          inputSpec: '→ 9 tokens',
          model: 'GPT-6 Astra',
          modelSpec: 'Draw · Write · Code'
        },
        teardown: {
          eyebrow: 'Take it apart / 05 layers',
          title: 'How one sentence becomes a picture',
          subtitle: 'From the words you type to the finished image, all five steps run on {site}.',
          timeline: 'Exploded view',
          layers: {
            prompt: 'Your sentence',
            promptSpec: 'Chinese · {n} characters',
            tokens: 'Cut into tokens',
            tokensSpec: '≈ 70 tokens',
            modelSpec: 'Works out what to draw',
            imageSpec: 'Draws in 2K',
            result: 'Done',
            resultSpec: 'About {price} each',
            resultSpecPlain: 'The finished picture'
          },
          phases: { p0: 'Stacked', p1: 'Taking it apart', p2: 'Understanding you', p3: 'Drawing', p4: 'Done', p5: 'Back together' }
        },
        stats: {
          eyebrow: 'Data in motion',
          titlePre: 'We have processed ',
          titlePost: ' tokens for our users',
          titlePlain: 'Every request flows through here',
          requests: 'requests',
          minTopup: '¥1',
          payg: 'minimum top-up, pay for what you use',
          chat: 'Chat',
          image: 'Images',
          code: 'Code',
          legend: 'Each light is one request (illustration)',
          source: 'From our billing records; cached tokens included'
        }
      },
      firstImage: {
        title: 'All set — draw your first picture',
        account: 'Your account is ready',
        balance: '${amount} credit added',
        key: 'A key that can draw is ready for you',
        keyFailed: 'No key yet; you can make one when you connect on the canvas',
        prompt: 'Your prompt: ',
        next: 'On the canvas, click Generate, then "Connect with {site}", and authorize in the small window that opens.',
        go: 'Open the canvas →',
        auto: 'Opening in {n} s',
        dashboard: 'Go to the console instead'
      }
    },
    v2: {
      nav: {
        start: 'Start creating',
        explore: 'Explore',
        tools: 'Tools'
      },
      hero: {
        eyebrow: '{site} · AI creation and API in one place',
        titleLead: 'One key for ',
        titleHighlight: 'popular AI models',
        subtitle: 'Images, video and coding assistants in one place. Pay as you go or subscribe.',
        chips: {
          models: 'Many models',
          billing: 'Pay as you go',
          stable: 'High availability'
        },
        promptPlaceholder: 'Describe what you want to draw, e.g. a moonlit classical courtyard, ink painting style, rich detail',
        examplesLabel: 'Try:',
        examples: 'Cyberpunk city at night,Watercolor panda in a bamboo forest,E-commerce shot: white sneakers on white,Minimal flat coffee shop logo',
        generate: 'Generate in canvas',
        getKey: 'Get an API key',
        hint: 'Paste your API key in the canvas to generate. The gateway URL is filled in for you.'
      },
      why: {
        title: 'Why {site}',
        subtitle: 'Turn scattered AI subscriptions into one stable, easy, controllable entry point'
      },
      scenarios: {
        title: 'Built for everyday AI work',
        subtitle: 'From writing code to making images and videos, all with one account',
        cta: 'Use now',
        coding: {
          title: 'AI coding assistants',
          badge: 'Dev',
          points: 'One-click setup for Codex / OpenCode,Copy-ready env vars and config files,Precise per-token billing'
        },
        image: {
          title: 'AI images',
          badge: 'Popular',
          points: 'Text-to-image and image-to-image,Free-form infinite canvas,Huge built-in prompt library'
        },
        video: {
          title: 'AI video',
          badge: 'New',
          points: 'Text-to-video and image-to-video,First/last frame and reference modes,Chain full workflows on the canvas'
        },
        batch: {
          title: 'Batch images',
          badge: 'Fast',
          points: 'Submit many prompts at once,Queued and generated automatically,Download all results together'
        },
        plaza: {
          title: 'Model plaza',
          badge: 'Open',
          points: 'See available models and rates,Understand pricing per group,Pick the right model'
        },
        contest: {
          title: 'Contests',
          badge: 'Prizes',
          points: 'Submit AI art to win prizes,Transparent community voting,Ranking frozen at the deadline'
        },
        scenes: {
          title: 'What AI can do for you',
          badge: 'Use cases',
          points: 'Ideas by role: office work / sales / teachers / students / creators,Copy-ready prompts,Let AI batch-process your files'
        }
      },
      community: {
        title: 'All made by people on HiveGPT',
        subtitle: 'Made with AI on HiveGPT Infinite Canvas. Like one? Remix it in a click.',
        more: 'Explore more',
        siteLabel: 'Web page',
        videoLabel: 'Video',
        aiLabel: 'AI-generated'
      },
      showcase: {
        title: 'Featured works',
        subtitle: 'Top-voted entries from community contests',
        more: 'See all contests',
        votes: '{n} votes'
      },
      steps: {
        title: 'Get started in three steps',
        subtitle: 'From sign-up to your first image in about a minute',
        register: {
          title: 'Create an account',
          desc: 'Sign up with email and open the console'
        },
        key: {
          title: 'Top up and create a key',
          desc: 'Add credit as needed; one key works for every enabled model'
        },
        use: {
          title: 'Start creating',
          desc: 'Draw on the canvas, or plug the key into Codex / OpenCode'
        },
        registerCta: 'Sign up free',
        keyCta: 'Manage API keys',
        partnerCta: 'Or buy on {site}',
        useCta: 'Open canvas'
      },
      support: {
        title: 'Help & support',
        subtitle: 'Guides, learning resources and contact info',
        docs: {
          title: 'Documentation',
          desc: 'Setup guides, client configs and FAQs'
        },
        learn: {
          title: 'AI learning hub',
          desc: 'Articles on LLMs, agents and RAG to learn while you build'
        },
        contact: {
          title: 'Contact us',
          desc: 'Questions about accounts, payments or partnerships',
          label: 'Contact: {info}'
        },
        partner: {
          title: 'Buy on {site}',
          desc: '{site} is our partner site; you can also sign up and buy API keys there.',
          cta: 'Buy on {site}'
        },
        open: 'Open'
      },
      faq: {
        title: 'FAQ',
        subtitle: 'You might find your answer here',
        items: {
          noCode: {
            q: 'Can I use it without coding?',
            a: 'Yes. Drawing and WeChat articles take a few clicks in the browser; only coding tools like Codex need installing, and the learning site walks you through it.'
          },
          bonus: {
            q: 'How far does the free credit go?',
            a: 'It is charged by actual use, and every image or question is listed under Usage. The "about N images" on this page is estimated from the last 30 days of real charges. Top up from ¥1 when it runs out.'
          },
          vsChatgpt: {
            q: 'How is this different from a ChatGPT subscription?',
            a: 'No foreign credit card: pay with WeChat. No monthly plan needed: pay for what you use. Images, chat and Codex share one balance.'
          },
          canvas: {
            q: 'How do the canvas and API keys relate?',
            a: 'The canvas is a free creative UI; generation runs on a key in your account. When you click Generate, choose "Connect with {site}" and authorize; no copying keys by hand.'
          },
          privacy: {
            q: 'Is my data safe?',
            a: 'Canvas artworks, history and keys stay in your own browser by default. The gateway only forwards requests and records the usage needed for billing.'
          },
          contact: {
            q: 'What if I run into problems?',
            a: 'Ask the assistant in the bottom-right corner or read the guides on the learning site. If that does not help, reach us through the contact info on this page.'
          }
        }
      },
      finalCta: {
        title: 'Ready to get more done with AI?',
        subtitle: 'Create an account and use images, video and coding assistants with one key',
        primary: 'Get started',
        secondary: 'Try the canvas first'
      },
      footer: {
        desc: 'Images, writing and code: say it in one sentence and AI gets it done.',
        product: 'Product',
        support: 'Support',
        canvas: 'Infinite Canvas',
        contests: 'Contests',
        plaza: 'Model plaza',
        batch: 'Batch images',
        docs: 'Documentation',
        learn: 'AI learning hub',
        scenes: 'AI use cases',
        partner: 'Partner site: {site}',
        keys: 'API keys'
      }
    },
    footer: {
      allRightsReserved: 'All rights reserved.'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key Usage',
    subtitle: 'Enter your API Key to view real-time spending and usage status',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: 'Query',
    querying: 'Querying...',
    privacyNote: 'Your Key is processed locally in the browser and will not be stored',
    dateRange: 'Date Range:',
    dateRangeToday: 'Today',
    dateRange7d: '7 Days',
    dateRange30d: '30 Days',
    dateRange90d: '90 Days',
    dateRangeCustom: 'Custom',
    apply: 'Apply',
    used: 'Used',
    detailInfo: 'Detail Information',
    tokenStats: 'Token Statistics',
    dailyDetail: 'Daily Detail',
    modelStats: 'Model Usage Statistics',
    // Table headers
    date: 'Date',
    model: 'Model',
    requests: 'Requests',
    inputTokens: 'Input Tokens',
    outputTokens: 'Output Tokens',
    cacheCreationTokens: 'Cache Creation',
    cacheReadTokens: 'Cache Read',
    cacheWriteTokens: 'Cache Write',
    totalTokens: 'Total Tokens',
    cost: 'Cost',
    // Status
    quotaMode: 'Key Quota Mode',
    walletBalance: 'Wallet Balance',
    // Ring card titles
    totalQuota: 'Total Quota',
    limit5h: '5-Hour Limit',
    limitDaily: 'Daily Limit',
    limit7d: '7-Day Limit',
    limitWeekly: 'Weekly Limit',
    limitMonthly: 'Monthly Limit',
    // Detail rows
    remainingQuota: 'Remaining Quota',
    expiresAt: 'Expires At',
    todayExpires: '(expires today)',
    daysLeft: '({days} days)',
    usedQuota: 'Used Quota',
    resetNow: 'Resetting soon',
    subscriptionType: 'Subscription Type',
    subscriptionExpires: 'Subscription Expires',
    // Usage stat cells
    todayRequests: 'Today Requests',
    todayInputTokens: 'Today Input',
    todayOutputTokens: 'Today Output',
    todayTokens: 'Today Tokens',
    todayCacheCreation: 'Today Cache Creation',
    todayCacheRead: 'Today Cache Read',
    todayCost: 'Today Cost',
    rpmTpm: 'RPM / TPM',
    totalRequests: 'Total Requests',
    totalInputTokens: 'Total Input',
    totalOutputTokens: 'Total Output',
    totalTokensLabel: 'Total Tokens',
    totalCacheCreation: 'Total Cache Creation',
    totalCacheRead: 'Total Cache Read',
    totalCost: 'Total Cost',
    avgDuration: 'Avg Duration',
    // Messages
    enterApiKey: 'Please enter an API Key',
    querySuccess: 'Query successful',
    queryFailed: 'Query failed',
    queryFailedRetry: 'Query failed, please try again later',
    noDailyUsage: 'No daily usage data',
  },

  // Setup Wizard
  setup: {
    title: 'Sub2API Setup',
    description: 'Configure your Sub2API instance',
    database: {
      title: 'Database Configuration',
      description: 'Connect to your PostgreSQL database',
      host: 'Host',
      port: 'Port',
      username: 'Username',
      password: 'Password',
      databaseName: 'Database Name',
      sslMode: 'SSL Mode',
      passwordPlaceholder: 'Password',
      ssl: {
        disable: 'Disable',
        require: 'Require',
        verifyCa: 'Verify CA',
        verifyFull: 'Verify Full'
      }
    },
    redis: {
      title: 'Redis Configuration',
      description: 'Connect to your Redis server',
      host: 'Host',
      port: 'Port',
      username: 'Username (optional)',
      password: 'Password (optional)',
      database: 'Database',
      usernamePlaceholder: 'Leave empty for default user',
      passwordPlaceholder: 'Password',
      enableTls: 'Enable TLS',
      enableTlsHint: 'Use TLS when connecting to Redis (public CA certs)'
    },
    admin: {
      title: 'Admin Account',
      description: 'Create your administrator account',
      email: 'Email',
      password: 'Password',
      confirmPassword: 'Confirm Password',
      passwordPlaceholder: 'Min 8 characters',
      confirmPasswordPlaceholder: 'Confirm password',
      passwordMismatch: 'Passwords do not match'
    },
    ready: {
      title: 'Ready to Install',
      description: 'Review your configuration and complete setup',
      database: 'Database',
      redis: 'Redis',
      adminEmail: 'Admin Email'
    },
    status: {
      testing: 'Testing...',
      success: 'Connection Successful',
      testConnection: 'Test Connection',
      installing: 'Installing...',
      completeInstallation: 'Complete Installation',
      completed: 'Installation completed!',
      redirecting: 'Redirecting to login page...',
      restarting: 'Service is restarting, please wait...',
      timeout: 'Service restart is taking longer than expected. Please refresh the page manually.'
    }
  },

  // Common
}
