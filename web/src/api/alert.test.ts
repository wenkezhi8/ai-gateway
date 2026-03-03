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
})
