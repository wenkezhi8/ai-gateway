import { request } from './request'

// 服务商相关API
export interface Provider {
  id: number
  name: string
  type: string
  endpoint: string
  enabled: boolean
  accounts: number
  latency: string
  createdAt: string
  updatedAt: string
}

export interface ProviderCreateParams {
  name: string
  api_key: string
  base_url?: string
  models?: string[]
  enabled?: boolean
}

export const providerApi = {
  // 获取服务商列表
  getList(params?: { page?: number; pageSize?: number; keyword?: string }) {
    return request.get<{ success: boolean; data: Provider[] }>('/admin/providers', { params })
  },

  // 获取服务商详情
  getDetail(id: string) {
    return request.get<Provider>(`/admin/providers/${id}`)
  },

  // 创建服务商
  create(data: ProviderCreateParams) {
    return request.post<Provider>('/admin/providers', data)
  },

  // 更新服务商
  update(id: string, data: Partial<ProviderCreateParams>) {
    return request.put<Provider>(`/admin/providers/${id}`, data)
  },

  // 删除服务商
  delete(id: string) {
    return request.delete(`/admin/providers/${id}`)
  },

  // 测试服务商连接
  testConnection(id: string) {
    return request.post<{ success: boolean; response_time_ms: number }>(`/admin/providers/${id}/test`)
  },

  // 切换服务商状态
  toggleStatus(id: string, enabled: boolean) {
    const action = enabled ? 'enable' : 'disable'
    return request.post(`/admin/providers/${id}/${action}`)
  }
}
