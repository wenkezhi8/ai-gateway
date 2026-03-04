import type { ProviderType, PublicProviderInfo } from '@/api/provider'

export interface ProviderDisplayMeta {
  label: string
  color: string
  logo: string
  defaultModel: string
  custom: boolean
}

interface ProviderDisplayMetaInput {
  providerTypes: ProviderType[]
  publicProviders: PublicProviderInfo[]
  fallbackColor: string
}

function firstNonEmpty(...values: Array<string | undefined>): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value
    }
  }
  return ''
}

export function resolveProviderDisplayMeta(providerId: string, input: ProviderDisplayMetaInput): ProviderDisplayMeta {
  const providerTypeMeta = input.providerTypes.find(item => item.id === providerId)
  const publicProviderMeta = input.publicProviders.find(item => item.id === providerId)

  return {
    label: firstNonEmpty(providerTypeMeta?.label, publicProviderMeta?.label, providerId),
    color: firstNonEmpty(providerTypeMeta?.color, publicProviderMeta?.color, input.fallbackColor),
    logo: firstNonEmpty(providerTypeMeta?.logo, publicProviderMeta?.logo),
    defaultModel: firstNonEmpty(publicProviderMeta?.default_model),
    custom: !providerTypeMeta && !publicProviderMeta
  }
}
