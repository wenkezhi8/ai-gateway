import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db collections page', () => {
  it('contains empty collection action wiring', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/collections/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('handleEmpty')
    expect(content).toContain('emptyVectorCollection')
    expect(content).toContain('清空该 Collection 的全部向量')
  })
})
