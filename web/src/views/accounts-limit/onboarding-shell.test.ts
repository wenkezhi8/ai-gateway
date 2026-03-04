import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('accounts-limit onboarding shell copy', () => {
  it('should unify provider naming as AI服务商 in menu/router/dashboard entry', () => {
    const menuConfigFile = readFileSync(join(process.cwd(), 'src/components/Layout/menu-config.ts'), 'utf-8')
    const routerFile = readFileSync(join(process.cwd(), 'src/router/index.ts'), 'utf-8')
    const dashboardFile = readFileSync(join(process.cwd(), 'src/views/dashboard/index.vue'), 'utf-8')

    expect(menuConfigFile).toContain("{ path: '/providers-accounts', title: 'AI服务商', icon: 'Key' }")
    expect(routerFile).toContain("path: '/providers-accounts'")
    expect(routerFile).toContain("meta: { title: 'AI服务商', icon: 'Key' }")
    expect(dashboardFile).toContain('>AI服务商</el-button>')

    expect(menuConfigFile).not.toContain('账号与限额')
    expect(routerFile).not.toContain('账号与限额')
    expect(dashboardFile).not.toContain('账号与限额')
  })

  it('should provide explicit 3-step onboarding actions in accounts-limit shell', () => {
    const accountsLimitFile = readFileSync(join(process.cwd(), 'src/views/accounts-limit/index.vue'), 'utf-8')

    expect(accountsLimitFile).toContain('按 3 步完成接入：添加账号 → 设置限额 → 回到仪表盘验证')
    expect(accountsLimitFile).toContain('第1步：添加账号')
    expect(accountsLimitFile).toContain('第2步：设置限额')
    expect(accountsLimitFile).toContain('第3步：前往仪表盘验证')
    expect(accountsLimitFile).toContain('完成后进入第2步')
    expect(accountsLimitFile).toContain('完成后前往第3步验证')
    expect(accountsLimitFile).toContain("router.push('/dashboard')")
  })
})
