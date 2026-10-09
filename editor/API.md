# 公众号排版工具 · 后端接口（/api/v1/editor）

排版本身纯前端、免登录。下面这些接口都要登录：`Authorization: Bearer <auth_token>`（和学习站一样，token 在同域的
`localStorage.auth_token`；未登录跳 `/login?redirect=%2Feditor%2F`）。

返回格式和主站一致：成功 `{"code":0,"message":"success","data":...}`；失败 HTTP 4xx/5xx +
`{"code":<status>,"message":"中文（English）","reason":"EDITOR_..."}`。

公众号的 AppID / AppSecret 只存在用户浏览器（`localStorage`），每次请求随 body 发来；服务器只用来换 access_token
（内存缓存，按 AppID + Secret 的哈希，不落库、不记日志）。调用微信接口的服务器出口 IP 必须在公众号
「设置与开发 → 基本配置 → IP 白名单」里；不在时接口返回 reason `EDITOR_WECHAT_IP`，message 里带上微信报告的 IP。

## 公众号

### POST /wechat/check
`{"appid":"wx...","secret":"..."}` → `{"ok":true}`。用于设置页「测试连接」。

### POST /wechat/upload（multipart/form-data）
字段：`appid`、`secret`、`kind`（`content` 正文图片 / `cover` 封面）、`file`。
- `content`：调用 `media/uploadimg`，只支持 jpg / png，**小于 1MB**（前端先压缩）→ `{"url":"https://mmbiz.qpic.cn/..."}`
- `cover`：调用 `material/add_material?type=image`（永久素材，≤ 10MB，jpg/png/gif/bmp）→ `{"media_id":"...","url":"..."}`

### POST /wechat/draft
```json
{"appid":"wx...","secret":"...","article":{
  "title":"标题","author":"作者","digest":"摘要（可空）",
  "content":"<section style=...>已内联样式的正文 HTML，图片都已换成 mmbiz 地址</section>",
  "content_source_url":"原文链接（可空）","thumb_media_id":"封面 media_id（必填）",
  "need_open_comment":false,"only_fans_can_comment":false}}
```
→ `{"media_id":"草稿的 media_id"}`。草稿在公众号后台「内容与互动 → 草稿箱」里。

## 导入公众号文章

### POST /article/import
`{"url":"https://mp.weixin.qq.com/s/..."}`（只接受 mp.weixin.qq.com 的文章链接）
→ `{"title":"...","author":"...","html":"正文 HTML（图片的 data-src 已换成 src，仍是 mmbiz 原图地址）","images":["https://mmbiz.qpic.cn/..."]}`

### GET /image?url=<编码后的地址>
只代理 `mmbiz.qpic.cn`、`mmbiz.qlogo.cn` 的图片（公众号图片有防盗链），返回图片本身（≤ 10MB）。前端用
`fetch` 带上 Authorization 取 Blob，存进 IndexedDB 换成 `img://` 链接。

## AI（用用户自己的 Key 计费）

Key 列表用学习站已有的 `GET /api/v1/learn/keys`（只列 GPT 分组的 Key：`[{id,name,group}]`）。

### POST /ai/text（SSE 流式）
`{"key_id":7,"action":"polish","text":"要处理的 Markdown","instruction":"自定义要求（action=custom 时必填）","title":"文章标题（可选，作上下文）"}`

action：`polish` 润色 · `shorten` 精简 · `expand` 扩写 · `title` 拟 5 个标题 · `digest` 写摘要（≤ 120 字）·
`outline` 列提纲 · `image_prompt` 根据内容写一段配图描述 · `custom` 按要求修改。text ≤ 20000 字。

返回 `text/event-stream`：`data: {"delta":"..."}` 若干条，最后 `data: {"done":true}`；出错 `data: {"error":"中文说明"}`。
输出是 Markdown（title 为每行一个标题；digest 为纯文本）。

### POST /ai/image
`{"key_id":7,"prompt":"画面描述","size":"1536x1024"}`（size：`1536x1024` 横图 / `1024x1536` 竖图 / `1024x1024`）
→ `{"b64_json":"...","mime":"image/png"}`。模型 gpt-image-2，费用记在所选 Key 上。

## AI 写文章（后台任务，用用户自己的 Key 计费）

服务器在后台搜资料（后台「联网搜索模拟」里配了 Tavily / Brave 时）、出大纲，用户确认后写全文、画封面和配图；关掉页面也继续。
推送到草稿箱仍在浏览器里做（AppSecret 不经过这些接口）：把文章载入编辑器，再走上面的 `/wechat/upload` + `/wechat/draft`。
每人同时最多 2 篇在生成，每次运行最长 15 分钟，文章和图片保存 30 天。模型用学习站设置的模型，图片 gpt-image-2。

状态：`outlining` 出大纲中 → `outline_ready` 待确认 → `writing` 写作中 → `drawing` 画图中 → `done`；出错 `failed`、停止 `canceled`。

### GET /articles/config?key_id=7
→ `{"search":true,"max_images":4,"default_images":1,"image_price":0.201}`（search：联网搜索是否可用；
image_price：在这个 Key 上画一张 1536×1024 图的价格（美元，和实际扣费同一套算法：分组图片价、用户分组倍率、独立图片倍率），
算不出来时没有这个字段）

### GET /articles
→ 最近 30 篇，不含正文、进度和图片：`[{id,status,title,error,created_at,updated_at,brief,pushed_at}]`

### POST /articles
`{"key_id":7,"topic":"主题（≤200 字）","materials":"参考资料（≤8000 字，可空）","audience":"读者","tone":"风格","length":"short|standard|long","images":2,"search":true}`
→ 文章（status=outlining）。images 是正文配图数（0–4），另外总有一张封面。

### GET /articles/:id
→ `{id,key_id,status,title,error,created_at,updated_at,brief,sources:[{title,url,snippet}],outline,markdown,images:[{n,kind,prompt,alt,status,error}],events:[{at,kind,text}],model,prompt_tokens,completion_tokens,images_drawn,searches,pushed_at}`

outline：`{titles:[3 个备选],title,digest,cover_prompt,sections:[{heading,points:[...],image?:{prompt,alt}}]}`。
markdown 里配图写成 `![说明](img:N)`，N 从 1 开始，对应 images 里 kind=body 的 n；n=0 是封面。

### POST /articles/:id/outline
只在 `outline_ready` 时可用。`{"outline":{...可选，用户改过的大纲},"feedback":"修改意见"}` 按意见重写大纲；
`{"outline":{...},"confirm":true}` 确认并开始写全文。

### POST /articles/:id/retry
`failed` / `canceled` 从中断处继续；`done` 但有图没画成时只补画失败的图。

### POST /articles/:id/cancel · POST /articles/:id/pushed · DELETE /articles/:id
停止运行 / 记录已推送到草稿箱 / 删除文章和图片（运行中不能删）。

### GET /articles/:id/images/:n
图片本身（JPEG，正文图 ≤ 900KB）。前端带 Authorization 取 Blob，存进 IndexedDB 换成 `img://` 链接。
