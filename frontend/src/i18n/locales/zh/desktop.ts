export default {
  desktopLogin: {
    title: '登录 HiveGPT 桌面助手',
    subtitle: '在电脑上用 AI 帮你处理文件、整理表格',
    asking: '电脑「{device}」正在请求登录你的账号。',
    explain: '确认后，这台电脑会获得一个专用 Key，用你账号的额度（试用或订阅优先）。随时可以在「API 密钥」里停用它。',
    notYou: '如果这不是你刚刚在自己电脑上打开的二维码，请点「不是我」。',
    phishing: '只确认你面前这台电脑。别人发给你的登录二维码（说是领福利、官方验证、帮忙登录等）一律点「不是我」：确认后对方的电脑会用你账号的余额和额度。',
    network: '电脑所在网络：{network}（{ago}发起）',
    sameNetwork: '这台电脑和你的手机在同一个网络。',
    otherNetwork: '这台电脑和你的手机不在同一个网络。手机用流量、电脑连 Wi-Fi 时也会这样；但如果电脑不在你身边，请点「不是我」。',
    confirmNearby: '这台电脑就在我面前，是我自己在登录',
    justNow: '刚刚',
    minutesAgo: '{n} 分钟前',
    unknownNetwork: '未知',
    approve: '确认登录',
    reject: '不是我',
    approved: '已确认，回到电脑上就可以开始用了',
    approvedHint: '电脑会在几秒内自动完成登录，这个页面可以关掉了。',
    approvedNotMe: '不是你本人操作的？立刻到「API 密钥」里停用「桌面助手 · {device}」。',
    goKeys: '去 API 密钥',
    rejected: '已拒绝这次登录',
    rejectedHint: '电脑上不会登录你的账号。',
    invalid: '这个登录二维码已经失效或不存在，请在电脑上刷新二维码后重新扫码。',
    handled: '这个登录请求已经处理过了，请回到电脑上查看。',
    loadFailed: '加载失败，请刷新页面重试'
  }
}
