<template>
  <aside class="ai-drawer article-drawer tietu-drawer" :class="{ open }" aria-label="AI 做贴图">
    <div class="ai-drawer-header">
      <h3>
        <button v-if="view !== 'list'" class="art-back" title="返回列表" @click="backToList">‹</button>
        {{ view === 'new' ? '做一条新贴图' : view === 'project' && current ? (current.title || '贴图') : 'AI 做贴图' }}
      </h3>
      <button class="ed-close" title="关闭" @click="$emit('close')">×</button>
    </div>

    <div class="ai-body">
      <div v-if="!editor.auth.loggedIn" class="ed-login-box">
        AI 做贴图需要先登录 HiveGPT，并使用你自己的 Key。<br>
        <a class="ed-btn primary" :href="editor.loginHref">登录</a>
      </div>
      <div v-else-if="!editor.aiKey" class="ed-login-box">
        还没有选择 AI Key。<br>
        构思和画图的费用记在你选择的 Key 上，请先在「设置」里选一个。<br>
        <button class="ed-btn primary" @click="$emit('open-settings')">打开设置</button>
      </div>
      <div v-else-if="denied" class="ed-login-box">AI 做贴图暂时只对管理员开放。</div>

      <!-- 列表 -->
      <template v-else-if="view === 'list'">
        <p class="ed-hint" style="margin-top: 0;">
          输入一首诗（或一个主题让 AI 推荐），AI 核对原文、构思画面细节和分镜给你确认；你挑封面后，它以封面为参考画出整组图，再逐张看图自检、写好标题和正文。在后台进行，关掉页面也会继续；做好后一键推送到公众号草稿箱（贴图）。
        </p>
        <button class="ed-btn primary art-wide" @click="view = 'new'">＋ 做一条新贴图</button>
        <div v-if="listError" class="ed-msg err">{{ listError }}</div>
        <ul class="art-list">
          <li v-for="p in list" :key="p.id" @click="openProject(p.id)">
            <div class="art-list-title">{{ p.title || p.brief.input }}</div>
            <div class="art-list-meta">
              <span class="art-badge" :class="badgeClass(p)">{{ statusText(p) }}</span>
              <span>{{ shortTime(p.created_at) }}</span>
            </div>
          </li>
        </ul>
        <p v-if="!list.length && !listLoading" class="ed-hint" style="text-align: center;">还没有贴图</p>
      </template>

      <!-- 新建 -->
      <template v-else-if="view === 'new'">
        <div class="ed-field">
          <div class="ai-segment">
            <button :class="{ active: !form.theme }" @click="form.theme = false">指定一首诗</button>
            <button :class="{ active: form.theme }" @click="form.theme = true">按主题推荐</button>
          </div>
        </div>
        <div class="ed-field">
          <div class="ed-label-row">
            <label for="tt-input">{{ form.theme ? '主题' : '篇名（可带作者）' }}</label>
            <span class="ed-counter">{{ form.input.length }}/100</span>
          </div>
          <input id="tt-input" v-model="form.input" class="ed-input" maxlength="100" :placeholder="form.theme ? '例如：冬天、乡村生活、节日热闹、少年意气' : '例如：桃花源记；游山西村 陆游；清平乐 村居'">
          <p class="ed-hint">适合画的诗：有人物和故事、场面大、相机拍不到（古人的生活、古代盛景）。红枫、桃花这类现实里能拍到的风景，画出来往往不吸引人。</p>
        </div>
        <div class="ed-field">
          <span class="ed-label">形式</span>
          <div class="ai-segment">
            <button v-for="m in modes" :key="m.id" :class="{ active: form.mode === m.id }" @click="form.mode = m.id">{{ m.label }}</button>
          </div>
        </div>
        <div class="ed-field">
          <span class="ed-label">封面之外再画几张</span>
          <div class="ai-segment">
            <button v-for="n in panelCounts" :key="n" :class="{ active: form.panels === n }" @click="form.panels = n">{{ n }}</button>
          </div>
        </div>
        <div class="ed-field">
          <label for="tt-style">画风</label>
          <select id="tt-style" v-model="form.style" class="ed-select">
            <option v-for="s in config.styles" :key="s.id" :value="s.id">{{ s.name }}</option>
            <option value="custom">自定义</option>
          </select>
          <textarea v-if="form.style === 'custom'" v-model="form.style_text" class="ed-textarea" rows="5" maxlength="2000" placeholder="贴上你的风格提示词（画风、构图、用色、禁忌）"></textarea>
          <details v-else-if="styleText" class="tt-fold">
            <summary>看看这个画风的提示词</summary>
            <pre class="art-preview">{{ styleText }}</pre>
          </details>
        </div>
        <div class="ed-field">
          <div class="ed-label-row">
            <label for="tt-notes">额外要求（可不填）</label>
            <span class="ed-counter">{{ form.notes.length }}/500</span>
          </div>
          <textarea id="tt-notes" v-model="form.notes" class="ed-textarea" rows="2" maxlength="500" placeholder="例如：主角画成小孩；重点画春社的热闹"></textarea>
        </div>
        <div class="ed-hint art-cost">
          <template v-if="config.image_price">
            每张图约 <b>{{ money(config.image_price) }}</b>（高质量竖图），这条共 {{ config.covers }} 张封面候选 + {{ form.panels }} 张，约 <b>{{ money(config.image_price * (config.covers + form.panels)) }}</b>；重画另计。另有核对原文、构思方案、看图自检的对话费用。
          </template>
          <template v-else>
            费用记在 Key「{{ editor.aiKey.name }}」上：画 {{ config.covers }} 张封面候选 + {{ form.panels }} 张图（gpt-image-2 高质量），另有构思和自检的对话费用。
          </template>
        </div>
        <div v-if="formError" class="ed-msg err">{{ formError }}</div>
        <div class="ed-row end" style="margin-top: 12px;">
          <button class="ed-btn" @click="view = 'list'">取消</button>
          <button class="ed-btn primary" :disabled="creating || !form.input.trim()" @click="create">
            <span v-if="creating" class="ed-spinner"></span> 开始
          </button>
        </div>
      </template>

      <!-- 一条贴图 -->
      <template v-else-if="view === 'project' && current">
        <div class="art-status">
          <span class="art-badge" :class="badgeClass(current)">{{ statusText(current) }}</span>
          <span v-if="active" class="ed-counter"><span class="ed-spinner"></span> {{ lastStep }}</span>
        </div>
        <div v-if="current.status === 'failed' || current.status === 'canceled'" class="ed-msg err">
          {{ current.status === 'canceled' ? '已停止。' : current.error }}
        </div>

        <!-- 推荐的诗 -->
        <div v-if="current.status === 'suggest_ready'" class="ed-field">
          <span class="ed-label">选一首继续</span>
          <div v-for="(s, i) in current.suggestions" :key="i" class="tt-card">
            <div class="tt-card-title">《{{ s.title }}》<small>{{ s.dynasty ? `〔${s.dynasty}〕` : '' }}{{ s.author }}</small></div>
            <div class="ed-hint">{{ s.reason }}</div>
            <div class="ed-row end"><button class="ed-btn primary small" :disabled="acting" @click="pick(i)">选这首</button></div>
          </div>
        </div>

        <!-- 原文 -->
        <details v-if="current.poem" class="tt-fold" :open="current.status === 'plan_ready'">
          <summary>原文：《{{ current.poem.title }}》〔{{ current.poem.dynasty }}〕{{ current.poem.author }} <span class="art-badge" :class="current.poem.agreed ? 'ok' : 'wait'">{{ current.poem.agreed ? '两个来源已核对' : '只有一个来源' }}</span></summary>
          <pre class="art-preview">{{ current.poem.text }}</pre>
          <ul v-if="current.poem.variants.length" class="tt-variants">
            <li v-for="(v, i) in current.poem.variants" :key="i">{{ v.context }}：{{ v.options.join(' / ') }} → <b>{{ v.chosen }}</b><span v-if="v.why">（{{ v.why }}）</span></li>
          </ul>
          <p v-if="current.poem.note" class="ed-hint">⚠️ {{ current.poem.note }}</p>
          <p class="ed-hint">来源：<template v-for="(s, i) in current.poem.sources" :key="i"><a v-if="s.url" :href="s.url" target="_blank" rel="noopener noreferrer">{{ s.name }}</a><span v-else>{{ s.name }}</span>{{ i < current.poem.sources.length - 1 ? '、' : '' }}</template></p>
        </details>

        <!-- 方案 -->
        <template v-if="current.status === 'plan_ready' && plan">
          <div class="ed-field">
            <span class="ed-label">形式</span>
            <div class="ai-segment">
              <button :class="{ active: plan.kind === 'series' }" @click="plan.kind = 'series'">连环画</button>
              <button :class="{ active: plan.kind === 'single' }" @click="plan.kind = 'single'">主画面 + 特写</button>
            </div>
            <p v-if="plan.kind_reason" class="ed-hint">{{ plan.kind_reason }}</p>
          </div>
          <div v-if="plan.concepts.length" class="ed-field">
            <span class="ed-label">立意（点别的立意，会写进意见让 AI 按它重做）</span>
            <div v-for="(c, i) in plan.concepts" :key="i" class="tt-concept" :class="{ active: i === plan.concept }" @click="chooseConcept(i)">
              <b>{{ c.name }}</b> {{ c.idea }}
            </div>
          </div>
          <div class="ed-field">
            <label>主要人物造型（全组一致）</label>
            <input v-model="plan.character" class="ed-input" maxlength="400" placeholder="没有主要人物可以留空">
          </div>
          <div class="ed-field">
            <label>用色</label>
            <input v-model="plan.color" class="ed-input" maxlength="300">
          </div>
          <div class="ed-field tt-scene">
            <span class="ed-label">封面</span>
            <textarea v-model="plan.cover.scene" class="ed-textarea" rows="3" placeholder="画面描述"></textarea>
            <textarea v-model="plan.cover.detailsText" class="ed-textarea" rows="4" placeholder="细节，每行一条"></textarea>
            <input v-model="plan.cover.find" class="ed-input" maxlength="60" placeholder="找一找：最值得放大去找的细节">
          </div>
          <div class="ed-field">
            <span class="ed-label">{{ plan.kind === 'series' ? '分镜（按原文顺序）' : '局部特写' }}</span>
            <div v-for="(s, i) in plan.panels" :key="i" class="art-section tt-scene">
              <div class="ed-row">
                <span class="art-num">{{ i + 1 }}</span>
                <input v-model="s.caption" class="ed-input" maxlength="16" placeholder="小标题">
                <button class="ed-close" title="删掉这一张" :disabled="plan.panels.length <= 2" @click="plan.panels.splice(i, 1)">×</button>
              </div>
              <input v-model="s.line" class="ed-input" maxlength="60" placeholder="一句话说这一幕发生了什么">
              <textarea v-model="s.scene" class="ed-textarea" rows="2" placeholder="画面描述"></textarea>
              <textarea v-model="s.detailsText" class="ed-textarea" rows="3" placeholder="细节，每行一条"></textarea>
              <input v-model="s.find" class="ed-input" maxlength="60" placeholder="找一找">
            </div>
            <button v-if="plan.panels.length < config.max_panels" class="ed-btn small" @click="plan.panels.push({ caption: '', line: '', scene: '', detailsText: '', find: '' })">＋ 加一张</button>
          </div>
          <div class="ed-field">
            <span class="ed-label">标题（点选或直接改，用第一个）</span>
            <div class="art-titles">
              <button v-for="(t, i) in plan.titles" :key="i" class="ai-chip" :class="{ active: i === 0 }" @click="plan.titles.unshift(plan.titles.splice(i, 1)[0])">{{ t }}</button>
            </div>
            <input v-model="plan.titles[0]" class="ed-input" maxlength="32">
          </div>
          <div class="ed-field">
            <label>结尾问读者</label>
            <input v-model="plan.question" class="ed-input" maxlength="40">
          </div>
          <div class="ed-field">
            <label for="tt-feedback">不满意？写下意见让 AI 重做方案</label>
            <textarea id="tt-feedback" v-model="feedback" class="ed-textarea" rows="2" maxlength="500" placeholder="例如：第三幕细节太少，加上村里人围观；主角画成少年"></textarea>
          </div>
          <div v-if="actionError" class="ed-msg err">{{ actionError }}</div>
          <div class="ed-row end">
            <button class="ed-btn" :disabled="acting || !feedback.trim()" @click="redoPlan">按意见重做</button>
            <button class="ed-btn primary" :disabled="acting" @click="confirmPlan">
              <span v-if="acting" class="ed-spinner"></span> 确认方案，画封面
            </button>
          </div>
        </template>

        <!-- 封面候选 -->
        <div v-if="current.covers.length" class="ed-field">
          <span class="ed-label">封面{{ current.status === 'cover_ready' ? '（选一张，全组都以它为参考）' : '' }}</span>
          <div class="tt-grid">
            <figure v-for="im in current.covers" :key="im.name" :class="{ picked: current.cover_pick === im.n, selectable: current.status === 'cover_ready' && im.status === 'ok' }" @click="current.status === 'cover_ready' && im.status === 'ok' && (coverChoice = im.n)">
              <img v-if="thumbs[im.name]" :src="thumbs[im.name]" :alt="im.name">
              <div v-else class="art-image-wait"><span v-if="im.status === 'pending' && active" class="ed-spinner"></span><span v-else-if="im.status === 'failed'" :title="im.error">✕ 没画成</span></div>
              <figcaption>
                <span v-if="current.status === 'cover_ready' && coverChoice === im.n">✔ </span>候选 {{ im.n }}
                <span v-if="im.check && !im.check.skipped" :class="im.check.ok ? 'tt-ok' : 'tt-warn'">{{ im.check.ok ? '自检通过' : '有问题' }}</span>
              </figcaption>
              <div v-if="im.check && !im.check.ok && im.check.issues.length" class="tt-issues">{{ im.check.issues.join('；') }}</div>
              <div v-if="im.check && im.check.minor && im.check.minor.length" class="tt-minor" :title="im.check.minor.join('；')">小问题：{{ im.check.minor.join('；') }}</div>
            </figure>
          </div>
          <template v-if="current.status === 'cover_ready'">
            <div class="ed-row end" style="margin-top: 8px;">
              <button class="ed-btn primary" :disabled="acting || !coverChoice" @click="pickCover">
                <span v-if="acting" class="ed-spinner"></span> 用这张，画后面 {{ current.plan.panels.length }} 张
              </button>
            </div>
            <div class="ed-field">
              <label>都不满意？写下意见重画三张</label>
              <textarea v-model="coverNote" class="ed-textarea" rows="2" maxlength="500" placeholder="例如：人物再大一些；不要那么粉"></textarea>
              <div class="ed-row end"><button class="ed-btn" :disabled="acting" @click="redrawCovers">重画封面</button></div>
            </div>
          </template>
        </div>

        <!-- 分镜 -->
        <div v-if="current.panels.length" class="ed-field">
          <span class="ed-label">{{ current.plan.kind === 'series' ? '分镜' : '局部特写' }}</span>
          <div class="tt-grid">
            <figure v-for="im in current.panels" :key="im.name">
              <img v-if="thumbs[im.name]" :src="thumbs[im.name]" :alt="im.name">
              <div v-else class="art-image-wait"><span v-if="im.status === 'pending' && active" class="ed-spinner"></span><span v-else-if="im.status === 'failed'" :title="im.error">✕ 没画成</span></div>
              <figcaption>
                {{ im.n }} {{ (current.plan.panels[im.n - 1] || {}).caption }}
                <span v-if="im.check && !im.check.skipped" :class="im.check.ok ? 'tt-ok' : 'tt-warn'">{{ im.check.ok ? '✓' : '⚠' }}</span>
              </figcaption>
              <div v-if="im.check && !im.check.ok && im.check.issues.length" class="tt-issues">{{ im.check.issues.join('；') }}</div>
              <div v-if="im.check && im.check.minor && im.check.minor.length" class="tt-minor" :title="im.check.minor.join('；')">小问题：{{ im.check.minor.join('；') }}</div>
              <button v-if="current.status === 'done'" class="ed-btn small" :disabled="acting" @click="redraw(im)">重画</button>
            </figure>
          </div>
        </div>

        <!-- 文案和推送 -->
        <template v-if="current.status === 'done' && post">
          <div class="ed-field">
            <div class="ed-label-row">
              <label for="tt-title">贴图标题</label>
              <span class="ed-counter">{{ post.title.length }}/32</span>
            </div>
            <input id="tt-title" v-model="post.title" class="ed-input" maxlength="32">
          </div>
          <div class="ed-field">
            <label for="tt-body">贴图正文（纯文本）</label>
            <textarea id="tt-body" v-model="post.body" class="ed-textarea" rows="12"></textarea>
            <div class="ed-row">
              <button class="ed-btn small" :disabled="acting || !postDirty" @click="savePost">保存修改</button>
              <button class="ed-btn small" @click="copy(post.title, '标题')">复制标题</button>
              <button class="ed-btn small" @click="copy(post.body, '正文')">复制正文</button>
              <button class="ed-btn small" :disabled="pushing" @click="download">下载装框后的图片</button>
            </div>
          </div>
          <div class="tt-push">
            <div class="ed-field">
              <label for="tt-account">推送到公众号草稿箱（贴图）</label>
              <select v-if="accounts.length" id="tt-account" v-model="accountId" class="ed-select">
                <option v-for="a in accounts" :key="a.id" :value="a.id">{{ a.name || a.appid }}</option>
              </select>
              <p v-else class="ed-hint">还没有配置公众号，先到「设置」里添加 AppID / AppSecret。<button class="ed-btn small" @click="$emit('open-settings')">打开设置</button></p>
            </div>
            <p class="ed-hint">会把封面和 {{ okPanels.length }} 张图排成 3:4（两边补纸色，不裁画面）、作为永久素材上传，再创建贴图草稿。<b>发表前记得在公众号后台勾选「AI 创作声明」。</b></p>
            <div v-if="pushMsg" class="ed-msg" :class="pushErr ? 'err' : 'ok'">{{ pushMsg }}</div>
            <div class="ed-row end">
              <button class="ed-btn primary" :disabled="pushing || !accounts.length" @click="push">
                <span v-if="pushing" class="ed-spinner"></span> {{ current.pushed_at ? '再推送一次' : '推送到草稿箱' }}
              </button>
            </div>
          </div>
        </template>

        <div v-if="actionError && current.status !== 'plan_ready'" class="ed-msg err">{{ actionError }}</div>

        <details class="art-events" :open="active">
          <summary>进度记录</summary>
          <ul>
            <li v-for="(e, i) in current.events" :key="i" :class="e.kind">{{ shortTime(e.at) }} {{ e.text }}</li>
          </ul>
        </details>

        <div class="ed-row end" style="margin-top: 10px;">
          <button v-if="active" class="ed-btn" :disabled="acting" @click="cancel">停止</button>
          <button v-if="current.status === 'failed' || current.status === 'canceled'" class="ed-btn primary" :disabled="acting" @click="retry">重试</button>
          <button v-if="!active" class="ed-btn danger" :disabled="acting" @click="remove">删除</button>
        </div>
      </template>
    </div>
  </aside>
