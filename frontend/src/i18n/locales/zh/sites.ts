export default {
  sites: {
    title: '我的网站',
    description: '把 HTML 网页一键发布成网站：上传一个 .html 文件，或一个包含 index.html 的 .zip 压缩包，马上得到一个可以分享的网址。订阅用户专享。',
    unavailable: '网站发布暂未开放，请稍后再来。',
    subscribeFirst: '网站发布只对订阅用户开放，购买任意订阅套餐后即可发布网站。',
    lapsedNotice: '你的订阅已到期：网站还会保留 {days} 天，之后暂停访问；续订后自动恢复，期间不能发布或更新。',
    subscribe: '购买订阅',
    quota: {
      sites: '网站数',
      free: '订阅赠送',
      freeValue: '已用 {used} / {total} 个',
      extra: '超出赠送的网站',
      extraValue: '¥{price} / 个 / 30 天，从余额扣',
      limits: '单个网站',
      limitsValue: '最大 {mb}MB，最多 {files} 个文件'
    },
    create: {
      title: '发布新网站',
      hint: '网站地址会自动生成，形如 https://abc1234.{domain}。压缩包里要有 index.html 作为首页，可以带 CSS、JS、图片、字体等文件。',
      name: '网站名称（只有你能看到）',
      namePlaceholder: '例如：活动落地页',
      file: '网页文件（.html 或 .zip）',
      publish: '发布',
      publishPaid: '发布（¥{price} / 30 天）'
    },
    rules: '只能托管静态网页，不支持 PHP 等服务器程序。禁止发布钓鱼、诈骗、赌博、色情、恶意程序和侵权内容，违规网站会被直接下线且不退款。每个页面右下角会显示「由 HiveGPT 托管 · 举报」。',
    publishing: '上传中…',
    published: '已发布：{url}',
    updated: '已更新',
    renewed: '已续费，网站已恢复访问',
    deleted: '已删除',
    copied: '网址已复制',
    wrongFile: '只能上传 .html 文件或 .zip 压缩包',
    tooLarge: '文件不能超过 {mb}MB',
    empty: '还没有网站。',
    free: '赠送',
    paidUntil: '已付费至 {date}',
    meta: '{size} · {files} 个文件 · 第 {version} 版 · 更新于 {time}',
    status: { active: '访问中', disabled: '已下线', unpaid: '欠费暂停', lapsed: '订阅到期暂停' },
    actions: { copy: '复制网址', update: '更新', renew: '续费 ¥{price}', delete: '删除' },
    edit: { title: '更新网站', file: '新的网页文件（可选）', fileHint: '不选文件就只修改名称；上传后网址不变，立刻替换成新内容。' },
    deleteConfirm: '删除后 {url} 立即无法访问，文件也会被删除，不能恢复。确定删除吗？',
    charges: { title: '扣费记录', time: '时间', site: '网站', amount: '金额', until: '有效期至' }
  },
  siteReport: {
    title: '举报网站',
    description: '发现 HiveGPT 托管的网站含有钓鱼、诈骗等违规内容？请告诉我们，我们会尽快核实处理。',
    site: '网站名（网址最前面那一段）',
    reason: '举报原因',
    reasons: { phishing: '钓鱼 / 仿冒登录', fraud: '诈骗', gambling: '赌博', porn: '色情', malware: '恶意程序', copyright: '侵权', other: '其他' },
    detail: '补充说明（可选）',
    detailPlaceholder: '例如：仿冒某银行登录页，诱导填写密码',
    contact: '联系方式（可选）',
    contactPlaceholder: '邮箱或其他联系方式，便于我们回复你',
    submit: '提交举报',
    done: '已收到你的举报，谢谢！我们会尽快核实处理。'
  }
}
