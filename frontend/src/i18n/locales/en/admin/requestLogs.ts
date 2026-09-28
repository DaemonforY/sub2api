export default {
  requestLogs: {
    title: 'Request logs',
    description: 'Every call that reaches the gateway is recorded, including failed, rejected and wrong-path requests, with the full URL and full API key. Kept for 30 days by default.',
    empty: 'No requests yet',
    loadFailed: 'Failed to load request logs',
    fullKeyNotice: 'This page shows full API keys. Do not share screenshots.',
    filters: {
      all: 'All',
      q: 'Keyword',
      qPlaceholder: 'URL / API key / model / email / IP',
      result: 'Result',
      success: 'Success',
      failure: 'Failure',
      statusCode: 'Status code',
      method: 'Method',
      timeRange: 'Time range',
      last1h: 'Last 1 hour',
      last24h: 'Last 24 hours',
      last7d: 'Last 7 days',
      last30d: 'Last 30 days'
    },
    columns: {
      time: 'Time',
      user: 'User',
      apiKey: 'API key',
      request: 'Request',
      result: 'Result',
      model: 'Model',
      duration: 'Duration',
      clientIp: 'Client IP',
      detail: 'Detail'
    },
    detail: {
      title: 'Request detail',
      url: 'Full URL',
      apiKey: 'Full API key',
      keyName: 'Key name',
      user: 'User',
      status: 'Status',
      errorCode: 'Error reason',
      model: 'Model',
      duration: 'Duration',
      clientIp: 'Client IP',
      userAgent: 'User-Agent',
      requestId: 'Request ID',
      groupId: 'Group ID',
      accountId: 'Upstream account ID'
    },
    unknownKey: 'Key not found',
    noKey: 'No key sent',
    copy: 'Copy',
    copied: 'Copied',
    total: '{count} requests'
  }
}
