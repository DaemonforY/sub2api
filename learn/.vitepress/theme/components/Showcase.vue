<script setup lang="ts">
// The learner wall: certificate holders who chose to show their project, with covers of their
// public canvas works. `compact` is the strip on the learning home page.
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'
import { showcase, type ShowcaseItem } from '../api'
import { findTrack, tracks } from '../tracks'

const props = withDefaults(defineProps<{ compact?: boolean; limit?: number }>(), { compact: false, limit: 0 })
const items = ref<ShowcaseItem[] | null>(null)
const filter = ref('')
const failed = ref(false)

async function load() {
  failed.value = false
  try {
    items.value = await showcase(filter.value, props.limit)
  } catch {
    failed.value = true
    items.value = []
  }
}

onMounted(load)

function pick(track: string) {
  filter.value = track
  void load()
}

function host(url: string) {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

const empty = computed(() => items.value !== null && items.value.length === 0)
</script>

<template>
  <div v-if="!(compact && (empty || items === null))" :class="['showcase', { compact }]" data-testid="showcase">
    <slot name="head" />
    <div v-if="!compact" class="showcase-filter">
      <button :class="['showcase-tab', { on: !filter }]" @click="pick('')">全部</button>
      <button v-for="t in tracks" :key="t.id" :class="['showcase-tab', { on: filter === t.id }]" @click="pick(t.id)">{{ t.letter }} · {{ t.title }}</button>
    </div>
    <p v-if="items === null" class="runbox-note">加载中…</p>
    <p v-else-if="failed" class="runbox-note">作品墙暂时打不开，请稍后再试。</p>
    <p v-else-if="empty" class="runbox-note">这里还没有作品。学完一条路线、领取证书时勾选「展示到学员作品墙」，你的项目就会出现在这里。</p>
    <div v-else class="showcase-grid">
      <div v-for="it in items" :key="it.code" class="showcase-card" data-testid="showcase-card">
        <div v-if="it.works.length" :class="['showcase-imgs', `n${it.works.length}`]">
          <a v-for="w in it.works" :key="w.url" :href="w.url" target="_blank" rel="noopener" :title="w.title">
            <img :src="w.image" :alt="w.title || '作品'" loading="lazy" />
          </a>
        </div>
        <div v-else class="showcase-cover" :style="{ background: findTrack(it.track)?.color }">
          <span>{{ findTrack(it.track)?.letter }}</span>
        </div>
        <div class="showcase-body">
          <strong>{{ it.display_name }}</strong>
          <span class="runbox-note">{{ it.track_title }} · 结业</span>
          <a v-if="it.project_url" class="showcase-link" :href="it.project_url" target="_blank" rel="noopener nofollow ugc">{{ host(it.project_url) }} ↗</a>
          <a class="showcase-cert" :href="withBase(`/cert.html?c=${it.code}`)">查看证书</a>
        </div>
      </div>
    </div>
  </div>
</template>
