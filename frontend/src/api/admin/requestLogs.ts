/**
 * Admin gateway request log API.
 *
 * One entry per request that reached the API gateway (success or failure),
 * including the full URL and the full API key the caller presented.
 * Admin-only.
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface GatewayRequestLog {
  id: number
  created_at: string
  request_id: string
  method: string
  url: string
  path: string
  status_code: number
  success: boolean
  error_code: string
  duration_ms: number
  api_key: string
  api_key_id?: number
  api_key_name?: string
  user_id?: number
  user_email?: string
  group_id?: number
  account_id?: number
  model: string
  client_ip: string
  user_agent: string
}

export interface GatewayRequestLogQuery {
  page?: number
  page_size?: number
  start_time?: string
  end_time?: string
  user_id?: number
  api_key_id?: number
  success?: string
  status_code?: number
  method?: string
  model?: string
  q?: string
}

export type GatewayRequestLogListResponse = PaginatedResponse<GatewayRequestLog>

export async function list(params: GatewayRequestLogQuery): Promise<GatewayRequestLogListResponse> {
  const { data } = await apiClient.get('/admin/request-logs', { params })
  return data
}

export const requestLogsAPI = { list }

export default requestLogsAPI
