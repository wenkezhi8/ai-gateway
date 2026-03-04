import type { Account, LimitConfig } from '@/api/account'

import { inferProviderFromAccountBaseURL, normalizeProviderID } from './provider-options'

export type CompletionState = 'done' | 'todo' | 'na'

export interface ProviderCompletionInputAccount
  extends Pick<Account, 'id' | 'name' | 'provider' | 'provider_type' | 'base_url' | 'enabled' | 'limits' | 'usage'> {}

export interface ProviderCompletionInputOption {
  value: string
  label?: string
}

export interface ProviderCompletionInput {
  providerOptions: ProviderCompletionInputOption[]
  accounts: ProviderCompletionInputAccount[]
  providerDefaults: Record<string, string>
}

export interface ProviderCompletionRow {
  provider: string
  label: string
  hasAccount: boolean
  hasLimit: boolean
  hasDefaultModel: boolean
  hasVerify: boolean
}

function asFiniteNumber(value: unknown): number {
  if (typeof value !== 'number' || Number.isNaN(value) || !Number.isFinite(value)) return 0
  return value
}

function normalizeProvider(provider: string): string {
  return normalizeProviderID(provider) || String(provider || '').trim().toLowerCase()
}

function resolveAccountProvider(account: ProviderCompletionInputAccount): string {
  const inferred = inferProviderFromAccountBaseURL(account.base_url)
  if (inferred) return inferred
  return normalizeProvider(account.provider || account.provider_type || '')
}

function collectLimits(limits?: Record<string, LimitConfig>): number[] {
  if (!limits) return []
  return Object.values(limits).map(limit => asFiniteNumber(limit?.limit))
}

function collectUsageLimits(account: ProviderCompletionInputAccount): number[] {
  const usage = account.usage
  if (!usage) return []
  return Object.values(usage).map(item => asFiniteNumber(item?.limit))
}

function hasUsageSignal(account: ProviderCompletionInputAccount): boolean {
  const usage = account.usage
  if (!usage) return false
  return Object.values(usage).some(item => asFiniteNumber(item?.used) > 0)
}

export function getAccountEffectiveLimit(account: ProviderCompletionInputAccount): number {
  const candidates = [...collectLimits(account.limits), ...collectUsageLimits(account)]
  if (!candidates.length) return 0
  return Math.max(...candidates, 0)
}

function normalizeProviderDefaults(providerDefaults: Record<string, string>): Record<string, string> {
  return Object.entries(providerDefaults || {}).reduce<Record<string, string>>((acc, [provider, model]) => {
    const providerId = normalizeProvider(provider)
    if (!providerId) return acc
    acc[providerId] = String(model || '').trim()
    return acc
  }, {})
}

export function buildProviderCompletionRows(input: ProviderCompletionInput): ProviderCompletionRow[] {
  const providerDefaults = normalizeProviderDefaults(input.providerDefaults)
  const labelMap = new Map<string, string>()
  const providerOrder: string[] = []

  const pushProvider = (provider: string, label?: string) => {
    const providerId = normalizeProvider(provider)
    if (!providerId) return
    if (!providerOrder.includes(providerId)) providerOrder.push(providerId)
    if (label && !labelMap.has(providerId)) labelMap.set(providerId, label)
  }

  for (const option of input.providerOptions || []) {
    pushProvider(option.value, option.label)
  }

  for (const account of input.accounts || []) {
    pushProvider(resolveAccountProvider(account))
  }

  for (const provider of Object.keys(providerDefaults)) {
    pushProvider(provider)
  }

  const accountsByProvider = new Map<string, ProviderCompletionInputAccount[]>()
  for (const account of input.accounts || []) {
    const providerId = resolveAccountProvider(account)
    if (!providerId) continue
    const list = accountsByProvider.get(providerId) || []
    list.push(account)
    accountsByProvider.set(providerId, list)
  }

  return providerOrder.map((provider) => {
    const scoped = accountsByProvider.get(provider) || []
    const hasAccount = scoped.some(account => account.enabled)
    const hasLimit = scoped.some(account => getAccountEffectiveLimit(account) > 0)
    const hasDefaultModel = Boolean(providerDefaults[provider])
    const hasVerify = scoped.some(account => account.enabled && hasUsageSignal(account))

    return {
      provider,
      label: labelMap.get(provider) || provider,
      hasAccount,
      hasLimit,
      hasDefaultModel,
      hasVerify
    }
  })
}
