import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router knowledge standalone route', () => {
  it('defines /knowledge as standalone layout and removes knowledge pages from /console', () => {
    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const content = readFileSync(routerFile, 'utf-8')

    expect(content).toContain("path: '/knowledge'")
    expect(content).toContain("component: () => import('@/components/Layout/KnowledgeLayout.vue')")
    expect(content).toContain("redirect: '/knowledge/documents'")
    expect(content).toContain("path: 'documents'")
    expect(content).toContain("path: 'chat'")
    expect(content).toContain("path: 'config'")

    const consoleBlock = content.match(
      /path:\s*'\/console'[\s\S]*?children:\s*\[([\s\S]*?)\]\s*\n\s*}\s*,\s*\n\s*{\s*\n\s*path:\s*'\/vector-db'/
    )

    expect(consoleBlock).not.toBeNull()
    expect(consoleBlock?.[1]).not.toContain("path: '/knowledge/documents'")
    expect(consoleBlock?.[1]).not.toContain("path: '/knowledge/chat'")
    expect(consoleBlock?.[1]).not.toContain("path: '/knowledge/config'")
  })
})
