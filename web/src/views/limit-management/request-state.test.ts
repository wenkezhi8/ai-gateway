import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('limit management request state', () => {
  it('should expose aggregated request error alert with retry action', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/limit-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('v-if="hasRequestError"')
    expect(viewFile).toContain('请求加载失败')
    expect(viewFile).toContain('@click="refreshAll"')
  })

  it('should track request errors for accounts, history and alerts separately', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/limit-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('const requestErrors = reactive({')
    expect(viewFile).toContain("accounts: ''")
    expect(viewFile).toContain("history: ''")
    expect(viewFile).toContain("alerts: ''")
    expect(viewFile).toContain('const hasRequestError = computed(')
  })
})
