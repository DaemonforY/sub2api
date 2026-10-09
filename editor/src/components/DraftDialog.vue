<template>
  <div class="ed-overlay" @click.self="close">
    <div class="ed-dialog wide" role="dialog" aria-label="发送到草稿箱">
      <div class="ed-dialog-header">
        <h3>发送到公众号草稿箱</h3>
        <button class="ed-close" title="关闭" :disabled="phase === 'sending'" @click="close">×</button>
      </div>

      <!-- 未登录 -->
      <div v-if="!editor.auth.loggedIn" class="ed-dialog-body">
        <div class="ed-login-box">
          发送到草稿箱需要先登录 HiveGPT。<br>
          <a class="ed-btn primary" :href="editor.loginHref">登录</a>
        </div>
      </div>

      <!-- 没有公众号账号 -->
      <div v-else-if="!accounts.length" class="ed-dialog-body">
        <div class="ed-login-box">
          还没有添加公众号账号。先准备好 AppID、AppSecret，并把服务器 IP 加入白名单：
          <WechatSetupGuide open />
          <button class="ed-btn primary" @click="$emit('open-settings')">去设置里添加公众号</button>
        </div>
      </div>

      <!-- 填写 -->
      <div v-else-if="phase === 'form'" class="ed-dialog-body">
        <div class="ed-field">
          <label for="ed-draft-account">公众号</label>
          <select id="ed-draft-account" v-model="accountId" class="ed-select">
            <option v-for="a in accounts" :key="a.id" :value="a.id">{{ a.name || '未命名公众号' }}（{{ a.appid }}）</option>
          </select>
        </div>

        <div class="ed-field">
          <div class="ed-label-row">
            <label for="ed-draft-title">标题</label>
            <span class="ed-counter">{{ title.length }}/64</span>
          </div>
          <input id="ed-draft-title" v-model="title" class="ed-input" maxlength="64" placeholder="文章标题">
          <label v-if="h1Text" class="ed-check-line">
            <input v-model="stripH1" type="checkbox">
            正文去掉第一个一级标题（「{{ h1Short }}」）
          </label>
        </div>

        <div class="ed-field">
          <label for="ed-draft-author">作者</label>
          <input id="ed-draft-author" v-model.trim="author" class="ed-input" maxlength="16" placeholder="可不填">
        </div>

        <div class="ed-field">
          <div class="ed-label-row">
            <label for="ed-draft-digest">摘要</label>
            <span class="ed-row">
              <span class="ed-counter">{{ digest.length }}/120</span>
              <button class="ed-btn small" :disabled="digestBusy" @click="aiDigest">
                <span v-if="digestBusy" class="ed-spinner"></span>
                {{ digestBusy ? '生成中' : 'AI 生成' }}
              </button>
            </span>
          </div>
          <textarea id="ed-draft-digest" v-model="digest" class="ed-textarea" maxlength="120" rows="2" placeholder="可不填，公众号会自动截取正文开头"></textarea>
        </div>

        <!-- 封面 -->
        <div class="ed-field">
          <span class="ed-label">封面 <span class="ed-counter">（自动裁成 2.35:1，至少 900×383）</span></span>
          <div class="ed-cover-preview">
            <img v-if="coverUrl" :src="coverUrl" alt="封面预览">
            <span v-else-if="coverBusy"><span class="ed-spinner"></span> 正在处理封面…</span>
            <span v-else>还没有选择封面</span>
          </div>
          <div v-if="coverError" class="ed-msg err">{{ coverError }}</div>

          <div v-if="candidates.length" class="ed-hint" style="margin: 6px 0 4px;">从文章图片里选一张：</div>
          <div v-if="candidates.length" class="ed-cover-grid">
            <button
              v-for="c in candidates"
              :key="c.key"
              class="ed-cover-thumb"
              :class="{ active: selected === c.key }"
              :title="c.label"
              @click="pickCandidate(c)"
            >
              <img :src="c.thumb" :alt="c.label">
            </button>
          </div>
          <div class="ed-row">
            <label class="ed-btn small ed-file-btn">
              上传图片
              <input type="file" accept="image/*" @change="pickFile">
            </label>
            <button class="ed-btn small" @click="showAiCover = !showAiCover">AI 生成封面</button>
          </div>

          <div v-if="showAiCover" class="ed-subpanel">
            <div v-if="!editor.aiKey" class="ed-msg info">
              AI 生成封面需要先在「设置」里选择 AI Key。
              <button class="ed-btn small" @click="$emit('open-settings')">打开设置</button>
            </div>
            <template v-else>
              <div class="ed-label-row" style="margin-bottom: 4px;">
                <span class="ed-label">画面描述</span>
                <button class="ed-btn small" :disabled="promptBusy || aiCoverBusy" @click="aiCoverPrompt">
                  <span v-if="promptBusy" class="ed-spinner"></span>
                  {{ promptBusy ? '撰写中' : '根据文章写描述' }}
                </button>
              </div>
              <textarea v-model="aiPrompt" class="ed-textarea" rows="3" placeholder="描述想要的封面画面，例如：清晨的城市天际线，柔和的蓝紫色调，留出左侧空白"></textarea>
              <div class="ed-row" style="margin-top: 8px; justify-content: space-between;">
                <span class="ed-counter">横图 1536×1024 · 费用记在 Key「{{ editor.aiKey.name }}」上</span>
                <button class="ed-btn primary small" :disabled="aiCoverBusy || promptBusy || !aiPrompt.trim()" @click="aiCover">
                  <span v-if="aiCoverBusy" class="ed-spinner"></span>
                  {{ aiCoverBusy ? `生成中 ${aiElapsed}s` : '生成封面' }}
                </button>
              </div>
              <div v-if="aiCoverBusy" class="ed-hint" style="margin-top: 6px;">生成一张图通常要 30–60 秒，请稍候…</div>
            </template>
          </div>
        </div>

        <div class="ed-field">
          <label for="ed-draft-source">原文链接</label>
          <input id="ed-draft-source" v-model.trim="sourceUrl" class="ed-input" placeholder="可不填，填了会显示「阅读原文」" spellcheck="false">
        </div>

        <label class="ed-check-line">
          <input v-model="openComment" type="checkbox">
          打开留言
        </label>

        <div v-if="formError" class="ed-msg err" style="margin-top: 10px;">{{ formError }}</div>
      </div>

      <!-- 发送中 / 结果 -->
      <div v-else class="ed-dialog-body">
        <ul class="ed-steps">
          <li v-for="s in steps" :key="s.key" :class="s.status">
            <span class="ed-step-icon">
              <span v-if="s.status === 'running'" class="ed-spinner"></span>
              <template v-else-if="s.status === 'done'">✓</template>
              <template v-else-if="s.status === 'error'">✕</template>
              <template v-else>○</template>
            </span>
            {{ s.label }}<span v-if="s.detail">　{{ s.detail }}</span>
          </li>
        </ul>

        <div v-if="sendError" class="ed-msg err">{{ sendError }}</div>

        <div v-if="phase === 'done'" class="ed-success">
          <h4>已保存到草稿箱</h4>
          <div>草稿 media_id：<span class="ed-mono">{{ mediaId }}</span></div>
          <div>
            去公众号后台「内容与互动 → 草稿箱」查看并发布：
            <a class="ed-link" href="https://mp.weixin.qq.com/" target="_blank" rel="noopener noreferrer">打开公众号后台</a>
          </div>
          <div v-for="w in warnings" :key="w" class="ed-msg info" style="margin-top: 8px; text-align: left;">{{ w }}</div>
        </div>
      </div>

      <div class="ed-dialog-footer">
        <template v-if="editor.auth.loggedIn && accounts.length && phase === 'form'">
          <button class="ed-btn" @click="close">取消</button>
          <button class="ed-btn wechat" :disabled="coverBusy" @click="send">发送到草稿箱</button>
        </template>
        <template v-else-if="phase === 'sending'">
          <button class="ed-btn" disabled>发送中…</button>
        </template>
        <template v-else-if="phase === 'error'">
          <button class="ed-btn" @click="close">关闭</button>
          <button class="ed-btn primary" @click="backToForm">返回修改</button>
        </template>
        <template v-else>
          <button class="ed-btn primary" @click="close">{{ phase === 'done' ? '完成' : '关闭' }}</button>
        </template>
      </div>
    </div>
  </div>
