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
  runtime: EditionSetupRuntime
  dependency_versions: Record<string, string>
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

export type EditionSetupRuntime = 'docker' | 'native'
export type EditionSetupStatus = 'pending' | 'running' | 'success' | 'failed'

export interface EditionSetupRequest {
  edition: EditionType
  runtime: EditionSetupRuntime
  apply_config: boolean
  pull_embedding_model: boolean
}

export interface EditionSetupTaskCreateResponse {
  task_id: string
  accepted_at: string
  message: string
}

export interface EditionSetupTask {
  task_id: string
  edition: EditionType
  runtime: EditionSetupRuntime
  status: EditionSetupStatus
  accepted_at: string
  started_at?: string
  finished_at?: string
  summary: string
  logs: string
  health: Record<string, DependencyStatus>
  message?: string
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

export async function setupEditionEnvironment(payload: EditionSetupRequest) {
  const raw = await request.post('/admin/edition/setup', payload)
  return unwrapEnvelope<EditionSetupTaskCreateResponse>(raw, { allowPlain: true })
}

export async function getEditionSetupTask(taskId: string) {
  const raw = await request.get(`/admin/edition/setup/tasks/${encodeURIComponent(taskId)}`)
  return unwrapEnvelope<EditionSetupTask>(raw, { allowPlain: true })
}
