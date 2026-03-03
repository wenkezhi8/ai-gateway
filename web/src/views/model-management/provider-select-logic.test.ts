import { describe, it, expect } from 'vitest'
import type { ProviderOption } from '@/views/providers-accounts/provider-options'
import {
  handleProviderLabelChange,
  type ProviderSelectState,
  createAutoFillContext
} from './provider-select-logic'

describe('provider-select-logic', () => {
  describe('should group provider options by category', () => {
    it('should separate options into four groups', () => {
      const options: ProviderOption[] = [
        { label: 'OpenAI', value: 'openai', category: 'international', color: '#10A37F', logo: '/logos/openai.svg', default_endpoint: 'https://api.openai.com/v1' },
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' },
        { label: 'Ollama', value: 'ollama', category: 'local', color: '#10B981', logo: '/logos/ollama.svg', default_endpoint: 'http://localhost:11434/v1' },
        { label: 'Custom Provider', value: 'custom-1', category: 'custom', color: '#6B7280', logo: '', default_endpoint: '' }
      ]

      const international = options.filter(p => p.category === 'international')
      const chinese = options.filter(p => p.category === 'chinese')
      const local = options.filter(p => p.category === 'local')
      const custom = options.filter(p => p.category === 'custom')

      expect(international).toHaveLength(1)
      expect(chinese).toHaveLength(1)
      expect(local).toHaveLength(1)
      expect(custom).toHaveLength(1)
    })
  })

  describe('should auto-fill provider id on first standard selection', () => {
    it('should fill id when empty', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: '',
        label: 'DeepSeek',
        color: '#10B981',
        logoFile: null,
        logoPreview: '',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      handleProviderLabelChange(state, options, 'DeepSeek', context)

      expect(state.id).toBe('deepseek')
      expect(state.color).toBe('#4D6BFE')
      expect(state.logoPreview).toBe('/logos/deepseek.svg')
      expect(context.idAutoFilled).toBe(true)
    })

    it('should not fill id when already has value', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: 'custom-id',
        label: 'DeepSeek',
        color: '#10B981',
        logoFile: null,
        logoPreview: '',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      handleProviderLabelChange(state, options, 'DeepSeek', context)

      expect(state.id).toBe('custom-id')
      expect(context.idAutoFilled).toBe(false)
    })
  })

  describe('should keep user-edited provider id when switching options', () => {
    it('should not override manually edited id', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' },
        { label: 'OpenAI', value: 'openai', category: 'international', color: '#10A37F', logo: '/logos/openai.svg', default_endpoint: 'https://api.openai.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: 'custom-id',
        label: 'DeepSeek',
        color: '#4D6BFE',
        logoFile: null,
        logoPreview: '/logos/deepseek.svg',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      // User manually edited id
      handleProviderLabelChange(state, options, 'DeepSeek', context)
      state.id = 'user-custom-id'
      context.idAutoFilled = false

      // Switch to another provider
      handleProviderLabelChange(state, options, 'OpenAI', context)

      expect(state.id).toBe('user-custom-id')
      expect(context.idAutoFilled).toBe(false)
    })

    it('should override auto-filled id when switching providers', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' },
        { label: 'OpenAI', value: 'openai', category: 'international', color: '#10A37F', logo: '/logos/openai.svg', default_endpoint: 'https://api.openai.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: '',
        label: 'DeepSeek',
        color: '#4D6BFE',
        logoFile: null,
        logoPreview: '/logos/deepseek.svg',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      // First selection (auto-fill)
      handleProviderLabelChange(state, options, 'DeepSeek', context)
      expect(state.id).toBe('deepseek')
      expect(context.idAutoFilled).toBe(true)

      // Switch to another provider (auto-fill again)
      handleProviderLabelChange(state, options, 'OpenAI', context)

      expect(state.id).toBe('openai')
      expect(context.idAutoFilled).toBe(true)
    })
  })

  describe('should preserve manual label input when no preset option matched', () => {
    it('should not update state when label not found in options', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: 'custom-id',
        label: 'My Custom Provider',
        color: '#6B7280',
        logoFile: null,
        logoPreview: '',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      handleProviderLabelChange(state, options, 'My Custom Provider', context)

      expect(state.id).toBe('custom-id')
      expect(state.color).toBe('#6B7280')
      expect(state.logoPreview).toBe('')
      expect(context.idAutoFilled).toBe(false)
    })

    it('should allow submitting custom provider with custom id', () => {
      const options: ProviderOption[] = [
        { label: 'DeepSeek', value: 'deepseek', category: 'chinese', color: '#4D6BFE', logo: '/logos/deepseek.svg', default_endpoint: 'https://api.deepseek.com/v1' }
      ]

      const state: ProviderSelectState = {
        id: 'my-custom-provider',
        label: 'My Custom Provider',
        color: '#FF5733',
        logoFile: null,
        logoPreview: '',
        idAutoFilled: false
      }

      const context = createAutoFillContext()

      handleProviderLabelChange(state, options, 'My Custom Provider', context)

      expect(state.id).toBe('my-custom-provider')
      expect(state.label).toBe('My Custom Provider')
      expect(state.color).toBe('#FF5733')
      expect(context.idAutoFilled).toBe(false)
    })
  })
})
