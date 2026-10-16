# 对话归档（Conversation archive）— design and API contract

Status: backend and admin UI built on branch `feat/conversation-archive` (2026-10-16), not merged. It ships disabled until the bucket is configured.

## Goal and decisions (agreed with the operator)

- Archive **every user's** API conversations and the images models generate. It covers all users once the admin enables it; there is no per-user opt-in.
- Store in **Tencent Cloud COS, Seoul**, over its S3-compatible API. The server is also in Tencent Cloud Seoul.
- Keep **180 days**.
- **Dedupe by session.** Agent clients (Codex, Claude Code, axonhub…) resend the whole conversation on every request. Each request stores only:
  - the input items that are new since the last archived request of the same session;
  - the full model response.
  
  A whole conversation can be rebuilt from the session's turns in order.
- **Content is stored as sent**, with no secret masking. Because of that:
  - the bucket must be private;
  - objects use server-side encryption (`x-amz-server-side-encryption: AES256`);
  - images are only served through an admin-authenticated backend proxy, never as public links;
  - every admin view is written to the access trail.
- **It must never break, change or noticeably slow a gateway request:**
  - all archive work after the response is asynchronous;
  - memory is bounded;
  - when the queue is full, items are dropped (and counted) rather than blocking.
- The existing 请求内容日志 stays as it is (per-account, 7-day quick view). It shares the access trail.

## What one archived turn contains

**Key and format:** one object per request, gzip JSON. Key: `{prefix}conv/{user_id}/{yyyymmdd}/{session_hash16}/{unix_ms}_{request_id_sanitized}.json.gz`

```json
{
  "v": 1,
  "request_id": "client:…",            // same as usage_logs.request_id
  "user_id": 45, "api_key_id": 35, "group_id": 2,
  "session_key": "…",                   // see "Session key" below
  "created_at": "RFC3339",
  "model": "gpt-6.1-sol", "upstream_model": "…",
  "inbound_endpoint": "/v1/responses", "protocol": "openai_responses|openai_chat|anthropic_messages|gemini|openai_images",
  "user_agent": "…", "ip_address": "…",
  "usage": { "input_tokens": 0, "output_tokens": 0, "cache_read_tokens": 0, "cost_usd": 0 },
  "instructions": "…",                  // system / instructions, only when changed since the session's last archived turn (else omitted)
  "reset": false,                       // true when the history no longer matched the stored prefix (compaction, edit, new client) — new_items then holds the whole input
  "prefix_items": 12,                   // how many input items were already archived (skipped)
  "new_items": [ … raw input items/messages exactly as sent, in order … ],
  "request_params": { "temperature": …, "tools": ["name", …], "tool_choice": …, "reasoning": …, "previous_response_id": … },   // small, no tool schemas
  "response": {
    "status": "completed|incomplete|failed|error|unknown",   // incomplete: a stream that ended without a terminal event; unknown: Gemini (reply not captured)
    "text": "full assistant text (cap 256 KiB)",
    "tool_calls": [ { "name": "…", "call_id": "…", "arguments": "… (cap 64 KiB each, max 200)" } ],
    "items": [ … raw output items, reasoning encrypted_content stripped, cap 1 MiB total … ]
  },
  "images": [ { "key": "…", "mime": "image/png", "bytes": 1234, "source": "response|image_generation_call" }, { "url": "…", "source": "response" } ],
  "truncated": { "request": false, "response": false }
}
```

- The stored object has no image `id`: it is written before its index rows. The admin API takes the ids from `conversation_archive_images`.
- An image the response only linked to (our own image-storage URL) is recorded as `{ url, source }` and is not fetched.
- `response.items` holds the raw output items for Responses (streamed or not) and the content blocks of non-stream Anthropic replies. Anthropic streams and Chat Completions keep `text` + `tool_calls` only. An `image_generation_call` item keeps `"result": ""` plus `"archived_image": n`, the index of the image in this turn.
- `new_items` is capped at 16 MiB per turn. Past that, the newest items that fit are kept and `truncated.request` is true.

