import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('providers accounts provider name layout', () => {
  it('should keep provider names on one line', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toMatch(/\.provider-name\s*\{[\s\S]*?white-space:\s*nowrap;/)
    expect(viewFile).toMatch(/\.provider-option\s*\{[\s\S]*?white-space:\s*nowrap;/)
  })

  it('should source provider metadata from api instead of static constants', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toContain('getProviderTypes')
    expect(viewFile).not.toContain('PROVIDERS_ACCOUNTS_BASE_TYPES')
    expect(viewFile).not.toContain('PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS')
    expect(viewFile).not.toContain('PROVIDERS_ACCOUNTS_CODING_PLAN_ENDPOINTS')
    expect(viewFile).not.toContain('PROVIDERS_ACCOUNTS_PROVIDER_COLORS')
    expect(viewFile).not.toContain('PROVIDERS_ACCOUNTS_PROVIDER_LOGOS')
  })
})
