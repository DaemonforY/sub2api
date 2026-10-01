export default {
  sites: {
    title: 'My sites',
    description: 'Publish HTML as a website: upload one .html file or a .zip with an index.html and get a link you can share. For subscribers.',
    unavailable: 'Site hosting is not available yet.',
    subscribeFirst: 'Site hosting is for subscribers: buy any subscription plan to publish sites.',
    lapsedNotice: 'Your subscription has ended: your sites stay up for {days} more days, then pause until you renew. Publishing and updating are paused meanwhile.',
    subscribe: 'Buy a subscription',
    quota: {
      sites: 'Sites',
      free: 'Included',
      freeValue: '{used} of {total} used',
      extra: 'Additional sites',
      extraValue: '¥{price} per site per 30 days, from the balance',
      limits: 'Per site',
      limitsValue: 'up to {mb}MB and {files} files'
    },
    create: {
      title: 'Publish a new site',
      hint: 'The address is generated for you, like https://abc1234.{domain}. A zip needs an index.html as the home page and may include CSS, JS, images and fonts.',
      name: 'Name (only you see it)',
      namePlaceholder: 'e.g. Launch page',
      file: 'Page file (.html or .zip)',
      publish: 'Publish',
      publishPaid: 'Publish (¥{price} / 30 days)'
    },
    rules: 'Static pages only (no PHP or other server code). Phishing, scams, gambling, adult content, malware and infringing content are forbidden; such sites are taken down without refund. Every page shows a small "Hosted by HiveGPT · Report" badge.',
    publishing: 'Uploading…',
    published: 'Published: {url}',
    updated: 'Updated',
    renewed: 'Renewed; the site is back online',
    deleted: 'Deleted',
    copied: 'Link copied',
    wrongFile: 'Upload an .html file or a .zip archive',
    tooLarge: 'The file must be under {mb}MB',
    empty: 'No sites yet.',
    free: 'Included',
    paidUntil: 'Paid until {date}',
    meta: '{size} · {files} files · version {version} · updated {time}',
    status: { active: 'Online', disabled: 'Taken down', unpaid: 'Paused (unpaid)', lapsed: 'Paused (subscription ended)' },
    actions: { copy: 'Copy link', update: 'Update', renew: 'Renew ¥{price}', delete: 'Delete' },
    edit: { title: 'Update site', file: 'New page file (optional)', fileHint: 'Without a file only the name changes; a new file replaces the content at the same address.' },
    deleteConfirm: '{url} goes offline at once and its files are deleted for good. Delete it?',
    charges: { title: 'Charges', time: 'Time', site: 'Site', amount: 'Amount', until: 'Paid until' }
  },
  siteReport: {
    title: 'Report a site',
    description: 'Found phishing, a scam or other abuse on a site hosted by HiveGPT? Tell us and we will look into it quickly.',
    site: 'Site name (the first part of the address)',
    reason: 'Reason',
    reasons: { phishing: 'Phishing / fake login', fraud: 'Scam', gambling: 'Gambling', porn: 'Adult content', malware: 'Malware', copyright: 'Infringement', other: 'Other' },
    detail: 'Details (optional)',
    detailPlaceholder: 'e.g. imitates a bank login page and asks for passwords',
    contact: 'Contact (optional)',
    contactPlaceholder: 'Email or another way to reach you',
    submit: 'Send report',
    done: 'Thanks, we received your report and will look into it.'
  }
}
