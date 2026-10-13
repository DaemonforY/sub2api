export default {
  desk: {
    intro: '用你自己的口吻写：先备好有出处的材料，再挑角度、比三稿、在编辑器里精修，最后一键改成各平台版本。AI 给你材料和选项，拿主意的是你。',
    tabs: {
      desk: '写作台',
      review: '待发布',
      topics: '选题池',
      persona: '作者档案',
      stats: '数据',
      settings: '设置'
    },
    summary: {
      desk: '写作中',
      pending: '待发布',
      topics: '待写选题',
      published: '已发布',
      goSettings: '去设置'
    },
    stage: {
      materials: '材料',
      angles: '角度',
      drafts: '三稿',
      edit: '精修',
      final: '定稿'
    },
    job: {
      research: '查资料、整理事实卡',
      angles: '想角度',
      reference: '拆解参考文章',
      drafts: '写三份初稿并请读者评审',
      merge: '把亮点融合进底稿',
      check: '检查全文',
      learn: '从你的改动里总结写作规律',
      platforms: '改写各平台版本'
    },
    common: {
      back: '上一步',
      cancel: '取消',
      chars: '字',
      close: '关闭',
      delete: '删除',
      loading: '加载中…',
      ok: '确定'
    },
    review: {
      empty: '还没有待发布的稿子。写作台里定稿后点「生成各平台版本」，就会出现在这里。'
    },
    list: {
      newTitle: '写一篇新稿子',
      newHint: '先说你想写什么、手里有什么。材料越具体，写出来越像你。',
      new: '新建',
      noPersona: '还没有作者档案，写出来的不会像你。',
      goPersona: '先去建档案',
      title: '写什么',
      titlePlaceholder: '例如：我用 Codex 改了一周公司的老代码',
      idea: '你的想法（选填）',
      ideaPlaceholder: '你最想说的判断，或者让你想写这篇的那件事',
      notes: '你手里的材料（强烈建议）',
      notesPlaceholder: '你自己的经历、看到的数字、踩过的坑、截图里的内容、相关链接……AI 编不出来的东西写在这里',
      notesHint: 'AI 只会用你给的材料和它查到、带出处的公开事实，不会替你编经历。',
      start: '开始：备材料',
      empty: '写作台还是空的',
      failed: '上一步失败了'
    },
    persona: {
      intro: '作者档案决定了写出来像不像你。先加 2–5 篇你自己写的公众号文章，让 AI 从里面提炼你的身份、经历、立场和说话习惯；你改到满意再保存。以后每一步都会用你的口吻写。',
      updated: '{time} 更新',
      samples: '我的范文',
      samplesHint: '贴公众号文章链接，或者直接粘贴正文。挑最像你、你最满意的几篇。',
      distill: '从范文提炼档案',
      distilling: '正在读你的文章…（约 1 分钟）',
      urlPlaceholder: 'https://mp.weixin.qq.com/s/…',
      addUrl: '读取文章',
      paste: '没有链接？直接粘贴正文',
      titlePlaceholder: '文章标题',
      textPlaceholder: '正文（至少 200 字）',
      addText: '加为范文',
      proposed: '这是 AI 从范文里提炼的档案，改到满意再点保存。经历只写真实发生过的事。',
      name: '自称 / 署名',
      identity: '我是谁',
      identityPlaceholder: '身份、在做什么、和读者是什么关系，用第一人称',
      experiences: '我真实做过、经历过的事',
      experiencesPlaceholder: '一行一件，例如：网站上线第一周被撞库 3.9 万次',
      experiencesHint: '写稿时可以直接当一手材料用，所以只写真实发生过的事。',
      stances: '我的立场和判断',
      phrases: '我常用的说法',
      likes: '我喜欢的',
      dislikes: '我讨厌的',
      voice: '我说话、写东西的习惯',
      voicePlaceholder: '人称和自称、句子长短、标点习惯、开头结尾怎么写、用不用小标题、幽默从哪来……',
      taboos: '我从不这样写',
      taboosPlaceholder: '一行一条，例如：家人们、赋能、不是……而是……',
      oneLine: '列表类的项目一行写一条。',
      saved: '作者档案已保存'
    },
    materials: {
      title: '材料关',
      hint: '文章靠材料立住。AI 查到的公开事实都带出处；你的经历和回答算一手材料。不可信的点「不用」，拿不准的点「存疑」。',
      usable: '件可用材料',
      notEnough: '少于 5 件时写不出像样的长文。回答下面的问题、补几条你自己的经历，或者让 AI 再查一轮。',
      questions: 'AI 想问你（回答会变成一手材料）',
      answerPlaceholder: '用你平时说话的方式回答就行',
      facts: '事实卡',
      researching: '正在查资料…',
      empty: '还没有事实卡',
      addPlaceholder: '补一条你自己知道的事实或经历',
      add: '添加',
      hintPlaceholder: '想让 AI 重点查什么（选填）',
      again: '再查一轮',
      save: '保存',
      saved: '已保存',
      next: '下一步：想角度'
    },
    fact: {
      kind: {
        user: '你给的',
        persona: '你的经历',
        answer: '你的回答',
        web: '公开资料'
      },
      status: {
        ok: '采用',
        doubt: '存疑',
        rejected: '不用'
      }
    },
    angles: {
      title: '选一个角度',
      hint: '同一个选题，角度不同就是不同的文章。H 有趣、K 有料、R 共鸣，三项都高才是好角度。选一个，可以直接改。',
      make: '想角度',
      again: '换一批',
      empty: '还没有角度',
      hkr: {
        H: '有趣',
        K: '有料',
        R: '共鸣'
      },
      reader: '写给：{reader}',
      edit: '改成你想要的样子',
      fTitle: '标题方向',
      fAngle: '角度',
      fHook: '开头',
      fNote: '补充要求（选填）',
      fNotePlaceholder: '例如：别写成教程；多讲我踩的坑',
      write: '用这个角度写三稿'
    },
    ref: {
      title: '参考一篇爆文（选填）',
      hint: '贴一篇你觉得写得好的同类文章，AI 只学它的结构和节奏（开头手法、每段作用、情绪怎么推），不搬内容。',
      urlPlaceholder: '公众号文章链接',
      analyze: '拆解结构',
      paste: '或者粘贴正文',
      from: '参考：{source}',
      opening: '开头手法：',
      emotion: '情绪推进：',
      escalation: '加码顺序：',
      callbacks: '前后呼应：'
    },
    drafts: {
      title: '三份初稿',
      hint: '同一个角度，三种结构。左边的色条是模拟读者的反应：黄色开始扫读，红色会划走。选一份当底稿，再勾选其他稿子里值得融合的亮点。',
      make: '写三稿',
      again: '重写三稿',
      empty: '还没有初稿',
      judge: '主编的看法：',
      skim: '开始扫读',
      leave: '会划走',
      label: '{label}稿',
      rank: '第 {n}',
      score: {
        hook: '开头',
        flow: '顺畅',
        voice: '像真人',
        substance: '干货'
      },
      useBase: '用这份当底稿',
      isBase: '✓ 底稿',
      highlights: '融合其他稿子的亮点',
      noHighlights: '主编没有挑出别的亮点，可以直接进入精修。',
      merge: '融合 {n} 处亮点，进入精修',
      edit: '用底稿进入精修'
    },
    editor: {
      placeholder: '开始写…',
      h2: '小标题',
      bold: '加粗',
      quote: '引用',
      list: '列表',
      count: '{chars} 字 · {paras} 段',
      saving: '保存中…',
      saved: '{time} 已保存',
      saveFailed: '保存失败',
      snapshot: '存个版本',
      snapshotPlaceholder: '给这个版本起个名',
      backDrafts: '回到三稿',
      finalize: '定稿'
    },
    assist: {
      voice: '更像我',
      tighten: '压缩',
      expand: '展开',
      sharper: '判断更鲜明',
      lighter: '更轻松',
      plain: '大白话',
      analogy: '找类比',
      evidence: '找证据',
      custom: '自己说',
      customPlaceholder: '想怎么改，例如：加一个我自己的例子的位置',
      comment: '批注',
      thinking: 'AI 正在改…',
      accept: '采用',
      retry: '再来一次',
      reject: '不要',
      insertAfter: '插到选中文字后面',
      keepFact: '加进事实卡',
      factKept: '已加进事实卡',
      nothing: '这次没有可用的结果，换个说法再试',
      lost: '原文已经改动，找不到要替换的位置了'
    },
    side: {
      check: '检查',
      facts: '事实卡',
      versions: '版本',
      comments: '批注',
      factsHint: '正文里的事实只能来自这些卡片；需要新事实时，选中文字点「找证据」。'
    },
    check: {
      run: '检查全文',
      again: '再检查一遍',
      hint: '先按固定规则扫（翻案腔、套话、破折号、没补的经历），再请主编逐段审读：像不像你、读者会不会走神、有没有编事实。',
      stats: '单句段 {single}% · 小标题 {headings} 个 · 待补经历 {placeholders} 处',
      locate: '定位',
      apply: '采用修改',
      ignore: '忽略',
      notFound: '正文里找不到这段文字了，可能已经改过',
      level: {
        must: '必须改',
        should: '建议改'
      },
      layer: {
        rule: '规则',
        voice: '不像你',
        flow: '读不下去',
        content: '撑不住',
        fact: '事实'
      },
      score: {
        voice: '像你',
        flow: '顺畅',
        substance: '干货',
        facts: '可靠'
      }
    },
    versions: {
      empty: '还没有版本',
      restore: '恢复这个版本',
      manual: '手动保存',
      saved: '已存为一个版本',
      kind: {
        draft: '选定底稿',
        merge: '融合亮点',
        manual: '手动保存',
        ai: 'AI 修改',
        restore: '恢复前',
        final: '定稿'
      }
    },
    comments: {
      hint: '选中文字点「批注」，把想法留在原文旁边。',
      placeholder: '写下你的想法',
      resolve: '解决',
      reopen: '重新打开'
    },
    final: {
      title: '定稿了',
      hint: '公众号版本就是这篇正文；知乎、CSDN、头条、小红书会用你的口吻各自重写，之后在「待发布」里审核。',
      platforms: '生成各平台版本',
      backEdit: '回去再改改',
      learn: '从你的改动里学到的',
      learnHint: 'AI 对比了它交给你的稿子和你的定稿。觉得对的加进写作规则，以后每篇都会照做。',
      noLearn: '这篇没有改动，或者还在总结。',
      addRule: '加进写作规则',
      dismiss: '不对',
      added: '已加进写作规则',
      dismissed: '已忽略'
    },
    view: {
      back: '返回写作台',
      tokens: '已用 {n} tokens',
      elapsed: '{s} 秒',
      stop: '停止',
      failed: '「{step}」没成功：{msg}',
      openDesk: '回到写作台'
    }
  }
}
