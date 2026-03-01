import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface VectorCollection {
  id: string
  name: string
  description: string
  dimension: number
  distance_metric: string
  index_type: string
  storage_backend: string
  tags: string[]
  environment: string
  status: string
  vector_count: number
  indexed_count: number
  size_bytes: number
  created_at: string
  updated_at: string
  created_by: string
  is_public: boolean
}

export interface ListCollectionsParams {
  search?: string
  environment?: string
  status?: string
  tag?: string
  is_public?: boolean
  offset?: number
  limit?: number
}

export interface CreateCollectionPayload {
  name: string
  description?: string
  dimension: number
  distance_metric?: string
  index_type?: string
  storage_backend?: string
  tags?: string[]
  environment?: string
  status?: string
  created_by?: string
  is_public?: boolean
}

export interface UpdateCollectionPayload {
  description?: string
  distance_metric?: string
  index_type?: string
  storage_backend?: string
  tags?: string[]
  environment?: string
  status?: string
  is_public?: boolean
}

function buildQuery(params: ListCollectionsParams = {}): string {
  const query = new URLSearchParams()
  if (params.search) query.set('search', params.search)
  if (params.environment) query.set('environment', params.environment)
  if (params.status) query.set('status', params.status)
  if (params.tag) query.set('tag', params.tag)
  if (typeof params.is_public === 'boolean') query.set('is_public', String(params.is_public))
  if (typeof params.offset === 'number') query.set('offset', String(params.offset))
  if (typeof params.limit === 'number') query.set('limit', String(params.limit))
  return query.toString()
}

export async function listVectorCollections(params: ListCollectionsParams = {}) {
  const query = buildQuery(params)
  const path = query ? `/admin/vector-db/collections?${query}` : '/admin/vector-db/collections'
  const raw = await request.get(path)
  return unwrapEnvelope<{ collections: VectorCollection[]; total: number }>(raw, { allowPlain: true })
}

export async function getVectorCollection(name: string) {
  const raw = await request.get(`/admin/vector-db/collections/${encodeURIComponent(name)}`)
  return unwrapEnvelope<{ collection: VectorCollection }>(raw, { allowPlain: true })
}

export async function createVectorCollection(payload: CreateCollectionPayload) {
  const raw = await request.post('/admin/vector-db/collections', payload)
  return unwrapEnvelope<VectorCollection>(raw, { allowPlain: true })
}

export async function updateVectorCollection(name: string, payload: UpdateCollectionPayload) {
  const raw = await request.put(`/admin/vector-db/collections/${encodeURIComponent(name)}`, payload)
  return unwrapEnvelope<VectorCollection>(raw, { allowPlain: true })
}

export async function deleteVectorCollection(name: string) {
  const raw = await request.delete(`/admin/vector-db/collections/${encodeURIComponent(name)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}
