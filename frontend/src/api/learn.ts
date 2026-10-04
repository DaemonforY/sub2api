/** AI 学习 (/learn) for the signed-in user: progress, quiz results, certificates and today's free calls. */
import { apiClient } from './client'

export interface LearnCertificate {
  code: string
  track: string
  track_title: string
  display_name: string
  project_url: string
  quiz_score: number
  issued_at: string
}

export interface LearnMe {
  completed: Record<string, string>
  runs_left: number
  tutor_left: number
  interviews_left: number
  runs_today: number
  tutor_today: number
  interviews_today: number
  quizzes: Record<string, { correct: number; total: number; attempts: number }>
  checkpoints: Record<string, string>
  certificates: LearnCertificate[]
}

export interface LearnTrack {
  id: string
  letter: string
  title: string
  project: string
  lessons: { id: string; title: string; minutes: number }[]
}

export async function me(): Promise<LearnMe> {
  const { data } = await apiClient.get('/learn/me')
  return data
}

/** The tracks and lessons, published by the learning site at build time. */
export async function tracks(): Promise<LearnTrack[]> {
  const res = await fetch('/learn/tracks.json', { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export default { me, tracks }
