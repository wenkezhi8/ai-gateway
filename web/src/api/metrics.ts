import request from './request'

// 类型定义
export interface OverviewData {
  total_requests: number
  requests_today: number
  success_rate: number
  avg_latency_ms: number
  total_tokens: number
  active_accounts: number
  active_providers: number
  cache_hit_rate: number
  provider_stats: ProviderStat[]
  top_models: TopModel[]
}

export interface ProviderStat {
  name: string
  requests: number
  tokens: number
  success_rate: number
  avg_latency_ms: number
}

export interface TopModel {
  name: string
  requests: number
  tokens: number
}

export interface RequestTrendPoint {
  timestamp: string
  requests: number
  success: number
  failed: number
  avg_latency_ms: number
}

export interface RequestTrendData {
  period: string
  interval: string
  data: RequestTrendPoint[]
}

export interface ProviderDetail {
  name: string
  models: string[]
  enabled: boolean
  requests: number
  tokens: number
  success_rate: number
  avg_latency_ms: number
  last_used: string
}

export interface ProvidersData {
  providers: ProviderDetail[]
  distribution: Record<string, number>
  total: number
}

export interface CacheStats {
  hits: number
  misses: number
  hit_rate: number
  size_bytes: number
  entries: number
  avg_latency_ms: number
  max_size: number
  evictions: number
}

export interface CacheData {
  request_cache: CacheStats
  context_cache: CacheStats
  route_cache: CacheStats
  usage_cache: CacheStats
  response_cache: CacheStats
  token_savings: number
}

export interface UsageData {
  start_time: string
  end_time: string
  total_tokens: number
  prompt_tokens: number
  output_tokens: number
  total_requests: number
  by_model: ModelUsage[]
  by_user: UserUsage[]
  daily_trend: DailyTrend[]
}

export interface ModelUsage {
  model: string
  requests: number
  tokens: number
  prompt_tokens: number
  output_tokens: number
  percent_of_total: number
}

export interface UserUsage {
  user_id: string
  requests: number
  tokens: number
  percent_of_total: number
}

export interface DailyTrend {
  date: string
  requests: number
  tokens: number
  users: number
}

export interface RealtimeData {
  timestamp: string
  active_connections: number
  requests_per_minute: number
  tokens_per_minute: number
  avg_latency_ms: number
  error_rate: number
  top_models: TopModel[]
  recent_errors: RecentError[]
}

export interface RecentError {
  timestamp: string
  provider: string
  model: string
  error: string
  count: number
}

// API 接口

/**
 * 获取系统概览
 */
export function getOverview() {
  return request.get<OverviewData>('/admin/dashboard/stats', { silent: true } as any)
}

/**
 * 获取请求趋势
 */
export function getRequestTrend(params?: { period?: string; interval?: string }) {
  return request.get<RequestTrendData>('/admin/dashboard/requests', { params, silent: true } as any)
}

/**
 * 获取服务商统计
 */
export function getProviders() {
  return request.get<ProvidersData>('/admin/dashboard/system', { silent: true } as any)
}

/**
 * 获取缓存统计
 */
export function getCacheStats() {
  return request.get<CacheData>('/admin/cache/stats', { silent: true } as any)
}

/**
 * 获取用量统计
 */
export function getUsage(params?: { start?: string; end?: string }) {
  return request.get<UsageData>('/admin/dashboard/stats', { params, silent: true } as any)
}

/**
 * 获取实时指标
 */
export function getRealtime() {
  return request.get<RealtimeData>('/admin/dashboard/realtime', { silent: true } as any)
}