**Images** go in separate objects, key `{prefix}img/{user_id}/{yyyymmdd}/{request_id_sanitized}_{n}.{png|jpg|webp}`. Sources:
- `/v1/images/generations` and `/v1/images/edits` responses: `data[].b64_json`. If a response carries our own image-storage URL instead, record the URL in the turn; don't fetch it.
- Responses API `image_generation_call.result` (base64).

Input images in edit requests are not stored. Only the prompt is stored, as the new item.

## Session key and dedupe

**Session key**
- Use `usage_logs.session_id` when the request has one; Codex and Claude Code send one.
- Otherwise use `h:` + sha256(api_key_id + raw JSON of the first input item)[:24]. Chat clients without a session id still share their first message.

**Dedupe state (Redis)**
- **Per thread, not per session.** Claude Code sub-agents and side calls share one session id but each starts with its own first message. So the state is keyed by session and thread: `conv_archive:sess:{user_id}:{sha256(session_key + "\0" + thread)[:24]}`, holding JSON `{instr_hash, item_hashes:[…], offset}`.
  - `thread` is the first 12 hex characters of the first input item's hash, or `chain` for `previous_response_id` requests. It is also stored in the turn object as `thread`. The UI still groups by `session_key`.
  - When a session already has an archived thread (marker `conv_archive:seen:{user_id}:{sha256(session_key)[:24]}`), the first turn of a new thread is marked `reset: true`: it is a fresh context (a sub-agent, a side call, a compaction).
- `item_hashes` keeps at most 4000 entries. Past that, the oldest hashes are dropped and `offset` counts them; the next turn compares from item `offset` on, so long sessions don't reset forever. A history shorter than `offset` is a reset.
- TTL 48h, refreshed on each write.
- Each item hash is sha256 of the item's JSON after a canonical re-encode: whitespace removed, key order kept, and `cache_control` removed at any depth. Claude Code moves the cache breakpoint to the newest message every turn. Only the first 24 hex characters are stored. The archived items keep `cache_control` as sent.
- The state is only written after the turn's object and index rows were stored. After a failed upload, the next turn still includes this turn's new items.
- Workers are sharded by session (or by API key when there is no session id), so one session's turns are deduped in order.

**Each turn**
1. Hash the current input items.
2. Let `k` be the length of the common prefix of the stored hashes with the current hashes from `offset` on.
3. If `k == len(stored)`: `new_items = items[offset+k:]`, `prefix_items = offset + k`.
4. Otherwise the history changed: `reset = true`, `new_items = items`, `prefix_items = 0`.
5. Replace the stored list with the current hashes.

If Redis is unavailable: archive the full input and mark `reset: true`. Never fail the archive because of Redis.

Responses API with `previous_response_id`: the input already holds only the new items, so archive them as-is. The stored item hashes are left unchanged.

Image endpoints are not deduped: each request is its own turn, with the prompt as its only item.

## Capture (request path, synchronous part kept tiny)

- **Gate:** a cached `ArchiveService.Enabled()` check (an atomic bool updated on config save and reloaded every 30s). When off, the request path pays nothing else.
- **Request body:**
  - The gateway handlers already hold the full body as `[]byte`. Where they read it (Responses, Chat Completions, Messages on OpenAI and Claude groups, Gemini v1beta, images JSON), stash a reference with `service.SetArchiveRequestBody(c, body)`. Don't copy it.
  - Multipart image edits: stash only the `prompt` value the handler already parsed (`service.SetArchiveRequestPrompt`). Don't read the body a second time. Grok image edits have no parsed prompt, so the `prompt` field is read from the multipart bytes already in memory, and only while archiving is on.
  - JSON image requests keep the body reference, but only `prompt` and small parameters are archived. Input images are never stored.
