// Gallery cards start their live player through this queue: at most LIMIT frames load at once, in the
// order the cards scrolled into view, so the first screen comes up before the rest compete with it.
const LIMIT = 4
const waiting = []
let running = 0

function next() {
  while (running < LIMIT && waiting.length) {
    const job = waiting.shift()
    running++
    let done = false
    const release = () => {
      if (done) return
      done = true
      running--
      next()
    }
    // A frame that never reports ready must not hold its slot forever.
    setTimeout(release, 8000)
    job(release)
  }
}

/** Queues start(release); call release() once the card's frame is ready (or the card goes away). */
export function enqueueCard(start) {
  let cancelled = false
  waiting.push((release) => (cancelled ? release() : start(release)))
  next()
  return () => {
    cancelled = true
  }
}
