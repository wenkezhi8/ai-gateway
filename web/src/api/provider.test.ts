import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getProviderTypes, getPublicProviders } from './provider'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('provider api', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
    requestMock.delete.mockReset()
  })

  it('should fetch provider types from /admin/providers/types', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'openai',
          label: 'OpenAI',
          category: 'international',
          color: '#10A37F',
          logo: '/logos/openai.svg',
          icon: 'ChatDotRound',
          default_endpoint: 'https://api.openai.com/v1',
          coding_endpoint: 'https://api.openai.com/v1',
          supports_coding_plan: true,
          models: ['gpt-4o']
        }
      ]
    })

    const data = await getProviderTypes()

    expect(requestMock.get).toHaveBeenCalledWith('/admin/providers/types')
    expect(data[0]!.id).toBe('openai')
    expect(data[0]!.models).toEqual(['gpt-4o'])
  })

  it('should throw when provider types response is not successful', async () => {
    requestMock.get.mockResolvedValue({
      success: false,
      error: 'PROVIDER_TYPES_LOAD_FAILED'
    })

    await expect(getProviderTypes()).rejects.toThrow('PROVIDER_TYPES_LOAD_FAILED')
  })

  it('should fetch public providers from /v1/config/providers', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: [
        { id: 'openai', label: 'OpenAI', color: '#10A37F', logo: '/logos/openai.svg', default_model: 'gpt-4o' }
      ]
    })

    const data = await getPublicProviders()

    expect(requestMock.get).toHaveBeenCalledWith('/v1/config/providers')
    expect(data[0]!.id).toBe('openai')
  })

  it('should normalize nested public providers payload from gateway', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: {
        providers: [
          {
            name: 'openai',
            models: ['gpt-4o'],
            enabled: true
          }
        ]
      }
    })

    const data = await getPublicProviders()

    expect(data).toEqual([
      {
        id: 'openai',
        label: 'openai',
        color: '',
        logo: '',
        default_model: 'gpt-4o'
      }
    ])
  })

  it('should throw when public providers response is not successful', async () => {
    requestMock.get.mockResolvedValue({
      success: false,
      error: 'PUBLIC_PROVIDERS_LOAD_FAILED'
    })

    await expect(getPublicProviders()).rejects.toThrow('PUBLIC_PROVIDERS_LOAD_FAILED')
  })
})
