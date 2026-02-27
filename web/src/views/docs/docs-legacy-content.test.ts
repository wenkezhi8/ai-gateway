import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('docs page legacy content integration', () => {
  it('keeps legacy docs as route pages source and supports initial tab rendering', () => {
    const docsFile = resolve(process.cwd(), 'src/views/docs/legacy-content.vue')
    const content = readFileSync(docsFile, 'utf-8')

    expect(content).toContain('initialTab')
    expect(content).toContain('hideTabs')
    expect(content).toContain('hideHeader')
  })
})
