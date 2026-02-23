import { request } from './request'

export interface RoutingConfig {
  strategy: 'round-robin' | 'weighted' | 'failover' | 'cost-optimized'
  failover_enabled: boolean
  health_check_interval: number
  timeout: number
  retry_count: number
  models: Record<string, string>
  provider_weights: Record<string, number>
}

export interface StrategyInfo {
  id: string
  name: string
  description: string
}

export const routingApi = {
  getRouting() {
    return request.get<RoutingConfig>('/admin/routing')
  },

  updateRouting(data: Partial<RoutingConfig>) {
    return request.put<RoutingConfig>('/admin/routing', data)
  },

  getStrategies() {
    return request.get<StrategyInfo[]>('/admin/routing/strategies')
  },

  setModelStrategy(model: string, strategy: string) {
    return request.put(`/admin/routing/models/${model}/strategy`, { strategy })
  },

  setProviderWeight(provider: string, weight: number) {
    return request.put(`/admin/routing/providers/${provider}/weight`, { weight })
  },

  resetRouting() {
    return request.post('/admin/routing/reset')
  }
}
