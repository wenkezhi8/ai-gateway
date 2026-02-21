import { request } from './request'

// 仪表盘相关API
export interface DashboardStats {
  todayRequests: number
  successRate: number
  avgLatency: number
  activeAccounts: number
}

export interface RecentRequest {
  time: string
  provider: string
  model: string
  status: 'success' | 'failed'
}

export const dashboardApi = {
  getStats() {
    return request.get<DashboardStats>('/admin/dashboard/stats')
  },

  getRequestTrend(params?: { startDate?: string; endDate?: string }) {
    return request.get<{ date: string; requests: number; success: number }[]>('/admin/dashboard/requests', { params })
  },

  getProviderDistribution() {
    return request.get<{ provider: string; count: number; percentage: number }[]>('/admin/dashboard/stats')
  },

  getRecentRequests(limit?: number) {
    return request.get<RecentRequest[]>('/admin/dashboard/stats', {
      params: { limit: limit || 10 }
    })
  },

  getRecentAlerts(limit?: number) {
    return request.get<{ time: string; level: string; message: string }[]>('/admin/dashboard/alerts', {
      params: { limit: limit || 5 }
    })
  }
}
