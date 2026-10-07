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
