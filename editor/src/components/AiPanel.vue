<template>
  <aside class="ai-drawer" :class="{ open }" aria-label="AI 助手">
    <div class="ai-drawer-header">
      <h3>AI 助手</h3>
      <button class="ed-close" title="关闭" @click="$emit('close')">×</button>
    </div>

    <div class="ai-tabs">
      <button class="ai-tab" :class="{ active: tab === 'text' }" @click="tab = 'text'">写作</button>
      <button class="ai-tab" :class="{ active: tab === 'image' }" @click="tab = 'image'">配图</button>
    </div>

    <div class="ai-body">
      <!-- 登录 / Key 检查 -->
      <div v-if="!editor.auth.loggedIn" class="ed-login-box">
        AI 助手需要先登录 HiveGPT，并使用你自己的 Key。<br>
        <a class="ed-btn primary" :href="editor.loginHref">登录</a>
      </div>
      <div v-else-if="!editor.aiKey" class="ed-login-box">
        还没有选择 AI Key。<br>
        AI 助手的费用记在你选择的 Key 上，请先在「设置」里选一个。<br>
        <button class="ed-btn primary" @click="$emit('open-settings')">打开设置</button>
      </div>

      <!-- 写作 -->
      <template v-else-if="tab === 'text'">
        <div class="ed-label" style="margin-bottom: 6px;">处理范围</div>
        <div class="ai-segment">
          <button :class="{ active: scope === 'selection' }" @click="scope = 'selection'">选中文本</button>
          <button :class="{ active: scope === 'all' }" @click="scope = 'all'">全文</button>
        </div>
        <div class="ed-hint" style="margin-top: -6px;">
          <template v-if="scope === 'selection'">
            {{ editor.editorSelection.length ? `已选中 ${editor.editorSelection.length} 个字符` : '先在左侧编辑器里选中一段文字' }}
          </template>
          <template v-else>全文 {{ editor.markdownInput.length }} 个字符（最多 20000）</template>
        </div>

        <div class="ed-label" style="margin-bottom: 6px;">要做什么</div>
        <div class="ai-chips">
          <button v-for="a in actions" :key="a.key" class="ai-chip" :class="{ active: action === a.key }" @click="action = a.key">{{ a.label }}</button>
        </div>

        <div v-if="action === 'custom'" class="ed-field">
          <label for="ai-instruction">修改要求</label>
          <textarea id="ai-instruction" v-model="instruction" class="ed-textarea" rows="2" placeholder="例如：改成更口语化的语气，加两个小标题"></textarea>
        </div>

        <div class="ed-row">
          <button v-if="!running" class="ed-btn primary" @click="run()">开始</button>
          <button v-else class="ed-btn" @click="stop">停止</button>
          <span v-if="running" class="ed-counter"><span class="ed-spinner"></span> 正在生成…</span>
        </div>

        <div v-if="error" class="ed-msg err">{{ error }}</div>

        <template v-if="result || running">
          <!-- 拟标题：每行一个，点击设为草稿标题 -->
          <div v-if="last && last.action === 'title' && !running && titleOptions.length" style="margin-top: 12px;">
            <div class="ed-hint">点击一个标题，设为草稿标题：</div>
            <button v-for="t in titleOptions" :key="t" class="ai-title-item" @click="useTitle(t)">{{ t }}</button>
          </div>
          <div v-else class="ai-result" :class="{ empty: !result }">{{ result || '…' }}</div>

          <div v-if="!running && result" class="ai-result-actions">
            <template v-if="last && last.action !== 'title'">
              <button class="ed-btn small" @click="replace">替换</button>
              <button class="ed-btn small" @click="insertAfter">插入到后面</button>
            </template>
            <button class="ed-btn small" @click="copy">复制</button>
            <button class="ed-btn small" @click="run(true)">重新生成</button>
            <button v-if="last && last.action === 'digest'" class="ed-btn small" @click="useDigest">填入草稿摘要</button>
          </div>
        </template>
      </template>

      <!-- 配图 -->
      <template v-else>
        <div class="ed-field">
          <div class="ed-label-row">
            <label for="ai-image-prompt">画面描述</label>
            <button class="ed-btn small" :disabled="promptBusy || imageBusy" @click="writePrompt">
              <span v-if="promptBusy" class="ed-spinner"></span>
              {{ promptBusy ? '撰写中' : '根据选中内容写描述' }}
            </button>
          </div>
          <textarea id="ai-image-prompt" v-model="prompt" class="ed-textarea" rows="4" placeholder="描述想要的画面，例如：一只橘猫趴在窗台上看雨，暖色调"></textarea>
        </div>

        <div class="ed-label" style="margin-bottom: 6px;">风格（可选）</div>
        <div class="ai-chips">
          <button v-for="s in imageStyles" :key="s" class="ai-chip" :class="{ active: imageStyle === s }" @click="imageStyle = imageStyle === s ? '' : s">{{ s }}</button>
        </div>

        <div class="ed-label" style="margin-bottom: 6px;">尺寸</div>
        <div class="ai-segment">
          <button v-for="s in sizes" :key="s.value" :class="{ active: size === s.value }" @click="size = s.value">{{ s.label }}</button>
        </div>

        <label class="ed-check-line" style="margin-bottom: 12px;">
          <input v-model="caption" type="checkbox">
          图下注明「AI 生成配图」
        </label>

        <div class="ed-row">
          <button class="ed-btn primary" :disabled="imageBusy || promptBusy || !prompt.trim()" @click="generateImage">
            {{ imageBusy ? '生成中…' : '生成并插入' }}
          </button>
          <span class="ed-counter">插入到编辑器的光标处</span>
        </div>

        <div v-if="imageError" class="ed-msg err">{{ imageError }}</div>

        <div v-if="imageBusy" class="ai-waiting">
          <span class="ed-spinner"></span> 正在生成配图… {{ elapsed }}s<br>
          <span class="ed-counter">通常需要 30–60 秒，可以继续写作</span>
        </div>

        <div v-if="lastImage && !imageBusy" class="ai-image-result">
          <img :src="lastImage.url" alt="AI 配图">
          <div class="ed-row">
            <button class="ed-btn small" @click="setCover">设为封面</button>
            <span class="ed-counter">已插入到编辑器，并保存在本浏览器里</span>
          </div>
        </div>
      </template>
    </div>

    <div v-if="editor.auth.loggedIn && editor.aiKey" class="ai-cost">
      <span>费用记在你选择的 Key 上：{{ editor.aiKey.name || `#${editor.aiKey.id}` }}</span>
      <button class="ed-btn small" @click="$emit('open-settings')">更换</button>
    </div>
  </aside>
