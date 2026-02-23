import { request } from './request'

export interface CacheConfig {
  enabled: boolean
  strategy: 'semantic' | 'exact' | 'prefix'
  similarityThreshold: number
  defaultTTL: number
  maxSize: number
}

export interface CacheStatDetail {
  hit_rate: number
  hits: number
  misses: number
  size: number
  entries: number
}

export interface CacheStatsResponse {
  request_cache: CacheStatDetail
  context_cache: CacheStatDetail
  route_cache: CacheStatDetail
  usage_cache: CacheStatDetail
  response_cache: CacheStatDetail
}

export interface CacheHealthResponse {
  status: string
  backend: string
  latency_ms: number
}

export interface CacheSummaryResponse {
  total_entries: number
  total_size: number
  by_type: Record<string, number>
}

export const cacheApi = {
  getConfig() {
    return request.get<CacheConfig>('/admin/cache/config')
  },

  updateConfig(data: Partial<CacheConfig>) {
    return request.put<CacheConfig>('/admin/cache/config', data)
  },

  getStats() {
    return request.get<CacheStatsResponse>('/admin/cache/stats')
  },

  getHealth() {
    return request.get<CacheHealthResponse>('/admin/cache/health')
  },

  getSummary() {
    return request.get<CacheSummaryResponse>('/admin/cache/summary')
  },

  clearCache() {
    return request.delete('/admin/cache')
  },

  invalidateProvider(provider: string) {
    return request.delete(`/admin/cache/provider/${provider}`)
  },

  invalidateModel(model: string) {
    return request.delete(`/admin/cache/model/${model}`)
  }
}
