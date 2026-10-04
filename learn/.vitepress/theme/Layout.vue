<script setup lang="ts">
// The default theme with lesson header / footer, the account button, and ✓ on finished lessons
// in the sidebar.
import DefaultTheme from 'vitepress/theme'
import { nextTick, onMounted, watch } from 'vue'
import { useRoute } from 'vitepress'
import LessonHeader from './components/LessonHeader.vue'
import LessonFooter from './components/LessonFooter.vue'
import NavUser from './components/NavUser.vue'
import Tutor from './components/Tutor.vue'
import { loadProgress, progress } from './api'

const { Layout } = DefaultTheme
const route = useRoute()

function markSidebar() {
  if (typeof document === 'undefined') return
  document.querySelectorAll<HTMLAnchorElement>('.VPSidebar a.VPLink').forEach((a) => {
    const m = a.getAttribute('href')?.match(/\/([a-z]\d{1,2})(\.html)?$/)
    a.classList.toggle('lesson-is-done', !!(m && progress.completed[m[1]]))
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
    <template #doc-before><LessonHeader /></template>
    <template #doc-footer-before><LessonFooter /></template>
    <template #nav-bar-content-after><NavUser /></template>
    <template #layout-bottom><Tutor /></template>
  </Layout>
</template>
