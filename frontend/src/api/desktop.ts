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
  /** The computer asked from this phone's network (same IP, IPv4 /24 or IPv6 /48). */
  same_network: boolean
  /** The computer's address with the last part hidden, e.g. 203.0.113.*. */
  computer_network: string
  /** Seconds since the computer asked. */
  started_ago: number
}

export async function getDesktopLogin(ticketId: string): Promise<DesktopLoginInfo> {
  const { data } = await apiClient.get<DesktopLoginInfo>('/desktop/login', { params: { t: ticketId } })
  return data
}

/** confirmNearby: 「这台电脑就在我面前」, required by the server when the networks differ. */
export async function decideDesktopLogin(
  ticketId: string,
  approve: boolean,
  confirmNearby = false
): Promise<{ status: DesktopLoginStatus }> {
  const { data } = await apiClient.post<{ status: DesktopLoginStatus }>('/desktop/login/decide', {
    ticket_id: ticketId,
    approve,
    confirm_nearby: confirmNearby
  })
  return data
}
