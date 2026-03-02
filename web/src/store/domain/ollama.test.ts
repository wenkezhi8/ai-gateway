import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useOllamaStore } from './ollama'

const ollamaApiMock = vi.hoisted(() => ({
  getOllamaStatus: vi.fn(),
  installOllama: vi.fn(),
  startOllama: vi.fn(),
  stopOllama: vi.fn(),
  pullModel: vi.fn(),
  deleteModel: vi.fn()
}))

vi.mock('@/api/ollama-domain', () => ollamaApiMock)

describe('ollama store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(ollamaApiMock).forEach((fn) => (fn as any).mockReset())
  })

  it('refreshes status and derives models', async () => {
    ollamaApiMock.getOllamaStatus.mockResolvedValue({
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
    })

    const store = useOllamaStore()
    await store.refreshStatus()

    expect(store.status?.running).toBe(true)
    expect(store.models).toEqual(['qwen2.5:0.5b-instruct'])
    expect(store.runningModels).toEqual(['qwen2.5:0.5b-instruct'])
  })
})
