import { describe, expect, it } from 'vitest'
import { buildProviderIdsForSettings } from './provider-settings-sources'

describe('provider-settings-sources', () => {
  it('includes providers from public providers and model registry', () => {
    const ids = buildProviderIdsForSettings({
      publicProviders: [{ id: 'openai', label: 'OpenAI', color: '', logo: '' }],
      modelsByProvider: {
        zhipu: ['glm-4-plus']
      },
      providerDefaults: {}
    })

    expect(new Set(ids)).toEqual(new Set(['openai', 'zhipu']))
  })

  it('ignores orphan provider-default keys that are not in public providers or model registry', () => {
    const ids = buildProviderIdsForSettings({
      publicProviders: [{ id: 'openai', label: 'OpenAI', color: '', logo: '' }],
      modelsByProvider: {},
      providerDefaults: {
        deepseek: 'deepseek-chat',
        openai: 'gpt-4o'
      }
    })

    expect(ids).toContain('openai')
    expect(ids).not.toContain('deepseek')
  })

  it('keeps provider-default keys when provider exists in model registry', () => {
    const ids = buildProviderIdsForSettings({
      publicProviders: [],
      modelsByProvider: {
        volcengine: ['doubao-pro-32k']
      },
      providerDefaults: {
        volcengine: 'doubao-pro-32k'
      }
    })

    expect(ids).toEqual(['volcengine'])
  })
})
