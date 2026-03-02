import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('ollama page', () => {
  it('contains consolidated ollama console tabs', () => {
    const file = resolve(process.cwd(), 'src/views/ollama/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('Ollama 控制台')
    expect(content).toContain('label="Ollama"')
    expect(content).toContain('label="意图路由"')
    expect(content).toContain('label="向量管理"')
  })
})
