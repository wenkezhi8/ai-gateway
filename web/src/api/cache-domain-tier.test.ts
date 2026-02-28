import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getVectorTierStats, promoteVectorTierEntry, triggerVectorTierMigrate } from './cache-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('cache domain tier apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
  })

  it('should call tier stats endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: { enabled: true } })
    const data = await getVectorTierStats()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/vector/tier/stats')
    expect(data.enabled).toBe(true)
  })

  it('should call manual migrate endpoint', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { triggered: true } })
    const data = await triggerVectorTierMigrate()
    expect(requestMock.post).toHaveBeenCalledWith('/admin/cache/vector/tier/migrate')
    expect(data.triggered).toBe(true)
  })

  it('should call manual promote endpoint with cache_key', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { cache_key: 'k1' } })
    const data = await promoteVectorTierEntry('k1')
    expect(requestMock.post).toHaveBeenCalledWith('/admin/cache/vector/tier/promote', { cache_key: 'k1' })
    expect(data.cache_key).toBe('k1')
  })
})
