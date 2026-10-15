export default {
  userInsights: {
    title: 'User insights',
    description: 'Sign-ups, usage and payments from a risk angle: who is draining free quota, which accounts belong to one person, who calls through relay tools.',
    tabs: {
      overview: 'Overview',
      profile: 'User profile',
      events: 'Risk events'
    },
    range: '{n} days',
    refresh: 'Refresh',
    since: 'Window: since {since}, {days} days; admin accounts excluded',
    loadFailed: 'Failed to load',
    empty: 'No data',
    total: 'Total',
    segments: {
      inactive: 'Inactive',
      trial: 'Trial',
      paying: 'Paying',
      churned: 'Churned',
      hint: {
        inactive: 'Never called the API',
        trial: 'Used the API, never paid',
        paying: 'Paid, active in the last 14 days (or paid in the last 30)',
        churned: 'Paid before, quiet since'
      }
    },
    signups: {
      title: 'Daily sign-ups',
      hint: 'By sign-up day; each account counts in the segment it is in now.'
    },
    spend: {
      title: 'Daily spend and revenue',
      hint: 'Free spend = usage of accounts that never paid (USD); total usage covers every account; revenue counts real payments only (CNY).',
      free: 'Free spend ($)',
      used: 'Total usage ($)',
      paid: 'Revenue (¥)'
    },
    clients: {
      title: 'Client mix',
      hint: 'Grouped by the request User-Agent.'
    },
    clientClass: {
      codex_desktop: 'Codex desktop',
      codex_cli: 'Codex CLI',
      claude_code: 'Claude Code',
      cherry_studio: 'Cherry Studio',
      relay: 'Relay tool',
      browser: 'Browser',
      sdk: 'SDK / script',
      other: 'Other'
    },
    clusters: {
      title: 'Linked account clusters',
      hint: 'Several accounts signed up from one browser or one IP range; free spend counts only the members that never paid.',
      kind: {
        visitor: 'Same browser',
        ip_prefix: 'Same IP range'
      },
      members: '{n} accounts'
    },
    relayUsers: {
      title: 'Relay tool users',
      hint: 'Accounts calling through relay / resale tools (AxonHub, new-api, …) or bare HTTP libraries, by spend.'
    },
    badge: {
      disabled: 'Banned',
      shadowed: 'Shadowed',
      paid: 'Paid',
      free: 'Trial',
      admin: 'Admin'
    },
    col: {
      date: 'Date',
      class: 'Client',
      requests: 'Requests',
      users: 'Users',
      cost: 'Spend ($)',
      user: 'User',
      userAgent: 'User-Agent',
      kind: 'Kind',
      key: 'Key',
      members: 'Accounts',
      freeUsd: 'Free spend ($)',
      paidCny: 'Paid (¥)',
      spendUsd: 'Window spend ($)',
      lastSeen: 'Last seen',
      count: 'Count',
      ip: 'IP',
      prefix: 'IP range',
      group: 'Group',
      model: 'Model',
      time: 'Time',
      rule: 'Rule',
      detail: 'Detail',
      orderType: 'Order type',
      paymentType: 'Method',
      amount: 'Amount',
      payAmount: 'Paid',
      status: 'Status',
      createdAt: 'Created',
      paidAt: 'Paid at',
      startsAt: 'Starts',
      expiresAt: 'Expires',
      dailyUsage: 'Today / daily limit ($)',
      via: 'Linked by'
    },
    profile: {
      searchPlaceholder: 'User ID or email',
      search: 'Search',
      prompt: 'Enter a user ID or email to see the profile; clicking a user in any list also opens it here.',
      id: 'ID',
      status: 'Status',
      role: 'Role',
      createdAt: 'Signed up',
      signupSource: 'Sign-up method',
      balance: 'Balance',
      riskScore: 'Risk score',
      riskLevel: {
        low: 'Low risk',
        medium: 'Medium risk',
        high: 'High risk'
      },
      noReasons: 'No risk signals found',
      flags: 'Flags',
      banned: 'Banned',
      shadowed: 'Silently restricted',
      shadowReason: 'Reason: {reason}',
      shadowSource: {
        auto: 'Sign-up rules',
        admin: 'Admin'
      },
      shadowMeta: '{source} · {time}',
      noFlags: 'None',
      signup: 'Sign-up',
      attribution: {
        source: 'Source',
        medium: 'Medium',
        campaign: 'Campaign',
        aff_code: 'Invite code',
        referrer_host: 'Referrer',
        landing_path: 'Landing page'
      },
      visitorId: 'Browser ID (visitor_id)',
      signupPrefix: 'Sign-up IP range',
      trialGranted: 'Trial granted',
      yes: 'Yes',
      no: 'No',
      webPrefixes: 'Web IP ranges',
      apiIps: 'API IPs',
      clients: 'API clients',
      groups: 'Groups used',
      models: 'Models used',
      spend: 'Spend and payments',
      spendTotal: 'Total spend',
      spendFreeSub: 'Trial subscription',
      spendPaidSub: 'Paid subscriptions',
      spendBalance: 'Balance',
      paidCny: 'Lifetime paid',
      everPaid: 'Ever paid',
      spendWindow: 'Usage covers the last 90 days',
      payments: 'Payments',
      subscriptions: 'Subscriptions',
      invites: 'Invites',
      inviteUp: 'Inviter chain',
      inviteDown: 'Invitees',
      noInviter: 'No inviter',
      noInvitees: 'Invited nobody',
      linked: 'Linked accounts',
      events: 'Risk events'
    },
    via: {
      visitor: 'Same browser',
      ip_prefix: 'Same IP range',
      api_ip: 'Same API IP',
      user_agent: 'Same relay UA',
      inviter: 'Inviter',
      invitee: 'Invitee'
    },
    rules: {
      linked_signup: 'New sign-up linked to a banned / shadowed account',
      trial_day1: 'Trial quota used up on day one',
      relay_free: 'Relay tool spending free quota',
      burst_signups: 'Burst of sign-ups from one IP range / browser',
      invitee_fingerprint: 'Invitee shares the inviter\'s fingerprint',
      trial_same_ip: 'Same IP /24 as another trial',
      shadow_paid: 'Order paid while restricted'
    },
    actions: {
      shadow: 'Silently restrict',
      unshadow: 'Lift restriction',
      ban: 'Ban',
      unban: 'Unban',
      shadowTitle: 'Silently restrict this account?',
      shadowMessage: 'Requests from {email} will be quietly limited; the user sees no ban notice. Write down why, for later review.',
      reasonLabel: 'Reason',
      reasonPlaceholder: 'e.g. same browser as a banned account, used up the trial on day one',
      reasonRequired: 'Please enter a reason',
      unshadowTitle: 'Lift the silent restriction?',
      unshadowMessage: 'Requests from {email} will be served normally again.',
      banTitle: 'Ban this account?',
      banMessage: '{email} will no longer be able to sign in or call the API.',
      unbanTitle: 'Unban this account?',
      unbanMessage: '{email} will be able to sign in and call the API again.',
      done: 'Done',
      failed: 'Action failed'
    },
    events: {
      title: 'Risk events',
      hint: 'New accounts are checked every 20 minutes; each rule hit is recorded here and mailed to the ops alert recipients.',
      run: 'Check now',
      running: 'Checking…',
      newEvents: 'Check finished: {n} new risk events',
      noNew: 'Check finished: no new risk events'
    }
  }
}
