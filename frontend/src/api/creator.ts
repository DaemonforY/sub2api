/**
 * Creator center: apply to sell courses, write them (an admin reviews the first listing and every
 * change), set the netdisk link (live at once), see sales and withdraw the creator's share.
 */
import { apiClient } from './client'
import type { Course, CourseDelivery, CourseEnrollment } from './courses'
import type { CourseInput, CourseCreator, CreatorBalance, CreatorWithdrawal, DeliveryInput } from './admin/courses'

export type { CourseCreator, CreatorBalance, CreatorWithdrawal }

export interface CreatorHome {
  /** Applications are open. */
  enabled: boolean
  /** null before applying. */
  creator: CourseCreator | null
  /** The creator's commission rate (their own or the site default), percent. */
  commission_percent: number
  settle_days: number
  withdraw_min_cny: number
  max_courses: number
  balance?: CreatorBalance
  withdrawals: CreatorWithdrawal[]
  last_method?: 'alipay' | 'wechat'
  last_account?: string
  last_real_name?: string
}

export interface CreatorApplication {
  display_name: string
  bio: string
  contact: string
  plan: string
}

export interface CreatorSale {
  order_id: number
  course_id: number
  course_title: string
  buyer: string
  gross: number
  commission_percent: number
  net: number
  status: 'frozen' | 'available' | 'refunding' | 'refunded' | 'partially_refunded'
  available_at: string
  created_at: string
}

export interface CreatorWithdrawInput {
  cny_amount: number
  method: 'alipay' | 'wechat'
  account: string
  real_name: string
  note: string
}

/** The creator's course input: slug is only read on create; status and order stay the platform's. */
export type CreatorCourseInput = Omit<CourseInput, 'status' | 'sort_order'>

export async function home(): Promise<CreatorHome> {
  const { data } = await apiClient.get('/user/creator')
  return data
}
export async function apply(input: CreatorApplication): Promise<CourseCreator> {
  const { data } = await apiClient.post('/user/creator/apply', input)
  return data
}
export async function courses(): Promise<Course[]> {
  const { data } = await apiClient.get('/user/creator/courses')
  return data
}
export async function get(id: number): Promise<Course> {
  const { data } = await apiClient.get(`/user/creator/courses/${id}`)
  return data
}
export async function create(input: CreatorCourseInput): Promise<Course> {
  const { data } = await apiClient.post('/user/creator/courses', input)
  return data
}
/** Saves the draft (a pending submission is withdrawn). */
export async function update(id: number, input: CreatorCourseInput): Promise<Course> {
  const { data } = await apiClient.put(`/user/creator/courses/${id}`, input)
  return data
}
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/user/creator/courses/${id}`)
}
export async function uploadCover(id: number, image: File): Promise<Course> {
  const form = new FormData()
  form.append('image', image)
  const { data } = await apiClient.post(`/user/creator/courses/${id}/cover`, form)
  return data
}
export async function uploadImage(image: File): Promise<string> {
  const form = new FormData()
  form.append('image', image)
  const { data } = await apiClient.post('/user/creator/images', form)
  return data.url
}
export async function submit(id: number): Promise<Course> {
  const { data } = await apiClient.post(`/user/creator/courses/${id}/submit`)
  return data
}
export async function setOnSale(id: number, onSale: boolean): Promise<Course> {
  const { data } = await apiClient.post(`/user/creator/courses/${id}/sale`, { on_sale: onSale })
  return data
}
export async function deliveries(id: number): Promise<CourseDelivery[]> {
  const { data } = await apiClient.get(`/user/creator/courses/${id}/deliveries`)
  return data
}
export async function saveDelivery(id: number, input: DeliveryInput): Promise<CourseDelivery> {
  const { data } = await apiClient.post(`/user/creator/courses/${id}/deliveries`, input)
  return data
}
export async function students(id: number, page = 1): Promise<{ items: CourseEnrollment[]; total: number }> {
  const { data } = await apiClient.get(`/user/creator/courses/${id}/students`, { params: { page } })
  return data
}
export async function sales(page = 1): Promise<{ items: CreatorSale[]; total: number }> {
  const { data } = await apiClient.get('/user/creator/sales', { params: { page } })
  return data
}
export async function withdraw(input: CreatorWithdrawInput): Promise<CreatorWithdrawal> {
  const { data } = await apiClient.post('/user/creator/withdrawals', input)
  return data
}
export async function cancelWithdraw(id: number): Promise<CreatorWithdrawal> {
  const { data } = await apiClient.post(`/user/creator/withdrawals/${id}/cancel`)
  return data
}

export default {
  home,
  apply,
  courses,
  get,
  create,
  update,
  remove,
  uploadCover,
  uploadImage,
  submit,
  setOnSale,
  deliveries,
  saveDelivery,
  students,
  sales,
  withdraw,
  cancelWithdraw
}