- **Response:**
  - Reuse the upstream response observer's content accumulator (`service/content_capture.go`, created in `beginUpstreamResponseModelObservation`).
  - Give it an "archive" mode with the larger caps above, keeping raw output items and image payloads. Create the accumulator when the request is a content-log target **or** archiving is enabled; content log keeps trimming to its own sizes when writing.
  - Buffered paths call `ObserveContentFinalResponse`.
  - Image endpoints: collect `data[].b64_json` from the response body wherever the images handler / service has it.
- **Hand-off:**
  - At usage recording (`recordContentLog` in `OpenAIGatewayService.RecordUsage` and `GatewayService.recordUsageCore`, where 请求内容日志 hooks in), enqueue an archive job built from the `UsageLog` + capture + body reference.
  - A request that never reaches usage recording (upstream error) is not archived; ops logs have it.
- **Queue:**
  - Bounded by count (4096) **and** by bytes (256 MiB counting body + capture).
  - When full: drop, count, and log at most once a minute.
  - 2 workers, each doing: parse → dedupe → gzip → PutObject with SSE (3 retries with backoff) → insert index rows.
- **Not covered (documented):**
  - WebSocket Responses (no traffic on prod).
  - Gemini replies (request only; `ObserveGemini` has no content feed).
  - Embeddings, audio, video.
  - Raw output items of streamed Anthropic replies (text and tool calls are kept).
- **Shutdown:** the archive stops after the usage worker pool drained. Queued jobs get a 5 s drain without retries. Every upload, Redis and DB call of the workers runs under a context cancelled at the end of that drain, so a slow upload can't hold up shutdown. Whatever is left is dropped and counted.
- **Failures:**
  - If the turn object can't be uploaded or indexed, the objects already uploaded for that request (images, the turn) are deleted best-effort.
  - An image that can't be decoded counts in `failed_total` and sets `truncated.response`.

## Index (PostgreSQL, next free migration number)

```sql
conversation_archive_turns(
  id BIGSERIAL PK, user_id BIGINT NOT NULL, api_key_id BIGINT, group_id BIGINT,
  session_key TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL,
  model TEXT NOT NULL DEFAULT '', inbound_endpoint TEXT NOT NULL DEFAULT '', user_agent TEXT NOT NULL DEFAULT '',
  ip_address TEXT NOT NULL DEFAULT '', input_tokens INT NOT NULL DEFAULT 0, output_tokens INT NOT NULL DEFAULT 0,
  cache_read_tokens INT NOT NULL DEFAULT 0, object_key TEXT NOT NULL, bytes INT NOT NULL DEFAULT 0,
  new_item_count INT NOT NULL DEFAULT 0, reset BOOLEAN NOT NULL DEFAULT false, image_count INT NOT NULL DEFAULT 0,
  preview TEXT NOT NULL DEFAULT ''   -- first 200 chars of the latest user text of this turn
)
INDEX (user_id, id DESC), (user_id, session_key, created_at, id), (created_at), (request_id)

conversation_archive_images(
  id BIGSERIAL PK, turn_id BIGINT NOT NULL REFERENCES conversation_archive_turns(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL, object_key TEXT NOT NULL, mime TEXT NOT NULL, bytes INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
)
INDEX (turn_id), (user_id, id DESC)
```

- **Access trail:** extend the `content_log_access_logs.action` CHECK with `'archive_view'` and `'archive_config'`.
- **Retention job** (hourly, batched):
  - Delete index rows older than `retention_days`.
  - Delete their objects with batched DeleteObjects on the keys of the rows being removed, so the bucket matches even if lifecycle isn't set.
  - The admin can also apply a bucket lifecycle rule (expire `{prefix}` after retention_days + 1) as a safety net.
