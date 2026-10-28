<template>
  <div class="space-y-4">
    <div class="card space-y-3 p-4">
      <textarea v-model="scene" rows="3" class="input" :placeholder="t('inspiration.find.scene')" data-testid="find-scene" />
      <div class="flex flex-wrap items-center gap-3 text-sm">
        <select v-model="dynasty" class="input w-32 py-1.5">
          <option value="">{{ t('inspiration.allDynasties') }}</option>
          <option v-for="d in dynasties" :key="d" :value="d">{{ d }}</option>
        </select>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300">
          {{ t('inspiration.find.count') }}
          <input v-model.number="count" type="number" min="1" max="20" class="input w-16 py-1" />
        </label>
        <label class="flex items-center gap-1 text-gray-600 dark:text-gray-300"><input v-model="rerank" type="checkbox" /> {{ t('inspiration.find.rerank') }}</label>
        <button class="btn btn-primary" :disabled="busy || !scene.trim()" data-testid="find-go" @click="go">
          {{ busy ? t('inspiration.find.going') : t('inspiration.find.go') }}
        </button>
      </div>
      <p class="text-xs text-gray-400">{{ t('inspiration.find.hint') }}</p>
    </div>

    <template v-if="result">
      <div class="card flex flex-wrap items-center gap-2 p-3 text-sm">
        <span class="text-gray-500">{{ t('inspiration.find.keywords') }}</span>
        <span v-for="w in result.keywords" :key="w" class="rounded-full bg-amber-50 px-2.5 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">{{ w }}</span>
      </div>

      <section v-if="rerankUsed">
        <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.find.picks') }}</h3>
        <p v-if="!result.picks.length" class="card py-6 text-center text-sm text-gray-400">{{ t('inspiration.find.noPicks') }}</p>
        <div class="grid gap-3 lg:grid-cols-2" data-testid="find-picks">
          <InspirationPoemCard v-for="p in result.picks" :key="p.poem.id" :poem="p.poem" :reason="p.reason" :words="result.keywords" @starred="onStarred" />
        </div>
      </section>

      <div class="grid gap-4 xl:grid-cols-[1fr_20rem]">
        <section>
          <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.find.candidates') }}</h3>
          <div class="grid gap-3 lg:grid-cols-2">
            <InspirationPoemCard v-for="p in others" :key="p.id" :poem="p" :words="result.keywords" @starred="onStarred" />
          </div>
        </section>
        <aside v-if="result.songs.length" class="card h-fit p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('inspiration.poems.songHits') }}</h3>
          <div v-for="s in result.songs" :key="s.id" class="mt-3 border-t border-gray-100 pt-3 text-sm first:border-0 first:pt-0 dark:border-dark-700">
            <div class="font-medium text-gray-800 dark:text-gray-100">{{ s.title }} <span class="text-xs font-normal text-gray-500">{{ s.artist }}</span></div>
            <p v-for="(m, i) in s.matches" :key="i" class="mt-1 text-xs text-gray-600 dark:text-gray-300">
              <template v-for="(p, j) in highlightParts(m, result.keywords)" :key="j"><mark v-if="p.hit" class="rounded bg-amber-100 px-0.5 text-inherit dark:bg-amber-900/40">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template>
            </p>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/inspiration'
import InspirationPoemCard from './InspirationPoemCard.vue'
import { highlightParts } from './highlight'

const { t } = useI18n()
const appStore = useAppStore()
const dynasties = ['先秦', '汉', '魏晋', '南北朝', '隋', '唐', '五代', '宋', '辽', '金', '元', '明', '清']

const scene = ref('')
const dynasty = ref('')
const count = ref(8)
const rerank = ref(true)
const rerankUsed = ref(true)
const busy = ref(false)
const result = ref<api.LibFindResult | null>(null)

const others = computed(() => {
  const picked = new Set(result.value?.picks.map((p) => p.poem.id))
  return (result.value?.candidates ?? []).filter((p) => !picked.has(p.id)).slice(0, 30)
})

async function go() {
  busy.value = true
  try {
    rerankUsed.value = rerank.value
    result.value = await api.findByScene({ scene: scene.value.trim(), dynasty: dynasty.value, count: count.value || 8, rerank: rerank.value })
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    busy.value = false
  }
}

function onStarred(id: number, value: boolean) {
  const r = result.value
  if (!r) return
  r.picks = r.picks.map((p) => (p.poem.id === id ? { ...p, poem: { ...p.poem, starred: value } } : p))
  r.candidates = r.candidates.map((p) => (p.id === id ? { ...p, starred: value } : p))
}
</script>
