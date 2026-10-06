<script setup lang="ts">
// Shown when the day's free calls are used up: go on with one of the learner's own keys
// (billed as usual on that key), or come back tomorrow.
import { onMounted, ref } from 'vue'
import { chooseOwnKey, listOwnKeys, ownKey, runMode, type KeyOption } from '../api'
import { MAIN_SITE } from '../tracks'

defineProps<{ what?: string }>()
const emit = defineEmits<{ chosen: [] }>()
const keys = ref<KeyOption[] | null>(null)
const picked = ref(0)

onMounted(async () => {
  keys.value = await listOwnKeys()
  picked.value = ownKey.id && keys.value.some((k) => k.id === ownKey.id) ? ownKey.id : keys.value[0]?.id || 0
})

function use() {
  const key = keys.value?.find((k) => k.id === picked.value)
  if (!key) return
  chooseOwnKey(key)
  emit('chosen')
}
</script>

<template>
  <div class="ownkey" data-testid="own-key-picker">
    <p v-if="runMode.ownKeyOnly">{{ what || '运行' }}使用你自己的 HiveGPT Key，按实际用量从这个 Key 计费。选一个 GPT 分组的 Key：</p>
    <p v-else>今天的免费{{ what || '运行' }}次数用完了。可以用你自己的 Key 继续，按正常价格从这个 Key 计费；也可以明天再来。</p>
    <div v-if="keys && keys.length" class="runbox-actions">
      <select v-model="picked" class="ownkey-select">
        <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
      </select>
      <button class="runbox-btn" @click="use">{{ runMode.ownKeyOnly ? '用这个 Key 运行' : '用这个 Key 继续' }}</button>
    </div>
    <p v-else-if="keys" class="runbox-note">
      你还没有可用的 GPT 分组 Key，<a :href="`${MAIN_SITE}/keys`" target="_blank" rel="noopener">去创建一个</a>（分组选「GPT-按量」），创建后回到这里刷新页面。
    </p>
  </div>
</template>
