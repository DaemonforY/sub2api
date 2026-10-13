/** Admin API for 内容运营 (the content engine): topics, AI-written platform versions, review, stats. */
import { apiClient } from '../client'

export type ContentPlatform = 'wechat' | 'zhihu' | 'csdn' | 'toutiao' | 'xiaohongshu'

export interface PlatformSpec {
  id: ContentPlatform
  name: string
  source: string
  title_max: number
  body_min: number
  body_max: number
  links: boolean
  read_more: boolean
  promo: boolean
  tags_min: number
  tags_max: number
  cards: boolean
  micro_max: number
  ai_label: string
  publish_url: string
}

export interface ContentOverview {
  pending: number
  running: number
  topics: number
  published: number
  platforms: PlatformSpec[]
}

export type NotifyType = '' | 'serverchan' | 'pushplus' | 'wecom'

export interface ContentSettings {
  enabled: boolean
  key_id: number
  model: string
  daily_time: string
  daily_count: number
  platforms: ContentPlatform[]
  search: boolean
  style_prompt: string
  product: string
  promo_bonus: number
  notify_type: NotifyType
  /** Write-only; empty keeps the stored key. */
  notify_key?: string
  notify_key_set: boolean
  key_name?: string
  key_problem?: string
  search_available: boolean
  last_run_day?: string
}

export interface KeyOption {
  id: number
  name: string
  group: string
}

export type TopicStatus = 'pending' | 'used' | 'dismissed'

export interface ContentTopic {
  id: number
  title: string
  angle: string
  notes: string
  source: string
  reason: string
  score: number
  status: TopicStatus
  item_id?: number
  created_at: string
}

export interface TopicInput {
  title: string
  angle: string
  notes: string
  score?: number
}

export type ItemStatus = 'outlining' | 'drafting' | 'reviewing' | 'humanizing' | 'adapting' | 'pending_review' | 'done' | 'failed' | 'rejected' | 'canceled'
export type VersionStatus = 'pending_review' | 'generating' | 'approved' | 'published' | 'skipped' | 'failed'

export interface ContentCheck {
  level: 'error' | 'warn' | 'info'
  code: string
  text: string
}

export interface ContentEvent {
  at: string
  kind: string
  text: string
}

export interface ContentSource {
  title: string
  url: string
  snippet?: string
}

export interface ContentOutline {
  title: string
  angle: string
  reader: string
  hook: string
  sections: { heading: string; points: string[] }[]
  materials_needed: string[]
  landing_path: string
}

export interface ContentItem {
  id: number
  topic_id?: number
  key_id: number
  status: ItemStatus
  title: string
  error: string
  created_at: string
  updated_at: string
  brief: { topic: string; angle?: string; notes?: string; source?: string; platforms: ContentPlatform[]; search: boolean }
  landing_path?: string
  sources: ContentSource[]
  outline?: ContentOutline
  draft?: string
  review_notes?: string[]
  ai_score?: number
  markdown?: string
  checks?: ContentCheck[]
  reject_reason?: string
  events: ContentEvent[]
  model?: string
  prompt_tokens: number
  completion_tokens: number
  variants?: { platform: ContentPlatform; status: VersionStatus }[]
  running: boolean
}

export interface NoteCard {
  title: string
  points: string[]
}

export interface ContentVersion {
  id: number
  item_id: number
  platform: ContentPlatform
  kind: 'article' | 'answer' | 'note'
  status: VersionStatus
  title: string
  body: string
  link_code: string
  promo_code: string
  post_url: string
  published_at?: string
  question?: string
  digest?: string
  micro?: string
  category?: string
  tags?: string[]
  cards?: NoteCard[]
  checks?: ContentCheck[]
  edited?: boolean
  feedback?: string
  error?: string
  rendered?: string
  rendered_micro?: string
  link_url?: string
  item_title?: string
}

export interface ContentItemDetail extends ContentItem {
  versions: ContentVersion[]
}

export interface VersionInput {
  title: string
  body: string
  question: string
  digest: string
  micro: string
  category: string
  tags: string[]
  cards: NoteCard[]
}

export interface ContentRule {
  id: number
  rule: string
  origin: 'manual' | 'reject'
  enabled: boolean
  created_at: string
}

export interface StatsRow extends ContentVersion {
  clicks: number
  visitors: number
  signups: number
  activated: number
  paid_users: number
  revenue: number
  promo_uses: number
}

export interface PlatformTotal {
  platform: ContentPlatform
  posts: number
  clicks: number
  signups: number
  paid_users: number
  revenue: number
  promo_uses: number
}

