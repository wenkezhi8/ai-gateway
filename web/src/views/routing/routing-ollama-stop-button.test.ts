import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing ollama stop button', () => {
  it('should include stop ollama button and stop domain api call', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/OllamaTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(tabFile).toContain('停止 Ollama')
    expect(logicFile).toContain('stopOllamaApi')
  })
})
