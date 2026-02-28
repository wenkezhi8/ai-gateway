import { request } from './request'
import { unwrapEnvelope } from './envelope'

export async function getAlerts() {
  const raw = await request.get('/admin/alerts', { silent: true } as any)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function acknowledgeAlert(alertId: string) {
  const raw = await request.put(`/admin/alerts/${encodeURIComponent(alertId)}/acknowledge`, {})
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function resolveAlert(alertId: string) {
  const raw = await request.put(`/admin/alerts/${encodeURIComponent(alertId)}/resolve`, {})
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function acknowledgeAllAlerts() {
  const raw = await request.post('/admin/alerts/acknowledge-all', {})
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function clearResolvedAlerts() {
  const raw = await request.delete('/admin/alerts/clear-resolved')
  return unwrapEnvelope(raw, { allowPlain: true })
}
