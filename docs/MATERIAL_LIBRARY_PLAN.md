# 诗画素材库方案（未开工）

目标：把「文字 → 画面」这件事做成可复用的素材库。古诗、歌曲、原创短句是文字的来源，意象把文字翻译成画面元素，画风、构图、配色决定怎么画，作品库记下每张图从哪来、用了什么、能不能公开带字发布。

现有能力可以直接复用：

| 现有 | 复用方式 |
|---|---|
| 课本诗画 `shihua_works`（`backend/migrations/310_shihua.sql`），带两次校对的诗文 | 第一期导入古诗库的种子数据；以后 `shihua_works` 加 `text_id` 指向素材库 |
| 课本诗画 `shihua_images` 的出图队列（排队、重试、记成本） | 作品库的出图直接走同一套队列，或照着它实现 |
| 课本诗画 `shihua_renders.layout` | 版式模板库的 `slots` 结构参照它 |
| `promo/lyrics-art/lyric2art.py` 的 9 个画风 | 画风库种子数据 |
| canvas `web/public/fonts` 的 OFL 字体 | 字体库种子数据 |
| 提示词市场 `prompt_listings`（`306_prompt_market.sql`） | 第三期把好用的配方上架 |

---

## 1. 版权规则（所有库共用，先于一切）

每条文字素材必须有 `license_status`，决定**原文能不能存、能不能展示、能不能印到画上**：

| license_status | 含义 | 存原文 | 前台展示原文 | 印在画上 |
|---|---|---|---|---|
| `public_domain` | 公有领域：作者去世满 50 年（从去世次年起算第 51 年 1 月 1 日起），或古代作品 | 可以 | 可以 | 可以 |
| `original` | 平台或用户原创，权利归属明确 | 可以 | 可以 | 可以 |
| `licensed` | 有书面授权，`license_note` 写清范围和期限 | 按授权 | 按授权 | 按授权 |
| `reference_only` | 只作参考：流行歌词、在世或去世未满 50 年作者的作品 | **不存** | 不可以 | 不可以 |

- 流行歌曲一律 `reference_only`：只存歌名、歌手、词作者、段落位置、我们自己写的画面描述和正版平台链接。数据库加 CHECK 约束：`reference_only` 时 `body`、`content` 必须为空。
- 译文、注释、赏析也有版权。网上和教材里的不能抄，只存自己写或 AI 生成、经人工改过的版本，标 `original`。
- 作者表记卒年，系统按卒年自动建议 `license_status`，人工确认后生效。例：徐志摩（1931 年卒）可用；海子（1989 年卒）要到 2040 年 1 月 1 日。
- 作品库出版式图时，文字层只允许取 `public_domain` / `original` / `licensed` 的内容，后端校验，不靠前端自觉。
- 所有出图标注「配图由 AI 生成」。

---

## 2. 各库字段

通用字段（每张表都有，下面不再重复）：`id`、`created_by`、`created_at`、`updated_at`。需要审核的库另有 `status`（`draft` / `review` / `published` / `hidden`）。

### 2.1 作者库 `mat_authors`

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(64) | 姓名 |
| aliases | jsonb | 字、号、笔名，如「太白」「青莲居士」 |
| dynasty | varchar(16) | 朝代；现代作者填「现代」「当代」 |
| birth_year / death_year | int, null | 生卒年，`death_year` 用来算版权；在世为空 |
| bio | text | 简介，自写 |
| kind | varchar(16) | `poet` / `lyricist` / `writer` / `platform`（平台原创） |

### 2.2 文字素材库 `mat_texts`

一首诗、一首歌、一组原创短句各是一条。

| 字段 | 类型 | 说明 |
|---|---|---|
| slug | varchar(64) unique | 如 `shi-zhi-saishang` |
| kind | varchar(16) | `shi` 诗 / `ci` 词 / `qu` 曲 / `wen` 文 / `modern_poem` 现代诗 / `song` 歌曲 / `original` 原创短句 |
| title | varchar(128) | 标题；歌曲填歌名 |
| author_id | bigint → mat_authors | 作者；歌曲指词作者 |
| performer | varchar(64) | 歌曲的演唱者，其它为空 |
| body | text | 全文。`reference_only` 时必须为空 |
| translation | text | 白话译文，自写 |
| appreciation | text | 赏析，自写 |
| license_status | varchar(16) | 见第 1 节 |
| license_note | text | 授权范围、期限、来源证明 |
| sources | text | 底本来源（如维基文库、某出版社版本），便于校对 |
| verify1_by / verify1_at / verify2_by / verify2_at | | 两次校对，沿用课本诗画做法；未两次校对不得上画 |
| external_url | text | 歌曲的正版平台链接 |
| themes | jsonb | 主题标签：送别、思乡、边塞、田园、爱情…… |
| seasons | jsonb | 春 / 夏 / 秋 / 冬 |
| textbook | jsonb | 收录教材与年级，如 `[{"stage":"初中","grade":"八上"}]` |
| shihua_work_id | bigint, null | 从课本诗画导入时记来源 |

