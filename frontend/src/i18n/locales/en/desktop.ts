export default {
  desktopLogin: {
    title: 'Sign in to HiveGPT Desktop',
    subtitle: 'Let AI work on files and spreadsheets on your computer',
    asking: 'The computer “{device}” is asking to sign in to your account.',
    explain: 'If you confirm, this computer gets its own key that draws on your account (trial or subscription first). You can disable it any time under API Keys.',
    notYou: 'If you did not just open this QR code on your own computer, choose “Not me”.',
    phishing: 'Only confirm the computer in front of you. If someone sent you this QR code (a reward, an “official check”, “help me sign in”…), choose “Not me”: confirming lets their computer spend your balance and allowance.',
    network: 'Computer’s network: {network} (asked {ago})',
    sameNetwork: 'This computer is on the same network as your phone.',
    otherNetwork: 'This computer is not on your phone’s network. That also happens when the phone uses mobile data and the computer Wi-Fi — but if the computer is not next to you, choose “Not me”.',
    confirmNearby: 'This computer is in front of me and I am the one signing in',
    justNow: 'just now',
    minutesAgo: '{n} min ago',
    unknownNetwork: 'unknown',
    approve: 'Confirm sign-in',
    reject: 'Not me',
    approved: 'Confirmed — go back to your computer',
    approvedHint: 'The computer finishes signing in within a few seconds. You can close this page.',
    approvedNotMe: 'Wasn’t you? Disable “桌面助手 · {device}” under API Keys right away.',
    goKeys: 'Go to API Keys',
    rejected: 'Sign-in rejected',
    rejectedHint: 'The computer will not be signed in to your account.',
    invalid: 'This sign-in QR code has expired or does not exist. Refresh it on the computer and scan again.',
    handled: 'This sign-in request has already been handled. Check your computer.',
    loadFailed: 'Failed to load. Refresh the page and try again.'
  }
}
