import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useCacheDomainStore } from './cache'

const cacheApiMock = vi.hoisted(() => ({
  cleanupInvalidEntries: vi.fn(),
  clearCacheByType: vi.fn(),
  createCacheRule: vi.fn(),
  deleteCacheEntry: vi.fn(),
  deleteCacheEntryGroup: vi.fn(),
  deleteCacheRule: vi.fn(),
  getCacheConfig: vi.fn(),
  getCacheEntries: vi.fn(),
  getCacheEntryDetail: vi.fn(),
  getCacheHealth: vi.fn(),
  getCacheRules: vi.fn(),
  getCacheStats: vi.fn(),
  getSemanticSignatures: vi.fn(),
  getTtlConfig: vi.fn(),
  updateCacheConfig: vi.fn(),
  updateCacheRule: vi.fn(),
  updateTtlConfig: vi.fn()
}))

vi.mock('@/api/cache-domain', () => cacheApiMock)

describe('cache domain store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(cacheApiMock).forEach((fn) => {
      if (typeof fn === 'function' && 'mockReset' in fn) {
        ;(fn as any).mockReset()
      }
    })
  })

  it('should load data and enter success state', async () => {
    cacheApiMock.getCacheStats.mockResolvedValue({ request_cache: {} })
    cacheApiMock.getCacheConfig.mockResolvedValue({})
    cacheApiMock.getCacheHealth.mockResolvedValue({ status: 'healthy' })
    cacheApiMock.getCacheRules.mockResolvedValue([{ id: 1 }])
    cacheApiMock.getSemanticSignatures.mockResolvedValue([])
    cacheApiMock.getTtlConfig.mockResolvedValue({})

    const store = useCacheDomainStore()
    await store.init()

    expect(store.status).toBe('success')
    expect(store.rules).toHaveLength(1)
  })

  it('should enter error state when api fails', async () => {
    cacheApiMock.getCacheStats.mockRejectedValue(new Error('stats failed'))

    const store = useCacheDomainStore()
    await store.init()

    expect(store.status).toBe('error')
    expect(store.error).toContain('stats failed')
  })

  it('should refresh list after deleting entry', async () => {
    cacheApiMock.getCacheStats.mockResolvedValue({})
    cacheApiMock.getCacheConfig.mockResolvedValue({})
    cacheApiMock.getCacheHealth.mockResolvedValue({})
    cacheApiMock.getCacheRules.mockResolvedValue([])
    cacheApiMock.getSemanticSignatures.mockResolvedValue([])
    cacheApiMock.getTtlConfig.mockResolvedValue({})
    cacheApiMock.deleteCacheEntry.mockResolvedValue({})
    cacheApiMock.getCacheEntries.mockResolvedValue({
      entries: [{ key: 'k1' }],
      total: 1
    })

    const store = useCacheDomainStore()
    await store.removeEntry('k1', 'page=1')

    expect(cacheApiMock.deleteCacheEntry).toHaveBeenCalledWith('k1')
    expect(cacheApiMock.getCacheEntries).toHaveBeenCalledWith('page=1')
    expect(store.entries).toHaveLength(1)
  })
})
