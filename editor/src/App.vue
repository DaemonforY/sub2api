<template>
  <!-- 顶部栏 -->
  <header class="header">
    <div class="logo">
      <img :src="logoUrl" alt="HiveGPT" width="20" height="20">
      <span>HiveGPT 公众号排版</span>
      <span class="creator-info">基于 <a href="https://github.com/alchaincyf/huasheng_editor" target="_blank" rel="noopener noreferrer">huasheng_editor</a>（MIT）</span>
    </div>
    <div class="header-actions">
      <nav class="header-links">
        <button class="header-link" type="button" @click="openImport">导入文章</button>
        <button class="header-link" type="button" @click="toggleAiPanel">AI 助手</button>
        <button class="header-link" type="button" @click="showSettings = true">设置</button>
        <a class="header-link" href="/">HiveGPT 首页</a>
        <a v-if="!auth.loggedIn" class="header-link primary" :href="loginHref">登录</a>
        <span v-else class="header-account" :title="accountTitle">{{ accountLabel }}</span>
      </nav>
      <a href="https://github.com/alchaincyf/huasheng_editor"
         target="_blank"
         rel="noopener noreferrer"
         class="github-link"
         title="上游项目 huasheng_editor（MIT）源码">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
        </svg>
      </a>
    </div>
  </header>

  <!-- 样式选择区 - 横向滚动 -->
  <section class="styles-section">
    <div class="styles-container">
      <div class="styles-label">选择样式</div>
      <div class="styles-scroll-wrapper">
        <div class="styles-list">
          <!-- 收藏的样式 -->
          <template v-if="starredStyles.length > 0">
            <div
              v-for="styleKey in starredStyles"
              :key="'star-' + styleKey"
              class="style-card starred"
              :class="{ active: currentStyle === styleKey }"
              @click="currentStyle = styleKey"
            >
              {{ getStyleName(styleKey) }}
            </div>
            <div style="width: 1px; height: 20px; background: var(--color-light-border); margin: 0 4px;"></div>
          </template>

          <!-- 所有样式，添加推荐标识 -->
          <div
            v-for="(style, key) in STYLES"
            :key="key"
            class="style-card"
            :class="{
              active: currentStyle === key,
              starred: isStyleStarred(key),
              recommended: isRecommended(key)
            }"
            @click="currentStyle = key"
          >
            {{ style.name }}
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- 主内容区 -->
  <div class="main-content">
    <!-- 编辑器面板 -->
    <div class="editor-panel">
      <div class="markdown-input-container">
        <textarea
          ref="mdInput"
          v-model="markdownInput"
          class="markdown-input"
          @select="trackSelection"
          @keyup="trackSelection"
          @mouseup="trackSelection"
          @input="trackSelection"
          :class="{ 'drag-over': isDraggingOver }"
          @paste="handleSmartPaste"
          @drop="handleDrop"
          @dragover="handleDragOver"
          @dragenter="handleDragEnter"
          @dragleave="handleDragLeave"
          placeholder="# 在这里粘贴或输入 Markdown 内容

## 功能特点

- **实时预览**：所见即所得
- **13 种样式**：随意切换
- **智能粘贴**：支持从飞书、Notion、Word 直接粘贴
- **图片支持**：直接粘贴或拖拽图片，自动上传
- **一键复制**：直接粘贴到公众号

💡 提示：可以直接粘贴截图，或拖拽图片文件到这里"
          spellcheck="false"
        ></textarea>
      </div>

      <!-- 编辑器底部工具栏 -->
      <div class="editor-footer">
        <div class="upload-section">
          <label for="file-upload" class="upload-label">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="17 8 12 3 7 8"></polyline>
              <line x1="12" y1="3" x2="12" y2="15"></line>
            </svg>
            <span>上传 MD 文件</span>
            <input type="file" id="file-upload" accept=".md,.markdown" @change="handleFileUpload" style="display: none;">
          </label>
          <div class="upload-hints">
            <span class="upload-hint">支持 .md / .markdown 文件</span>
            <span class="feishu-hint">
              💡 支持从飞书、Notion、Word 等直接粘贴，自动转换为 Markdown
            </span>
          </div>
        </div>
        <div class="char-count">
          {{ markdownInput.length }} 字符
        </div>
      </div>
    </div>

    <!-- 预览面板 -->
    <div class="preview-panel">
      <div class="preview-header">
        <div class="preview-title">
          预览
          <!-- 小红书功能入口已隐藏，待调试完成后再开放 -->
          <!-- <div class="mode-switcher" style="display: none;">
            <button
              class="mode-toggle-btn"
              :class="{ active: previewMode === 'wechat' }"
              @click="previewMode = 'wechat'"
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
              </svg>
              公众号
            </button>
            <button
              class="mode-toggle-btn"
              :class="{ active: previewMode === 'xiaohongshu' }"
              @click="previewMode = 'xiaohongshu'"
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="7" rx="1"/>
                <rect x="14" y="3" width="7" height="7" rx="1"/>
                <rect x="3" y="14" width="7" height="7" rx="1"/>
                <rect x="14" y="14" width="7" height="7" rx="1"/>
              </svg>
              小红书
            </button>
          </div> -->
        </div>
        <div class="preview-actions">
          <button
            class="star-btn"
            :class="{ active: isStyleStarred(currentStyle) }"
            @click="toggleStarStyle(currentStyle)"
            title="收藏此样式"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" :fill="isStyleStarred(currentStyle) ? 'currentColor' : 'none'">
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"></path>
            </svg>
          </button>
          <button
            class="history-btn"
            @click="toggleHistoryPanel"
            title="历史记录"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <span class="history-btn-text">历史</span>
          </button>
          <button
            class="save-btn"
            @click="saveToHistory"
            title="保存到历史记录"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path>
            </svg>
            <span class="save-btn-text">保存</span>
          </button>
          <button
            class="copy-btn copy-btn-draft"
            :disabled="!renderedContent"
            title="上传图片和封面，保存到公众号草稿箱"
            @click="openDraft"
          >
            发送到草稿箱
          </button>
          <button
            class="copy-btn copy-btn-x"
            :class="{ success: copyXSuccess }"
            :disabled="!renderedContent"
            @click="copyToXArticles"
          >
            {{ copyXSuccess ? '✓ 已复制' : '复制到 X' }}
          </button>
          <button
            class="copy-btn"
            :class="{ success: copySuccess }"
            :disabled="!renderedContent"
            @click="copyToClipboard"
          >
            {{ copySuccess ? '✓ 已复制' : '复制到公众号' }}
          </button>
        </div>
      </div>

      <!-- 公众号预览模式 -->
      <div class="preview-content">
        <div v-if="!renderedContent" class="empty-state">
          <svg class="empty-state-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h3>开始编辑</h3>
          <p>左侧输入 Markdown，右侧实时预览</p>
        </div>
        <div v-else class="preview-container" v-html="renderedContent"></div>
      </div>

      <!-- 小红书预览模式（已隐藏） -->
      <!-- <div v-if="previewMode === 'xiaohongshu'" class="xiaohongshu-preview">
        <div class="xiaohongshu-controls">
          <button
            class="xiaohongshu-btn"
            :disabled="!renderedContent || xiaohongshuGenerating"
            @click="generateXiaohongshuImages"
          >
            {{ xiaohongshuGenerating ? '生成中...' : '生成小红书图片' }}
          </button>
          <button
            class="xiaohongshu-btn secondary"
            :disabled="xiaohongshuImages.length === 0"
            @click="downloadAllXiaohongshuImages"
          >
            批量下载 ({{ xiaohongshuImages.length }})
          </button>
        </div>

        <div v-if="xiaohongshuGenerating" class="xiaohongshu-loading">
          <p>正在生成小红书图片，请稍候...</p>
        </div>

        <div v-else-if="xiaohongshuImages.length === 0" class="xiaohongshu-empty">
          <p>点击「生成小红书图片」按钮，将内容转换为适合小红书发布的多图</p>
        </div>

        <div v-else class="xiaohongshu-images-grid">
          <div
            v-for="(image, index) in xiaohongshuImages"
            :key="index"
            class="xiaohongshu-image-card"
            @click="downloadXiaohongshuImage(image, index)"
          >
            <img :src="image.dataUrl" :alt="`第 ${index + 1} 张`">
            <div class="xiaohongshu-image-label">
              第 {{ index + 1 }} / {{ xiaohongshuImages.length }} 张
            </div>
          </div>
        </div>
      </div> -->
    </div>
  </div>

  <!-- 历史记录侧边栏 -->
  <aside class="history-sidebar" :class="{ open: showHistoryPanel }">
    <div class="history-header">
      <div class="history-header-left">
        <h3>历史记录</h3>
        <span class="history-count">{{ articleHistory.length }}/20</span>
      </div>
      <button class="history-close-btn" @click="toggleHistoryPanel">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M6 18L18 6M6 6l12 12"></path>
        </svg>
      </button>
    </div>
    <div class="history-list">
      <div
        v-for="article in articleHistory"
        :key="article.id"
        class="history-item"
      >
        <div class="history-item-content">
          <div class="history-title">{{ article.title }}</div>
          <div class="history-meta">
            <span class="history-time">{{ formatHistoryDate(article.updatedAt) }}</span>
            <span class="history-style">{{ STYLES[article.style]?.name || article.style }}</span>
          </div>
        </div>
        <div class="history-actions">
          <button class="history-action-btn load" @click="loadFromHistory(article.id)" title="加载">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path>
            </svg>
          </button>
          <button class="history-action-btn delete" @click="deleteFromHistory(article.id)" title="删除">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
            </svg>
          </button>
        </div>
      </div>
      <div v-if="!articleHistory.length" class="history-empty">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
        </svg>
        <p>暂无历史记录</p>
        <span>复制文章或点击保存按钮后自动记录</span>
      </div>
    </div>
  </aside>

  <!-- 侧边栏遮罩层 -->
  <div class="history-overlay" :class="{ visible: showHistoryPanel }" @click="toggleHistoryPanel"></div>

  <!-- AI 助手 -->
  <AiPanel :open="showAiPanel" @close="showAiPanel = false" @open-settings="showSettings = true" />

  <!-- 弹窗 -->
  <SettingsDialog v-if="showSettings" @close="showSettings = false" />
  <ImportDialog v-if="showImport" @close="showImport = false" />
  <DraftDialog v-if="showDraft" @close="showDraft = false" @open-settings="showSettings = true" />

  <!-- Toast 提示 -->
  <div v-if="toast.show" class="toast" :class="toast.type">
    {{ toast.message }}
  </div>
</template>

<script>
import markdownit from 'markdown-it';
import hljs from './lib/highlight.js';
import TurndownService from 'turndown';
import { STYLES } from './styles.js';
import { ImageStore } from './lib/imageStore.js';
import { ImageCompressor } from './lib/imageCompressor.js';
import { ImageHostManager } from './lib/imageHostManager.js';
import { EMPHASIS_MARKERS, isCjkLetter, isCjkPunctuation, withTimeout } from './lib/helpers.js';
import { token, currentUser, loginUrl, isWechatImageUrl, fetchProxiedImage } from './lib/api.js';
import { loadAiKey, loadDraftMeta, saveDraftMeta } from './lib/settings.js';
import { dataUrlToBlob, loadImageCrossOrigin, imageElementToBlob } from './lib/imageTools.js';
import SettingsDialog from './components/SettingsDialog.vue';
import ImportDialog from './components/ImportDialog.vue';
import DraftDialog from './components/DraftDialog.vue';
import AiPanel from './components/AiPanel.vue';

