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
  desk: number
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

export type ItemStatus = 'desk' | 'outlining' | 'drafting' | 'reviewing' | 'humanizing' | 'adapting' | 'pending_review' | 'done' | 'failed' | 'rejected' | 'canceled'
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
  desk?: ContentDesk
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

// --- 写作台 ------------------------------------------------------------------------------------

export interface PersonaSample {
  id: string
  title: string
  url?: string
  text: string
  added_at: string
}

/** The owner's 作者档案: every 写作台 prompt writes as this person. */
export interface Persona {
  name: string
  identity: string
  experiences: string[]
  stances: string[]
  likes: string[]
  dislikes: string[]
  voice: string
  phrases: string[]
  taboos: string[]
  samples: PersonaSample[]
  updated_at?: string
}

export type DeskStage = 'materials' | 'angles' | 'drafts' | 'edit' | 'final'
export type FactStatus = 'new' | 'ok' | 'doubt' | 'rejected'

export interface Fact {
  id: string
  text: string
  kind: 'user' | 'persona' | 'answer' | 'web'
  source?: string
  source_title?: string
  status: FactStatus
}

export interface Question {
  id: string
  question: string
  why?: string
  answer?: string
}

export interface Angle {
  id: string
  title: string
  angle: string
  archetype: string
  reader: string
  hook: string
  h: number
  k: number
  r: number
  why: string
  note?: string
}

export interface Skeleton {
  source: string
  opening: string
  parts: { role: string; how: string }[]
  emotion: string
  escalation: string
  callbacks: string
  title_pattern: string
}

export interface ParaMark {
  i: number
  v: 'read' | 'skim' | 'leave'
  why?: string
}

export interface DraftReview {
  paras: ParaMark[]
  hook: number
  flow: number
  voice: number
  substance: number
  best_line: string
  problems: string[]
  summary: string
}

export interface Draft {
  id: string
  structure: string
  label: string
  markdown: string
  review?: DraftReview
  error?: string
}

export interface Highlight {
  id: string
  draft_id: string
  quote: string
  why: string
}

export interface Issue {
  id: string
  para?: number
  layer: 'rule' | 'voice' | 'flow' | 'content' | 'fact'
  level: 'must' | 'should'
  quote: string
  problem: string
  fix?: string
}

export interface ProseStats {
  chars: number
  paras: number
  single_paras: number
  single_ratio: number
  headings: number
  bullets: number
  dashes: number
  colons: number
  placeholders: number
}

export interface CheckReport {
  at: string
  stats: ProseStats
  issues: Issue[]
  scores?: Record<string, number>
  summary?: string
}

export interface Comment {
  id: string
  quote: string
  text: string
  at: string
  resolved?: boolean
}

export interface Learn {
  id: string
  rule: string
  evidence: string
  status: 'new' | 'added' | 'dismissed'
}

export interface DeskJob {
  kind: 'research' | 'angles' | 'reference' | 'drafts' | 'merge' | 'check' | 'learn' | 'platforms'
  status: 'running' | 'done' | 'failed'
  error?: string
  started_at: string
  finished_at?: string
}

export interface ContentDesk {
  stage: DeskStage
  job?: DeskJob
  research_hint?: string
  facts: Fact[]
  questions: Question[]
  angles: Angle[]
  angle?: Angle
  skeleton?: Skeleton
  drafts: Draft[]
  judge?: { ranking: string[]; reasons: string[]; highlights: Highlight[] }
  base_id?: string
  picked?: string[]
  body: string
  body_ai?: string
  check?: CheckReport
  comments: Comment[]
  learn: Learn[]
  final_at?: string
}

export interface TextVersion {
  id: number
  item_id: number
  kind: 'draft' | 'merge' | 'manual' | 'ai' | 'restore' | 'final'
  label: string
  markdown?: string
  chars: number
  created_at: string
}

