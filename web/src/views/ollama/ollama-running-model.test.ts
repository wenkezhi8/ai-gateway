import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama running model visibility', () => {
  it('shows running model details and polling controls', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')

    expect(tabFile).toContain('当前运行模型')
    expect(tabFile).toContain('运行中模型')
    expect(tabFile).toContain('显存占用')
    expect(tabFile).toContain('自动轮询')
    expect(tabFile).toContain('轮询间隔')
    expect(tabFile).toContain('formatVramBytes(item.size_vram)')
    expect(logicFile).toContain('running_model')
    expect(logicFile).toContain('running_model_details')
  })

  it('shows intent/vector switches at top and persists config', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')

    expect(tabFile).toContain('启用意图分类器')
    expect(tabFile).toContain('启用向量 Pipeline')
    expect(tabFile).toContain('dualModelConfig.classifier_enabled')
    expect(tabFile).toContain('dualModelConfig.vector_pipeline_enabled')
    expect(tabFile).toContain('saveDualModelConfigData')
  })
})
