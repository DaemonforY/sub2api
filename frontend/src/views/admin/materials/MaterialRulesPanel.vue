<template>
  <div class="grid gap-4 xl:grid-cols-2">
    <div class="card p-4">
      <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.rules.styles') }}</h3>
      <div class="space-y-3">
        <div v-for="s in styles" :key="s.id" class="rounded-lg border border-gray-100 p-3 dark:border-dark-700" :data-testid="`materials-rule-style-${s.id}`">
          <div class="flex flex-wrap items-baseline gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ s.name }}</span>
            <span class="text-xs text-gray-500">{{ s.category }} · {{ s.default_size }} · {{ s.default_quality }}</span>
            <span v-if="!s.enabled" class="text-xs text-gray-400">{{ t('materials.rules.disabled') }}</span>
          </div>
          <p class="text-sm text-gray-700 dark:text-gray-200">{{ s.desc_zh }}</p>
          <p class="text-xs italic text-gray-400">{{ s.prompt_en }}</p>
          <p v-if="s.suits_emotions.length" class="mt-1 text-xs text-gray-500">{{ t('materials.rules.suits') }}：{{ s.suits_emotions.join('、') }}</p>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <div class="card space-y-2 p-4">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.rules.check') }}</h3>
        <textarea v-model="checkInput" rows="4" class="input text-sm" :placeholder="t('materials.rules.checkPlaceholder')" data-testid="materials-check-input" />
        <div class="flex items-center gap-2">
          <select v-model="checkTarget" class="input w-40 text-sm">
            <option value="prompt">{{ t('materials.rules.checkTarget.prompt') }}</option>
            <option value="text">{{ t('materials.rules.checkTarget.text') }}</option>
          </select>
          <button class="btn btn-primary btn-sm" :disabled="!checkInput.trim()" data-testid="materials-check-run" @click="runCheck">{{ t('materials.rules.checkRun') }}</button>
        </div>
        <div v-if="checkResult" class="space-y-1 text-sm" data-testid="materials-check-result">
          <p v-if="!checkResult.hits.length" class="text-emerald-600">{{ t('materials.rules.checkOk') }}</p>
          <p v-if="checkResult.blocked" class="font-medium text-red-600">{{ t('materials.rules.checkBlocked') }}</p>
          <ul class="text-xs">
            <li v-for="(h, i) in checkResult.hits" :key="i" :class="h.action === 'block' ? 'text-red-600' : 'text-amber-600'">
              {{ t(`materials.rules.actions.${h.action}`) }}：{{ h.match }}<template v-if="h.reason"> — {{ h.reason }}</template>
            </li>
          </ul>
          <template v-if="checkResult.prompt !== checkInput">
            <p class="text-xs text-gray-500">{{ t('materials.rules.checkStripped') }}</p>
            <pre class="whitespace-pre-wrap rounded bg-gray-50 p-2 text-xs dark:bg-dark-800">{{ checkResult.prompt }}</pre>
          </template>
        </div>
      </div>

      <div class="card p-4">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('materials.rules.blocklist') }}</h3>
        <table class="w-full text-left text-xs">
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="b in blocklist" :key="b.id" :class="b.enabled ? '' : 'opacity-40'">
              <td class="py-1.5 pr-2 font-mono text-gray-800 dark:text-gray-100">{{ b.term }}</td>
              <td class="pr-2 text-gray-500">{{ t(`materials.rules.kinds.${b.kind}`) }}</td>
              <td class="pr-2" :class="b.action === 'block' ? 'text-red-600' : 'text-amber-600'">{{ t(`materials.rules.actions.${b.action}`) }}</td>
              <td class="pr-2 text-gray-500">{{ t(`materials.rules.appliesTo.${b.applies_to}`) }}</td>
              <td class="text-gray-400">{{ b.reason }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import * as api from '@/api/admin/materials'

const { t } = useI18n()
const appStore = useAppStore()
const styles = ref<api.MaterialStyle[]>([])
const blocklist = ref<api.MaterialBlockTerm[]>([])
const checkInput = ref('')
const checkTarget = ref<'prompt' | 'text'>('prompt')
const checkResult = ref<Awaited<ReturnType<typeof api.checkPrompt>> | null>(null)

function fail(e: unknown) {
  appStore.showError(extractApiErrorMessage(e, t('common.error')))
}

async function runCheck() {
  try {
    checkResult.value = await api.checkPrompt(checkInput.value, checkTarget.value)
  } catch (e) {
    fail(e)
  }
}

onMounted(async () => {
  try {
    const [s, b] = await Promise.all([api.listStyles(), api.listBlocklist()])
    styles.value = s
    blocklist.value = b
  } catch (e) {
    fail(e)
  }
})
</script>
