import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface ProviderType {
  id: string
  label: string
  category: 'international' | 'chinese' | 'local' | 'custom'
  color: string
  logo: string
  icon?: string
  default_endpoint: string
  coding_endpoint?: string
  supports_coding_plan: boolean
  models: string[]
}

export interface PublicProviderInfo {
  id: string
  label: string
  color: string
  logo: string
  default_model?: string
}

interface ProviderTypesResponse {
  success: boolean
  data?: ProviderType[]
  error?: string
}

interface PublicProvidersResponse {
  success: boolean
  data?: PublicProviderInfo[] | { providers?: Array<{ id?: string; name?: string; label?: string; color?: string; logo?: string; default_model?: string; models?: string[] }> }
  error?: string
}

const PROVIDER_TYPE_CATEGORIES = new Set<ProviderType['category']>(['international', 'chinese', 'local', 'custom'])

const PROVIDER_LOGO_FILE_MAP: Record<string, string> = {
  'azure-openai': 'azure',
  kimi: 'moonshot'
}

function resolveProviderLogo(providerId: string, explicitLogo?: string): string {
  if (explicitLogo) return String(explicitLogo)
  const normalizedId = (PROVIDER_LOGO_FILE_MAP[providerId] || providerId).trim()
  return normalizedId ? `/logos/${normalizedId}.svg` : ''
}

function normalizePublicProviders(payload: PublicProvidersResponse['data']): PublicProviderInfo[] {
  if (Array.isArray(payload)) {
    return payload
      .filter((item): item is PublicProviderInfo => Boolean(item?.id))
      .map((item) => {
        const normalized: PublicProviderInfo = {
          id: String(item.id),
          label: String(item.label || item.id),
          color: String(item.color || ''),
          logo: resolveProviderLogo(String(item.id), item.logo)
        }
        if (item.default_model) {
          normalized.default_model = item.default_model
        }
        return normalized
      })
  }

  const providers = Array.isArray(payload?.providers) ? payload.providers : []
  return providers.reduce<PublicProviderInfo[]>((acc, item) => {
      const id = String(item.id || item.name || '').trim()
      if (!id) return acc
      const firstModel = Array.isArray(item.models) && item.models.length > 0 ? item.models[0] : undefined
      const normalized: PublicProviderInfo = {
        id,
        label: String(item.label || id),
        color: String(item.color || ''),
        logo: resolveProviderLogo(id, item.logo)
      }
      const defaultModel = item.default_model || firstModel
      if (defaultModel) {
        normalized.default_model = defaultModel
      }
      acc.push(normalized)
      return acc
    }, [])
}

function isProviderType(value: unknown): value is ProviderType {
  if (!value || typeof value !== 'object') return false
  const item = value as Record<string, unknown>
  return (
    typeof item.id === 'string' &&
    typeof item.label === 'string' &&
    typeof item.category === 'string' &&
    PROVIDER_TYPE_CATEGORIES.has(item.category as ProviderType['category']) &&
    typeof item.color === 'string' &&
    typeof item.logo === 'string' &&
    typeof item.default_endpoint === 'string' &&
    typeof item.supports_coding_plan === 'boolean' &&
    Array.isArray(item.models)
  )
}

export async function getProviderTypes(): Promise<ProviderType[]> {
  const response = await request.get<ProviderTypesResponse>('/admin/providers/types')
  if (!response?.success || !Array.isArray(response.data)) {
    throw new Error(response?.error || 'PROVIDER_TYPES_LOAD_FAILED')
  }
  if (!response.data.every(isProviderType)) {
    throw new Error('PROVIDER_TYPES_INVALID_PAYLOAD')
  }
  return response.data
}

export async function getPublicProviders(): Promise<PublicProviderInfo[]> {
  const raw = await request.get<PublicProvidersResponse>('/v1/config/providers')
  if (raw && typeof raw === 'object' && 'success' in raw && (raw as PublicProvidersResponse).success === false) {
    const message = (raw as PublicProvidersResponse).error
    throw new Error(typeof message === 'string' && message ? message : 'PUBLIC_PROVIDERS_LOAD_FAILED')
  }
  const payload = unwrapEnvelope<PublicProvidersResponse['data']>(raw, { allowPlain: true })
  return normalizePublicProviders(payload)
}

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
