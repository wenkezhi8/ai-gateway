import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama vector scope', () => {
  it('contains vector model management and hot-cold tier controls', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/VectorManagementTab.vue'), 'utf-8')

    expect(tabFile).toContain('向量模型管理')
    expect(tabFile).toContain('切换模型')
    expect(tabFile).toContain('@click="ctx.switchVectorModel"')
    expect(tabFile).toContain('启动模型')
    expect(tabFile).not.toContain('下载模型')
    expect(tabFile).not.toContain('删除模型')
    expect(tabFile).toContain('vector_ollama_embedding_model')
    expect(tabFile).toContain('冷热向量分层')
    expect(tabFile).toContain('手动迁移')
    expect(tabFile).toContain('手动回暖')
    expect(tabFile).toContain('cold_vector_backend')
    expect(tabFile).toContain('cold_archive_enabled')
    expect(tabFile).toContain('cold_archive_mode')
    expect(tabFile).toContain('cold_archive_near_expiry_seconds')
    expect(tabFile).toContain('archive_queue_depth')

    expect(logicFile).toContain('getVectorTierConfig')
    expect(logicFile).toContain('updateVectorTierConfig')
    expect(logicFile).toContain('triggerVectorTierMigrate')
    expect(logicFile).toContain('promoteVectorTierEntry')
    expect(logicFile).toContain('async function switchVectorModel')
    expect(logicFile).toContain('async function startVectorModel')
    expect(logicFile).toContain('await switchVectorModel(')
  })
})
