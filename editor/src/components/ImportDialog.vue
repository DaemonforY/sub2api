<template>
  <div class="ed-overlay" @click.self="close">
    <div class="ed-dialog" role="dialog" aria-label="导入公众号文章">
      <div class="ed-dialog-header">
        <h3>导入公众号文章</h3>
        <button class="ed-close" title="关闭" :disabled="busy" @click="close">×</button>
      </div>

      <div class="ed-dialog-body">
        <div v-if="!editor.auth.loggedIn" class="ed-login-box">
          导入文章需要先登录 HiveGPT。<br>
          <a class="ed-btn primary" :href="editor.loginHref">登录</a>
        </div>
        <template v-else>
          <div class="ed-field">
            <label for="ed-import-url">文章链接</label>
            <input
              id="ed-import-url"
              ref="urlInput"
              v-model.trim="url"
              class="ed-input"
              placeholder="https://mp.weixin.qq.com/s/..."
              :disabled="busy"
              spellcheck="false"
              @keydown.enter="run"
            >
          </div>
          <p class="ed-hint">
            只支持 mp.weixin.qq.com 的文章链接。正文会转成 Markdown，图片会下载并压缩后保存在本浏览器里（不会上传到别处），文章标题会放进草稿标题。导入会替换编辑器里的当前内容，当前内容会先存入「历史」。
          </p>
          <div v-if="status" class="ed-msg" :class="statusType">
            <span v-if="busy" class="ed-spinner"></span>
            {{ status }}
          </div>
        </template>
      </div>

      <div class="ed-dialog-footer">
        <button class="ed-btn" :disabled="busy" @click="close">取消</button>
        <button v-if="editor.auth.loggedIn" class="ed-btn primary" :disabled="busy || !url" @click="run">
          {{ busy ? '导入中…' : '导入' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { importArticle, fetchProxiedImage, isWechatImageUrl } from '../lib/api.js';

const ARTICLE_URL = /^https?:\/\/mp\.weixin\.qq\.com\/.+/i;
const MD_IMAGE = /!\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)/g;

export default {
  name: 'ImportDialog',
  inject: ['editor'],
  emits: ['close'],
  data() {
    return {
      url: '',
      busy: false,
      status: '',
      statusType: 'info'
    };
  },
  mounted() {
    this.$nextTick(() => this.$refs.urlInput && this.$refs.urlInput.focus());
  },
  methods: {
    close() {
      if (!this.busy) this.$emit('close');
    },
    setStatus(text, type = 'info') {
      this.status = text;
      this.statusType = type;
    },
    async run() {
      if (this.busy || !this.url) return;
      if (!ARTICLE_URL.test(this.url)) {
        this.setStatus('请输入 mp.weixin.qq.com 的文章链接', 'err');
        return;
      }
      if (!this.editor.turndownService) {
        this.setStatus('HTML 转 Markdown 组件未就绪，请刷新页面后重试', 'err');
        return;
      }

      this.busy = true;
      try {
        this.setStatus('正在获取文章…');
        const article = await importArticle(this.url);

        let markdown = this.editor.turndownService.turndown(article.html || '');
        markdown = markdown.replace(/\n{3,}/g, '\n\n').trim();

        // 正文里的公众号图片：经服务器代理下载 → 压缩 → 存入 IndexedDB → 换成 img:// 短链接
        const urls = [];
        for (const m of markdown.matchAll(MD_IMAGE)) {
          if (isWechatImageUrl(m[1]) && !urls.includes(m[1])) urls.push(m[1]);
        }
        let failed = 0;
        for (let i = 0; i < urls.length; i++) {
          const text = `正在导入图片 ${i + 1}/${urls.length}`;
          this.setStatus(text);
          this.editor.showToast(text, 'processing');
          try {
            const blob = await fetchProxiedImage(urls[i]);
            const id = await this.editor.storeImageBlob(blob, `公众号图片-${i + 1}`);
            markdown = markdown.split(urls[i]).join(`img://${id}`);
          } catch (error) {
            if (error && error.status === 401) throw error;
            console.warn('导入图片失败:', urls[i], error);
            failed++;
          }
        }

        if (this.editor.markdownInput.trim()) {
          this.editor.saveToHistory();
        }
        this.editor.markdownInput = markdown;
        this.editor.currentArticleId = null;
        this.editor.setDraftTitle(article.title || '', 'import');
        if (article.author) this.editor.draftMeta.author = article.author;

        if (failed) {
          this.editor.showToast(`导入完成，有 ${failed} 张图片下载失败（保留了原链接，公众号图片有防盗链，预览里可能显示不出来）`, 'error');
        } else {
          const imgNote = urls.length ? `，${urls.length} 张图片已保存到本地` : '';
          this.editor.showToast(`导入完成${imgNote}`, 'success');
        }
        this.busy = false;
        this.$emit('close');
      } catch (error) {
        this.editor.noteApiError(error);
        this.setStatus(error.message || '导入失败', 'err');
        this.busy = false;
      }
    }
  }
};
</script>
