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
    expect(content).toContain('安装依赖')
    expect(content).toContain('runtime')
    expect(content).toContain('checkDependencies')
  })

  it('uses independent setup edition target for install action', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('setupEdition')
    expect(content).toContain('安装目标版本')
    expect(content).toContain('edition: setupEdition.value')
  })
})
