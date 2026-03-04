import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('providers accounts onboarding markers', () => {
  it('renders onboarding CTA and AI服务商 title markers', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toContain('继续未完成配置')
    expect(viewFile).toContain('AI服务商')
    expect(viewFile).toContain('步骤4：可调用验证')
  })

  it('keeps per-pill action routing markers', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(viewFile).toContain("path: '/limit-management'")
    expect(viewFile).toContain("goToModelManagement(normalizedProvider, 'default-model', 'defaultModel')")
    expect(viewFile).toContain("goToModelManagement(normalizedProvider, 'verify-call', 'verify')")
    expect(viewFile).toContain('showAddAccountDialog(normalizedProvider)')
  })
})
