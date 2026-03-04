import { describe, expect, it } from 'vitest'
import type { ProviderType, PublicProviderInfo } from '@/api/provider'
import { resolveProviderDisplayMeta } from './provider-display-meta-resolver'

describe('provider-display-meta-resolver', () => {
  it('prefers providerTypes metadata when publicProviders metadata is incomplete', () => {
    const providerTypes: ProviderType[] = [
      {
        id: 'qwen',
        label: '阿里云通义千问',
        category: 'chinese',
        color: '#FF6A00',
        logo: '/logos/qwen.svg',
        default_endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
        coding_endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
        supports_coding_plan: true,
        models: []
      }
    ]

    const publicProviders: PublicProviderInfo[] = [
      {
        id: 'qwen',
        label: 'Qwen',
        color: '',
        logo: '',
        default_model: 'qwen-turbo'
      }
    ]

    const meta = resolveProviderDisplayMeta('qwen', {
      providerTypes,
      publicProviders,
      fallbackColor: '#6B7280'
    })

    expect(meta.label).toBe('阿里云通义千问')
    expect(meta.color).toBe('#FF6A00')
    expect(meta.logo).toBe('/logos/qwen.svg')
    expect(meta.defaultModel).toBe('qwen-turbo')
    expect(meta.custom).toBe(false)
  })

  it('falls back to publicProviders metadata when providerTypes metadata is unavailable', () => {
    const meta = resolveProviderDisplayMeta('moonshot', {
      providerTypes: [],
      publicProviders: [
        {
          id: 'moonshot',
          label: '月之暗面',
          color: '#1A1A1A',
          logo: '/logos/moonshot.svg',
          default_model: 'moonshot-v1-8k'
        }
      ],
      fallbackColor: '#6B7280'
    })

    expect(meta.label).toBe('月之暗面')
    expect(meta.color).toBe('#1A1A1A')
    expect(meta.logo).toBe('/logos/moonshot.svg')
    expect(meta.defaultModel).toBe('moonshot-v1-8k')
    expect(meta.custom).toBe(false)
  })

  it('uses fallback metadata when both sources miss the provider', () => {
    const meta = resolveProviderDisplayMeta('my-provider', {
      providerTypes: [],
      publicProviders: [],
      fallbackColor: '#6B7280'
    })

    expect(meta.label).toBe('my-provider')
    expect(meta.color).toBe('#6B7280')
    expect(meta.logo).toBe('')
    expect(meta.defaultModel).toBe('')
    expect(meta.custom).toBe(true)
  })
})
