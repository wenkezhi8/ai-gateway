import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing classifier switch error handling', () => {
  it('should surface backend classifier switch error message', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('const err = e as any')
    expect(viewFile).toContain('err?.response?.data?.error?.message')
    expect(viewFile).toContain('handleApiError(new Error(detailMessage),')
  })
})