</template>

<script>
import WechatSetupGuide from './WechatSetupGuide.vue';
import { loadAccounts, saveDraftMeta } from '../lib/settings.js';
import { wechatUpload, wechatDraft, streamAiText, aiImage } from '../lib/api.js';
import { compressUnder, cropToCover, b64ToBlob, extensionFor } from '../lib/imageTools.js';

export default {
  name: 'DraftDialog',
  components: { WechatSetupGuide },
  inject: ['editor'],
  props: {
    // 打开后直接发送（AI 写文章的「一键推送」）；表单不完整时停在表单上
    auto: { type: Boolean, default: false }
  },
  emits: ['close', 'open-settings', 'sent'],
  data() {
    const acc = loadAccounts();
    return {
      accounts: acc.accounts,
      accountId: acc.defaultId,
      phase: 'form', // form | sending | done | error
      title: '',
      stripH1: false,
      h1Text: '',
      author: '',
      digest: '',
      sourceUrl: '',
      openComment: false,
      digestBusy: false,
      candidates: [],
      selected: '',
      coverBlob: null,
      coverUrl: '',
      coverBusy: false,
      coverError: '',
      showAiCover: false,
      aiPrompt: '',
      promptBusy: false,
      aiCoverBusy: false,
      aiElapsed: 0,
      formError: '',
      steps: [],
      sendError: '',
      mediaId: '',
      warnings: []
    };
  },
  computed: {
    h1Short() {
      return this.h1Text.length > 20 ? this.h1Text.slice(0, 20) + '…' : this.h1Text;
    },
    account() {
      return this.accounts.find(a => a.id === this.accountId) || null;
    }
  },
  watch: {
    title(v) {
      this.editor.draftMeta.title = v;
      if (this.editor.draftMeta.titleSource === '' && v) this.editor.draftMeta.titleSource = 'manual';
    },
    author(v) { this.editor.draftMeta.author = v; },
    digest(v) { this.editor.draftMeta.digest = v; },
    sourceUrl(v) { this.editor.draftMeta.sourceUrl = v; },
    openComment(v) { this.editor.draftMeta.openComment = v; }
  },
  async mounted() {
    const meta = this.editor.draftMeta;
    this.h1Text = this.editor.firstH1Text();
    if (meta.title) {
      this.title = meta.title;
      this.stripH1 = !!this.h1Text && this.h1Text.trim() === meta.title.trim();
    } else if (this.h1Text) {
      this.title = this.h1Text.slice(0, 64);
      this.stripH1 = true;
    }
    this.author = meta.author || '';
    this.digest = meta.digest || '';
    this.sourceUrl = meta.sourceUrl || '';
    this.openComment = !!meta.openComment;

    if (this.editor.auth.loggedIn) {
      this.candidates = await this.editor.articleImageCandidates();
      const preset = meta.coverImageId ? this.candidates.find(c => c.imageId === meta.coverImageId) : null;
      if (preset) {
        await this.useCoverFrom(() => this.editor.getImageBlobBySrc(preset.src), preset.key);
      } else if (meta.coverImageId) {
        // 设为封面的 AI 配图已不在正文里，也照样用
        await this.useCoverFrom(() => this.editor.getImageBlobBySrc(`img://${meta.coverImageId}`), `img://${meta.coverImageId}`);
      }
      if (this.auto && this.accounts.length) this.send();
    }
  },
  beforeUnmount() {
    if (this.coverUrl) URL.revokeObjectURL(this.coverUrl);
    clearInterval(this._timer);
    if (this._abort) this._abort.abort();
  },
  methods: {
    close() {
      if (this.phase === 'sending') return;
      saveDraftMeta(this.editor.draftMeta);
      this.$emit('close');
    },
    backToForm() {
      this.phase = 'form';
      this.sendError = '';
    },

    // ---------- 封面 ----------
    async useCoverFrom(getBlob, key) {
      this.coverBusy = true;
      this.coverError = '';
      this.selected = key;
      try {
        const blob = await getBlob();
        if (!blob) throw new Error('这张图片读取不到（可能有跨域限制），请换一张或上传图片');
        const cropped = await cropToCover(blob);
        if (this.coverUrl) URL.revokeObjectURL(this.coverUrl);
        this.coverBlob = cropped;
        this.coverUrl = URL.createObjectURL(cropped);
        return true;
      } catch (error) {
        this.coverError = error.message || '封面处理失败';
        this.selected = '';
        return false;
      } finally {
        this.coverBusy = false;
      }
    },
    pickCandidate(c) {
      this.useCoverFrom(() => this.editor.getImageBlobBySrc(c.src), c.key).then(ok => {
        if (ok) this.editor.draftMeta.coverImageId = c.imageId || '';
      });
    },
    pickFile(event) {
      const file = event.target.files && event.target.files[0];
      event.target.value = '';
      if (!file) return;
      if (!file.type.startsWith('image/')) {
        this.coverError = '请选择图片文件';
        return;
      }
      this.useCoverFrom(async () => file, 'upload');
    },
    requireKey() {
      if (!this.editor.aiKey) {
        this.editor.showToast('请先在「设置」里选择 AI Key', 'error');
        return false;
      }
      return true;
    },
    async aiDigest() {
      if (!this.requireKey()) return;
      const text = this.editor.markdownInput.trim();
      if (!text) {
        this.editor.showToast('正文是空的', 'error');
        return;
      }
      this.digestBusy = true;
      const before = this.digest;
      this.digest = '';
      try {
        await streamAiText({ key_id: this.editor.aiKey.id, action: 'digest', text: text.slice(0, 20000), title: this.title || undefined },
          delta => { this.digest = (this.digest + delta).slice(0, 120); });
        this.digest = this.digest.trim();
      } catch (error) {
        this.editor.noteApiError(error);
        if (!this.digest) this.digest = before;
        this.editor.showToast(error.message || '生成摘要失败', 'error');
      } finally {
        this.digestBusy = false;
      }
    },
    async aiCoverPrompt() {
      if (!this.requireKey()) return;
      const text = this.editor.markdownInput.trim();
      if (!text) {
        this.editor.showToast('正文是空的', 'error');
        return;
      }
      this.promptBusy = true;
      this.aiPrompt = '';
      try {
        await streamAiText({ key_id: this.editor.aiKey.id, action: 'image_prompt', text: text.slice(0, 20000), title: this.title || undefined },
          delta => { this.aiPrompt += delta; });
        this.aiPrompt = this.aiPrompt.trim();
      } catch (error) {
        this.editor.noteApiError(error);
        this.editor.showToast(error.message || '生成描述失败', 'error');
      } finally {
        this.promptBusy = false;
      }
    },
    async aiCover() {
      if (!this.requireKey() || !this.aiPrompt.trim()) return;
      this.aiCoverBusy = true;
      this.aiElapsed = 0;
      const started = Date.now();
      this._timer = setInterval(() => { this.aiElapsed = Math.round((Date.now() - started) / 1000); }, 1000);
      try {
        const res = await aiImage(this.editor.aiKey.id, this.aiPrompt.trim(), '1536x1024');
        const blob = b64ToBlob(res.b64_json, res.mime || 'image/png');
        const ok = await this.useCoverFrom(async () => blob, 'ai');
        if (ok) {
          this.editor.draftMeta.coverImageId = '';
          this.editor.showToast('AI 封面已生成', 'success');
        }
      } catch (error) {
        this.editor.noteApiError(error);
        this.editor.showToast(error.message || '生成封面失败', 'error');
      } finally {
        clearInterval(this._timer);
        this.aiCoverBusy = false;
      }
    },

    // ---------- 发送 ----------
    validate() {
      if (!this.account) return '请选择公众号';
      if (!this.title.trim()) return '请填写标题';
      if (!this.editor.renderedContent) return '正文是空的';
      if (!this.coverBlob) return '请选择或生成一张封面（草稿必须有封面）';
      if (this.sourceUrl && !/^https?:\/\//i.test(this.sourceUrl)) return '原文链接需要以 http:// 或 https:// 开头';
      return '';
    },
    setStep(key, status, detail) {
      const s = this.steps.find(x => x.key === key);
      if (!s) return;
      s.status = status;
      if (detail !== undefined) s.detail = detail;
    },
    async send() {
      this.formError = this.validate();
      if (this.formError) return;

      const account = { ...this.account };
      saveDraftMeta(this.editor.draftMeta);
      this.steps = [
        { key: 'images', label: '上传图片', status: 'pending', detail: '' },
        { key: 'cover', label: '上传封面', status: 'pending', detail: '' },
        { key: 'draft', label: '创建草稿', status: 'pending', detail: '' }
      ];
      this.sendError = '';
      this.warnings = [];
      this.phase = 'sending';
      let current = 'images';

      try {
        // 1. 正文图片逐张上传到公众号（media/uploadimg，jpg/png < 1MB）
        this.setStep('images', 'running');
        const built = await this.editor.buildWechatHTML({
          mode: 'upload',
          stripFirstH1: this.stripH1 && !!this.h1Text,
          onImageProgress: (i, n) => this.setStep('images', 'running', `${i}/${n}`),
          uploadImage: async (img) => {
            const blob = await this.editor.getImageBlobForElement(img);
            if (!blob) return null;
            const small = await compressUnder(blob);
            const res = await wechatUpload(account, 'content', small, `image.${extensionFor(small)}`);
            if (!res || !res.url) throw new Error('上传图片失败：没有返回图片地址');
            return res.url;
          }
        });
        const { stats } = built;
        this.setStep('images', 'done', stats.total ? `${stats.success}/${stats.total}` : '没有图片');
        if (stats.gif) this.warnings.push(`有 ${stats.gif} 张 GIF 动图没有上传（公众号正文图片接口只收 jpg/png），已替换为提示，请在公众号后台重新插入。`);
        if (stats.skipped) this.warnings.push(`有 ${stats.skipped} 张图片读取不到（外链图片可能有跨域限制），已从草稿里去掉，请在公众号后台补上。`);

        // 2. 封面（永久素材）
        current = 'cover';
        this.setStep('cover', 'running');
        const cover = await wechatUpload(account, 'cover', this.coverBlob, 'cover.jpg');
        if (!cover || !cover.media_id) throw new Error('上传封面失败：没有返回 media_id');
        this.setStep('cover', 'done');

        // 3. 草稿
        current = 'draft';
        this.setStep('draft', 'running');
        const draft = await wechatDraft(account, {
          title: this.title.trim(),
          author: this.author,
          digest: this.digest.trim(),
          content: built.html,
          content_source_url: this.sourceUrl,
          thumb_media_id: cover.media_id,
          need_open_comment: this.openComment,
          only_fans_can_comment: false
        });
        this.setStep('draft', 'done');
        this.mediaId = draft && draft.media_id ? draft.media_id : '';
        this.phase = 'done';
        this.$emit('sent');
        this.editor.saveToHistory();
        this.editor.showToast('已保存到草稿箱', 'success');
      } catch (error) {
        this.editor.noteApiError(error);
        this.setStep(current, 'error');
        this.sendError = error.message || '发送失败';
        this.phase = 'error';
      }
    }
  }
};
</script>
