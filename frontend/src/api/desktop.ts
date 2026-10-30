/**
 * HiveGPT 桌面助手 sign-in: the desktop app shows a QR of /desktop-login?t=<ticket>; the phone
 * signs in on the site and confirms here. The app gets its device key by polling (not from this page).
 */
import { apiClient } from './client'

export type DesktopLoginStatus = 'pending' | 'approved' | 'rejected' | 'expired' | 'done'

export interface DesktopLoginInfo {
  device_name: string
  status: DesktopLoginStatus
  expires_in: number
}

export async function getDesktopLogin(ticketId: string): Promise<DesktopLoginInfo> {
  const { data } = await apiClient.get<DesktopLoginInfo>('/desktop/login', { params: { t: ticketId } })
  return data
}

export async function decideDesktopLogin(ticketId: string, approve: boolean): Promise<{ status: DesktopLoginStatus }> {
  const { data } = await apiClient.post<{ status: DesktopLoginStatus }>('/desktop/login/decide', {
    ticket_id: ticketId,
    approve
  })
  return data
}
