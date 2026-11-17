// HiveGPT: Chinese text for Rusted Cannon (MIT, Cyril Dieumegard).
// The game draws all its text with fillText, so the Chinese version translates at that point:
// exact phrases from ZH, then patterns for the strings built at runtime. 「EN」 switches back.
(function () {
  var lang = 'zh';
  try {
    if (localStorage.getItem('rc_lang') === 'en') lang = 'en';
  } catch (e) {}
  window.RC_LANG = lang;
  document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';

  var ZH = {
    // menus and screens
    'PLAY': '开始游戏',
    'OPTIONS': '设置',
    'ESCAPE': '离开',
    'TITLE': '标题',
    'MERCENARY ASSAULT': '佣兵突击',
    'ARCADE RUN & GUN  //  RUSTED CANNON UNIT': '街机横版射击  //  锈炮小队',
    '↑ ↓ / W S  SELECT     ENTER / Z  CONFIRM     M  SOUND': '↑ ↓ / W S 选择    回车 / Z 确认    M 声音',
    'SYSTEM CONFIGURATION': '系统设置',
    'DIFFICULTY': '难度',
    'MUSIC': '音乐',
    'SFX': '音效',
    'SCREEN SHAKE': '屏幕震动',
    'SCANLINES': '扫描线',
    'INTEGER SCALE': '整数缩放',
    'BACK': '返回',
    'ENTER': '回车',
    'ON': '开',
    'OFF': '关',
    'LOW': '弱',
    'NORMAL': '正常',
    'SELECT / CHANGE': '选择 / 修改',
    'MOVE WASD/ARROWS   FIRE Z/J   JUMP X/K   GRENADE C/L': '移动 WASD/方向键  射击 Z/J  跳 X/K  手雷 C/L',
    'ESC BACK   ENTER CONFIRM   M MUTE': 'ESC 返回   回车 确认   M 静音',
    'MISSION BRIEFING': '任务简报',
    'ENCRYPTED RADIO LINK': '加密无线电',
    'LIVE': '在线',
    'ENTER / Z': '回车 / Z',
    'NEXT MESSAGE': '下一条',
    'START MISSION': '开始任务',
    'RUSTED CANNON UNIT // SECURE CHANNEL': '锈炮小队 // 加密频道',
    'FIELD PAUSE': '战地暂停',
    'PAUSED': '已暂停',
    'MISSION': '任务',
    'SCORE': '得分',
    'POWS': '俘虏',
    'LOADOUT': '装备',
    'GRENADES': '手雷',
    'ESC / ENTER': 'ESC / 回车',
    'RESUME': '继续',
    'MUTE AUDIO': '静音',
    'AFTER ACTION REPORT': '战后报告',
    'MISSION COMPLETE': '任务完成',
    'VIEW ENDING': '观看结局',
    'NEXT MISSION': '下一关',
    'RUSTED CANNON CASUALTY REPORT': '锈炮阵亡报告',
    'OPERATION FAILED': '行动失败',
    'FINAL SCORE': '最终得分',
    'CONTINUE TOKENS': '剩余续关',
    'THE GUN FALLS SILENT. THE WAR GOES ON.': '枪声停了，战争还在继续。',
    'CONTINUE MISSION': '续关再战',
    'ESC': 'ESC',
    'RETURN TO TITLE': '回到标题',
    'CAMPAIGN ARCHIVE': '战役档案',
    'IRON HIVE DESTROYED': '钢铁蜂巢已摧毁',
    'WASTED': '阵亡',
    'GAMEOVER': '游戏结束',
    'GAME OVER': '游戏结束',
    'CLEAR': '过关',
    'START!': '开始！',
    'PAUSE': '暂停',
    'ENDING': '结局',
    'CLOSE THIS TAB TO EXIT.': '关掉这个页面就能离开。',
    'LOADING...': '正在加载……',
    'LOADING ASSETS...': '正在加载素材……',
    'PRESS ENTER / Z': '按回车 / Z 开始',
    // hud
    'REST': '剩余',
    'HP': '血',
    'RUN': '跑',
    // speakers
    'RADIO': '无线电',
    'RADIO OPS': '无线电指挥',
    'PLAYER': '鲁克',
    'ANNOUNCER': '播报',
    'SYSTEM': '系统',
    'BOSS': '首领',
    'POW': '俘虏',
    // world
    'RUSTED CANNON': '锈炮突击',
    'ELON "ROOK" KANE': '“鲁克”凯恩',
    'RUSTED CANNON UNIT': '锈炮小队',
    'FREE FLARE IRREGULARS': '自由信号弹游击队',
    'CORDITE SYNDICATE': '柯代特财团',
    'SERVICE PISTOL': '制式手枪',
    'RIPPER RIFLE': '撕裂者机枪',
    'HOWLER ROCKET': '嚎叫火箭',
    'SCRAP MULE': '废铁骡战车',
    'ASH BOULEVARD': '灰烬大道',
    'CITY TO SCORCHED HIGHWAY': '从城区杀到焦土公路',
    'GREEN HELL RUN': '绿色地狱',
    'JUNGLE RIVER PUSH': '丛林河道推进',
    'IRON HIVE': '钢铁蜂巢',
    'BASE / HANGAR CORE': '基地 / 机库核心',
    'IRONCLAD MARAUDER': '铁甲掠夺者',
    'MOSS GUNBOAT': '苔藓炮艇',
    'IRON HIVE CORE': '钢铁蜂巢核心',
    'RECRUIT': '新兵',
    'ARCADE': '街机',
    'VETERAN': '老兵',
    'RESCUE ALL 5 POWS': '救出全部 5 名俘虏',
    'DESTROY THE CONVOY': '摧毁敌方车队',
    'CONVOY': '车队',
    'FINISH WITHOUT DYING': '一命通关',
    'NO DEATH': '不死',
    'CHECKPOINT +HP': '检查点 +血',
    'THANK YOU!  +HP': '谢谢你！ +血',
    '+GRENADES': '+手雷',
    '+HP': '+血',
    'REPAIRED': '已修复',
    'VEHICLE ACQUIRED!': '获得载具！',
    // radio and shouts
    'RUSTED CANNON! ONE MERCENARY. NO RETREAT!': '锈炮突击！一名佣兵，绝不后退！',
    'MISSION START!': '任务开始！',
    'MISSION CLEAR!': '任务完成！',
    'RUSTED CANNON, MISSION COMPLETE.': '锈炮，任务完成。',
    'CAMPAIGN CLEAR!': '战役通关！',
    'CONTINUE?': '要续关吗？',
    'ROOK, CORDITE HAS SEIZED ASH BOULEVARD.': '鲁克，柯代特财团占领了灰烬大道。',
    'BREAK THROUGH THE CHECKPOINT AND FREE THE MARKED POWS.': '突破检查站，救出标记的俘虏。',
    'EXPECT CONVOY ARMOR, THEN AN IRONCLAD MARAUDER.': '先是装甲车队，然后是一台铁甲掠夺者。',
    'THE CITY IS BURNING. KEEP MOVING.': '城市在燃烧，别停下。',
    'COPY. RUSTED CANNON MOVING IN.': '收到，锈炮出发。',
    'JUNGLE GREEN, RIVER GRAY — BAD POETRY, WORSE FLAK.': '丛林是绿的，河是灰的——诗很烂，防空炮更烂。',
    'SCRAP MULE IS STAGED MID-ROUTE. MOUNT IT.': '废铁骡战车停在半路，开上它。',
    'RESCUE CAGES ARE TUCKED UNDER THE CANOPY.': '俘虏笼藏在树冠下面。',
    'A GUNBOAT CONTROLS THE RIVER BEND.': '一艘炮艇守着河湾。',
    'THEN WE TAKE THE RIVER.': '那我们就拿下这条河。',
    'IRON HIVE FRONT DOOR IS A SUGGESTION. USE EXPLOSIVES.': '钢铁蜂巢的大门只是个摆设，用炸药开。',
    'TURRET HALLWAY, AMBUSH CLOSETS, NO POLITE LOBBY.': '炮塔走廊、伏兵暗格，没有客气的前台。',
    'THE CORE CHAMBER HOUSES THEIR COMMAND MECH.': '核心舱里是他们的指挥机甲。',
    'BRING THE WHOLE FACILITY DOWN.': '把整座设施给我拆了。',
    'RUSTED CANNON. FINAL RUN.': '锈炮，最后一程。',
    'STILL STANDING.': '还站着呢。',
    'NOT TODAY.': '今天还不行。',
    'I HAVE HAD WORSE.': '比这惨的我都见过。',
    'CLEAR OUT!': '清场！',
    'FIRE IN THE HOLE!': '小心手雷！',
    'CATCH!': '接着！',
    'MULE MINES!': '骡车布雷！',
    'TARGET DOWN.': '目标击毁。',
    'AREA SECURE.': '区域安全。',
    'THANK YOU!': '谢谢你！',
    'I AM GETTING OUT!': '我要出去了！',
    'RUNNING FREE!': '自由啦！',
    'YOU CAME BACK FOR US!': '你回来救我们了！',
    'YOU SAVED MY LIFE!': '你救了我的命！',
    'GO GO GO!': '冲冲冲！',
    'YOU STOP HERE!': '你到此为止了！',
    'CRUSH THE MERCENARY!': '碾碎那个佣兵！',
    'PAINT THE ASPHALT!': '把柏油路染红！',
    'THE RIVER IS OURS!': '这条河是我们的！',
    'SWIM IN TRACERS!': '在曳光弹里游泳吧！',
    'MOSS EATS HEROES!': '苔藓专吃英雄！',
    'PHASE LOCK ENGAGED!': '相位锁定启动！',
    'THE HIVE REMEMBERS!': '蜂巢会记住的！',
    'RUSTED CANNON DIES HERE!': '锈炮就死在这里！',
    'CORE OVERCLOCK!': '核心超频！',
    'KNEEL BEFORE THE HIVE!': '向蜂巢跪下！',
    'ALL WEAPONS FREE!': '全武器开火！',
    'IMPOSSIBLE... THE CORE...': '不可能……核心……',
    'RE-LINKED. STAY MEAN.': '重新连线。保持凶狠。',
    'ARMOR SPLIT. TURRET OVERDRIVE!': '装甲裂开，炮塔超载！',
    'RIVER GUNS UNLOCKED!': '河道炮台解锁！',
    'CORE SHIELDS COLLAPSING!': '核心护盾正在崩溃！',
    'IRONCLAD OVERDRIVE!': '铁甲超载！',
    'GUNBOAT FINAL BARRAGE!': '炮艇最后齐射！',
    'CORE MELTDOWN PROTOCOL!': '核心熔毁协议！',
    'HOSTILE BIRD ON THE SKYLINE — KEEP LOW!': '天上有敌机——压低身子！',
    'A BATTLE-SCARRED MERCENARY TEARS THROUGH A ROGUE PMC AND ITS SCRAP-IRON WAR MACHINES.': '一名伤痕累累的佣兵，杀穿失控的私人军队和它的废铁战争机器。'
  }

  var SHORT = { POW: '俘虏', CONVOY: '车队', 'NO DEATH': '不死' }
  var WEAPON = { PST: '手枪', HMG: '重机枪', RKT: '火箭' }

  function word(s) {
    var k = String(s).trim().toUpperCase()
    return ZH[k] !== undefined ? ZH[k] : s
  }

  var RULES = [
    [/^MISSION (\d+)$/, function (m) { return '第 ' + m[1] + ' 关' }],
    [/^LIVES X(\d+)$/, function (m) { return '剩余 ' + m[1] + ' 条命' }],
    [/^BOSS PHASE (\d+)$/, function (m) { return '首领 第 ' + m[1] + ' 阶段' }],
    [/^(\d+) HIT\s+X([\d.]+)$/, function (m) { return m[1] + ' 连击 ×' + m[2] }],
    [/^OBJ NO DEATH (OK|FAIL)$/, function (m) { return '目标 不死 ' + (m[1] === 'OK' ? '达成' : '失败') }],
    [/^OBJ (.+?) (\d+)\/(\d+)$/, function (m) { return '目标 ' + (SHORT[m[1]] || m[1]) + ' ' + m[2] + '/' + m[3] }],
    [/^W:(PST|HMG|RKT)$/, function (m) { return '枪:' + WEAPON[m[1]] }],
    [/^A:INF$/, function () { return '弹:∞' }],
    [/^A:(\d+)$/, function (m) { return '弹:' + m[1] }],
    [/^G:(\d+)$/, function (m) { return '雷:' + m[1] }],
    [/^BONUS OBJECTIVE: (.+)$/, function (m) { return '附加目标：' + word(m[1]) }],
    [/^OBJECTIVE COMPLETE\s+(.+)$/, function (m) { return '目标达成  ' + word(m[1]) }],
    [/^OBJECTIVE FAILED\s+(.+)$/, function (m) { return '目标失败  ' + word(m[1]) }],
    [/^SCORE\s+(\d+)$/, function (m) { return '得分  ' + m[1] }],
    [/^FINAL SCORE\s+(\d+)$/, function (m) { return '最终得分  ' + m[1] }],
    [/^POWS FREED\s+(\d+)$/, function (m) { return '救出俘虏  ' + m[1] }],
    [/^HOSTILES\s+(\d+)$/, function (m) { return '击毙敌人  ' + m[1] }],
    [/^POWS (\d+)\s+KILLS (\d+)\s+DEATHS (\d+)$/, function (m) { return '俘虏 ' + m[1] + '   击毙 ' + m[2] + '   阵亡 ' + m[3] }],
    [/^MAX COMBO (\d+)\s+SCENERY (\d+)$/, function (m) { return '最高连击 ' + m[1] + '   破坏场景 ' + m[2] }],
    [/^MISSION BONUS\s+\+(\d+)$/, function (m) { return '任务奖励  +' + m[1] }],
    [/^(M\d)\s+(.+)$/, function (m) { return m[1] + '  ' + word(m[2]) }],
    [/^CALLSIGN \/\/ (.+)$/, function (m) { return '呼号 // ' + word(m[1]) }],
    [/^(RECRUIT|ARCADE|VETERAN) \/\/ (.+)$/, function (m) { return word(m[1]) + ' // ' + word(m[2]) }],
    [/^(.+) WINS\.$/, function (m) { return word(m[1]) + ' 获胜。' }],
    [/^RADIO: (.+)$/, function (m) { return '无线电：' + word(m[1]) }],
    [/^ANNOUNCER: (.+)$/, function (m) { return '播报：' + word(m[1]) }]
  ]

  function t(s) {
    if (lang !== 'zh' || typeof s !== 'string' || !s) return s
    var k = s.trim().toUpperCase()
    if (ZH[k] !== undefined) return ZH[k]
    for (var i = 0; i < RULES.length; i++) {
      var m = RULES[i][0].exec(k)
      if (m) return RULES[i][1](m)
    }
    return s
  }
  window.RC_T = t

  if (lang === 'zh' && window.CanvasRenderingContext2D) {
    var P = CanvasRenderingContext2D.prototype
    var fill = P.fillText
    var stroke = P.strokeText
    var measure = P.measureText
    P.fillText = function (s, x, y, w) {
      return w === undefined ? fill.call(this, t(String(s)), x, y) : fill.call(this, t(String(s)), x, y, w)
    }
    P.strokeText = function (s, x, y, w) {
      return w === undefined ? stroke.call(this, t(String(s)), x, y) : stroke.call(this, t(String(s)), x, y, w)
    }
    P.measureText = function (s) {
      return measure.call(this, t(String(s)))
    }
  }

  // page chrome: elements carry both languages
  function chrome() {
    var els = document.querySelectorAll('[data-en]')
    for (var i = 0; i < els.length; i++) {
      var el = els[i]
      if (!el.hasAttribute('data-zh')) el.setAttribute('data-zh', el.innerHTML)
      el.innerHTML = lang === 'en' ? el.getAttribute('data-en') : el.getAttribute('data-zh')
    }
    var b = document.getElementById('lang-toggle')
    if (b) {
      b.textContent = lang === 'zh' ? 'EN' : '中文'
      b.onclick = function () {
        try {
          localStorage.setItem('rc_lang', lang === 'zh' ? 'en' : 'zh')
        } catch (e) {}
        location.reload()
      }
    }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', chrome)
  else chrome()
})()
