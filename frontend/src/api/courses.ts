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
  /** Limited-time price (until sale_ends_at); 0 = none. */
  sale_price: number
  sale_ends_at?: string
  /** Verified students / teachers get the education discount. */
  edu_discount: boolean
  /** A direct video link or a Bilibili page, shown as the free preview. */
  trial_video_url: string
  /** What the viewer pays now (limited-time price, then their education discount). */
  current_price: number
  sale_active: boolean
  edu_applied: boolean
  /** For viewers not verified: the discount verification would give (percent). */
  edu_percent?: number
  lesson_count: number
  student_count: number
  /** The viewer has bought it (and it was not refunded). */
  owned: boolean
  created_at: string
  updated_at: string
  /** Admin only. */
  revenue?: number
  delivery_version?: number
  views_30d?: number
  orders_30d?: number
  paid_30d?: number
  refunds?: number
  /** Creator courses: the creator's display name (public) and id. */
  creator_name?: string
  owner_id?: number
  /** Creator / admin only: the unpublished edit and its review. */
  draft?: CourseDraft
  review_status?: CourseReviewStatus
  review_note?: string
  submitted_at?: string
  approved_at?: string
}

export type CourseReviewStatus = '' | 'draft' | 'pending' | 'approved' | 'rejected'

/** A creator's edit waiting for review (same fields as the course input). */
export interface CourseDraft {
  title: string
  subtitle: string
  category: string
  price: number
  original_price: number
  intro_md: string
  outline: CourseSection[]
  trial_md: string
  faq_md: string
  sale_price: number
  sale_ends_at: string | null
  edu_discount: boolean
  trial_video_url: string
  cover_url?: string
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

/** The crossed-out price next to current_price: the list price when discounted, else the original price. */
export function strikePrice(c: Pick<Course, 'price' | 'current_price' | 'original_price'>): number {
  if (c.current_price < c.price) return c.price
  return c.original_price > c.current_price ? c.original_price : 0
}

/** Bilibili page → embeddable player URL; '' for anything else. */
export function bilibiliEmbed(url: string): string {
  const bv = url.match(/bilibili\.com\/video\/(BV[0-9A-Za-z]{10})/)
  if (bv) return `https://player.bilibili.com/player.html?bvid=${bv[1]}&autoplay=0&high_quality=1`
  const av = url.match(/bilibili\.com\/video\/av(\d+)/i)
  if (av) return `https://player.bilibili.com/player.html?aid=${av[1]}&autoplay=0&high_quality=1`
  return ''
}

export function totalLessons(outline: CourseSection[]): number {
  return outline.reduce((n, s) => n + s.lessons.length, 0)
}

export default { listCourses, getCourse, myCourses, getDelivery }
