import type { PublicProviderInfo } from '@/api/provider'

export interface BuildProviderIdsInput {
  publicProviders: PublicProviderInfo[]
  modelsByProvider: Record<string, string[]>
  providerDefaults: Record<string, string>
}

function normalizeProviderID(input: string): string {
  return String(input || '').trim().toLowerCase()
}

export function buildProviderIdsForSettings(input: BuildProviderIdsInput): string[] {
  const providerIds = new Set<string>()

  for (const item of input.publicProviders) {
    const normalized = normalizeProviderID(item.id)
    if (!normalized) continue
    providerIds.add(normalized)
  }

  for (const providerId of Object.keys(input.modelsByProvider || {})) {
    const normalized = normalizeProviderID(providerId)
    if (!normalized) continue
    providerIds.add(normalized)
  }

  // Defaults only become visible when provider already exists in a concrete source.
  for (const providerId of Object.keys(input.providerDefaults || {})) {
    const normalized = normalizeProviderID(providerId)
    if (!normalized) continue
    if (providerIds.has(normalized)) {
      providerIds.add(normalized)
    }
  }

  return Array.from(providerIds)
}

