export default {
  assistant: {
    title: 'Support assistant',
    description: 'The assistant on the homepage: answers from the site docs and the AI learning tutorials with the key you pick.',
    settingsTitle: 'Settings',
    openHome: 'Try it on the homepage',
    hint: 'Each answer is one model call billed to the key below (it shows in Usage). Set a quota on that key on the API Keys page as a hard cap on spending.',
    enabled: 'Enable the assistant',
    key: 'Key',
    keyPlaceholder: '— pick one of your keys —',
    keyHint: 'Only your own active keys in a GPT group. The server stores the key ID only, never the key.',
    keyCreate: 'Create a key',
    keyOther: "Using another admin's key (ID {id}), kept as is",
    keyProblem: "The current key can't be used: {msg}",
    noKeys: 'You have no active GPT key yet',
    model: 'Model',
    modelHint: 'Default gpt-5.6-terra (about $0.01 a question); gpt-5.6-luna costs about a tenth, with simpler answers.',
    userPerDay: 'Per signed-in user per day',
    guestPerDay: 'Per visitor IP per day',
    guestHint: 'With 0, visitors who are not signed in do not see the assistant.',
    dailyCap: 'Site-wide per day',
    dailyCapHint: 'Most answers a day for everyone together; 0 = no limit.',
    today: 'Answered today',
    pages: 'Knowledge pages',
    pagesHint: 'AI learning pages + site FAQ',
    saved: 'Saved'
  }
}
