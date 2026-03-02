import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('edition guard wiring', () => {
  it('checks vector-db and knowledge routes by edition features', () => {
    const guardFile = resolve(process.cwd(), 'src/router/guards/edition-guard.ts')
    const guardContent = readFileSync(guardFile, 'utf-8')
    expect(guardContent).toContain("path.startsWith('/vector-db')")
    expect(guardContent).toContain('hasVectorDBManagement')
    expect(guardContent).toContain("path.startsWith('/knowledge')")
    expect(guardContent).toContain('hasKnowledgeBase')

    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const routerContent = readFileSync(routerFile, 'utf-8')
    expect(routerContent).toContain('canAccessEditionRoute')
    expect(routerContent).toContain('await canAccessEditionRoute(to.path)')
  })
})
