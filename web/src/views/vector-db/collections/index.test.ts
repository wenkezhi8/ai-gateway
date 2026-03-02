import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db collections page', () => {
  it('contains pinia state and echarts integration markers', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/collections/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('useVectorDbStore')
    expect(content).toContain("from 'echarts'")
    expect(content).toContain('collectionTrendChartRef')
  })
})
