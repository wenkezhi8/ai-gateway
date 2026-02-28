import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing classifier switch error handling', () => {
  it('should surface backend classifier switch error message', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(logicFile).toContain('const err = e as any')
    expect(logicFile).toContain('err?.response?.data?.error?.message')
    expect(logicFile).toContain('handleApiError(new Error(detailMessage),')
  })
})
