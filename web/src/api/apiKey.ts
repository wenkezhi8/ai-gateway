import { request } from './request'

export interface ApiKey {
  id: string
  name: string
  key: string
  description?: string
  created_at: string
  last_used?: string
  enabled: boolean
}

export interface ApiKeyCreateParams {
  name: string
  description?: string
}

export const apiKeyApi = {
  getList() {
    return request.get<{ success: boolean; data: ApiKey[] }>('/admin/api-keys')
  },

  create(data: ApiKeyCreateParams) {
    return request.post<{ success: boolean; data: { id: string; key: string } }>('/admin/api-keys', data)
  },

  update(id: string, data: Partial<{ name: string; description: string; enabled: boolean }>) {
    return request.put<{ success: boolean }>(`/admin/api-keys/${id}`, data)
  },

  delete(id: string) {
    return request.delete<{ success: boolean }>(`/admin/api-keys/${id}`)
  },

  toggleStatus(id: string, enabled: boolean) {
    return request.put<{ success: boolean }>(`/admin/api-keys/${id}`, { enabled })
  }
}