### 2.3 片段库 `mat_text_segments`

画面是按「一段」画的，意象挂在片段上，不挂在整首上。

| 字段 | 类型 | 说明 |
|---|---|---|
| text_id | bigint → mat_texts | 所属作品 |
| seq | int | 第几段 |
| label | varchar(32) | 段落位置：「首联」「下阕」「第二段主歌」「副歌」 |
| content | text | 片段原文。`reference_only` 时必须为空 |
| scene_desc | text | 自写的画面描述（中文），歌曲只靠这一栏 |
| is_famous | bool | 名句，优先出图 |

### 2.4 意象库 `mat_imagery`（核心）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) unique | 「孤烟」「长河」「落日」「柳」「孤灯」「萤火虫」 |
| aliases | jsonb | 同义写法：「杨柳」「垂柳」 |
| category | varchar(16) | 天象 / 地理 / 植物 / 动物 / 器物 / 建筑 / 人物动作 / 气候 |
| meaning | text | 文化含义，如「柳：谐音留，送别」 |
| visual_zh | text | 画面里具体画什么，中文 |
| visual_en | text | 同上，英文提示词片段 |
| emotions | jsonb | 常带的情绪：离别、孤独、壮阔…… |
| seasons / times / weathers | jsonb | 常配季节、时辰、天气 |
| eras | jsonb | 有时代限制的意象写朝代范围，如「烽火台」 |
| palette_ids | jsonb | 推荐配色 |
| style_ids | jsonb | 推荐画风 |
| example_artwork_id | bigint, null | 代表作品 |

关联表 `mat_segment_imagery`：`segment_id`、`imagery_id`、`weight`（主次，1–3）、`source`（`ai` / `manual`）、`confirmed`（AI 拆出来的要人工确认）。

### 2.5 画风库 `mat_styles`

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) unique | 「工笔青花」「淡彩水墨」 |
| category | varchar(16) | 国画 / 西画 / 动画 / 摄影 / 版画 / 插画 |
| desc_zh | text | 一句话说明 |
| prompt_en | text | 风格提示词。**不得出现画家、工作室、作品名**（由禁忌库校验） |
| negative_en | text | 补充排除项 |
| suits_emotions / suits_categories | jsonb | 适合的情绪和意象类别，用于推荐 |
| default_palette_id | bigint, null | 默认配色 |
| default_size / default_quality | varchar | 如 `1024x1536` / `high` |
| sample_artwork_ids | jsonb | 样图 |
| enabled / sort | bool / int | 上下架与排序 |

### 2.6 构图库 `mat_compositions`

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) | 背影、远景小人、大留白、框景、俯瞰、近景特写、三分法 |
| desc_zh / prompt_en | text | 说明与提示词片段 |
| aspects | jsonb | 适合的比例：竖 / 横 / 方 |
| suits_emotions | jsonb | 孤独常配「远景小人 + 大留白」 |
| example_artwork_id | bigint, null | 示例 |

### 2.7 配色库 `mat_palettes`

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) | 青花、宋画绢色、敦煌、黛蓝、莫兰迪 |
| colors | jsonb | `[{"hex":"#1E3A8A","name":"霁蓝"}]`，3–6 色 |
| prompt_en | text | 色彩描述提示词 |
| moods | jsonb | 适合的情绪 |
| source | text | 色名出处（传统色名、文物） |

### 2.8 情绪光线库 `mat_moods`

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(16) | 离别、思乡、壮阔、怀旧、闲适、孤独 |
| light_en | text | 光线：「低角度暖黄夕光」「冷白月光、长影」 |
| weather_en | text | 天气、空气感：薄雾、细雨、风 |
| palette_ids | jsonb | 推荐配色 |

### 2.9 角色库 `mat_characters`（第三期）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) | 原创角色名 |
| appearance_zh / prompt_en | text | 外貌、服饰设定 |
| era | varchar(16) | 适用朝代或「现代」 |
| ref_images | jsonb | 参考图，用于保持一致 |
| license_status | varchar(16) | 只能是 `original` |

