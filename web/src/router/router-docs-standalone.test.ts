import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router docs route', () => {
  it('keeps /docs as a standalone public top-level route', () => {
    const routerFile = resolve(process.cwd(), 'src/router/index.ts')
    const content = readFileSync(routerFile, 'utf-8')

    const docsRouteIndex = content.indexOf("path: '/docs'")
    const consoleRouteIndex = content.indexOf("path: '/console'")

    expect(docsRouteIndex).toBeGreaterThan(-1)
    expect(consoleRouteIndex).toBeGreaterThan(-1)
    expect(docsRouteIndex).toBeLessThan(consoleRouteIndex)
    expect(content).toMatch(/path:\s*'\/docs'[\s\S]*?public:\s*true/)

    const consoleBlock = content.match(
      /path:\s*'\/console'[\s\S]*?children:\s*\[([\s\S]*?)\]\s*\n\s*}\s*,\s*\n\s*{\s*\n\s*path:\s*'\/login'/
    )

    expect(consoleBlock).not.toBeNull()
    expect(consoleBlock?.[1]).not.toContain("path: '/docs'")
  })
})
