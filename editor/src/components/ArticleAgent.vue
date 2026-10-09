<template>
  <aside class="ai-drawer article-drawer" :class="{ open }" aria-label="AI 写文章">
    <div class="ai-drawer-header">
      <h3>
        <button v-if="view !== 'list'" class="art-back" title="返回列表" @click="backToList">‹</button>
        {{ view === 'new' ? '写一篇新文章' : view === 'project' && current ? (current.title || '文章') : 'AI 写文章' }}
      </h3>
      <button class="ed-close" title="关闭" @click="$emit('close')">×</button>
    </div>

    <div class="ai-body">
      <div v-if="!editor.auth.loggedIn" class="ed-login-box">
        AI 写文章需要先登录 HiveGPT，并使用你自己的 Key。<br>
        <a class="ed-btn primary" :href="editor.loginHref">登录</a>
      </div>
      <div v-else-if="!editor.aiKey" class="ed-login-box">
        还没有选择 AI Key。<br>
        写文章和配图的费用记在你选择的 Key 上，请先在「设置」里选一个。<br>
        <button class="ed-btn primary" @click="$emit('open-settings')">打开设置</button>
      </div>

      <!-- 列表 -->
      <template v-else-if="view === 'list'">
        <p class="ed-hint" style="margin-top: 0;">
          说出主题，AI 先{{ config.search ? '搜资料、' : '' }}出大纲给你确认，再写全文、画封面和配图。在后台进行，关掉页面也会继续；写好后点一下就能推送到公众号草稿箱。
        </p>
        <button class="ed-btn primary art-wide" @click="view = 'new'">＋ 写一篇新文章</button>
        <div v-if="listError" class="ed-msg err">{{ listError }}</div>
        <ul class="art-list">
          <li v-for="p in list" :key="p.id" @click="openProject(p.id)">
            <div class="art-list-title">{{ p.title || p.brief.topic }}</div>
            <div class="art-list-meta">
              <span class="art-badge" :class="badgeClass(p)">{{ statusText(p) }}</span>
              <span>{{ shortTime(p.created_at) }}</span>
            </div>
          </li>
        </ul>
        <p v-if="!list.length && !listLoading" class="ed-hint" style="text-align: center;">还没有文章</p>
      </template>

      <!-- 新建 -->
      <template v-else-if="view === 'new'">
        <div class="ed-field">
          <div class="ed-label-row">
            <label for="art-topic">主题</label>
            <span class="ed-counter">{{ form.topic.length }}/200</span>
          </div>
          <textarea id="art-topic" v-model="form.topic" class="ed-textarea" rows="2" maxlength="200" placeholder="例如：国庆出游怎么省钱；给新手的 Codex 入门；我们学校的 AI 课程上线了"></textarea>
        </div>
        <div class="ed-field">
          <div class="ed-label-row">
            <label for="art-materials">参考资料（可不填）</label>
            <span class="ed-counter">{{ form.materials.length }}/8000</span>
          </div>
          <textarea id="art-materials" v-model="form.materials" class="ed-textarea" rows="4" maxlength="8000" placeholder="贴上你的笔记、要点、数据或活动信息，AI 会优先用这些，不会乱编"></textarea>
        </div>
        <div class="art-grid2">
          <div class="ed-field">
            <label for="art-audience">读者</label>
            <input id="art-audience" v-model="form.audience" class="ed-input" maxlength="100" placeholder="如：大学生、家长、程序员">
          </div>
          <div class="ed-field">
            <label for="art-tone">风格</label>
            <input id="art-tone" v-model="form.tone" class="ed-input" maxlength="100" placeholder="如：轻松口语、专业严谨">
          </div>
        </div>
        <div class="ed-field">
          <span class="ed-label">篇幅</span>
          <div class="ai-segment">
            <button v-for="l in lengths" :key="l.id" :class="{ active: form.length === l.id }" @click="form.length = l.id">{{ l.label }}</button>
          </div>
        </div>
        <div class="ed-field">
          <span class="ed-label">正文配图（另有一张封面）</span>
          <div class="ai-segment">
            <button v-for="n in imageCounts" :key="n" :class="{ active: form.images === n }" @click="form.images = n">{{ n }} 张</button>
          </div>
        </div>
        <label class="ed-check-line" :class="{ disabled: !config.search }">
          <input v-model="form.search" type="checkbox" :disabled="!config.search">
          联网搜索最新资料
          <span v-if="!config.search" class="ed-counter">（本站暂未开启联网搜索）</span>
        </label>
        <div class="ed-hint art-cost">
          <template v-if="config.image_price">
            配图每张约 <b>{{ money(config.image_price) }}</b>（Key「{{ editor.aiKey.name }}」所在分组的价格），这篇共 {{ form.images + 1 }} 张（含封面）约 <b>{{ money(config.image_price * (form.images + 1)) }}</b>。
            另有出大纲、写全文的对话费用，按实际用量扣费。
          </template>
          <template v-else>
            费用记在 Key「{{ editor.aiKey.name }}」上：出大纲和写全文各约 1 次对话，另画 {{ form.images + 1 }} 张图（gpt-image-2，含封面）。
          </template>
        </div>
        <div v-if="formError" class="ed-msg err">{{ formError }}</div>
        <div class="ed-row end" style="margin-top: 12px;">
          <button class="ed-btn" @click="view = 'list'">取消</button>
          <button class="ed-btn primary" :disabled="creating || !form.topic.trim()" @click="create">
            <span v-if="creating" class="ed-spinner"></span> 开始
          </button>
        </div>
      </template>

      <!-- 一篇文章 -->
      <template v-else-if="view === 'project' && current">
        <div class="art-status">
          <span class="art-badge" :class="badgeClass(current)">{{ statusText(current) }}</span>
          <span v-if="active" class="ed-counter"><span class="ed-spinner"></span> {{ lastStep }}</span>
        </div>

        <!-- 写好了：推送卡片 -->
        <div v-if="current.status === 'done'" class="art-ready">
          <ArticleReadyCard :project="current" @push="push(current)" @open="openInEditor(current)" @open-settings="$emit('open-settings')" />
        </div>

        <div v-if="current.status === 'failed' || current.status === 'canceled'" class="ed-msg err">
          {{ current.status === 'canceled' ? '已停止。' : current.error }}
        </div>

        <!-- 大纲 -->
        <template v-if="current.status === 'outline_ready' && outline">
          <div class="ed-field">
            <span class="ed-label">标题（点选或直接改）</span>
            <div class="art-titles">
              <button v-for="t in outline.titles" :key="t" class="ai-chip" :class="{ active: outline.title === t }" @click="outline.title = t">{{ t }}</button>
            </div>
            <input v-model="outline.title" class="ed-input" maxlength="64">
          </div>
          <div class="ed-field">
            <label for="art-digest">摘要</label>
            <textarea id="art-digest" v-model="outline.digest" class="ed-textarea" rows="2" maxlength="120"></textarea>
          </div>
          <div class="ed-field">
            <span class="ed-label">小节</span>
            <div v-for="(s, i) in outline.sections" :key="i" class="art-section">
              <div class="ed-row">
                <span class="art-num">{{ i + 1 }}</span>
                <input v-model="s.heading" class="ed-input" maxlength="60" placeholder="小标题">
                <button class="ed-close" title="删掉这一节" @click="outline.sections.splice(i, 1)">×</button>
              </div>
              <textarea v-model="s.pointsText" class="ed-textarea" rows="2" placeholder="要点，每行一条"></textarea>
              <div v-if="s.image" class="art-image-line">
                🖼 <input v-model="s.image.prompt" class="ed-input" maxlength="600" placeholder="配图画面描述">
                <button class="ed-close" title="这一节不配图" @click="s.image = null">×</button>
              </div>
              <button v-else-if="imagesInOutline < current.brief.images" class="ed-btn small" @click="s.image = { prompt: s.heading, alt: '' }">＋ 配图</button>
            </div>
            <button class="ed-btn small" @click="outline.sections.push({ heading: '', pointsText: '', image: null })">＋ 加一节</button>
          </div>
          <div class="ed-field">
            <label for="art-cover">封面画面</label>
            <input id="art-cover" v-model="outline.cover_prompt" class="ed-input" maxlength="600">
          </div>
          <div v-if="current.sources.length" class="ed-field">
            <span class="ed-label">搜到的资料（{{ current.sources.length }}）</span>
            <ul class="art-sources">
              <li v-for="s in current.sources" :key="s.url"><a :href="s.url" target="_blank" rel="noopener noreferrer">{{ s.title || s.url }}</a></li>
            </ul>
          </div>
          <div class="ed-field">
            <label for="art-feedback">不满意？写下意见让 AI 重写大纲</label>
            <textarea id="art-feedback" v-model="feedback" class="ed-textarea" rows="2" maxlength="500" placeholder="例如：第二节太空泛，加一个具体案例；标题再口语一点"></textarea>
          </div>
          <div v-if="actionError" class="ed-msg err">{{ actionError }}</div>
          <div class="ed-row end">
            <button class="ed-btn" :disabled="acting || !feedback.trim()" @click="redoOutline">按意见重写</button>
            <button class="ed-btn primary" :disabled="acting" @click="confirmOutline">
              <span v-if="acting" class="ed-spinner"></span> 确认大纲，写全文
            </button>
          </div>
        </template>

        <!-- 正文预览 -->
        <div v-if="current.markdown && current.status !== 'outline_ready'" class="ed-field">
          <span class="ed-label">正文{{ current.status === 'writing' ? '（写作中）' : '' }}</span>
          <pre class="art-preview">{{ current.markdown }}</pre>
        </div>

        <!-- 配图 -->
        <div v-if="current.images.length" class="ed-field">
          <span class="ed-label">封面和配图</span>
          <div class="art-images">
            <figure v-for="im in current.images" :key="im.n">
              <img v-if="thumbs[im.n]" :src="thumbs[im.n]" :alt="im.alt">
              <div v-else class="art-image-wait">
                <span v-if="im.status === 'pending' && active" class="ed-spinner"></span>
                <span v-else-if="im.status === 'failed'" :title="im.error">✕ 没画成</span>
              </div>
              <figcaption>{{ im.kind === 'cover' ? '封面' : `配图 ${im.n}` }}</figcaption>
            </figure>
          </div>
        </div>

        <!-- 进度 -->
        <details class="art-events" :open="active">
          <summary>进度记录</summary>
          <ul>
            <li v-for="(e, i) in current.events" :key="i" :class="e.kind">{{ shortTime(e.at) }} {{ e.text }}</li>
          </ul>
        </details>

        <div class="ed-row end" style="margin-top: 10px;">
          <button v-if="active" class="ed-btn" :disabled="acting" @click="cancel">停止</button>
          <button v-if="canRetry" class="ed-btn primary" :disabled="acting" @click="retry">重试</button>
          <button v-if="!active" class="ed-btn danger" :disabled="acting" @click="remove">删除</button>
        </div>
      </template>
    </div>
  </aside>

  <!-- 后台任务完成时的提醒（面板关着也会弹） -->
  <div v-if="notice" class="art-notice" role="status">
    <button class="ed-close art-notice-close" title="关闭" @click="notice = null">×</button>
    <template v-if="notice.status === 'done'">
      <ArticleReadyCard :project="notice" compact @push="push(notice)" @open="openInEditor(notice)" @open-settings="$emit('open-settings')" />
    </template>
    <template v-else-if="notice.status === 'outline_ready'">
      <div class="art-notice-title">📝 大纲好了</div>
      <div class="art-notice-text">「{{ notice.title }}」的大纲已经出来了，确认后开始写全文。</div>
      <div class="ed-row end"><button class="ed-btn primary small" @click="showProject(notice.id)">去确认</button></div>
    </template>
    <template v-else>
      <div class="art-notice-title">⚠️ 没写成</div>
      <div class="art-notice-text">「{{ notice.title }}」：{{ notice.error || '生成失败' }}</div>
      <div class="ed-row end"><button class="ed-btn small" @click="showProject(notice.id)">查看</button></div>
    </template>
  </div>
