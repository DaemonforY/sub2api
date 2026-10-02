export default {
  community: {
    title: 'Canvas community',
    description: 'Work and comment review, reports, editor’s picks and publishing restrictions for the Infinite Canvas community.',
    tabs: { pending: 'Pending', reported: 'Reported', approved: 'Public', hidden: 'Hidden', comments: 'Comments', reports: 'Reports', restricted: 'Restricted', settings: 'Settings' },
    status: { approved: 'Public', pending: 'Pending', rejected: 'Rejected', hidden: 'Hidden' },
    untitled: 'Untitled',
    featured: 'Featured',
    images: 'images',
    reports: '{count} reports',
    flags: 'Flagged',
    actions: { approve: 'Approve', reject: 'Reject', hide: 'Hide', feature: 'Feature', unfeature: 'Unfeature', ban: 'Restrict author', unban: 'Lift restriction' },
    reasonPlaceholder: 'Reason (sent to the author), e.g. copyright',
    banConfirm: 'Restrict {\'@\'}{handle} from publishing? Their page and works are hidden from others; lift it under Restricted.',
    banned: 'Author restricted',
    unrestricted: 'Lifted the restriction on {\'@\'}{handle}',
    noRestricted: 'No restricted authors',
    done: 'Done',
    empty: 'Nothing here',
    noReports: 'No reports yet',
    prev: 'Previous',
    next: 'Next',
    resolve: 'Resolved',
    dismiss: 'Dismiss',
    columns: { time: 'Time', comment: 'Comment', work: 'Work', reason: 'Reason', status: 'Status', author: 'Author', email: 'Email', works: 'Works', since: 'Restricted at' },
    reasons: { porn: 'Sexual content', violence: 'Violence', politics: 'Illegal content', copyright: 'Copyright', fraud: 'Scam or ads', spam: 'Spam', other: 'Other' },
    reportStatus: { open: 'Open', resolved: 'Resolved', dismissed: 'Dismissed' },
    commentFilters: { pending: 'Pending', reported: 'Reported', hidden: 'Hidden', all: 'All' },
    commentStatus: { approved: 'Public', pending: 'Pending', hidden: 'Hidden', deleted: 'Deleted' },
    noComments: 'No comments here',
    reportedComment: 'Reported comment',
    commentSettings: {
      title: 'Comments on works',
      enabled: 'Turn on comments',
      reviewAll: 'Every comment needs manual approval before it shows',
      hint: 'Turning comments off hides the comment section on work pages; existing comments are kept. Without full review, only comments with flagged words wait for review. New accounts may post 20 comments a day, others 200, and at most 5 a minute.'
    },
    reviewAll: 'Every new work needs manual approval before it goes public',
    cloud: {
      title: 'Canvas cloud sync space',
      hint: 'Signed-in canvas users can sync canvases, assets and workbench records to their account; uploads stop when the space is full. Lowering it does not delete files.',
      free: 'Everyone (MB)',
      subscriber: 'Subscribers (MB)'
    },
    reviewAllHint: 'When off, only works with flagged words wait for review; anything can be reported.'
  }
}
