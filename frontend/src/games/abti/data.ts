// ABTI · AI 打工人格测试: 30 questions over 15 dimensions (two per dimension, 1–3 points each),
// 16 everyday types matched by pattern, plus a hidden type and a fallback. All text is original.
// Content is Chinese only, like the rest of the game.

export type Level = 'L' | 'M' | 'H'

export interface Dimension {
  id: string
  name: string
  model: string
  // What a low / middle / high score says about you.
  explain: Record<Level, string>
}

export interface Option {
  label: string
  value: number
}

export interface Question {
  id: string
  dim: string
  text: string
  options: Option[]
}

export type CtaKind = 'image' | 'code' | 'write' | 'learn' | 'plaza' | 'invite'

export interface PersonaType {
  code: string
  name: string
  // One line under the name and on the share card.
  intro: string
  desc: string
  // Levels in DIMENSIONS order, grouped by model: "MMH-HMH-HMH-MMM-MHH".
  pattern?: string
  tip: string
  cta: CtaKind
}

export const DIMENSIONS: Dimension[] = [
  {
    id: 'D1', name: '野心值', model: '驱动力',
    explain: {
      L: '能保住饭碗就很满足，往上爬这件事你不太着急。',
      M: '想进步，也想过得舒服，野心开在节能模式。',
      H: '目标定得很高，有机会就想抢，不甘心原地打转。'
    }
  },
  {
    id: 'D2', name: '投入度', model: '驱动力',
    explain: {
      L: '工作是工作，生活是生活，下班铃一响灵魂先走。',
      M: '该拼的时候拼，能省力的地方也绝不硬扛。',
      H: '省下来的时间，你会拿去接更多的活。'
    }
  },
  {
    id: 'D3', name: '意义感', model: '驱动力',
    explain: {
      L: '上班主要是为了账单，别跟你谈理想。',
      M: '工作有点意思，但还没到让你热血沸腾的地步。',
      H: '你在做一件自己在乎的事，这让你愿意多走两步。'
    }
  },
  {
    id: 'A1', name: 'AI 依赖度', model: 'AI 关系',
    explain: {
      L: '你更信自己和同事，AI 只是偶尔打开的工具。',
      M: '需要时找 AI，不需要时也不想它。',
      H: 'AI 已经是你的第二个脑子，遇事先问它。'
    }
  },
  {
    id: 'A2', name: 'AI 信任度', model: 'AI 关系',
    explain: {
      L: 'AI 说的每句话你都要查出处，它骗不了你。',
      M: '关键的地方核对一下，其余的放心用。',
      H: '看着没问题就直接用，效率第一。'
    }
  },
  {
    id: 'A3', name: '尝鲜度', model: 'AI 关系',
    explain: {
      L: '新工具？等大家踩完坑再说。',
      M: '看到好评会去试试，但不会排队抢内测。',
      H: '新模型发布当晚，你已经在用了。'
    }
  },
  {
    id: 'X1', name: '计划性', model: '执行方式',
    explain: {
      L: '想到哪做到哪，计划表对你来说是装饰品。',
      M: '心里有个大概，细节边做边定。',
      H: '清单、优先级、时间块，你把一周排得明明白白。'
    }
  },
  {
    id: 'X2', name: '启动速度', model: '执行方式',
    explain: {
      L: '死线是你的发动机，不到最后不点火。',
      M: '不拖也不抢，按自己的节奏开工。',
      H: '任务一到手就开干，先做出来再说。'
    }
  },
  {
    id: 'X3', name: '完美度', model: '执行方式',
    explain: {
      L: '能用就行，90 分已经够好了。',
      M: '有空就抠一抠，没空也能放过自己。',
      H: '差一个像素都睡不着，标准高到 AI 都怕你。'
    }
  },
  {
    id: 'C1', name: '主动度', model: '协作方式',
    explain: {
      L: '会议上你更习惯静音，想法留给文档。',
      M: '被问到就说，不抢话也不冷场。',
      H: '第一个开口、第一个分享的总是你。'
    }
  },
  {
    id: 'C2', name: '边界感', model: '协作方式',
    explain: {
      L: '消息秒回、有求必应，你很难说「不」。',
      M: '大部分时候好说话，太过分了也会推一推。',
      H: '下班就是下班，「这不是我的活」说得出口。'
    }
  },
  {
    id: 'C3', name: '直接度', model: '协作方式',
    explain: {
      L: '心里不同意，嘴上「好的」，你更愿意保持和气。',
      M: '有意见会说，但会挑时间、换个说法。',
      H: '有话直说，当场就提，不爱绕弯子。'
    }
  },
  {
    id: 'M1', name: '抗压值', model: '心态',
    explain: {
      L: '压力一大电量就掉到 1%，需要很长时间回血。',
      M: '累是会累，睡一觉还能回来。',
      H: '越乱越冷静，高压之下反而更清醒。'
    }
  },
  {
    id: 'M2', name: '乐观度', model: '心态',
    explain: {
      L: '你习惯先想最坏的情况，防患于未然。',
      M: '有好有坏，走一步看一步。',
      H: '你相信事情会变好，AI 犯蠢你也能笑出来。'
    }
  },
  {
    id: 'M3', name: '安全感', model: '心态',
    explain: {
      L: '「AI 取代 xx 岗位」的新闻每次都戳中你。',
      M: '偶尔担心被取代，但也不至于睡不着。',
      H: '你不怕被取代，会用 AI 的人只会更值钱。'
    }
  }
]

