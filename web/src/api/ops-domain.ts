import { request } from './request'
import { unwrapEnvelope } from './envelope'

export async function getOpsDashboard(range: string) {
  const raw = await request.get(`/admin/ops/dashboard?range=${encodeURIComponent(range)}`)
  return unwrapEnvelope<any>(raw)
}

export async function getOpsServices() {
  const raw = await request.get('/admin/ops/services')
  return unwrapEnvelope<any[]>(raw)
}

export async function getOpsProviderHealth() {
  const raw = await request.get('/admin/ops/providers/health')
  return unwrapEnvelope<any[]>(raw)
}

export async function getOpsExportMetrics() {
  const raw = await request.get('/admin/ops/export')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}
