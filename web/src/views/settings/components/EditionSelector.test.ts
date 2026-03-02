import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('edition selector component', () => {
  it('renders edition cards and dependency checks', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('版本管理')
    expect(content).toContain('edition-cards')
    expect(content).toContain('canSelectEdition')
    expect(content).toContain('保存配置')
    expect(content).toContain('checkDependencies')
  })
})
