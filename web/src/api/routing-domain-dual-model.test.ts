import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getOllamaDualModelConfig, updateOllamaDualModelConfig } from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('routing domain dual model apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
  })

  it('should request dual-model config endpoint', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: { classifier_active_model: 'qwen2.5:0.5b-instruct' }
    })

    const data = await getOllamaDualModelConfig()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/ollama/dual-model/config')
    expect(data.classifier_active_model).toBe('qwen2.5:0.5b-instruct')
  })

  it('should update dual-model config endpoint', async () => {
    const payload = {
      classifier_active_model: 'qwen2.5:1.5b-instruct',
      vector_ollama_embedding_model: 'nomic-embed-text'
    }
    requestMock.put.mockResolvedValue({
      success: true,
      data: payload
    })

    const data = await updateOllamaDualModelConfig(payload)
    expect(requestMock.put).toHaveBeenCalledWith('/admin/router/ollama/dual-model/config', payload)
    expect(data.classifier_active_model).toBe('qwen2.5:1.5b-instruct')
  })
})
