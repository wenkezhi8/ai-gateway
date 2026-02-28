import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface PublicProviderRecord {
  name: string
  enabled: boolean
  models: string[]
}

export interface AdminProviderConfigRecord {
  value: string
  label: string
  color: string
}

export interface AdminAccountRecord {
  provider: string
  enabled: boolean
}

export async function getPublicProvidersConfig() {
  const raw = await request.get('/v1/config/providers', { silent: true } as any)
  const payload = unwrapEnvelope<{ providers?: PublicProviderRecord[] }>(raw, { allowPlain: true })
  return Array.isArray(payload?.providers) ? payload.providers : []
}

export async function getAdminProviderConfigs() {
  const raw = await request.get('/admin/providers/configs', { silent: true } as any)
  const payload = unwrapEnvelope<AdminProviderConfigRecord[]>(raw, { allowPlain: true })
  return Array.isArray(payload) ? payload : []
}

export async function getAdminAccounts() {
  const raw = await request.get('/admin/accounts', { silent: true } as any)
  const payload = unwrapEnvelope<AdminAccountRecord[]>(raw, { allowPlain: true })
  return Array.isArray(payload) ? payload : []
}
