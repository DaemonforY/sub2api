<template>
  <details class="ed-guide" :open="open">
    <summary>怎么获取 AppID、AppSecret，并设置 IP 白名单？</summary>
    <ol>
      <li>用公众号<b>管理员</b>的微信扫码登录 <a :href="devPlatformUrl" target="_blank" rel="noopener">微信开发者平台 ↗</a>。</li>
      <li>进入「我的业务 → 公众号」，点开你的公众号 →「基础信息」，复制 <b>AppID</b>（wx 开头的 18 位）。</li>
      <li>
        同一页的「开发密钥」里获取 <b>AppSecret</b>：平台不会再显示已生成的密钥，第一次用点「启用」，忘了就点「重置」，管理员扫码确认后复制 32 位密钥。
        <span class="ed-guide-warn">重置后旧密钥就不能用了，其他用这个密钥的工具也要一起改。</span>
      </li>
      <li>
        同一处的「API IP 白名单」点「编辑」，加入本站服务器 IP
        <code>{{ serverIp }}</code>
        <button class="ed-btn small" type="button" @click="copyIp">{{ copied ? '已复制' : '复制' }}</button>
        ，保存后等几分钟生效。白名单整个账号共用，原有的 IP 不要删。
      </li>
      <li>回到这里「添加公众号」，填好后点「测试连接」。</li>
    </ol>
    <p class="ed-guide-foot">
      老后台的「设置与开发 → 开发接口管理 → 基本配置」已逐步迁到开发者平台，找不到时以开发者平台为准。
      个人或未认证的公众号可能用不了草稿箱接口，见<a :href="faqUrl" target="_blank" rel="noopener">常见问题</a>。
      <a :href="tutorialUrl" target="_blank" rel="noopener">图文教程 ↗</a>
    </p>
  </details>
</template>

<script>
import { SERVER_IP, DEV_PLATFORM_URL, TUTORIAL_SETUP_URL, TUTORIAL_FAQ_URL } from '../lib/guide.js';

export default {
  name: 'WechatSetupGuide',
  props: {
    open: { type: Boolean, default: false }
  },
  data() {
    return {
      serverIp: SERVER_IP,
      devPlatformUrl: DEV_PLATFORM_URL,
      tutorialUrl: TUTORIAL_SETUP_URL,
      faqUrl: TUTORIAL_FAQ_URL,
      copied: false
    };
  },
  methods: {
    async copyIp() {
      try {
        await navigator.clipboard.writeText(SERVER_IP);
        this.copied = true;
        setTimeout(() => { this.copied = false; }, 2000);
      } catch {
        window.prompt('复制这个 IP：', SERVER_IP);
      }
    }
  }
};
</script>
