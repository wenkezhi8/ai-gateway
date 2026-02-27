import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('index.html auth precheck', () => {
  it('injects public precheck paths from navigation constants at build time', () => {
    const indexHtmlPath = resolve(process.cwd(), 'index.html')
    const viteConfigPath = resolve(process.cwd(), 'vite.config.ts')
    const navigationConstantsPath = resolve(process.cwd(), 'src/constants/navigation.ts')

    const indexContent = readFileSync(indexHtmlPath, 'utf-8')
    const viteConfigContent = readFileSync(viteConfigPath, 'utf-8')
    const navigationConstantsContent = readFileSync(navigationConstantsPath, 'utf-8')

    expect(indexContent).toContain('var publicPaths = __PUBLIC_PRECHECK_PATHS__')
    expect(viteConfigContent).toContain('PUBLIC_PRECHECK_PATHS')
    expect(viteConfigContent).toContain('__PUBLIC_PRECHECK_PATHS__')
    expect(viteConfigContent).toContain('transformIndexHtml')

    expect(navigationConstantsContent).toContain('PUBLIC_PRECHECK_PATHS')
    expect(navigationConstantsContent).toContain('HOME_ROUTE')
    expect(navigationConstantsContent).toContain('DOCS_ROUTE')
    expect(navigationConstantsContent).toContain('LOGIN_ROUTE')
  })
})
