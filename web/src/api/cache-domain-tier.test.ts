import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getVectorStats } from './cache-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('cache domain vector apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
  })

  it('should call vector stats endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: { enabled: true } })
    const data = await getVectorStats()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/vector/stats')
    expect(data.enabled).toBe(true)
  })
})
