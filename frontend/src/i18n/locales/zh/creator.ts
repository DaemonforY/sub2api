export default {
  creator: {
    title: '创作者中心',
    description: '发布你自己的付费课程，学员付款后自动拿到你的网盘资料，收入可以申请提现。',
    closed: '暂未开放课程创作者申请，开放后会在这里通知。',
    intro: {
      title: '在这里卖你的课程',
      step1: '提交申请，平台审核通过后成为创作者。',
      step2: '编写课程介绍、大纲和价格，填好网盘链接后提交审核；审核通过就会上架。之后修改介绍或价格也要重新审核，网盘链接随时可以换。',
      step3: '学员付款后自动开通课程、在「我的课程」里拿到你的网盘链接。每卖出一份，平台按实付金额收取 {percent}% 手续费，其余归你。',
      step4: '每笔收入过了 {days} 天结算期后可以提现（最低 ¥{min}），提现到支付宝或微信，平台人工打款。'
    },
    apply: {
      title: '申请成为创作者',
      displayName: '讲师名称',
      displayNameHint: '显示在课程页上，例如「王老师」',
      contact: '联系方式',
      bio: '个人简介',
      plan: '课程计划',
      planPlaceholder: '准备讲什么、适合谁、大概多少节课、有没有已发布的作品或试看链接',
      submit: '提交申请',
      sent: '申请已提交，审核结果会显示在这里'
    },
    rejected: '上次申请没有通过：',
    pending: {
      title: '申请审核中',
      body: '「{name}」的申请已提交，平台审核通过后就可以发布课程。'
    },
    suspended: '你的创作者资格已暂停，课程已下架，已有收入仍可提现。',
    balance: {
      available: '可提现',
      frozen: '结算中',
      paid: '已提现（含处理中）',
      net: '累计收入'
    },
    rules: '平台手续费 {percent}%（按学员实付金额计算，不含支付手续费），每笔收入付款 {days} 天后可提现；学员退款会扣回对应收入。',
    tabs: { courses: '我的课程', earnings: '收入与提现' },
    coursesHint: '最多 {max} 门课程。课程要提交审核、通过后才会上架。',
    newCourse: '新建课程',
    noCourses: '还没有课程，点「新建课程」开始。',
    studentCount: '{count} 人已购',
    status: { draft: '未上架', published: '在售', archived: '已下架' },
    review: {
      draft: '有修改未提交',
      pending: '审核中',
      approved: '已通过审核',
      rejected: '未通过审核'
    },
    reviewHint: {
      draft: '修改已保存为草稿，提交审核并通过后才会对学员显示。',
      pending: '平台正在审核，通常 1 个工作日内处理。审核期间如果再修改，需要重新提交。',
      approved: '学员看到的就是当前内容。',
      rejected: '按审核意见修改后可以重新提交。'
    },
    rejectReason: '审核意见：',
    liveVersion: '线上版本：「{title}」¥{price}',
    backToCourses: '返回课程列表',
    viewPage: '查看课程页',
    unlist: '下架',
    relist: '重新上架',
    slugFixed: '课程地址创建后不能修改',
    shareHint: '平台收 {percent}% 手续费，每卖出一份你约得 ¥{share}（使用教育优惠或限时价时按实付计算）',
    saveDraft: '保存修改',
    createCourse: '创建课程',
    submit: '提交审核',
    draftSaved: '已保存',
    submitted: '已提交审核',
    deleteConfirm: '删除「{title}」？删除后不能恢复。',
    blocker: {
      pending: '正在审核中',
      noChanges: '没有需要审核的修改',
      noDelivery: '先在下方填写网盘链接'
    },
    deliveryTitle: '网盘发货',
    deliveryHint: '学员付款后在「我的课程」里看到网盘链接、提取码和解压密码（加密保存）。网盘信息不需要审核，保存后立即生效。',
    students: '已购学员（{count}）',
    buyer: '学员',
    boughtAt: '购买时间',
    viewed: '查看次数',
    refunded: '已退款',
    withdraw: {
      title: '申请提现',
      hint: '最低提现 ¥{min}，每笔收入付款 {days} 天后可提现；同一时间只能有一笔提现在处理中。',
      pending: '你有一笔提现正在处理中，处理完成后才能再申请。',
      belowMin: '可提现金额满 ¥{min} 后可以申请提现。',
      amount: '提现金额（元）',
      max: '最多可提 ¥{amount}',
      method: '收款方式',
      alipay: '支付宝',
      wechat: '微信',
      account: '收款账号',
      realName: '真实姓名',
      note: '备注（选填）',
      submit: '申请提现',
      requested: '提现申请已提交，平台打款后会在这里显示',
      cancel: '撤销',
      adminNote: '平台备注：',
      status: { pending: '处理中', paid: '已打款', rejected: '已驳回', cancelled: '已撤销' }
    },
    sales: {
      title: '销售记录',
      hint: '实付不含支付手续费；退款会按退款比例扣回。',
      empty: '还没有销售记录',
      time: '时间',
      course: '课程',
      gross: '实付',
      fee: '手续费',
      net: '你的收入',
      status: '状态',
      availableAt: '{time} 可提现',
      statuses: {
        frozen: '结算中',
        available: '已结算',
        refunding: '退款处理中',
        refunded: '已退款',
        partially_refunded: '部分退款'
      },
      prev: '上一页',
      next: '下一页'
    }
  }
}
