<template>
  <div class="ed-overlay" @click.self="$emit('close')">
    <div class="ed-dialog" role="dialog" aria-label="设置">
      <div class="ed-dialog-header">
        <h3>设置</h3>
        <button class="ed-close" title="关闭" @click="$emit('close')">×</button>
      </div>

      <div class="ed-dialog-body">
        <!-- 公众号账号 -->
        <section class="ed-section">
          <h4>公众号账号</h4>
          <p class="ed-hint">
            AppSecret 只保存在本浏览器；在公众号后台「设置与开发 → 基本配置」获取 AppID / AppSecret，并把本站服务器 IP <b>43.133.80.169</b> 加入「IP 白名单」（以「测试连接」的提示为准）。
          </p>

          <div v-if="!state.accounts.length && !form" class="ed-empty">还没有添加公众号账号</div>

          <div v-for="acc in state.accounts" :key="acc.id" class="ed-account">
            <label class="ed-check-line" title="发送草稿时默认使用这个账号">
              <input type="radio" name="ed-default-account" :checked="state.defaultId === acc.id" @change="setDefault(acc.id)">
              默认
            </label>
            <div class="ed-account-info">
              <div class="ed-account-name">{{ acc.name || '未命名公众号' }}</div>
              <div class="ed-account-meta">{{ acc.appid }}</div>
              <div v-if="results[acc.id]" class="ed-msg" :class="results[acc.id].ok ? 'ok' : 'err'">{{ results[acc.id].message }}</div>
            </div>
            <div class="ed-account-actions">
              <button class="ed-btn small" :disabled="testing === acc.id" @click="test(acc, acc.id)">
                <span v-if="testing === acc.id" class="ed-spinner"></span>
                {{ testing === acc.id ? '测试中' : '测试连接' }}
              </button>
              <button class="ed-btn small" @click="edit(acc)">编辑</button>
              <button class="ed-btn small danger" @click="remove(acc)">删除</button>
            </div>
          </div>

          <div v-if="form" class="ed-form">
            <div class="ed-field">
              <label for="ed-acc-name">备注名</label>
              <input id="ed-acc-name" v-model.trim="form.name" class="ed-input" maxlength="40" placeholder="例如：我的技术号">
            </div>
            <div class="ed-field">
              <label for="ed-acc-appid">AppID</label>
              <input id="ed-acc-appid" v-model.trim="form.appid" class="ed-input" maxlength="64" placeholder="wx 开头的 18 位" autocomplete="off" spellcheck="false">
            </div>
            <div class="ed-field">
              <label for="ed-acc-secret">AppSecret</label>
              <div class="ed-row">
                <input
                  id="ed-acc-secret"
                  v-model.trim="form.secret"
                  class="ed-input"
                  style="flex: 1;"
                  :type="showSecret ? 'text' : 'password'"
                  maxlength="128"
                  autocomplete="off"
                  spellcheck="false"
                  placeholder="32 位开发者密码"
                >
                <button class="ed-btn small" type="button" @click="showSecret = !showSecret">{{ showSecret ? '隐藏' : '显示' }}</button>
              </div>
            </div>
            <div v-if="results.form" class="ed-msg" :class="results.form.ok ? 'ok' : 'err'">{{ results.form.message }}</div>
            <div class="ed-row end" style="margin-top: 10px;">
              <button class="ed-btn" @click="cancelForm">取消</button>
              <button class="ed-btn" :disabled="testing === 'form' || !formValid" @click="test(form, 'form')">
                <span v-if="testing === 'form'" class="ed-spinner"></span>
                {{ testing === 'form' ? '测试中' : '测试连接' }}
              </button>
              <button class="ed-btn primary" :disabled="!formValid" @click="saveForm">保存</button>
            </div>
          </div>
          <button v-else class="ed-btn" style="margin-top: 10px;" @click="add">+ 添加公众号</button>
        </section>

        <!-- AI Key -->
        <section class="ed-section">
          <h4>AI Key</h4>
          <p class="ed-hint">AI 助手、AI 配图和 AI 封面使用你自己的 HiveGPT Key，费用记在你选择的 Key 上。</p>

          <div v-if="!editor.auth.loggedIn" class="ed-msg info">
            <a class="ed-link" :href="editor.loginHref">登录</a> 后选择 Key。
          </div>
          <template v-else>
            <div v-if="keysLoading" class="ed-empty"><span class="ed-spinner"></span> 正在加载 Key…</div>
            <div v-else-if="keysError" class="ed-msg err">
              {{ keysError }}
              <button class="ed-btn small" style="margin-left: 6px;" @click="loadKeys">重试</button>
            </div>
            <div v-else-if="!keys.length" class="ed-msg info">
              还没有可用的 Key。<a class="ed-link" href="/keys" target="_blank" rel="noopener">去创建一个</a>（分组选「GPT-按量」），创建后点
              <button class="ed-btn small" @click="loadKeys">刷新</button>
            </div>
            <div v-else class="ed-row">
              <select v-model="selectedKeyId" class="ed-select" style="flex: 1;" @change="chooseKey">
                <option :value="0">— 请选择 —</option>
                <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}{{ k.group ? `（${k.group}）` : '' }}</option>
              </select>
              <button class="ed-btn small" @click="loadKeys">刷新</button>
            </div>
            <p v-if="keys.length" class="ed-hint" style="margin-top: 8px;">
              只列出 GPT 分组的 Key。需要新的 Key？<a href="/keys" target="_blank" rel="noopener">去创建</a>（分组选「GPT-按量」）。
            </p>
          </template>
        </section>
      </div>

      <div class="ed-dialog-footer">
        <button class="ed-btn primary" @click="$emit('close')">完成</button>
      </div>
    </div>
  </div>
