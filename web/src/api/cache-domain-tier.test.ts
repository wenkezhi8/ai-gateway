import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  getCacheRequestHits,
  getCacheRequestStats,
  getCacheTaskTTLConfig,
  getVectorStats,
  isCacheTaskTTLNotFoundError
} from './cache-domain'

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

  it('should call cache task ttl config endpoint', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: {
        task_types: [{ key: 'fact', label: '事实查询', description: 'desc', default_ttl: 24, ttl_unit: 'hours' }],
        model_options: [{ provider_id: 'openai', provider_label: 'OpenAI', models: ['gpt-4o'] }]
      }
    })

    const data = await getCacheTaskTTLConfig()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/task-ttl')
    expect(data.task_types[0]!.key).toBe('fact')
    expect(data.model_options[0]!.models).toEqual(['gpt-4o'])
  })

  it('should throw when cache task ttl config response fails', async () => {
    requestMock.get.mockResolvedValue({
      success: false,
      error: 'CACHE_TTL_CONFIG_LOAD_FAILED'
    })

    await expect(getCacheTaskTTLConfig()).rejects.toThrow('CACHE_TTL_CONFIG_LOAD_FAILED')
  })

  it('should identify 404 error from cache task ttl request', () => {
    expect(isCacheTaskTTLNotFoundError({ response: { status: 404 } })).toBe(true)
    expect(isCacheTaskTTLNotFoundError({ response: { status: 500 } })).toBe(false)
    expect(isCacheTaskTTLNotFoundError(null)).toBe(false)
  })

  it('should request cache request stats with default 24h window', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { total_requests: 12, hit_rate: 0.5 }
    })

    await getCacheRequestStats()

    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/request-stats?window=24h&source=all')
  })

  it('should request cache request stats without default window when start/end provided', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { total_requests: 8, hit_rate: 0.25 }
    })

    await getCacheRequestStats({
      start: 's',
      end: 'e',
      source: 'exact_prompt'
    })

    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/request-stats?start_time=s&end_time=e&source=exact_prompt')
  })

  it('should request cache request hits with source and paging', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { hits: [], total: 0 }
    })

    await getCacheRequestHits({
      source: 'v2',
      page: 2,
      page_size: 50
    })

    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/request-hits?window=24h&source=v2&page=2&page_size=50')
  })
})
