import { beforeEach, describe, expect, it, vi } from 'vitest'

import { clearTraces, getTraces, getTraceDetail } from './trace-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('trace-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.delete.mockReset()
  })

  it('should unwrap list and total from traces response', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: [{ request_id: 'req-1', answer_source: 'cache_v2', task_type: 'analysis' }],
      total: 101
    })

    const result = await getTraces({ limit: 20, offset: 0 })
    expect(requestMock.get).toHaveBeenCalledWith('/admin/traces', { params: { limit: 20, offset: 0 } })
    expect(result.total).toBe(101)
    expect(result.data).toHaveLength(1)
    expect(result.data[0]?.answer_source).toBe('cache_v2')
    expect(result.data[0]?.task_type).toBe('analysis')
  })

  it('should keep detail endpoint unchanged', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: [{ request_id: 'req-2', operation: 'http.entry' }] })

    const result = await getTraceDetail('req-2')
    expect(requestMock.get).toHaveBeenCalledWith('/admin/traces/req-2')
    expect(result).toHaveLength(1)
    expect(result[0]?.request_id).toBe('req-2')
  })

  it('should call clear traces endpoint and return deleted count', async () => {
    requestMock.delete.mockResolvedValue({ success: true, data: { deleted: 42 } })

    const result = await clearTraces()
    expect(requestMock.delete).toHaveBeenCalledWith('/admin/traces')
    expect(result.deleted).toBe(42)
  })
})
