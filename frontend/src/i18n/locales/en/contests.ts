export default {
  contests: {
    title: 'Contests',
    subtitle: 'Create with AI, join contests and win prizes',
    empty: 'No contests yet. Stay tuned!',
    loadFailed: 'Failed to load contests',
    backToList: 'Back to contests',
    phase: {
      draft: 'Draft',
      upcoming: 'Upcoming',
      submitting: 'Accepting entries',
      submitting_voting: 'Entries & voting open',
      waiting_vote: 'Voting soon',
      voting: 'Voting',
      tallying: 'Tallying',
      settled: 'Ended',
      cancelled: 'Cancelled'
    },
    schedule: {
      submission: 'Submissions',
      voting: 'Voting',
      deadline: 'Voting ends',
      endsIn: '{time} left',
      startsIn: 'Starts in {time}',
      days: '{n}d',
      hours: '{n}h',
      minutes: '{n}m'
    },
    countdown: {
      starts: 'Starts in',
      submissionEnds: 'Submissions close in',
      votingStarts: 'Voting opens in',
      votingEnds: 'Voting closes in'
    },
    stats: {
      entries: '{n} entries',
      votes: '{n} votes'
    },
    prizes: {
      title: 'Prizes',
      place: '#{n}',
      places: '#{from}–{to}',
      balance: '${amount} account balance',
      rulesNote: 'Ranked by votes at the voting deadline. Ties go to the entry that reached its count first.'
    },
    rules: 'Rules',
    entries: {
      viewWork: 'View on the canvas',
      title: 'Entries',
      sortVotes: 'Most votes',
      sortRandom: 'Shuffle',
      sortNew: 'Newest',
      empty: 'No entries yet. Be the first to submit!',
      votes: '{n} votes',
      vote: 'Vote',
      voted: 'Voted',
      unvote: 'Remove vote',
      share: 'Ask for votes',
      by: 'by {name}',
      mine: 'Mine',
      prompt: 'Prompt',
      withdraw: 'Withdraw',
      withdrawConfirm: 'Withdraw this entry? It leaves the ranking and its votes are returned to voters.',
      status: {
        pending: 'In review',
        approved: 'Approved',
        rejected: 'Rejected',
        withdrawn: 'Withdrawn',
        disqualified: 'Disqualified'
      }
    },
    viewer: {
      loginPitch: 'Sign up to get {n} votes for the entries you like',
      myVotes: 'My votes',
      votesLeft: '{left} of {total} votes left',
      loginToVote: 'Log in to vote',
      loginToSubmit: 'Log in to submit',
      accountTooNew: 'Accounts younger than {hours} hours cannot vote',
      noVotesLeft: 'You have used all your votes. Remove one to vote again.',
      myEntries: 'My entries'
    },
    submit: {
      fromUpload: 'Upload an image',
      fromWork: 'Pick a canvas work',
      pickWork: 'Pick a work you published on HiveGPT Infinite Canvas',
      noWorks: 'You haven’t published any canvas works yet',
      goCreate: 'Create on the canvas',
      workHint: 'The entry uses the work’s cover. For another image of a multi-image work, use “Enter a contest” on the work’s canvas page.',
      workPromptHint: 'The prompt follows the work: it shows only if the work shares its prompt.',
      workRequired: 'Pick a work',
      blocked: { private: 'Private', review: 'In review' },
      button: 'Submit entry',
      title: 'Submit your entry',
      image: 'Image',
      imageHint: 'PNG / JPG / WebP / GIF, up to 10MB',
      chooseImage: 'Choose image',
      titleLabel: 'Title',
      titlePlaceholder: 'Name your artwork',
      description: 'Description',
      descriptionPlaceholder: 'Optional: tell us about it',
      prompt: 'Prompt',
      promptPlaceholder: 'Optional: share the prompt you used',
      submit: 'Submit',
      submitting: 'Submitting...',
      success: 'Entry submitted',
      pendingReview: 'Entry submitted. It will appear after review.',
      imageRequired: 'Please choose an image',
      titleRequired: 'Please enter a title',
      imageTooLarge: 'Image must be 10MB or smaller',
      limit: 'Up to {n} entries per person, {left} left'
    },
    leaderboard: {
      title: 'Leaderboard',
      live: 'Live ranking',
      final: 'Final ranking',
      empty: 'No ranking yet'
    },
    hero: {
      stats: '{entries} entries · {votes} votes',
      create: 'Draw one',
      share: 'Share the contest',
      upcomingHint: 'Entries open at {time}. You can start drawing now',
      prizeNote: 'The entry with the most votes at the deadline wins, with at least {n} valid votes',
      days: 'days',
      hours: 'hrs',
      minutes: 'min'
    },
    timeline: {
      title: 'Schedule',
      open: 'Entries and votes open',
      submissionOpen: 'Entries open',
      votingOpen: 'Votes open',
      submissionClose: 'Entries close',
      votingClose: 'Votes close, ranking is final',
      award: 'Prizes after the results are posted'
    },
    howto: {
      title: 'How to take part',
      drawTitle: 'Draw a picture with AI',
      drawText: 'Create it on the HiveGPT canvas or in the image workbench. Any theme.',
      submitTitle: 'Enter it',
      submitText: 'Upload a picture or pick a work you published on the canvas, up to {n} per person, shown once approved.',
      shareTitle: 'Ask friends to vote',
      shareText: 'Every entry has a poster and a link. Signed-up friends get {n} votes each.'
    },
    noEntries: {
      title: 'No entries yet. Be the first',
      text: 'The first entries are the first ones people see.',
      upcoming: 'Entries open at {time}. You can start drawing now.',
      ideas: 'Need an idea? Pick one and start drawing',
      idea1: 'An orange cat napping on a sunny windowsill, watercolor',
      idea2: 'A street food stall on a rainy neon night, cyberpunk',
      idea3: 'A small boat in an ink-wash landscape',
      idea4: 'A shiba in a spacesuit picnicking on the moon',
      idea5: 'A corner shop from a childhood summer, film colors',
      idea6: 'A paper-cut rabbit in the snow'
    },
    shareCard: {
      schedule: 'From {start}, votes close {end}',
      how: 'Draw a picture with AI and enter it; sign up to vote ({n} votes each)',
      cta: 'Scan to join',
      pitch: '{title} is on! Draw a picture with AI to enter; the most-voted entry wins: {link}'
    },
    entry: {
      votesLabel: 'votes',
      back: 'Back to the contest',
      notFound: 'This entry does not exist or was withdrawn',
      pendingNote: 'This entry is still in review. Others can see and vote for it once it is approved. You can already share the link.',
      rank: 'Currently #{n}',
      voteCta: 'Vote for it',
      votedCta: 'Voted',
      votedAfterLogin: 'Your vote is in ✓',
      votesLeftAfter: 'You have {n} votes left. Have a look at the other entries',
      noVotesLeft: 'You have used all your votes',
      selfVote: 'You cannot vote for your own entry',
      votingClosed: 'Voting is not open now',
      shareMine: 'Ask friends to vote',
      share: 'Share with friends',
      joinTitle: 'Join the contest',
      joinPrize: 'Draw a picture with AI and enter it. {prize}',
      joinNoPrize: 'Draw a picture with AI and enter it',
      goCreate: 'Draw one',
      seeOthers: 'See other entries',
      firstPrize: 'First place wins {reward}',
      posterCaption: 'By {author} · {contest}',
      posterCta: 'Scan to vote for it',
      posterMeta: 'Sign up to vote',
      shareTitle: 'Ask for votes',
      submittedShare: 'Entry submitted! Share it and ask your friends to vote',
      submittedPending: 'Entry submitted. It can get votes once approved, and you can share the link now'
    },
    winners: {
      title: 'Winners'
    },
    toast: {
      voted: 'Vote cast',
      unvoted: 'Vote removed',
      withdrawn: 'Entry withdrawn'
    }
  }
}
