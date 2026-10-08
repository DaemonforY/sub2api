// Tool landing pages (/tools/:slug). Each one maps to a gallery category and a create-page preset.

export const TOOLS = [
  {
    slug: 'html-video', category: 'film', mode: 'film', name: 'HTML 视频生成器', icon: 'Clapperboard',
    headline: '一句话做出带配音和字幕的讲解视频',
    summary: '输入主题或完整口播稿，AI 先确认讲解方向，再写脚本、拆分镜、配音、做动画。每个分镜的画面和旁白逐字对齐，可以继续对话修改。',
    points: ['自动写脚本和分镜，也可以贴自己的口播稿', '12 种中文音色，字幕按发音逐字对齐', '18 种视觉风格：科技博主、手绘白板、黑板粉笔…', '每个分镜都能单独改，不满意可以回退版本'],
    prompts: ['用 60 秒讲清楚区块链为什么不能篡改', '给初中生讲牛顿第一定律，配一个生活中的例子', '介绍我们的 AI 编程助手：三个核心功能和一个使用场景']
  },
  {
    slug: 'science', category: 'science', mode: 'film', name: '理工科普动画', icon: 'Atom',
    headline: '把公式和原理变成看得见的动画',
    summary: '物理、数学、化学、生物的知识点，用分步动画讲清楚：标注、受力分析、函数图像、分子结构，适合课堂和科普账号。',
    points: ['先给结论，再一步步推导', '图形和标注严格对应旁白', '适合老师备课、学生复习、科普短视频'],
    prompts: ['光的折射：光线从空气射入水中发生偏折，标出入射角和折射角', '用单位圆讲清楚正弦函数是怎么来的', '细胞有丝分裂的四个时期']
  },
  {
    slug: 'data-visualization', category: 'data', mode: 'motion', name: '数据可视化动画', icon: 'ChartColumn',
    headline: '数据一贴，图表自己长出来',
    summary: '柱状图、折线图、饼图、环形进度、数字滚动……把数据写进描述，AI 生成会动的图表，可以下载或嵌入网页。',
    points: ['每个数据点都会画出来并标值', '重点数字放大强调', '支持横屏、竖屏和方形'],
    prompts: ['1–6 月用户数 120、180、260、410、530、800，折线逐点画出', '流量来源饼图：搜索 46%、社交 32%、直接 22%', '三款手机的续航对比柱状图']
  },
  {
    slug: 'logo-animation', category: 'logo', mode: 'motion', name: 'Logo 动画生成器', icon: 'Hexagon',
    headline: '给品牌做一个会动的开场',
    summary: '描述你的品牌名和标志，AI 让图形自己拼出来、线条描绘、字标浮现，适合视频片头和官网首屏。',
    points: ['线条描绘、拼合、变形多种出场', '最多 3 种颜色，干净利落', '可导出 HTML 直接放到网页'],
    prompts: ['HiveGPT：六边形蜂巢逐格点亮，最后出现字标', '「山海」：一笔画出远山和海浪，再浮现文字', '咖啡品牌 Morning：一杯咖啡升起热气，变成字母 M']
  },
  {
    slug: 'text-animation', category: 'text', mode: 'motion', name: '文字动画生成器', icon: 'Type',
    headline: '动态排版，让一句话更有冲击力',
    summary: '逐字出现、放大强调、打字机、弹跳……把口号、标题、金句做成动画，适合短视频封面和开场。',
    points: ['关键字自动放大变色', '竖屏也不会溢出', '多种节奏：快剪、优雅、俏皮'],
    prompts: ['「一个 Key，畅用主流 AI」逐字出现，「主流 AI」放大变色', '倒计时 3、2、1，然后炸开「新年快乐」', '打字机效果打出：Hello, World']
  },
  {
    slug: 'flowchart', category: 'flowchart', mode: 'motion', name: '流程图动画', icon: 'Workflow',
    headline: '流程和架构，一步一步亮起来',
    summary: '节点按顺序出现，连线逐段画出，高亮沿着路径流动。适合讲解业务流程、系统架构、算法步骤。',
    points: ['支持分支、循环和并行', '节点文字简短清晰', '适合技术分享和产品文档'],
    prompts: ['HTTPS 握手：客户端问候 → 服务器证书 → 密钥交换 → 加密通信', '用户下单到签收的完整流程，失败时退款', '一次网页请求经过 DNS、CDN、负载均衡到服务器']
  },
  {
    slug: 'map-animation', category: 'map', mode: 'motion', name: '地图路线动画', icon: 'Map',
    headline: '路线、迁徙和分布，在地图上动起来',
    summary: '风格化地图上画出路线、移动标记、弹出地名和数据，适合历史、旅行和物流内容。',
    points: ['路线逐段绘制，标记沿线移动', '地名和距离依次出现', '多种地图风格'],
    prompts: ['丝绸之路：从长安出发，途经敦煌、撒马尔罕到罗马', '郑和下西洋的七次航线', '一个包裹从深圳到北京的物流路线']
  },
  {
    slug: 'product-demo', category: 'product', mode: 'motion', name: '产品演示动画', icon: 'MousePointerClick',
    headline: '不用录屏，也能演示产品操作',
    summary: '在手机或浏览器框里画出 App 界面，鼠标移动点击、页面切换、功能标注，适合官网和发布会。',
    points: ['自动生成界面和交互过程', '重点功能有标注说明', '适合应用商店预览和官网'],
    prompts: ['一个待办 App：点击加号、输入任务、勾选完成', '在线文档里选中文字、点 AI 改写、替换结果', '电商 App 加入购物车到下单的过程']
  },
  {
    slug: 'loading-animation', category: 'loader', mode: 'motion', name: '加载动画生成器', icon: 'Loader',
    headline: '一句话生成 Loading 和微交互',
    summary: '加载动画、进度环、骨架屏、按钮反馈，无缝循环，导出 HTML 直接用在项目里。',
    points: ['无缝循环，没有跳帧', '尺寸小，适配深浅背景', '适合前端项目和设计稿'],
    prompts: ['三个圆点依次跳动，像波浪一样', '环形进度条从 0% 走到 100% 后重来', '一个心跳形状的加载动画']
  },
  {
    slug: 'stick-figure', category: 'stick-figure', mode: 'motion', name: '火柴人动画', icon: 'PersonStanding',
    headline: '火柴人也能演一出好戏',
    summary: '跑、跳、打招呼、对话……关节动作自然，适合做表情包、小剧场和讲解里的角色。',
    points: ['骨骼动画，动作流畅', '可以多人互动', '适合轻松幽默的内容'],
    prompts: ['火柴人跑步、起跳、翻越栏杆的循环', '两个火柴人击掌庆祝', '火柴人在白板前讲课']
  },
  {
    slug: 'line-drawing', category: 'line-drawing', mode: 'motion', name: '线条描绘动画', icon: 'PenTool',
    headline: '一笔一画，描出你想要的图',
    summary: '线稿按顺序逐笔描出，再点亮色彩和细节。适合城市天际线、建筑、肖像轮廓和品牌插画。',
    points: ['描绘顺序有设计感', '描完再上色和点亮', '线条粗细自然'],
    prompts: ['上海天际线线稿逐段描出，东方明珠最后亮起', '一笔画出一只猫，最后眨一下眼', '描出一辆复古汽车的轮廓']
  },
  {
    slug: '3d-animation', category: '3d', mode: 'motion', name: '3D 动画', icon: 'Box',
    headline: '纯网页实现的立体动画',
    summary: '用 CSS 3D 和透视计算做出旋转、翻转、立体展开的效果，不依赖任何外部库。',
    points: ['透视、旋转、景深', '适合产品展示和数学几何', '网页直接播放'],
    prompts: ['一个旋转的魔方，逐层转动后还原', '一张名片 3D 翻转，正反面不同', '立方体展开成六个面']
  },
  {
    slug: 'hand-drawn', category: 'hand-drawn', mode: 'motion', name: '手绘动画', icon: 'Brush',
    headline: '像在白板上现场画出来',
    summary: '略带抖动的手绘线条、马克笔上色、手写字，亲切又有温度，适合讲解和教育内容。',
    points: ['手绘质感，线条有粗细变化', '画完再上色', '适合知识讲解'],
    prompts: ['在白板上画出 TCP 三次握手', '画一棵树从种子长大，四季变化', '手绘一张学习计划表']
  },
  {
    slug: 'mini-game', category: 'game', mode: 'motion', name: '小游戏演示动画', icon: 'Gamepad2',
    headline: '经典小游戏的自动演示',
    summary: '贪吃蛇、打砖块、跑酷……AI 写出按时间自动进行的游戏画面，适合讲解算法或做趣味内容。',
    points: ['画面按时间确定，可导出', '像素风或现代风', '适合算法讲解'],
    prompts: ['贪吃蛇自动吃到 5 个果子', '打砖块自动打掉第一排', '迷宫寻路：BFS 一步步找到出口']
  },
  {
    slug: 'others', category: 'other', mode: 'motion', name: '自由创作', icon: 'Sparkles',
    headline: '想到什么，就做什么',
    summary: '任何会动的画面：插画场景、天气、节日贺卡、Vlog 片头……写下想法，AI 帮你实现。',
    points: ['不限题材和风格', '可以继续对话修改', '支持横竖屏'],
    prompts: ['地球自转，卫星绕轨道飞行，夜晚一侧城市亮灯', '雨夜的城市街道，窗户灯光闪烁', '中秋节贺卡：月亮升起，玉兔跳出来']
  }
]

export const findTool = (slug) => TOOLS.find((t) => t.slug === slug)
