import { useEditionStore } from '@/store/domain/edition'
import { canAccessPath } from '@/constants/edition-visibility'

export async function canAccessEditionRoute(path: string): Promise<boolean> {
  const editionStore = useEditionStore()
  if (!editionStore.config) {
    await editionStore.fetchEditionConfig()
  }

  const currentEdition = editionStore.config?.type ?? 'basic'
  return canAccessPath(path, currentEdition)
}
