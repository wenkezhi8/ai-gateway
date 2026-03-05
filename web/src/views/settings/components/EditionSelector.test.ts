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

  it('shows realtime setup logs panel and copy action', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('安装过程日志')
    expect(content).toContain('setup-log-panel')
    expect(content).toContain('复制日志')
    expect(content).toContain('setupTask.logs')
  })

  it('shows native runtime no-auto-fallback guidance', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('native 模式')
    expect(content).toContain('不会自动切换到 Docker')
  })

  it('shows manual script install guide with command synced to current selection', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('手动脚本安装（与当前选择一致）')
    expect(content).toContain('./scripts/setup-edition-env.sh')
    expect(content).toContain('--edition')
    expect(content).toContain('--runtime')
    expect(content).toContain('manualInstallCommand')
    expect(content).toContain('copyManualInstallCommand')
  })

  it('uses unified basic edition wording as stop all dependencies', () => {
    const file = resolve(process.cwd(), 'src/views/settings/components/EditionSelector.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('基础版：停止所有依赖')
    expect(content).not.toContain('卸载所有依赖')
  })
})
