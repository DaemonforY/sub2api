export default {
  pricing: {
    navLabel: 'Pricing',
    guideNav: 'Setup guides',
    compareLink: 'Subscription or pay as you go? Compare →',
    title: 'Subscription or pay as you go?',
    subtitle: 'Same account, same tools — two ways to pay. See which fits how much you use.',
    loadFailed: 'Could not load pricing, please try again later',
    buy: 'Top up or subscribe',
    guide: 'Setup guides',
    payg: {
      title: 'Pay as you go',
      badge: 'Flexible',
      unit: 'of usage',
      p1: 'Top up what you need, from ¥{min}',
      p2: 'Charged at standard model prices, no daily cap',
      p3: 'Every request and its cost is listed under Usage',
      groups: 'Groups: ',
      fit: 'For occasional or uneven use, or to try things out'
    },
    sub: {
      title: 'Subscription',
      badge: 'Better value for daily use',
      from: 'from / {days} days',
      p1: 'A fixed price with daily, weekly and monthly usage allowances',
      p2: 'For heavy, steady use it works out cheaper than pay as you go',
      p3: 'Allowances reset each period and do not roll over; the plan ends when it expires',
      fit: 'For coding in Codex and similar tools every day'
    },
    plans: {
      title: 'Plans on sale',
      plan: 'Plan',
      price: 'Price',
      perDay: 'Per day',
      limits: 'Allowance',
      value: 'Worth',
      days: '{n} days',
      daily: '${v} a day',
      weekly: '${v} a week',
      monthly: '${v} a month',
      noLimit: 'No limit',
      valueText: 'Up to ${usd} of usage in {days} days — about ¥{cny} pay as you go',
      note: 'Allowances are in standard USD model prices; daily, weekly and monthly limits all apply, whichever runs out first. Prices on the purchase page are final.'
    },
    estimate: {
      title: 'Estimate from your usage',
      basis: 'Based on the site-wide average cost of a GPT request over the last 30 days: about ${cost} ({n} requests)',
      perDay: 'Requests per day',
      days: 'Days a month',
      month: 'month',
      overLimit: 'This usage exceeds the plan allowance',
      pickPayg: 'At this usage, pay as you go is cheaper.',
      pickPlan: 'At this usage, "{name}" is cheaper.',
      disclaimer: 'A rough estimate: real costs depend on the model, context length and reasoning effort. Try pay as you go for a few days and check Usage for real numbers.'
    },
    faq: {
      title: 'FAQ',
      q1: 'How do I use a subscription?',
      q1a: 'When creating an API key, pick the group of your plan. Requests with that key use the subscription allowance, not your balance.',
      q2: 'What if the allowance runs out?',
      q2a: 'Daily, weekly and monthly allowances reset on their own. If you can\'t wait, create a pay-as-you-go key and continue on balance.',
      q3: 'Can I use both?',
      q3a: 'Yes. An account can have several keys in different groups — e.g. a subscription for daily coding and pay as you go for occasional big jobs.',
      q4: 'What is a "USD allowance"?',
      q4a: 'Models are billed at standard USD prices per million tokens; balance and subscription allowances are both deducted on that basis, and every request is listed under Usage.',
      q5: 'Discounts for students and teachers?',
      q5a: 'Verify your school email on the Invite page to get a discounted subscription price, as shown on the purchase page.',
      q6: 'Can I try first?',
      q6a: 'Yes. Top up from ¥{min} and try with a pay-as-you-go key; subscribe once you know you\'ll use it daily.'
    }
  }
}
