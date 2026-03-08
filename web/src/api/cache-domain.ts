import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface CacheTaskTypeConfig {
  key: string
  label: string
  description: string
  default_ttl: number
  ttl_unit: 'hours'
}

export interface CacheModelOptionGroup {
  provider_id: string
  provider_label: string
  models: string[]
}

export type CacheRequestSource = 'all' | 'exact_raw' | 'exact_prompt' | 'semantic' | 'v2' | 'provider_chat'

export interface CacheRequestQuery {
  window?: string
  start_time?: string
  end_time?: string
  start?: string
  end?: string
  source?: CacheRequestSource
}

export interface CacheRequestHitsQuery extends CacheRequestQuery {
  page?: number
  page_size?: number
  task_type?: string
  search?: string
  aggregate?: boolean | '1' | '0'
  readable_only?: boolean | '1' | '0'
}

interface CacheTaskTTLConfigResponse {
  success: boolean
  data?: {
    task_types: CacheTaskTypeConfig[]
    model_options: CacheModelOptionGroup[]
  }
  error?: string
}

export function isCacheTaskTTLNotFoundError(error: unknown): boolean {
  const status = (error as { response?: { status?: number } })?.response?.status
  return status === 404
}

export async function getCacheTaskTTLConfig() {
  const response = await request.get<CacheTaskTTLConfigResponse>('/admin/cache/task-ttl')
  if (!response?.success || !response.data) {
    throw new Error(response?.error || 'CACHE_TTL_CONFIG_LOAD_FAILED')
  }
  return response.data
}

export async function getCacheStats() {
  const raw = await request.get('/admin/cache/stats')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

function buildCacheRequestQuery(query: CacheRequestQuery | CacheRequestHitsQuery = {}) {
  const params = new URLSearchParams()

  const source = typeof query.source === 'string' && query.source ? query.source : 'all'
  const startTime = query.start_time || query.start
  const endTime = query.end_time || query.end
  const hasExplicitRange = Boolean(startTime || endTime)
  if (hasExplicitRange) {
    if (startTime) params.append('start_time', String(startTime))
    if (endTime) params.append('end_time', String(endTime))
  } else {
    params.append('window', String(query.window || '24h'))
  }
  params.append('source', source)

  for (const [key, value] of Object.entries(query as Record<string, unknown>)) {
    if (key === 'window' || key === 'start_time' || key === 'end_time' || key === 'start' || key === 'end' || key === 'source') {
      continue
    }
    if (value === undefined || value === null || value === '') {
      continue
    }
    if (typeof value === 'boolean') {
      params.append(key, value ? '1' : '0')
      continue
    }
    params.append(key, String(value))
  }

  return params.toString()
}

export async function getCacheRequestStats(query: CacheRequestQuery = {}) {
  const raw = await request.get(`/admin/cache/request-stats?${buildCacheRequestQuery(query)}`)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getCacheRequestHits(query: CacheRequestHitsQuery = {}) {
  const raw = await request.get(`/admin/cache/request-hits?${buildCacheRequestQuery(query)}`)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getCacheConfig() {
  const raw = await request.get('/admin/cache/config')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function updateCacheConfig(payload: Record<string, unknown>) {
  const raw = await request.put('/admin/cache/config', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getCacheHealth() {
  const raw = await request.get('/admin/cache/health')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function clearCacheByType(cacheType: string) {
  const raw = await request.delete(`/admin/cache?type=${encodeURIComponent(cacheType)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getCacheRules() {
  const raw = await request.get('/admin/cache/rules')
  return unwrapEnvelope<any[]>(raw)
}

export async function createCacheRule(payload: Record<string, unknown>) {
  const raw = await request.post('/admin/cache/rules', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function updateCacheRule(ruleId: number, payload: Record<string, unknown>) {
  const raw = await request.put(`/admin/cache/rules/${ruleId}`, payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteCacheRule(ruleId: number) {
  const raw = await request.delete(`/admin/cache/rules/${ruleId}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getSemanticSignatures(limit = 12) {
  const raw = await request.get(`/admin/cache/semantic-signatures?limit=${limit}`)
  return unwrapEnvelope<any[]>(raw)
}

export async function getTtlConfig() {
  const raw = await request.get('/admin/router/ttl-config')
  return unwrapEnvelope<any>(raw)
}

export async function updateTtlConfig(payload: Record<string, unknown>) {
  const raw = await request.put('/admin/router/ttl-config', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getCacheEntries(query: string) {
  const raw = await request.get(`/admin/cache/entries?${query}`)
  return unwrapEnvelope<any>(raw)
}

export async function getCacheEntryDetail(key: string) {
  const raw = await request.get(`/admin/cache/entries/${encodeURIComponent(key)}`)
  return unwrapEnvelope<any>(raw)
}

export async function deleteCacheEntry(key: string) {
  const raw = await request.delete(`/admin/cache/entries/${encodeURIComponent(key)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteCacheEntryGroup(payload: Record<string, unknown>) {
  const raw = await request.post('/admin/cache/entries/delete-group', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function cleanupInvalidEntries() {
  const raw = await request.post('/admin/cache/entries/cleanup-invalid')
  return unwrapEnvelope<any>(raw)
}

export async function addTestCacheEntry(payload: Record<string, unknown>) {
  const raw = await request.post('/admin/cache/test-entry', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getVectorStats() {
  const raw = await request.get('/admin/cache/vector/stats')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function rebuildVectorIndex() {
  const raw = await request.post('/admin/cache/vector/rebuild')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getVectorPipelineHealth() {
  const raw = await request.get('/admin/cache/vector/pipeline/health')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function testVectorPipeline(payload: Record<string, unknown>) {
  const raw = await request.post('/admin/cache/vector/pipeline/test', payload)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}
