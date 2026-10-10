<script setup lang="ts">
// The default theme with lesson header / footer, the account button, ✓ on finished lessons in the
// sidebar, and share cards (分享 button, select-to-quote).
import DefaultTheme from 'vitepress/theme'
import { nextTick, onMounted, watch } from 'vue'
import { useRoute } from 'vitepress'
import LessonHeader from './components/LessonHeader.vue'
import LessonFooter from './components/LessonFooter.vue'
import NavUser from './components/NavUser.vue'
import Tutor from './components/Tutor.vue'
import ShareButton from './components/ShareButton.vue'
import SelectionShare from './components/SelectionShare.vue'
import ShareCardDialog from './components/ShareCardDialog.vue'
import { loadProgress, progress } from './api'
import { lessonByPath } from './tracks'

const { Layout } = DefaultTheme
const route = useRoute()

function markSidebar() {
  if (typeof document === 'undefined') return
  document.querySelectorAll<HTMLAnchorElement>('.VPSidebar a.VPLink').forEach((a) => {
    const id = lessonByPath(new URL(a.href, window.location.href).pathname)
    a.classList.toggle('lesson-is-done', !!(id && progress.completed[id]))
  })
}

onMounted(() => {
  void loadProgress().then(() => nextTick(markSidebar))
})
watch(() => route.path, () => nextTick(markSidebar))
watch(() => progress.completed, () => nextTick(markSidebar), { deep: true })
</script>

<template>
  <Layout>
    <template #doc-before><ShareButton place="top" /><LessonHeader /></template>
    <template #doc-footer-before><LessonFooter /><ShareButton place="bottom" /></template>
    <template #nav-bar-content-after><NavUser /></template>
    <template #layout-bottom><Tutor /><SelectionShare /><ShareCardDialog /></template>
  </Layout>
</template>
