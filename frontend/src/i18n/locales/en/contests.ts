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
      title: 'Entries',
      sortVotes: 'Most votes',
      sortNew: 'Newest',
      empty: 'No entries yet. Be the first to submit!',
      votes: '{n} votes',
      vote: 'Vote',
      voted: 'Voted',
      unvote: 'Remove vote',
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
      votesLeft: '{left} of {total} votes left',
      loginToVote: 'Log in to vote',
      loginToSubmit: 'Log in to submit',
      accountTooNew: 'Accounts younger than {hours} hours cannot vote',
      noVotesLeft: 'You have used all your votes. Remove one to vote again.',
      myEntries: 'My entries'
    },
    submit: {
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
