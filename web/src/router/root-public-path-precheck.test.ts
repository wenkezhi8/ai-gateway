import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('index.html auth precheck', () => {
  it('treats root path as public to avoid forcing login redirect on landing page', () => {
    const indexHtmlPath = resolve(process.cwd(), 'index.html')
    const content = readFileSync(indexHtmlPath, 'utf-8')
    const publicPathsMatch = content.match(/publicPaths\s*=\s*\[([^\]]+)\]/)

    expect(publicPathsMatch).not.toBeNull()
    expect(publicPathsMatch?.[1]).toContain("'/'")
    expect(publicPathsMatch?.[1]).toContain("'/docs'")
  })
})