</template>

<script>
import ArticleReadyCard from './ArticleReadyCard.vue';
import { articleConfig, listArticles, createArticle, getArticle, articleOutline, retryArticle, cancelArticle, deleteArticle, fetchArticleImage } from '../lib/api.js';

const ACTIVE = ['outlining', 'writing', 'drawing'];
const STATUS = {
  outlining: '出大纲中',
  outline_ready: '待确认大纲',
  writing: '写作中',
  drawing: '画图中',
  done: '已完成',
  failed: '失败',
  canceled: '已停止'
};

export default {
  name: 'ArticleAgent',
  components: { ArticleReadyCard },
  inject: ['editor'],
  props: { open: { type: Boolean, default: false } },
  emits: ['close', 'open-settings', 'open'],
  data() {
    return {
      config: { search: false, max_images: 4, default_images: 1, image_price: null },
      view: 'list',
      list: [],
      listLoading: false,
      listError: '',
      form: { topic: '', materials: '', audience: '', tone: '', length: 'standard', images: 1, search: false },
      lengths: [{ id: 'short', label: '短 · 约 1000 字' }, { id: 'standard', label: '中 · 约 2000 字' }, { id: 'long', label: '长 · 约 3000 字' }],
      formError: '',
      creating: false,
      current: null,
      outline: null,
      feedback: '',
      acting: false,
      actionError: '',
      thumbs: {},
      notice: null,
      watched: {} // id → last known status, for the background reminders
    };
  },
  computed: {
    imageCounts() {
      return Array.from({ length: (this.config.max_images || 4) + 1 }, (_, i) => i);
    },
    active() {
      return !!this.current && ACTIVE.includes(this.current.status);
    },
    canRetry() {
      const p = this.current;
      if (!p) return false;
      return p.status === 'failed' || p.status === 'canceled' || (p.status === 'done' && p.images.some(im => im.status !== 'ok'));
    },
    lastStep() {
      const ev = this.current && [...this.current.events].reverse().find(e => e.kind === 'step');
      return ev ? ev.text : '处理中…';
    },
    imagesInOutline() {
      return this.outline ? this.outline.sections.filter(s => s.image).length : 0;
    }
  },
  watch: {
    open(v) {
      if (v) {
        this.loadConfig();
        if (this.view === 'list') this.loadList();
      }
    },
    'editor.aiKey.id'() {
      if (this.open) this.loadConfig();
    },
    'editor.auth.loggedIn': {
      immediate: true,
      handler(v) {
        if (v) this.watchExisting();
      }
    }
  },
  mounted() {
    this._timer = setInterval(() => this.tick(), 3000);
    this._ticks = 0;
    this._title = document.title;
    // Back on the tab: drop the 写好了 title and check right away.
    this._onVisible = () => {
      if (document.hidden) return;
      document.title = this._title;
      this._ticks = 3;
      this.tick();
    };
    document.addEventListener('visibilitychange', this._onVisible);
  },
  beforeUnmount() {
    clearInterval(this._timer);
    document.removeEventListener('visibilitychange', this._onVisible);
    this.dropThumbs();
  },
  methods: {
    statusText(p) {
      if (p.status === 'done' && p.pushed_at) return '已推送';
      return STATUS[p.status] || p.status;
    },
    badgeClass(p) {
      if (ACTIVE.includes(p.status)) return 'busy';
      if (p.status === 'outline_ready') return 'wait';
      if (p.status === 'done') return 'ok';
      return 'bad';
    },
    shortTime(iso) {
      const d = new Date(iso);
      const pad = n => String(n).padStart(2, '0');
      return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    },
    money(v) {
      return `$${v >= 1 ? v.toFixed(2) : v.toFixed(3).replace(/0$/, '')}`;
    },
    async loadConfig() {
      try {
        this.config = await articleConfig(this.editor.aiKey && this.editor.aiKey.id);
        if (!this.config.search) this.form.search = false;
      } catch {
        // keep defaults
      }
    },
    async loadList() {
      if (!this.editor.auth.loggedIn) return;
      this.listLoading = true;
      this.listError = '';
      try {
        this.list = await listArticles();
        for (const p of this.list) this.watched[p.id] = p.status;
      } catch (error) {
        this.listError = error.message;
      } finally {
        this.listLoading = false;
      }
    },
    /** On load: remember articles still running, so their outcome is announced even with the panel closed. */
    async watchExisting() {
      try {
        const list = await listArticles();
        for (const p of list) this.watched[p.id] = p.status;
        this.list = list;
      } catch {
        // not fatal
      }
    },
    backToList() {
      this.view = 'list';
      this.current = null;
      this.outline = null;
      this.dropThumbs();
      this.loadList();
    },
    async create() {
      this.formError = '';
      this.creating = true;
      try {
        const p = await createArticle({ key_id: this.editor.aiKey.id, ...this.form, topic: this.form.topic.trim() });
        this.watched[p.id] = p.status;
        this.form.topic = '';
        this.form.materials = '';
        await this.openProject(p.id);
      } catch (error) {
        this.editor.noteApiError(error);
        this.formError = error.message;
      } finally {
        this.creating = false;
      }
    },
    /** Opens the panel on one article (from a reminder). */
    showProject(id) {
      this.notice = null;
      this.$emit('open');
      this.openProject(id);
    },
    async openProject(id) {
      this.dropThumbs();
      this.feedback = '';
      this.actionError = '';
      this.view = 'project';
      await this.refresh(id);
    },
    async refresh(id) {
      try {
        const p = await getArticle(id);
        this.applyProject(p);
      } catch (error) {
        this.actionError = error.message;
      }
    },
    applyProject(p) {
      const before = this.current && this.current.id === p.id ? this.current.status : null;
      this.current = p;
      this.watched[p.id] = p.status;
      if (p.status === 'outline_ready' && (before !== 'outline_ready' || !this.outline)) {
        this.outline = JSON.parse(JSON.stringify(p.outline));
        for (const s of this.outline.sections) {
          s.pointsText = (s.points || []).join('\n');
          s.image = s.image || null;
        }
      }
      if (p.status !== 'outline_ready') this.outline = null;
      this.loadThumbs(p);
    },
    async loadThumbs(p) {
      for (const im of p.images) {
        if (im.status !== 'ok' || this.thumbs[im.n]) continue;
        this.thumbs[im.n] = 'loading';
        try {
          const blob = await fetchArticleImage(p.id, im.n);
          if (!this.current || this.current.id !== p.id) return;
          this.thumbs[im.n] = URL.createObjectURL(blob);
        } catch {
          delete this.thumbs[im.n];
        }
      }
    },
    dropThumbs() {
      for (const url of Object.values(this.thumbs)) {
        if (url && url !== 'loading') URL.revokeObjectURL(url);
      }
      this.thumbs = {};
    },
    outlinePayload() {
      const o = this.outline;
      return {
        titles: o.titles,
        title: o.title,
        digest: o.digest,
        cover_prompt: o.cover_prompt,
        sections: o.sections.map(s => ({
          heading: s.heading,
          points: (s.pointsText || '').split('\n').map(x => x.trim()).filter(Boolean),
          image: s.image && s.image.prompt.trim() ? { prompt: s.image.prompt, alt: s.image.alt || s.heading } : undefined
        }))
      };
    },
    async act(fn) {
      this.acting = true;
      this.actionError = '';
      try {
        const p = await fn();
        if (p && p.id) this.applyProject(p);
        else if (this.current) await this.refresh(this.current.id);
      } catch (error) {
        this.editor.noteApiError(error);
        this.actionError = error.message;
      } finally {
        this.acting = false;
      }
    },
    redoOutline() {
      this.act(() => articleOutline(this.current.id, { outline: this.outlinePayload(), feedback: this.feedback.trim() }));
      this.feedback = '';
    },
    confirmOutline() {
      this.act(() => articleOutline(this.current.id, { outline: this.outlinePayload(), confirm: true }));
    },
    retry() {
      this.act(() => retryArticle(this.current.id));
    },
    cancel() {
      this.act(() => cancelArticle(this.current.id));
    },
    async remove() {
      if (!window.confirm('删除这篇文章和它的配图？（已经载入编辑器或推送到草稿箱的内容不受影响）')) return;
      await this.act(() => deleteArticle(this.current.id));
      if (!this.actionError) this.backToList();
    },
    push(p) {
      this.notice = null;
      this.editor.openAgentArticle(p.id, { push: true });
    },
    openInEditor(p) {
      this.notice = null;
      this.editor.openAgentArticle(p.id, { push: false });
    },
    /**
     * Every 3 s: refresh the open article while it runs (tab visible). Every 12 s, even with the tab
     * in the background, check the articles still running so their outcome is announced.
     */
    async tick() {
      if (!this.editor.auth.loggedIn) return;
      this._ticks++;
      if (!document.hidden && this.open && this.view === 'project' && this.active) await this.refresh(this.current.id);
      if (this._ticks % 4 !== 0) return;
      const running = Object.entries(this.watched).filter(([, s]) => ACTIVE.includes(s)).map(([id]) => Number(id));
      for (const id of running) {
        if (!document.hidden && this.open && this.current && this.current.id === id) continue;
        try {
          const p = await getArticle(id);
          this.watched[id] = p.status;
          if (!ACTIVE.includes(p.status)) this.announce(p);
        } catch {
          delete this.watched[id];
        }
      }
    },
    /** Shows the reminder card, unless the user is already looking at that article. */
    announce(p) {
      if (document.hidden) {
        document.title = (p.status === 'done' ? '✅ 文章写好了 · ' : p.status === 'outline_ready' ? '📝 大纲好了 · ' : '⚠️ ') + this._title;
      }
      if (this.current && this.current.id === p.id) this.applyProject(p);
      if (this.open && !document.hidden && this.current && this.current.id === p.id) return;
      this.notice = p;
      const i = this.list.findIndex(x => x.id === p.id);
      if (i >= 0) this.list.splice(i, 1, { ...this.list[i], status: p.status, title: p.title });
    }
  }
};
</script>
