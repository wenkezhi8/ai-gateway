import { request } from './request'
import { unwrapEnvelope } from './envelope'
import type { UseAutoModeContract } from '@/constants/router-mode'

export type OllamaStartupMode = 'auto' | 'app' | 'cli' | 'manual'

export interface OllamaRuntimeMonitoringConfig {
  enabled: boolean
  check_interval_seconds: number
  auto_restart: boolean
  max_restart_attempts: number
  restart_cooldown_seconds: number
}

export type OllamaPreloadTarget = 'intent' | 'embedding'

export interface OllamaRuntimePreloadConfig {
  auto_on_startup: boolean
  targets: OllamaPreloadTarget[]
  timeout_seconds: number
}

export interface OllamaRuntimeConfig {
  startup_mode: OllamaStartupMode
  auto_detect_priority: OllamaStartupMode[]
  preload: OllamaRuntimePreloadConfig
  monitoring: OllamaRuntimeMonitoringConfig
  startup_timeout_seconds: number
  health_check_timeout_ms: number
}

export interface OllamaPreloadResult {
  kind: OllamaPreloadTarget
  model: string
  status: 'success' | 'failed'
  duration_ms: number
  error?: string
}

export interface OllamaPreloadResponse {
  results: OllamaPreloadResult[]
  total: number
  success_count: number
}

export interface OllamaMonitoringStats {
  enabled: boolean
  health_status: string
  last_check_time: string
  restart_attempts: number
  last_restart_time: string
  last_error: string
}

export interface ModelRegistryItem {
  model: string
  provider: string
  display_name?: string
  enabled: boolean
}

export interface UpsertModelRegistryPayload {
  provider: string
  display_name?: string
  enabled?: boolean
}

export interface RouterConfigResponseData {
  use_auto_mode: string | boolean
  default_strategy: string
  default_model: string
  classifier?: Record<string, unknown>
  strategies?: Array<{ value: string; label: string; description: string }>
  use_auto_mode_contract?: UseAutoModeContract
  migration_notice?: string
}

export interface RouterModeMigration {
  from: string
  to: string
}

export interface UpdateRouterConfigResponseData {
  success: boolean
  message?: string
  use_auto_mode?: string
  use_auto_mode_contract?: UseAutoModeContract
  migration_notice?: string
  mode_migration?: RouterModeMigration
}

export async function getRouterConfig() {
  const raw = await request.get('/admin/router/config')
  return unwrapEnvelope<RouterConfigResponseData>(raw)
}

export async function getModelRegistry() {
  const raw = await request.get('/admin/router/model-registry')
  return unwrapEnvelope<ModelRegistryItem[]>(raw, { allowPlain: true })
}

export async function getAvailableModels() {
  const raw = await request.get('/admin/router/available-models?format=object')
  return unwrapEnvelope<any[]>(raw)
}

export async function getTopModels() {
  const raw = await request.get('/admin/router/top-models')
  return unwrapEnvelope<string[]>(raw)
}

export async function getProviderDefaults() {
  const raw = await request.get('/admin/router/provider-defaults')
  return unwrapEnvelope<Record<string, string>>(raw)
}

export async function getCascadeRules() {
  const raw = await request.get('/admin/router/cascade-rules')
  return unwrapEnvelope<any[]>(raw, { allowPlain: true })
}

export async function getTaskModelMapping() {
  const raw = await request.get('/admin/router/task-model-mapping')
  return unwrapEnvelope<Record<string, string>>(raw)
}

export async function putTaskModelMapping(data: Record<string, string>) {
  const raw = await request.put('/admin/router/task-model-mapping', data)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function updateRouterConfig(data: Record<string, unknown>) {
  return request.put('/admin/router/config', data) as Promise<UpdateRouterConfigResponseData>
}

export async function upsertModelRegistry(model: string, payload: UpsertModelRegistryPayload) {
  const raw = await request.put(`/admin/router/model-registry/${encodeURIComponent(model)}`, payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteModelRegistry(model: string) {
  const raw = await request.delete(`/admin/router/model-registry/${encodeURIComponent(model)}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getClassifierHealth() {
  const raw = await request.get('/admin/router/classifier/health')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getClassifierStats() {
  const raw = await request.get('/admin/router/classifier/stats')
  return unwrapEnvelope<any>(raw)
}

export async function getClassifierModels() {
  const raw = await request.get('/admin/router/classifier/models')
  return unwrapEnvelope<any>(raw)
}

export async function switchClassifierModelAsync(model: string) {
  const raw = await request.post('/admin/router/classifier/switch-async', { model })
  return unwrapEnvelope<{ task_id: string; taskId?: string }>(raw, { allowPlain: true })
}

export async function getClassifierSwitchTask(taskPath: string) {
  const raw = await request.get(taskPath)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getOllamaDualModelConfig() {
  const raw = await request.get('/admin/router/ollama/dual-model/config')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function updateOllamaDualModelConfig(payload: Record<string, unknown>) {
  const raw = await request.put('/admin/router/ollama/dual-model/config', payload)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getOllamaStatus(model: string) {
  const raw = await request.get(`/admin/router/ollama/status?model=${encodeURIComponent(model)}`)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getOllamaRuntimeConfig() {
  const raw = await request.get('/admin/router/ollama/runtime-config')
  return unwrapEnvelope<{ config: OllamaRuntimeConfig; monitoring_stats: OllamaMonitoringStats }>(raw, { allowPlain: true })
}

export async function updateOllamaRuntimeConfig(payload: Partial<OllamaRuntimeConfig>) {
  const raw = await request.put('/admin/router/ollama/runtime-config', payload)
  return unwrapEnvelope<OllamaRuntimeConfig>(raw, { allowPlain: true })
}

export async function preloadOllamaModels(payload?: { targets?: OllamaPreloadTarget[] }) {
  const raw = await request.post('/admin/router/ollama/preload', payload || {})
  return unwrapEnvelope<OllamaPreloadResponse>(raw, { allowPlain: true })
}

export async function installOllama() {
  const raw = await request.post('/admin/router/ollama/install')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function startOllama() {
  const raw = await request.post('/admin/router/ollama/start')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function stopOllama() {
  const raw = await request.post('/admin/router/ollama/stop')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function pullOllamaModel(model: string) {
  const raw = await request.post('/admin/router/ollama/pull', { model })
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteOllamaModel(model: string) {
  const raw = await request.post('/admin/router/ollama/delete', { model })
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function getFeedbackStats() {
  const raw = await request.get('/admin/feedback/stats')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function triggerFeedbackOptimization() {
  const raw = await request.post('/admin/feedback/optimize')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getTaskTypeDistribution(options?: { refresh?: boolean }) {
	const refresh = options?.refresh
	const raw = refresh
		? await request.get('/admin/feedback/task-type-distribution', { params: { refresh: 'true' } })
		: await request.get('/admin/feedback/task-type-distribution')
	return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getVectorTierConfig() {
  const raw = await request.get('/admin/router/vector/tier/config')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function updateVectorTierConfig(payload: Record<string, unknown>) {
  const raw = await request.put('/admin/router/vector/tier/config', payload)
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function getVectorTierStats() {
  const raw = await request.get('/admin/router/vector/tier/stats')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function triggerVectorTierMigrate() {
  const raw = await request.post('/admin/router/vector/tier/migrate')
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}

export async function promoteVectorTierEntry(cacheKey: string) {
  const raw = await request.post('/admin/router/vector/tier/promote', {
    cache_key: cacheKey
  })
  return unwrapEnvelope<any>(raw, { allowPlain: true })
}
