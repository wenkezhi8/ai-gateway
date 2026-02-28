import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing timeout input range', () => {
  it('should allow timeout greater than 2 seconds for heavier models', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/RoutePolicyTab.vue'), 'utf-8')

    expect(tabFile).toContain(':max="10000"')
  })
})
