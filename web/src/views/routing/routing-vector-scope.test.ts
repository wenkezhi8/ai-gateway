import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing vector scope', () => {
  it('should include cold-tier configuration and tier actions in routing vector tab', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')
    const vectorTabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/VectorManagementTab.vue'), 'utf-8')

    expect(logicFile).toContain('getVectorTierConfig')
    expect(logicFile).toContain('updateVectorTierConfig')
    expect(logicFile).toContain('triggerVectorTierMigrate')
    expect(logicFile).toContain('promoteVectorTierEntry')
    expect(vectorTabFile).toContain('冷热向量分层')
    expect(vectorTabFile).toContain('手动迁移')
    expect(vectorTabFile).toContain('手动回暖')
    expect(vectorTabFile).toContain('cold_vector_backend')
  })
})
