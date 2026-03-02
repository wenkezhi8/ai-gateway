import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama service stop button', () => {
  it('contains stop service button and stop api call', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')

    expect(tabFile).toContain('停止服务')
    expect(tabFile).not.toContain('启用意图分类器')
    expect(tabFile).not.toContain('启用向量 Pipeline')
    expect(logicFile).toContain('stopOllamaApi')
  })
})
