import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db audit page', () => {
  it('contains audit title and audit api usage', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/audit/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('向量审计日志')
    expect(content).toContain('listVectorAuditLogs')
    expect(content).toContain('<el-table')
  })
})