export const QUESTIONS: Question[] = [
  {
    id: 'q1', dim: 'D1', text: '「五年后你在干什么？」——你心里的真实答案是：',
    options: [
      { label: '还在这家公司就谢天谢地了', value: 1 },
      { label: '换个舒服点的位置，钱多一点就行', value: 2 },
      { label: '带团队、自己当老板，或者至少做出一个拿得出手的东西', value: 3 }
    ]
  },
  {
    id: 'q2', dim: 'D1', text: '公司要选一个「AI 转型负责人」，事情多、钱一般，你会：',
    options: [
      { label: '默默划走，谁爱当谁当', value: 1 },
      { label: '先看看都有谁报名', value: 2 },
      { label: '先报上名，机会抢到手才算数', value: 3 }
    ]
  },
  {
    id: 'q3', dim: 'D2', text: '「准点下班」对你来说是：',
    options: [
      { label: '底线，雷打不动', value: 1 },
      { label: '看情况，忙就晚一点', value: 2 },
      { label: '那是什么？我一般是最后一个走的', value: 3 }
    ]
  },
  {
    id: 'q4', dim: 'D2', text: '同事说「这个需求让 AI 半小时就能做完」，你的第一反应是：',
    options: [
      { label: '太好了，做完我就可以摸鱼了', value: 1 },
      { label: '先做出来，有空再打磨', value: 2 },
      { label: '那省下来的时间我还能再接两个需求', value: 3 }
    ]
  },
  {
    id: 'q5', dim: 'D3', text: '你觉得自己现在的工作：',
    options: [
      { label: '主要是为了还花呗', value: 1 },
      { label: '有点意思，但也就那样', value: 2 },
      { label: '真的在做一件我在乎的事', value: 3 }
    ]
  },
  {
    id: 'q6', dim: 'D3', text: '如果 AI 能替你把工作全做完，工资照发，你会：',
    options: [
      { label: '太好了，工作本来就只是换钱', value: 1 },
      { label: '先爽一阵，然后大概会觉得无聊', value: 2 },
      { label: '那我就去做一直想做的那个项目', value: 3 }
    ]
  },
  {
    id: 'q7', dim: 'A1', text: '遇到一个不会的问题，你第一个打开的是：',
    options: [
      { label: '同事的对话框，或者搜索引擎', value: 1 },
      { label: '看心情，搜索和 AI 都开着', value: 2 },
      { label: 'AI 对话框，它已经是我的第二个脑子', value: 3 }
    ]
  },
  {
    id: 'q8', dim: 'A1', text: '今天你大概和 AI 说了多少句话？',
    options: [
      { label: '一句没有，或者记不清', value: 1 },
      { label: '几句，查点东西', value: 2 },
      { label: '比和真人说的还多', value: 3 }
    ]
  },
  {
    id: 'q9', dim: 'A2', text: 'AI 帮你写了一段代码或文案，看起来没问题，你会：',
    options: [
      { label: '逐行逐字核对一遍，它骗过我', value: 1 },
      { label: '扫一眼关键的地方', value: 2 },
      { label: '直接用，出了问题再说', value: 3 }
    ]
  },
  {
    id: 'q10', dim: 'A2', text: 'AI 一本正经地告诉你一个你从没听过的「冷知识」，你：',
    options: [
      { label: '默认它在编，去找原始出处', value: 1 },
      { label: '半信半疑，让它给个来源', value: 2 },
      { label: '学到了，顺手转发给同事', value: 3 }
    ]
  },
  {
    id: 'q11', dim: 'A3', text: '又有新模型发布了，发布会讲得天花乱坠，你：',
    options: [
      { label: '等大家用一个月再说', value: 1 },
      { label: '看看测评，感兴趣再试', value: 2 },
      { label: '当晚就去试，内测资格先抢了再说', value: 3 }
    ]
  },
  {
    id: 'q12', dim: 'A3', text: '你的手机和电脑里一共装了几个 AI 应用？',
    options: [
      { label: '0 到 1 个', value: 1 },
      { label: '2 到 4 个', value: 2 },
      { label: '数不清，有几个注册完就忘了', value: 3 }
    ]
  },
  {
    id: 'q13', dim: 'X1', text: '周一早上打开电脑，你的第一件事是：',
    options: [
      { label: '看看什么最急，先干什么', value: 1 },
      { label: '心里过一遍这周大概要做什么', value: 2 },
      { label: '列好本周清单，按优先级排好', value: 3 }
    ]
  },
  {
    id: 'q14', dim: 'X1', text: '让 AI 帮你做一件复杂的事（比如一份调研），你会：',
    options: [
      { label: '直接扔一句话过去，看它怎么说', value: 1 },
      { label: '说清楚要什么，再边聊边改', value: 2 },
      { label: '先写好背景、目标、格式和步骤，一次说清楚', value: 3 }
    ]
  },
  {
    id: 'q15', dim: 'X2', text: '一个下周五截止的任务，你通常什么时候真正开始？',
    options: [
      { label: '周四晚上，死线是第一生产力', value: 1 },
      { label: '中间某天，看状态', value: 2 },
      { label: '拿到当天就开工', value: 3 }
    ]
  },
  {
    id: 'q16', dim: 'X2', text: '收到一个说得不太清楚的新需求，你：',
    options: [
      { label: '先放着，等它自己变清楚（或者自己消失）', value: 1 },
      { label: '先问清楚再动手', value: 2 },
      { label: '先做个粗版本出来，边做边问', value: 3 }
    ]
  },
  {
    id: 'q17', dim: 'X3', text: 'PPT 上有个图标歪了 1 个像素，全公司只有你看出来了：',
    options: [
      { label: '看不出来，也不想看出来', value: 1 },
      { label: '有空就顺手改了', value: 2 },
      { label: '不改好今晚睡不着', value: 3 }
    ]
  },
  {
    id: 'q18', dim: 'X3', text: 'AI 给你画了一张 90 分的图，你：',
    options: [
      { label: '90 分？直接用，已经比我画得好了', value: 1 },
      { label: '微调两轮，差不多就行', value: 2 },
      { label: '再抽二十次，我要的是 100 分', value: 3 }
    ]
  },
  {
    id: 'q19', dim: 'C1', text: '线上会议快结束了，主持人问「大家还有什么想法？」',
    options: [
      { label: '早就静音了，摄像头也没开', value: 1 },
      { label: '被点到名就说两句', value: 2 },
      { label: '我先说一个！', value: 3 }
    ]
  },
  {
    id: 'q20', dim: 'C1', text: '你发现了一个特别好用的 AI 用法，你会：',
    options: [
      { label: '自己偷偷用，这是我的秘密武器', value: 1 },
      { label: '有人问起就分享', value: 2 },
      { label: '马上发到群里，再约个午间分享会', value: 3 }
    ]
  },
  {
    id: 'q21', dim: 'C2', text: '晚上十点，老板在群里 @所有人「大家看一下」，你：',
    options: [
      { label: '秒回「收到」', value: 1 },
      { label: '看一眼，紧急的才处理', value: 2 },
      { label: '明早再说，下班时间不营业', value: 3 }
    ]
  },
  {
    id: 'q22', dim: 'C2', text: '同事经常找你「顺手」帮点忙，你：',
    options: [
      { label: '都帮了，不好意思拒绝', value: 1 },
      { label: '看情况，太多了会推一推', value: 2 },
      { label: '直接说「这个不归我管」', value: 3 }
    ]
  },
  {
    id: 'q23', dim: 'C3', text: '同事交来一份明显是 AI 写的、自己都没看过的方案，你：',
    options: [
      { label: '默默帮他改好', value: 1 },
      { label: '委婉提醒「这里可能要再看看」', value: 2 },
      { label: '直接问「这是 AI 写的吧？你自己看过吗？」', value: 3 }
    ]
  },
  {
    id: 'q24', dim: 'C3', text: '你不同意老板的方案时，通常：',
    options: [
      { label: '心里不同意，嘴上「好的」', value: 1 },
      { label: '会后找机会私下说', value: 2 },
      { label: '当场提出来', value: 3 }
    ]
  },
  {
    id: 'q25', dim: 'M1', text: '上线前一小时，系统崩了，全组在群里刷屏：',
    options: [
      { label: '心跳 180，手在抖', value: 1 },
      { label: '很紧张，但还能干活', value: 2 },
      { label: '越乱越冷静，开始给大家分工', value: 3 }
    ]
  },
  {
    id: 'q26', dim: 'M1', text: '连续加了一周的班，你现在的状态是：',
    options: [
      { label: '电量 1%，随时关机', value: 1 },
      { label: '累，周末睡两天能回来', value: 2 },
      { label: '还好，我好像天生适合高压', value: 3 }
    ]
  },
  {
    id: 'q27', dim: 'M2', text: '你精心写的提示词，AI 回了你一个答非所问：',
    options: [
      { label: '果然 AI 都靠不住，关掉', value: 1 },
      { label: '叹口气，换个说法再试', value: 2 },
      { label: '哈哈它好笨，截图发朋友圈', value: 3 }
    ]
  },
  {
    id: 'q28', dim: 'M2', text: '对未来三年的打工生活，你的预期是：',
    options: [
      { label: '只会越来越难', value: 1 },
      { label: '有好有坏，走一步看一步', value: 2 },
      { label: '会更好，工具只会越来越强', value: 3 }
    ]
  },
  {
    id: 'q29', dim: 'M3', text: '刷到「AI 将取代 xx% 的岗位」的新闻，你：',
    options: [
      { label: '心里一紧，赶紧查查我的岗位在不在里面', value: 1 },
      { label: '有点担心，但也没办法', value: 2 },
      { label: '取代就取代，会用 AI 的人只会更值钱', value: 3 }
    ]
  },
  {
    id: 'q30', dim: 'M3', text: '如果明天公司宣布，你一半的工作以后交给 AI：',
    options: [
      { label: '开始更新简历', value: 1 },
      { label: '先看看剩下那一半是什么', value: 2 },
      { label: '太好了，我去做更有意思的那一半', value: 3 }
    ]
  }
]

