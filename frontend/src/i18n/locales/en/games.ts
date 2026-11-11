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
      ink: { title: 'Ink in water', desc: 'Stir a pool of ink with your finger and watch it bleed into rice paper; guide it into a character.', tag: 'Ink' },
      craft: { title: 'AI Infinite Alchemy', desc: 'Start from metal, wood, water, fire and earth; drag two together and the AI invents something new. The first maker is remembered site-wide.', tag: 'AI' },
      fps: { title: 'HiveGPT 3A · Harbour breakout', desc: 'A browser first-person shooter written from scratch by AI: cut three relays, break the lockdown, reach the boat. Keyboard, touch or gamepad.', tag: 'FPS' },
      abti: { title: 'ABTI · AI worker personality test', desc: '31 questions over 15 dimensions: what kind of worker are you in the AI era?', tag: 'Personality' },
      style: { title: 'Find your writing style', desc: 'Paste a few of your articles and see your rhythm, catchphrases and AI flavour.', tag: 'Writing' }
    },
    craft: {
      title: 'AI Infinite Alchemy',
      tagline: 'Start from the five elements, drag two together and the AI invents the result',
      siteTotal: '{n} found site-wide',
      mine: 'You have {n}',
      latest: 'Latest',
      hint: 'Drag one element onto another to combine; double-click to copy, drag back right to remove',
      hintTouch: 'Drag one element onto another to combine; tap an element below to place it',
      clear: 'Clear board',
      search: 'Search {n} elements',
      shelfNote: 'Drag or click to place on the board; ★ marks elements you made first on the site.',
      reset: 'Start over',
      resetConfirm: 'Forget your collection and go back to the five elements?',
      thinking1: 'Inventing…',
      thinking2: 'Nobody has tried this pair yet…',
      thinking3: 'Naming the new thing…',
      thinking4: 'Almost there…',
      newForYou: 'New for you: {e} {n}',
      failed: 'Combining failed, try again',
      firstKicker: 'FIRST DISCOVERY',
      firstNo: 'Element #{n} on the site, made first by you',
      share: 'Make a share card',
      keepGoing: 'Keep going',
      shareTitle: 'I was the first to make {e} {n} in AI Infinite Alchemy',
      shareCta: 'Scan to make your own discovery'
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
