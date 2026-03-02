import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('layout vector-db entry', () => {
  it('adds a top-right vector-db entry and removes vector-db sidebar menu items', () => {
    const file = resolve(process.cwd(), 'src/components/Layout/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('vectorDBConsoleURL')
    expect(content).toContain(':href="vectorDBConsoleURL"')
    expect(content).toContain('向量管理')

    expect(content).not.toContain("{ path: '/vector-db/collections', title: '向量集合'")
    expect(content).not.toContain("{ path: '/vector-db/search', title: '向量检索'")
    expect(content).not.toContain("{ path: '/vector-db/import', title: '向量导入'")
  })
})
