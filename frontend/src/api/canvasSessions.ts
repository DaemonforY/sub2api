/**
 * Canvas sign-in sessions: the connect popup creates one (the API sets an HttpOnly cookie that only
 * the canvas endpoints read), and the profile page lists and signs out devices.
 */
import { apiClient } from './client'

export interface CanvasSession {
  id: number
  user_agent: string
  ip: string
  created_at: string
  last_used_at: string
  expires_at: string
}

export async function createCanvasSession(): Promise<CanvasSession> {
  const { data } = await apiClient.post('/user/canvas-sessions')
  return data
}
export async function listCanvasSessions(): Promise<CanvasSession[]> {
  const { data } = await apiClient.get('/user/canvas-sessions')
  return data
}
export async function revokeCanvasSession(id: number): Promise<void> {
  await apiClient.delete(`/user/canvas-sessions/${id}`)
}

/** "Chrome · macOS" from a user agent (best effort). */
export function describeUserAgent(ua: string): string {
  const browser = /Edg\//.test(ua) ? 'Edge' : /Chrome\//.test(ua) ? 'Chrome' : /Firefox\//.test(ua) ? 'Firefox' : /Safari\//.test(ua) ? 'Safari' : ''
  const os = /iPhone|iPad/.test(ua) ? 'iOS' : /Android/.test(ua) ? 'Android' : /Mac OS X/.test(ua) ? 'macOS' : /Windows/.test(ua) ? 'Windows' : /Linux/.test(ua) ? 'Linux' : ''
  return [browser, os].filter(Boolean).join(' · ')
}
