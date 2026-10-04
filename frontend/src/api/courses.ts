/**
 * Paid courses: public catalogue and pages, the buyer's courses and their netdisk delivery.
 * Courses are bought through /purchase?course=<id> (order_type = course).
 */
import { apiClient } from './client'

export interface CourseLesson {
  title: string
  duration: string
  trial: boolean
}

export interface CourseSection {
  title: string
  lessons: CourseLesson[]
}

export type CourseStatus = 'draft' | 'published' | 'archived'

export interface Course {
  id: number
  slug: string
  title: string
  subtitle: string
  category: string
  cover_url: string
  /** CNY */
  price: number
  original_price: number
  /** Course page only. */
  intro_md?: string
  outline: CourseSection[]
  trial_md?: string
  faq_md?: string
  status: CourseStatus
  sort_order: number
  lesson_count: number
  student_count: number
  /** The viewer has bought it (and it was not refunded). */
  owned: boolean
  created_at: string
  updated_at: string
  /** Admin only. */
  revenue?: number
  delivery_version?: number
}

export interface CourseDelivery {
  version: number
  link: string
  code: string
  password: string
  note: string
  updated_at: string
}

export interface CourseEnrollment {
  user_id: number
  course_id: number
  order_id?: number
  source: 'purchase' | 'admin'
  status: 'active' | 'revoked'
  note?: string
  first_viewed_at?: string
  last_viewed_at?: string
  view_count: number
  seen_version: number
  refunded: boolean
  created_at: string
  user_email?: string
  username?: string
  /** Distinct IPs that fetched the link in the last 7 days. */
  recent_ips: number
}

export interface MyCourse {
  course: Course
  enrollment: CourseEnrollment
  /** The link changed since the buyer last looked. */
  updated: boolean
}

export async function listCourses(): Promise<Course[]> {
  const { data } = await apiClient.get('/courses')
  return data
}

export async function getCourse(slug: string): Promise<Course> {
  const { data } = await apiClient.get(`/courses/${encodeURIComponent(slug)}`)
  return data
}

export async function myCourses(): Promise<MyCourse[]> {
  const { data } = await apiClient.get('/user/courses')
  return data
}

export async function getDelivery(courseId: number): Promise<CourseDelivery> {
  const { data } = await apiClient.get(`/user/courses/${courseId}/delivery`)
  return data
}

export function totalLessons(outline: CourseSection[]): number {
  return outline.reduce((n, s) => n + s.lessons.length, 0)
}

export default { listCourses, getCourse, myCourses, getDelivery }
