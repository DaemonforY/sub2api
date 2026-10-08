export default {
  creator: {
    title: 'Creator center',
    description: 'Sell your own courses: buyers get your netdisk materials once they pay, and you can withdraw what you earn.',
    closed: 'Creator applications are not open yet.',
    intro: {
      title: 'Sell your courses here',
      step1: 'Apply; once approved you are a creator.',
      step2: 'Write the description, outline and price, add the netdisk link and submit for review; approved courses go on sale. Later changes to the description or price are reviewed again; the netdisk link can change at any time.',
      step3: 'Buyers get the course and your netdisk link under My courses right after paying. The platform keeps {percent}% of what each buyer paid; the rest is yours.',
      step4: 'Each sale can be withdrawn after a {days}-day settlement period (minimum ¥{min}), paid to Alipay or WeChat by hand.'
    },
    apply: {
      title: 'Apply to be a creator',
      displayName: 'Instructor name',
      displayNameHint: 'Shown on the course page',
      contact: 'Contact',
      bio: 'About you',
      plan: 'Course plan',
      planPlaceholder: 'What you will teach, for whom, about how many lessons, links to earlier work or a preview',
      submit: 'Apply',
      sent: 'Application sent; the result will show here'
    },
    rejected: 'Your last application was not approved: ',
    pending: {
      title: 'Application under review',
      body: 'The application for “{name}” is in; you can publish courses once it is approved.'
    },
    suspended: 'Your creator account is suspended and your courses are off sale. Earnings can still be withdrawn.',
    balance: {
      available: 'Withdrawable',
      frozen: 'Settling',
      paid: 'Withdrawn (incl. pending)',
      net: 'Total earned'
    },
    rules: 'Platform fee {percent}% of what buyers paid (payment fees excluded); each sale can be withdrawn {days} days after payment; refunds are taken back.',
    tabs: { courses: 'My courses', earnings: 'Earnings & withdrawals' },
    coursesHint: 'Up to {max} courses. Courses go on sale after review.',
    newCourse: 'New course',
    noCourses: 'No courses yet — click New course.',
    studentCount: '{count} sold',
    status: { draft: 'Not listed', published: 'On sale', archived: 'Off sale' },
    review: {
      draft: 'Unsubmitted changes',
      pending: 'In review',
      approved: 'Approved',
      rejected: 'Not approved'
    },
    reviewHint: {
      draft: 'Saved as a draft; buyers see it after it is submitted and approved.',
      pending: 'Usually reviewed within one working day. Editing now means submitting again.',
      approved: 'Buyers see exactly this.',
      rejected: 'Make the requested changes and submit again.'
    },
    rejectReason: 'Reviewer: ',
    liveVersion: 'Live version: “{title}” ¥{price}',
    backToCourses: 'Back to courses',
    viewPage: 'View course page',
    unlist: 'Take off sale',
    relist: 'Put back on sale',
    slugFixed: 'The course address can’t change after creation',
    shareHint: 'The platform keeps {percent}%; you get about ¥{share} per sale (less when a discount applies)',
    saveDraft: 'Save changes',
    createCourse: 'Create course',
    submit: 'Submit for review',
    draftSaved: 'Saved',
    submitted: 'Submitted for review',
    deleteConfirm: 'Delete “{title}”? This can’t be undone.',
    blocker: {
      pending: 'In review',
      noChanges: 'Nothing to review',
      noDelivery: 'Add the netdisk link below first'
    },
    deliveryTitle: 'Netdisk delivery',
    deliveryHint: 'Buyers see the netdisk link, code and unzip password under My courses (stored encrypted). No review needed; changes apply at once.',
    students: 'Buyers ({count})',
    buyer: 'Buyer',
    boughtAt: 'Bought',
    viewed: 'Views',
    refunded: 'Refunded',
    withdraw: {
      title: 'Withdraw',
      hint: 'Minimum ¥{min}; each sale can be withdrawn {days} days after payment; one request at a time.',
      pending: 'A withdrawal is being processed; you can request another after it is done.',
      belowMin: 'You can withdraw once ¥{min} is available.',
      amount: 'Amount (CNY)',
      max: 'Up to ¥{amount}',
      method: 'Paid to',
      alipay: 'Alipay',
      wechat: 'WeChat',
      account: 'Account',
      realName: 'Real name',
      note: 'Note (optional)',
      submit: 'Request withdrawal',
      requested: 'Withdrawal requested; it shows here once paid',
      cancel: 'Cancel',
      adminNote: 'Platform note: ',
      status: { pending: 'Processing', paid: 'Paid', rejected: 'Rejected', cancelled: 'Cancelled' }
    },
    sales: {
      title: 'Sales',
      hint: 'Paid amounts exclude payment fees; refunds are taken back in proportion.',
      empty: 'No sales yet',
      time: 'Time',
      course: 'Course',
      gross: 'Paid',
      fee: 'Fee',
      net: 'You earn',
      status: 'Status',
      availableAt: 'withdrawable {time}',
      statuses: {
        frozen: 'Settling',
        available: 'Settled',
        refunding: 'Refund in progress',
        refunded: 'Refunded',
        partially_refunded: 'Partly refunded'
      },
      prev: 'Previous',
      next: 'Next'
    }
  }
}
