/**
 * Admin API for 对话归档 (conversation archive), see docs/CONVERSATION_ARCHIVE.md.
 *
 * Every browse call (sessions, sessions/turns, turns/by-request) writes an
 * `archive_view` row to the access trail server-side, so callers must only fetch
 * them on an explicit admin action — no polling, no prefetch. The image wall
 * list (`/images`) is audited too; the image bytes (`/images/:id`, `/images/:id/thumb`)
 * and `/stats` are not. PUT /config, POST /config/test and POST /config/lifecycle need step-up
 * auth, and so does POST /config/test: wrap them in useStepUp().run().
 */
import { apiClient } from '../client'

const BASE = '/admin/conversation-archive'

// ---------- Config ----------

export interface ArchiveStatus {
  queue_len: number
  queue_bytes: number
  dropped_total: number
  uploaded_today: number
  last_upload_at: string | null
  last_error: string
  last_error_at: string | null
  /** uploads that failed after all retries */
  failed_total?: number
  /** whether the archive workers are running on this instance */
  active?: boolean
  /** result of applying the bucket lifecycle rule after an enabled save */
  lifecycle_applied?: boolean
  lifecycle_error?: string
}

export interface ArchiveConfigFields {
  enabled: boolean
  endpoint: string
  region: string
  bucket: string
  prefix: string
  access_key_id: string
  force_path_style: boolean
  /** 7..730 */
  retention_days: number
  sse: boolean
}

export interface ArchiveConfig extends ArchiveConfigFields {
  secret_configured: boolean
  status: ArchiveStatus
}

/** PUT body: an empty / missing secret_access_key keeps the stored one. */
export interface ArchiveConfigUpdate extends ArchiveConfigFields {
  secret_access_key?: string
}

export interface ArchiveOk {
  ok: boolean
  message?: string
}

// ---------- Browse ----------

export interface ArchiveSession {
  session_key: string
  user_id: number
  email: string
  first_at: string
  last_at: string
  turns: number
  models: string[]
  endpoints: string[]
  user_agent: string
  preview: string
  image_count: number
  input_tokens: number
  output_tokens: number
}

export interface ArchiveSessionPage {
  items: ArchiveSession[]
  /** RFC3339 cursor for the next (older) page, null at the end */
  next_before: string | null
}

export interface ArchiveSessionQuery {
  user_id?: number
  /** a user id, email or username; resolved to a user id server-side */
  q?: string
  /** RFC3339 */
  from?: string
  /** RFC3339 */
  to?: string
  before?: string
  limit?: number
}

export interface ArchiveToolCall {
  name: string
  call_id?: string
  arguments?: string
}

export interface ArchiveResponse {
  status?: 'completed' | 'incomplete' | 'failed' | 'error' | string
  text?: string
  tool_calls?: ArchiveToolCall[] | null
  /** raw output items (reasoning encrypted_content stripped) */
  items?: unknown[] | null
}

export interface ArchiveImageRef {
  id: number
  mime: string
  bytes: number
}

/** An image the response returned as a link (e.g. our image-storage URL), not stored by the archive. */
export interface ArchiveImageUrl {
  url: string
  source?: string
}

export interface ArchiveTurn {
  id: number
  created_at: string
  request_id: string
  model: string
  inbound_endpoint: string
  user_agent: string
  ip_address: string
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  reset: boolean
  prefix_items: number
  /** present only when it changed since the session's previous archived turn */
  instructions?: string
  /** openai_responses | openai_chat | anthropic_messages | gemini | openai_images (when the backend passes it on) */
  protocol?: string
  new_items: unknown[] | null
  request_params?: Record<string, unknown> | null
  response?: ArchiveResponse | null
  images?: ArchiveImageRef[] | null
  image_urls?: ArchiveImageUrl[] | null
  /** set when the object couldn't be read from storage; the rest of the page is fine */
  fetch_error?: string
}

export interface ArchiveTurnPage {
  /** newest first */
  items: ArchiveTurn[]
  next_before_id: number | null
}

export interface ArchiveTurnQuery {
  user_id: number
  session_key: string
  before_id?: number
  limit?: number
}

/** by-request returns one turn plus where it belongs, so the UI can open the session. */
export interface ArchiveTurnLookup extends ArchiveTurn {
  user_id?: number
  session_key?: string
  email?: string
}

/** One tile of the image wall (图片墙), with the turn it came from. */
export interface ArchiveWallImage {
  id: number
  turn_id: number
  user_id: number
  email: string
  mime: string
  bytes: number
  created_at: string
  model: string
  inbound_endpoint: string
  session_key: string
  request_id: string
  /** head of the turn's latest user prompt (200 characters) */
  prompt: string
}

