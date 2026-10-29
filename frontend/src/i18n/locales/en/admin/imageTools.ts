export default {
  imageTools: {
    aspect: {
      title: 'Image aspect hint',
      hint: 'The upstream treats size as a hint and often returns another orientation. When on, the gateway appends an orientation and ratio sentence to gpt-image prompts (e.g. "【画幅】portrait, 2:3"), for all users and clients; prompts that already have 【画幅】 are left alone. Below: requests by the size asked for, and how many came back in another orientation. Takes effect within a minute.',
      enabled: 'Append the aspect hint',
      day: 'Last 24 hours',
      week: 'Last 7 days',
      none: 'No image requests with a size',
      asked: { portrait: 'Portrait asked', landscape: 'Landscape asked', square: 'Square asked' },
      wrong: '{n} wrong orientation ({pct}%)',
      saved: 'Saved; takes effect within a minute'
    },
    title: 'Image tools',
    description: 'AI background removal and AI upscaling for the canvas: set prices, the daily free runs for subscribers and the on/off switch, and see every run and charge. Only successful runs are charged.',
    settings: {
      title: 'Prices and free runs',
      hint: 'Prices are taken from the balance. Subscribers use their daily free runs first (both tools combined), then pay per run; users without a subscription always pay. Changes apply within 30 seconds.',
      enabled: 'Image tools available',
      priceRemoveBg: 'AI cut-out (per image)',
      priceUpscale: 'AI upscale (per image)',
      freeDaily: 'Free runs per day for subscribers',
      save: 'Save',
      saved: 'Saved',
      notConfigured: 'The server has no image service configured (IMAGE_TOOLS_BASE_URL); the tools cannot run even when switched on.'
    },
    stats: {
      today: 'Today',
      week: 'Last 7 days',
      month: 'This month',
      runs: 'Runs',
      free: 'Free',
      revenue: 'Revenue',
      users: 'Users'
    },
    tools: { remove_bg: 'AI cut-out', upscale: 'AI upscale' },
    filters: { search: 'Search user email', allTools: 'All tools', from: 'From', to: 'To' },
    columns: { time: 'Time', user: 'User', tool: 'Tool', charge: 'Charge', size: 'Image size', duration: 'Time taken', key: 'API key' },
    free: 'Free',
    empty: 'No runs yet'
  }
}