- **Account deletion (self-service):**
  - Before the account goes, the purge is persisted in `conversation_archive_purges(user_id PK, prefixes TEXT[], attempts, last_error, created_at, updated_at)`. It records the prefix in use and every previous prefix kept in the settings (`previous_prefixes`). If that write fails, the deletion fails.
  - `PurgeOwnedContent` deletes the user's turn and image index rows.
  - A purge job (every 10 min, and kicked right after the deletion) deletes `{prefix}conv/{user_id}/` and `{prefix}img/{user_id}/` for each recorded prefix (ListObjectsV2 + DeleteObjects), then removes the row.
  - Failures and a missing storage configuration are retried. A row whose account was never deleted (the deletion failed later) is dropped after a day.
  - Jobs still queued for that user are skipped for 24 h.
  - Admin-initiated user deletion intentionally keeps the archive.
- **Retention order:** the objects are deleted first, then the rows. If DeleteObjects fails, the rows stay and the job retries the next hour.

## Settings

- Settings key `conversation_archive_config` (JSON). The secret is encrypted with the existing `SecretEncryptor`, as `ImageStorageSettings` does.
- Fields: `{ enabled, endpoint, region, bucket, prefix (default "archive/"), access_key_id, secret_access_key (write-only), force_path_style (false for COS), retention_days (default 180, 7..730), sse (default true) }`.
- COS S3 values for Seoul: endpoint `https://cos.ap-seoul.myqcloud.com`, region `ap-seoul`, bucket `name-appid`.
- Build the client with `repository/s3_client.go` `newS3Client`.

## Admin API (all under `/api/v1/admin/conversation-archive`, admin auth; writes need step-up auth like `/admin/backup/image-storage`)

Every endpoint uses the usual `{ code, message, data }` wrapper, except `GET /images/:id`, which returns the raw image bytes.

**Config**
- `GET /config` → `{ enabled, endpoint, region, bucket, prefix, access_key_id, secret_configured, force_path_style, retention_days, sse, status: { active, queue_len, queue_bytes, dropped_total, failed_total, uploaded_today, last_upload_at|null, last_error, last_error_at|null, lifecycle_applied, lifecycle_applied_at|null, lifecycle_error, pending_purges } }`
  - `active` is true when new requests are being archived: enabled, and a storage client was built.
  - `failed_total` counts turns or images whose upload or index write failed after retries, and images that couldn't be decoded.
  - The `lifecycle_*` fields report the last lifecycle-rule attempt by this instance, in memory only: after a restart they are empty until the next save or apply.
  - `pending_purges` is the number of account purges seen by the last purge run.
- `PUT /config` (stepUpAuth): body = the same fields plus optional `secret_access_key`.
  - A missing or empty secret keeps the stored one. A missing `sse` means on.
  - Enabling needs bucket, access key ID and a secret. A new secret needs a fixed `TOTP_ENCRYPTION_KEY`, as for image storage.
  - Returns the same as GET. Audited as `archive_config`, detail = JSON of the changed non-secret fields (`"secret_access_key":"changed"` when the secret changed). The settings row and its audit row commit together.
  - Saving while enabled also applies the lifecycle rule. A failure doesn't undo the save: see `status.lifecycle_applied` / `lifecycle_error` in the response.
  - A changed prefix is appended to `previous_prefixes` (internal, at most 20), so account deletion sweeps the old prefixes too.
- `POST /config/test` (stepUpAuth: it signs requests with the stored key against an admin-supplied endpoint) → `{ ok: true }` or a Chinese error. Runs PutObject + GetObject + DeleteObjects of `{prefix}_healthcheck/{ts}.txt` with SSE.
  - The body is the current (possibly unsaved) form. Without a secret it uses the stored secret; an empty body tests the saved configuration.
- `POST /config/lifecycle` (stepUpAuth) → adds the bucket lifecycle rule `sub2api-conversation-archive` (expire `{prefix}` after retention_days + 1). The bucket's other rules are kept. The `archive_config` audit row is written first; without it nothing is applied. Returns `{ ok: true }` or a Chinese error with the storage error text.

