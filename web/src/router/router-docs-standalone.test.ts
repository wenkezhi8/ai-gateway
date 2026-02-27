import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router docs route', () => {
  it('defines /docs as a standalone route with child pages and default redirect', () => {
    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const content = readFileSync(routerFile, 'utf-8')

    const docsRouteIndex = content.indexOf("path: '/docs'")
    const consoleRouteIndex = content.indexOf("path: '/console'")

    expect(docsRouteIndex).toBeGreaterThan(-1)
    expect(consoleRouteIndex).toBeGreaterThan(-1)
    expect(docsRouteIndex).toBeLessThan(consoleRouteIndex)
    expect(content).toMatch(/path:\s*'\/docs'[\s\S]*?public:\s*true/)
    expect(content).toContain("path: '',")
    expect(content).toContain("redirect: '/docs/getting-started'")
    expect(content).toContain("path: 'getting-started'")
    expect(content).toContain("path: 'wizard'")
    expect(content).toContain("path: 'api'")
    expect(content).toContain("path: 'sdk'")
    expect(content).toContain("path: 'providers'")
    expect(content).toContain("path: 'admin'")
    expect(content).toContain("path: 'errors'")

    const consoleBlock = content.match(
      /path:\s*'\/console'[\s\S]*?children:\s*\[([\s\S]*?)\]\s*\n\s*}\s*,\s*\n\s*{\s*\n\s*path:\s*'\/login'/
    )

    expect(consoleBlock).not.toBeNull()
    expect(consoleBlock?.[1]).not.toContain("path: '/docs'")
  })
})
