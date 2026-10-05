/**
 * Admin Agent Tokens API.
 * Manages fine-grained tokens for AI agents to automate maintenance and ops tasks.
 */

import { apiClient } from '../client'
import type { ApiResponse, PaginatedResponse } from '@/types'

export interface AgentToken {
  id: number
  name: string
  description?: string
  token_prefix: string
  scopes: string[]
  created_by: number
  last_used_at?: string
  expires_at?: string
  revoked_at?: string
  created_at: string
}

export interface AgentTokenQuery {
  page?: number
  page_size?: number
  include_revoked?: boolean
}

export interface CreateAgentTokenRequest {
  name: string
  scopes: string[]
  expires_in_days?: number
}

export interface CreateAgentTokenResponse {
  id: number
  name: string
  raw_token: string
  token_prefix: string
  scopes: string[]
  created_at: string
  expires_at?: string
}

export interface AgentScopeItem {
  scope: string
  description: string
}

export interface SkillDocResponse {
  content: string
}

/**
 * List agent tokens (paginated)
 */
export async function listAgentTokens(params?: AgentTokenQuery): Promise<PaginatedResponse<AgentToken>> {
  const { data } = await apiClient.get('/admin/agent-tokens', { params })
  return data
}

/**
 * Create a new agent token (returns raw token once)
 */
export async function createAgentToken(body: CreateAgentTokenRequest): Promise<ApiResponse<CreateAgentTokenResponse>> {
  const { data } = await apiClient.post('/admin/agent-tokens', body)
  return data
}

/**
 * Revoke an existing agent token
 */
export async function revokeAgentToken(id: number): Promise<ApiResponse<{ revoked: boolean; id: number }>> {
  const { data } = await apiClient.post(`/admin/agent-tokens/${id}/revoke`)
  return data
}

/**
 * Get available agent scopes list
 */
export async function getAgentScopes(): Promise<ApiResponse<AgentScopeItem[]>> {
  const { data } = await apiClient.get('/admin/agent-tokens/scopes')
  return data
}

/**
 * Get internal Agent Skill documentation content
 */
export async function getAgentSkillDoc(): Promise<ApiResponse<SkillDocResponse>> {
  const { data } = await apiClient.get('/admin/agent-tokens/skill-doc')
  return data
}
