import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db monitoring alerts page', () => {
  it('contains alert rule CRUD api usage', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/monitoring/alerts.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('告警规则')
    expect(content).toContain('listAlertRules')
    expect(content).toContain('createAlertRule')
    expect(content).toContain('deleteAlertRule')
  })
})
