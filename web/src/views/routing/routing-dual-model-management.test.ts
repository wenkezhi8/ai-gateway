import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing dual model management', () => {
  it('should centralize ollama intent+vector model config in model tab', () => {
    const modelTabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/ModelManagementTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')
    const pageFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(modelTabFile).toContain('Ollama 双模型管理（意图 + 向量）')
    expect(modelTabFile).toContain('ctx.dualModelConfig.classifier_active_model')
    expect(modelTabFile).toContain('ctx.dualModelConfig.vector_ollama_embedding_model')
    expect(modelTabFile).toContain('ctx.saveDualModelConfigData')
    expect(modelTabFile).not.toContain('模型评分管理')
    expect(modelTabFile).not.toContain('Intent Engine（本地意图+向量）')

    expect(logicFile).toContain('getOllamaDualModelConfig')
    expect(logicFile).toContain('updateOllamaDualModelConfig')
    expect(logicFile).not.toContain('getIntentEngineConfig')
    expect(logicFile).not.toContain('updateIntentEngineConfig')
    expect(logicFile).not.toContain('getRouterModels')
    expect(logicFile).not.toContain('updateModelScore')
    expect(logicFile).not.toContain('modelSearch')
    expect(logicFile).not.toContain('filteredModels')
    expect(pageFile).not.toContain('模型评分')
  })
})
