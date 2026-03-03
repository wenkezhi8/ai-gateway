import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  deleteModelRegistry,
  getModelRegistry,
  upsertModelRegistry
} from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  post: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('routing domain model-registry apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
    requestMock.delete.mockReset()
  })

  it('calls model-registry list endpoint', async () => {
    requestMock.get.mockResolvedValue({ success: true, data: [] })

    await getModelRegistry()

    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/model-registry')
  })

  it('calls model-registry upsert endpoint', async () => {
    requestMock.put.mockResolvedValue({ success: true })

    await upsertModelRegistry('gpt-4o', { provider: 'openai', enabled: true })

    expect(requestMock.put).toHaveBeenCalledWith('/admin/router/model-registry/gpt-4o', {
      provider: 'openai',
      enabled: true
    })
  })

  it('calls model-registry delete endpoint', async () => {
    requestMock.delete.mockResolvedValue({ success: true })

    await deleteModelRegistry('gpt-4o')

    expect(requestMock.delete).toHaveBeenCalledWith('/admin/router/model-registry/gpt-4o')
  })

  it('does not expose deprecated model score api exports', async () => {
    const mod = await import('./routing-domain')

    expect(mod).not.toHaveProperty('getRouterModels')
    expect(mod).not.toHaveProperty('updateModelScore')
  })
})
