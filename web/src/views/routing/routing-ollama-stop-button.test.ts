import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing ollama stop button', () => {
  it('should include stop ollama button and stop domain api call', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('停止 Ollama')
    expect(viewFile).toContain('stopOllamaApi')
  })
})
