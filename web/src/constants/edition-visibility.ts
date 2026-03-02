import type { EditionType } from '@/api/edition-domain'

const EDITION_LEVEL: Record<EditionType, number> = {
  basic: 1,
  standard: 2,
  enterprise: 3
}

export const ROUTE_MIN_EDITION: Readonly<Record<string, EditionType>> = {
  '/ollama': 'standard',
  '/vector-db': 'enterprise',
  '/knowledge': 'enterprise'
}

export function isEditionAtLeast(current: EditionType, minimum: EditionType): boolean {
  return EDITION_LEVEL[current] >= EDITION_LEVEL[minimum]
}

export function canAccessPath(path: string, edition: EditionType): boolean {
  for (const [prefix, minEdition] of Object.entries(ROUTE_MIN_EDITION)) {
    if (path.startsWith(prefix)) {
      return isEditionAtLeast(edition, minEdition)
    }
  }
  return true
}