export interface ArchiveWallPage {
  /** newest first */
  items: ArchiveWallImage[]
  next_before_id: number | null
  /** models that drew images lately; first page only */
  models?: string[] | null
}

export interface ArchiveWallQuery {
  /** a user id, email or username */
  q?: string
  model?: string
  /** RFC3339 */
  from?: string
  /** RFC3339 */
  to?: string
  before_id?: number
  limit?: number
}

export interface ArchiveStatsDay {
  day: string
  turns: number
  users: number
  bytes: number
  images: number
}

// ---------- Calls ----------

export async function getConfig(): Promise<ArchiveConfig> {
  const { data } = await apiClient.get(`${BASE}/config`)
  return data
}

/** Step-up protected. */
export async function updateConfig(cfg: ArchiveConfigUpdate): Promise<ArchiveConfig> {
  const { data } = await apiClient.put(`${BASE}/config`, withoutEmptySecret(cfg))
  return data
}

function withoutEmptySecret(cfg: ArchiveConfigUpdate): ArchiveConfigUpdate {
  const body: ArchiveConfigUpdate = { ...cfg }
  if (!body.secret_access_key) delete body.secret_access_key
  return body
}

/**
 * Step-up protected.
 * PutObject + GetObject + DeleteObject of `{prefix}_healthcheck/{ts}.txt` with SSE.
 * The form is sent along (empty secret = the stored one) so a config can be
 * tested before it is saved; a backend that only tests the saved config ignores it.
 */
export async function testConnection(cfg?: ArchiveConfigUpdate): Promise<ArchiveOk> {
  const { data } = await apiClient.post(`${BASE}/config/test`, cfg ? withoutEmptySecret(cfg) : undefined)
  return data
}

/** Step-up protected. Applies the bucket lifecycle rule for the prefix. */
export async function applyLifecycle(): Promise<ArchiveOk> {
  const { data } = await apiClient.post(`${BASE}/config/lifecycle`)
  return data
}

/** Audited (archive_view). */
export async function listSessions(q: ArchiveSessionQuery): Promise<ArchiveSessionPage> {
  const params: Record<string, string | number> = { limit: q.limit ?? 30 }
  if (q.user_id) params.user_id = q.user_id
  if (q.q) params.q = q.q
  if (q.from) params.from = q.from
  if (q.to) params.to = q.to
  if (q.before) params.before = q.before
  const { data } = await apiClient.get(`${BASE}/sessions`, { params })
  return data
}

/** Audited (archive_view). Newest first. */
export async function listTurns(q: ArchiveTurnQuery): Promise<ArchiveTurnPage> {
  const params: Record<string, string | number> = { user_id: q.user_id, session_key: q.session_key, limit: q.limit ?? 20 }
  if (q.before_id) params.before_id = q.before_id
  const { data } = await apiClient.get(`${BASE}/sessions/turns`, { params })
  return data
}

/** Audited (archive_view). */
export async function getTurnByRequest(requestId: string): Promise<ArchiveTurnLookup> {
  const { data } = await apiClient.get(`${BASE}/turns/by-request`, { params: { request_id: requestId } })
  return data
}

/** The image bytes, fetched with the admin auth header (never a token in a URL). */
export async function getImage(id: number): Promise<Blob> {
  const { data } = await apiClient.get(`${BASE}/images/${id}`, { responseType: 'blob' })
  return data
}

/** Audited (archive_view). The image wall, newest first. */
export async function listImages(q: ArchiveWallQuery): Promise<ArchiveWallPage> {
  const params: Record<string, string | number> = { limit: q.limit ?? 60 }
  if (q.q) params.q = q.q
  if (q.model) params.model = q.model
  if (q.from) params.from = q.from
  if (q.to) params.to = q.to
  if (q.before_id) params.before_id = q.before_id
  const { data } = await apiClient.get(`${BASE}/images`, { params })
  return data
}

/** A small JPEG of the image for the wall (made once server-side, kept in the bucket). Not audited. */
export async function getImageThumb(id: number): Promise<Blob> {
  const { data } = await apiClient.get(`${BASE}/images/${id}/thumb`, { responseType: 'blob' })
  return data
}

/** Last 30 days from the index. Not audited. */
export async function getStats(): Promise<{ days: ArchiveStatsDay[] }> {
  const { data } = await apiClient.get(`${BASE}/stats`)
  return data
}

export const conversationArchiveAPI = {
  getConfig,
  updateConfig,
  testConnection,
  applyLifecycle,
  listSessions,
  listTurns,
  getTurnByRequest,
  getImage,
  listImages,
  getImageThumb,
  getStats
}
export default conversationArchiveAPI
