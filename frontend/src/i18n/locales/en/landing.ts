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
    getStarted: 'Get Started',
    goToDashboard: 'Go to Dashboard',
    // User-focused value proposition
    heroSubtitle: 'One Key, All AI Models',
    heroDescription: 'No need to manage multiple subscriptions. Access Claude, GPT, Gemini and more with a single API key',
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
        desc: '33 in-depth articles covering LLM basics, Agents, RAG and AI system design, plus interview questions and a learning roadmap. Free to read.',
        tags: 'LLM Basics,AI Agent,RAG,System Design,Interview',
        cta: 'Read for free'
      }
    },
    // Home v2 (inspired by mxai.cn: create-from-the-hero, scenario entry points, showcase wall, onboarding path, FAQ)
    v2: {
      nav: {
        start: 'Start creating'
      },
      hero: {
        eyebrow: '{site} · AI creation and API in one place',
        titleLead: 'One key for ',
        titleHighlight: 'top-tier AI',
        subtitle: 'Images, video and coding assistants in one place. Pay as you go, always on.',
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
          points: 'One-click setup for Claude Code / Codex / OpenCode,Copy-ready env vars and config files,Precise per-token billing'
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
        }
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
          desc: 'Draw on the canvas, or plug the key into Claude Code / Codex'
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
          what: {
            q: 'What is {site}?',
            a: '{site} is an AI service hub. With one account and one API key you can make images and videos on the canvas and plug coding tools like Claude Code and Codex into it.'
          },
          models: {
            q: 'Which models and tools are supported?',
            a: 'See the model plaza and the supported list on this page; more are added over time. Coding tools include Claude Code, Codex CLI and OpenCode, with copy-ready configs under "Use key" in the console.'
          },
          billing: {
            q: 'How does billing work?',
            a: 'You pay for what you use. Each key can have a spending cap, and usage details are always visible in the console.'
          },
          canvas: {
            q: 'How do the canvas and API keys relate?',
            a: 'The canvas is a free creative UI; the actual generation runs through your API key on this site. Opening the canvas from here fills in the gateway URL, so you only paste your key.'
          },
          privacy: {
            q: 'Is my data safe?',
            a: 'Canvas artworks, history and keys stay in your own browser by default. The gateway only forwards requests and records the usage needed for billing.'
          },
          contact: {
            q: 'What if I run into problems?',
            a: 'Check the documentation first. If that does not help, reach us through the contact info on this page.'
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
        desc: 'One key for top-tier AI: images, video and coding assistants in one place.',
        product: 'Product',
        support: 'Support',
        canvas: 'Infinite Canvas',
        contests: 'Contests',
        plaza: 'Model plaza',
        batch: 'Batch images',
        docs: 'Documentation',
        learn: 'AI learning hub',
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