export interface ContentStats {
  days: number
  rows: StatsRow[]
  platforms: PlatformTotal[]
}

// Model calls (suggesting topics, 立即运行) take up to a few minutes.
const SLOW = { timeout: 240000 }

export async function getOverview(): Promise<ContentOverview> {
  return (await apiClient.get('/admin/content/overview')).data
}
export async function getSettings(): Promise<ContentSettings> {
  return (await apiClient.get('/admin/content/settings')).data
}
export async function saveSettings(s: ContentSettings): Promise<ContentSettings> {
  return (await apiClient.put('/admin/content/settings', s)).data
}
export async function listKeys(): Promise<KeyOption[]> {
  return (await apiClient.get('/admin/content/keys')).data
}
export async function testNotify(): Promise<void> {
  await apiClient.post('/admin/content/notify/test')
}
export async function runNow(): Promise<void> {
  await apiClient.post('/admin/content/run', {}, SLOW)
}

export async function listTopics(status: TopicStatus | 'all' = 'pending'): Promise<ContentTopic[]> {
  return (await apiClient.get('/admin/content/topics', { params: { status } })).data
}
export async function createTopic(input: TopicInput): Promise<ContentTopic> {
  return (await apiClient.post('/admin/content/topics', input)).data
}
export async function suggestTopics(): Promise<ContentTopic[]> {
  return (await apiClient.post('/admin/content/topics/suggest', {}, SLOW)).data
}
export async function updateTopic(id: number, input: TopicInput): Promise<ContentTopic> {
  return (await apiClient.put(`/admin/content/topics/${id}`, input)).data
}
export async function setTopicStatus(id: number, status: 'pending' | 'dismissed'): Promise<ContentTopic> {
  return (await apiClient.post(`/admin/content/topics/${id}/status`, { status })).data
}
export async function startTopic(id: number): Promise<ContentItem> {
  return (await apiClient.post(`/admin/content/topics/${id}/start`)).data
}
export async function deleteTopic(id: number): Promise<void> {
  await apiClient.delete(`/admin/content/topics/${id}`)
}

export async function listItems(status = ''): Promise<ContentItem[]> {
  return (await apiClient.get('/admin/content/items', { params: { status } })).data
}
export async function getItem(id: number): Promise<ContentItemDetail> {
  return (await apiClient.get(`/admin/content/items/${id}`)).data
}
export async function retryItem(id: number): Promise<ContentItem> {
  return (await apiClient.post(`/admin/content/items/${id}/retry`)).data
}
export async function cancelItem(id: number): Promise<void> {
  await apiClient.post(`/admin/content/items/${id}/cancel`)
}
export async function rejectItem(id: number, reason: string, asRule: boolean): Promise<ContentItem> {
  return (await apiClient.post(`/admin/content/items/${id}/reject`, { reason, as_rule: asRule })).data
}
export async function deleteItem(id: number): Promise<void> {
  await apiClient.delete(`/admin/content/items/${id}`)
}

export async function updateVersion(id: number, input: VersionInput): Promise<ContentVersion> {
  return (await apiClient.put(`/admin/content/versions/${id}`, input)).data
}
export async function approveVersion(id: number): Promise<ContentVersion> {
  return (await apiClient.post(`/admin/content/versions/${id}/approve`)).data
}
export async function skipVersion(id: number): Promise<ContentVersion> {
  return (await apiClient.post(`/admin/content/versions/${id}/skip`)).data
}
export async function publishVersion(id: number, postUrl: string): Promise<ContentVersion> {
  return (await apiClient.post(`/admin/content/versions/${id}/publish`, { post_url: postUrl })).data
}
export async function regenerateVersion(id: number, feedback: string): Promise<ContentVersion> {
  return (await apiClient.post(`/admin/content/versions/${id}/regenerate`, { feedback })).data
}

export async function listRules(): Promise<ContentRule[]> {
  return (await apiClient.get('/admin/content/rules')).data
}
export async function createRule(rule: string): Promise<ContentRule> {
  return (await apiClient.post('/admin/content/rules', { rule })).data
}
export async function setRuleEnabled(id: number, enabled: boolean): Promise<void> {
  await apiClient.put(`/admin/content/rules/${id}`, { enabled })
}
export async function deleteRule(id: number): Promise<void> {
  await apiClient.delete(`/admin/content/rules/${id}`)
}

export async function getStats(days = 30): Promise<ContentStats> {
  return (await apiClient.get('/admin/content/stats', { params: { days } })).data
}
