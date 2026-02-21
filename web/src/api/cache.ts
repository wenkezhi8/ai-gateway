import { request } from './request'

// 缓存管理相关API
export interface CacheConfig {
  enabled: boolean
  strategy: 'semantic' | 'exact' | 'prefix'
  similarityThreshold: number
  defaultTTL: number
  maxSize: number
}

export interface CacheRule {
  id: number
  pattern: string
  ttl: number
  enabled: boolean
}

export interface CacheStats {
  hitRate: number
  size: number
  entries: number
  avgResponseTime: number
}

export const cacheApi = {
  // 获取缓存配置
  getConfig() {
    return request.get<CacheConfig>('/cache/config')
  },

  // 更新缓存配置
  updateConfig(data: Partial<CacheConfig>) {
    return request.put<CacheConfig>('/cache/config', data)
  },

  // 获取缓存统计
  getStats() {
    return request.get<CacheStats>('/cache/stats')
  },

  // 获取缓存规则
  getRules() {
    return request.get<CacheRule[]>('/cache/rules')
  },

  // 创建缓存规则
  createRule(data: Omit<CacheRule, 'id'>) {
    return request.post<CacheRule>('/cache/rules', data)
  },

  // 删除缓存规则
  deleteRule(id: number) {
    return request.delete(`/cache/rules/${id}`)
  },

  // 获取热门缓存
  getHotCaches(limit?: number) {
    return request.get<{ query: string; hits: number; lastHit: string }[]>('/cache/hot', {
      params: { limit: limit || 10 }
    })
  },

  // 清空缓存
  clearCache() {
    return request.post('/cache/clear')
  }
}
