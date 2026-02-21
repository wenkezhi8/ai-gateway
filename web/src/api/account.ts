import { request } from './request'

export type LimitType = 'token' | 'rpm' | 'concurrent' | 'request'
export type PeriodType = 'minute' | 'hour' | '5hour' | 'day' | 'week' | 'month'

export interface LimitConfig {
  type: LimitType
  period: PeriodType
  limit: number
  warning: number
}

export interface UsageInfo {
  key: string
  used: number
  limit: number
  remaining: number
  reset_at: string
  period: PeriodType
  percent_used: number
  warning_level?: 'normal' | 'warning' | 'critical'
}

export interface Account {
  id: string
  name: string
  provider: string
  provider_type?: string
  api_key?: string
  base_url?: string
  enabled: boolean
  priority?: number
  remark?: string
  limits?: Record<string, LimitConfig>
  is_active?: boolean
  plan_type?: string
  usage?: {
    token?: UsageInfo
    rpm?: UsageInfo
    hour5?: UsageInfo
    week?: UsageInfo
    month?: UsageInfo
  }
  last_switch?: string
}

export interface AccountCreateParams {
  id?: string
  name: string
  provider: string
  api_key?: string
  base_url?: string
  enabled?: boolean
  priority?: number
  remark?: string
  limits?: Record<string, LimitConfig>
  coding_plan_enabled?: boolean
}

export interface SwitchEvent {
  from_account: string
  to_account: string
  reason: string
  timestamp: string
  duration?: number
}

export interface LimitAlert {
  type: 'warning' | 'critical' | 'exceeded'
  account_id: string
  limit_type: LimitType
  current_used: number
  limit: number
  percent_used: number
  timestamp: string
  message: string
}

export const accountApi = {
  getList(params?: { page?: number; pageSize?: number; provider?: string; keyword?: string }) {
    return request.get<{ success: boolean; data: Account[] }>('/admin/accounts', { params })
  },

  getDetail(id: string) {
    return request.get<Account>(`/admin/accounts/${id}`)
  },

  create(data: AccountCreateParams) {
    return request.post<Account>('/admin/accounts', data)
  },

  update(id: string, data: Partial<AccountCreateParams>) {
    return request.put<Account>(`/admin/accounts/${id}`, data)
  },

  delete(id: string) {
    return request.delete(`/admin/accounts/${id}`)
  },

  toggleStatus(id: string, enabled: boolean) {
    return request.put(`/admin/accounts/${id}/status`, { enabled })
  },

  getUsageStats(id: string, params?: { startDate?: string; endDate?: string }) {
    return request.get<{
      dailyUsage: { date: string; requests: number; tokens: number }[]
      totalRequests: number
      totalTokens: number
    }>(`/admin/accounts/${id}/usage`, { params })
  },

  getAccountUsage(id: string) {
    return request.get<{ success: boolean; data: Account }>(`/admin/accounts/${id}/usage`)
  },

  forceSwitch(provider: string, accountId: string) {
    return request.post<{ success: boolean }>(`/admin/accounts/${accountId}/switch`, { provider })
  },

  getSwitchHistory(limit?: number) {
    return request.get<{ success: boolean; data: SwitchEvent[] }>('/admin/accounts/switch-history', { 
      params: { limit } 
    })
  },

  updateLimits(id: string, limits: Record<string, LimitConfig>) {
    return request.put<Account>(`/admin/accounts/${id}`, { limits })
  },

  getLimitAlerts() {
    return request.get<{ success: boolean; data: LimitAlert[] }>('/admin/dashboard/alerts')
  },

  fetchModels(id: string, sync: boolean = false) {
    const params = sync ? { sync: 'true' } : {}
    return request.get<{ success: boolean; data: { account_id: string; provider: string; models: string[]; total_models: number; synced?: boolean; synced_count?: number } }>(`/admin/accounts/${id}/fetch-models`, { params })
  }
}