export default {
  components: { SettingsDialog, ImportDialog, DraftDialog, AiPanel },

  // 子组件（设置 / 导入 / 草稿箱 / AI 助手）通过 inject('editor') 使用编辑器的状态和方法
  provide() {
    return { editor: this };
  },

  data() {
    return {
      // ---- HiveGPT：登录、设置、草稿、AI ----
      auth: { loggedIn: false, user: null },
      loginHref: loginUrl(),
      aiKey: loadAiKey(),               // {id, name}，AI 功能计费用的 Key
      draftMeta: loadDraftMeta(),       // 草稿标题 / 作者 / 摘要 / 封面等
      editorSelection: { start: 0, end: 0, length: 0 },
      showSettings: false,
      showImport: false,
      showDraft: false,
      showAiPanel: false,

      markdownInput: '',
      renderedContent: '',
      currentStyle: 'wechat-default',
      copySuccess: false,
      starredStyles: [],
      toast: {
        show: false,
        message: '',
        type: 'success'
      },
      md: null,
      scanDelimsPatched: false,
      STYLES: STYLES,  // 将样式对象暴露给模板
      logoUrl: import.meta.env.BASE_URL + 'logo.svg',
      turndownService: null,  // Turndown 服务实例
      isDraggingOver: false,  // 拖拽状态
      imageHostManager: new ImageHostManager(),  // 图床管理器（已废弃，保留兼容）
      imageStore: null,  // 图片存储管理器（IndexedDB）
      imageCompressor: null,  // 图片压缩器
      imageIdToObjectURL: {},  // 图片 ID 到 Object URL 的映射（用于预览时替换）
      // 小红书相关
      previewMode: 'wechat',  // 预览模式：'wechat' 或 'xiaohongshu'
      xiaohongshuImages: [],  // 生成的小红书图片数组
      xiaohongshuGenerating: false,  // 是否正在生成小红书图片
      // 文章历史记录
      articleHistory: [],           // 历史文章列表
      showHistoryPanel: false,      // 侧边栏显示状态
      currentArticleId: null,       // 当前编辑的文章ID（用于防止重复保存）
      copyXSuccess: false            // 复制到 X 成功状态
    };
  },

  async mounted() {
    // 登录状态（主站登录后 token 在同域 localStorage）
    this.refreshAuth();
    this._onAuthChange = () => this.refreshAuth();
    window.addEventListener('storage', this._onAuthChange);
    window.addEventListener('focus', this._onAuthChange);

    // 加载星标样式
    this.loadStarredStyles();

    // 加载用户偏好设置
    this.loadUserPreferences();

    // 加载文章历史记录
    this.loadArticleHistory();

    // 初始化图片存储管理器
    this.imageStore = new ImageStore();
    try {
      await this.imageStore.init();
      console.log('图片存储系统已就绪');
    } catch (error) {
      console.error('图片存储系统初始化失败:', error);
      this.showToast('图片存储系统初始化失败', 'error');
    }

    // 初始化图片压缩器（最大宽度 1920px，质量 85%）
    this.imageCompressor = new ImageCompressor({
      maxWidth: 1920,
      maxHeight: 1920,
      quality: 0.85
    });

    // 初始化 Turndown 服务（HTML 转 Markdown）
    this.initTurndownService();

    // 初始化 markdown-it
    const md = markdownit({
      html: true,
      linkify: true,
      typographer: false,  // 禁用 typographer 以避免智能引号干扰加粗标记
      highlight: function (str, lang) {
        // macOS 风格的窗口装饰
        const dots = '<div style="display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: #2a2c33; border-bottom: 1px solid #1e1f24;"><span style="width: 12px; height: 12px; border-radius: 50%; background: #ff5f56;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #ffbd2e;"></span><span style="width: 12px; height: 12px; border-radius: 50%; background: #27c93f;"></span></div>';

        // 检查 hljs 是否加载
        let codeContent = '';
        if (lang && typeof hljs !== 'undefined') {
          try {
            if (hljs.getLanguage(lang)) {
              codeContent = hljs.highlight(str, { language: lang }).value;
            } else {
              codeContent = md.utils.escapeHtml(str);
            }
          } catch (__) {
            codeContent = md.utils.escapeHtml(str);
          }
        } else {
          codeContent = md.utils.escapeHtml(str);
        }

        return `<div style="margin: 20px 0; border-radius: 8px; overflow: hidden; background: #383a42; box-shadow: 0 2px 8px rgba(0,0,0,0.15);">${dots}<div style="padding: 16px; overflow-x: auto; background: #383a42;"><code style="display: block; color: #abb2bf; font-family: 'SF Mono', Monaco, 'Cascadia Code', Consolas, monospace; font-size: 14px; line-height: 1.6; white-space: pre;">${codeContent}</code></div></div>`;
      }
    });

    this.patchMarkdownScanner(md);
    this.md = md;

    // 手动触发一次渲染（确保初始内容显示）
    this.$nextTick(() => {
      this.renderMarkdown();
    });
  },

  beforeUnmount() {
    window.removeEventListener('storage', this._onAuthChange);
    window.removeEventListener('focus', this._onAuthChange);
  },

  computed: {
    accountLabel() {
      const u = this.auth.user;
      return (u && (u.username || u.email)) || '已登录';
    },
    accountTitle() {
      const u = this.auth.user;
      return u && u.email ? `已登录：${u.email}` : '已登录 HiveGPT';
    }
  },

  watch: {
    draftMeta: {
      deep: true,
      handler(meta) {
        clearTimeout(this._metaTimer);
        this._metaTimer = setTimeout(() => saveDraftMeta(meta), 300);
      }
    },
    currentStyle() {
      if (this.md) {
        this.renderMarkdown();
      }
      // 保存样式偏好
      this.saveUserPreferences();
    },
    markdownInput(newVal, oldVal) {
      if (this.md) {
        this.renderMarkdown();
      }
      // 自动保存内容（防抖）
      clearTimeout(this._saveTimeout);
      this._saveTimeout = setTimeout(() => {
        this.saveUserPreferences();
      }, 1000); // 1秒后保存

      // 当内容被清空时，重置当前文章ID（下次保存会创建新文章）
      if (!newVal || !newVal.trim()) {
        this.currentArticleId = null;
      }
      // 当从空内容粘贴大量内容时，也视为新文章
      else if ((!oldVal || oldVal.trim().length < 10) && newVal.trim().length > 100) {
        this.currentArticleId = null;
      }
    }
  },

  methods: {
    // ==================== HiveGPT 新功能 ====================

    refreshAuth() {
      const loggedIn = !!token();
      this.auth.loggedIn = loggedIn;
      this.auth.user = loggedIn ? currentUser() : null;
    },

    /** API 出错时调用：401 说明登录失效，刷新顶栏的登录状态。 */
    noteApiError(error) {
      if (error && error.status === 401) {
        this.refreshAuth();
        if (this.auth.loggedIn) {
          // token 还在但已失效
          this.auth.loggedIn = false;
          this.auth.user = null;
        }
      }
    },

    openImport() {
      this.showImport = true;
    },

    openDraft() {
      if (!this.renderedContent) {
        this.showToast('没有内容可发送', 'error');
        return;
      }
      this.refreshAuth();
      this.showDraft = true;
    },

    toggleAiPanel() {
      this.refreshAuth();
      this.showAiPanel = !this.showAiPanel;
      if (this.showAiPanel) this.showHistoryPanel = false;
    },

    trackSelection() {
      const ta = this.$refs.mdInput;
      if (!ta) return;
      const start = ta.selectionStart || 0;
      const end = ta.selectionEnd || 0;
      this.editorSelection = { start, end, length: Math.max(0, end - start) };
    },

    /** 编辑器当前选区（焦点离开后浏览器仍保留 selectionStart/End）。 */
    getSelectionInfo() {
      const ta = this.$refs.mdInput;
      const len = this.markdownInput.length;
      let start = ta ? ta.selectionStart : len;
      let end = ta ? ta.selectionEnd : len;
      if (typeof start !== 'number') start = len;
      if (typeof end !== 'number') end = start;
      return { start, end, text: this.markdownInput.slice(start, end) };
    },

    replaceRange(start, end, text) {
      this.markdownInput = this.markdownInput.slice(0, start) + text + this.markdownInput.slice(end);
      this.$nextTick(() => {
        const ta = this.$refs.mdInput;
        if (ta) {
          ta.focus();
          ta.selectionStart = start;
          ta.selectionEnd = start + text.length;
          this.trackSelection();
        }
      });
    },

    /** 在 pos 处插入一段独立的 Markdown 块（前后自动补空行）。 */
    insertBlockAt(pos, block) {
      const value = this.markdownInput;
      pos = Math.max(0, Math.min(pos, value.length));
      const before = value.slice(0, pos);
      const after = value.slice(pos);
      let lead = '';
      if (before && !before.endsWith('\n\n')) lead = before.endsWith('\n') ? '\n' : '\n\n';
      let tail = '';
      if (after && !after.startsWith('\n\n')) tail = after.startsWith('\n') ? '\n' : '\n\n';
      const inserted = lead + block + tail;
      this.markdownInput = before + inserted + after;
      this.$nextTick(() => {
        const ta = this.$refs.mdInput;
        if (ta) {
          const caret = pos + lead.length + block.length;
          ta.selectionStart = ta.selectionEnd = caret;
          this.trackSelection();
        }
      });
    },

    setDraftTitle(title, source = 'manual') {
      this.draftMeta.title = (title || '').trim().slice(0, 64);
      this.draftMeta.titleSource = source;
    },

    /** 预览里第一个一级标题的文字。 */
    firstH1Text() {
      if (!this.renderedContent) return '';
      const doc = new DOMParser().parseFromString(this.renderedContent, 'text/html');
      const h1 = doc.querySelector('h1');
      return h1 ? h1.textContent.replace(/\s+/g, ' ').trim() : '';
    },

    /** 压缩并存入 IndexedDB，返回图片 ID（编辑器里写成 img://ID）。 */
    async storeImageBlob(blob, name = '图片') {
      if (!this.imageStore) throw new Error('图片存储系统未就绪');
      const originalSize = blob.size;
      const compressed = await this.imageCompressor.compress(blob);
      const imageId = `img-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      await this.imageStore.saveImage(imageId, compressed, {
        name,
        originalName: name,
        originalSize,
        compressedSize: compressed.size,
        compressionRatio: ((1 - compressed.size / originalSize) * 100).toFixed(0),
        mimeType: compressed.type || blob.type
      });
      return imageId;
    },

    /** 取图片原始数据：img://（IndexedDB）、data:、公众号图片（代理）、同源、支持 CORS 的外链。取不到返回 null。 */
    async getImageBlobBySrc(src) {
      if (!src) return null;
      try {
        if (src.startsWith('img://')) {
          const record = await this.imageStore.getImageRecord(src.slice('img://'.length));
          return record && record.blob ? record.blob : null;
        }
        if (src.startsWith('data:')) return dataUrlToBlob(src);
        if (isWechatImageUrl(src)) return await fetchProxiedImage(src);
        const url = new URL(src, window.location.href);
        if (url.origin === window.location.origin) {
          const res = await fetch(url.href);
          return res.ok ? await res.blob() : null;
        }
        if (url.protocol === 'https:' || url.protocol === 'http:') {
          // connect-src 不允许直接 fetch 外链；用 <img crossorigin> 读进 canvas（需要对方支持 CORS）
          const el = await loadImageCrossOrigin(url.href);
          return await imageElementToBlob(el);
        }
      } catch (error) {
        if (error && error.status === 401) throw error;
        console.warn('读取图片失败:', src, error);
      }
      return null;
    },

    async getImageBlobForElement(img) {
      const imageId = img.getAttribute('data-image-id');
      if (imageId && this.imageStore) {
        const record = await this.imageStore.getImageRecord(imageId);
        if (record && record.blob) return record.blob;
      }
      return this.getImageBlobBySrc(img.getAttribute('src') || '');
    },

    /** 文章里可作封面的图片（草稿箱弹窗用）。 */
    async articleImageCandidates() {
      const out = [];
      const seen = new Set();
      const re = /!\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)/g;
      for (const m of this.markdownInput.matchAll(re)) {
        const src = m[2];
        if (seen.has(src)) continue;
        seen.add(src);
        const label = m[1] || '文章图片';
        if (src.startsWith('img://')) {
          const imageId = src.slice('img://'.length);
          let thumb = this.imageIdToObjectURL[imageId];
          if (!thumb) {
            try {
              thumb = await this.imageStore.getImage(imageId);
              if (thumb) this.imageIdToObjectURL[imageId] = thumb;
            } catch (e) {
              thumb = null;
            }
          }
          if (thumb) out.push({ key: src, src, imageId, thumb, label });
        } else if (src.startsWith('data:image/') || (/^https?:\/\//i.test(src) && !isWechatImageUrl(src))) {
          out.push({ key: src, src, imageId: '', thumb: src, label });
        }
        if (out.length >= 12) break;
      }
      return out;
    },

    loadStarredStyles() {
      try {
        const saved = localStorage.getItem('starredStyles');
        if (saved) {
          this.starredStyles = JSON.parse(saved);
        }
      } catch (error) {
        console.error('加载星标样式失败:', error);
        this.starredStyles = [];
      }
    },

    // 加载用户偏好设置（样式和内容）
    loadUserPreferences() {
      try {
        // 加载样式偏好
        const savedStyle = localStorage.getItem('currentStyle');
        if (savedStyle && STYLES[savedStyle]) {
          this.currentStyle = savedStyle;
        }

        // 加载上次的内容
        const savedContent = localStorage.getItem('markdownInput');
        if (savedContent) {
          this.markdownInput = savedContent;
        } else {
          // 如果没有保存的内容，加载默认示例
          this.loadDefaultExample();
        }
      } catch (error) {
        console.error('加载用户偏好失败:', error);
        // 加载失败时使用默认示例
        this.loadDefaultExample();
      }
    },

    // 保存用户偏好设置
    saveUserPreferences() {
      try {
        // 保存当前样式
        localStorage.setItem('currentStyle', this.currentStyle);

        // 保存当前内容
        localStorage.setItem('markdownInput', this.markdownInput);
      } catch (error) {
        console.error('保存用户偏好失败:', error);
      }
    },

    // 加载默认示例文章
    loadDefaultExample() {
      this.markdownInput = `![](/editor/samples/sample-1.jpg)

# 公众号 Markdown 编辑器

欢迎使用这款专为**微信公众号**设计的 Markdown 编辑器！✨

## 🎯 核心功能

### 1. 智能图片处理

![](/editor/samples/sample-2.jpg)

- **粘贴即用**：支持从任何地方复制粘贴图片（截图、浏览器、文件管理器）
- **自动压缩**：图片自动压缩，平均压缩 50%-80%
- **本地存储**：使用 IndexedDB 持久化，刷新不丢失
- **编辑流畅**：编辑器中使用短链接，告别卡顿

### 2. 多图排版展示

支持朋友圈式的多图网格布局，2-3 列自动排版：

![](/editor/samples/sample-3.jpg)
![](/editor/samples/sample-4.jpg)
![](/editor/samples/sample-5.jpg)

### 3. 13 种精美样式

1. **经典公众号系列**：默认、技术、优雅、深度阅读
2. **传统媒体系列**：杂志、纽约时报、金融时报、Jony Ive
3. **现代数字系列**：Wired、Medium、Apple、陶土暖调、AI Coder

### 4. 一键复制

点击「复制到公众号」按钮，直接粘贴到公众号后台，格式完美保留！

## 💻 代码示例

\`\`\`javascript
// 图片自动压缩并存储到 IndexedDB
const compressedBlob = await imageCompressor.compress(file);
await imageStore.saveImage(imageId, compressedBlob);

// 编辑器中插入短链接
const markdown = \`![图片](img://\${imageId})\`;
\`\`\`

## 📖 引用样式

> 这是一段引用文字，展示编辑器的引用样式效果。
>
> 不同的样式主题会有不同的引用样式，试试切换样式看看效果！

## 📊 表格支持

| 功能 | 支持情况 | 说明 |
|------|---------|------|
| 图片粘贴 | ✅ | 100% 成功率 |
| 刷新保留 | ✅ | IndexedDB 存储 |
| 样式主题 | ✅ | 13 种精选样式 |
| 代码高亮 | ✅ | 多语言支持 |

---

**💡 提示**：

- 试着切换不同的样式主题，体验各种风格的排版效果
- 粘贴图片试试智能压缩功能
- 刷新页面看看内容是否保留

**🌟 开源项目**：如果觉得有用，欢迎访问 [GitHub 仓库](https://github.com/alchaincyf/huasheng_editor) 给个 Star！`;
    },

    handleFileUpload(event) {
      const file = event.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (e) => {
        this.markdownInput = e.target.result;
      };
      reader.onerror = () => {
        this.showToast('文件读取失败', 'error');
      };
      reader.readAsText(file);

      // 清空 input，允许重复上传同一文件
      event.target.value = '';
    },

    async renderMarkdown() {
      if (!this.markdownInput.trim()) {
        this.renderedContent = '';
        return;
      }

      // 检查 markdown-it 是否已初始化
      if (!this.md) {
        console.warn('markdown-it 尚未初始化，跳过渲染');
        return;
      }

      // 预处理 Markdown
      const processedContent = this.preprocessMarkdown(this.markdownInput);

      // 渲染
      let html = this.md.render(processedContent);

      // 处理 img:// 协议（从 IndexedDB 加载图片）
      html = await this.processImageProtocol(html);

      // 应用样式
      html = this.applyInlineStyles(html);

      this.renderedContent = html;
    },

    preprocessMarkdown(content) {
      // 规范化水平分割线格式（修复从飞书等复制时的解析问题）
      // 匹配 * * *、- - -、_ _ _ 等格式（包括带空格的变体）
      // 确保它们被正确解析为 <hr> 而非无序列表
      content = content.replace(/^[ ]{0,3}(\*[ ]*\*[ ]*\*[\* ]*)[ \t]*$/gm, '***');
      content = content.replace(/^[ ]{0,3}(-[ ]*-[ ]*-[- ]*)[ \t]*$/gm, '---');
      content = content.replace(/^[ ]{0,3}(_[ ]*_[ ]*_[_ ]*)[ \t]*$/gm, '___');

      // 修复飞书等复制时的加粗格式断裂问题
      // 例如：**text** **more** -> **text more**（合并相邻的加粗片段）
      // 处理 **空白** (结束后紧跟开始，中间有任意空白) -> 单个空格
      content = content.replace(/\*\*\s+\*\*/g, ' ');
      // 处理 **** 或更多连续星号（通常是格式错误）-> 移除
      content = content.replace(/\*{4,}/g, '');
      // 处理 word** 或 **word 紧贴标点的情况（中文标点）
      // 在中文右标点前的 ** 后添加零宽空格，帮助解析
      content = content.replace(/\*\*([）」』》〉】〕〗］｝"'。，、；：？！])/g, '**\u200B$1');
      // 在中文左标点后的 ** 前添加零宽空格
      content = content.replace(/([（「『《〈【〔〖［｛"'])\*\*/g, '$1\u200B**');
      // 同样处理下划线格式
      content = content.replace(/__\s+__/g, ' ');
      content = content.replace(/_{4,}/g, '');

      // 规范化列表项格式
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n\s*:\s*(.+?)$/gm, '$1: $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?:)\s*\n\s+(.+?)$/gm, '$1 $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+[^:\n]+)\n:\s*(.+?)$/gm, '$1: $2');
      content = content.replace(/^(\s*(?:\d+\.|-|\*)\s+.+?)\n\n\s+(.+?)$/gm, '$1 $2');

      return content;
    },

    // 处理 img:// 协议（从 IndexedDB 加载图片）
    async processImageProtocol(html) {
      if (!this.imageStore) {
        return html;
      }

      // 使用 DOMParser 解析 HTML
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, 'text/html');

      // 查找所有 img 标签
      const images = doc.querySelectorAll('img');

      // 处理每个图片
      for (const img of images) {
        const src = img.getAttribute('src');

        // 检查是否是 img:// 协议
        if (src && src.startsWith('img://')) {
          // 提取图片 ID
          const imageId = src.replace('img://', '');

          try {
            // 从 IndexedDB 获取图片
            let objectURL = this.imageIdToObjectURL[imageId];

            if (!objectURL) {
              // 如果还没有创建 Object URL，现在创建
              objectURL = await this.imageStore.getImage(imageId);

              if (objectURL) {
                // 缓存 Object URL
                this.imageIdToObjectURL[imageId] = objectURL;
              } else {
                console.warn(`图片不存在: ${imageId}`);
                // 图片不存在，显示占位符
                img.setAttribute('src', 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="200" height="200"%3E%3Crect fill="%23ddd" width="200" height="200"/%3E%3Ctext fill="%23999" x="50%25" y="50%25" text-anchor="middle" dy=".3em"%3E图片丢失%3C/text%3E%3C/svg%3E');
                continue;
              }
            }

            // 替换 src 为 Object URL
            img.setAttribute('src', objectURL);

            // 添加 data-image-id 属性（用于复制时识别）
            img.setAttribute('data-image-id', imageId);
          } catch (error) {
            console.error(`加载图片失败 (${imageId}):`, error);
            // 显示错误占位符
            img.setAttribute('src', 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="200" height="200"%3E%3Crect fill="%23fee" width="200" height="200"/%3E%3Ctext fill="%23c00" x="50%25" y="50%25" text-anchor="middle" dy=".3em"%3E加载失败%3C/text%3E%3C/svg%3E');
          }
        }
      }

      return doc.body.innerHTML;
    },

    applyInlineStyles(html) {
      const style = STYLES[this.currentStyle].styles;
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, 'text/html');
      const headingInlineOverrides = {
        strong: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
        em: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
        a: 'color: inherit !important; text-decoration: none !important; border-bottom: 1px solid currentColor !important; background-color: transparent !important;',
        code: 'color: inherit !important; background-color: transparent !important; border: none !important; padding: 0 !important;',
        span: 'color: inherit !important; background-color: transparent !important;',
        b: 'font-weight: 700; color: inherit !important; background-color: transparent !important;',
        i: 'font-style: italic; color: inherit !important; background-color: transparent !important;',
        del: 'color: inherit !important; background-color: transparent !important;',
        mark: 'color: inherit !important; background-color: transparent !important;',
        s: 'color: inherit !important; background-color: transparent !important;',
        u: 'color: inherit !important; text-decoration: underline !important; background-color: transparent !important;',
        ins: 'color: inherit !important; text-decoration: underline !important; background-color: transparent !important;',
        kbd: 'color: inherit !important; background-color: transparent !important; border: none !important; padding: 0 !important;',
        sub: 'color: inherit !important; background-color: transparent !important;',
        sup: 'color: inherit !important; background-color: transparent !important;'
      };
      const headingInlineSelectorList = Object.keys(headingInlineOverrides).join(', ');

      // 先处理图片网格布局（在应用样式之前）
      this.groupConsecutiveImages(doc);

      Object.keys(style).forEach(selector => {
        if (selector === 'pre' || selector === 'code' || selector === 'pre code') {
          return;
        }

        // 跳过已经在网格容器中的图片
        const elements = doc.querySelectorAll(selector);
        elements.forEach(el => {
          // 如果是图片且在网格容器内，跳过样式应用
          if (el.tagName === 'IMG' && el.closest('.image-grid')) {
            return;
          }

          const currentStyle = el.getAttribute('style') || '';
          el.setAttribute('style', currentStyle + '; ' + style[selector]);
        });
      });

      // 标题内的行内元素统一继承标题颜色，避免各主题样式冲突
      const headings = doc.querySelectorAll('h1, h2, h3, h4, h5, h6');
      headings.forEach(heading => {
        const inlineNodes = heading.querySelectorAll(headingInlineSelectorList);
        inlineNodes.forEach(node => {
          const tag = node.tagName.toLowerCase();
          let override = headingInlineOverrides[tag];
          if (!override) {
            return;
          }

          const currentStyle = node.getAttribute('style') || '';
          const sanitizedStyle = currentStyle
            .replace(/color:\s*[^;]+;?/gi, '')
            .replace(/background(?:-color)?:\s*[^;]+;?/gi, '')
            .replace(/border(?:-bottom)?:\s*[^;]+;?/gi, '')
            .replace(/text-decoration:\s*[^;]+;?/gi, '')
            .replace(/box-shadow:\s*[^;]+;?/gi, '')
            .replace(/padding:\s*[^;]+;?/gi, '')
            .replace(/;\s*;/g, ';')
            .trim();
          node.setAttribute('style', sanitizedStyle + '; ' + override);
        });
      });

      const container = doc.createElement('div');
      container.setAttribute('style', style.container);
      container.innerHTML = doc.body.innerHTML;

      return container.outerHTML;
    },

    groupConsecutiveImages(doc) {
      const body = doc.body;
      const children = Array.from(body.children);

      let imagesToProcess = [];

      // 找出所有图片元素，处理两种情况：
      // 1. 多个图片在同一个<p>标签内（连续图片）
      // 2. 每个图片在单独的<p>标签内（分隔的图片）
      children.forEach((child, index) => {
        if (child.tagName === 'P') {
          const images = child.querySelectorAll('img');
          if (images.length > 0) {
            // 如果一个P标签内有多个图片，它们肯定是连续的
            if (images.length > 1) {
              // 多个图片在同一个P标签内，作为一组
              const group = Array.from(images).map(img => ({
                element: child,
                img: img,
                index: index,
                inSameParagraph: true,
                paragraphImageCount: images.length
              }));
              imagesToProcess.push(...group);
            } else if (images.length === 1) {
              // 单个图片在P标签内
              imagesToProcess.push({
                element: child,
                img: images[0],
                index: index,
                inSameParagraph: false,
                paragraphImageCount: 1
              });
            }
          }
        } else if (child.tagName === 'IMG') {
          // 直接是图片元素（少见情况）
          imagesToProcess.push({
            element: child,
            img: child,
            index: index,
            inSameParagraph: false,
            paragraphImageCount: 1
          });
        }
      });

      // 分组逻辑
      let groups = [];
      let currentGroup = [];

      imagesToProcess.forEach((item, i) => {
        if (i === 0) {
          currentGroup.push(item);
        } else {
          const prevItem = imagesToProcess[i - 1];

          // 判断是否连续的条件：
          // 1. 在同一个P标签内的图片肯定是连续的
          // 2. 不同P标签的图片，要看索引是否相邻（差值为1表示相邻）
          let isContinuous = false;

          if (item.index === prevItem.index) {
            // 同一个P标签内的图片
            isContinuous = true;
          } else if (item.index - prevItem.index === 1) {
            // 相邻的P标签，表示连续（没有空行）
            isContinuous = true;
          }
          // 如果索引差大于1，说明中间有其他元素或空行，不连续

          if (isContinuous) {
            currentGroup.push(item);
          } else {
            if (currentGroup.length > 0) {
              groups.push([...currentGroup]);
            }
            currentGroup = [item];
          }
        }
      });

      if (currentGroup.length > 0) {
        groups.push(currentGroup);
      }

      // 对每组图片进行处理
      groups.forEach(group => {
        // 只有2张及以上的图片才需要特殊布局
        if (group.length < 2) return;

        const imageCount = group.length;
        const firstElement = group[0].element;

        // 创建容器
        const gridContainer = doc.createElement('div');
        gridContainer.setAttribute('class', 'image-grid');
        gridContainer.setAttribute('data-image-count', imageCount);

        // 根据图片数量设置网格样式
        let gridStyle = '';
        let columns = 2; // 默认2列

        if (imageCount === 2) {
          gridStyle = `
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 2;
        } else if (imageCount === 3) {
          gridStyle = `
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 3;
        } else if (imageCount === 4) {
          gridStyle = `
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 2;
        } else {
          // 5张及以上，使用3列
          gridStyle = `
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
            margin: 20px auto;
            max-width: 100%;
            align-items: start;
          `;
          columns = 3;
        }

        gridContainer.setAttribute('style', gridStyle);
        gridContainer.setAttribute('data-columns', columns);

        // 将图片添加到容器中
        group.forEach((item) => {
          const imgWrapper = doc.createElement('div');

          imgWrapper.setAttribute('style', `
            width: 100%;
            height: auto;
            overflow: hidden;
          `);

          const img = item.img.cloneNode(true);
          // 修改图片样式以适应容器，添加圆角
          img.setAttribute('style', `
            width: 100%;
            height: auto;
            display: block;
            border-radius: 8px;
          `.trim());

          imgWrapper.appendChild(img);
          gridContainer.appendChild(imgWrapper);
        });

        // 替换原来的图片元素
        firstElement.parentNode.insertBefore(gridContainer, firstElement);

        // 删除原来的图片元素（需要去重，避免重复删除同一个元素）
        const elementsToRemove = new Set();
        group.forEach(item => {
          elementsToRemove.add(item.element);
        });
        elementsToRemove.forEach(element => {
          if (element.parentNode) {
            element.parentNode.removeChild(element);
          }
        });
      });
    },

    convertGridToTable(doc) {
      // 找到所有的图片网格容器
      const imageGrids = doc.querySelectorAll('.image-grid');

      imageGrids.forEach(grid => {
        // 从data属性获取列数（我们在创建时设置的）
        const columns = parseInt(grid.getAttribute('data-columns')) || 2;
        this.convertToTable(doc, grid, columns);
      });
    },

    convertToTable(doc, grid, columns) {
      // 获取所有图片包装器
      const imgWrappers = Array.from(grid.children);

      // 创建 table 元素
      const table = doc.createElement('table');
      table.setAttribute('style', `
        width: 100% !important;
        border-collapse: collapse !important;
        margin: 20px auto !important;
        table-layout: fixed !important;
        border: none !important;
        background: transparent !important;
      `.trim());

      // 计算需要多少行
      const rows = Math.ceil(imgWrappers.length / columns);

      // 创建表格行
      for (let i = 0; i < rows; i++) {
        const tr = doc.createElement('tr');

        // 创建表格单元格
        for (let j = 0; j < columns; j++) {
          const index = i * columns + j;
          const td = doc.createElement('td');

          td.setAttribute('style', `
            padding: 4px !important;
            vertical-align: top !important;
            width: ${100 / columns}% !important;
            border: none !important;
            background: transparent !important;
          `.trim());

          // 如果有对应的图片，添加到单元格
          if (index < imgWrappers.length) {
            const imgWrapper = imgWrappers[index];
            const img = imgWrapper.querySelector('img');

            if (img) {
              // 根据列数设置不同的图片最大高度 - 确保单行最高360px
              let imgMaxHeight;
              let containerHeight;
              if (columns === 2) {
                imgMaxHeight = '340px';  // 2列布局单张最高340px（留出padding空间）
                containerHeight = '360px';  // 容器高度360px
              } else if (columns === 3) {
                imgMaxHeight = '340px';  // 3列布局单张最高340px
                containerHeight = '360px';  // 容器高度360px
              } else {
                imgMaxHeight = '340px';  // 默认高度340px
                containerHeight = '360px';  // 容器高度360px
              }

              // 创建一个新的包装 div - 添加背景和居中样式（使用table-cell方式，更兼容）
              const wrapper = doc.createElement('div');
              wrapper.setAttribute('style', `
                width: 100% !important;
                height: ${containerHeight} !important;
                text-align: center !important;
                background-color: #f5f5f5 !important;
                border-radius: 4px !important;
                padding: 10px !important;
                box-sizing: border-box !important;
                overflow: hidden !important;
                display: table !important;
              `.trim());

              // 创建内部居中容器
              const innerWrapper = doc.createElement('div');
              innerWrapper.setAttribute('style', `
                display: table-cell !important;
                vertical-align: middle !important;
                text-align: center !important;
              `.trim());

              // 克隆图片并直接设置最大高度
              const newImg = img.cloneNode(true);
              newImg.setAttribute('style', `
                max-width: calc(100% - 20px) !important;
                max-height: ${imgMaxHeight} !important;
                width: auto !important;
                height: auto !important;
                display: inline-block !important;
                margin: 0 auto !important;
                border-radius: 4px !important;
                object-fit: contain !important;
              `.trim());

              innerWrapper.appendChild(newImg);
              wrapper.appendChild(innerWrapper);
              td.appendChild(wrapper);
            }
          }

          tr.appendChild(td);
        }

        table.appendChild(tr);
      }

      // 替换网格为 table
      grid.parentNode.replaceChild(table, grid);
    },

    /**
     * 生成粘贴/发送到公众号用的最终 HTML（全部内联样式）。
     * mode 'base64'：图片转 Base64（复制到公众号）；
     * mode 'upload'：每张图片交给 uploadImage(img) 上传，返回的地址写回 src（发送到草稿箱）。
     *   uploadImage 返回 null 表示这张图拿不到，从正文里去掉；抛错则整个流程中止。
     * stripFirstH1：去掉正文里的第一个一级标题（标题已放进草稿标题时用）。
     * 返回 { html, plainText, stats: { total, success, fail, gif, skipped } }。
     */
    async buildWechatHTML({ mode = 'base64', stripFirstH1 = false, uploadImage = null, onImageProgress = null } = {}) {
      const parser = new DOMParser();
      const doc = parser.parseFromString(this.renderedContent, 'text/html');

      if (stripFirstH1) {
        const firstH1 = doc.querySelector('h1');
        if (firstH1) firstH1.remove();
      }

      // 将图片网格转换为 table 布局（公众号兼容）
      this.convertGridToTable(doc);

      const stats = { total: 0, success: 0, fail: 0, gif: 0, skipped: 0 };

      // 处理图片（串行处理，降低内存峰值）
      const images = doc.querySelectorAll('img');
      stats.total = images.length;
      if (images.length > 0) {
        if (mode === 'base64') {
          this.showToast(`正在处理 ${images.length} 张图片...`, 'success');
        }

        let index = 0;
        for (const img of Array.from(images)) {
          index++;
          // 检测 GIF：跳过并替换为提示信息
          let isGif = false;
          try {
            isGif = await this.isGifImage(img);
          } catch (e) {
            isGif = false;
          }
          if (isGif) {
            stats.gif++;
            const placeholder = doc.createElement('p');
            placeholder.setAttribute('style',
              'background-color: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; ' +
              'padding: 12px 16px; color: #856404; font-size: 14px; text-align: center; ' +
              'margin: 16px 0; line-height: 1.6;'
            );
            placeholder.textContent = '⚠️ 此处为 GIF 动图，无法直接复制到公众号排版，请在公众号编辑器中重新上传。';
            img.parentNode.replaceChild(placeholder, img);
            continue;
          }

          if (mode === 'upload') {
            if (onImageProgress) onImageProgress(index, images.length);
            const url = await uploadImage(img); // 上传失败直接抛出，中止发送
            if (url) {
              img.setAttribute('src', url);
              img.removeAttribute('data-image-id');
              stats.success++;
            } else {
              stats.skipped++;
              img.remove();
            }
            continue;
          }

          try {
            const base64 = await this.convertImageToBase64(img);
            img.setAttribute('src', base64);
            stats.success++;
          } catch (error) {
            console.error('图片转换失败:', img.getAttribute('src'), error);
            stats.fail++;
            // 失败时保持原URL
          }
        }
      }

      // Section 容器包裹
      const styleConfig = STYLES[this.currentStyle];
      const containerBg = this.extractBackgroundColor(styleConfig.styles.container);

      if (containerBg && containerBg !== '#fff' && containerBg !== '#ffffff') {
        const section = doc.createElement('section');
        const containerStyle = styleConfig.styles.container;
        const paddingMatch = containerStyle.match(/padding:\s*([^;]+)/);
        const maxWidthMatch = containerStyle.match(/max-width:\s*([^;]+)/);
        const padding = paddingMatch ? paddingMatch[1].trim() : '40px 20px';
        const maxWidth = maxWidthMatch ? maxWidthMatch[1].trim() : '100%';

        section.setAttribute('style',
          `background-color: ${containerBg}; ` +
          `padding: ${padding}; ` +
          `max-width: ${maxWidth}; ` +
          `margin: 0 auto; ` +
          `box-sizing: border-box; ` +
          `word-wrap: break-word;`
        );

        while (doc.body.firstChild) {
          section.appendChild(doc.body.firstChild);
        }

        const allElements = section.querySelectorAll('*');
        allElements.forEach(el => {
          const currentStyle = el.getAttribute('style') || '';
          let newStyle = currentStyle;
          newStyle = newStyle.replace(/max-width:\s*[^;]+;?/g, '');
          newStyle = newStyle.replace(/margin:\s*0\s+auto;?/g, '');
          if (newStyle.includes(`background-color: ${containerBg}`)) {
            newStyle = newStyle.replace(new RegExp(`background-color:\\s*${containerBg.replace(/[()]/g, '\\$&')};?`, 'g'), '');
          }
          newStyle = newStyle.replace(/;\s*;/g, ';').replace(/^\s*;\s*|\s*;\s*$/g, '').trim();
          if (newStyle) {
            el.setAttribute('style', newStyle);
          } else {
            el.removeAttribute('style');
          }
        });

        doc.body.appendChild(section);
      }

      // 代码块简化
      const codeBlocks = doc.querySelectorAll('div[style*="border-radius: 8px"]');
      codeBlocks.forEach(block => {
        const codeElement = block.querySelector('code');
        if (codeElement) {
          const codeText = codeElement.textContent || codeElement.innerText;
          const pre = doc.createElement('pre');
          const code = doc.createElement('code');

          pre.setAttribute('style',
            'background: linear-gradient(to bottom, #2a2c33 0%, #383a42 8px, #383a42 100%);' +
            'padding: 0;' +
            'border-radius: 6px;' +
            'overflow: hidden;' +
            'margin: 24px 0;' +
            'box-shadow: 0 2px 8px rgba(0,0,0,0.15);'
          );

          code.setAttribute('style',
            'color: #abb2bf;' +
            'font-family: "SF Mono", Consolas, Monaco, "Courier New", monospace;' +
            'font-size: 14px;' +
            'line-height: 1.7;' +
            'display: block;' +
            'white-space: pre;' +
            'padding: 16px 20px;' +
            '-webkit-font-smoothing: antialiased;' +
            '-moz-osx-font-smoothing: grayscale;'
          );

          code.textContent = codeText;
          pre.appendChild(code);
          block.parentNode.replaceChild(pre, block);
        }
      });

      // 列表项扁平化
      const listItems = doc.querySelectorAll('li');
      listItems.forEach(li => {
        let text = li.textContent || li.innerText;
        text = text.replace(/\n/g, ' ').replace(/\r/g, ' ').replace(/\s+/g, ' ').trim();
        li.innerHTML = '';
        li.textContent = text;
        const currentStyle = li.getAttribute('style') || '';
        li.setAttribute('style', currentStyle);
      });

      // 深色模式适配：调整引用块样式，使用透明黑色让微信自动转换
      const blockquotes = doc.querySelectorAll('blockquote');
      blockquotes.forEach(blockquote => {
        const currentStyle = blockquote.getAttribute('style') || '';

        // 移除现有的背景色和文字颜色
        let newStyle = currentStyle
          .replace(/background(?:-color)?:\s*[^;]+;?/gi, '')
          .replace(/color:\s*[^;]+;?/gi, '');

        // 添加深色模式友好的样式
        // 使用半透明黑色背景和文字，微信会在深色模式下自动反转
        newStyle += '; background: rgba(0, 0, 0, 0.05) !important';
        newStyle += '; color: rgba(0, 0, 0, 0.8) !important';

        // 清理多余的分号
        newStyle = newStyle.replace(/;\s*;/g, ';').replace(/^\s*;\s*|\s*;\s*$/g, '').trim();
        blockquote.setAttribute('style', newStyle);
      });

      return {
        html: doc.body.innerHTML,
        plainText: doc.body.textContent || '',
        stats
      };
    },

    async copyToClipboard() {
      if (!this.renderedContent) {
        this.showToast('没有内容可复制', 'error');
        return;
      }

      let built = null;
      try {
        built = await this.buildWechatHTML({ mode: 'base64' });

        if (built.stats.gif > 0) {
          this.showToast(`${built.stats.gif} 张 GIF 动图已替换为提示，请在公众号中重新上传`, 'warning');
        }

        if (built.stats.fail > 0) {
          this.showToast(`图片处理完成：${built.stats.success} 成功，${built.stats.fail} 失败（保留原链接）`, 'error');
        }

        const simplifiedHTML = built.html;
        const plainText = built.plainText;

        // 检查焦点：异步处理图片后窗口可能失焦
        if (document.hasFocus()) {
          const htmlBlob = new Blob([simplifiedHTML], { type: 'text/html' });
          const textBlob = new Blob([plainText], { type: 'text/plain' });

          const clipboardItem = new ClipboardItem({
            'text/html': htmlBlob,
            'text/plain': textBlob
          });

          await navigator.clipboard.write([clipboardItem]);

          this.copySuccess = true;
          this.showToast('复制成功', 'success');

          // 自动保存到历史记录
          this.saveToHistory();

          setTimeout(() => {
            this.copySuccess = false;
          }, 2000);
        } else {
          // 焦点丢失，降级到 execCommand
          console.warn('窗口失焦，使用降级复制方案');
          this.clipboardFallback(simplifiedHTML);
        }
      } catch (error) {
        console.error('复制失败:', error);
        // 尝试降级方案
        try {
          const fallbackHTML = built ? built.html : this.renderedContent;
          this.clipboardFallback(fallbackHTML);
        } catch (fallbackError) {
          console.error('降级复制也失败:', fallbackError);
          this.showToast('复制失败', 'error');
        }
      }
    },

    // 复制到 X Articles（纯净语义化 HTML）
    async copyToXArticles() {
      if (!this.renderedContent) return;

      try {
        this.showToast('正在准备 X Articles 格式...', 'processing');

        const parser = new DOMParser();
        const doc = parser.parseFromString(this.renderedContent, 'text/html');

        this.sanitizeForXArticles(doc);

        const html = doc.body.innerHTML;
        const plainText = doc.body.innerText || doc.body.textContent || '';

        if (document.hasFocus()) {
          const blob = new Blob([html], { type: 'text/html' });
          const textBlob = new Blob([plainText], { type: 'text/plain' });
          await navigator.clipboard.write([
            new ClipboardItem({
              'text/html': blob,
              'text/plain': textBlob
            })
          ]);

          this.copyXSuccess = true;
          this.showToast('已复制 X Articles 格式', 'success');
          this.saveToHistory();

          setTimeout(() => {
            this.copyXSuccess = false;
          }, 2000);
        } else {
          console.warn('窗口失焦，使用降级复制方案');
          this.clipboardFallbackX(html);
        }
      } catch (error) {
        console.error('复制到 X 失败:', error);
        try {
          const parser = new DOMParser();
          const doc = parser.parseFromString(this.renderedContent, 'text/html');
          this.sanitizeForXArticles(doc);
          this.clipboardFallbackX(doc.body.innerHTML);
        } catch (fallbackError) {
          console.error('降级复制也失败:', fallbackError);
          this.showToast('复制失败', 'error');
        }
      }
    },

    // X Articles 降级复制
    clipboardFallbackX(html) {
      try {
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = html;
        tempDiv.setAttribute('style', 'position:fixed;left:-9999px;top:-9999px;opacity:0;');
        document.body.appendChild(tempDiv);

        const range = document.createRange();
        range.selectNodeContents(tempDiv);
        const selection = window.getSelection();
        selection.removeAllRanges();
        selection.addRange(range);

        const success = document.execCommand('copy');
        selection.removeAllRanges();
        document.body.removeChild(tempDiv);

        if (success) {
          this.copyXSuccess = true;
          this.showToast('已复制 X Articles 格式（降级模式）', 'success');
          this.saveToHistory();
          setTimeout(() => { this.copyXSuccess = false; }, 2000);
        } else {
          this.showToast('复制失败', 'error');
        }
      } catch (e) {
        console.error('降级复制也失败:', e);
        this.showToast('复制失败', 'error');
      }
    },

    // 清洗 HTML 为 X Articles 兼容格式
    sanitizeForXArticles(doc) {
      // X Articles 白名单标签
      const ALLOWED_TAGS = new Set([
        'h2', 'h3', 'p', 'strong', 'em', 'del', 'a',
        'ul', 'ol', 'li', 'blockquote', 'br', 'b', 'i', 's'
      ]);

      // 1. h1 → h2（X Articles 的 H1 保留给文章标题）
      doc.querySelectorAll('h1').forEach(h1 => {
        const h2 = doc.createElement('h2');
        h2.innerHTML = h1.innerHTML;
        h1.parentNode.replaceChild(h2, h1);
      });

      // 2. h4/h5/h6 → h3（仅支持 H2 和 H3）
      doc.querySelectorAll('h4, h5, h6').forEach(h => {
        const h3 = doc.createElement('h3');
        h3.innerHTML = h.innerHTML;
        h.parentNode.replaceChild(h3, h);
      });

      // 3. 代码块 → blockquote（不支持 pre/code）
      //    匹配代码块容器（div 包含 code 或 pre）
      doc.querySelectorAll('div[style*="border-radius"], pre').forEach(block => {
        const codeEl = block.querySelector('code') || block;
        const codeText = codeEl.textContent || codeEl.innerText || '';
        const bq = doc.createElement('blockquote');
        const lines = codeText.split('\n');
        lines.forEach((line, i) => {
          if (i > 0) bq.appendChild(doc.createElement('br'));
          bq.appendChild(doc.createTextNode(line || '\u00A0'));
        });
        block.parentNode.replaceChild(bq, block);
      });

      // 4. table → blockquote 管道符分隔文本
      doc.querySelectorAll('table').forEach(table => {
        const bq = doc.createElement('blockquote');
        const rows = table.querySelectorAll('tr');
        rows.forEach((row, ri) => {
          if (ri > 0) {
            bq.appendChild(doc.createElement('br'));
          }
          const cells = row.querySelectorAll('th, td');
          const cellTexts = Array.from(cells).map(c => (c.textContent || '').trim());
          bq.appendChild(doc.createTextNode('| ' + cellTexts.join(' | ') + ' |'));
        });
        table.parentNode.replaceChild(bq, table);
      });

      // 5. img → 占位符（不支持 img 标签粘贴）
      doc.querySelectorAll('img').forEach(img => {
        const p = doc.createElement('p');
        const alt = img.getAttribute('alt') || '图片';
        p.textContent = `[${alt}]`;
        // 如果 img 在 p 内，替换整个 p
        const parent = img.parentNode;
        if (parent && parent.tagName === 'P') {
          parent.parentNode.replaceChild(p, parent);
        } else {
          img.parentNode.replaceChild(p, img);
        }
      });

      // 6. hr → 分隔符文本
      doc.querySelectorAll('hr').forEach(hr => {
        const p = doc.createElement('p');
        p.textContent = '———';
        hr.parentNode.replaceChild(p, hr);
      });

      // 7. 移除所有 style/class 属性，解包非白名单标签
      const cleanNode = (node) => {
        if (node.nodeType === Node.TEXT_NODE) return;
        if (node.nodeType !== Node.ELEMENT_NODE) return;

        // 递归处理子节点（从后向前遍历，避免修改导致跳过）
        const children = Array.from(node.childNodes);
        children.forEach(child => cleanNode(child));

        const tag = node.tagName.toLowerCase();

        // 移除所有属性（保留 href）
        if (tag !== 'a') {
          while (node.attributes.length > 0) {
            node.removeAttribute(node.attributes[0].name);
          }
        } else {
          // a 标签只保留 href
          const href = node.getAttribute('href');
          while (node.attributes.length > 0) {
            node.removeAttribute(node.attributes[0].name);
          }
          if (href) node.setAttribute('href', href);
        }

        // 解包非白名单标签（保留内容）
        if (!ALLOWED_TAGS.has(tag) && tag !== 'body') {
          const fragment = doc.createDocumentFragment();
          while (node.firstChild) {
            fragment.appendChild(node.firstChild);
          }
          node.parentNode.replaceChild(fragment, node);
        }
      };

      cleanNode(doc.body);

      // 8. 清理空的 blockquote/p
      doc.querySelectorAll('blockquote, p').forEach(el => {
        if (!el.textContent.trim() && !el.querySelector('br, img')) {
          el.remove();
        }
      });
    },

    // 判断图片是否为 GIF（通过 IndexedDB 记录或 src 后缀判断）
    async isGifImage(imgElement) {
      const imageId = imgElement.getAttribute('data-image-id');
      if (imageId && this.imageStore) {
        try {
          const record = await withTimeout(
            this.imageStore.getImageRecord(imageId),
            3000,
            'GIF 检测超时'
          );
          if (record) {
            const mime = record.mimeType || (record.blob && record.blob.type) || '';
            if (mime === 'image/gif') return true;
          }
        } catch (e) {
          console.warn('GIF 检测失败:', e);
        }
      }
      // 后备：通过 src 后缀判断
      const src = (imgElement.getAttribute('src') || '').toLowerCase();
      return src.endsWith('.gif') || src.includes('.gif?');
    },

    // 复制时压缩大图（>1MB 二次压缩）
    async compressForClipboard(blob, mimeType) {
      if (blob.size > 1024 * 1024) {
        try {
          return await this.recompressForClipboard(blob);
        } catch (e) {
          console.warn('二次压缩失败，使用原图:', e);
        }
      }
      return blob;
    },

    // 二次压缩大图（质量 0.6，最大 1200x1200）
    async recompressForClipboard(blob) {
      return new Promise((resolve, reject) => {
        const img = new Image();
        const url = URL.createObjectURL(blob);
        img.onload = () => {
          try {
            const maxDim = 1200;
            let w = img.naturalWidth;
            let h = img.naturalHeight;
            const ratio = Math.min(maxDim / w, maxDim / h, 1);
            w = Math.round(w * ratio);
            h = Math.round(h * ratio);
            const canvas = document.createElement('canvas');
            canvas.width = w;
            canvas.height = h;
            const ctx = canvas.getContext('2d');
            ctx.fillStyle = '#ffffff';
            ctx.fillRect(0, 0, w, h);
            ctx.drawImage(img, 0, 0, w, h);
            canvas.toBlob(
              (result) => {
                URL.revokeObjectURL(url);
                if (result && result.size < blob.size) {
                  resolve(result);
                } else {
                  resolve(blob); // 压缩后更大则用原图
                }
              },
              'image/jpeg',
              0.6
            );
          } catch (e) {
            URL.revokeObjectURL(url);
            reject(e);
          }
        };
        img.onerror = () => {
          URL.revokeObjectURL(url);
          reject(new Error('图片加载失败'));
        };
        img.src = url;
      });
    },

    // 剪贴板降级方案：使用 execCommand('copy')
    clipboardFallback(html) {
      try {
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = html;
        tempDiv.setAttribute('style', 'position:fixed;left:-9999px;top:-9999px;opacity:0;');
        document.body.appendChild(tempDiv);

        const range = document.createRange();
        range.selectNodeContents(tempDiv);
        const selection = window.getSelection();
        selection.removeAllRanges();
        selection.addRange(range);

        const success = document.execCommand('copy');
        selection.removeAllRanges();
        document.body.removeChild(tempDiv);

        if (success) {
          this.copySuccess = true;
          this.showToast('复制成功（降级模式）', 'success');
          this.saveToHistory();
          setTimeout(() => { this.copySuccess = false; }, 2000);
        } else {
          this.showToast('复制失败', 'error');
        }
      } catch (e) {
        console.error('降级复制也失败:', e);
        this.showToast('复制失败', 'error');
      }
    },

    async convertImageToBase64(imgElement) {
      const src = imgElement.getAttribute('src');

      // 如果已经是Base64，直接返回
      if (src.startsWith('data:')) {
        return src;
      }

      // 优先处理：检查是否有 data-image-id（来自 IndexedDB）
      const imageId = imgElement.getAttribute('data-image-id');
      if (imageId && this.imageStore) {
        try {
          // 从 IndexedDB 获取完整记录（含 mimeType），加 8s 超时
          const record = await withTimeout(
            this.imageStore.getImageRecord(imageId),
            8000,
            `IndexedDB 读取超时: ${imageId}`
          );

          if (record && record.blob) {
            // 复制前压缩：GIF 降级首帧，大图二次压缩
            const processedBlob = await this.compressForClipboard(
              record.blob,
              record.mimeType || record.blob.type || 'image/jpeg'
            );

            // 将 Blob 转为 Base64
            return new Promise((resolve, reject) => {
              const reader = new FileReader();
              reader.onloadend = () => resolve(reader.result);
              reader.onerror = (error) => reject(new Error('FileReader failed: ' + error));
              reader.readAsDataURL(processedBlob);
            });
          } else {
            console.warn(`图片记录不存在: ${imageId}`);
            // 继续尝试用 fetch 方式（兜底）
          }
        } catch (error) {
          console.error(`从 IndexedDB 读取图片失败 (${imageId}):`, error);
          // 继续尝试用 fetch 方式（兜底）
        }
      }

      // 公众号图片（mmbiz）有防盗链：登录后走服务器代理取图
      if (isWechatImageUrl(src) && token()) {
        try {
          const blob = await withTimeout(fetchProxiedImage(src), 15000, '图片代理超时');
          const processedBlob = await this.compressForClipboard(blob, blob.type || 'image/jpeg');
          return await new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result);
            reader.onerror = (error) => reject(new Error('FileReader failed: ' + error));
            reader.readAsDataURL(processedBlob);
          });
        } catch (error) {
          throw new Error(`图片加载失败 (${src}): ${error.message}`);
        }
      }

      // 后备方案：尝试通过 URL 获取图片（加 AbortController 超时）
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 8000);

        const response = await fetch(src, {
          mode: 'cors',
          cache: 'default',
          signal: controller.signal
        });
        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const blob = await response.blob();

        return new Promise((resolve, reject) => {
          const reader = new FileReader();
          reader.onloadend = () => resolve(reader.result);
          reader.onerror = (error) => reject(new Error('FileReader failed: ' + error));
          reader.readAsDataURL(blob);
        });
      } catch (error) {
        // CORS、网络或超时错误时，抛出错误让外层处理
        throw new Error(`图片加载失败 (${src}): ${error.message}`);
      }
    },

    extractBackgroundColor(styleString) {
      if (!styleString) return null;

      const bgColorMatch = styleString.match(/background-color:\s*([^;]+)/);
      if (bgColorMatch) {
        return bgColorMatch[1].trim();
      }

      const bgMatch = styleString.match(/background:\s*([#rgb][^;]+)/);
      if (bgMatch) {
        const bgValue = bgMatch[1].trim();
        if (bgValue.startsWith('#') || bgValue.startsWith('rgb')) {
          return bgValue;
        }
      }

      return null;
    },

    isStyleStarred(styleKey) {
      return this.starredStyles.includes(styleKey);
    },

    isRecommended(styleKey) {
      // 推荐的样式
      const recommended = ['nikkei', 'wechat-anthropic', 'wechat-ft', 'wechat-nyt', 'latepost-depth', 'wechat-tech'];
      return recommended.includes(styleKey);
    },

    toggleStarStyle(styleKey) {
      const index = this.starredStyles.indexOf(styleKey);
      if (index > -1) {
        this.starredStyles.splice(index, 1);
        this.showToast('已取消收藏', 'success');
      } else {
        this.starredStyles.push(styleKey);
        this.showToast('已收藏样式', 'success');
      }
      this.saveStarredStyles();
    },

    saveStarredStyles() {
      try {
        localStorage.setItem('starredStyles', JSON.stringify(this.starredStyles));
      } catch (error) {
        console.error('保存星标样式失败:', error);
      }
    },

    getStyleName(styleKey) {
      const style = STYLES[styleKey];
      return style ? style.name : styleKey;
    },

    showToast(message, type = 'success') {
      this.toast.show = true;
      this.toast.message = message;
      this.toast.type = type;

      clearTimeout(this._toastTimer);
      this._toastTimer = setTimeout(() => {
        this.toast.show = false;
      }, 3000);
    },

    patchMarkdownScanner(md) {
      if (!md || !md.inline || !md.inline.State || this.scanDelimsPatched) {
        return;
      }

      const utils = md.utils;
      const StateInline = md.inline.State;
      const allowLeadingPunctuation = this.createSafeLeadingPunctuationChecker();

      const originalScanDelims = StateInline.prototype.scanDelims;

      StateInline.prototype.scanDelims = function (start, canSplitWord) {
        const max = this.posMax;
        const marker = this.src.charCodeAt(start);

        if (!EMPHASIS_MARKERS.has(marker)) {
          return originalScanDelims.call(this, start, canSplitWord);
        }

        const lastChar = start > 0 ? this.src.charCodeAt(start - 1) : 0x20;

        let pos = start;
        while (pos < max && this.src.charCodeAt(pos) === marker) {
          pos++;
        }

        const count = pos - start;
        const nextChar = pos < max ? this.src.charCodeAt(pos) : 0x20;

        const isLastWhiteSpace = utils.isWhiteSpace(lastChar);
        const isNextWhiteSpace = utils.isWhiteSpace(nextChar);

        let isLastPunctChar =
          utils.isMdAsciiPunct(lastChar) || utils.isPunctChar(String.fromCharCode(lastChar));

        let isNextPunctChar =
          utils.isMdAsciiPunct(nextChar) || utils.isPunctChar(String.fromCharCode(nextChar));

        if (isNextPunctChar && allowLeadingPunctuation(nextChar, marker)) {
          isNextPunctChar = false;
        }

        if (marker === 0x5F /* _ */) {
          if (!isLastWhiteSpace && !isLastPunctChar && isCjkLetter(lastChar)) {
            isLastPunctChar = true;
          }
          if (!isNextWhiteSpace && !isNextPunctChar && isCjkLetter(nextChar)) {
            isNextPunctChar = true;
          }
        }

        // 修复 * 标记：CJK 标点（如中文逗号、句号）不应阻止加粗闭合
        // 例如 **提示词妙招，** 中的 "，" 是 CJK 标点，不应视为 ASCII 标点
        if (marker === 0x2A /* * */) {
          if (isLastPunctChar && isCjkPunctuation(lastChar) && !utils.isMdAsciiPunct(lastChar)) {
            isLastPunctChar = false;
          }
          if (isNextPunctChar && isCjkPunctuation(nextChar) && !utils.isMdAsciiPunct(nextChar)) {
            isNextPunctChar = false;
          }
        }

        const left_flanking =
          !isNextWhiteSpace && (!isNextPunctChar || isLastWhiteSpace || isLastPunctChar);
        const right_flanking =
          !isLastWhiteSpace && (!isLastPunctChar || isNextWhiteSpace || isNextPunctChar);

        const can_open = left_flanking && (canSplitWord || !right_flanking || isLastPunctChar);
        const can_close = right_flanking && (canSplitWord || !left_flanking || isNextPunctChar);

        return { can_open, can_close, length: count };
      };

      this.scanDelimsPatched = true;
    },

    createSafeLeadingPunctuationChecker() {
      const fallbackChars = '「『《〈（【〔〖［｛﹁﹃﹙﹛﹝“‘（';
      const fallbackSet = new Set(
        fallbackChars.split('').map(char => char.codePointAt(0))
      );

      let unicodeRegex = null;
      try {
        unicodeRegex = new RegExp('[\\p{Ps}\\p{Pi}]', 'u');
      } catch (_error) {
        unicodeRegex = null;
      }

      return (charCode, marker) => {
        if (!EMPHASIS_MARKERS.has(marker)) {
          return false;
        }

        if (unicodeRegex) {
          const char = String.fromCharCode(charCode);
          if (unicodeRegex.test(char)) {
            return true;
          }
        }

        return fallbackSet.has(charCode);
      };
    },

    // 初始化 Turndown 服务
    initTurndownService() {
      if (typeof TurndownService === 'undefined') {
        console.warn('Turndown 库未加载，智能粘贴功能将不可用');
        return;
      }

      this.turndownService = new TurndownService({
        headingStyle: 'atx',        // 使用 # 样式的标题
        bulletListMarker: '-',       // 无序列表使用 -
        codeBlockStyle: 'fenced',    // 代码块使用 ```
        fence: '```',                // 代码块围栏
        emDelimiter: '*',            // 斜体使用 *
        strongDelimiter: '**',       // 加粗使用 **
        linkStyle: 'inlined'         // 链接使用内联样式
      });

      // 配置表格支持
      this.turndownService.keep(['table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td']);

      // 自定义规则：保留表格结构
      this.turndownService.addRule('table', {
        filter: 'table',
        replacement: (_content, node) => {
          // 简单的表格转换为 Markdown 表格
          const rows = Array.from(node.querySelectorAll('tr'));
          if (rows.length === 0) return '';

          let markdown = '\n\n';
          let headerProcessed = false;

          rows.forEach((row, index) => {
            const cells = Array.from(row.querySelectorAll('td, th'));
            const cellContents = cells.map(cell => {
              // 清理单元格内容
              const text = cell.textContent.replace(/\n/g, ' ').trim();
              return text;
            });

            if (cellContents.length > 0) {
              markdown += '| ' + cellContents.join(' | ') + ' |\n';

              // 第一行后添加分隔符
              if (index === 0 || (!headerProcessed && row.querySelector('th'))) {
                markdown += '| ' + cells.map(() => '---').join(' | ') + ' |\n';
                headerProcessed = true;
              }
            }
          });

          return markdown + '\n';
        }
      });

      // 自定义规则：优化图片处理
      this.turndownService.addRule('image', {
        filter: 'img',
        replacement: (_content, node) => {
          const alt = node.alt || '图片';
          const src = node.src || '';
          const title = node.title || '';

          // 处理 base64 图片（截取前30个字符作为标识）
          if (src.startsWith('data:image')) {
            const type = src.match(/data:image\/(\w+);/)?.[1] || 'image';
            return `![${alt}](data:image/${type};base64,...)${title ? ` "${title}"` : ''}\n`;
          }

          return `![${alt}](${src})${title ? ` "${title}"` : ''}\n`;
        }
      });
    },

    // 处理粘贴事件
    async handleSmartPaste(event) {
      console.log('===== handleSmartPaste 被调用 =====');

      const clipboardData = event.clipboardData || event.originalEvent?.clipboardData;

      if (!clipboardData) {
        console.log('不支持 clipboardData');
        return; // 不支持的浏览器，使用默认行为
      }

      // 调试模式（需要时可以打开）
      const DEBUG = true;
      if (DEBUG) {
        console.log('剪贴板数据类型:', Array.from(clipboardData.types || []));
      }

      // 检查是否有文件（某些应用复制图片会作为文件）
      if (clipboardData.files && clipboardData.files.length > 0) {
        if (DEBUG) console.log('检测到文件:', clipboardData.files[0]);
        const file = clipboardData.files[0];
        if (file && file.type && file.type.startsWith('image/')) {
          event.preventDefault();
          await this.handleImageUpload(file, event.target);
          return;
        }
      }

      // 检查 items（浏览器复制的图片通常在这里）
      const items = clipboardData.items;
      if (items) {
        for (let item of items) {
          if (DEBUG) console.log('Item 类型:', item.type, 'Kind:', item.kind);

          // 检查是否是图片
          if (item.kind === 'file' && item.type && item.type.indexOf('image') !== -1) {
            event.preventDefault();
            const file = item.getAsFile();
            if (file) {
              await this.handleImageUpload(file, event.target);
              return; // 处理完图片就返回
            }
          }
        }
      }

      // 获取剪贴板中的各种格式数据
      const htmlData = clipboardData.getData('text/html');
      const textData = clipboardData.getData('text/plain');

      // 检查是否是类似 [Image #2] 这样的占位符文本
      if (textData && /^\[Image\s*#?\d*\]$/i.test(textData.trim())) {
        if (DEBUG) console.warn('检测到图片占位符文本，但无法获取实际图片数据');
        this.showToast('⚠️ 请尝试：截图工具 / 浏览器复制 / 拖拽文件', 'error');
        event.preventDefault();
        return; // 不插入占位符文本
      }

      if (DEBUG) {
        console.log('纯文本数据:', textData?.substring(0, 200));
        console.log('HTML 数据:', htmlData?.substring(0, 200));
        console.log('是否检测为 Markdown:', textData && this.isMarkdown(textData));
        console.log('是否有 turndownService:', !!this.turndownService);
      }

      // 检查是否来自 IDE/代码编辑器的 HTML（需要特殊处理）
      const isFromIDE = this.isIDEFormattedHTML(htmlData, textData);

      if (DEBUG) {
        console.log('是否来自 IDE:', isFromIDE);
      }

      if (isFromIDE && textData && this.isMarkdown(textData)) {
        // 来自 IDE 的 Markdown 代码，直接使用纯文本（避免转义）
        if (DEBUG) console.log('检测到 IDE 复制的 Markdown 代码，使用纯文本');
        return; // 使用默认粘贴行为
      }

      // 处理 HTML 数据（富文本编辑器或其他来源）
      if (htmlData && htmlData.trim() !== '' && this.turndownService) {
        // 检查是否是从代码编辑器复制的（精确匹配真正的代码块标签，避免误判）
        // 只有当 HTML 主要由 <pre> 或 <code> 组成时才跳过转换
        const hasPreTag = /<pre[\s>]/.test(htmlData);
        const hasCodeTag = /<code[\s>]/.test(htmlData);
        const isMainlyCode = (hasPreTag || hasCodeTag) && !htmlData.includes('<p') && !htmlData.includes('<div');

        if (isMainlyCode) {
          // 真正的代码编辑器内容，使用纯文本
          if (DEBUG) console.log('检测到代码编辑器格式，使用纯文本');
          return; // 使用默认粘贴行为
        }

        // 检查 HTML 中是否包含本地文件路径的图片（如 file:/// 协议）
        if (htmlData.includes('file:///') || htmlData.includes('src="file:')) {
          if (DEBUG) console.warn('检测到本地文件路径的图片，无法直接上传');
          this.showToast('⚠️ 本地图片请直接拖拽文件到编辑器', 'error');
          event.preventDefault();
          return;
        }

        event.preventDefault(); // 阻止默认粘贴

        try {
          // 将 HTML 转换为 Markdown
          let markdown = this.turndownService.turndown(htmlData);

          // 清理多余的空行
          markdown = markdown.replace(/\n{3,}/g, '\n\n');

          // 获取当前光标位置
          const textarea = event.target;
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          const value = textarea.value;

          // 插入转换后的 Markdown
          const newValue = value.substring(0, start) + markdown + value.substring(end);

          // 更新文本框内容
          this.markdownInput = newValue;

          // 恢复光标位置
          this.$nextTick(() => {
            textarea.selectionStart = textarea.selectionEnd = start + markdown.length;
            textarea.focus();
          });

          // 显示提示
          this.showToast('✨ 已智能转换为 Markdown 格式', 'success');
        } catch (error) {
          if (DEBUG) console.error('HTML 转 Markdown 失败:', error);
          // 转换失败，使用纯文本
          this.insertTextAtCursor(event.target, textData);
        }
      }
      // 检查纯文本是否为 Markdown（后备方案，只有在没有 HTML 时才检查）
      else if (textData && this.isMarkdown(textData)) {
        // 已经是 Markdown，直接使用纯文本
        if (DEBUG) console.log('没有 HTML，但检测到 Markdown 格式，使用纯文本');
        return; // 使用默认粘贴行为
      }
      // 普通文本，使用默认粘贴行为
      else {
        if (DEBUG) console.log('普通文本，使用默认粘贴行为');
        return; // 使用默认行为
      }
    },

    // 检测文本是否为 Markdown 格式
    isMarkdown(text) {
      if (!text) return false;

      // Markdown 特征模式
      const patterns = [
        /^#{1,6}\s+/m,           // 标题
        /\*\*[^*]+\*\*/,         // 加粗
        /\*[^*\n]+\*/,           // 斜体
        /\[[^\]]+\]\([^)]+\)/,   // 链接
        /!\[[^\]]*\]\([^)]+\)/,  // 图片
        /^[\*\-\+]\s+/m,         // 无序列表
        /^\d+\.\s+/m,            // 有序列表
        /^>\s+/m,                // 引用
        /`[^`]+`/,               // 内联代码
        /```[\s\S]*?```/,        // 代码块
        /^\|.*\|$/m,             // 表格
        /<!--.*?-->/,            // HTML 注释（我们的图片注释）
        /^---+$/m                // 分隔线
      ];

      // 计算匹配的特征数量
      const matchCount = patterns.filter(pattern => pattern.test(text)).length;

      // 如果有 2 个或以上的 Markdown 特征，认为是 Markdown
      // 或者如果包含我们的图片注释，也认为是 Markdown
      return matchCount >= 2 || text.includes('<!-- img:');
    },

    // 检测 HTML 是否来自 IDE/代码编辑器
    isIDEFormattedHTML(htmlData, textData) {
      if (!htmlData || !textData) return false;

      // IDE 复制的 HTML 特征（VS Code、Cursor、Sublime Text 等）
      const ideSignatures = [
        // VS Code 特征
        /<meta\s+charset=['"]utf-8['"]/i,
        /<div\s+class=["']ace_line["']/,
        /style=["'][^"']*font-family:\s*['"]?(?:Consolas|Monaco|Menlo|Courier)/i,

        // 简单的 div/span 结构（没有富文本语义标签）
        // 检查：有 HTML 标签，但几乎没有 <p>, <h1-h6>, <strong>, <em> 等富文本标签
        function(html) {
          const hasDivSpan = /<(?:div|span)[\s>]/.test(html);
          const hasSemanticTags = /<(?:p|h[1-6]|strong|em|ul|ol|li|blockquote)[\s>]/i.test(html);
          // 如果有 div/span 但几乎没有语义标签，可能是代码编辑器
          return hasDivSpan && !hasSemanticTags;
        },

        // 检查 HTML 是否只是简单包裹纯文本（几乎没有格式化）
        function(html) {
          // 去除所有 HTML 标签，看是否与纯文本几乎一致
          const strippedHtml = html.replace(/<[^>]+>/g, '').trim();
          const similarity = strippedHtml === textData.trim();
          return similarity;
        }
      ];

      // 检查是否匹配任何 IDE 特征
      let matchCount = 0;
      for (const signature of ideSignatures) {
        if (typeof signature === 'function') {
          if (signature(htmlData)) matchCount++;
        } else if (signature.test(htmlData)) {
          matchCount++;
        }
      }

      // 如果匹配 2 个或以上特征，认为是 IDE 格式
      return matchCount >= 2;
    },

    // 在光标位置插入文本
    insertTextAtCursor(textarea, text) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const value = textarea.value;

      const newValue = value.substring(0, start) + text + value.substring(end);
      this.markdownInput = newValue;

      this.$nextTick(() => {
        textarea.selectionStart = textarea.selectionEnd = start + text.length;
        textarea.focus();
      });
    },

    // 处理图片上传 - 压缩并存储到 IndexedDB
    async handleImageUpload(file, textarea) {
      // 检查文件类型
      if (!file.type.startsWith('image/')) {
        this.showToast('请上传图片文件', 'error');
        return;
      }

      // 检查文件大小（10MB限制）
      const maxSize = 10 * 1024 * 1024;
      if (file.size > maxSize) {
        this.showToast('图片大小不能超过 10MB', 'error');
        return;
      }

      const imageName = file.name.replace(/\.[^/.]+$/, '') || '图片';
      const originalSize = file.size;

      try {
        // 第一步：压缩图片
        this.showToast('🔄 正在压缩图片...', 'success');

        const compressedBlob = await this.imageCompressor.compress(file);
        const compressedSize = compressedBlob.size;

        // 计算压缩率
        const compressionRatio = ((1 - compressedSize / originalSize) * 100).toFixed(0);
        console.log(`图片压缩完成: ${ImageCompressor.formatSize(originalSize)} → ${ImageCompressor.formatSize(compressedSize)} (压缩 ${compressionRatio}%)`);

        // 第二步：生成唯一 ID
        const imageId = `img-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // 第三步：存储到 IndexedDB
        await this.imageStore.saveImage(imageId, compressedBlob, {
          name: imageName,
          originalName: file.name,
          originalSize: originalSize,
          compressedSize: compressedSize,
          compressionRatio: compressionRatio,
          mimeType: compressedBlob.type || file.type
        });

        // 第四步：插入 img:// 协议的短链接到编辑器
        const markdownImage = `![${imageName}](img://${imageId})`;

        if (textarea) {
          const currentPos = textarea.selectionStart;
          const before = this.markdownInput.substring(0, currentPos);
          const after = this.markdownInput.substring(currentPos);

          this.markdownInput = before + markdownImage + after;

          this.$nextTick(() => {
            const newPos = currentPos + markdownImage.length;
            textarea.selectionStart = textarea.selectionEnd = newPos;
            textarea.focus();
          });
        } else {
          this.markdownInput += '\n' + markdownImage;
        }

        // 第五步：显示成功提示
        if (compressionRatio > 10) {
          this.showToast(`✅ 已保存 (${ImageCompressor.formatSize(originalSize)} → ${ImageCompressor.formatSize(compressedSize)})`, 'success');
        } else {
          this.showToast(`✅ 已保存 (${ImageCompressor.formatSize(compressedSize)})`, 'success');
        }
      } catch (error) {
        console.error('图片处理失败:', error);
        this.showToast('❌ 图片处理失败: ' + error.message, 'error');
      }
    },

    // 处理文件拖拽
    handleDrop(event) {
      event.preventDefault();
      event.stopPropagation();

      this.isDraggingOver = false;

      const files = event.dataTransfer.files;
      if (files.length > 0) {
        const file = files[0];
        if (file.type.startsWith('image/')) {
          this.handleImageUpload(file, event.target);
        } else {
          this.showToast('只支持拖拽图片文件', 'error');
        }
      }
    },

    // 阻止默认拖拽行为
    handleDragOver(event) {
      event.preventDefault();
      event.stopPropagation();
      event.dataTransfer.dropEffect = 'copy';
      this.isDraggingOver = true;
    },

    // 处理拖拽进入
    handleDragEnter(event) {
      event.preventDefault();
      this.isDraggingOver = true;
    },

    // 处理拖拽离开
    handleDragLeave(event) {
      event.preventDefault();
      // 只有当真正离开编辑器时才移除状态
      if (event.target.classList.contains('markdown-input')) {
        this.isDraggingOver = false;
      }
    },

    // ============ 小红书功能相关方法 ============

    // 生成小红书图片
    async generateXiaohongshuImages() {
      if (!this.renderedContent) {
        this.showToast('没有内容可生成', 'error');
        return;
      }

      // html2canvas 按需加载（小红书功能目前在界面上隐藏，避免拖大首屏包）
      let html2canvas;
      try {
        html2canvas = (await import('html2canvas')).default;
      } catch (error) {
        console.error('html2canvas 加载失败:', error);
        this.showToast('html2canvas 库未加载', 'error');
        return;
      }

      this.xiaohongshuGenerating = true;
      this.xiaohongshuImages = [];

      try {
        // 创建临时渲染容器
        const tempContainer = this.createXiaohongshuContainer();
        document.body.appendChild(tempContainer);

        // 计算文章信息
        const articleInfo = this.calculateArticleInfo();

        // 分页
        const pages = await this.splitContentIntoPages(tempContainer, articleInfo);

        if (pages.length === 0) {
          throw new Error('内容为空，无法生成图片');
        }

        // 生成每一页的图片
        for (let i = 0; i < pages.length; i++) {
          const pageElement = pages[i];

          // 添加页码
          this.addPageNumber(pageElement, i + 1, pages.length);

          // 如果是首页，添加信息面板
          if (i === 0) {
            this.addInfoPanel(pageElement, articleInfo);
          }

          // 将页面元素添加到容器中，确保 html2canvas 可以找到它
          tempContainer.appendChild(pageElement);

          // 等待一小段时间确保元素渲染完成
          await new Promise(resolve => setTimeout(resolve, 100));

          // 生成图片
          const canvas = await html2canvas(pageElement, {
            scale: 2,
            useCORS: true,
            allowTaint: true,
            backgroundColor: this.getBackgroundColor(),
            width: 750,
            height: 1000,
            windowWidth: 750,
            windowHeight: 1000,
            logging: false
          });

          const dataUrl = canvas.toDataURL('image/png');
          this.xiaohongshuImages.push({
            dataUrl: dataUrl,
            pageNumber: i + 1,
            totalPages: pages.length
          });

          // 移除页面元素，准备下一页
          tempContainer.removeChild(pageElement);
        }

        // 清理临时容器
        document.body.removeChild(tempContainer);

        this.showToast(`成功生成 ${pages.length} 张小红书图片`, 'success');
      } catch (error) {
        console.error('生成小红书图片失败:', error);
        this.showToast('生成失败: ' + error.message, 'error');

        // 确保清理临时容器
        const existingContainer = document.querySelector('div[style*="-9999px"]');
        if (existingContainer) {
          document.body.removeChild(existingContainer);
        }
      } finally {
        this.xiaohongshuGenerating = false;
      }
    },

    // 创建小红书渲染容器
    createXiaohongshuContainer() {
      const container = document.createElement('div');
      container.style.position = 'fixed';
      container.style.left = '-9999px';
      container.style.top = '0';
      container.style.width = '750px';
      container.style.pointerEvents = 'none';
      container.style.zIndex = '-1';
      // 不设置 visibility: hidden，因为 html2canvas 需要可见元素
      return container;
    },

    // 计算文章信息
    calculateArticleInfo() {
      const parser = new DOMParser();
      const doc = parser.parseFromString(this.renderedContent, 'text/html');

      // 计算字数（去除HTML标签）
      const textContent = doc.body.textContent || '';
      const charCount = textContent.replace(/\s/g, '').length;

      // 计算阅读时长（假设每分钟阅读400字）
      const readingTime = Math.ceil(charCount / 400);

      // 计算图片数量
      const imageCount = doc.querySelectorAll('img').length;

      return {
        charCount,
        readingTime,
        imageCount
      };
    },

    // 分页算法 - 完全简化版本
    async splitContentIntoPages(container, articleInfo) {
      // 解析 Markdown 为纯文本结构（不使用复杂的渲染样式）
      const simplifiedContent = this.createSimplifiedContent();

      const pages = [];
      const maxPageHeight = 850; // 留出空间给页码和首页信息面板

      // 创建测量容器
      const measureContainer = this.createPageElement();
      container.appendChild(measureContainer);

      let currentPageContent = [];
      let currentHeight = 0;
      const firstPageOffset = 120; // 首页信息面板占用空间

      for (let i = 0; i < simplifiedContent.length; i++) {
        const block = simplifiedContent[i];

        // 创建元素
        const element = this.createSimplifiedElement(block);

        // 添加到测量容器
        measureContainer.appendChild(element);
        const elementHeight = element.offsetHeight || 50;

        // 计算是否超出页面高度
        const heightLimit = pages.length === 0 ? maxPageHeight - firstPageOffset : maxPageHeight;
        const wouldExceed = currentHeight + elementHeight > heightLimit;

        if (wouldExceed && currentPageContent.length > 0) {
          // 创建新页面
          const page = this.createPageElement();
          currentPageContent.forEach(el => page.appendChild(el));
          pages.push(page);

          currentPageContent = [];
          currentHeight = 0;
        }

        // 从测量容器移除
        measureContainer.removeChild(element);
        currentPageContent.push(element);
        currentHeight += elementHeight;
      }

      // 添加最后一页
      if (currentPageContent.length > 0) {
        const page = this.createPageElement();
        currentPageContent.forEach(el => page.appendChild(el));
        pages.push(page);
      }

      // 清理测量容器
      container.removeChild(measureContainer);

      return pages;
    },

    // 创建简化的内容结构（纯文本，无复杂样式）
    createSimplifiedContent() {
      const lines = this.markdownInput.split('\n');
      const content = [];

      lines.forEach(line => {
        line = line.trim();
        if (!line) return;

        // 标题
        if (line.startsWith('# ')) {
          content.push({ type: 'h1', text: line.substring(2) });
        } else if (line.startsWith('## ')) {
          content.push({ type: 'h2', text: line.substring(3) });
        } else if (line.startsWith('### ')) {
          content.push({ type: 'h3', text: line.substring(4) });
        }
        // 列表
        else if (line.startsWith('- ') || line.startsWith('* ')) {
          content.push({ type: 'li', text: line.substring(2) });
        }
        // 引用
        else if (line.startsWith('> ')) {
          content.push({ type: 'quote', text: line.substring(2) });
        }
        // 代码块标记（跳过）
        else if (line.startsWith('```')) {
          // 跳过代码块
        }
        // 图片（跳过，小红书图片由外链显示）
        else if (line.startsWith('![')) {
          // 跳过图片
        }
        // 分隔线
        else if (line === '---') {
          content.push({ type: 'hr' });
        }
        // 普通段落
        else {
          // 移除 Markdown 格式标记
          let text = line.replace(/\*\*(.+?)\*\*/g, '$1'); // 粗体
          text = text.replace(/\*(.+?)\*/g, '$1'); // 斜体
          text = text.replace(/`(.+?)`/g, '$1'); // 行内代码
          content.push({ type: 'p', text: text });
        }
      });

      return content;
    },

    // 创建简化的元素（只使用基本的内联样式）
    createSimplifiedElement(block) {
      const el = document.createElement('div');

      switch (block.type) {
        case 'h1':
          el.textContent = block.text;
          el.style.fontSize = '28px';
          el.style.fontWeight = 'bold';
          el.style.margin = '20px 0 10px 0';
          el.style.color = '#000';
          break;
        case 'h2':
          el.textContent = block.text;
          el.style.fontSize = '24px';
          el.style.fontWeight = 'bold';
          el.style.margin = '16px 0 8px 0';
          el.style.color = '#000';
          break;
        case 'h3':
          el.textContent = block.text;
          el.style.fontSize = '20px';
          el.style.fontWeight = 'bold';
          el.style.margin = '12px 0 6px 0';
          el.style.color = '#333';
          break;
        case 'p':
          el.textContent = block.text;
          el.style.fontSize = '16px';
          el.style.lineHeight = '1.8';
          el.style.margin = '8px 0';
          el.style.color = '#333';
          break;
        case 'li':
          el.textContent = '• ' + block.text;
          el.style.fontSize = '16px';
          el.style.lineHeight = '1.8';
          el.style.margin = '4px 0';
          el.style.paddingLeft = '10px';
          el.style.color = '#333';
          break;
        case 'quote':
          el.textContent = block.text;
          el.style.fontSize = '15px';
          el.style.lineHeight = '1.8';
          el.style.margin = '8px 0';
          el.style.padding = '10px 15px';
          el.style.borderLeft = '3px solid #0066FF';
          el.style.background = '#f5f5f5';
          el.style.color = '#666';
          break;
        case 'hr':
          el.style.height = '1px';
          el.style.background = '#ddd';
          el.style.margin = '20px 0';
          el.style.border = 'none';
          break;
      }

      return el;
    },

    // 创建页面元素
    createPageElement() {
      const page = document.createElement('div');
      page.style.width = '750px';
      page.style.height = '1000px';
      page.style.backgroundColor = this.getBackgroundColor();
      page.style.padding = '80px 40px 40px 40px';
      page.style.boxSizing = 'border-box';
      page.style.position = 'relative';
      page.style.overflow = 'hidden';
      page.style.fontFamily = 'Arial';
      page.style.fontSize = '16px';
      page.style.lineHeight = '1.8';
      page.style.color = '#333';
      return page;
    },

    // 添加页码
    addPageNumber(pageElement, currentPage, totalPages) {
      const pageNumber = document.createElement('div');
      pageNumber.textContent = `${currentPage}/${totalPages}`;
      pageNumber.style.position = 'absolute';
      pageNumber.style.bottom = '30px';
      pageNumber.style.right = '40px';
      pageNumber.style.fontSize = '14px';
      pageNumber.style.color = '#999';
      pageNumber.style.fontWeight = '500';
      pageElement.appendChild(pageNumber);
    },

    // 添加首页信息面板
    addInfoPanel(pageElement, articleInfo) {
      const panel = document.createElement('div');
      panel.style.position = 'absolute';
      panel.style.top = '20px';
      panel.style.left = '40px';
      panel.style.right = '40px';
      panel.style.padding = '20px';
      panel.style.backgroundColor = '#E6F0FF';
      panel.style.borderRadius = '8px';
      panel.style.border = '1px solid #99CCFF';

      const infoItems = [
        { label: '字数', value: articleInfo.charCount },
        { label: '阅读', value: `${articleInfo.readingTime}分钟` },
        { label: '图片', value: `${articleInfo.imageCount}张` }
      ];

      // 创建容器（使用 table 布局）
      const table = document.createElement('table');
      table.style.width = '100%';
      table.style.borderCollapse = 'collapse';
      const tr = document.createElement('tr');

      infoItems.forEach(item => {
        const td = document.createElement('td');
        td.style.textAlign = 'center';
        td.style.padding = '5px';

        const valueDiv = document.createElement('div');
        valueDiv.textContent = item.value;
        valueDiv.style.fontSize = '24px';
        valueDiv.style.fontWeight = 'bold';
        valueDiv.style.color = '#0066FF';
        valueDiv.style.marginBottom = '4px';

        const labelDiv = document.createElement('div');
        labelDiv.textContent = item.label;
        labelDiv.style.fontSize = '12px';
        labelDiv.style.color = '#666';

        td.appendChild(valueDiv);
        td.appendChild(labelDiv);
        tr.appendChild(td);
      });

      table.appendChild(tr);
      panel.appendChild(table);

      // 插入到页面顶部
      pageElement.insertBefore(panel, pageElement.firstChild);
    },

    // 获取背景色
    getBackgroundColor() {
      const styleConfig = STYLES[this.currentStyle];
      if (styleConfig && styleConfig.styles && styleConfig.styles.container) {
        const bgColor = this.extractBackgroundColor(styleConfig.styles.container);
        return bgColor || '#FFFFFF';
      }
      return '#FFFFFF';
    },

    // 下载单张小红书图片
    downloadXiaohongshuImage(image, index) {
      const link = document.createElement('a');
      link.download = `小红书-第${index + 1}张-共${this.xiaohongshuImages.length}张.png`;
      link.href = image.dataUrl;
      link.click();
      this.showToast(`下载第 ${index + 1} 张图片`, 'success');
    },

    // 批量下载小红书图片
    async downloadAllXiaohongshuImages() {
      if (this.xiaohongshuImages.length === 0) {
        this.showToast('没有图片可下载', 'error');
        return;
      }

      this.showToast(`开始下载 ${this.xiaohongshuImages.length} 张图片...`, 'success');

      for (let i = 0; i < this.xiaohongshuImages.length; i++) {
        const image = this.xiaohongshuImages[i];

        // 添加延迟，避免浏览器阻止批量下载
        await new Promise(resolve => setTimeout(resolve, 300));

        const link = document.createElement('a');
        link.download = `小红书-第${i + 1}张-共${this.xiaohongshuImages.length}张.png`;
        link.href = image.dataUrl;
        link.click();
      }

      this.showToast('批量下载完成', 'success');
    },

    // ==================== 文章历史记录功能 ====================

    // 从 Markdown 内容提取标题
    extractTitle(markdownContent) {
      if (!markdownContent || !markdownContent.trim()) {
        return '无标题';
      }

      // 尝试匹配第一个 # 标题
      const titleMatch = markdownContent.match(/^#\s+(.+)$/m);
      if (titleMatch && titleMatch[1]) {
        // 清理标题中的 markdown 格式
        let title = titleMatch[1].trim();
        title = title.replace(/\*\*/g, '').replace(/\*/g, '').replace(/`/g, '');
        return title.substring(0, 50); // 最多 50 字符
      }

      // 如果没有标题，取前 20 个字符
      const cleanContent = markdownContent
        .replace(/^!\[.*?\]\(.*?\)$/gm, '') // 移除图片
        .replace(/^#+\s*/gm, '') // 移除标题标记
        .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // 移除链接格式
        .replace(/[*_~`]/g, '') // 移除格式标记
        .trim();

      if (cleanContent) {
        return cleanContent.substring(0, 20) + (cleanContent.length > 20 ? '...' : '');
      }

      return '无标题';
    },

    // 保存当前文章到历史记录
    saveToHistory() {
      const content = this.markdownInput;
      if (!content || !content.trim()) {
        this.showToast('内容为空，无法保存', 'error');
        return;
      }

      const title = this.extractTitle(content);
      const now = Date.now();

      // 如果有当前文章ID，直接更新该文章
      if (this.currentArticleId) {
        const existingIndex = this.articleHistory.findIndex(
          article => article.id === this.currentArticleId
        );

        if (existingIndex !== -1) {
          // 更新已存在的文章
          this.articleHistory[existingIndex].title = title;
          this.articleHistory[existingIndex].content = content;
          this.articleHistory[existingIndex].style = this.currentStyle;
          this.articleHistory[existingIndex].updatedAt = now;

          // 移到最前面
          const article = this.articleHistory.splice(existingIndex, 1)[0];
          this.articleHistory.unshift(article);

          this.saveArticleHistory();
          this.showToast('已更新历史记录', 'success');
          return;
        }
      }

      // 没有当前文章ID，创建新文章
      const newArticleId = `article-${now}-${Math.random().toString(36).substring(2, 8)}`;
      const newArticle = {
        id: newArticleId,
        title: title,
        content: content,
        style: this.currentStyle,
        createdAt: now,
        updatedAt: now
      };

      // 添加到列表开头
      this.articleHistory.unshift(newArticle);

      // 设置为当前文章
      this.currentArticleId = newArticleId;

      // 限制最多 20 篇
      if (this.articleHistory.length > 20) {
        this.articleHistory = this.articleHistory.slice(0, 20);
      }

      // 保存到 localStorage
      this.saveArticleHistory();
      this.showToast('已保存到历史记录', 'success');
    },

    // 从历史记录加载文章
    loadFromHistory(articleId) {
      const article = this.articleHistory.find(a => a.id === articleId);
      if (!article) {
        this.showToast('文章不存在', 'error');
        return;
      }

      // 恢复内容和样式
      this.markdownInput = article.content;
      if (article.style && STYLES[article.style]) {
        this.currentStyle = article.style;
      }

      // 设置当前文章ID，后续编辑会更新这篇文章
      this.currentArticleId = articleId;

      // 关闭侧边栏
      this.showHistoryPanel = false;

      this.showToast('已加载文章', 'success');
    },

    // 从历史记录删除文章
    deleteFromHistory(articleId) {
      const index = this.articleHistory.findIndex(a => a.id === articleId);
      if (index === -1) {
        this.showToast('文章不存在', 'error');
        return;
      }

      this.articleHistory.splice(index, 1);
      this.saveArticleHistory();
      this.showToast('已删除', 'success');
    },

    // 从 localStorage 加载历史记录
    loadArticleHistory() {
      try {
        const saved = localStorage.getItem('articleHistory');
        if (saved) {
          const data = JSON.parse(saved);
          if (data && Array.isArray(data.articles)) {
            this.articleHistory = data.articles;
          }
        }
      } catch (error) {
        console.error('加载历史记录失败:', error);
        this.articleHistory = [];
      }
    },

    // 保存历史记录到 localStorage
    saveArticleHistory() {
      try {
        const data = {
          articles: this.articleHistory
        };
        localStorage.setItem('articleHistory', JSON.stringify(data));
      } catch (error) {
        console.error('保存历史记录失败:', error);
        this.showToast('保存历史记录失败', 'error');
      }
    },

    // 切换历史记录侧边栏
    toggleHistoryPanel() {
      this.showHistoryPanel = !this.showHistoryPanel;
    },

    // 格式化历史记录时间显示
    formatHistoryDate(timestamp) {
      if (!timestamp) return '';

      const date = new Date(timestamp);
      const now = new Date();
      const diff = now - date;

      // 不到 1 分钟
      if (diff < 60 * 1000) {
        return '刚刚';
      }

      // 不到 1 小时
      if (diff < 60 * 60 * 1000) {
        const minutes = Math.floor(diff / (60 * 1000));
        return `${minutes} 分钟前`;
      }

      // 不到 24 小时
      if (diff < 24 * 60 * 60 * 1000) {
        const hours = Math.floor(diff / (60 * 60 * 1000));
        return `${hours} 小时前`;
      }

      // 今年内
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hour = String(date.getHours()).padStart(2, '0');
      const minute = String(date.getMinutes()).padStart(2, '0');

      if (year === now.getFullYear()) {
        return `${month}-${day} ${hour}:${minute}`;
      }

      // 往年
      return `${year}-${month}-${day}`;
    }
  }
};
</script>
