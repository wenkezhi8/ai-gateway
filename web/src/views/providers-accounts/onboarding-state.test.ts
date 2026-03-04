import { describe, expect, it } from 'vitest'

import {
  buildOnboardingState,
  resolveContinueAction,
  type ProviderCompletionRow
} from './onboarding-state'

function row(overrides: Partial<ProviderCompletionRow>): ProviderCompletionRow {
  return {
    provider: 'deepseek',
    label: 'DeepSeek',
    hasAccount: false,
    hasLimit: false,
    hasDefaultModel: false,
    hasVerify: false,
    ...overrides
  }
}

describe('providers onboarding state', () => {
  it('builds core global steps with expected completion rules', () => {
    const state = buildOnboardingState([
      row({ provider: 'deepseek', hasAccount: true, hasLimit: true, hasDefaultModel: false }),
      row({ provider: 'qwen', hasAccount: false, hasLimit: false, hasDefaultModel: false })
    ])

    expect(state.totalCoreSteps).toBe(3)
    expect(state.completedCoreSteps).toBe(2)
    expect(state.progressPercent).toBe(67)
    expect(state.steps.map(step => step.key)).toEqual(['account', 'limit', 'defaultModel', 'verify'])
    expect(state.steps.find(step => step.key === 'account')?.completed).toBe(true)
    expect(state.steps.find(step => step.key === 'limit')?.completed).toBe(true)
    expect(state.steps.find(step => step.key === 'defaultModel')?.completed).toBe(false)
  })

  it('resolves continue action in provider-first deterministic order', () => {
    const rows: ProviderCompletionRow[] = [
      row({ provider: 'qwen', label: '通义千问', hasAccount: true, hasLimit: false, hasDefaultModel: true }),
      row({ provider: 'deepseek', hasAccount: false, hasLimit: false, hasDefaultModel: false })
    ]

    expect(resolveContinueAction(rows)).toEqual({ key: 'account', provider: 'deepseek', label: 'DeepSeek' })
  })

  it('falls through to verify action after core steps completed', () => {
    const rows: ProviderCompletionRow[] = [
      row({ provider: 'deepseek', hasAccount: true, hasLimit: true, hasDefaultModel: true, hasVerify: false })
    ]

    expect(resolveContinueAction(rows)).toEqual({ key: 'verify', provider: 'deepseek', label: 'DeepSeek' })
  })
})
