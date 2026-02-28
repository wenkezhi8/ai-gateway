import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useRoutingDomainStore } from './routing'

const routingApiMock = vi.hoisted(() => ({
  getRouterConfig: vi.fn(),
  getRouterModels: vi.fn(),
  getAvailableModels: vi.fn(),
  getCascadeRules: vi.fn(),
  getFeedbackStats: vi.fn(),
  getClassifierHealth: vi.fn(),
  getClassifierStats: vi.fn(),
  getClassifierModels: vi.fn(),
  getTaskModelMapping: vi.fn(),
  getTaskTypeDistribution: vi.fn(),
  getOllamaStatus: vi.fn(),
  getClassifierSwitchTask: vi.fn(),
  putTaskModelMapping: vi.fn(),
  switchClassifierModelAsync: vi.fn(),
  triggerFeedbackOptimization: vi.fn(),
  updateModelScore: vi.fn(),
  updateRouterConfig: vi.fn()
}))

vi.mock('@/api/routing-domain', () => routingApiMock)

describe('routing domain store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(routingApiMock).forEach((fn) => {
      if (typeof fn === 'function' && 'mockReset' in fn) {
        ;(fn as any).mockReset()
      }
    })
  })

  it('should load data and enter success state', async () => {
    routingApiMock.getRouterConfig.mockResolvedValue({ default_strategy: 'auto' })
    routingApiMock.getRouterModels.mockResolvedValue([{ model: 'gpt-4o' }])
    routingApiMock.getAvailableModels.mockResolvedValue([{ id: 'gpt-4o' }])
    routingApiMock.getCascadeRules.mockResolvedValue([])
    routingApiMock.getFeedbackStats.mockResolvedValue({})
    routingApiMock.getClassifierHealth.mockResolvedValue({ healthy: true })
    routingApiMock.getClassifierStats.mockResolvedValue({})
    routingApiMock.getClassifierModels.mockResolvedValue({ models: ['qwen3:4b'] })
    routingApiMock.getTaskModelMapping.mockResolvedValue({})
    routingApiMock.getTaskTypeDistribution.mockResolvedValue({ distribution: [] })

    const store = useRoutingDomainStore()
    await store.init()

    expect(store.status).toBe('success')
    expect(store.modelScores).toHaveLength(1)
    expect(store.availableModels).toHaveLength(1)
  })

  it('should enter empty state when no core data is returned', async () => {
    routingApiMock.getRouterConfig.mockResolvedValue({})
    routingApiMock.getRouterModels.mockResolvedValue([])
    routingApiMock.getAvailableModels.mockResolvedValue([])
    routingApiMock.getCascadeRules.mockResolvedValue([])
    routingApiMock.getFeedbackStats.mockResolvedValue({})
    routingApiMock.getClassifierHealth.mockResolvedValue({})
    routingApiMock.getClassifierStats.mockResolvedValue({})
    routingApiMock.getClassifierModels.mockResolvedValue({})
    routingApiMock.getTaskModelMapping.mockResolvedValue({})
    routingApiMock.getTaskTypeDistribution.mockResolvedValue({})

    const store = useRoutingDomainStore()
    await store.init()

    expect(store.status).toBe('empty')
  })

  it('should enter error state when api fails', async () => {
    routingApiMock.getRouterConfig.mockRejectedValue(new Error('boom'))

    const store = useRoutingDomainStore()
    await store.init()

    expect(store.status).toBe('error')
    expect(store.error).toContain('boom')
  })
})
