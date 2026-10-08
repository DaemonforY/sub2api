import { ref } from 'vue'

const KEY = 'hivegpt-video:theme'
export const dark = ref(false)

export function initTheme() {
  const saved = localStorage.getItem(KEY)
  dark.value = saved ? saved === 'dark' : false
  document.documentElement.classList.toggle('dark', dark.value)
}

export function toggleTheme() {
  dark.value = !dark.value
  localStorage.setItem(KEY, dark.value ? 'dark' : 'light')
  document.documentElement.classList.toggle('dark', dark.value)
}