### 2.10 器物服饰建筑库 `mat_period_items`（第三期）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) | 襦裙、幞头、长亭、乌篷船、油灯、琵琶 |
| kind | varchar(16) | 服饰 / 发式 / 器物 / 建筑 / 交通 / 乐器 |
| dynasty_from / dynasty_to | varchar(16) | 适用朝代范围 |
| prompt_en | text | 画法描述 |
| common_mistakes | text | 常见错误，如「唐代人物不要画成清装」 |
| ref_source | text | 依据（博物馆藏品、考古资料） |

### 2.11 节气节日库 `mat_calendar`（第二期）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(16) | 立春、清明、中秋…… |
| kind | varchar(8) | `term` 节气 / `festival` 节日 |
| date_rule | varchar(32) | 公历或农历规则，节气每年按表算 |
| imagery_ids / text_ids | jsonb | 关联意象和诗文，用于定时出内容 |

### 2.12 字体库 `mat_fonts`（第二期）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(64) | 字体名 |
| file | text | 文件路径 |
| license | varchar(32) | OFL / Apache / 已购商用授权 |
| commercial_ok | bool | 非 true 不得用于出图 |
| tags | jsonb | 楷 / 行 / 宋 / 手写 / 篆 |
| supports_vertical | bool | 能否竖排 |

### 2.13 版式模板库 `mat_layouts`（第二期）

| 字段 | 类型 | 说明 |
|---|---|---|
| name | varchar(32) | 竖排题诗、小红书封面、公众号头图 |
| aspect | varchar(8) | 3:4、2:3、16:9、2.35:1 |
| width / height | int | 输出尺寸 |
| slots | jsonb | 标题、正文、作者、印章、「AI 生成」标注的位置、字号、字体、方向，结构参照 `shihua_renders.layout` |

### 2.14 作品库 `mat_artworks`

每张生成的图一条，能追溯、能重画。

| 字段 | 类型 | 说明 |
|---|---|---|
| segment_id | bigint, null → mat_text_segments | 画的是哪一段；纯画风测试可为空 |
| style_id / composition_id / palette_id / mood_id | bigint, null | 用了什么 |
| prompt_final | text | 最终发给模型的完整提示词 |
| prompt_parts | jsonb | 各部分原样记下：方向、构图、意象、器物、光线、画风、配色、禁字句，便于单独替换重画 |
| model / size / quality | varchar | 如 `gpt-image-2` / `1024x1536` / `high` |
| file / width / height | text / int | 文件与实际尺寸 |
| cost_usd / duration_ms | numeric / int | 成本与耗时 |
| parent_id | bigint, null | 重画自哪一张 |
| source | varchar(16) | `library` / `shihua` / `lyric2art` / `canvas` |
| rating | smallint | 人工评分 1–5，用于推荐和提示词库 |
| review_status | varchar(16) | `pending` / `approved` / `rejected`；拒绝时写 `reject_reason`（有字、多指、时代错误……） |
| text_overlay_allowed | bool | 由所用文字的 `license_status` 自动算出，不可手改 |
| published_to | jsonb | 发到了哪里：小红书、公众号、站内 |

关联表 `mat_artwork_imagery`（作品用了哪些意象）、`mat_artwork_characters`（第三期）。成品排版图沿用 `shihua_renders` 的结构，另建 `mat_artwork_renders`：`artwork_id`、`layout_id`、`file`、`layout`（实际用的文字、字体、位置）。

### 2.15 禁忌库 `mat_blocklist`

| 字段 | 类型 | 说明 |
|---|---|---|
| term | varchar(128) | 词或正则 |
| kind | varchar(16) | `artist` 在世画家 / `studio` 工作室 / `character` 版权角色 / `person` 真人 / `symbol` 敏感标志 / `text` 让模型写字的说法 |
| match | varchar(8) | `contains` / `regex` |
| action | varchar(8) | `block` 拒绝 / `warn` 提醒 / `strip` 自动删掉 |
| applies_to | varchar(8) | `prompt` / `text` / `both` |
| reason | text | 为什么禁 |

组装提示词后、发给模型前统一过一遍；画风库保存时也过一遍。

### 2.16 提示词配方库 `mat_recipes`（第三期）

