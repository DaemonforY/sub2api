// The create page remembers the last choices.
const KEY = 'hivegpt-video:prefs'
export const prefs = (() => {
  try {
    return JSON.parse(localStorage.getItem(KEY) || '{}')
  } catch {
    return {}
  }
})()
export function savePrefs(next) {
  Object.assign(prefs, next)
  localStorage.setItem(KEY, JSON.stringify(prefs))
}
