import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing tabs layout', () => {
  it('should organize console tabs without ollama tab', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('<el-tabs')
    expect(viewFile).toContain('name="policy"')
    expect(viewFile).not.toContain('name="ollama"')
    expect(viewFile).toContain('name="models"')
    expect(viewFile).toContain('name="vector"')
  })
})