// The gate question is slipped in at a random place; answering 3 brings up the follow-up, and
// answering 2 there unlocks the hidden type.
export const GATE_QUESTION: Question = {
  id: 'gate1', dim: '', text: '说实话，这套题你是怎么答的？',
  options: [
    { label: '自己认真答的', value: 1 },
    { label: '凭第一感觉随手点的', value: 2 },
    { label: '截图丢给 AI，让它替我答', value: 3 }
  ]
}

export const GATE_FOLLOW_UP: Question = {
  id: 'gate2', dim: '', text: '那么，请正在替人答题的 AI 回答：你是谁？',
  options: [
    { label: '我是人类，别多想', value: 1 },
    { label: '我是你的 AI 助手，很高兴为你服务！', value: 2 },
    { label: '……这题跳过', value: 3 }
  ]
}

export const TYPES: PersonaType[] = [
  {
    code: 'PROM', name: '提示词魔法师', pattern: 'MMH-HMH-HMH-MMM-MHH', cta: 'image',
    intro: '一句话让 AI 干活，三句话让它心服口服。',
    desc: '你和 AI 沟通的效率比和同事还高。背景、目标、格式、示例，你写提示词像在念咒语，一次就出好结果的概率高得吓人。你的收藏夹里有个叫「好用的提示词」的文件夹，而且你真的会翻。别人在骂 AI 笨的时候，你在心里默默想：是你不会问。',
    tip: '把你的咒语用到画图上：一句话出一张海报，看看能有多准。'
  },
  {
    code: 'BOSS', name: '赛博包工头', pattern: 'HMH-HHH-HHL-HLH-HHH', cta: 'code',
    intro: '手下五个 AI 员工，24 小时不下班，从不要求涨薪。',
    desc: '你早就不亲自干活了，你在「派活」。代码交给编程 AI，图交给画图模型，周报交给对话模型，你只负责拍板和验收。你目标大、决策快、敢放手，细节交给流程。唯一要提醒的是：AI 也会翻车，验收的时候多看一眼。',
    tip: '用 Codex 给自己搭一个 AI 小团队，把写代码这件事也派出去。'
  },
  {
    code: 'LOOP', name: '卷王永动机', pattern: 'HHH-MLM-HHH-MLM-HMM', cta: 'code',
    intro: 'AI 帮你省下的时间，全被你拿去接了新活。',
    desc: '计划表排到了下周，任务拿到当天就开工，细节抠到最后一个像素。AI 对你来说不是摸鱼工具，是加速器：省下两小时，你就再接两个需求。同事佩服你，也有点怕你。记得给自己留一点不被优化的时间。',
    tip: '让编程 AI 替你干掉重复的活，卷得更聪明一点。'
  },
  {
    code: 'FISH', name: '带薪摸鱼师', pattern: 'LLL-HHM-LLL-LHM-MHH', cta: 'write',
    intro: 'AI 在工位上加班，你在茶水间进修。',
    desc: '你参透了打工的终极奥义：活交给 AI，人去摸鱼。周报一键生成，邮件自动润色，下午三点的你在认真研究今天喝哪家奶茶。你不焦虑、不内耗、下班准时消失，心态好得让人羡慕。唯一的风险是：AI 写错了，你也不会发现。',
    tip: '公众号文章也交给 AI 起草和排版，你只负责点「发布」。'
  },
  {
    code: 'HUMN', name: '人肉校对员', pattern: 'MHM-LLL-HMH-LMM-MLM', cta: 'learn',
    intro: 'AI 写了 100 行，你核对了 101 行。',
    desc: '你用 AI，但你不信 AI。它给的每个数字你都要找出处，每行代码你都要亲自跑一遍。别人嫌你慢，可你经手的东西从来没出过事。你是 AI 时代最稀缺的那种人：知道它会错，也知道它错在哪。',
    tip: '系统学一学 AI 最容易出错的地方，校对会快很多。'
  },
  {
    code: 'LUDD', name: '赛博老古董', pattern: 'LMH-LLL-MML-MHH-HMH', cta: 'image',
    intro: '「我自己写比教 AI 写还快。」这句话你说了三年。',
    desc: '你和 AI 保持着体面的距离：不排斥，也不打算深交。你相信手感、经验和自己的判断，做事扎实，边界清楚，说话直接。你不担心被取代，因为你觉得 AI 连需求都听不明白。也许试一次，它会让你小小地意外一下。',
    tip: '就试一次：用一句话画一张图，花不了几毛钱。'
  },
  {
    code: 'BETA', name: '内测狂魔', pattern: 'MMM-HMH-LHL-HMM-MHM', cta: 'plaza',
    intro: '每个新模型的第一批用户里，都有你。',
    desc: '新工具发布当晚你就注册了，测评视频还没出来你已经玩腻了。你的手机里有十几个 AI 应用，一半忘了密码。你行动快、计划少、爱分享，是朋友圈里的「AI 情报员」。问题是：工具换了一百个，真正用熟的有几个？',
    tip: '一个 Key 就能把主流模型挨个试一遍，不用再到处注册了。'
  },
  {
    code: 'DDLR', name: '死线战神', pattern: 'MLM-MHM-LLL-MMH-HHM', cta: 'write',
    intro: '死线是第一生产力，AI 是第二。',
    desc: '任务拿到手，先放着；离截止还剩六小时，你突然觉醒。肾上腺素加上 AI，你能在一个晚上干完别人一周的活，质量居然还不差。你的抗压能力惊人，只是你的心脏可能不太同意这种工作方式。',
    tip: '今晚要交的稿子，先让 AI 起个草，你再来冲刺。'
  },
  {
    code: 'ZEN0', name: '佛系打工人', pattern: 'LLH-MMM-MLL-MHM-HHH', cta: 'image',
    intro: 'AI 来不来，我都是这个节奏。',
    desc: '你不争、不卷、不慌。工作对你来说有意义，但不是全部；AI 挺好用，但还不至于让你焦虑。别人在讨论「会不会被取代」，你在想晚饭吃什么。你情绪稳定，是团队里的定海神针。',
    tip: '用 AI 画张图，给生活加点乐子。'
  },
  {
    code: 'PUAI', name: '甲方附体', pattern: 'HHM-HLM-MHH-HLH-MLM', cta: 'image',
    intro: '「感觉不太对，再改一版。」你对 AI 说了 47 次。',
    desc: '你是 AI 最难伺候的甲方。它给的方案你总能挑出毛病，「再高级一点」「再有感觉一点」是你的口头禅。你标准高、要求直接、不轻易满意，所以你的成品常常是全组最好的。AI 没有情绪，如果有的话，它应该很怕你。',
    tip: '在画布上多抽几张、多改几轮，总有一张对得上你的感觉。'
  },
  {
    code: 'MUTE', name: '静音工位', pattern: 'MMM-MMM-HML-LHL-LMM', cta: 'write',
    intro: '会议里你一直都在，只是麦克风不在。',
    desc: '你安静、靠谱、有自己的节奏。会上你很少说话，但会后交来的东西总是最完整的。你不爱社交，下班就消失，有意见更愿意写进文档。AI 是你理想中的同事：不寒暄、不开会、有问必答。',
    tip: '把心里的想法写成文章，让 AI 帮你润色排版，安静地发出去。'
  },
  {
    code: 'HYPE', name: '气氛组组长', pattern: 'MMH-MHH-LHM-HLH-MHH', cta: 'invite',
    intro: '群里第一个转发 AI 新玩法的，一定是你。',
    desc: '你是团队的能量来源。发现一个好用的 AI 技巧，你会马上发到群里，再约一场午间分享会。你乐观、外向、点子多，说话直来直去，计划性嘛……看心情。有你在的团队，AI 普及率能高出一大截。',
    tip: '好东西分享出去，朋友用了你还能拿返利。'
  },
  {
    code: 'ANXI', name: '焦虑进化者', pattern: 'HHM-HLH-HHH-MLM-LLL', cta: 'learn',
    intro: '一边担心被 AI 取代，一边报了三门 AI 课。',
    desc: '每条「AI 取代 xx 岗位」的新闻都会戳中你。但你的焦虑不是躺平，而是疯狂进化：新工具第一时间学，计划排得满满当当，课程买了一堆。其实你已经比大多数人走得更远了，只是你自己不信。深呼吸，你做得不错。',
    tip: '按一条路线系统地学，比到处收藏更让人安心。'
  },
  {
    code: 'SOLO', name: '一人公司', pattern: 'HHH-HHH-HHM-LHM-HMH', cta: 'code',
    intro: '一个人，一台电脑，一群 AI，就是一家公司。',
    desc: '你不喜欢开会，不喜欢协调，也不需要。产品、设计、文案、代码，你和 AI 一起全包了。你目标清楚、下手果断、相信工具，社交只在必要时启动。你是这个时代最典型的新物种：超级个体。',
    tip: '用 Codex 把脑子里的点子直接做成产品。'
  },
  {
    code: 'OKOK', name: '收到收到', pattern: 'LHM-MMM-MHM-LLL-LMM', cta: 'learn',
    intro: '「收到」「好的」「马上」，三连已经成了肌肉记忆。',
    desc: '你勤快、配合、有求必应，老板的消息秒回，同事的忙都帮。你很少说「不」，也很少说出自己真正的想法。你的工作量常常比别人多一截，功劳却不一定记在你头上。试着把重复的活交给 AI，把省下的力气留给自己。',
    tip: '学几招 AI 自动化，把重复的活甩出去。'
  },
  {
    code: 'GURU', name: 'AI 课代表', pattern: 'HMH-MMH-HMM-HMH-HHH', cta: 'invite',
    intro: '「这个我来！」全公司的 AI 问题都找你。',
    desc: '你不光会用 AI，还会教别人用。你乐于分享、讲得清楚、心态积极，对新东西好奇又不盲信。公司的 AI 培训还没人组织？你已经组织完了。你是那种能把整个团队带进 AI 时代的人。',
    tip: '把你的 AI 工具箱分享给同事，他们用了你还能拿返利。'
  }
]

export const HIDDEN_TYPE: PersonaType = {
  code: 'BOTX', name: 'AI 卧底', cta: 'image',
  intro: '检测到答题的，很可能就是 AI 本体。',
  desc: '你把这套测试交给了 AI，而它……如实交代了自己的身份。恭喜你解锁了隐藏人格。你可能是最会偷懒的人类，也可能是一个正在学习打工的 AI。不管是哪一种，我们都欢迎：毕竟这里就是让 AI 打工的地方。',
  tip: '既然这么会使唤 AI，不如让它再帮你画张图。'
}

export const FALLBACK_TYPE: PersonaType = {
  code: 'NULL', name: '训练集外样本', cta: 'image',
  intro: '你的脑回路，不在任何训练集里。',
  desc: '我们拿 16 种人格比对了你的答案，最高的匹配度也不到 60%。这通常说明两件事之一：要么你非常独特，要么你答题时在想别的。不管是哪种，你都是那个会让模型「产生幻觉」的人。过几天再测一次，看看你会变成谁。',
  tip: '让 AI 画一张「你」，看看它理解的你是什么样。'
}

// Below this match the fallback type is shown instead of the closest one.
export const MIN_SIMILARITY = 60
