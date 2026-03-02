import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router vector-db standalone route', () => {
  it('defines /vector-db as standalone layout and removes vector pages from /console', () => {
    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const content = readFileSync(routerFile, 'utf-8')

    expect(content).toContain("path: '/vector-db'")
    expect(content).toContain("component: () => import('@/components/Layout/VectorDBLayout.vue')")
    expect(content).toContain("redirect: '/vector-db/collections'")
    expect(content).toContain("path: 'collections'")
    expect(content).toContain("path: 'search'")
    expect(content).toContain("path: 'import'")
    expect(content).toContain("path: 'monitoring'")
    expect(content).toContain("path: 'permissions'")
    expect(content).toContain("path: 'backup'")
    expect(content).toContain("path: 'audit'")
    expect(content).toContain("path: 'visualization'")

    const consoleBlock = content.match(
      /path:\s*'\/console'[\s\S]*?children:\s*\[([\s\S]*?)\]\s*\n\s*}\s*,\s*\n\s*{\s*\n\s*path:\s*'\/vector-db'/
    )

    expect(consoleBlock).not.toBeNull()
    expect(consoleBlock?.[1]).not.toContain("path: '/vector-db/collections'")
    expect(consoleBlock?.[1]).not.toContain("path: '/vector-db/search'")
  })
})
