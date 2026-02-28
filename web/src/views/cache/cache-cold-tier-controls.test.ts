import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('cache cold tier controls', () => {
  it('should expose cold tier configuration and tier actions in cache page', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).toContain('冷热向量分层')
    expect(viewFile).toContain('coldVectorEnabled')
    expect(viewFile).toContain('coldVectorQueryEnabled')
    expect(viewFile).toContain('coldVectorBackend')
    expect(viewFile).toContain('hotMemoryHighWatermarkPercent')
    expect(viewFile).toContain('hotMemoryReliefPercent')
    expect(viewFile).toContain('migrateHotToCold')
    expect(viewFile).toContain('promoteToHotTier')
    expect(viewFile).toContain('loadVectorTierStats')
  })
})
