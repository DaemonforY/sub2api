// Office task templates: the first thing a non-developer sees in an empty chat.
//
// Each one has two prompts:
//   prompt       — runs on the user's own files in the chosen folder (put in the composer to edit);
//   samplePrompt — first makes realistic fake files in 「示例」, then does the same task there, so
//                  someone with nothing at hand can see what it does in a minute.
// Results always go to 「输出」 (see AGENTS.md written by src-tauri/src/bridge.rs).

export interface TaskTemplate {
  id: string
  icon: string
  title: string
  desc: string
  /** What to put in the folder before using it on real files. */
  prepare: string
  prompt: string
  samplePrompt: string
  /** Needs the office kit (Python + libraries). */
  office: boolean
}

const SAMPLE_NOTE = '这些是演示用的假数据，生成后告诉我放在哪里。'

export const TEMPLATES: TaskTemplate[] = [
  {
    id: 'invoices',
    icon: '🧾',
    title: '发票单据整理成 Excel',
    desc: '把一堆发票、报销单整理成一张带月度汇总的表',
    prepare: '把发票或报销单（PDF、图片、Excel 都行）放进这个文件夹。',
    prompt:
      '这个文件夹里有一些发票或报销单据（可能是 PDF、图片或 Excel）。请逐个读取，把每张的日期、开票方、项目、金额、税额、发票号整理成一张 Excel 表「输出/发票汇总.xlsx」：「明细」工作表按日期排序，「月度汇总」工作表按月合计金额和税额。看不清或识别不确定的放到「待核对」工作表并注明原因。最后告诉我总金额和有几张需要核对。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成 8 份模拟报销单 Excel（每份一张单据：日期在最近三个月、开票方是常见公司名、金额几十到几千元、税额按 6% 或 13%），${SAMPLE_NOTE}然后把「示例」里的单据整理成「输出/发票汇总.xlsx」：「明细」按日期排序，「月度汇总」按月合计金额和税额。`,
    office: true
  },
  {
    id: 'rename',
    icon: '🗂️',
    title: '批量整理和重命名文件',
    desc: '照片按拍摄日期、文件按类型整理好，原文件不动',
    prepare: '把要整理的照片或文件放进这个文件夹。',
    prompt:
      '帮我整理这个文件夹里的文件：照片按拍摄日期（读取照片信息，读不到就用修改时间）重命名为「年-月-日_序号」，其他文件按类型（文档、表格、图片、压缩包、其他）分类。不要改动原文件，把整理后的副本放进「输出/整理后」，再生成一张「输出/整理清单.xlsx」列出原名、新名和位置。动手前先告诉我大概会怎么分。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成 12 个乱命名的文件（几张带不同日期的小图片、几个 Word 和 Excel、几个文本文件），${SAMPLE_NOTE}然后把「示例」里的文件按规则整理：图片按日期重命名为「年-月-日_序号」，其他文件按类型分类，副本放进「输出/整理后」，并生成「输出/整理清单.xlsx」。`,
    office: true
  },
  {
    id: 'merge-excel',
    icon: '📊',
    title: '多个 Excel 合并去重',
    desc: '把格式差不多的多张表合成一张，去掉重复行',
    prepare: '把要合并的 Excel 或 CSV 文件放进这个文件夹。',
    prompt:
      '把这个文件夹里所有 Excel 和 CSV 文件合并成一张表「输出/合并结果.xlsx」。列名不完全一样的先对齐（意思相同的列合并，拿不准的列保留并告诉我），加一列「来源文件」，去掉完全重复的行，再按第一列排序。最后告诉我合并了几个文件、多少行、去掉了多少重复行。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成 4 张客户名单 Excel（列名略有不同，比如「手机」和「联系电话」；其中有十几行重复客户），${SAMPLE_NOTE}然后把「示例」里的表合并去重成「输出/合并结果.xlsx」，加一列「来源文件」，并告诉我去掉了多少重复行。`,
    office: true
  },
  {
    id: 'word-format',
    icon: '📝',
    title: 'Word 报告统一格式',
    desc: '统一字体、标题层级、行距和页码',
    prepare: '把要调整的 Word 文档（.docx）放进这个文件夹。',
    prompt:
      '把这个文件夹里的 Word 文档统一成公文常用格式：标题用黑体二号居中，一级标题黑体三号，二级标题楷体三号，正文仿宋小四、1.5 倍行距、首行缩进两字，页边距上下 2.54 厘米、左右 3.17 厘米，加页码。不要改动原文件，改好的另存到「输出」里，文件名加「_排版」。改不了的地方告诉我。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成一份格式混乱的 Word 工作总结（三四个标题、十来段正文，字体字号杂乱），${SAMPLE_NOTE}然后把它统一排版：标题黑体二号居中，一级标题黑体三号，正文仿宋小四、1.5 倍行距、首行缩进两字，另存为「输出/工作总结_排版.docx」。`,
    office: true
  },
  {
    id: 'meeting',
    icon: '✅',
    title: '会议纪要整理成待办',
    desc: '从纪要里提取待办事项、负责人和截止时间',
    prepare: '把会议纪要或聊天记录（Word、文本都行）放进这个文件夹。',
    prompt:
      '读一下这个文件夹里的会议纪要（Word 或文本），提取所有待办事项：事项、负责人、截止时间、优先级（按语气判断）、原文出处。生成「输出/待办清单.xlsx」，按截止时间排序，没写负责人或时间的标成「待确认」。再写一段三五句话的会议摘要放在回复里。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成一份一千字左右的项目周会纪要 Word（有几个人发言，夹带七八个待办，有的写了负责人和时间、有的没写），${SAMPLE_NOTE}然后从里面提取待办，生成「输出/待办清单.xlsx」，并在回复里写三五句会议摘要。`,
    office: true
  },
  {
    id: 'pdf-table',
    icon: '📄',
    title: 'PDF 里的表格转 Excel',
    desc: '把 PDF 报表、对账单里的表格提取出来',
    prepare: '把 PDF 文件放进这个文件夹（扫描件效果会差一些）。',
    prompt:
      '把这个文件夹里每个 PDF 中的表格提取出来，转成 Excel：每个 PDF 一个文件放在「输出」里，每张表一个工作表，保留表头，数字列转成数字格式。跨页的表格尽量拼成一张。如果某个 PDF 是扫描图片、提取不出文字，告诉我是哪几个。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 的 fpdf2 生成一份两页的模拟银行对账单 PDF（英文表头和数字即可，第一页一张交易明细表、第二页一张月度汇总表），${SAMPLE_NOTE}然后把「示例」里这个 PDF 的表格提取成「输出/对账单表格.xlsx」，每张表一个工作表，数字列转成数字格式。`,
    office: true
  },
  {
    id: 'ppt-outline',
    icon: '📽️',
    title: '根据资料做 PPT',
    desc: '读资料，生成一份能直接改的 PPT 初稿',
    prepare: '把要用的资料（Word、PDF、文本、Excel）放进这个文件夹。',
    prompt:
      '读一下这个文件夹里的资料，帮我做一份 10 页左右的汇报 PPT 初稿「输出/汇报初稿.pptx」：封面、目录、几页要点（每页标题加 3–5 条要点，数据用简单表格）、总结和下一步。版式简洁统一，中文字体用微软雅黑。另外把每页的讲稿要点写在备注里。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成一份关于「门店第三季度经营情况」的 Word 资料（含几段文字和一张各门店销售额表格），${SAMPLE_NOTE}然后根据它做一份 8 页左右的汇报 PPT「输出/汇报初稿.pptx」，每页标题加要点，讲稿写在备注里。`,
    office: true
  },
  {
    id: 'report',
    icon: '📚',
    title: '一堆资料汇总成报告',
    desc: '把多份文档的要点汇总成一份 Word 报告',
    prepare: '把要汇总的资料（Word、PDF、文本、Excel）放进这个文件夹。',
    prompt:
      '读完这个文件夹里的所有资料，写一份汇总报告「输出/汇总报告.docx」：开头一段总结，然后按主题分节整理要点，每条要点后面用括号注明来自哪个文件；资料之间有矛盾的地方单独列一节。用中文，正式但好读。',
    samplePrompt: `先在当前文件夹下建一个「示例」子文件夹，用 Python 生成 3 份关于「公司要不要引入 AI 办公工具」的 Word 材料（一份调研、一份成本测算、一份员工意见，观点有一处互相矛盾），${SAMPLE_NOTE}然后把它们汇总成「输出/汇总报告.docx」，要点注明出处，矛盾之处单独列出。`,
    office: true
  }
]
