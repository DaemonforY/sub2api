/** Admin API for paid courses: editing, covers and images, netdisk delivery versions, students. */
import { apiClient } from '../client'
import type { Course, CourseDelivery, CourseEnrollment, CourseSection, CourseStatus } from '../courses'

export interface CourseInput {
  slug: string
  title: string
  subtitle: string
  category: string
  price: number
  original_price: number
  intro_md: string
  outline: CourseSection[]
  trial_md: string
  faq_md: string
  status: CourseStatus
  sort_order: number
  sale_price: number
  sale_ends_at: string | null
  edu_discount: boolean
  trial_video_url: string
}

export interface CourseSettings {
  /** Percent of a course order credited to the buyer's inviter (0 = none). */
  affiliate_rate_percent: number
  /** Creators: applications open, default commission, days before a sale can be withdrawn, minimum withdrawal. */
  creator_enabled: boolean
  creator_commission_percent: number
  creator_settle_days: number
  creator_withdraw_min_cny: number
}

export type CreatorStatus = 'pending' | 'approved' | 'rejected' | 'suspended'

export interface CreatorBalance {
  orders: number
  gross: number
  net: number
  frozen: number
  paid: number
  pending: number
  /** Can be negative after a refund of a sale already paid out. */
  available: number
}

export interface CourseCreator {
  user_id: number
  status: CreatorStatus
  display_name: string
  bio: string
  contact: string
  plan: string
  admin_note: string
  reviewed_at?: string
  created_at: string
  user_email?: string
  username?: string
  course_count: number
  on_sale_count: number
  pending_count: number
  /** The creator's own rate; null = the site default. */
  commission_percent: number | null
  balance?: CreatorBalance
}

export interface CreatorUpdate {
  status: CreatorStatus
  commission_percent: number | null
  admin_note: string
}

export interface CreatorWithdrawal {
  id: number
  user_id: number
  user_email?: string
  display_name?: string
  cny_amount: number
  method: 'alipay' | 'wechat'
  account: string
  real_name: string
  user_note: string
  status: 'pending' | 'paid' | 'rejected' | 'cancelled'
  admin_note: string
  reviewed_at?: string
  created_at: string
}

export interface DeliveryInput {
  link: string
  code: string
  password: string
  note: string
  /** Email the course's buyers that the link changed. */
  notify: boolean
}

export async function list(): Promise<Course[]> {
  const { data } = await apiClient.get('/admin/courses')
  return data
}
export async function get(id: number): Promise<Course> {
  const { data } = await apiClient.get(`/admin/courses/${id}`)
  return data
}
export async function create(input: CourseInput): Promise<Course> {
  const { data } = await apiClient.post('/admin/courses', input)
  return data
}
export async function update(id: number, input: CourseInput): Promise<Course> {
  const { data } = await apiClient.put(`/admin/courses/${id}`, input)
  return data
}
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/courses/${id}`)
}
export async function uploadCover(id: number, image: File): Promise<Course> {
  const form = new FormData()
  form.append('image', image)
  const { data } = await apiClient.post(`/admin/courses/${id}/cover`, form)
  return data
}
/** An image for the course's Markdown; returns its URL. */
export async function uploadImage(image: File): Promise<string> {
  const form = new FormData()
  form.append('image', image)
  const { data } = await apiClient.post('/admin/courses/images', form)
  return data.url
}
export async function deliveries(id: number): Promise<CourseDelivery[]> {
  const { data } = await apiClient.get(`/admin/courses/${id}/deliveries`)
  return data
}
export async function saveDelivery(id: number, input: DeliveryInput): Promise<CourseDelivery> {
  const { data } = await apiClient.post(`/admin/courses/${id}/deliveries`, input)
  return data
}
export async function enrollments(id: number, page = 1): Promise<{ items: CourseEnrollment[]; total: number }> {
  const { data } = await apiClient.get(`/admin/courses/${id}/enrollments`, { params: { page } })
  return data
}
/** user: user ID or sign-up email. */
export async function grant(id: number, user: string, note: string): Promise<void> {
  await apiClient.post(`/admin/courses/${id}/enrollments`, { user, note })
}
export async function revoke(id: number, userId: number): Promise<void> {
  await apiClient.delete(`/admin/courses/${id}/enrollments/${userId}`)
}

export async function getSettings(): Promise<CourseSettings> {
  const { data } = await apiClient.get('/admin/courses/settings')
  return data
}
export async function saveSettings(settings: CourseSettings): Promise<CourseSettings> {
  const { data } = await apiClient.put('/admin/courses/settings', settings)
  return data
}

export async function creators(status = ''): Promise<CourseCreator[]> {
  const { data } = await apiClient.get('/admin/courses/creators', { params: status ? { status } : {} })
  return data
}
export async function updateCreator(userId: number, input: CreatorUpdate): Promise<CourseCreator> {
  const { data } = await apiClient.put(`/admin/courses/creators/${userId}`, input)
  return data
}
/** Creator courses waiting for review, with their drafts. */
export async function reviews(): Promise<Course[]> {
  const { data } = await apiClient.get('/admin/courses/reviews')
  return data
}
export async function review(id: number, approve: boolean, note: string): Promise<Course> {
  const { data } = await apiClient.post(`/admin/courses/${id}/review`, { approve, note })
  return data
}
export async function creatorWithdrawals(status: string, page = 1): Promise<{ items: CreatorWithdrawal[]; total: number }> {
  const { data } = await apiClient.get('/admin/courses/creator-withdrawals', { params: { status, page } })
  return data
}
export async function payCreatorWithdrawal(id: number, note: string): Promise<CreatorWithdrawal> {
  const { data } = await apiClient.post(`/admin/courses/creator-withdrawals/${id}/paid`, { note })
  return data
}
export async function rejectCreatorWithdrawal(id: number, note: string): Promise<CreatorWithdrawal> {
  const { data } = await apiClient.post(`/admin/courses/creator-withdrawals/${id}/reject`, { note })
  return data
}

export default {
  creators,
  updateCreator,
  reviews,
  review,
  creatorWithdrawals,
  payCreatorWithdrawal,
  rejectCreatorWithdrawal,
  getSettings, saveSettings, list, get, create, update, remove, uploadCover, uploadImage, deliveries, saveDelivery, enrollments, grant, revoke }
