import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama runtime config panel', () => {
  it('groups nonessential runtime settings into advanced section', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')

    expect(tabFile).toContain('基础设置')
    expect(tabFile).toContain('高级设置')
    expect(tabFile).toContain('系统已使用推荐运行参数，非必要无需调整')
    expect(tabFile).toContain('启动方式')
    expect(tabFile).toContain('自动轮询')
    expect(tabFile).toContain('启动时自动预热')
    expect(tabFile).toContain('自动重启次数')
    expect(tabFile).toContain('el-collapse')
    expect(logicFile).toContain('getOllamaRuntimeConfig')
    expect(logicFile).toContain('preloadOllamaModels')
    expect(logicFile).toContain('updateOllamaRuntimeConfig')
    expect(logicFile).toContain('ollamaPreloadResults')
    expect(logicFile).toContain('saveOllamaRuntimeConfig')
  })
})
