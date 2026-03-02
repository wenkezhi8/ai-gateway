import { useEditionStore } from '@/store/domain/edition'

export async function canAccessEditionRoute(path: string): Promise<boolean> {
  const editionStore = useEditionStore()
  if (!editionStore.config) {
    await editionStore.fetchEditionConfig()
  }

  if (path.startsWith('/vector-db')) {
    return editionStore.hasVectorDBManagement
  }
  if (path.startsWith('/knowledge')) {
    return editionStore.hasKnowledgeBase
  }
  if (path.startsWith('/ollama')) {
    return editionStore.isStandard || editionStore.isEnterprise
  }

  return true
}