把评分高的作品沉淀成配方：`name`、`style_id`、`composition_id`、`palette_id`、`mood_id`、`template`（带 `{imagery}` 占位的提示词模板）、`avg_rating`、`use_count`、`prompt_listing_id`（上架到提示词市场后关联）。

---

## 3. 库之间怎么关联

```
mat_authors ─1:n─ mat_texts ─1:n─ mat_text_segments ─n:n─ mat_imagery
                     │                    │                  │ 推荐
                     │ license_status     │                  ├─→ mat_styles
                     │ 决定能否上字        │                  ├─→ mat_palettes
                     ▼                    ▼                  └─→ mat_moods
              mat_artworks ◀── segment_id, style_id, composition_id, palette_id, mood_id
                     │  n:n mat_imagery（实际用了哪些意象）
                     │  n:n mat_characters、引用 mat_period_items（第三期）
                     ▼
            mat_artwork_renders ── mat_layouts ── mat_fonts

mat_calendar ─→ mat_imagery / mat_texts（按日期挑内容）
mat_blocklist  在保存画风、组装提示词时校验
mat_recipes    从高分 mat_artworks 沉淀，可上架提示词市场
```

### 出图流程

1. **选片段**：从古诗库挑一段（或节气日历自动挑）。
2. **拆意象**：AI 从片段原文或 `scene_desc` 拆出意象，匹配 `mat_imagery`；没有的意象生成草稿进审核。人工确认后写 `mat_segment_imagery`。
3. **推荐搭配**：按意象的情绪、季节打分，推荐画风、构图、配色、情绪光线；人可改。
4. **补时代元素**（第三期）：按诗的朝代从 `mat_period_items` 取服饰、建筑，避免时代错误。
5. **组装提示词**：方向 → 构图 → 意象画面（按 weight 排序）→ 时代元素 → 光线天气 → 画风 → 配色 → 禁字句。
6. **禁忌校验**：过 `mat_blocklist`。
7. **出图**：走现有出图队列，写 `mat_artworks`。
8. **审核评分**：看有没有字、多指、时代错误，打分。
9. **排版**：选版式和字体；只有 `text_overlay_allowed = true` 才能放原文，否则只放标题和自写短句。

---

## 4. 分期

### 第一期：古诗出图闭环（后台使用）

| 做 | 内容 |
|---|---|
| 作者库 + 文字素材库 + 片段库 | 只收古诗文（`kind` 为 shi / ci / qu / wen），全部 `public_domain`；先导入课本诗画里两次校对过的诗文，再从维基文库补 |
| 意象库 | 种子 150 个常见古诗意象，AI 起草、人工审 |
| 画风库 | 种子 9 个（来自 `lyric2art.py`），每个配 2–3 张样图 |
| 作品库 + 作品意象关联 | 出图、审核、评分、重画 |
| 禁忌库（最小版） | 先放「让模型写字」的说法和一批画家、工作室名 |
| 版权字段与校验 | `license_status`、CHECK 约束、`text_overlay_allowed` 后端计算 |

后台入口 `/admin/materials`，形态参照 `/admin/shihua`。构图、配色、情绪光线第一期先做成画风上的可选 jsonb 字段，不单独建表。

验收标准：从古诗库任选一首未画过的诗，10 分钟内完成「拆意象 → 选搭配 → 出 3 张 → 选 1 张排版导出」，导出图上的诗文与校对版本一字不差。

### 第二期：扩素材、做内容排期

- 歌曲意象（`kind = song`，全部 `reference_only`，只存自写画面描述）
- 原创短句（`kind = original`）
- 构图库、配色库、情绪光线库拆成独立表
- 节气节日库，按日期给出当天可画的内容
- 字体库、版式模板库、成品排版

### 第三期：一致性与对外

- 角色库、器物服饰建筑库
- 提示词配方库，高分配方上架提示词市场
- 对用户开放的素材广场：只展示可公开的文字和已审核作品；用户投稿走审核，投稿须声明权利归属

---

## 5. 待确认

- 第一期只给运营后台用，还是直接做成用户可用的产品？这决定第一期要不要做前台和计费。
- 图片存哪里：沿用课本诗画的本地文件，还是放 COS（对话归档已在用）？
- 表放在主库，还是单独 schema？建议主库加 `mat_` 前缀，迁移号接现有序列。
- 意象种子 150 个由谁审：需要懂古诗的人，建议按类别分批，每批 30 个。
- 课本诗画是否迁移到素材库：建议第一期只导入、不迁移，`shihua_works` 加 `text_id` 后再逐步合并。
