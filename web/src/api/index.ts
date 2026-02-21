import { request } from './request'

// 路由策略相关API
export interface RoutingRule {
  id: number
  name: string
  priority: number
  conditions: { type: string; value: string }[]
  target: string
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface GlobalStrategy {
  loadBalance: 'round-robin' | 'weighted' | 'least-conn' | 'random'
  failover: boolean
  healthCheckInterval: number
  timeout: number
  retryCount: number
}

export const routingApi = {
  // 获取路由规则列表
  getRules() {
    return request.get<RoutingRule[]>('/routing/rules')
  },

  // 创建路由规则
  createRule(data: Omit<RoutingRule, 'id' | 'createdAt' | 'updatedAt'>) {
    return request.post<RoutingRule>('/routing/rules', data)
  },

  // 更新路由规则
  updateRule(id: number, data: Partial<RoutingRule>) {
    return request.put<RoutingRule>(`/routing/rules/${id}`, data)
  },

  // 删除路由规则
  deleteRule(id: number) {
    return request.delete(`/routing/rules/${id}`)
  },

  // 获取全局策略
  getGlobalStrategy() {
    return request.get<GlobalStrategy>('/routing/strategy')
  },

  // 更新全局策略
  updateGlobalStrategy(data: Partial<GlobalStrategy>) {
    return request.put<GlobalStrategy>('/routing/strategy', data)
  },

  // 获取服务商权重
  getProviderWeights() {
    return request.get<{ provider: string; weight: number }[]>('/routing/weights')
  },

  // 更新服务商权重
  updateProviderWeights(weights: { provider: string; weight: number }[]) {
    return request.put('/routing/weights', { weights })
  }
}
