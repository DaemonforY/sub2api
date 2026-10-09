<template>
  <div class="art-ready-card" :class="{ compact }">
    <div class="art-notice-title">✅ 文章写好了</div>
    <div class="art-notice-text">
      「{{ project.title }}」，{{ okImages }} 张图{{ failedImages ? `（${failedImages} 张没画成）` : '' }}。
      <template v-if="project.pushed_at">已经推送过一次。</template>
    </div>
    <template v-if="account">
      <div class="art-notice-text strong">推送到「{{ account.name || account.appid }}」的草稿箱？</div>
      <div class="ed-row end">
        <button class="ed-btn small" @click="$emit('open')">先在编辑器里看看</button>
        <button class="ed-btn wechat small" @click="$emit('push')">一键推送</button>
      </div>
      <div class="ed-hint">会用编辑器当前的排版风格；编辑器里原来的内容先存进「历史」。</div>
    </template>
    <template v-else>
      <div class="art-notice-text">添加公众号后可以一键推送到草稿箱。</div>
      <div class="ed-row end">
        <button class="ed-btn small" @click="$emit('open')">在编辑器里打开</button>
        <button class="ed-btn primary small" @click="$emit('open-settings')">添加公众号</button>
      </div>
    </template>
  </div>
</template>

<script>
import { loadAccounts } from '../lib/settings.js';

export default {
  name: 'ArticleReadyCard',
  props: {
    project: { type: Object, required: true },
    compact: { type: Boolean, default: false }
  },
  emits: ['push', 'open', 'open-settings'],
  computed: {
    account() {
      const acc = loadAccounts();
      return acc.accounts.find(a => a.id === acc.defaultId) || acc.accounts[0] || null;
    },
    okImages() {
      return (this.project.images || []).filter(im => im.status === 'ok').length;
    },
    failedImages() {
      return (this.project.images || []).filter(im => im.status !== 'ok').length;
    }
  }
};
</script>
