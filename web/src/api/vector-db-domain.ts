import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface VectorCollection {
  id: string
  name: string
  description: string
  dimension: number
  distance_metric: string
  index_type: string
  hnsw_m: number
  hnsw_ef_construct: number
  ivf_nlist: number
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
  hnsw_m?: number
  hnsw_ef_construct?: number
  ivf_nlist?: number
  storage_backend?: string
  tags?: string[]
  environment?: string
  status?: string
  is_public?: boolean
}

export type ImportJobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'retrying'

export interface ImportJob {
  id: string
  collection_id: string
  collection_name?: string
  file_name: string
  file_path: string
  file_size: number
  total_records: number
  processed_records: number
  failed_records: number
  retry_count: number
  max_retries: number
  status: ImportJobStatus
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
  created_by: string
}

export interface ImportJobErrorLog {
  id: number
  user_id: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  ip_address?: string
  created_at: string
}

export interface AuditLogItem {
  id: number
  user_id: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  created_at: string
}

export interface ListAuditLogsParams {
  resource_type?: string
  resource_id?: string
  action?: string
  limit?: number
  offset?: number
}

export interface ImportJobSummary {
  pending: number
  running: number
  retrying: number
  completed: number
  failed: number
  cancelled: number
  total: number
}

export interface CreateImportJobPayload {
  collection_name: string
  file_name: string
  file_path: string
  file_size: number
  total_records: number
  max_retries?: number
  created_by?: string
}

export interface UpdateImportJobStatusPayload {
  status: ImportJobStatus
  processed_records?: number
  failed_records?: number
  retry_count?: number
  error_message?: string
  started_at?: string
  completed_at?: string
}

export interface ListImportJobsParams {
  collection_name?: string
  status?: ImportJobStatus
  offset?: number
  limit?: number
}

export interface VectorSearchPayload {
  top_k: number
  min_score?: number
  vector: number[]
  text?: string
}

export interface VectorSearchResult {
  id: string
  score: number
  payload: Record<string, unknown>
}

export interface VectorSearchResponse {
  results: VectorSearchResult[]
  total: number
}

