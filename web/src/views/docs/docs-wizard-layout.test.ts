import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('docs wizard page layout', () => {
  it('uses a wizard-like docs structure with step navigation and step panels', () => {
    const docsFile = resolve(process.cwd(), 'src/views/docs/index.vue')
    const content = readFileSync(docsFile, 'utf-8')

    expect(content).toContain('wizard-layout')
    expect(content).toContain('step-nav')
    expect(content).toContain('step-panel')
  })
})
