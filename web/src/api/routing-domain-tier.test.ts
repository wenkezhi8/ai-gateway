import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  getVectorTierConfig,
  getVectorTierStats,
  promoteVectorTierEntry,
  triggerVectorTierMigrate,
  updateVectorTierConfig
} from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('routing domain tier apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
  })

  it('should call tier config endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: { cold_vector_enabled: true } })
    const data = await getVectorTierConfig()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/vector/tier/config')
    expect(data.cold_vector_enabled).toBe(true)
  })

  it('should update tier config endpoint', async () => {
    requestMock.put.mockResolvedValue({ success: true, data: { cold_vector_backend: 'sqlite' } })
    const payload = { cold_vector_backend: 'sqlite' }
    const data = await updateVectorTierConfig(payload)
    expect(requestMock.put).toHaveBeenCalledWith('/admin/router/vector/tier/config', payload)
    expect(data.cold_vector_backend).toBe('sqlite')
  })

  it('should call tier stats endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: { enabled: true } })
    const data = await getVectorTierStats()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/vector/tier/stats')
    expect(data.enabled).toBe(true)
  })

  it('should call manual migrate endpoint', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { triggered: true } })
    const data = await triggerVectorTierMigrate()
    expect(requestMock.post).toHaveBeenCalledWith('/admin/router/vector/tier/migrate')
    expect(data.triggered).toBe(true)
  })

  it('should call manual promote endpoint with cache_key', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: { cache_key: 'k1' } })
    const data = await promoteVectorTierEntry('k1')
    expect(requestMock.post).toHaveBeenCalledWith('/admin/router/vector/tier/promote', { cache_key: 'k1' })
    expect(data.cache_key).toBe('k1')
  })
})