export interface AlertRule {
  id: number
  name: string
  metric: string
  operator: string
  threshold: number
  duration: string
  channels: string[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateAlertRulePayload {
  name: string
  metric: string
  operator: string
  threshold: number
  duration: string
  channels?: string[]
  enabled?: boolean
}

export interface UpdateAlertRulePayload {
  name?: string
  metric?: string
  operator?: string
  threshold?: number
  duration?: string
  channels?: string[]
  enabled?: boolean
}

export interface VectorMetricsSummary {
  collections_total: number
  import_jobs: ImportJobSummary
  alert_rules_total: number
  enabled_rules: number
}

export interface VectorPermissionItem {
  id: number
  role: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateVectorPermissionPayload {
  api_key: string
  role: string
}

export type BackupAction = 'backup' | 'restore'
export type BackupStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface BackupTask {
  id: number
  collection_name: string
  snapshot_name: string
  action: BackupAction
  status: BackupStatus
  source_backup_id: number
  error_message?: string
  created_at: string
  updated_at: string
  started_at?: string
  completed_at?: string
  created_by: string
}

export interface VectorScatterPoint {
  id: string
  x: number
  y: number
  label: string
  score: number
}

export interface VectorScatterResponse {
  points: VectorScatterPoint[]
  total: number
}

export interface CreateBackupTaskPayload {
  collection_name: string
  snapshot_name?: string
  created_by?: string
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

function buildImportJobQuery(params: ListImportJobsParams = {}): string {
  const query = new URLSearchParams()
  if (params.collection_name) query.set('collection_name', params.collection_name)
  if (params.status) query.set('status', params.status)
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

export async function emptyVectorCollection(name: string) {
  const raw = await request.post(`/admin/vector-db/collections/${encodeURIComponent(name)}/empty`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function createImportJob(payload: CreateImportJobPayload) {
  const raw = await request.post('/admin/vector-db/import-jobs', payload)
  return unwrapEnvelope<ImportJob>(raw, { allowPlain: true })
}

export async function listImportJobs(params: ListImportJobsParams = {}) {
  const query = buildImportJobQuery(params)
  const path = query ? `/admin/vector-db/import-jobs?${query}` : '/admin/vector-db/import-jobs'
  const raw = await request.get(path)
  return unwrapEnvelope<{ jobs: ImportJob[]; total: number }>(raw, { allowPlain: true })
}

export async function getImportJob(id: string) {
  const raw = await request.get(`/admin/vector-db/import-jobs/${encodeURIComponent(id)}`)
  return unwrapEnvelope<ImportJob>(raw, { allowPlain: true })
}

export async function updateImportJobStatus(id: string, payload: UpdateImportJobStatusPayload) {
  const raw = await request.put(`/admin/vector-db/import-jobs/${encodeURIComponent(id)}/status`, payload)
  return unwrapEnvelope<ImportJob>(raw, { allowPlain: true })
}

export async function runImportJob(id: string) {
  const raw = await request.post(`/admin/vector-db/import-jobs/${encodeURIComponent(id)}/run`)
  return unwrapEnvelope<ImportJob>(raw, { allowPlain: true })
}

export async function retryImportJob(id: string) {
  const raw = await request.post(`/admin/vector-db/import-jobs/${encodeURIComponent(id)}/retry`)
  return unwrapEnvelope<ImportJob>(raw, { allowPlain: true })
}

export async function retryFailedImportJobs(limit = 20) {
  const raw = await request.post(`/admin/vector-db/import-jobs/retry-failed?limit=${encodeURIComponent(String(limit))}`)
  return unwrapEnvelope<{ jobs: ImportJob[]; total: number }>(raw, { allowPlain: true })
}

export async function getImportJobErrors(id: string, limit = 20, action?: string, offset = 0) {
  const query = new URLSearchParams()
  query.set('limit', String(limit))
  query.set('offset', String(offset))
  if (action) query.set('action', action)
  const raw = await request.get(`/admin/vector-db/import-jobs/${encodeURIComponent(id)}/errors?${query.toString()}`)
  return unwrapEnvelope<{ logs: ImportJobErrorLog[]; total: number }>(raw, { allowPlain: true })
}

export async function getImportJobSummary(collectionName?: string) {
  const query = new URLSearchParams()
  if (collectionName) query.set('collection_name', collectionName)
  const suffix = query.toString() ? `?${query.toString()}` : ''
  const raw = await request.get(`/admin/vector-db/import-jobs/summary${suffix}`)
  return unwrapEnvelope<ImportJobSummary>(raw, { allowPlain: true })
}

export async function listVectorAuditLogs(params: ListAuditLogsParams = {}) {
  const query = new URLSearchParams()
  if (params.resource_type) query.set('resource_type', params.resource_type)
  if (params.resource_id) query.set('resource_id', params.resource_id)
  if (params.action) query.set('action', params.action)
  if (typeof params.limit === 'number') query.set('limit', String(params.limit))
  if (typeof params.offset === 'number') query.set('offset', String(params.offset))
  const suffix = query.toString() ? `?${query.toString()}` : ''
  const raw = await request.get(`/admin/vector-db/audit/logs${suffix}`)
  return unwrapEnvelope<{ items: AuditLogItem[]; total: number }>(raw, { allowPlain: true })
}

export async function searchVectorCollection(collectionName: string, payload: VectorSearchPayload) {
  const raw = await request.post(`/v1/vector/collections/${encodeURIComponent(collectionName)}/search`, payload)
  return unwrapEnvelope<VectorSearchResponse>(raw, { allowPlain: true })
}

export async function recommendVectorCollection(collectionName: string, payload: VectorSearchPayload) {
  const raw = await request.post(`/v1/vector/collections/${encodeURIComponent(collectionName)}/recommend`, payload)
  return unwrapEnvelope<VectorSearchResponse>(raw, { allowPlain: true })
}

export async function getVectorByID(collectionName: string, id: string) {
  const raw = await request.get(`/v1/vector/collections/${encodeURIComponent(collectionName)}/vectors/${encodeURIComponent(id)}`)
  return unwrapEnvelope<VectorSearchResult>(raw, { allowPlain: true })
}

export async function getVectorMetricsSummary() {
  const raw = await request.get('/admin/vector-db/metrics/summary')
  return unwrapEnvelope<VectorMetricsSummary>(raw, { allowPlain: true })
}

export async function listAlertRules() {
  const raw = await request.get('/admin/vector-db/alerts/rules')
  return unwrapEnvelope<{ rules: AlertRule[]; total: number }>(raw, { allowPlain: true })
}

export async function createAlertRule(payload: CreateAlertRulePayload) {
  const raw = await request.post('/admin/vector-db/alerts/rules', payload)
  return unwrapEnvelope<AlertRule>(raw, { allowPlain: true })
}

export async function updateAlertRule(id: number, payload: UpdateAlertRulePayload) {
  const raw = await request.put(`/admin/vector-db/alerts/rules/${encodeURIComponent(String(id))}`, payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteAlertRule(id: number) {
  const raw = await request.delete(`/admin/vector-db/alerts/rules/${encodeURIComponent(String(id))}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function listVectorPermissions() {
  const raw = await request.get('/admin/vector-db/permissions')
  return unwrapEnvelope<{ items: VectorPermissionItem[]; total: number }>(raw, { allowPlain: true })
}

export async function createVectorPermission(payload: CreateVectorPermissionPayload) {
  const raw = await request.post('/admin/vector-db/permissions', payload)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteVectorPermission(id: number) {
  const raw = await request.delete(`/admin/vector-db/permissions/${encodeURIComponent(String(id))}`)
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function listBackupTasks(params: {
  collection_name?: string
  action?: BackupAction
  status?: BackupStatus
  offset?: number
  limit?: number
} = {}) {
  const query = new URLSearchParams()
  if (params.collection_name) query.set('collection_name', params.collection_name)
  if (params.action) query.set('action', params.action)
  if (params.status) query.set('status', params.status)
  if (typeof params.offset === 'number') query.set('offset', String(params.offset))
  if (typeof params.limit === 'number') query.set('limit', String(params.limit))
  const suffix = query.toString() ? `?${query.toString()}` : ''
  const raw = await request.get(`/admin/vector-db/backups${suffix}`)
  return unwrapEnvelope<{ items: BackupTask[]; total: number }>(raw, { allowPlain: true })
}

export async function createBackupTask(payload: CreateBackupTaskPayload) {
  const raw = await request.post('/admin/vector-db/backups', payload)
  return unwrapEnvelope<BackupTask>(raw, { allowPlain: true })
}

export async function triggerBackupRestore(id: number) {
  const raw = await request.post(`/admin/vector-db/backups/${encodeURIComponent(String(id))}/restore`)
  return unwrapEnvelope<BackupTask>(raw, { allowPlain: true })
}

export async function retryBackupTask(id: number) {
  const raw = await request.post(`/admin/vector-db/backups/${encodeURIComponent(String(id))}/retry`)
  return unwrapEnvelope<BackupTask>(raw, { allowPlain: true })
}

export async function runBackupPolicy(payload: {
  collection_name: string
  retention_count?: number
  created_by?: string
}) {
  const raw = await request.post('/admin/vector-db/backups/policy/run', payload)
  return unwrapEnvelope<{ created_task: BackupTask; deleted_count: number }>(raw, { allowPlain: true })
}

export async function getVectorScatterData(collectionName: string, sampleSize = 200) {
  const query = new URLSearchParams()
  query.set('collection_name', collectionName)
  query.set('sample_size', String(sampleSize))
  const raw = await request.get(`/admin/vector-db/visualization/scatter?${query.toString()}`)
  return unwrapEnvelope<VectorScatterResponse>(raw, { allowPlain: true })
}
