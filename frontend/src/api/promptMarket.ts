/**
 * 提示词交易 (admin): users sell their prompts or packs for 余额 in the canvas; listings and changes
 * are reviewed here, and purchases can be refunded (the seller's share is taken back).
 */
import { apiClient } from './client'

export interface PromptMarketConfig {
  enabled: boolean
  fee_rate: number
  min_price: number
  max_price: number
}

export interface PromptListingItem {
  position: number
  title: string
  prompt?: string
  tags: string[]
  kind: string
}

export interface PromptListingDraft {
  title: string
  description: string
  cover_url: string
  sample_urls: string[]
  price: number
  items: PromptListingItem[]
}

export type PromptListingStatus = 'pending' | 'active' | 'rejected' | 'hidden' | 'withdrawn'

export interface PromptListing {
  id: number
  seller_name: string
  seller_email?: string
  title: string
  description: string
  cover_url: string
  sample_urls: string[]
  price: number
  item_count: number
  status: PromptListingStatus
  review_note?: string
  flags?: string[]
  draft?: PromptListingDraft
  sales_count: number
  published_at?: string
  created_at: string
  updated_at: string
  items: PromptListingItem[]
}

export interface PromptPurchase {
  id: number
  listing_id: number
  buyer_user_id: number
  seller_user_id: number
  price: number
  fee_rate: number
  seller_amount: number
  status: 'paid' | 'refunded'
  refunded_at?: string
  created_at: string
  listing_title: string
  buyer_email?: string
  seller_email?: string
}

export async function getPromptMarketConfig(): Promise<PromptMarketConfig> {
  const { data } = await apiClient.get('/admin/prompt-market/config')
  return data
}

export async function savePromptMarketConfig(config: PromptMarketConfig): Promise<PromptMarketConfig> {
  const { data } = await apiClient.put('/admin/prompt-market/config', config)
  return data
}

/** status: review (pending or with a change) / pending / active / rejected / hidden / withdrawn / '' (all). */
export async function listPromptListings(params: { status?: string; q?: string; page?: number; page_size?: number }): Promise<{ items: PromptListing[]; total: number }> {
  const { data } = await apiClient.get('/admin/prompt-market/listings', { params })
  return data
}

export async function reviewPromptListing(id: number, action: 'approve' | 'reject' | 'hide' | 'restore', note = ''): Promise<PromptListing> {
  const { data } = await apiClient.post(`/admin/prompt-market/listings/${id}/${action}`, { note })
  return data
}

export async function listPromptPurchases(params: { status?: string; page?: number; page_size?: number }): Promise<{ items: PromptPurchase[]; total: number }> {
  const { data } = await apiClient.get('/admin/prompt-market/purchases', { params })
  return data
}

export async function refundPromptPurchase(id: number): Promise<PromptPurchase> {
  const { data } = await apiClient.post(`/admin/prompt-market/purchases/${id}/refund`)
  return data
}
