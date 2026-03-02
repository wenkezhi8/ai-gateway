import { useRoutingConsole } from '@/views/routing/composables/useRoutingConsole'

export function useOllamaConsole() {
  const ctx = useRoutingConsole()

  async function reloadAllPanels() {
    await Promise.all([
      ctx.reloadOllamaPanel(),
      ctx.reloadModelsPanel(),
      ctx.reloadVectorPanel()
    ])
  }

  return {
    ...ctx,
    reloadAllPanels
  }
}
