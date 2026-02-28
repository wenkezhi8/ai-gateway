import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useOpsDomainStore } from './ops'

const opsApiMock = vi.hoisted(() => ({
  getOpsDashboard: vi.fn(),
  getOpsExportMetrics: vi.fn(),
  getOpsProviderHealth: vi.fn(),
  getOpsServices: vi.fn()
}))

vi.mock('@/api/ops-domain', () => opsApiMock)

describe('ops domain store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(opsApiMock).forEach((fn) => {
      if (typeof fn === 'function' && 'mockReset' in fn) {
        ;(fn as any).mockReset()
      }
    })
  })

  it('should load dashboard/service/provider data into success state', async () => {
    opsApiMock.getOpsDashboard.mockResolvedValue({
      system: { cpu_percent: 10 },
      realtime: { qps: 20 },
      resources: {},
      diagnosis: {}
    })
    opsApiMock.getOpsServices.mockResolvedValue([{ name: 'gateway' }])
    opsApiMock.getOpsProviderHealth.mockResolvedValue([{ name: 'openai' }])

    const store = useOpsDomainStore()
    await store.init('1h')

    expect(store.status).toBe('success')
    expect(store.system.cpu_percent).toBe(10)
    expect(store.services).toHaveLength(1)
    expect(store.providers).toHaveLength(1)
  })

  it('should enter error state on api exception', async () => {
    opsApiMock.getOpsDashboard.mockRejectedValue(new Error('dashboard failed'))

    const store = useOpsDomainStore()
    await store.init()

    expect(store.status).toBe('error')
    expect(store.error).toContain('dashboard failed')
  })

  it('should export metrics payload', async () => {
    opsApiMock.getOpsExportMetrics.mockResolvedValue({
      export_time: '2026-02-28T00:00:00Z'
    })

    const store = useOpsDomainStore()
    const payload = await store.exportMetrics()

    expect(payload.export_time).toBe('2026-02-28T00:00:00Z')
  })
})