**Browse**
- `GET /sessions?user_id=&q=&from=&to=&before=&limit=30`
  - `user_id` optional; `q` matches user id, email or username (exact, case-insensitive), resolved to a user id first. A `q` that matches nobody returns an empty list.
  - `from` / `to` / `before` are RFC3339 (a plain `YYYY-MM-DD` is read as local midnight). `to` is exclusive. Without a user, `from` defaults to 30 days ago. `limit` max 100.
  - Returns `{ items: [ { session_key, user_id, email, first_at, last_at, turns, models: ["…"], endpoints: ["…"], user_agent, preview, image_count, input_tokens, output_tokens } ], next_before: "RFC3339|null" }`, ordered by `last_at` desc.
  - The range selects sessions by their turns; the counts, tokens and `first_at` cover the whole session. `user_agent` is the latest turn's. `preview` is the session's first non-empty preview.
- `GET /sessions/turns?user_id=45&session_key=…&before_id=&limit=20`
  - Index rows newest first by `(created_at, id)` (the UI reverses them for reading), `limit` default 20, max 50.
  - `before_id` is a cursor. If that row is in the session, the page starts strictly before it. Otherwise (the UI's `id + 1` after a jump) it starts at the session's row with the largest smaller id, inclusive.
  - A page also stops early once its objects add up to 64 MiB compressed (index `bytes`); `next_before_id` is then set. A turn object larger than 32 MiB decoded is reported as `fetch_error`.
  - Each row has its object fetched from COS and decoded: `{ items: [ { id, user_id, session_key, thread, protocol, created_at, request_id, model, inbound_endpoint, user_agent, ip_address, input_tokens, output_tokens, cache_read_tokens, reset, prefix_items, instructions?, new_items: [...], request_params, response: {...}, images: [ { id, mime, bytes } ], image_urls?, truncated, fetch_error? } ], next_before_id }`.
  - Objects are fetched in parallel, at most 8 at a time. A failed fetch sets `fetch_error` on that item and doesn't fail the page.
- `GET /turns/by-request?request_id=` → the same item shape for one turn, to jump from a usage row. It also has `email`. The UI opens the session with `user_id`, `session_key` and `before_id = id + 1`. 404 when the request wasn't archived.
- `GET /images/:id` → streams the raw image bytes from COS with its content-type (anything not `image/*` is sent as `application/octet-stream`), `Cache-Control: private, max-age=300` and `X-Content-Type-Options: nosniff`.
- `GET /stats` → `{ days: [ { day, turns, users, bytes, images } ] }` for the last 30 days, from the index. Not audited.

**Audit:** every browse call except `/images/:id` and `/stats` writes one `archive_view` access row. The image proxy is covered by the turns view.

## Admin UI

New admin page **对话归档** at `/admin/conversation-archive`, in the admin sidebar next to 用户画像.

**Settings card**
- Status chip: 未配置 / 已启用 / 已暂停 / 上传出错.
- The fields above, with a COS hint: Endpoint `https://cos.ap-seoul.myqcloud.com`, Region `ap-seoul`, bucket name format `名称-APPID`.
- Buttons 测试连接 / 应用生命周期规则 / 保存.
- A red notice: 「开启后保存所有用户的对话原文（含密钥、密码）和生成的图片 N 天。请确认隐私政策已发布相应条款，桶必须是私有的。」
- The status block: queue, dropped, today's uploads, last error.
- Collapsed once configured.

**Browser**
- User search (id or email), then a session list: last time, client, model, turns, preview, images.
- Opening a session shows a chat-style transcript, oldest first, loading earlier turns on scroll or with 「加载更早」. It shows:
  - user messages and assistant text (pre-wrap);
  - tool calls (collapsed, name + arguments) and tool outputs (`function_call_output` / `tool_result`, collapsed);
  - images as thumbnails through `/images/:id`, fetched with the admin auth header and shown as object URLs (never put the token in a URL);
  - a divider 「上下文已重置」 on reset turns;
  - instructions as a collapsed 「系统提示」 block when present;
  - a small meta line per turn: time, model, tokens, request id.

**Access trail:** shared with 内容日志; it shows `archive_view` / `archive_config` actions labelled 查看归档 / 归档设置.
