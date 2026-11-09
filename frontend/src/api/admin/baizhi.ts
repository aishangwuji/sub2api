/**
 * Baizhi Cloud (百智云) Admin API
 * Manages configuration, multi-account key pools, and connectivity tests for Baizhi Cloud services.
 */

import { apiClient } from '../client'

export interface BaizhiKeyEntry {
  id: string
  account_name: string
  api_key: string
  cookie?: string
  enabled: boolean
  weight: number
  temporary_credits?: number
  permanent_credits?: number
  available_credits?: number
  daily_claimed?: boolean
  user_id?: number
  user_name?: string
  avatar?: string
  last_sync_at?: string
  created_at?: string
}

export interface BaizhiDiscoveredKey {
  name: string
  key: string
  total_used: number
  today_used: number
}

export interface BaizhiAccountInspectResult {
  user_id: number
  user_name: string
  avatar: string
  temporary_credits: number
  permanent_credits: number
  available_credits: number
  daily_claimed: boolean
  discovered_keys: BaizhiDiscoveredKey[]
}

export interface BaizhiServiceConfig {
  enabled: boolean
  strategy: 'round_robin' | 'failover' | 'temp_credit_first'
  base_url: string
  wss_url?: string
  keys: BaizhiKeyEntry[]
}

export interface BaizhiConfig {
  auth_required: boolean
  web_scrape: BaizhiServiceConfig
  web_search: BaizhiServiceConfig
  voice: BaizhiServiceConfig
  image_generate: BaizhiServiceConfig
}

export interface BaizhiTestRequest {
  service_type: 'web_scrape' | 'web_search' | 'voice' | 'image_generate'
  api_key?: string
  key_id?: string
  custom_base_url?: string
}

export interface BaizhiTestResult {
  success: boolean
  status_code: number
  latency_ms: number
  message: string
  response_body?: string
}

export async function getBaizhiConfig(): Promise<BaizhiConfig> {
  const res = await apiClient.get<BaizhiConfig>('/admin/baizhi/config')
  return res.data
}

export async function updateBaizhiConfig(config: BaizhiConfig): Promise<void> {
  await apiClient.put<{ updated: boolean }>('/admin/baizhi/config', config)
}

export async function testBaizhiConnection(req: BaizhiTestRequest): Promise<BaizhiTestResult> {
  const res = await apiClient.post<BaizhiTestResult>('/admin/baizhi/test', req)
  return res.data
}

export async function inspectBaizhiAccount(cookie: string): Promise<BaizhiAccountInspectResult> {
  const res = await apiClient.post<BaizhiAccountInspectResult>('/admin/baizhi/account/inspect', { cookie })
  return res.data
}

export async function syncBaizhiBalances(): Promise<BaizhiConfig> {
  const res = await apiClient.post<BaizhiConfig>('/admin/baizhi/sync-balances')
  return res.data
}

export default {
  getBaizhiConfig,
  updateBaizhiConfig,
  testBaizhiConnection,
  inspectBaizhiAccount,
  syncBaizhiBalances,
}
