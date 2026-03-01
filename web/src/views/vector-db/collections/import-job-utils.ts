import type { ImportJob, ImportJobErrorLog, ImportJobStatus } from '@/api/vector-db-domain'

export interface ImportJobSummary {
  pending: number
  running: number
  retrying: number
  completed: number
  failed: number
  total: number
}

export interface ImportJobErrorGroup {
  date: string
  items: ImportJobErrorLog[]
}

export interface ImportJobErrorDateRange {
  startDate: string
  endDate: string
}

export function mergeImportJobErrorLogs(existing: ImportJobErrorLog[], incoming: ImportJobErrorLog[]): ImportJobErrorLog[] {
  const seen = new Set<number>()
  const merged: ImportJobErrorLog[] = []

  for (const item of existing || []) {
    if (!seen.has(item.id)) {
      seen.add(item.id)
      merged.push(item)
    }
  }

  for (const item of incoming || []) {
    if (!seen.has(item.id)) {
      seen.add(item.id)
      merged.push(item)
    }
  }

  return merged
}

export function buildImportJobErrorSummaryText(logs: ImportJobErrorLog[]): string {
  if (!logs || logs.length === 0) {
    return '暂无错误记录'
  }
  const lines: string[] = []
  for (const item of logs) {
    lines.push(`[${item.created_at}] ${item.action} - ${item.details || '-'}`)
  }
  return lines.join('\n')
}

export function buildImportJobErrorExportFileName(jobId: string, action?: string): string {
  const id = (jobId || 'unknown').trim() || 'unknown'
  const actionPart = action?.trim() ? `-${action.trim()}` : ''
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  const hh = String(now.getHours()).padStart(2, '0')
  const mm = String(now.getMinutes()).padStart(2, '0')
  const ss = String(now.getSeconds()).padStart(2, '0')
  return `import-job-${id}${actionPart}-${y}${m}${d}-${hh}${mm}${ss}.txt`
}

export function normalizeImportJobErrorAction(action?: string): string {
  const value = (action || '').trim()
  if (!value) {
    return 'all'
  }
  if (value === 'import_run_failed') return value
  if (value === 'import_upsert_failed') return value
  if (value === 'import_retry_exceeded') return value
  return 'all'
}

export function canRetryImportJob(job: ImportJob): boolean {
  const maxRetries = Number(job.max_retries || 0)
  const retryCount = Number(job.retry_count || 0)
  if (maxRetries <= 0) {
    return true
  }
  return retryCount < maxRetries
}

export function normalizeImportJobStatus(value: string): ImportJobStatus | undefined {
  const status = (value || '').trim() as ImportJobStatus
  if (!status) {
    return undefined
  }
  const allowed: ImportJobStatus[] = ['pending', 'running', 'completed', 'failed', 'cancelled', 'retrying']
  if (!allowed.includes(status)) {
    return undefined
  }
  return status
}

export function summarizeImportJobs(jobs: ImportJob[]): ImportJobSummary {
  const summary: ImportJobSummary = {
    pending: 0,
    running: 0,
    retrying: 0,
    completed: 0,
    failed: 0,
    total: 0
  }

  for (const item of jobs) {
    summary.total += 1
    if (item.status === 'pending') summary.pending += 1
    else if (item.status === 'running') summary.running += 1
    else if (item.status === 'retrying') summary.retrying += 1
    else if (item.status === 'completed') summary.completed += 1
    else if (item.status === 'failed') summary.failed += 1
  }

  return summary
}

export function resolveLastRunAt(job: ImportJob): string {
  return job.completed_at || job.started_at || job.updated_at || job.created_at
}

export function buildRetryHint(job: ImportJob): string {
  const retryCount = Number(job.retry_count || 0)
  const maxRetries = Number(job.max_retries || 0)

  if (maxRetries <= 0) {
    return '未设置最大重试限制'
  }

  if (retryCount >= maxRetries) {
    return `已达重试上限（${retryCount}/${maxRetries}），请检查源文件或参数后新建任务`
  }

  if (job.status === 'failed') {
    return `可重试（${retryCount}/${maxRetries}），建议先查看错误摘要`
  }

  if (job.status === 'retrying') {
    return `正在重试（${retryCount}/${maxRetries}）`
  }

  return `当前重试进度：${retryCount}/${maxRetries}`
}

export function groupImportJobErrorsByDate(logs: ImportJobErrorLog[]): ImportJobErrorGroup[] {
  const buckets = new Map<string, ImportJobErrorLog[]>()

  for (const item of logs || []) {
    const key = normalizeDateKey(item.created_at)
    const current = buckets.get(key) || []
    current.push(item)
    buckets.set(key, current)
  }

  return [...buckets.entries()]
    .sort((a, b) => b[0].localeCompare(a[0]))
    .map(([date, items]) => ({ date, items }))
}

export function resolveImportJobErrorDateRange(groups: ImportJobErrorGroup[]): ImportJobErrorDateRange | undefined {
  if (!groups || groups.length === 0) {
    return undefined
  }
  const dates = groups.map(item => item.date).filter(Boolean)
  if (dates.length === 0) {
    return undefined
  }
  const sorted = [...dates].sort((a, b) => a.localeCompare(b))
  return {
    startDate: sorted[0]!,
    endDate: sorted[sorted.length - 1]!
  }
}

export function filterImportJobErrorsByDateRange(logs: ImportJobErrorLog[], range?: ImportJobErrorDateRange): ImportJobErrorLog[] {
  if (!range) {
    return logs || []
  }
  const { startDate, endDate } = range
  return (logs || []).filter(item => {
    const key = normalizeDateKey(item.created_at)
    return key >= startDate && key <= endDate
  })
}

function normalizeDateKey(value: string): string {
  const raw = (value || '').trim()
  if (!raw) {
    return '未知日期'
  }
  if (/^\d{4}-\d{2}-\d{2}/.test(raw)) {
    return raw.slice(0, 10)
  }
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return raw.length >= 10 ? raw.slice(0, 10) : raw
  }

  const y = parsed.getFullYear()
  const m = String(parsed.getMonth() + 1).padStart(2, '0')
  const d = String(parsed.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}
