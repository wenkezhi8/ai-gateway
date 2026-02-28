import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing vector scope', () => {
  it('should only include vector observe plus rebuild capabilities', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')
    const vectorTabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/VectorManagementTab.vue'), 'utf-8')

    expect(logicFile).toContain('getVectorStats')
    expect(logicFile).toContain('getVectorTierStats')
    expect(logicFile).toContain('rebuildVectorIndex')
    expect(vectorTabFile).not.toContain('triggerVectorTierMigrate')
    expect(vectorTabFile).not.toContain('promoteVectorTierEntry')
  })
})
