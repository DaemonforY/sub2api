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

export default { getSettings, saveSettings, list, get, create, update, remove, uploadCover, uploadImage, deliveries, saveDelivery, enrollments, grant, revoke }
