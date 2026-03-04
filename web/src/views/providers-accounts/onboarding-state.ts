import type { ProviderCompletionInputAccount, ProviderCompletionRow as CompletionRow } from './provider-completion'
import { getAccountEffectiveLimit } from './provider-completion'

export type ProviderCompletionRow = CompletionRow
export type NextStep = 'account' | 'limit' | 'defaultModel' | 'done'
export type ContinueActionKey = 'account' | 'limit' | 'defaultModel' | 'verify' | 'done'

export interface OnboardingStepsState {
  hasAccount: boolean
  hasLimit: boolean
  hasDefaultModel: boolean
  nextStep: NextStep
}

export interface OnboardingStepItem {
  key: ContinueActionKey extends infer T ? Exclude<T, 'done'> : never
  title: string
  completed: boolean
}

export interface OnboardingState {
  steps: OnboardingStepItem[]
  totalCoreSteps: number
  completedCoreSteps: number
  progressPercent: number
}

export interface ContinueAction {
  key: ContinueActionKey
  provider?: string
  label?: string
}

function hasAnyDefaultModel(providerDefaults: Record<string, string>): boolean {
  return Object.values(providerDefaults).some(model => Boolean(String(model || '').trim()))
}

export function computeOnboardingStepsState(
  accounts: ProviderCompletionInputAccount[],
  providerDefaults: Record<string, string>
): OnboardingStepsState {
  const hasAccount = accounts.some(account => account.enabled)
  const hasLimit = accounts.some(account => getAccountEffectiveLimit(account) > 0)
  const hasDefaultModel = hasAnyDefaultModel(providerDefaults)

  let nextStep: NextStep = 'done'
  if (!hasAccount) nextStep = 'account'
  else if (!hasLimit) nextStep = 'limit'
  else if (!hasDefaultModel) nextStep = 'defaultModel'

  return { hasAccount, hasLimit, hasDefaultModel, nextStep }
}

export function buildOnboardingState(rows: ProviderCompletionRow[]): OnboardingState {
  const hasAccount = rows.some(row => row.hasAccount)
  const hasLimit = rows.some(row => row.hasLimit)
  const hasDefaultModel = rows.some(row => row.hasDefaultModel)
  const hasVerify = rows.some(row => row.hasVerify)

  const steps: OnboardingStepItem[] = [
    { key: 'account', title: '步骤1：添加账号', completed: hasAccount },
    { key: 'limit', title: '步骤2：设置限额', completed: hasLimit },
    { key: 'defaultModel', title: '步骤3：设置默认模型', completed: hasDefaultModel },
    { key: 'verify', title: '步骤4：可调用验证', completed: hasVerify }
  ]

  const totalCoreSteps = 3
  const completedCoreSteps = [hasAccount, hasLimit, hasDefaultModel].filter(Boolean).length
  const progressPercent = Math.round((completedCoreSteps / totalCoreSteps) * 100)

  return {
    steps,
    totalCoreSteps,
    completedCoreSteps,
    progressPercent
  }
}

export function resolveContinueAction(rows: ProviderCompletionRow[]): ContinueAction {
  const firstMissingAccount = rows.find(row => !row.hasAccount)
  if (firstMissingAccount) {
    return { key: 'account', provider: firstMissingAccount.provider, label: firstMissingAccount.label }
  }

  const firstMissingLimit = rows.find(row => !row.hasLimit)
  if (firstMissingLimit) {
    return { key: 'limit', provider: firstMissingLimit.provider, label: firstMissingLimit.label }
  }

  const firstMissingDefaultModel = rows.find(row => !row.hasDefaultModel)
  if (firstMissingDefaultModel) {
    return { key: 'defaultModel', provider: firstMissingDefaultModel.provider, label: firstMissingDefaultModel.label }
  }

  const firstMissingVerify = rows.find(row => !row.hasVerify)
  if (firstMissingVerify) {
    return { key: 'verify', provider: firstMissingVerify.provider, label: firstMissingVerify.label }
  }

  return { key: 'done' }
}
