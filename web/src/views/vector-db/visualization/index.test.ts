import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db visualization page', () => {
  it('contains scatter chart and filter controls', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/visualization/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('向量可视化')
    expect(content).toContain('getVectorScatterData')
    expect(content).toContain('sample_size')
    expect(content).toContain('散点图')
  })
})
