export default {
  games: {
    title: 'Games',
    subtitle: 'A few minutes a round, then turn your result into a share card.',
    play: 'Play',
    minutes: 'About {n} min',
    soon: 'More games on the way',
    soonDesc: 'What would you like to play? Tell the assistant; the most wanted ones come first.',
    ask: 'Tell the assistant',
    askDraft: 'I would like a game where ',
    share: { brand: 'HiveGPT Games', title: 'AI games, a few minutes a round', cta: 'Scan to play' },
    list: {
      abti: { title: 'ABTI · AI worker personality test', desc: '31 questions over 15 dimensions: what kind of worker are you in the AI era?', tag: 'Personality' },
      style: { title: 'Find your writing style', desc: 'Paste a few of your articles and see your rhythm, catchphrases and AI flavour.', tag: 'Writing' }
    },
    abti: {
      title: 'ABTI · AI worker personality test',
      badge: 'Game · about 3 min',
      subtitle: 'Now that AI is here, do you hand out the work, slack off, or proofread the AI? 31 questions to find your AI worker type. (Questions are in Chinese.)',
      facts: { questions: '31 questions', dims: '15 dimensions', types: '16 types + hidden ones' },
      start: 'Start',
      restart: 'Take it again',
      lastResult: 'See your last result ({code} {name})',
      friend: 'Your friend got',
      friendAsk: 'Which one are you?',
      disclaimer: 'Just for fun. We made up the questions and the types; this is not a psychological assessment.',
      aiArt: 'Illustrations are AI-generated.',
      quiz: {
        progress: 'Question {n} / {total}',
        back: 'Previous',
        quit: 'Quit',
        extra: 'Bonus question'
      },
      result: {
        kicker: { normal: 'Your AI worker type', hidden: 'Hidden type unlocked', fallback: 'Fallback type' },
        match: '{n}% match · {exact}/15 dimensions exact',
        hiddenMatch: 'Hidden type · closest regular type {code} {name} ({n}%)',
        fallbackMatch: 'The best regular match was only {n}% ({code} {name})',
        dims: 'Your 15 dimensions',
        level: { L: 'Low', M: 'Mid', H: 'High' },
        close: 'Types most like you',
        all: 'All types',
        hiddenSlot: 'Hidden',
        tip: 'AI tip',
        share: 'Make a share card',
        restart: 'Take it again',
        hall: 'All games',
        cta: {
          image: 'Have AI draw your type as an avatar',
          code: 'See how to connect Codex',
          write: 'Open the AI article editor',
          learn: 'Go to the AI learning site',
          plaza: 'See the available models',
          invite: 'Get my invite link'
        },
        signup: 'Sign up free, get ${n} credit',
        signupPlain: 'Sign up free',
        imagePrompt: 'Draw a cute chibi office worker whose persona is "{name}": {intro} Sticker style, thick outlines, soft lavender and honey-yellow colours, clean light background, no text.'
      },
      share: {
        brand: 'ABTI · AI worker personality test',
        introTitle: 'What kind of AI worker are you?',
        title: 'I am {code} "{name}"',
        match: '{n}% match',
        cta: 'Scan to find your AI worker type'
      }
    }
  }
}