</template>

<script>
import { tietuConfig, listTietu, createTietu, getTietu, tietuPick, tietuPlan, tietuCover, tietuRedraw, tietuSavePost, retryTietu, cancelTietu, tietuPushed, deleteTietu, fetchTietuImage, wechatUpload, wechatNewspic } from '../lib/api.js';
import { loadAccounts } from '../lib/settings.js';
import { frameTietu, tietuPaperColor } from '../lib/tietuFrame.js';

const ACTIVE = ['suggesting', 'planning', 'covering', 'drawing', 'checking'];
const STATUS = {
  suggesting: '挑诗中',
  suggest_ready: '待选诗',
  planning: '构思中',
  plan_ready: '待确认方案',
  covering: '画封面中',
  cover_ready: '待选封面',
  drawing: '画图中',
  checking: '自检中',
  done: '已完成',
  failed: '失败',
  canceled: '已停止'
};
const WAITING = ['suggest_ready', 'plan_ready', 'cover_ready'];

export default {
  name: 'TietuAgent',
  inject: ['editor'],
  props: { open: { type: Boolean, default: false } },
  emits: ['close', 'open-settings'],
  data() {
    return {
      config: { styles: [], default_panels: 6, max_panels: 8, covers: 3, image_price: null },
      denied: false,
      view: 'list',
      list: [],
      listLoading: false,
      listError: '',
      form: { input: '', theme: false, mode: 'auto', panels: 6, style: 'xinzhongshi', style_text: '', notes: '' },
      modes: [{ id: 'auto', label: '自动判断' }, { id: 'series', label: '连环画' }, { id: 'single', label: '主画面 + 特写' }],
      formError: '',
      creating: false,
      current: null,
      plan: null,
      feedback: '',
      coverChoice: 0,
      coverNote: '',
      post: null,
      savedPost: '',
      acting: false,
      actionError: '',
      thumbs: {},
      accounts: [],
      accountId: '',
      pushing: false,
      pushMsg: '',
      pushErr: false,
      watched: {}
    };
  },
  computed: {
    panelCounts() {
      return Array.from({ length: (this.config.max_panels || 8) - 1 }, (_, i) => i + 2);
    },
    styleText() {
      const s = this.config.styles.find(x => x.id === this.form.style);
      return s ? s.text : '';
    },
    active() {
      return !!this.current && ACTIVE.includes(this.current.status);
    },
    lastStep() {
      const ev = this.current && [...this.current.events].reverse().find(e => e.kind === 'step');
      return ev ? ev.text : '处理中…';
    },
    okPanels() {
      return this.current ? this.current.panels.filter(im => im.status === 'ok') : [];
    },
    postDirty() {
      return !!this.post && JSON.stringify(this.post) !== this.savedPost;
    },
    account() {
      return this.accounts.find(a => a.id === this.accountId) || null;
    }
  },
  watch: {
    open(v) {
      if (v) {
        this.loadConfig();
        if (this.view === 'list') this.loadList();
        const acc = loadAccounts();
        this.accounts = acc.accounts;
        this.accountId = acc.defaultId;
      }
    },
    'editor.aiKey.id'() {
      if (this.open) this.loadConfig();
    }
  },
  mounted() {
    this._timer = setInterval(() => this.tick(), 3000);
    this._ticks = 0;
  },
  beforeUnmount() {
    clearInterval(this._timer);
    this.dropThumbs();
  },
  methods: {
    statusText(p) {
      if (p.status === 'done' && p.pushed_at) return '已推送';
      return STATUS[p.status] || p.status;
    },
    badgeClass(p) {
      if (ACTIVE.includes(p.status)) return 'busy';
      if (WAITING.includes(p.status)) return 'wait';
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
        this.config = await tietuConfig(this.editor.aiKey && this.editor.aiKey.id);
        this.denied = false;
        if (!this.form.panels) this.form.panels = this.config.default_panels;
      } catch (error) {
        if (error.reason === 'TIETU_ADMIN') this.denied = true;
      }
    },
    async loadList() {
      if (!this.editor.auth.loggedIn || this.denied) return;
      this.listLoading = true;
      this.listError = '';
      try {
        this.list = await listTietu();
        for (const p of this.list) this.watched[p.id] = p.status;
      } catch (error) {
        if (error.reason === 'TIETU_ADMIN') this.denied = true;
        else this.listError = error.message;
      } finally {
        this.listLoading = false;
      }
    },
    backToList() {
      this.view = 'list';
      this.current = null;
      this.plan = null;
      this.post = null;
      this.dropThumbs();
      this.loadList();
    },
    async create() {
      this.formError = '';
      this.creating = true;
      try {
        const p = await createTietu({ key_id: this.editor.aiKey.id, ...this.form, input: this.form.input.trim() });
        this.watched[p.id] = p.status;
        this.form.input = '';
        this.form.notes = '';
        await this.openProject(p.id);
      } catch (error) {
        this.editor.noteApiError(error);
        this.formError = error.message;
      } finally {
        this.creating = false;
      }
    },
    async openProject(id) {
      this.dropThumbs();
      this.feedback = '';
      this.coverNote = '';
      this.actionError = '';
      this.pushMsg = '';
      this.view = 'project';
      await this.refresh(id);
    },
    async refresh(id) {
      try {
        this.applyProject(await getTietu(id));
      } catch (error) {
        this.actionError = error.message;
      }
    },
    applyProject(p) {
      const before = this.current && this.current.id === p.id ? this.current.status : null;
      this.current = p;
      this.watched[p.id] = p.status;
      if (p.status === 'plan_ready' && (before !== 'plan_ready' || !this.plan)) {
        const plan = JSON.parse(JSON.stringify(p.plan));
        plan.cover.detailsText = (plan.cover.details || []).join('\n');
        for (const s of plan.panels) s.detailsText = (s.details || []).join('\n');
        if (!plan.titles.length) plan.titles.push(p.poem ? p.poem.title : '');
        this.plan = plan;
      }
      if (p.status !== 'plan_ready') this.plan = null;
      if (p.status === 'cover_ready' && before !== 'cover_ready') this.coverChoice = (p.covers.find(c => c.status === 'ok') || {}).n || 0;
      if (p.status === 'done' && p.post && (before !== 'done' || !this.post)) {
        this.post = { title: p.post.title, body: p.post.body };
        this.savedPost = JSON.stringify(this.post);
      }
      if (p.status !== 'done') this.post = null;
      this.loadThumbs(p);
    },
    async loadThumbs(p) {
      // Pictures are redrawn under the same name: keep only thumbnails of pictures that are drawn now.
      const ok = new Set([...p.covers, ...p.panels].filter(im => im.status === 'ok').map(im => im.name));
      for (const name of Object.keys(this.thumbs)) {
        if (ok.has(name)) continue;
        if (this.thumbs[name] !== 'loading') URL.revokeObjectURL(this.thumbs[name]);
        delete this.thumbs[name];
      }
      for (const im of [...p.covers, ...p.panels]) {
        if (im.status !== 'ok' || this.thumbs[im.name]) continue;
        this.thumbs[im.name] = 'loading';
        try {
          const blob = await fetchTietuImage(p.id, im.name);
          if (!this.current || this.current.id !== p.id) return;
          this.thumbs[im.name] = URL.createObjectURL(blob);
        } catch {
          delete this.thumbs[im.name];
        }
      }
    },
    dropThumbs() {
      for (const url of Object.values(this.thumbs)) {
        if (url && url !== 'loading') URL.revokeObjectURL(url);
      }
      this.thumbs = {};
    },
    lines(text) {
      return (text || '').split('\n').map(x => x.trim()).filter(Boolean);
    },
    planPayload() {
      const pl = this.plan;
      const scene = s => ({ caption: s.caption || '', line: s.line || '', scene: s.scene || '', details: this.lines(s.detailsText), find: s.find || '' });
      return {
        kind: pl.kind,
        kind_reason: pl.kind_reason,
        concepts: pl.concepts,
        concept: pl.concept,
        character: pl.character,
        color: pl.color,
        cover: scene(pl.cover),
        panels: pl.panels.map(scene),
        titles: pl.titles.filter(t => t && t.trim()),
        question: pl.question
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
    pick(i) {
      this.act(() => tietuPick(this.current.id, i));
    },
    chooseConcept(i) {
      if (i === this.plan.concept) return;
      const c = this.plan.concepts[i];
      this.feedback = `改用立意「${c.name}」：${c.idea}`;
    },
    redoPlan() {
      const fb = this.feedback.trim();
      this.feedback = '';
      this.act(() => tietuPlan(this.current.id, { plan: this.planPayload(), feedback: fb }));
    },
    confirmPlan() {
      this.act(() => tietuPlan(this.current.id, { plan: this.planPayload(), confirm: true }));
    },
    pickCover() {
      this.act(() => tietuCover(this.current.id, { pick: this.coverChoice }));
    },
    redrawCovers() {
      const note = this.coverNote.trim();
      this.coverNote = '';
      this.act(() => tietuCover(this.current.id, { redraw: true, note }));
    },
    redraw(im) {
      const note = window.prompt(`重画第 ${im.n} 张。想怎么改？（可以留空，直接重画）`, (im.check && im.check.issues.join('；')) || '');
      if (note === null) return;
      this.act(() => tietuRedraw(this.current.id, im.name, note.trim()));
    },
    async savePost() {
      await this.act(() => tietuSavePost(this.current.id, this.post.title.trim(), this.post.body.trim()));
      if (!this.actionError) this.savedPost = JSON.stringify(this.post);
    },
    async copy(text, what) {
      try {
        await navigator.clipboard.writeText(text);
        this.editor.showToast(`${what}已复制`, 'success');
      } catch {
        this.editor.showToast('复制失败，请手动选中复制', 'error');
      }
    },
    retry() {
      this.act(() => retryTietu(this.current.id));
    },
    cancel() {
      this.act(() => cancelTietu(this.current.id));
    },
    async remove() {
      if (!window.confirm('删除这条贴图和它的图片？（已经推送到草稿箱的内容不受影响）')) return;
      await this.act(() => deleteTietu(this.current.id));
      if (!this.actionError) this.backToList();
    },
    /** The pictures in post order (chosen cover first), framed to 3:4. */
    async framedPictures(onStep) {
      const p = this.current;
      const names = [`c${p.cover_pick}`, ...this.okPanels.map(im => im.name)];
      const out = [];
      let paper = null;
      for (let i = 0; i < names.length; i++) {
        onStep && onStep(`排版第 ${i + 1}/${names.length} 张…`);
        const blob = await fetchTietuImage(p.id, names[i]);
        if (i === 0) paper = await tietuPaperColor(blob); // the cover's paper colour frames the whole set
        out.push(await frameTietu(blob, { label: i === 0, paper }));
      }
      return out;
    },
    async download() {
      this.pushing = true;
      this.pushErr = false;
      try {
        const pics = await this.framedPictures(t => { this.pushMsg = t; });
        pics.forEach((blob, i) => {
          const a = document.createElement('a');
          a.href = URL.createObjectURL(blob);
          a.download = `${this.current.poem.title}-${String(i + 1).padStart(2, '0')}.jpg`;
          a.click();
          setTimeout(() => URL.revokeObjectURL(a.href), 5000);
        });
        this.pushMsg = `已下载 ${pics.length} 张（3:4，第一张是封面）`;
      } catch (error) {
        this.pushErr = true;
        this.pushMsg = error.message || '下载失败';
      } finally {
        this.pushing = false;
      }
    },
    async push() {
      if (!this.account) return;
      if (this.postDirty) await this.savePost();
      this.pushing = true;
      this.pushErr = false;
      try {
        const account = { ...this.account };
        const pics = await this.framedPictures(t => { this.pushMsg = t; });
        const ids = [];
        for (let i = 0; i < pics.length; i++) {
          this.pushMsg = `上传第 ${i + 1}/${pics.length} 张…`;
          const res = await wechatUpload(account, 'cover', pics[i], `tietu-${i + 1}.jpg`);
          ids.push(res.media_id);
        }
        this.pushMsg = '创建贴图草稿…';
        await wechatNewspic(account, { title: this.post.title.trim(), content: this.post.body.trim(), imageMediaIds: ids, openComment: true });
        await tietuPushed(this.current.id);
        this.pushMsg = `已推送到「${account.name || account.appid}」的草稿箱（贴图，${ids.length} 张）。发表前记得勾选「AI 创作声明」。`;
        await this.refresh(this.current.id);
      } catch (error) {
        this.editor.noteApiError(error);
        this.pushErr = true;
        this.pushMsg = error.message || '推送失败';
      } finally {
        this.pushing = false;
      }
    },
    /** Every 3 s: refresh the open project while it runs. Every 12 s: announce finished background runs. */
    async tick() {
      if (!this.editor.auth.loggedIn || this.denied) return;
      this._ticks++;
      if (!document.hidden && this.open && this.view === 'project' && this.active) await this.refresh(this.current.id);
      if (this._ticks % 4 !== 0) return;
      const running = Object.entries(this.watched).filter(([, s]) => ACTIVE.includes(s)).map(([id]) => Number(id));
      for (const id of running) {
        if (this.open && this.current && this.current.id === id) continue;
        try {
          const p = await getTietu(id);
          this.watched[id] = p.status;
          if (!ACTIVE.includes(p.status)) {
            const text = p.status === 'done' ? `贴图「${p.title}」做好了` : WAITING.includes(p.status) ? `贴图「${p.title}」等你确认（${STATUS[p.status]}）` : `贴图「${p.title}」没做成`;
            this.editor.showToast(text, p.status === 'failed' ? 'error' : 'success');
          }
        } catch {
          delete this.watched[id];
        }
      }
    }
  }
};
</script>