export type AssistAction = 'voice' | 'tighten' | 'expand' | 'sharper' | 'lighter' | 'plain' | 'custom' | 'analogy' | 'evidence'

export interface AssistResult {
  text?: string
  note?: string
  options?: { text: string; note?: string; source?: string; source_title?: string }[]
}

export async function getPersona(): Promise<Persona> {
  return (await apiClient.get('/admin/content/persona')).data
}
export async function savePersona(p: Persona): Promise<Persona> {
  return (await apiClient.put('/admin/content/persona', p)).data
}
export async function addSample(input: { url?: string; title?: string; text?: string }): Promise<Persona> {
  return (await apiClient.post('/admin/content/persona/samples', input, { timeout: 60000 })).data
}
export async function deleteSample(id: string): Promise<Persona> {
  return (await apiClient.delete(`/admin/content/persona/samples/${id}`)).data
}
export async function distillPersona(): Promise<Persona> {
  return (await apiClient.post('/admin/content/persona/distill', {}, SLOW)).data
}

export async function createDesk(input: { topic_id?: number; title?: string; idea?: string; notes?: string }): Promise<ContentItem> {
  return (await apiClient.post('/admin/content/desk', input)).data
}

const desk = (id: number, path: string) => `/admin/content/items/${id}/desk/${path}`

export async function research(id: number, hint = ''): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'research'), { hint })).data
}
export async function saveMaterials(id: number, facts: Fact[], answers: Record<string, string>): Promise<ContentItem> {
  return (await apiClient.put(desk(id, 'materials'), { facts, answers })).data
}
export async function makeAngles(id: number): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'angles'))).data
}
export async function chooseAngle(id: number, angle: Angle): Promise<ContentItem> {
  return (await apiClient.put(desk(id, 'angle'), angle)).data
}
export async function addReference(id: number, input: { url?: string; text?: string }): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'reference'), input, { timeout: 60000 })).data
}
export async function clearReference(id: number): Promise<ContentItem> {
  return (await apiClient.delete(desk(id, 'reference'))).data
}
export async function makeDrafts(id: number): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'drafts'))).data
}
export async function chooseBase(id: number, draftId: string, highlights: string[]): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'base'), { draft_id: draftId, highlights })).data
}
export async function saveBody(id: number, body: string, label = ''): Promise<ContentItem> {
  return (await apiClient.put(desk(id, 'body'), { body, label })).data
}
export async function assist(id: number, input: { action: AssistAction; selection: string; before: string; after: string; instruction?: string }): Promise<AssistResult> {
  return (await apiClient.post(desk(id, 'assist'), input, SLOW)).data
}
export async function addFact(id: number, fact: { text: string; source?: string; source_title?: string }): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'facts'), fact)).data
}
export async function runCheck(id: number): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'check'))).data
}
export async function saveComments(id: number, comments: Comment[]): Promise<ContentItem> {
  return (await apiClient.put(desk(id, 'comments'), { comments })).data
}
export async function finalize(id: number): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'finalize'))).data
}
export async function applyLearn(id: number, learnId: string, accept: boolean): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'learn'), { id: learnId, accept })).data
}
export async function makePlatforms(id: number): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'platforms'))).data
}
export async function cancelDeskJob(id: number): Promise<void> {
  await apiClient.post(desk(id, 'cancel'))
}
export async function setStage(id: number, stage: DeskStage): Promise<ContentItem> {
  return (await apiClient.post(desk(id, 'stage'), { stage })).data
}
export async function listVersions(id: number): Promise<TextVersion[]> {
  return (await apiClient.get(`/admin/content/items/${id}/versions`)).data
}
export async function getVersion(id: number, vid: number): Promise<TextVersion> {
  return (await apiClient.get(`/admin/content/items/${id}/versions/${vid}`)).data
}
export async function restoreVersion(id: number, vid: number): Promise<ContentItem> {
  return (await apiClient.post(`/admin/content/items/${id}/versions/${vid}/restore`)).data
}
