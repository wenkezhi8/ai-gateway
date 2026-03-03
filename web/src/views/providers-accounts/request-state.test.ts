import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('providers accounts request state', () => {
  it('should render accounts error state with retry action', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toContain('v-if="accountsError"')
    expect(viewFile).toContain('账号列表加载失败')
    expect(viewFile).toContain('@click="loadAccounts"')
  })

  it('should avoid showing empty state when accounts request failed', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toContain('!accountsError && filteredAccounts.length === 0')
    expect(viewFile).toContain("const accountsError = ref('')")
  })
})
