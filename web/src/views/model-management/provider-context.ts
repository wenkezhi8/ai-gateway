import type { LocationQuery } from 'vue-router'

export type ModelFocusTarget = 'default-model' | 'verify-call' | null
export type ModelHighlightStep = 'account' | 'limit' | 'defaultModel' | 'verify' | null
export type ModelManagementFrom = 'provider' | 'providers-accounts' | 'dashboard' | null

export interface ModelManagementContext {
  providerId: string | null
  from: ModelManagementFrom
  focus: ModelFocusTarget
  highlightStep: ModelHighlightStep
}

const SAFE_QUERY = /^[a-z0-9-]+$/i

export function toSingleQueryValue(value: LocationQuery[string] | undefined): string {
  if (Array.isArray(value)) return String(value[0] ?? '')
  return String(value ?? '')
}

export function sanitizeProviderId(value: string): string | null {
  const v = value.trim().toLowerCase()
  if (!v || !SAFE_QUERY.test(v)) return null
  return v
}

export function sanitizeFrom(value: string): ModelManagementFrom {
  if (value === 'provider' || value === 'providers-accounts' || value === 'dashboard') return value
  return null
}

export function sanitizeFocus(value: string): ModelFocusTarget {
  if (value === 'default-model' || value === 'verify-call') return value
  return null
}

export function sanitizeHighlightStep(value: string): ModelHighlightStep {
  if (value === 'account' || value === 'limit' || value === 'defaultModel' || value === 'verify') return value
  return null
}

export function parseModelManagementContext(query: LocationQuery): ModelManagementContext {
  return {
    providerId: sanitizeProviderId(toSingleQueryValue(query.provider)),
    from: sanitizeFrom(toSingleQueryValue(query.from)),
    focus: sanitizeFocus(toSingleQueryValue(query.focus)),
    highlightStep: sanitizeHighlightStep(toSingleQueryValue(query.highlightStep))
  }
}

export function buildProvidersAccountsBackQuery(context: ModelManagementContext) {
  return context.providerId ? { provider: context.providerId } : {}
}
