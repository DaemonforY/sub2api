export default {
  learn: {
    title: 'AI Learning',
    description: 'Example-run settings and learning stats of the /learn site.',
    runTitle: 'Running examples',
    openSite: 'Open the learning site',
    runHint: "Lessons' “Try it” boxes call this site's gateway with the learning key below; costs go to that key (see Usage). Signed-in learners get free runs a day, after which they are asked to use their own key.",
    enabled: 'Enable running examples',
    apiKey: 'Learning key',
    apiKeySet: 'Set; leave empty to keep it',
    apiKeyPlaceholder: 'Paste a key starting with sk-',
    apiKeyHint: 'Create a key under API Keys with an admin account, in the GPT pay-as-you-go group, and give it a usage limit. Stored encrypted and never shown again.',
    model: 'Model',
    modelHint: 'A model available in the key’s group; pick a cheap, fast one such as gpt-5.5.',
    freeRuns: 'Free runs per learner per day',
    dailyCap: 'Site-wide daily cap',
    dailyCapHint: 'Most runs a day across everyone; 0 = no cap.',
    stats: { learners: 'Learners', learnersToday: 'Learners today', runsToday: 'Runs today', runs7d: 'Runs (7 days)', failed7d: 'Failed (7 days)', tokens7d: 'Tokens (7 days)' },
    lesson: 'Lesson',
    completed: 'Completed',
    runs: 'Successful runs',
    noData: 'No learning data yet'
  }
}
