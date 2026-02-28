import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing classifier async switch flow', () => {
  it('should use async switch domain api and friendly timeout message', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(logicFile).toContain('switchClassifierModelAsync')
    expect(logicFile).toContain('getClassifierSwitchTask')
    expect(logicFile).toContain('正在加载模型，首次可能较慢（最多180秒）')
    expect(logicFile).toContain('模型加载超时，请继续等待Ollama完成加载后重试')
  })
})
