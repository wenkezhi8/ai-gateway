import { describe, expect, it } from 'vitest'

import {
  buildRetryHint,
  buildImportJobErrorSummaryText,
  buildImportJobErrorExportFileName,
  canRetryImportJob,
  filterImportJobErrorsByDateRange,
  groupImportJobErrorsByDate,
  mergeImportJobErrorLogs,
  normalizeImportJobErrorAction,
  normalizeImportJobStatus,
  resolveImportJobErrorDateRange,
  resolveLastRunAt,
  summarizeImportJobs
} from './import-job-utils'

describe('import job utils', () => {
  it('should allow retry when retry_count below max_retries', () => {
    const ok = canRetryImportJob({
      id: 'job_1',
      collection_id: 'col_1',
      file_name: 'docs.json',
      file_path: '/tmp/docs.json',
      file_size: 100,
      total_records: 10,
      processed_records: 1,
      failed_records: 9,
      retry_count: 1,
      max_retries: 3,
      status: 'failed',
      created_at: '',
      updated_at: '',
      created_by: 'tester'
    })

    expect(ok).toBe(true)
  })

  it('should block retry when retry_count reached max_retries', () => {
    const ok = canRetryImportJob({
      id: 'job_2',
      collection_id: 'col_1',
      file_name: 'docs.json',
      file_path: '/tmp/docs.json',
      file_size: 100,
      total_records: 10,
      processed_records: 1,
      failed_records: 9,
      retry_count: 3,
      max_retries: 3,
      status: 'failed',
      created_at: '',
      updated_at: '',
      created_by: 'tester'
    })

    expect(ok).toBe(false)
  })

  it('should normalize valid status and reject invalid status', () => {
    expect(normalizeImportJobStatus('failed')).toBe('failed')
    expect(normalizeImportJobStatus('unknown')).toBeUndefined()
    expect(normalizeImportJobStatus('')).toBeUndefined()
  })

  it('should summarize import jobs by status', () => {
    const summary = summarizeImportJobs([
      {
        id: 'job_1',
        collection_id: 'col_1',
        file_name: 'a.json',
        file_path: '/tmp/a.json',
        file_size: 10,
        total_records: 1,
        processed_records: 0,
        failed_records: 0,
        retry_count: 0,
        max_retries: 3,
        status: 'pending',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      },
      {
        id: 'job_2',
        collection_id: 'col_1',
        file_name: 'b.json',
        file_path: '/tmp/b.json',
        file_size: 10,
        total_records: 1,
        processed_records: 1,
        failed_records: 0,
        retry_count: 0,
        max_retries: 3,
        status: 'running',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      },
      {
        id: 'job_3',
        collection_id: 'col_1',
        file_name: 'c.json',
        file_path: '/tmp/c.json',
        file_size: 10,
        total_records: 1,
        processed_records: 1,
        failed_records: 0,
        retry_count: 1,
        max_retries: 3,
        status: 'retrying',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      },
      {
        id: 'job_4',
        collection_id: 'col_1',
        file_name: 'd.json',
        file_path: '/tmp/d.json',
        file_size: 10,
        total_records: 1,
        processed_records: 1,
        failed_records: 0,
        retry_count: 0,
        max_retries: 3,
        status: 'completed',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      },
      {
        id: 'job_5',
        collection_id: 'col_1',
        file_name: 'e.json',
        file_path: '/tmp/e.json',
        file_size: 10,
        total_records: 1,
        processed_records: 0,
        failed_records: 1,
        retry_count: 2,
        max_retries: 3,
        status: 'failed',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      }
    ])

    expect(summary.total).toBe(5)
    expect(summary.pending).toBe(1)
    expect(summary.running).toBe(1)
    expect(summary.retrying).toBe(1)
    expect(summary.completed).toBe(1)
    expect(summary.failed).toBe(1)
  })

  it('should resolve last run time with fallback order', () => {
    expect(
      resolveLastRunAt({
        id: 'job_1',
        collection_id: 'col_1',
        file_name: 'a.json',
        file_path: '/tmp/a.json',
        file_size: 10,
        total_records: 1,
        processed_records: 1,
        failed_records: 0,
        retry_count: 0,
        max_retries: 3,
        status: 'completed',
        completed_at: '2026-03-02T10:00:00Z',
        started_at: '2026-03-02T09:00:00Z',
        created_at: '2026-03-02T08:00:00Z',
        updated_at: '2026-03-02T10:00:01Z',
        created_by: 'tester'
      })
    ).toBe('2026-03-02T10:00:00Z')
  })

  it('should build retry hints by status and retry limit', () => {
    expect(
      buildRetryHint({
        id: 'job_1',
        collection_id: 'col_1',
        file_name: 'a.json',
        file_path: '/tmp/a.json',
        file_size: 10,
        total_records: 1,
        processed_records: 0,
        failed_records: 1,
        retry_count: 3,
        max_retries: 3,
        status: 'failed',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      })
    ).toContain('已达重试上限')

    expect(
      buildRetryHint({
        id: 'job_2',
        collection_id: 'col_1',
        file_name: 'a.json',
        file_path: '/tmp/a.json',
        file_size: 10,
        total_records: 1,
        processed_records: 0,
        failed_records: 1,
        retry_count: 1,
        max_retries: 3,
        status: 'failed',
        created_at: '',
        updated_at: '',
        created_by: 'tester'
      })
    ).toContain('可重试')
  })

  it('should group error logs by date in descending order', () => {
    const groups = groupImportJobErrorsByDate([
      {
        id: 1,
        user_id: 'tester',
        action: 'import_run_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd1',
        created_at: '2026-03-03T02:00:00Z'
      },
      {
        id: 2,
        user_id: 'tester',
        action: 'import_upsert_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd2',
        created_at: '2026-03-02T18:00:00Z'
      },
      {
        id: 3,
        user_id: 'tester',
        action: 'import_retry_exceeded',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd3',
        created_at: '2026-03-03T10:00:00Z'
      }
    ])

    expect(groups.length).toBe(2)
    expect(groups[0]).toBeDefined()
    expect(groups[1]).toBeDefined()
    expect(groups[0]!.date).toBe('2026-03-03')
    expect(groups[0]!.items.length).toBe(2)
    expect(groups[1]!.date).toBe('2026-03-02')
    expect(groups[1]!.items.length).toBe(1)
  })

  it('should merge error logs with deduplication by id', () => {
    const existing = [
      {
        id: 1,
        user_id: 'tester',
        action: 'import_run_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd1',
        created_at: '2026-03-03T02:00:00Z'
      }
    ]
    const incoming = [
      {
        id: 1,
        user_id: 'tester',
        action: 'import_run_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd1',
        created_at: '2026-03-03T02:00:00Z'
      },
      {
        id: 2,
        user_id: 'tester',
        action: 'import_upsert_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd2',
        created_at: '2026-03-03T03:00:00Z'
      }
    ]

    const merged = mergeImportJobErrorLogs(existing, incoming)
    expect(merged.length).toBe(2)
    expect(merged[0]!.id).toBe(1)
    expect(merged[1]!.id).toBe(2)
  })

  it('should build copyable error summary text', () => {
    const text = buildImportJobErrorSummaryText([
      {
        id: 1,
        user_id: 'tester',
        action: 'import_run_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'parse failed',
        created_at: '2026-03-03T02:00:00Z'
      },
      {
        id: 2,
        user_id: 'tester',
        action: 'import_upsert_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'backend timeout',
        created_at: '2026-03-03T02:10:00Z'
      }
    ])

    expect(text).toContain('[2026-03-03T02:00:00Z] import_run_failed - parse failed')
    expect(text).toContain('[2026-03-03T02:10:00Z] import_upsert_failed - backend timeout')
  })

  it('should build export filename with job id and action', () => {
    const file1 = buildImportJobErrorExportFileName('job_1')
    const file2 = buildImportJobErrorExportFileName('job_2', 'import_run_failed')
    expect(file1.startsWith('import-job-job_1-')).toBe(true)
    expect(file1.endsWith('.txt')).toBe(true)
    expect(file2.includes('import_run_failed')).toBe(true)
    expect(file2.endsWith('.txt')).toBe(true)
  })

  it('should resolve visible date range and filter logs by range', () => {
    const logs = [
      {
        id: 1,
        user_id: 'tester',
        action: 'import_run_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd1',
        created_at: '2026-03-03T02:00:00Z'
      },
      {
        id: 2,
        user_id: 'tester',
        action: 'import_upsert_failed',
        resource_type: 'import_job',
        resource_id: 'job_1',
        details: 'd2',
        created_at: '2026-03-02T18:00:00Z'
      }
    ]
    const groups = groupImportJobErrorsByDate(logs)
    const range = resolveImportJobErrorDateRange(groups)
    const filtered = filterImportJobErrorsByDateRange(logs, range)

    expect(range).toBeDefined()
    expect(range!.startDate).toBe('2026-03-02')
    expect(range!.endDate).toBe('2026-03-03')
    expect(filtered.length).toBe(2)
  })

  it('should normalize error action for export header', () => {
    expect(normalizeImportJobErrorAction('import_run_failed')).toBe('import_run_failed')
    expect(normalizeImportJobErrorAction('import_upsert_failed')).toBe('import_upsert_failed')
    expect(normalizeImportJobErrorAction('unknown')).toBe('all')
    expect(normalizeImportJobErrorAction('')).toBe('all')
  })
})
