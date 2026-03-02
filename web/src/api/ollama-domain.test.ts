import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getOllamaStatus, pullModel, deleteModel } from './ollama-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('ollama-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
    requestMock.delete.mockReset()
  })

  it('gets ollama status', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: {
        installed: true,
        running: true,
        model: 'qwen2.5:0.5b-instruct',
        model_installed: true,
        models: ['qwen2.5:0.5b-instruct'],
        running_models: ['qwen2.5:0.5b-instruct'],
        running_model_details: [],
        running_vram_bytes_total: 0,
        running_model: 'qwen2.5:0.5b-instruct',
        keep_alive_disabled: true,
        message: 'ok',
        os: 'darwin'
      }
    })

    const data = await getOllamaStatus('qwen2.5:0.5b-instruct')

    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/ollama/status?model=qwen2.5%3A0.5b-instruct')
    expect(data.installed).toBe(true)
    expect(data.running).toBe(true)
  })

  it('pulls and deletes model', async () => {
    requestMock.post.mockResolvedValue({ success: true, data: {} })

    await pullModel('qwen2.5:0.5b-instruct')
    await deleteModel('qwen2.5:0.5b-instruct')

    expect(requestMock.post).toHaveBeenNthCalledWith(1, '/admin/router/ollama/pull', {
      model: 'qwen2.5:0.5b-instruct'
    })
    expect(requestMock.post).toHaveBeenNthCalledWith(2, '/admin/router/ollama/delete', {
      model: 'qwen2.5:0.5b-instruct'
    })
  })
})
