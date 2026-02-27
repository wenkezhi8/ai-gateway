import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router home route', () => {
  it('defines a public Home route at root path', () => {
    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const content = readFileSync(routerFile, 'utf-8')

    expect(content).toContain("name: 'Home'")
    expect(content).toContain("path: '/'")
    expect(content).toContain('public: true')
  })
})
