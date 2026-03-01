import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('cache cold tier controls', () => {
  it('should remove cold tier controls from cache page', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).not.toContain('冷热向量分层')
    expect(viewFile).not.toContain('coldVectorEnabled')
    expect(viewFile).not.toContain('coldVectorQueryEnabled')
    expect(viewFile).not.toContain('coldVectorBackend')
    expect(viewFile).not.toContain('hotMemoryHighWatermarkPercent')
    expect(viewFile).not.toContain('hotMemoryReliefPercent')
    expect(viewFile).not.toContain('migrateHotToCold')
    expect(viewFile).not.toContain('promoteToHotTier')
    expect(viewFile).not.toContain('loadVectorTierStats')
  })
})
