/**
 * 卡密商品 (card-key goods, e.g. Muse): the product page /goods/<slug>, the buyer's delivered keys
 * and the admin's products and stock. Bought through /purchase?goods=<slug> (order_type = goods,
 * plan_id = goods id); a paid order gets one key from stock automatically.
 */
import { apiClient } from './client'

export interface GoodsDelivery {
  order_id: number
  goods_id: number
  goods_slug: string
  goods_name: string
  content: string
  usage_note: string
  delivered_at: string
}

export interface GoodsProduct {
  id: number
  slug: string
  name: string
  subtitle: string
  description: string
  price: number
  original_price: number
  /** 0 = no limit. */
  per_user_limit: number
  /** Keys still offered for sale. */
  stock: number
  /** Keys the viewer bought or has open orders for. */
  bought: number
  deliveries: GoodsDelivery[]
}

export interface Goods {
  id: number
  slug: string
  name: string
  subtitle: string
  description: string
  usage_note: string
  price: number
  original_price: number
  per_user_limit: number
  enabled: boolean
  sort_order: number
  created_at?: string
  updated_at?: string
}

export interface GoodsStock {
  available: number
  sold: number
  void: number
  /** Keys promised to open orders. */
  reserved: number
}

export interface GoodsAdminRow extends Goods {
  stock: GoodsStock
  /** Paid orders waiting for a key. */
  waiting: number
}

export type GoodsItemStatus = 'available' | 'sold' | 'void'

export interface GoodsItem {
  id: number
  preview: string
  status: GoodsItemStatus
  order_id?: number
  user_id?: number
  user_email?: string
  username?: string
  batch_note: string
  created_at: string
  delivered_at?: string
}

export interface GoodsImportResult {
  added: number
  duplicates: number
  retried: number
}

export async function getGoods(slug: string): Promise<GoodsProduct> {
  const { data } = await apiClient.get(`/goods/${encodeURIComponent(slug)}`)
  return data
}

export async function getOrderDelivery(orderId: number): Promise<GoodsDelivery | null> {
  const { data } = await apiClient.get(`/user/goods/orders/${orderId}/delivery`)
  return data?.delivery ?? null
}

export async function adminListGoods(): Promise<GoodsAdminRow[]> {
  const { data } = await apiClient.get('/admin/goods')
  return data
}

export async function adminSaveGoods(goods: Goods): Promise<Goods> {
  const { data } = goods.id
    ? await apiClient.put(`/admin/goods/${goods.id}`, goods)
    : await apiClient.post('/admin/goods', goods)
  return data
}

export async function adminDeleteGoods(id: number): Promise<void> {
  await apiClient.delete(`/admin/goods/${id}`)
}

export async function adminImportGoodsKeys(id: number, keys: string, batchNote: string): Promise<GoodsImportResult> {
  const { data } = await apiClient.post(`/admin/goods/${id}/import`, { keys, batch_note: batchNote })
  return data
}

export async function adminListGoodsItems(id: number, params: { status?: string; page?: number; page_size?: number }): Promise<{ items: GoodsItem[]; total: number }> {
  const { data } = await apiClient.get(`/admin/goods/${id}/items`, { params })
  return data
}

export async function adminVoidGoodsItems(id: number, ids: number[]): Promise<{ voided: number }> {
  const { data } = await apiClient.post(`/admin/goods/${id}/items/void`, { ids })
  return data
}
