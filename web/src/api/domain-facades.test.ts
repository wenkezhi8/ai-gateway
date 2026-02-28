import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getCacheStats } from './cache-domain'
import { getOpsDashboard, getOpsExportMetrics } from './ops-domain'
import { getRouterConfig } from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('domain api facades', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
    requestMock.delete.mockReset()
  })

  it('routing facade unwraps /admin/router/config', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { default_strategy: 'auto' }
    })

    const data = await getRouterConfig()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/config')
    expect(data.default_strategy).toBe('auto')
  })

  it('cache facade unwraps /admin/cache/stats', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { request_cache: { hit_rate: 0.5 } }
    })

    const data = await getCacheStats()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/cache/stats')
    expect(data.request_cache.hit_rate).toBe(0.5)
  })

  it('ops facade unwraps envelope responses and accepts plain export payload', async () => {
    requestMock.get
      .mockResolvedValueOnce({
        success: true,
        data: { system: { cpu_percent: 30 } }
      })
      .mockResolvedValueOnce({
        export_time: '2026-02-28T00:00:00Z',
        resources: {}
      })

    const dashboard = await getOpsDashboard('1h')
    const exportData = await getOpsExportMetrics()

    expect(requestMock.get).toHaveBeenNthCalledWith(1, '/admin/ops/dashboard?range=1h')
    expect(requestMock.get).toHaveBeenNthCalledWith(2, '/admin/ops/export')
    expect(dashboard.system.cpu_percent).toBe(30)
    expect(exportData.export_time).toBe('2026-02-28T00:00:00Z')
  })
})
