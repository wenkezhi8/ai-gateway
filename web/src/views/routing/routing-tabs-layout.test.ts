import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing tabs layout', () => {
  it('should keep routing page focused on route policy only', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('RoutePolicyTab')
    expect(viewFile).not.toContain('<el-tabs')
    expect(viewFile).not.toContain('ModelManagementTab')
    expect(viewFile).not.toContain('VectorManagementTab')
  })
})
