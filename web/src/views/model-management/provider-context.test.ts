import { describe, expect, it } from 'vitest'

import { buildProvidersAccountsBackQuery, parseModelManagementContext } from './provider-context'

describe('model management provider context', () => {
  it('parses valid query fields', () => {
    const context = parseModelManagementContext({
      provider: 'deepseek',
      from: 'provider',
      focus: 'default-model',
      highlightStep: 'defaultModel'
    })

    expect(context.providerId).toBe('deepseek')
    expect(context.from).toBe('provider')
    expect(context.focus).toBe('default-model')
    expect(context.highlightStep).toBe('defaultModel')
  })

  it('uses first query value when query field is an array', () => {
    const context = parseModelManagementContext({
      provider: ['Qwen', 'deepseek'],
      from: ['providers-accounts', 'dashboard'],
      focus: ['verify-call', 'default-model'],
      highlightStep: ['verify', 'defaultModel']
    })

    expect(context.providerId).toBe('qwen')
    expect(context.from).toBe('providers-accounts')
    expect(context.focus).toBe('verify-call')
    expect(context.highlightStep).toBe('verify')
  })

  it('sanitizes invalid query fields', () => {
    const context = parseModelManagementContext({
      provider: '../bad',
      from: 'unknown',
      focus: 'other',
      highlightStep: 'other'
    })

    expect(context.providerId).toBeNull()
    expect(context.from).toBeNull()
    expect(context.focus).toBeNull()
    expect(context.highlightStep).toBeNull()
  })

  it('builds providers accounts back query', () => {
    expect(
      buildProvidersAccountsBackQuery({ providerId: 'qwen', from: null, focus: null, highlightStep: null })
    ).toEqual({ provider: 'qwen' })
    expect(
      buildProvidersAccountsBackQuery({ providerId: null, from: null, focus: null, highlightStep: null })
    ).toEqual({})
  })
})
