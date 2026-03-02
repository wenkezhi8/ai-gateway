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

export interface SettingsDefaultsPayload {
  gateway: Record<string, unknown>
  cache: Record<string, unknown>
  logging: Record<string, unknown>
  security: Record<string, unknown>
}

export async function getSettingsDefaults() {
  const raw = await request.get('/admin/settings/defaults')
  return unwrapEnvelope<SettingsDefaultsPayload>(raw, { allowPlain: true })
}

export async function getUiSettings() {
  const raw = await request.get('/admin/settings/ui')
  return unwrapEnvelope<UiSettingsPayload>(raw, { allowPlain: true })
}

export async function updateUiSettings(payload: UiSettingsPayload) {
  const raw = await request.put('/admin/settings/ui', payload)
  return unwrapEnvelope<UiSettingsPayload>(raw, { allowPlain: true })
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

let pendingPayload: UiSettingsPayload | null = null
let throttleTimer: ReturnType<typeof setTimeout> | null = null
let pendingResolvers: { resolve: (value: UiSettingsPayload) => void; reject: (error: unknown) => void }[] = []

export function updateGeneralUiSettingsThrottled(
  payload: Record<string, unknown>,
  delay = 500
): Promise<UiSettingsPayload> {
  pendingPayload = {
    ...pendingPayload,
    settings: { ...pendingPayload?.settings, ...payload }
  }

  if (throttleTimer) {
    clearTimeout(throttleTimer)
  }

  return new Promise((resolve, reject) => {
    pendingResolvers.push({ resolve, reject })
    throttleTimer = setTimeout(async () => {
      throttleTimer = null
      const currentPayload = pendingPayload
      const currentResolvers = pendingResolvers
      pendingPayload = null
      pendingResolvers = []

      try {
        const result = await updateUiSettings(currentPayload!)
        currentResolvers.forEach(r => r.resolve(result))
      } catch (e) {
        currentResolvers.forEach(r => r.reject(e))
      }
    }, delay)
  })
}

export function flushThrottledSettings(): void {
  if (throttleTimer) {
    clearTimeout(throttleTimer)
    throttleTimer = null
  }
  pendingPayload = null
  pendingResolvers = []
}
