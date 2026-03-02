import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing timeout input range', () => {
  it('should remove classifier timeout controls from basic routing page', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/RoutePolicyTab.vue'), 'utf-8')

    expect(tabFile).not.toContain('超时(ms)')
    expect(tabFile).not.toContain(':max="10000"')
  })
})
