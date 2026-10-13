export default {
  desk: {
    intro: 'Write in your own voice: gather sourced material, pick an angle, compare three drafts, polish in the editor, then turn it into every platform’s version. The AI brings material and options; you decide.',
    tabs: {
      desk: 'Writing desk',
      review: 'To publish',
      topics: 'Topics',
      persona: 'Author profile',
      stats: 'Stats',
      settings: 'Settings'
    },
    summary: {
      desk: 'Being written',
      pending: 'To publish',
      topics: 'Pending topics',
      published: 'Published',
      goSettings: 'Open settings'
    },
    stage: {
      materials: 'Material',
      angles: 'Angle',
      drafts: 'Drafts',
      edit: 'Polish',
      final: 'Final'
    },
    job: {
      research: 'Researching fact cards',
      angles: 'Thinking of angles',
      reference: 'Analysing the reference article',
      drafts: 'Writing three drafts and asking a reader',
      merge: 'Merging highlights into the base',
      check: 'Checking the text',
      learn: 'Learning from your edits',
      platforms: 'Writing platform versions'
    },
    common: {
      back: 'Back',
      cancel: 'Cancel',
      chars: 'chars',
      close: 'Close',
      delete: 'Delete',
      loading: 'Loading…',
      ok: 'OK'
    },
    review: {
      empty: 'Nothing to publish yet. Finalise an article on the desk and generate its platform versions.'
    },
    list: {
      newTitle: 'Start a new article',
      newHint: 'Say what you want to write and what you have. The more concrete the material, the more it sounds like you.',
      new: 'New',
      noPersona: 'No author profile yet, so it won’t sound like you.',
      goPersona: 'Create it first',
      title: 'What about',
      titlePlaceholder: 'e.g. A week of letting Codex change our old code',
      idea: 'Your take (optional)',
      ideaPlaceholder: 'The point you most want to make, or what made you want to write this',
      notes: 'Your material (recommended)',
      notesPlaceholder: 'Your experience, numbers you saw, pitfalls, screenshots, links… things the AI cannot make up',
      notesHint: 'The AI only uses what you give it and public facts it finds with sources; it never invents experiences for you.',
      start: 'Start: gather material',
      empty: 'The desk is empty',
      failed: 'The last step failed'
    },
    persona: {
      intro: 'The author profile decides whether it sounds like you. Add 2–5 articles you wrote; the AI distills who you are, what you did, your stances and how you talk; edit it until it is right, then save. Every step writes in this voice.',
      updated: 'Updated {time}',
      samples: 'My sample articles',
      samplesHint: 'Paste WeChat article links or the text itself. Pick the ones that sound most like you.',
      distill: 'Distill from samples',
      distilling: 'Reading your articles… (about a minute)',
      urlPlaceholder: 'https://mp.weixin.qq.com/s/…',
      addUrl: 'Fetch',
      paste: 'No link? Paste the text',
      titlePlaceholder: 'Title',
      textPlaceholder: 'Text (at least 200 characters)',
      addText: 'Add as sample',
      proposed: 'This profile was distilled from your samples. Edit it, then save. Experiences must be things that really happened.',
      name: 'Name you write under',
      identity: 'Who I am',
      identityPlaceholder: 'Who you are, what you do, how you relate to readers, in the first person',
      experiences: 'Things I really did or went through',
      experiencesPlaceholder: 'One per line, e.g. 39,000 credential-stuffing attempts in the site’s first week',
      experiencesHint: 'Used as first-hand material when writing, so only real events.',
      stances: 'My stances',
      phrases: 'Phrases I use',
      likes: 'Things I like',
      dislikes: 'Things I dislike',
      voice: 'How I talk and write',
      voicePlaceholder: 'Person, sentence length, punctuation habits, openings and endings, headings or not, where the humour comes from…',
      taboos: 'I never write like this',
      taboosPlaceholder: 'One per line',
      oneLine: 'List fields take one item per line.',
      saved: 'Author profile saved'
    },
    materials: {
      title: 'Material gate',
      hint: 'An article stands on its material. Public facts come with sources; your experiences and answers are first-hand. Mark doubtful ones as Doubt and unreliable ones as Drop.',
      usable: 'usable pieces',
      notEnough: 'Fewer than 5 pieces won’t carry a long article. Answer the questions, add your own experiences, or research again.',
      questions: 'The AI would like to ask (answers become first-hand material)',
      answerPlaceholder: 'Answer the way you would say it',
      facts: 'Fact cards',
      researching: 'Researching…',
      empty: 'No fact cards yet',
      addPlaceholder: 'Add a fact or experience you know',
      add: 'Add',
      hintPlaceholder: 'What to look for this time (optional)',
      again: 'Research again',
      save: 'Save',
      saved: 'Saved',
      next: 'Next: angles'
    },
    fact: {
      kind: {
        user: 'From you',
        persona: 'Your experience',
        answer: 'Your answer',
        web: 'Public source'
      },
      status: {
        ok: 'Use',
        doubt: 'Doubt',
        rejected: 'Drop'
      }
    },
    angles: {
      title: 'Pick an angle',
      hint: 'Different angles make different articles. H is fun, K is substance, R is resonance; good angles score on all three. Pick one and edit it freely.',
      make: 'Suggest angles',
      again: 'Other angles',
      empty: 'No angles yet',
      hkr: {
        H: 'Fun',
        K: 'Substance',
        R: 'Resonance'
      },
      reader: 'For: {reader}',
      edit: 'Make it yours',
      fTitle: 'Title direction',
      fAngle: 'Angle',
      fHook: 'Opening',
      fNote: 'Extra instructions (optional)',
      fNotePlaceholder: 'e.g. Not a tutorial; more about my mistakes',
      write: 'Write three drafts'
    },
    ref: {
      title: 'Learn from a hit article (optional)',
      hint: 'Paste a well-written article of the same kind. Only its structure and rhythm are used, never its content.',
      urlPlaceholder: 'WeChat article link',
      analyze: 'Analyse',
      paste: 'Or paste the text',
      from: 'Reference: {source}',
      opening: 'Opening: ',
      emotion: 'Emotion: ',
      escalation: 'Escalation: ',
      callbacks: 'Callbacks: '
    },
    drafts: {
      title: 'Three drafts',
      hint: 'One angle, three structures. The bars show a simulated reader: yellow starts skimming, red leaves. Pick a base and tick highlights from the others to merge.',
      make: 'Write drafts',
      again: 'Rewrite drafts',
      empty: 'No drafts yet',
      judge: 'Editor’s view: ',
      skim: 'Skimming',
      leave: 'Leaves here',
      label: '{label}',
      rank: '#{n}',
      score: {
        hook: 'Hook',
        flow: 'Flow',
        voice: 'Human',
        substance: 'Substance'
      },
      useBase: 'Use as base',
      isBase: '✓ Base',
      highlights: 'Merge highlights from the other drafts',
      noHighlights: 'No other highlights; go straight to polishing.',
      merge: 'Merge {n} highlights and polish',
      edit: 'Polish the base'
    },
    editor: {
      placeholder: 'Start writing…',
      h2: 'Heading',
      bold: 'Bold',
      quote: 'Quote',
      list: 'List',
      count: '{chars} chars · {paras} paragraphs',
      saving: 'Saving…',
      saved: 'Saved {time}',
      saveFailed: 'Save failed',
      snapshot: 'Save a version',
      snapshotPlaceholder: 'Name this version',
      backDrafts: 'Back to drafts',
      finalize: 'Finalise'
    },
    assist: {
      voice: 'More like me',
      tighten: 'Tighten',
      expand: 'Expand',
      sharper: 'Sharper',
      lighter: 'Lighter',
      plain: 'Plain words',
      analogy: 'Analogies',
      evidence: 'Evidence',
      custom: 'Custom',
      customPlaceholder: 'How to change it',
      comment: 'Comment',
      thinking: 'Rewriting…',
      accept: 'Accept',
      retry: 'Try again',
      reject: 'Discard',
      insertAfter: 'Insert after the selection',
      keepFact: 'Add to fact cards',
      factKept: 'Added to the fact cards',
      nothing: 'Nothing usable this time; try different wording',
      lost: 'The text changed; the place to replace can’t be found'
    },
    side: {
      check: 'Check',
      facts: 'Facts',
      versions: 'Versions',
      comments: 'Comments',
      factsHint: 'Facts in the text may only come from these cards; for a new one, select text and use Evidence.'
    },
    check: {
      run: 'Check the text',
      again: 'Check again',
      hint: 'Fixed rules first (pivot sentences, jargon, dashes, unfilled experiences), then an editor reads each paragraph: does it sound like you, would readers drift, any invented facts.',
      stats: 'One-sentence paragraphs {single}% · headings {headings} · experiences to fill {placeholders}',
      locate: 'Show',
      apply: 'Apply fix',
      ignore: 'Ignore',
      notFound: 'That text is no longer in the article',
      level: {
        must: 'Must fix',
        should: 'Should fix'
      },
      layer: {
        rule: 'Rule',
        voice: 'Not you',
        flow: 'Hard to read',
        content: 'Thin',
        fact: 'Fact'
      },
      score: {
        voice: 'Your voice',
        flow: 'Flow',
        substance: 'Substance',
        facts: 'Reliable'
      }
    },
    versions: {
      empty: 'No versions yet',
      restore: 'Restore this version',
      manual: 'Manual save',
      saved: 'Saved as a version',
      kind: {
        draft: 'Base chosen',
        merge: 'Highlights merged',
        manual: 'Manual save',
        ai: 'AI edit',
        restore: 'Before restore',
        final: 'Final'
      }
    },
    comments: {
      hint: 'Select text and use Comment to leave a note next to it.',
      placeholder: 'Your note',
      resolve: 'Resolve',
      reopen: 'Reopen'
    },
    final: {
      title: 'Final',
      hint: 'The WeChat version is this text; Zhihu, CSDN, Toutiao and Xiaohongshu are rewritten in your voice, then reviewed under To publish.',
      platforms: 'Write platform versions',
      backEdit: 'Keep editing',
      learn: 'Learned from your edits',
      learnHint: 'The AI compared the draft it handed over with your final text. Add the rules you agree with; every later article follows them.',
      noLearn: 'No edits to learn from, or still working.',
      addRule: 'Add to writing rules',
      dismiss: 'Not right',
      added: 'Added to the writing rules',
      dismissed: 'Dismissed'
    },
    view: {
      back: 'Back to the desk',
      tokens: '{n} tokens used',
      elapsed: '{s} s',
      stop: 'Stop',
      failed: '“{step}” failed: {msg}',
      openDesk: 'Open on the desk'
    }
  }
}
