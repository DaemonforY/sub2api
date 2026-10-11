// 测测你的写作风格 (/style): the free part, worked out in the browser before any login — sentence
// and paragraph length, how often the writer says 我 / 你, their recurring openers (catchphrases)
// and the AI-flavoured words they use. Pure functions, no DOM.

/** Words AI drafts lean on and people rarely write (used for the 「AI 味」 reading). */
export const AI_FLAVOR_WORDS = [
  '值得注意的是', '总而言之', '综上所述', '总的来说', '不可或缺', '至关重要', '深入探讨', '不言而喻', '毋庸置疑',
  '众所周知', '日新月异', '在当今', '赋能', '抓手', '底层逻辑', '闭环', '颗粒度', '扮演着重要', '发挥着重要',
  '一站式', '全方位', '多维度', '助力', '深度解析', '让我们一起', '革命性', '无缝', '显著提升', '与此同时'
]

/** Two-character words too common to count as a catchphrase on their own. */
const COMMON = new Set([
  '我们', '你们', '他们', '这个', '那个', '一个', '没有', '就是', '因为', '所以', '但是', '如果', '可以', '这样', '时候',
  '自己', '什么', '怎么', '还是', '已经', '然后', '这些', '那些', '而且', '其中', '大家', '一些', '这种', '现在', '可能'
])

const SENTENCE_END = /[。！？!?…]+|\n+/
const CLAUSE_SPLIT = /[，,。！？!?；;：:…、“”"「」『』（）()《》\s]+/
const EMOJI = /\p{Extended_Pictographic}/gu

export interface Catchphrase {
  text: string
  count: number
}

export interface WritingStats {
  articles: number
  /** Characters without whitespace. */
  chars: number
  sentences: number
  avgSentence: number
  /** Share of sentences of 12 characters or fewer, 0–100. */
  shortPct: number
  /** Share of sentences of 40 characters or more, 0–100. */
  longPct: number
  paragraphs: number
  avgParagraph: number
  /** Per 1000 characters. */
  questions: number
  exclaims: number
  /** 我 (not 我们) / 你 (not 你们) per 1000 characters. */
  iRate: number
  youRate: number
  emojis: number
  catchphrases: Catchphrase[]
  aiWords: Catchphrase[]
  /** AI-flavoured words per 1000 characters. */
  aiRate: number
  rhythm: 'short' | 'balanced' | 'long'
  aiLevel: 'low' | 'some' | 'high'
}

function countOf(text: string, word: string): number {
  let n = 0
  for (let i = text.indexOf(word); i >= 0; i = text.indexOf(word, i + word.length)) n++
  return n
}

function cjkLength(s: string): number {
  return s.replace(/\s+/g, '').replace(/[，,。！？!?；;：:…、“”"「」『』（）()《》\-—·.]/g, '').length
}

/**
 * Openers repeated across the texts: the leading 2–6 CJK characters of clauses. Longer phrases need
 * fewer repeats (2) than two-character words (3, or 4 in a single article); a phrase wins over its
 * prefixes when they occur about as often (「你也许会问」, not 「你也」).
 */
export function findCatchphrases(texts: string[], limit = 8): Catchphrase[] {
  const counts = new Map<string, number>()
  for (const text of texts) {
    for (const clause of text.split(CLAUSE_SPLIT)) {
      const head = /^[一-鿿]+/.exec(clause)?.[0] || ''
      for (let n = 2; n <= Math.min(6, head.length); n++) {
        const gram = head.slice(0, n)
        counts.set(gram, (counts.get(gram) || 0) + 1)
      }
    }
  }
  const minShort = texts.length > 1 ? 3 : 4
  const candidates = [...counts.entries()]
    .filter(([g, c]) => (g.length === 2 ? c >= minShort && !COMMON.has(g) : c >= 2) && !/^(的|了|是|在|和|也|都|就|又|不|而|与|或|把|被|让|对|从|为)/.test(g))
    .sort((a, b) => b[1] * (b[0].length - 1) - a[1] * (a[0].length - 1) || b[0].length - a[0].length)
  const picked: Catchphrase[] = []
  for (const [text, count] of candidates) {
    if (picked.some((p) => (p.text.includes(text) || text.includes(p.text)) && Math.abs(p.count - count) <= Math.max(1, count / 3))) continue
    picked.push({ text, count })
    if (picked.length >= limit) break
  }
  return picked
}

export function analyzeWriting(texts: string[]): WritingStats {
  const clean = texts.map((t) => t.trim()).filter(Boolean)
  const all = clean.join('\n\n')
  const chars = all.replace(/\s+/g, '').length
  const per1k = (n: number) => (chars ? Math.round((n / chars) * 10000) / 10 : 0)

  const sentences = all.split(SENTENCE_END).map(cjkLength).filter((n) => n > 0)
  const avgSentence = sentences.length ? Math.round(sentences.reduce((a, b) => a + b, 0) / sentences.length) : 0
  const pct = (n: number) => (sentences.length ? Math.round((n / sentences.length) * 100) : 0)
  const paragraphs = clean.flatMap((t) => t.split(/\n\s*\n|\n/)).map(cjkLength).filter((n) => n > 0)
  const avgParagraph = paragraphs.length ? Math.round(paragraphs.reduce((a, b) => a + b, 0) / paragraphs.length) : 0

  const iCount = countOf(all, '我') - countOf(all, '我们')
  const youCount = countOf(all, '你') - countOf(all, '你们')
  const aiWords = AI_FLAVOR_WORDS.map((text) => ({ text, count: countOf(all, text) }))
    .filter((w) => w.count > 0)
    .sort((a, b) => b.count - a.count)
  const aiRate = per1k(aiWords.reduce((a, w) => a + w.count, 0))

  return {
    articles: clean.length,
    chars,
    sentences: sentences.length,
    avgSentence,
    shortPct: pct(sentences.filter((n) => n <= 12).length),
    longPct: pct(sentences.filter((n) => n >= 40).length),
    paragraphs: paragraphs.length,
    avgParagraph,
    questions: per1k((all.match(/[？?]/g) || []).length),
    exclaims: per1k((all.match(/[！!]/g) || []).length),
    iRate: per1k(iCount),
    youRate: per1k(youCount),
    emojis: (all.match(EMOJI) || []).length,
    catchphrases: findCatchphrases(clean),
    aiWords,
    aiRate,
    rhythm: avgSentence <= 15 ? 'short' : avgSentence <= 25 ? 'balanced' : 'long',
    aiLevel: aiRate < 1 ? 'low' : aiRate < 2.5 ? 'some' : 'high'
  }
}
