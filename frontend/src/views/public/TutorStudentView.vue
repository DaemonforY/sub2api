<template>
  <div class="flex min-h-[100dvh] flex-col bg-gray-50 dark:bg-dark-950">
    <header class="bg-gradient-to-br from-primary-600 to-violet-600 px-4 py-3 text-white">
      <div class="mx-auto flex max-w-2xl items-center justify-between gap-3">
        <div class="min-w-0">
          <div class="truncate font-semibold">{{ info?.name || t('tutors.studentTitle') }}</div>
          <div class="truncate text-xs text-white/80">{{ joined ? t('tutors.studentHello', { name: joined.name }) : t('tutors.studentSub') }}</div>
        </div>
        <button v-if="joined" class="rounded-md px-2 py-1 text-xs text-white/80 hover:bg-white/15" @click="leave">{{ t('tutors.switchName') }}</button>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-2xl flex-1 flex-col p-4">
      <div v-if="loading" class="py-16 text-center text-sm text-gray-400">{{ t('common.loading') }}</div>
      <div v-else-if="fatal" class="card p-6 text-center text-sm text-gray-600 dark:text-dark-300" data-testid="tutor-student-error">{{ fatal }}</div>

      <!-- 进入：口令 + 名字 -->
      <form v-else-if="!joined" class="card space-y-4 p-5" data-testid="tutor-join" @submit.prevent="join">
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ info?.greeting }}</p>
        <label class="block text-sm">
          <span class="input-label">{{ t('tutors.yourName') }}</span>
          <input v-model.trim="name" class="input" maxlength="20" :placeholder="t('tutors.yourNamePlaceholder')" autocomplete="name" data-testid="tutor-join-name" />
        </label>
        <label v-if="info?.needs_pass" class="block text-sm">
          <span class="input-label">{{ t('tutors.classPass') }}</span>
          <input v-model.trim="pass" class="input" maxlength="20" inputmode="numeric" :placeholder="t('tutors.classPassPlaceholder')" data-testid="tutor-join-pass" />
        </label>
        <div v-if="joinError" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ joinError }}</div>
        <button class="btn btn-primary w-full" :disabled="joining || !name" data-testid="tutor-join-submit">{{ joining ? t('tutors.joining') : t('tutors.join') }}</button>
        <p class="text-[11px] leading-4 text-gray-400">{{ t('tutors.studentNotice') }}</p>
      </form>

      <div v-else class="card flex flex-1 flex-col p-4">
        <TutorChat
          :key="joined.token"
          :greeting="info?.greeting || ''"
          :suggestions="info?.suggestions || []"
          :send="send"
          :initial-left="joined.left"
          :storage-key="`tutor_chat_${code}`"
          class="flex-1"
          @error="onChatError"
        />
        <p class="mt-2 text-[11px] leading-4 text-gray-400">{{ t('tutors.studentDisclaimer') }}</p>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TutorChat from '@/components/tutor/TutorChat.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import { getPublic, joinTutor, streamStudentChat, type ChatMessage, type TutorJoin, type TutorPublic } from '@/api/tutors'

const { t } = useI18n()
const route = useRoute()
const code = String(route.params.code || '').toLowerCase()
const storeKey = `tutor_session_${code}`

const info = ref<TutorPublic | null>(null)
const loading = ref(true)
const fatal = ref('')
const joined = ref<TutorJoin | null>(null)
const name = ref('')
const pass = ref('')
const joining = ref(false)
const joinError = ref('')

onMounted(async () => {
  try {
    info.value = await getPublic(code)
    document.title = info.value.name
    if (!info.value.enabled) {
      fatal.value = t('tutors.closed')
      return
    }
    const saved = JSON.parse(localStorage.getItem(storeKey) || 'null') as TutorJoin | null
    if (saved?.token) {
      joined.value = saved
      name.value = saved.name
    }
  } catch (e) {
    fatal.value = extractApiErrorMessage(e, t('tutors.notFound'))
  } finally {
    loading.value = false
  }
})

async function join() {
  joining.value = true
  joinError.value = ''
  try {
    const j = await joinTutor(code, name.value, pass.value)
    joined.value = j
    localStorage.setItem(storeKey, JSON.stringify(j))
  } catch (e) {
    joinError.value = extractApiErrorMessage(e, t('common.error'))
  } finally {
    joining.value = false
  }
}

function leave() {
  localStorage.removeItem(storeKey)
  sessionStorage.removeItem(`tutor_chat_${code}`)
  joined.value = null
}

async function send(messages: ChatMessage[], onDelta: (t: string) => void, signal: AbortSignal) {
  const res = await streamStudentChat(code, joined.value!.token, messages, onDelta, signal)
  if (joined.value) {
    joined.value.left = res.left
    localStorage.setItem(storeKey, JSON.stringify(joined.value))
  }
  return res
}

/** The session ended (the same name joined on another device, or the assistant was reset): join again. */
function onChatError(err: Error & { reason?: string }) {
  if (err.reason === 'TUTOR_SESSION') {
    leave()
    joinError.value = t('tutors.sessionGone')
  }
}
</script>
