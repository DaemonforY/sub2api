export default {
  supportAssistant: {
    open: 'Help',
    title: '{site} assistant',
    subtitle: 'Answers from the site docs and the AI learning tutorials',
    close: 'Close',
    clear: 'Clear chat',
    greeting: "Hi, I'm the {site} assistant. Ask me about signing up, API keys, top-ups and subscriptions, connecting coding tools, or the AI learning tutorials.",
    greetingTools: ' I can also check your own balance, API keys and recent errors.',
    suggestions: {
      diagnose: 'Why did my key fail recently?',
      q1: 'How do I create an API key?',
      q2: 'How do I connect Codex to HiveGPT?',
      q3: 'How do I top up? Subscription or pay-as-you-go?',
      q4: 'Where do I get the WeChat AppID?'
    },
    placeholder: 'Ask a question — Enter to send, Shift+Enter for a new line',
    send: 'Send',
    stop: 'Stop',
    thinking: 'Looking it up…',
    checked: 'Checked: {list}',
    sources: 'Related pages',
    left: '{n} questions left today',
    none: 'No questions left today',
    guestHint: 'Sign in for {n} a day',
    login: 'Sign in',
    disclaimer: 'AI answers are for reference; prices and rules on the pages apply. Never send keys or passwords here.',
    disclaimerTools: 'AI answers are for reference. When needed, a summary of your account (never the key itself) is looked up and sent to the model service; chats are kept for 30 days. Never send keys or passwords here.',
    tooLong: 'Up to {n} characters',
    error: 'Something went wrong: {msg}'
  }
}
