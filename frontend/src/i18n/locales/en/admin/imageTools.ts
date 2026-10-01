export default {
  imageTools: {
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
