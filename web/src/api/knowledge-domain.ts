import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface KnowledgeDocument {
  id: string
  name: string
  type: string
  size: number
  chunk_count: number
  status: 'pending' | 'processing' | 'completed' | 'failed'
  collection_id: string
  created_at: string
  updated_at: string
}

export interface KnowledgeSource {
  chunk_id: string
  document_id: string
  document_name: string
  content: string
  score: number
}

export interface KnowledgeConfig {
  vector_backend: string
  embedding_model: string
  chunking_strategy: {
    type: string
    chunk_size: number
    chunk_overlap: number
  }
  retrieval: {
    top_k: number
    similarity_threshold: number
  }
  collections: Array<{
    id: string
    name: string
    document_count: number
    chunk_count: number
  }>
}

export async function listKnowledgeDocuments(params: {
  page: number
  page_size: number
  status?: string
  search?: string
}) {
  const query = new URLSearchParams()
  query.set('page', String(params.page))
  query.set('page_size', String(params.page_size))
  if (params.status) query.set('status', params.status)
  if (params.search) query.set('search', params.search)
  const raw = await request.get(`/admin/knowledge/documents?${query.toString()}`)
  return unwrapEnvelope<{ total: number; items: KnowledgeDocument[] }>(raw, { allowPlain: true })
}

export async function uploadKnowledgeDocument(file: File, collection = 'default') {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('collection', collection)
  const raw = await request.post('/admin/knowledge/documents/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapEnvelope<{ document_id: string; status: string; message: string }>(raw, { allowPlain: true })
}

export async function getKnowledgeDocument(id: string) {
  const raw = await request.get(`/admin/knowledge/documents/${encodeURIComponent(id)}`)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function deleteKnowledgeDocument(id: string) {
  const raw = await request.delete(`/admin/knowledge/documents/${encodeURIComponent(id)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function vectorizeKnowledgeDocument(id: string) {
  const raw = await request.post(`/admin/knowledge/documents/${encodeURIComponent(id)}/vectorize`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getKnowledgeConfig() {
  const raw = await request.get('/admin/knowledge/config')
  return unwrapEnvelope<KnowledgeConfig>(raw, { allowPlain: true })
}

export async function updateKnowledgeConfig(payload: Partial<KnowledgeConfig>) {
  const raw = await request.put('/admin/knowledge/config', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function sendKnowledgeChatMessage(payload: {
  query: string
  collection_id?: string
  top_k?: number
  similarity_threshold?: number
}) {
  const raw = await request.post('/admin/knowledge/chat/message', payload)
  return unwrapEnvelope<{ answer: string; sources: KnowledgeSource[]; metadata: Record<string, unknown> }>(raw, {
    allowPlain: true
  })
}
