import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface UiSettingsPayload {
  routing?: {
    auto_save_enabled?: boolean
    last_saved_at?: string
  }
  model_management?: {
    provider_defaults_updated_at?: string
  }
  settings?: Record<string, unknown>
}

export async function getUiSettings() {
  const raw = await request.get('/admin/settings/ui')
  return unwrapEnvelope<UiSettingsPayload>(raw)
}

export async function updateUiSettings(payload: UiSettingsPayload) {
  const raw = await request.put('/admin/settings/ui', payload)
  return unwrapEnvelope<UiSettingsPayload>(raw)
}