</template>

<script>
import { streamAiText, aiImage } from '../lib/api.js';
import { b64ToBlob } from '../lib/imageTools.js';

const MAX_TEXT = 20000;

export default {
  name: 'AiPanel',
  inject: ['editor'],
  props: {
    open: { type: Boolean, default: false }
  },
  emits: ['close', 'open-settings'],
  data() {
    return {
      tab: 'text',
      scope: 'all',
      action: 'polish',
      instruction: '',
      actions: [
        { key: 'polish', label: '润色' },
        { key: 'shorten', label: '精简' },
        { key: 'expand', label: '扩写' },
        { key: 'title', label: '拟标题' },
        { key: 'digest', label: '写摘要' },
        { key: 'outline', label: '列提纲' },
        { key: 'custom', label: '自定义' }
      ],
      running: false,
      result: '',
      error: '',
      last: null,
      // 配图
      prompt: '',
      promptBusy: false,
      imageStyles: ['日系二次元', '扁平插画', '水彩', '写实摄影', '科技感 3D'],
      imageStyle: '',
      sizes: [
        { value: '1536x1024', label: '横图' },
        { value: '1024x1536', label: '竖图' },
        { value: '1024x1024', label: '方图' }
      ],
      size: '1536x1024',
      caption: true,
      imageBusy: false,
      imageError: '',
      elapsed: 0,
      lastImage: null
    };
  },
  computed: {
    titleOptions() {
      return this.result
        .split('\n')
        .map(line => line
          .replace(/^\s*(?:\d+\s*[.、)）:：]|[-*•·])\s*/, '')
          .replace(/\*\*/g, '')
          .replace(/^[《“"「]+|[》”"」]+$/g, '')
          .trim())
        .filter(Boolean)
        .slice(0, 10);
    }
  },
  watch: {
    open(v) {
      if (v) this.scope = this.editor.editorSelection.length ? 'selection' : 'all';
    }
  },
  beforeUnmount() {
    if (this._abort) this._abort.abort();
    clearInterval(this._timer);
    if (this.lastImage) URL.revokeObjectURL(this.lastImage.url);
  },
  methods: {
    async run(again = false) {
      if (this.running) return;
      this.error = '';
      let job;
      if (again && this.last) {
        job = { ...this.last };
      } else {
        const sel = this.editor.getSelectionInfo();
        const all = this.scope === 'all';
        if (!all && !sel.text) {
          this.error = '请先在左侧编辑器里选中要处理的文字，或切换到「全文」';
          return;
        }
        const text = all ? this.editor.markdownInput : sel.text;
        if (!text.trim()) {
          this.error = '没有可处理的内容';
          return;
        }
        if (text.length > MAX_TEXT) {
          this.error = `内容太长（${text.length} 字），一次最多处理 ${MAX_TEXT} 字，请选中一部分再试`;
          return;
        }
        if (this.action === 'custom' && !this.instruction.trim()) {
          this.error = '请填写修改要求';
          return;
        }
        job = {
          action: this.action,
          instruction: this.action === 'custom' ? this.instruction.trim() : '',
          text,
          start: all ? 0 : sel.start,
          end: all ? this.editor.markdownInput.length : sel.end
        };
      }
      this.last = job;
      this.result = '';
      this.running = true;
      this._abort = new AbortController();
      try {
        await streamAiText({
          key_id: this.editor.aiKey.id,
          action: job.action,
          text: job.text,
          instruction: job.instruction || undefined,
          title: this.editor.draftMeta.title || undefined
        }, delta => { this.result += delta; }, this._abort.signal);
        this.result = this.result.replace(/^\s+|\s+$/g, '');
        if (!this.result) this.error = 'AI 没有返回内容，请重试';
      } catch (error) {
        if (error && error.name === 'AbortError') {
          this.error = '已停止';
        } else {
          this.editor.noteApiError(error);
          this.error = error.message || '生成失败';
        }
      } finally {
        this.running = false;
        this._abort = null;
      }
    },
    stop() {
      if (this._abort) this._abort.abort();
    },
    replace() {
      const job = this.last;
      if (!job || !this.result) return;
      const current = this.editor.markdownInput.slice(job.start, job.end);
      if (current !== job.text) {
        this.editor.showToast('左侧内容已经改动过，无法直接替换。请重新选中后再生成，或用「插入到后面」', 'error');
        return;
      }
      this.editor.replaceRange(job.start, job.end, this.result);
      this.last = { ...job, text: this.result, end: job.start + this.result.length };
      this.editor.showToast('已替换', 'success');
    },
    insertAfter() {
      const job = this.last;
      if (!job || !this.result) return;
      const pos = Math.min(job.end, this.editor.markdownInput.length);
      this.editor.insertBlockAt(pos, this.result);
      this.editor.showToast('已插入', 'success');
    },
    async copy() {
      try {
        await navigator.clipboard.writeText(this.result);
        this.editor.showToast('已复制', 'success');
      } catch {
        this.editor.showToast('复制失败，请手动选择文字复制', 'error');
      }
    },
    useTitle(t) {
      this.editor.setDraftTitle(t, 'ai');
      this.editor.showToast(`已设为草稿标题：${t}`, 'success');
    },
    useDigest() {
      this.editor.draftMeta.digest = this.result.replace(/\s+/g, ' ').trim().slice(0, 120);
      this.editor.showToast('已填入草稿摘要', 'success');
    },

    // ---------- 配图 ----------
    async writePrompt() {
      const sel = this.editor.getSelectionInfo();
      let text = sel.text;
      if (!text.trim()) {
        text = this.editor.markdownInput;
        this.editor.showToast('没有选中文字，根据全文写描述', 'success');
      }
      if (!text.trim()) {
        this.imageError = '没有可参考的内容';
        return;
      }
      this.promptBusy = true;
      this.imageError = '';
      this.prompt = '';
      try {
        await streamAiText({ key_id: this.editor.aiKey.id, action: 'image_prompt', text: text.slice(0, MAX_TEXT), title: this.editor.draftMeta.title || undefined },
          delta => { this.prompt += delta; });
        this.prompt = this.prompt.trim();
      } catch (error) {
        this.editor.noteApiError(error);
        this.imageError = error.message || '生成描述失败';
      } finally {
        this.promptBusy = false;
      }
    },
    async generateImage() {
      if (this.imageBusy || this.promptBusy || !this.prompt.trim()) return;
      const pos = this.editor.getSelectionInfo().end;
      const fullPrompt = this.prompt.trim() + (this.imageStyle ? `\n风格：${this.imageStyle}` : '');
      this.imageBusy = true;
      this.imageError = '';
      this.elapsed = 0;
      const started = Date.now();
      this._timer = setInterval(() => { this.elapsed = Math.round((Date.now() - started) / 1000); }, 1000);
      try {
        const res = await aiImage(this.editor.aiKey.id, fullPrompt, this.size);
        if (!res || !res.b64_json) throw new Error('AI 没有返回图片，请重试');
        const blob = b64ToBlob(res.b64_json, res.mime || 'image/png');
        const id = await this.editor.storeImageBlob(blob, 'AI 配图');
        let block = `![AI 配图](img://${id})`;
        if (this.caption) {
          block += '\n\n<span style="display: block; text-align: center; font-size: 12px; color: #999; font-style: italic;">AI 生成配图</span>';
        }
        this.editor.insertBlockAt(pos, block);
        if (this.lastImage) URL.revokeObjectURL(this.lastImage.url);
        this.lastImage = { id, url: URL.createObjectURL(blob) };
        this.editor.showToast('配图已插入', 'success');
      } catch (error) {
        this.editor.noteApiError(error);
        this.imageError = error.message || '生成配图失败';
      } finally {
        clearInterval(this._timer);
        this.imageBusy = false;
      }
    },
    setCover() {
      if (!this.lastImage) return;
      this.editor.draftMeta.coverImageId = this.lastImage.id;
      this.editor.showToast('已设为封面（发送到草稿箱时使用）', 'success');
    }
  }
};
</script>
