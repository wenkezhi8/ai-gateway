import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama runtime config panel', () => {
  it('shows startup mode config and monitoring details', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')

    expect(tabFile).toContain('启动方式')
    expect(tabFile).toContain('保存配置')
    expect(tabFile).toContain('自动重启次数')
    expect(tabFile).toContain('monitoring_stats')
    expect(tabFile).toContain('启动时自动预热')
    expect(tabFile).toContain('立即预热')
    expect(tabFile).toContain('预热结果')
    expect(logicFile).toContain('getOllamaRuntimeConfig')
    expect(logicFile).toContain('preloadOllamaModels')
    expect(logicFile).toContain('updateOllamaRuntimeConfig')
    expect(logicFile).toContain('ollamaPreloadResults')
    expect(logicFile).toContain('saveOllamaRuntimeConfig')
  })
})
