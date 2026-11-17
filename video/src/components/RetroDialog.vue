<template>
  <Modal title="复盘这部片子" wide @close="$emit('close')">
    <div data-testid="retro">
      <div v-if="loading" class="flex flex-col items-center gap-3 py-10 text-sm text-ink-500">
        <Loader2 class="h-6 w-6 animate-spin text-brand-500" />
        AI 正在翻这部片子的修改记录，找返工的原因…（约半分钟，按用量从余额扣费）
      </div>
      <div v-else-if="error" class="rounded-xl bg-red-50 p-4 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
        {{ error }}
        <button class="btn-ghost btn-sm ml-2" @click="run">再试一次</button>
      </div>
      <template v-else-if="retro">
        <p class="rounded-xl bg-ink-50 px-4 py-3 text-sm leading-6 dark:bg-ink-950">{{ retro.summary }}</p>

        <h3 class="mt-5 text-sm font-semibold">返工了什么</h3>
        <p v-if="!retro.rework.length" class="mt-1 text-sm text-ink-500">这部片子没有明显的返工。</p>
        <ul v-else class="mt-2 space-y-1.5 text-sm">
          <li v-for="(w, i) in retro.rework" :key="i" class="flex items-start gap-2">
            <span class="mt-0.5 shrink-0 rounded-md px-1.5 py-0.5 text-[11px]" :class="causeClass(w.cause)">{{ w.cause || '其他' }}</span>
            <span class="leading-6">{{ w.what }}</span>
          </li>
        </ul>

        <template v-if="retro.keep.length">
          <h3 class="mt-5 text-sm font-semibold">值得留着当对标的</h3>
          <ul class="mt-2 list-disc space-y-1 pl-5 text-sm leading-6 text-ink-600 dark:text-ink-300">
            <li v-for="(k, i) in retro.keep" :key="i">{{ k }}</li>
          </ul>
        </template>

        <h3 class="mt-5 text-sm font-semibold">建议写进系列规则</h3>
        <p class="mt-1 text-xs text-ink-500">规则不是越多越好——只勾真正会再犯的。勾上的规则存进风格卡，以后用这个风格做的片子都会遵守。</p>
        <p v-if="!retro.rules.length && !retro.drop.length" class="mt-2 text-sm text-ink-500">这次没有需要新增的规则。</p>
        <label v-for="(r, i) in retro.rules" :key="'r' + i" class="mt-2 flex items-start gap-2 text-sm">
          <input v-model="pickedRules" type="checkbox" :value="r" class="mt-1 h-4 w-4 accent-brand-500" />
          <span class="leading-6">{{ r }}</span>
        </label>
        <label v-for="(r, i) in retro.drop" :key="'d' + i" class="mt-2 flex items-start gap-2 text-sm">
          <input v-model="pickedDrops" type="checkbox" :value="r" class="mt-1 h-4 w-4 accent-brand-500" />
          <span class="leading-6"><span class="mr-1 rounded-md bg-ink-100 px-1.5 py-0.5 text-[11px] dark:bg-ink-800">删掉旧规则</span>{{ r }}</span>
        </label>

        <div v-if="retro.rules.length || retro.drop.length" class="mt-5 flex flex-wrap items-center gap-2 border-t border-ink-100 pt-4 dark:border-ink-800">
          <span class="text-xs text-ink-500">存到</span>
          <select v-model="target" class="rounded-lg border border-ink-200 bg-white px-2 py-1.5 text-sm dark:border-ink-700 dark:bg-ink-900">
            <option v-for="st in styles" :key="st.id" :value="st.id">风格卡「{{ st.name }}」</option>
            <option v-if="canSaveNew" value="new">新建风格卡（用这部片子的外观）</option>
          </select>
          <span class="flex-1"></span>
          <button class="btn-primary" :disabled="saving || !target || (!pickedRules.length && !pickedDrops.length)" @click="save"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" />存进风格卡</button>
        </div>
      </template>
    </div>
  </Modal>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import Modal from './Modal.vue'
import { api, myStyles } from '../lib/api'
import { toast, toastError } from '../lib/toast'

const props = defineProps({ project: { type: Object, required: true } })
const emit = defineEmits(['close'])
const loading = ref(false)
const error = ref('')
const retro = ref(null)
const styles = ref([])
const pickedRules = ref([])
const pickedDrops = ref([])
const target = ref('')
const saving = ref(false)
const canSaveNew = computed(() => !!props.project.spec?.common)

function causeClass(c) {
  if (c === '规则没写') return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
  if (c === '写了没执行') return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
  return 'bg-ink-100 text-ink-600 dark:bg-ink-800 dark:text-ink-300'
}

async function run() {
  loading.value = true
  error.value = ''
  try {
    retro.value = await api(`/projects/${props.project.id}/retro`, { method: 'POST', body: {} })
    pickedRules.value = [...retro.value.rules]
  } catch (err) {
    error.value = err.message || '复盘失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    if (target.value === 'new') {
      const name = (props.project.title || '我的风格').slice(0, 20)
      const card = await api('/styles', { method: 'POST', body: { project_id: props.project.id, name, rules: pickedRules.value } })
      toast(`已新建风格卡「${card.name}」并存入 ${pickedRules.value.length} 条规则`, 'success')
    } else {
      const card = styles.value.find((s) => s.id === target.value)
      const rules = [...(card.rules || []).filter((r) => !pickedDrops.value.includes(r)), ...pickedRules.value]
      await api(`/styles/${card.id}`, { method: 'PUT', body: { rules } })
      toast(`风格卡「${card.name}」已更新，现在共 ${rules.length} 条规则`, 'success')
    }
    emit('close')
  } catch (err) {
    toastError(err)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  styles.value = await myStyles()
  const own = styles.value.find((s) => s.id === props.project.options?.my_style)
  target.value = own ? own.id : styles.value[0]?.id || (canSaveNew.value ? 'new' : '')
  run()
})
</script>
