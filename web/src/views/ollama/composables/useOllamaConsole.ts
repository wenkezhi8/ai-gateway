import { useOllamaConsoleCore } from './useOllamaConsoleCore'

export function useOllamaConsole() {
  const ctx = useOllamaConsoleCore()

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
