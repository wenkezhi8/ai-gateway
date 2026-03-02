import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('layout edition badge and feature gates', () => {
  it('shows edition badge and gates enterprise entries', () => {
    const file = resolve(process.cwd(), 'src/components/Layout/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('edition-tag')
    expect(content).toContain('editionLabel')
    expect(content).toContain('v-if="showVectorDBEntry"')
    expect(content).toContain('v-if="showKnowledgeEntry"')
  })
})
