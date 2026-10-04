<script setup lang="ts">
// Below a lesson: mark it done and go on.
import { computed, ref } from 'vue'
import { useData, withBase } from 'vitepress'
import { findLesson } from '../tracks'
import { loginUrl, markDone, progress, token } from '../api'
import Quiz from './Quiz.vue'

const { frontmatter } = useData()
const found = computed(() => (frontmatter.value.lesson ? findLesson(String(frontmatter.value.lesson)) : null))
const next = computed(() => (found.value ? found.value.track.lessons[found.value.index + 1] : undefined))
const done = computed(() => !!(found.value && progress.completed[found.value.lesson.id]))
const saving = ref(false)
const signedIn = computed(() => progress.loaded && !!token())
const trackDone = computed(() => {
  if (!found.value) return 0
  return found.value.track.lessons.filter((l) => progress.completed[l.id]).length
})

async function complete() {
  if (!found.value || saving.value) return
  saving.value = true
  await markDone(found.value.lesson.id)
  saving.value = false
}
</script>

<template>
  <Quiz v-if="found && found.track.quizzes" :key="found.lesson.id" :lesson="found.lesson.id" />
  <div v-if="found" class="lesson-foot" data-testid="lesson-footer">
    <div class="lesson-foot-row">
      <button v-if="!done" class="runbox-btn" :disabled="saving" data-testid="lesson-complete" @click="complete">✓ 完成本课</button>
      <span v-else class="lesson-done big">✓ 已完成本课</span>
      <span class="runbox-note">本路线已完成 {{ trackDone }} / {{ found.track.lessons.length }} 课</span>
    </div>
    <div class="lesson-foot-row">
      <a v-if="next && next.ready" class="lesson-next" :href="withBase(`/${found.track.id}/${next.id}`)">下一课：{{ next.title }} →</a>
      <span v-else-if="next" class="runbox-note">下一课「{{ next.title }}」即将上线，先去 <a :href="withBase('/guide/')">延伸阅读</a> 看看。</span>
      <span v-else class="runbox-note">这条路线学完了 🎉 <a :href="withBase(`/${found.track.id}/#结业`)">去路线页领取结业证书 →</a></span>
    </div>
    <p v-if="progress.loaded && !signedIn" class="runbox-note">
      进度先保存在这个浏览器里，<a :href="loginUrl()">登录 HiveGPT</a> 后会同步到账号，换设备也在。
    </p>
  </div>
</template>
