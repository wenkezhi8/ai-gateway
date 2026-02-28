import { request } from './request'
import { unwrapEnvelope } from './envelope'

export async function getCacheStats() {
  const raw = await request.get('/admin/cache/stats')
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

export async function getVectorTierStats() {
  const raw = await request.get('/admin/cache/vector/tier/stats')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function triggerVectorTierMigrate() {
  const raw = await request.post('/admin/cache/vector/tier/migrate')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function promoteVectorTierEntry(cacheKey: string) {
  const raw = await request.post('/admin/cache/vector/tier/promote', {
    cache_key: cacheKey
  })
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}
