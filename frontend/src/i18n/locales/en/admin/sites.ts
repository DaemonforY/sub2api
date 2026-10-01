export default {
  sites: {
    title: 'Site hosting',
    description: 'Sites published by subscribers: review and take down abusive sites, handle visitor reports, set quotas and prices.',
    tabs: { sites: 'Sites', reports: 'Reports', settings: 'Settings' },
    search: 'Search site name, title or user email',
    allStatuses: 'All statuses',
    status: { active: 'Online', disabled: 'Taken down', unpaid: 'Paused (unpaid)', lapsed: 'Paused (subscription ended)' },
    columns: { site: 'Site', owner: 'User', status: 'Status', size: 'Size · files', billing: 'Billing', updated: 'Updated' },
    free: 'Included',
    paidUntil: 'Paid until {date}',
    empty: 'No sites yet',
    actions: { disable: 'Take down', enable: 'Bring back', delete: 'Delete' },
    disableHint: 'The site goes offline at once and the owner cannot update it; the reason is shown to the owner.',
    disablePlaceholder: 'Reason, e.g. imitates a bank login page',
    disabled: 'Taken down',
    enabled: 'Back online',
    deleteConfirm: 'Delete {url} and all its files for good?',
    reports: {
      time: 'Time',
      site: 'Site',
      reason: 'Reason',
      contact: 'Reporter',
      resolve: 'Resolved',
      dismiss: 'Dismiss',
      gone: 'deleted',
      empty: 'No reports',
      statuses: { open: 'Open', resolved: 'Resolved', dismissed: 'Dismissed' }
    },
    settings: {
      hint: 'Sites live at NAME.{domain}. Only subscribers can publish; the first sites are included with the subscription, more are charged every 30 days from the balance and pause when it runs out. After a subscription ends sites stay up for the grace days, then pause, and are deleted after the retention days. Changes apply within 30 seconds.',
      noDomain: 'No hosting domain is configured on the server (SITES_DOMAIN); site hosting is unavailable.',
      enabled: 'Site hosting available',
      max_per_user: 'Max sites per user',
      free_per_user: 'Sites included with a subscription',
      extra_price: 'Price per extra site (per 30 days)',
      max_mb: 'Max size per site (MB)',
      max_files: 'Max files per site',
      grace_days: 'Days kept after a subscription ends',
      retention_days: 'Days kept while paused (then deleted)',
      saved: 'Saved'
    }
  }
}
