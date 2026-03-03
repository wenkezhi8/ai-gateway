import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useEditionStore } from './edition'

const editionApiMock = vi.hoisted(() => ({
  checkEditionDependencies: vi.fn(),
  getEditionSetupTask: vi.fn(),
  getEditionConfig: vi.fn(),
  getEditionDefinitions: vi.fn(),
  setupEditionEnvironment: vi.fn(),
  updateEditionConfig: vi.fn()
}))

vi.mock('@/api/edition-domain', () => editionApiMock)

describe('edition domain store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(editionApiMock).forEach((fn) => {
      if (typeof fn === 'function' && 'mockReset' in fn) {
        ;(fn as any).mockReset()
      }
    })
  })

  it('should load edition config and feature getters', async () => {
    editionApiMock.getEditionConfig.mockResolvedValue({
      type: 'standard',
      features: {
        vector_cache: true,
        vector_db_management: false,
        knowledge_base: false,
        cold_hot_tiering: false
      },
      display_name: '标准版',
      description: '网关 + 语义缓存',
      dependencies: ['redis', 'ollama']
    })

    const store = useEditionStore()
    await store.fetchEditionConfig()

    expect(store.isStandard).toBe(true)
    expect(store.hasVectorCache).toBe(true)
    expect(store.hasVectorDBManagement).toBe(false)
  })

  it('should fallback to standard edition when loading fails', async () => {
    editionApiMock.getEditionConfig.mockRejectedValue(new Error('network error'))

    const store = useEditionStore()
    await store.fetchEditionConfig()

    expect(store.config?.type).toBe('standard')
    expect(store.hasVectorCache).toBe(true)
  })

  it('should update edition and return success payload', async () => {
    editionApiMock.updateEditionConfig.mockResolvedValue({
      restart_required: true,
      edition: {
        type: 'enterprise',
        features: {
          vector_cache: true,
          vector_db_management: true,
          knowledge_base: true,
          cold_hot_tiering: true
        },
        display_name: '企业版',
        description: '完整功能',
        dependencies: ['redis', 'ollama', 'qdrant']
      }
    })

    const store = useEditionStore()
    const result = await store.updateEdition('enterprise')

    expect(result.success).toBe(true)
    expect(result.restartRequired).toBe(true)
    expect(store.config?.type).toBe('enterprise')
  })

  it('should start setup task and fetch setup status', async () => {
    editionApiMock.setupEditionEnvironment.mockResolvedValue({
      task_id: 'task-1',
      accepted_at: '2026-03-04T00:00:00Z',
      message: 'accepted'
    })
    editionApiMock.getEditionSetupTask.mockResolvedValue({
      task_id: 'task-1',
      status: 'success',
      summary: 'installed redis, ollama',
      logs: 'ok',
      health: {
        redis: { name: 'Redis', address: '127.0.0.1:6379', healthy: true, message: '正常' }
      }
    })

    const store = useEditionStore()
    const created = await store.startSetup({
      edition: 'standard',
      runtime: 'docker',
      apply_config: true,
      pull_embedding_model: false
    })

    expect(created.task_id).toBe('task-1')

    const task = await store.fetchSetupTask('task-1')
    expect(task.status).toBe('success')
    expect(store.setupTask?.task_id).toBe('task-1')
  })
})
