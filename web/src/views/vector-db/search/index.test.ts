import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db search page', () => {
  it('contains search and recommend actions with result table', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/search/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('向量检索')
    expect(content).toContain('searchVectorCollection')
    expect(content).toContain('recommendVectorCollection')
    expect(content).toContain('<el-table')
  })
})
