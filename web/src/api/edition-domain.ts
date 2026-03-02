import { request } from './request'
import { unwrapEnvelope } from './envelope'

export type EditionType = 'basic' | 'standard' | 'enterprise'

export interface EditionFeatures {
  vector_cache: boolean
  vector_db_management: boolean
  knowledge_base: boolean
  cold_hot_tiering: boolean
}

export interface EditionConfig {
  type: EditionType
  features: EditionFeatures
  display_name: string
  description: string
  dependencies: string[]
}

export interface DependencyStatus {
  name: string
  address: string
  healthy: boolean
  message: string
}

export interface UpdateEditionResponse {
  restart_required: boolean
  edition: EditionConfig
}

export async function getEditionConfig() {
  const raw = await request.get('/admin/edition')
  return unwrapEnvelope<EditionConfig>(raw, { allowPlain: true })
}

export async function getEditionDefinitions() {
  const raw = await request.get('/admin/edition/definitions')
  return unwrapEnvelope<EditionConfig[]>(raw, { allowPlain: true })
}

export async function checkEditionDependencies() {
  const raw = await request.get('/admin/edition/dependencies')
  return unwrapEnvelope<Record<string, DependencyStatus>>(raw, { allowPlain: true })
}

export async function updateEditionConfig(type: EditionType) {
  const raw = await request.put('/admin/edition', { type })
  return unwrapEnvelope<UpdateEditionResponse>(raw, { allowPlain: true })
}
