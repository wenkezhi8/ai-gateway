import { beforeEach, describe, expect, it, vi } from 'vitest'

import { alertApi } from './alert'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('alert api', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
    requestMock.delete.mockReset()
  })

  it('posts resolve-similar request to batch endpoint', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { affected: 3 } })

    await alertApi.resolveSimilar({
      level: 'warning',
      source: 'system',
      message: 'cpu high'
    })

    expect(requestMock.post).toHaveBeenCalledWith('/admin/alerts/resolve-similar', {
      level: 'warning',
      source: 'system',
      message: 'cpu high'
    })
  })

  it('posts resolve-similar request with dedup_key when provided', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { affected: 5 } })

    await alertApi.resolveSimilar({
      level: 'warning',
      source: 'system',
      message: 'ignored message',
      dedup_key: 'memory_warning'
    })

    expect(requestMock.post).toHaveBeenCalledWith('/admin/alerts/resolve-similar', {
      level: 'warning',
      source: 'system',
      message: 'ignored message',
      dedup_key: 'memory_warning'
    })
  })

  it('gets alert history from unified alert facade endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: { list: [], total: 0 } })

    await alertApi.getHistory({ level: 'warning' })

    expect(requestMock.get).toHaveBeenCalledWith('/admin/alerts/history', {
      params: { level: 'warning' }
    })
  })

  it('deletes alert history from admin endpoint', async () => {
    requestMock.delete.mockResolvedValue({ success: true, data: { affected: 2 } })

    await alertApi.clearHistory()

    expect(requestMock.delete).toHaveBeenCalledWith('/admin/alerts/history')
  })
})
