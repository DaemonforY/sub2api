export default {
  requestLogs: {
    title: '请求日志',
    description: '网关收到的每一次调用都会记录，包括失败、鉴权被拒和路径错误的请求，含完整 URL 与完整 API Key。默认保留 30 天。',
    empty: '暂无请求记录',
    loadFailed: '加载请求日志失败',
    fullKeyNotice: '本页显示完整 API Key，请勿截图外传。',
    filters: {
      all: '全部',
      q: '关键字',
      qPlaceholder: 'URL / API Key / 模型 / 邮箱 / IP',
      result: '结果',
      success: '成功',
      failure: '失败',
      statusCode: '状态码',
      method: '方法',
      timeRange: '时间范围',
      last1h: '最近 1 小时',
      last24h: '最近 24 小时',
      last7d: '最近 7 天',
      last30d: '最近 30 天'
    },
    columns: {
      time: '时间',
      user: '用户',
      apiKey: 'API Key',
      request: '请求',
      result: '结果',
      model: '模型',
      duration: '耗时',
      clientIp: '来源 IP',
      detail: '详情'
    },
    detail: {
      title: '请求详情',
      url: '完整 URL',
      apiKey: '完整 API Key',
      keyName: '密钥名称',
      user: '用户',
      status: '状态码',
      errorCode: '错误原因',
      model: '模型',
      duration: '耗时',
      clientIp: '来源 IP',
      userAgent: 'User-Agent',
      requestId: '请求 ID',
      groupId: '分组 ID',
      accountId: '上游账号 ID'
    },
    unknownKey: '未匹配到密钥',
    noKey: '未携带密钥',
    copy: '复制',
    copied: '已复制',
    total: '共 {count} 条'
  }
}
