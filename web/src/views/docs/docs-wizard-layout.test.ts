import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('docs layout shell', () => {
  it('uses a clawd-like docs shell with sidebar navigation and route content outlet', () => {
    const docsFile = resolve(process.cwd(), 'src/views/docs/index.vue')
    const content = readFileSync(docsFile, 'utf-8')

    expect(content).toContain('docs-layout-shell')
    expect(content).toContain('docs-sidebar')
    expect(content).toContain('docs-content')
    expect(content).toContain('<router-view')
  })
})
