import { request } from './request'

export interface AlertRule {
  id: string
  name: string
  enabled: boolean
  condition: {
    type: 'latency' | 'error_rate' | 'quota' | 'availability'
    operator: string
    threshold: number
    duration?: number
  }
  notifyChannels: string[]
  createdAt: string
  updatedAt?: string
}

export interface Alert {
  id: string
  time: string
  level: 'critical' | 'warning' | 'info'
  source: string
  message: string
  status: 'pending' | 'resolved'
  ruleId?: string
  resolvedAt?: string
  dedup_key?: string
  first_triggered_at?: string
  last_triggered_at?: string
  trigger_count?: number
  auto_resolved?: boolean
}

export interface AlertStats {
  critical: number
  warning: number
  todayTotal: number
  resolved: number
}

export interface ResolveSimilarAlertsRequest {
  level: string
  source: string
  message: string
  dedup_key?: string
}

export const alertApi = {
  getStats() {
    return request.get<{ success: boolean; data: AlertStats }>('/admin/alerts/stats')
  },

  getRules() {
    return request.get<{ success: boolean; data: AlertRule[] }>('/admin/alerts/rules')
  },

  createRule(data: Omit<AlertRule, 'id' | 'createdAt'>) {
    return request.post<{ success: boolean; data: AlertRule }>('/admin/alerts/rules', data)
  },

  updateRule(id: string, data: Partial<AlertRule>) {
    return request.put<{ success: boolean; data: AlertRule }>(`/admin/alerts/rules/${id}`, data)
  },

  deleteRule(id: string) {
    return request.delete<{ success: boolean; message: string }>(`/admin/alerts/rules/${id}`)
  },

  getHistory(params?: {
    level?: string
    startDate?: string
    endDate?: string
  }) {
    return request.get<{ success: boolean; data: { list: Alert[]; total: number } }>('/admin/alerts/history', { params })
  },

  clearHistory() {
    return request.delete<{ success: boolean; data: { affected: number } }>('/admin/alerts/history')
  },

  resolveAlert(id: string) {
    return request.put<{ success: boolean; message: string }>(`/admin/alerts/${id}/resolve`)
  },

  resolveSimilar(payload: ResolveSimilarAlertsRequest) {
    return request.post<{ success: boolean; data: { affected: number; key: string } }>('/admin/alerts/resolve-similar', payload)
  },

  getDetail(id: string) {
    return request.get<{ success: boolean; data: Alert }>(`/admin/alerts/${id}`)
  }
}
