import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useModelLabels } from './useModelLabels'

const requestMock = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/request', () => ({
  request: requestMock
}))

describe('useModelLabels', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    const { resetLabels } = useModelLabels()
    resetLabels()
  })

  it('loads labels from model-registry endpoint', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: [
        { model: 'gpt-4o', provider: 'openai', display_name: 'GPT-4o', enabled: true },
        { model: 'qwen-max', provider: 'qwen', display_name: '通义千问 Max', enabled: true }
      ]
    })

    const { fetchModelLabels, getModelLabel } = useModelLabels()
    await fetchModelLabels()

    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/model-registry', { silent: true })
    expect(getModelLabel('openai', 'gpt-4o')).toBe('GPT-4o')
    expect(getModelLabel('qwen', 'qwen-max')).toBe('通义千问 Max')
  })

  it('keeps provider filter behavior unchanged', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: [
        { model: 'gpt-4o', provider: 'openai', display_name: 'GPT-4o', enabled: true },
        { model: 'qwen-max', provider: 'qwen', display_name: '通义千问 Max', enabled: true }
      ]
    })

    const { fetchModelLabels, getModelLabelsForProvider } = useModelLabels()
    await fetchModelLabels('openai')

    const openaiLabels = getModelLabelsForProvider('openai')
    const qwenLabels = getModelLabelsForProvider('qwen')

    expect(openaiLabels).toEqual({ 'gpt-4o': 'GPT-4o' })
    expect(qwenLabels).toEqual({})
  })
})
