import { describe, expect, it } from 'vitest'

import type { ProviderType, PublicProviderInfo } from '@/api/provider'
import { buildProviderOptions } from './provider-options'

type AccountLike = {
  provider?: string
  provider_type?: string
  base_url?: string
}

function makeProviderType(overrides: Partial<ProviderType> = {}): ProviderType {
  return {
    id: 'openai',
    label: 'OpenAI',
    category: 'international',
    color: '#10A37F',
    logo: '/logos/openai.svg',
    default_endpoint: 'https://api.openai.com/v1',
    coding_endpoint: 'https://api.openai.com/v1',
    supports_coding_plan: true,
    models: ['gpt-4o'],
    ...overrides
  }
}

function makePublicProvider(overrides: Partial<PublicProviderInfo> = {}): PublicProviderInfo {
  return {
    id: 'openai',
    label: 'OpenAI',
    color: '#10A37F',
    logo: '/logos/openai.svg',
    ...overrides
  }
}

describe('provider options builder', () => {
  it('should keep provider options when /admin/providers/types fails', () => {
    const options = buildProviderOptions({
      types: [],
      publicProviders: [makePublicProvider({ id: 'deepseek', label: 'DeepSeek' })],
      accounts: [{ provider: 'my-custom', base_url: 'https://api.my-custom.ai/v1' }]
    })

    expect(options.some(option => option.value === 'deepseek')).toBe(true)
    expect(options.some(option => option.value === 'my-custom')).toBe(true)
  })

  it('should merge provider ids from accounts and public providers', () => {
    const options = buildProviderOptions({
      types: [makeProviderType({ id: 'openai' })],
      publicProviders: [makePublicProvider({ id: 'qwen', label: '通义千问' })],
      accounts: [{ provider: 'zhipu', base_url: 'https://open.bigmodel.cn/api/paas/v4' }]
    })

    const ids = options.map(option => option.value)
    expect(ids).toContain('openai')
    expect(ids).toContain('qwen')
    expect(ids).toContain('zhipu')
  })

  it('should classify unknown provider as custom', () => {
    const options = buildProviderOptions({
      types: [],
      publicProviders: [],
      accounts: [{ provider: 'acme-ai', base_url: 'https://api.acme-ai.com/v1' }]
    })

    const custom = options.find(option => option.value === 'acme-ai')
    expect(custom?.category).toBe('custom')
  })

  it('should keep deterministic sort and dedupe', () => {
    const types: ProviderType[] = [
      makeProviderType({ id: 'qwen', label: '通义千问', category: 'chinese' }),
      makeProviderType({ id: 'openai', label: 'OpenAI', category: 'international' })
    ]
    const publicProviders: PublicProviderInfo[] = [
      makePublicProvider({ id: 'openai', label: 'OpenAI' }),
      makePublicProvider({ id: 'acme-ai', label: 'Acme AI' })
    ]
    const accounts: AccountLike[] = [
      { provider: 'qwen', base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1' },
      { provider: 'ollama', base_url: 'http://localhost:11434/v1' },
      { provider: 'acme-ai', base_url: 'https://api.acme-ai.com/v1' }
    ]

    const options = buildProviderOptions({
      types,
      publicProviders,
      accounts
    })

    const ids = options.map(option => option.value)
    expect(ids).toEqual(['openai', 'qwen', 'ollama', 'acme-ai'])
  })
})
