import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface UiSettingsPayload {
  routing?: {
    auto_save_enabled?: boolean
    last_saved_at?: string
  }
  model_management?: {
    last_saved_at?: string
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

export async function updateRoutingUiSettings(payload: NonNullable<UiSettingsPayload['routing']>) {
  return updateUiSettings({ routing: payload })
}

export async function updateModelManagementUiSettings(payload: NonNullable<UiSettingsPayload['model_management']>) {
  return updateUiSettings({ model_management: payload })
}

export async function updateGeneralUiSettings(payload: Record<string, unknown>) {
  return updateUiSettings({ settings: payload })
}