</template>

<script>
import { loadAccounts, saveAccounts, newAccountId, saveAiKey } from '../lib/settings.js';
import { wechatCheck, listKeys } from '../lib/api.js';

export default {
  name: 'SettingsDialog',
  inject: ['editor'],
  emits: ['close'],
  data() {
    return {
      state: loadAccounts(),
      form: null,
      showSecret: false,
      testing: '',
      results: {},
      keys: [],
      keysLoading: false,
      keysError: '',
      selectedKeyId: this.editor.aiKey ? this.editor.aiKey.id : 0
    };
  },
  computed: {
    formValid() {
      return !!(this.form && this.form.appid && this.form.secret);
    }
  },
  mounted() {
    if (this.editor.auth.loggedIn) this.loadKeys();
  },
  methods: {
    persist() {
      saveAccounts(this.state);
    },
    setDefault(id) {
      this.state.defaultId = id;
      this.persist();
    },
    add() {
      this.form = { id: '', name: '', appid: '', secret: '' };
      this.showSecret = false;
      delete this.results.form;
    },
    edit(acc) {
      this.form = { ...acc };
      this.showSecret = false;
      delete this.results.form;
    },
    cancelForm() {
      this.form = null;
    },
    saveForm() {
      if (!this.formValid) return;
      if (!/^wx[0-9a-zA-Z]{16}$/.test(this.form.appid)) {
        if (!window.confirm('AppID 一般是 wx 开头的 18 位字符，确定要保存吗？')) return;
      }
      const item = { id: this.form.id || newAccountId(), name: this.form.name, appid: this.form.appid, secret: this.form.secret };
      const idx = this.state.accounts.findIndex(a => a.id === item.id);
      if (idx >= 0) this.state.accounts.splice(idx, 1, item);
      else this.state.accounts.push(item);
      if (!this.state.defaultId) this.state.defaultId = item.id;
      if (this.results.form) this.results[item.id] = this.results.form;
      this.persist();
      this.form = null;
      this.editor.showToast('公众号账号已保存', 'success');
    },
    remove(acc) {
      if (!window.confirm(`删除公众号「${acc.name || acc.appid}」？（只删除本浏览器里保存的信息）`)) return;
      this.state.accounts = this.state.accounts.filter(a => a.id !== acc.id);
      if (this.state.defaultId === acc.id) this.state.defaultId = this.state.accounts[0]?.id || '';
      delete this.results[acc.id];
      this.persist();
    },
    async test(acc, slot) {
      if (!this.editor.auth.loggedIn) {
        this.results[slot] = { ok: false, message: '请先登录 HiveGPT，再测试连接。' };
        return;
      }
      this.testing = slot;
      try {
        await wechatCheck(acc);
        this.results[slot] = { ok: true, message: '连接成功，可以发送草稿了。' };
      } catch (error) {
        this.editor.noteApiError(error);
        this.results[slot] = { ok: false, message: error.message };
      } finally {
        this.testing = '';
      }
    },
    async loadKeys() {
      this.keysLoading = true;
      this.keysError = '';
      try {
        const keys = await listKeys();
        this.keys = Array.isArray(keys) ? keys : [];
        const current = this.editor.aiKey;
        if (current && !this.keys.some(k => k.id === current.id)) {
          // 保存的 Key 已不存在（被删或换了分组）
          this.selectedKeyId = 0;
          this.applyKey(null);
        } else if (!current && this.keys.length === 1) {
          this.selectedKeyId = this.keys[0].id;
          this.applyKey(this.keys[0]);
        }
      } catch (error) {
        this.editor.noteApiError(error);
        this.keysError = error.message;
      } finally {
        this.keysLoading = false;
      }
    },
    chooseKey() {
      const key = this.keys.find(k => k.id === Number(this.selectedKeyId)) || null;
      this.applyKey(key);
      if (key) this.editor.showToast(`AI 将使用 Key「${key.name}」`, 'success');
    },
    applyKey(key) {
      const value = key ? { id: key.id, name: key.name } : null;
      saveAiKey(value);
      this.editor.aiKey = value;
    }
  }
};
</script>
