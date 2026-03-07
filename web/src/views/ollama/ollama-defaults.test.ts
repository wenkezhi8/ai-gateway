import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama defaults alignment', () => {
  it('keeps classifier and vector timeout fallbacks aligned with shared defaults', () => {
    const constantsFile = readFileSync(join(process.cwd(), 'src/constants/routing.ts'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')

    expect(constantsFile).toContain('ROUTING_OLLAMA_DEFAULT_EMBEDDING_TIMEOUT_MS = 3000')
    expect(logicFile).toContain('classifier_timeout_ms: DEFAULT_CLASSIFIER_CONFIG.timeout_ms')
    expect(logicFile).toContain('vector_ollama_embedding_timeout_ms: ROUTING_OLLAMA_DEFAULT_EMBEDDING_TIMEOUT_MS')
    expect(logicFile).toContain('Number(dualModelConfig.classifier_timeout_ms || DEFAULT_CLASSIFIER_CONFIG.timeout_ms)')
    expect(logicFile).toContain('Number(dualModelConfig.vector_ollama_embedding_timeout_ms || ROUTING_OLLAMA_DEFAULT_EMBEDDING_TIMEOUT_MS)')
  })
})
