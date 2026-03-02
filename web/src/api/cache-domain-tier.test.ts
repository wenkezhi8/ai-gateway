import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getCacheTaskTTLConfig, getVectorStats } from './cache-domain'

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
    expect(data.task_types[0].key).toBe('fact')
    expect(data.model_options[0].models).toEqual(['gpt-4o'])
  })

  it('should throw when cache task ttl config response fails', async () => {
    requestMock.get.mockResolvedValue({
      success: false,
      error: 'CACHE_TTL_CONFIG_LOAD_FAILED'
    })

    await expect(getCacheTaskTTLConfig()).rejects.toThrow('CACHE_TTL_CONFIG_LOAD_FAILED')
  })
})
